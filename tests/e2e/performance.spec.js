const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const {
  appFormRequest,
  ensureBrowserCSRFCookie,
  gotoApp,
  manualSync,
  taskIDFromSearch,
  taskVersion
} = require('./helpers/app');
const { appURL, readState, repoRoot } = require('./helpers/state');
const { createRemoteCalendar, createRemoteTask, deleteRemoteTask, resetStage } = require('./helpers/stage');

const performanceEnabled = process.env.CALDO_E2E_PERF === '1';
const dataset = normalizeDataset({
  projects: envInt('CALDO_E2E_PERF_PROJECTS', 8),
  tasks: envInt('CALDO_E2E_PERF_TASKS', 400),
  labels: envInt('CALDO_E2E_PERF_LABELS', 24),
  completedTasks: envInt('CALDO_E2E_PERF_COMPLETED_TASKS', 0),
  subtasks: envInt('CALDO_E2E_PERF_SUBTASKS', 0),
  conflicts: envInt('CALDO_E2E_PERF_CONFLICTS', 0)
});
const targets = {
  initialImportMs: envInt('CALDO_E2E_PERF_INITIAL_IMPORT_MS', 30_000),
  navigationMs: envInt('CALDO_E2E_PERF_NAVIGATION_MS', 2_000),
  searchMs: envInt('CALDO_E2E_PERF_SEARCH_MS', 2_000),
  liveSearchMs: envInt('CALDO_E2E_PERF_LIVE_SEARCH_MS', 2_000),
  taskEditMs: envInt('CALDO_E2E_PERF_TASK_EDIT_MS', 2_000),
  syncStartMs: envInt('CALDO_E2E_PERF_SYNC_START_MS', 1_000),
  syncInteractionMs: envInt('CALDO_E2E_PERF_SYNC_INTERACTION_MS', 500),
  syncCompleteMs: envInt('CALDO_E2E_PERF_SYNC_COMPLETE_MS', 10_000),
  conflictSyncMs: envInt('CALDO_E2E_PERF_CONFLICT_SYNC_MS', 10_000)
};
const testTimeoutMs = envInt('CALDO_E2E_PERF_TIMEOUT_MS', Math.max(240_000, dataset.tasks * 750));

test.describe('performance scenarios and measurements', () => {
  test.skip(!performanceEnabled, 'set CALDO_E2E_PERF=1 or run npm run test:e2e:performance');
  test.setTimeout(testTimeoutMs);

  test('large calendar flows stay within documented targets', async ({ page, browser }, testInfo) => {
    const state = readState();
    const measurements = {
      commit: process.env.GITHUB_SHA || currentCommit(),
      project: testInfo.project.name,
      dataset,
      targets,
      environment: {
        server_type: 'stagecaldav',
        server_version: 'local in-memory cmd/stagecaldav',
        os: `${os.type()} ${os.release()} ${os.arch()}`,
        cpu: os.cpus()[0]?.model || '',
        ram_mb: Math.round(os.totalmem() / 1024 / 1024),
        node: process.version,
        go: goVersion(),
        browser: testInfo.project.name,
        browser_version: browser.version(),
        timeout_ms: testTimeoutMs
      },
      runs: []
    };

    await seedPerformanceDataset();
    await completeSetup(page, state, measurements);

    await measureNavigation(page, measurements);
    await measureSearch(page, measurements);
    await measureTaskEdit(page, measurements);
    await measureManualSyncResponsiveness(page, measurements, state);
    await measureConflictScenario(page, measurements, state);

    writePerformanceResult(testInfo, measurements);
  });
});

