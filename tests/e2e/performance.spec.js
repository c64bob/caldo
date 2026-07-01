const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
const path = require('node:path');
const { ensureBrowserCSRFCookie, gotoApp } = require('./helpers/app');
const { appURL, readState, repoRoot } = require('./helpers/state');
const { createRemoteCalendar, createRemoteTask, resetStage } = require('./helpers/stage');

const performanceEnabled = process.env.CALDO_E2E_PERF === '1';
const dataset = {
  projects: envInt('CALDO_E2E_PERF_PROJECTS', 8),
  tasks: envInt('CALDO_E2E_PERF_TASKS', 400),
  labels: envInt('CALDO_E2E_PERF_LABELS', 24)
};
const targets = {
  navigationMs: envInt('CALDO_E2E_PERF_NAVIGATION_MS', 2_000),
  searchMs: envInt('CALDO_E2E_PERF_SEARCH_MS', 2_000),
  liveSearchMs: envInt('CALDO_E2E_PERF_LIVE_SEARCH_MS', 2_000),
  syncStartMs: envInt('CALDO_E2E_PERF_SYNC_START_MS', 1_000),
  syncInteractionMs: envInt('CALDO_E2E_PERF_SYNC_INTERACTION_MS', 500),
  syncCompleteMs: envInt('CALDO_E2E_PERF_SYNC_COMPLETE_MS', 10_000)
};

test.describe('performance scenarios and measurements', () => {
  test.skip(!performanceEnabled, 'set CALDO_E2E_PERF=1 or run npm run test:e2e:performance');
  test.setTimeout(240_000);

  test('search, navigation, and manual sync stay within documented targets', async ({ page }, testInfo) => {
    const state = readState();
    const measurements = {
      commit: process.env.GITHUB_SHA || currentCommit(),
      project: testInfo.project.name,
      dataset,
      targets,
      runs: []
    };

    await seedPerformanceDataset();
    await completeSetup(page, state);

    await measureNavigation(page, measurements);
    await measureSearch(page, measurements);
    await measureManualSyncResponsiveness(page, measurements, state);

    writePerformanceResult(testInfo, measurements);
  });
});

async function completeSetup(page, state) {
  const expectedProjectLinks = dataset.projects + 1;

  await gotoApp(page, '/');
  await expect(page).toHaveURL(/\/setup$/);
  await ensureBrowserCSRFCookie(page);
  await page.locator('[name="caldav_url"]').fill(state.stageBaseURL);
  await page.locator('[name="caldav_username"]').fill(state.stageUsername);
  await page.locator('[name="caldav_password"]').fill(state.stagePassword);
  await page.getByRole('button', { name: 'Verbindung testen' }).click();
  await expect(page).toHaveURL(/\/setup\/calendars$/);
  await expect(page.locator('[name="calendar_href"]')).toHaveCount(dataset.projects);

  await ensureBrowserCSRFCookie(page);
  await page.getByRole('button', { name: 'Weiter zum Import' }).click();
  await expect(page.locator('[data-setup-import]')).toBeVisible();
  await expect(page).toHaveURL(/\/$/, { timeout: 120_000 });
  await expect(page.locator('.caldo-sidebar [data-nav-projects] a')).toHaveCount(expectedProjectLinks);
}

async function measureNavigation(page, measurements) {
  for (const item of [
    { name: 'today', path: '/today', selector: '[data-task-id], .caldo-empty-state, .caldo-list' },
    { name: 'upcoming', path: '/upcoming', selector: '.caldo-page' },
    { name: 'projects', path: '/projects', selector: '[data-navigation-overview]' },
    { name: 'labels', path: '/labels', selector: '.caldo-page' }
  ]) {
    const ms = await measurePageLoad(page, item.path, item.selector);
    record(measurements, 'navigation', item.name, ms, targets.navigationMs);
  }
}

async function measureSearch(page, measurements) {
  const titleSearchIndex = Math.max(1, Math.min(dataset.tasks, Math.floor(dataset.tasks / 2)));
  const liveSearchIndex = Math.max(1, Math.min(dataset.tasks, dataset.tasks - 1));
  const titleSearchText = taskTitle(titleSearchIndex);
  const liveSearchText = taskTitle(liveSearchIndex);
  const searchCases = [
    { name: 'title', query: titleSearchText, text: titleSearchText },
    { name: 'project', query: '#Perf Project 03', text: 'Perf Project 03' },
    { name: 'label', query: '@perflabel07', text: 'perflabel07' }
  ];

  for (const item of searchCases) {
    const started = Date.now();
    await gotoApp(page, `/search?q=${encodeURIComponent(item.query)}`);
    await expect(page.locator('[data-search-results]').filter({ hasText: item.text }).first()).toBeVisible();
    record(measurements, 'search', item.name, Date.now() - started, targets.searchMs);
  }

  await gotoApp(page, '/search');
  const liveStarted = Date.now();
  await page.locator('#global-search').fill(liveSearchText);
  await expect(page.locator('[data-search-results]').filter({ hasText: liveSearchText }).first()).toBeVisible();
  record(measurements, 'live_search', 'typing_to_result', Date.now() - liveStarted, targets.liveSearchMs);
}

