const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
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
const { createRemoteTask, deleteRemoteTask, resetStage, updateRemoteTask } = require('./helpers/stage');

const enabled = process.env.CALDO_E2E_SYNC_LONGRUN === '1';
const cycles = envInt('CALDO_E2E_SYNC_LONGRUN_CYCLES', 3);
const cycleDelayMs = envInt('CALDO_E2E_SYNC_LONGRUN_CYCLE_DELAY_MS', 0);
const syncTimeoutMs = envInt('CALDO_E2E_SYNC_LONGRUN_SYNC_TIMEOUT_MS', 30_000);

test.describe('long-running sync validation', () => {
  test.skip(!enabled, 'set CALDO_E2E_SYNC_LONGRUN=1 or run npm run test:e2e:sync-longrun');
  test.setTimeout(Math.max(180_000, cycles * (syncTimeoutMs + cycleDelayMs + 15_000)));

  test('repeated local and remote changes remain stable across sync cycles', async ({ page }, testInfo) => {
    const state = readState();
    const startedAt = new Date();
    const report = {
      started_at: startedAt.toISOString(),
      finished_at: '',
      commit: process.env.GITHUB_SHA || currentCommit(),
      server: 'stagecaldav',
      build: 'playwright global setup using go run',
      cycles,
      cycle_delay_ms: cycleDelayMs,
      checks: [],
      result: 'running'
    };

    await resetStage();
    await completeSetup(page, state);

    const localTitles = [];
    const remoteTitles = [];
    const deletedRemoteTitles = [];

    for (let cycle = 1; cycle <= cycles; cycle += 1) {
      const padded = String(cycle).padStart(2, '0');

      const remoteUID = `longrun-remote-${padded}`;
      const remoteHref = `/cal/work/${remoteUID}.ics`;
      const remoteCreatedTitle = `Longrun Remote ${padded} Created`;
      const remoteUpdatedTitle = `Longrun Remote ${padded} Updated`;
      await createRemoteTask({ href: remoteHref, uid: remoteUID, title: remoteCreatedTitle });
      await syncAndWait(page, report, cycle, 'remote_create');
      await expectSingleSearchResult(page, remoteCreatedTitle);

      await updateRemoteTask({ href: remoteHref, uid: remoteUID, title: remoteUpdatedTitle });
      await syncAndWait(page, report, cycle, 'remote_update');
      await expectSingleSearchResult(page, remoteUpdatedTitle);
      await expectNoSearchResult(page, remoteCreatedTitle);
      remoteTitles.push(remoteUpdatedTitle);

      const localCreatedTitle = `Longrun Local ${padded} Created`;
      const localUpdatedTitle = `Longrun Local ${padded} Updated`;
      await createLocalTask(page, localCreatedTitle, `longrun-local-create-${padded}`);
      await expectSingleSearchResult(page, localCreatedTitle);
      const localTaskID = await taskIDFromSearch(page, localCreatedTitle);
      const version = await taskVersion(page, localTaskID);
      const response = await appFormRequest(page, 'PATCH', `/tasks/${localTaskID}`, {
        expected_version: String(version),
        title: localUpdatedTitle,
        description: `synthetic longrun cycle ${padded}`,
        status: 'needs-action'
      }, { tabID: `longrun-local-edit-${padded}` });
      expect(response.status()).toBe(200);
      await syncAndWait(page, report, cycle, 'local_update');
      await expectSingleSearchResult(page, localUpdatedTitle);
      await expectNoSearchResult(page, localCreatedTitle);
      localTitles.push(localUpdatedTitle);

      if (cycle > 1) {
        const deleteCycle = String(cycle - 1).padStart(2, '0');
        const deleteHref = `/cal/work/longrun-remote-${deleteCycle}.ics`;
        const deleteTitle = `Longrun Remote ${deleteCycle} Updated`;
        await deleteRemoteTask(deleteHref);
        await syncAndWait(page, report, cycle, 'remote_delete');
        await expectNoSearchResult(page, deleteTitle);
        deletedRemoteTitles.push(deleteTitle);
        removeValue(remoteTitles, deleteTitle);
      }

      await assertNoUnexpectedConflicts(page);
      await assertExpectedSearchState(page, localTitles, remoteTitles, deletedRemoteTitles);

      if (cycleDelayMs > 0 && cycle < cycles) {
        await page.waitForTimeout(cycleDelayMs);
      }
    }

    report.finished_at = new Date().toISOString();
    report.duration_seconds = Math.round((Date.parse(report.finished_at) - startedAt.getTime()) / 1000);
    report.result = 'pass';
    writeReport(testInfo, report);
  });
});