async function completeSetup(page, state, measurements) {
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
  const started = Date.now();
  await page.getByRole('button', { name: 'Weiter zum Import' }).click();
  await expect(page.locator('[data-setup-import]')).toBeVisible();
  await expect(page).toHaveURL(/\/$/, { timeout: 120_000 });
  record(measurements, 'initial_import', 'setup_to_home', Date.now() - started, targets.initialImportMs);
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
  const titleSearchIndex = activeRootTaskIndex(Math.floor(dataset.tasks / 2));
  const liveSearchIndex = activeRootTaskIndex(Math.max(1, dataset.tasks - 1));
  const titleSearchText = taskTitle(titleSearchIndex);
  const liveSearchText = taskTitle(liveSearchIndex);
  const projectSearchIndex = Math.min(dataset.projects, Math.max(1, Math.min(3, activeRootCount())));
  const projectSearchText = projectName(projectSearchIndex);
  const labelSearchTaskIndex = activeRootTaskIndex(Math.min(7, activeRootCount()));
  const labelSearchIndex = ((labelSearchTaskIndex - 1) % dataset.labels) + 1;
  const labelSearchText = labelName(labelSearchIndex);
  const searchCases = [
    { name: 'title', query: titleSearchText, text: titleSearchText },
    { name: 'project', query: `#${projectSearchText}`, text: projectSearchText },
    { name: 'label', query: `@${labelSearchText}`, text: labelSearchText }
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

async function measureTaskEdit(page, measurements) {
  const originalTitle = taskTitle(activeRootTaskIndex(1));
  const editedTitle = `${originalTitle} Edited`;
  const taskID = await taskIDFromSearch(page, originalTitle);
  const version = await taskVersion(page, taskID);
  const started = Date.now();
  const response = await appFormRequest(page, 'PATCH', `/tasks/${taskID}`, {
    expected_version: String(version),
    title: editedTitle,
    description: 'synthetic performance edit',
    status: 'needs-action'
  }, { tabID: 'performance-task-edit' });
  expect(response.status()).toBe(200);
  record(measurements, 'task_edit', 'write_through_patch', Date.now() - started, targets.taskEditMs);

  await gotoApp(page, `/search?q=${encodeURIComponent(editedTitle)}`);
  await expect(page.locator('[data-search-results]').filter({ hasText: editedTitle }).first()).toBeVisible();
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

  await waitForSyncIdle(page, state, targets.syncCompleteMs);
  record(measurements, 'manual_sync', 'completed', Date.now() - syncStarted, targets.syncCompleteMs);
}

async function measureConflictScenario(page, measurements, state) {
  if (dataset.conflicts === 0) {
    return;
  }

  const projects = performanceProjects();
  for (let offset = 0; offset < dataset.conflicts; offset += 1) {
    const index = activeRootTaskIndex(offset + 2);
    const uid = taskUID(index);
    const project = projectForTask(index, projects);
    const href = taskHref(project, uid);
    const taskID = await taskIDFromSearch(page, taskTitle(index));
    const version = await taskVersion(page, taskID);
    await deleteRemoteTask(href);
    const localResponse = await appFormRequest(page, 'PATCH', `/tasks/${taskID}`, {
      expected_version: String(version),
      title: `Perf Local Conflict ${String(offset + 1).padStart(2, '0')}`,
      status: 'needs-action'
    }, { tabID: `performance-conflict-${offset + 1}` });
    expect(localResponse.status()).toBe(502);
    await createRemoteTask({
      calendarHref: project.href,
      href,
      uid,
      rawVTODO: syntheticVTODO(index, uid, {
        title: `Perf Remote Conflict ${String(offset + 1).padStart(2, '0')}`
      })
    });
  }

  const syncStarted = Date.now();
  await manualSync(page);
  await waitForSyncIdle(page, state, targets.conflictSyncMs);
  record(measurements, 'fullscan_conflict', 'manual_sync', Date.now() - syncStarted, targets.conflictSyncMs);

  const navigationStarted = Date.now();
  await gotoApp(page, '/conflicts');
  await expect(page.locator('[data-conflict-list-row]')).toHaveCount(dataset.conflicts);
  record(measurements, 'navigation', 'conflicts_with_open_conflicts', Date.now() - navigationStarted, targets.navigationMs);
}

async function waitForSyncIdle(page, state, timeoutMs) {
  await expect.poll(async () => {
    const response = await page.request.get(appURL('/sync/status'), {
      headers: { [state.proxyUserHeader]: 'e2e-user' },
      failOnStatusCode: false
    });
    if (response.status() !== 200) return '';
    return response.text();
  }, { timeout: timeoutMs + 5_000 }).toMatch(/Status: idle[\s\S]*Letzter Sync: (?!nie)/);
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

  const rootTasks = rootTaskCount();
  for (let index = 1; index <= rootTasks; index += 1) {
    const project = projectForTask(index, projects);
    const uid = taskUID(index);
    await createRemoteTask({
      calendarHref: project.href,
      href: taskHref(project, uid),
      uid,
      rawVTODO: syntheticVTODO(index, uid, {
        completed: isCompletedTask(index)
      })
    });
  }

  for (let subtask = 1; subtask <= dataset.subtasks; subtask += 1) {
    const index = rootTasks + subtask;
    const parentIndex = ((subtask - 1) % rootTasks) + 1;
    const project = projectForTask(parentIndex, projects);
    const uid = taskUID(index);
    await createRemoteTask({
      calendarHref: project.href,
      href: taskHref(project, uid),
      uid,
      rawVTODO: syntheticVTODO(index, uid, {
        completed: isCompletedTask(index),
        parentUID: taskUID(parentIndex)
      })
    });
  }
}

function performanceProjects() {
  const projects = [{ href: '/cal/work/', displayName: 'Work' }];
  for (let index = 2; index <= dataset.projects; index += 1) {
    projects.push({
      href: `/cal/perf-project-${String(index).padStart(2, '0')}/`,
      displayName: projectName(index)
    });
  }
  return projects;
}

function syntheticVTODO(index, uid, options = {}) {
  const labels = Array.from(new Set([
    labelName(((index - 1) % dataset.labels) + 1),
    labelName((index % dataset.labels) + 1)
  ]));
  const priority = (index % 3) + 1;
  const dueDay = String((index % 28) + 1).padStart(2, '0');
  const lines = [
    'BEGIN:VCALENDAR',
    'VERSION:2.0',
    'PRODID:-//Caldo E2E Performance//EN',
    'BEGIN:VTODO',
    `UID:${uid}`,
    `SUMMARY:${options.title || taskTitle(index)}`,
    `DUE;VALUE=DATE:202607${dueDay}`,
    `PRIORITY:${priority}`,
    `CATEGORIES:${labels.join(',')}`
  ];
  if (options.parentUID) {
    lines.push(`RELATED-TO;RELTYPE=PARENT:${options.parentUID}`);
  }
  if (options.completed) {
    lines.push('STATUS:COMPLETED', 'COMPLETED:20260701T120000Z', 'PERCENT-COMPLETE:100');
  } else {
    lines.push('STATUS:NEEDS-ACTION');
  }
  lines.push('END:VTODO', 'END:VCALENDAR', '');
  return lines.join('\r\n');
}

function taskTitle(index) {
  return `Perf Task ${String(index).padStart(4, '0')}`;
}

function taskUID(index) {
  return `perf-${String(index).padStart(4, '0')}`;
}

function taskHref(project, uid) {
  return `${project.href}${uid}.ics`;
}

function projectForTask(index, projects) {
  return projects[(index - 1) % projects.length];
}

function projectName(index) {
  if (index <= 1) {
    return 'Work';
  }
  return `Perf Project ${String(index).padStart(2, '0')}`;
}

function labelName(index) {
  return `perflabel${String(index).padStart(2, '0')}`;
}

function rootTaskCount() {
  return dataset.tasks - dataset.subtasks;
}

function activeRootCount() {
  return rootTaskCount() - Math.max(0, dataset.completedTasks - dataset.subtasks);
}

function activeRootTaskIndex(preferred) {
  return Math.max(1, Math.min(activeRootCount(), preferred));
}

function isCompletedTask(index) {
  return dataset.completedTasks > 0 && index > dataset.tasks - dataset.completedTasks;
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

function normalizeDataset(input) {
  const tasks = Math.max(1, input.tasks);
  const subtasks = Math.min(input.subtasks, Math.max(0, tasks - 1));
  const completedTasks = Math.min(input.completedTasks, Math.max(0, tasks - 1));
  const rootTasks = tasks - subtasks;
  const activeRoots = rootTasks - Math.max(0, completedTasks - subtasks);
  return {
    projects: Math.max(1, input.projects),
    tasks,
    labels: Math.max(1, input.labels),
    completedTasks,
    subtasks,
    conflicts: Math.min(input.conflicts, Math.max(0, activeRoots - 1))
  };
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

function goVersion() {
  try {
    return require('node:child_process')
      .execFileSync('go', ['version'], { cwd: repoRoot, encoding: 'utf8' })
      .trim();
  } catch {
    return '';
  }
}