async function measureManualSyncResponsiveness(page, measurements, state) {
  await gotoApp(page, '/search?q=Perf');
  await ensureBrowserCSRFCookie(page);
  const syncStatus = page.locator('.caldo-topbar #sync-status');
  const searchInput = page.locator('#global-search');

  const syncStarted = Date.now();
  await syncStatus.getByRole('button', { name: 'Jetzt synchronisieren' }).click();
  record(measurements, 'manual_sync', 'request_returned', Date.now() - syncStarted, targets.syncStartMs);

  const interactionStarted = Date.now();
  await searchInput.fill('Perf Task 0001');
  await expect(searchInput).toHaveValue('Perf Task 0001');
  record(measurements, 'manual_sync', 'ui_interaction_after_trigger', Date.now() - interactionStarted, targets.syncInteractionMs);

  await expect.poll(async () => {
    const response = await page.request.get(appURL('/sync/status'), {
      headers: { [state.proxyUserHeader]: 'e2e-user' },
      failOnStatusCode: false
    });
    if (response.status() !== 200) return '';
    return response.text();
  }, { timeout: targets.syncCompleteMs + 5_000 }).toMatch(/Status: idle[\s\S]*Letzter Sync: (?!nie)/);
  record(measurements, 'manual_sync', 'completed', Date.now() - syncStarted, targets.syncCompleteMs);
}

async function measurePageLoad(page, pathname, readySelector) {
  const started = Date.now();
  await gotoApp(page, pathname);
  await expect(page.locator(readySelector).first()).toBeVisible();
  return Date.now() - started;
}

async function seedPerformanceDataset() {
  await resetStage();
  const projects = performanceProjects();
  for (const project of projects.slice(1)) {
    await createRemoteCalendar(project);
  }

  for (let index = 1; index <= dataset.tasks; index += 1) {
    const project = projects[(index - 1) % projects.length];
    const uid = `perf-${String(index).padStart(4, '0')}`;
    await createRemoteTask({
      calendarHref: project.href,
      href: `${project.href}${uid}.ics`,
      uid,
      rawVTODO: syntheticVTODO(index, uid)
    });
  }
}

function performanceProjects() {
  const projects = [{ href: '/cal/work/', displayName: 'Work' }];
  for (let index = 2; index <= dataset.projects; index += 1) {
    projects.push({
      href: `/cal/perf-project-${String(index).padStart(2, '0')}/`,
      displayName: `Perf Project ${String(index).padStart(2, '0')}`
    });
  }
  return projects;
}

function syntheticVTODO(index, uid) {
  const labels = [
    `perflabel${String(((index - 1) % dataset.labels) + 1).padStart(2, '0')}`,
    `perflabel${String((index % dataset.labels) + 1).padStart(2, '0')}`
  ];
  const priority = (index % 3) + 1;
  const dueDay = String((index % 28) + 1).padStart(2, '0');
  return [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//Caldo E2E Performance//EN',
    'BEGIN:VTODO',
    `UID:${uid}`,
    `SUMMARY:${taskTitle(index)}`,
    `DUE;VALUE=DATE:202607${dueDay}`,
    `PRIORITY:${priority}`,
    `CATEGORIES:${labels.join(',')}`,
    'END:VTODO',
    'END:VCALENDAR',
    ''
  ].join('\r\n');
}

function taskTitle(index) {
  return `Perf Task ${String(index).padStart(4, '0')}`;
}

function record(measurements, scenario, name, ms, targetMs) {
  measurements.runs.push({ scenario, name, milliseconds: ms, target_ms: targetMs, pass: ms <= targetMs });
  expect(ms, `${scenario} ${name} took ${ms}ms, target ${targetMs}ms`).toBeLessThanOrEqual(targetMs);
}

function writePerformanceResult(testInfo, measurements) {
  const date = new Date().toISOString().slice(0, 10);
  const outDir = path.join(repoRoot, 'test-results', 'performance', date);
  fs.mkdirSync(outDir, { recursive: true });
  const safeProjectName = testInfo.project.name.replace(/[^a-z0-9_-]+/gi, '-').toLowerCase();
  const outPath = path.join(outDir, `playwright-${safeProjectName}.json`);
  fs.writeFileSync(outPath, `${JSON.stringify(measurements, null, 2)}\n`);
}

function envInt(name, fallback) {
  const raw = process.env[name];
  if (!raw) return fallback;
  const value = Number.parseInt(raw, 10);
  return Number.isFinite(value) && value > 0 ? value : fallback;
}

function currentCommit() {
  try {
    return require('node:child_process')
      .execFileSync('git', ['rev-parse', '--short', 'HEAD'], { cwd: repoRoot, encoding: 'utf8' })
      .trim();
  } catch {
    return '';
  }
}
