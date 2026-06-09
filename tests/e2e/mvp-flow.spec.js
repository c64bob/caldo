const { test, expect } = require('@playwright/test');
const fs = require('node:fs');
const {
  appFormRequest,
  ensureBrowserCSRFCookie,
  expectNoSearchResult,
  expectSearchResult,
  gotoApp,
  manualSync,
  taskIDFromSearch,
  taskVersion,
  waitForNoSearchResult,
  waitForSearchResult
} = require('./helpers/app');
const { appURL, readState } = require('./helpers/state');
const { createRemoteTask, deleteRemoteTask, stageState, updateRemoteTask } = require('./helpers/stage');

const baselineDir = 'test-results/e2e/baselines';
const desktopViewport = { width: 1440, height: 1000 };
const mobileViewport = { width: 390, height: 844 };

test('MVP setup, sync, write-through, and conflict flow works in a browser session', async ({ page }) => {
  fs.mkdirSync(baselineDir, { recursive: true });
  const state = readState();

  const health = await page.request.get(appURL('/health'));
  expect(health.status()).toBe(200);

  await gotoApp(page, '/');
  await expect(page).toHaveURL(/\/setup$/);
  await expect(page.getByRole('heading', { name: 'CalDAV einrichten' })).toBeVisible();
  await captureBaselineSet(page, 'setup');
  await page.setViewportSize(desktopViewport);

  let response = await appFormRequest(page, 'POST', '/setup/caldav', {
    caldav_url: state.stageBaseURL,
    caldav_username: state.stageUsername,
    caldav_password: state.stagePassword
  });
  expect(response.status()).toBe(302);
  expect(response.headers().location).toBe('/setup/calendars');

  await gotoApp(page, '/setup/calendars');
  await expect(page.getByRole('heading', { name: 'Kalender auswählen' })).toBeVisible();
  await expect(page.getByText('Work')).toBeVisible();

  response = await appFormRequest(page, 'POST', '/setup/calendars', {
    calendar_href: '/cal/work/',
    default_calendar_href: '/cal/work/',
    new_default_project_name: ''
  });
  expect(response.status()).toBe(302);
  expect(response.headers().location).toBe('/setup/import');

  response = await appFormRequest(page, 'POST', '/setup/import');
  expect(response.status()).toBe(202);

  await expect.poll(async () => {
    const complete = await appFormRequest(page, 'POST', '/setup/complete');
    return complete.status();
  }).toBe(302);

  await gotoApp(page, '/search?q=Stage');
  await expect(page.getByRole('heading', { name: 'Globale Suche' })).toBeVisible();
  await expect(page.locator('[data-search-results]').filter({ hasText: 'Stage Seed Task' })).toBeVisible();

  await gotoApp(page, '/projects');
  await captureBaselineSet(page, 'inbox-equivalent-default-project');
  await gotoApp(page, '/today');
  await captureBaselineSet(page, 'today');
  await gotoApp(page, '/search?q=Stage');
  await captureBaselineSet(page, 'search');
  await gotoApp(page, '/quick-add');
  await captureBaselineSet(page, 'quick-add');
  await gotoApp(page, '/settings');
  await captureBaselineSet(page, 'settings');
  await gotoApp(page, '/search?q=Stage');

  await page.setViewportSize(mobileViewport);
  await expect(page.getByRole('button', { name: 'Navigation öffnen' })).toBeVisible();
  await page.getByRole('button', { name: 'Navigation öffnen' }).click();
  await expect(page.getByRole('navigation', { name: 'Mobile Hauptnavigation' })).toBeVisible();
  await expect(page.locator('[data-mobile-nav-dialog] nav[aria-label="Mobile Hauptnavigation"] a[href="/today"]')).toBeVisible();
  await page.screenshot({ path: 'test-results/e2e/search-mobile.png', fullPage: true });
  await page.getByRole('button', { name: 'Schließen' }).click();
  await expect(page.locator('.caldo-topbar a[href="/search"]')).toBeVisible();
  await expect(page.locator('.caldo-topbar a[href="/quick-add"]')).toBeVisible();
  await page.setViewportSize(desktopViewport);

  await gotoApp(page, '/search?q=%23Work');
  const inlineTrigger = page.locator('[data-inline-task-create-trigger]');
  await expect(inlineTrigger).toBeVisible();
  await ensureBrowserCSRFCookie(page);
  await inlineTrigger.click();
  const inlineTitle = page.locator('[data-inline-task-create-title]');
  await expect(inlineTitle).toBeFocused();
  await inlineTitle.fill('E2E Inline Canceled');
  await inlineTitle.press('Escape');
  await expect(inlineTitle).toBeHidden();
  await inlineTrigger.click();
  await expect(inlineTitle).toHaveValue('');
  await inlineTitle.fill('E2E Inline Created');
  await inlineTitle.press('Enter');
  await expect.poll(async () => page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Created' }).count()).toBe(1);
  await waitForSearchResult(page, 'E2E Inline Created');

  const beforeCreate = await stageState();
  response = await appFormRequest(page, 'POST', '/tasks/', {
    title: 'E2E Local Created'
  }, { tabID: 'e2e-create' });
  expect(response.status()).toBe(201);
  const afterCreate = await stageState();
  const createdRemote = afterCreate.tasks.find((task) => !beforeCreate.tasks.some((existing) => existing.href === task.href));
  expect(createdRemote).toBeTruthy();

  const createdID = await taskIDFromSearch(page, 'E2E Local Created');
  let version = await taskVersion(page, createdID);
  response = await appFormRequest(page, 'PATCH', `/tasks/${createdID}`, {
    expected_version: String(version),
    title: 'E2E Local Edited',
    description: 'edited through browser qa',
    status: 'needs-action'
  }, { tabID: 'e2e-edit' });
  expect(response.status()).toBe(200);
  await expectSearchResult(page, 'E2E Local Edited');

  version = await taskVersion(page, createdID);
  response = await appFormRequest(page, 'POST', `/tasks/${createdID}/complete`, {
    expected_version: String(version)
  }, { tabID: 'e2e-complete' });
  expect(response.status()).toBe(200);

  version = await taskVersion(page, createdID);
  response = await appFormRequest(page, 'POST', `/tasks/${createdID}/reopen`, {
    expected_version: String(version)
  }, { tabID: 'e2e-reopen' });
  expect(response.status()).toBe(200);

  version = await taskVersion(page, createdID);
  response = await appFormRequest(page, 'DELETE', `/tasks/${createdID}`, {
    expected_version: String(version)
  }, { tabID: 'e2e-delete' });
  expect(response.status()).toBe(200);
  const afterDelete = await stageState();
  expect(afterDelete.tasks.some((task) => task.href === createdRemote.href)).toBe(false);

  await createRemoteTask({ uid: 'e2e-remote-sync', title: 'E2E Remote Created' });
  await manualSync(page);
  await waitForSearchResult(page, 'E2E Remote Created');

  await updateRemoteTask({
    href: '/cal/work/e2e-remote-sync.ics',
    uid: 'e2e-remote-sync',
    title: 'E2E Remote Updated'
  });
  await manualSync(page);
  await waitForSearchResult(page, 'E2E Remote Updated');

  await deleteRemoteTask('/cal/work/e2e-remote-sync.ics');
  await manualSync(page);
  await waitForNoSearchResult(page, 'E2E Remote Updated');

  const seedID = await taskIDFromSearch(page, 'Stage Seed Task');
  const seedVersion = await taskVersion(page, seedID);
  await deleteRemoteTask('/cal/work/stage-seed.ics');
  response = await appFormRequest(page, 'PATCH', `/tasks/${seedID}`, {
    expected_version: String(seedVersion),
    title: 'E2E Local Conflict Edit',
    status: 'needs-action'
  }, { tabID: 'e2e-local-dirty' });
  expect(response.status()).toBe(502);

  await createRemoteTask({
    href: '/cal/work/stage-seed.ics',
    uid: 'stage-seed',
    title: 'E2E Remote Conflict Edit'
  });
  await manualSync(page);

  await expect.poll(async () => {
    const conflicts = await page.request.get(appURL('/conflicts'), {
      headers: { [state.proxyUserHeader]: 'e2e-user' }
    });
    return conflicts.text();
  }).toContain('/conflicts/');

  await gotoApp(page, '/conflicts');
  const conflictLink = page.locator('[data-conflict-list] a').first();
  await expect(conflictLink).toBeVisible();
  await captureBaselineSet(page, 'conflicts');
  const conflictHref = await conflictLink.getAttribute('href');
  expect(conflictHref).toMatch(/^\/conflicts\//);

  await gotoApp(page, conflictHref);
  await expect(page.getByRole('heading', { name: 'Konfliktdetail' })).toBeVisible();
  await expect(page.getByText('E2E Remote Conflict Edit')).toBeVisible();

  response = await appFormRequest(page, 'POST', `${conflictHref}/resolve`, {
    resolution: 'remote'
  }, { tabID: 'e2e-resolve' });
  expect(response.status()).toBe(200);

  await gotoApp(page, '/conflicts');
  await expect(page.getByText('Keine ungelösten Konflikte')).toBeVisible();
  await expectSearchResult(page, 'E2E Remote Conflict Edit');
});

async function captureBaselineSet(page, name) {
  await captureBaseline(page, `${name}-desktop`, desktopViewport);
  await captureBaseline(page, `${name}-mobile`, mobileViewport);
}

async function captureBaseline(page, name, viewport) {
  await page.setViewportSize(viewport);
  await page.screenshot({ path: `${baselineDir}/${name}.png`, fullPage: true });
}