async function completeSetup(page, state) {
  await gotoApp(page, '/');
  await expect(page).toHaveURL(/\/setup$/);
  await ensureBrowserCSRFCookie(page);
  await page.locator('[name="caldav_url"]').fill(state.stageBaseURL);
  await page.locator('[name="caldav_username"]').fill(state.stageUsername);
  await page.locator('[name="caldav_password"]').fill(state.stagePassword);
  await page.getByRole('button', { name: 'Verbindung testen' }).click();
  await expect(page).toHaveURL(/\/setup\/calendars$/);
  await expect(page.getByText('Work')).toBeVisible();

  await ensureBrowserCSRFCookie(page);
  await page.getByRole('button', { name: 'Weiter zum Import' }).click();
  await expect(page.locator('[data-setup-import]')).toBeVisible();
  await expect(page).toHaveURL(/\/today$/, { timeout: 60_000 });
}

async function createLocalTask(page, title, tabID) {
  const response = await appFormRequest(page, 'POST', '/tasks/', {
    title
  }, { tabID });
  expect(response.status()).toBe(201);
}

async function syncAndWait(page, report, cycle, phase) {
  const started = Date.now();
  await manualSync(page);
  const status = await waitForSyncIdle(page);
  const milliseconds = Date.now() - started;
  const passed = /data-sync-state="idle"[\s\S]*Letzter erfolgreicher Sync: (?!nie)/.test(status);
  report.checks.push({ cycle, phase, milliseconds, sync_idle: passed });
  expect(passed, `sync did not return to idle after ${phase} cycle ${cycle}`).toBe(true);
}

async function waitForSyncIdle(page) {
  const state = readState();
  let latest = '';
  await expect.poll(async () => {
    const response = await page.request.get(appURL('/sync/status'), {
      headers: { [state.proxyUserHeader]: 'e2e-user' },
      failOnStatusCode: false
    });
    if (response.status() !== 200) {
      latest = '';
      return '';
    }
    latest = await response.text();
    return latest;
  }, { timeout: syncTimeoutMs }).toMatch(/data-sync-state="idle"[\s\S]*Letzter erfolgreicher Sync: (?!nie)/);
  return latest;
}

async function expectSingleSearchResult(page, title) {
  await gotoApp(page, `/search?q=${encodeURIComponent(title)}`);
  const rows = page.locator('[data-task-id]').filter({ hasText: title });
  await expect(rows).toHaveCount(1);
}

async function expectNoSearchResult(page, title) {
  await gotoApp(page, `/search?q=${encodeURIComponent(title)}`);
  await expect(page.locator('[data-task-id]').filter({ hasText: title })).toHaveCount(0);
}

async function assertExpectedSearchState(page, localTitles, remoteTitles, deletedRemoteTitles) {
  for (const title of localTitles) {
    await expectSingleSearchResult(page, title);
  }
  for (const title of remoteTitles) {
    await expectSingleSearchResult(page, title);
  }
  for (const title of deletedRemoteTitles) {
    await expectNoSearchResult(page, title);
  }
}

async function assertNoUnexpectedConflicts(page) {
  const state = readState();
  const response = await page.request.get(appURL('/conflicts'), {
    headers: { [state.proxyUserHeader]: 'e2e-user' },
    failOnStatusCode: false
  });
  expect(response.status()).toBe(200);
  const html = await response.text();
  expect(html).not.toContain('data-conflict-list-row');
}

function removeValue(values, target) {
  const index = values.indexOf(target);
  if (index >= 0) {
    values.splice(index, 1);
  }
}

function writeReport(testInfo, report) {
  const date = new Date().toISOString().slice(0, 10);
  const dir = path.join(repoRoot, 'test-results', 'sync-longrun', date);
  fs.mkdirSync(dir, { recursive: true });
  const browser = testInfo.project.name.replace(/[^a-z0-9_-]+/gi, '-').toLowerCase();
  fs.writeFileSync(path.join(dir, `playwright-${browser}.json`), `${JSON.stringify(report, null, 2)}\n`);
}

function envInt(name, fallback) {
  const raw = process.env[name];
  if (!raw) return fallback;
  const value = Number.parseInt(raw, 10);
  return Number.isFinite(value) && value >= 0 ? value : fallback;
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
