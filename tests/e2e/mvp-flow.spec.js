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
  await expect(page.getByRole('heading', { name: 'CalDAV einrichten' }).first()).toBeVisible();
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
  await expect(page.getByRole('heading', { name: 'Kalender auswählen' }).first()).toBeVisible();
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
  const projectCreateForm = page.locator('[data-project-create-form]');
  await expect(projectCreateForm).toBeVisible();
  await ensureBrowserCSRFCookie(page);
  await projectCreateForm.locator('[name="display_name"]').fill('E2E Empty Project');
  await projectCreateForm.getByRole('button', { name: 'Projekt anlegen' }).click();
  await expect(page.locator('[data-navigation-overview]').filter({ hasText: 'E2E Empty Project' })).toBeVisible();
  await expect.poll(async () => {
    const remoteState = await stageState();
    return remoteState.calendars.some((calendar) => calendar.display_name === 'E2E Empty Project');
  }).toBe(true);
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
  const mainInlineCreate = page.locator('.caldo-inline-create').first();
  const inlineTrigger = mainInlineCreate.locator('[data-inline-task-create-trigger]');
  await expect(inlineTrigger).toBeVisible();
  await ensureBrowserCSRFCookie(page);
  await inlineTrigger.click();
  const inlineTitle = mainInlineCreate.locator('[data-inline-task-create-title]');
  await expect(inlineTitle).toBeFocused();
  await inlineTitle.fill('E2E Inline Canceled');
  await inlineTitle.press('Escape');
  await expect(inlineTitle).toBeHidden();
  await inlineTrigger.click();
  await expect(inlineTitle).toHaveValue('');
  await inlineTitle.fill('E2E Inline Created');
  await inlineTitle.press('Enter');
  await expect.poll(async () => page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Created' }).count()).toBe(1);
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Created' }).first()).toBeVisible();

  await gotoApp(page, '/search?q=%23Work');
  let inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Created' }).first();
  let inlineEditForm = inlineEditRow.locator('[data-inline-task-edit-form]');
  await expect(inlineEditForm).toBeHidden();
  await inlineEditRow.getByRole('button', { name: 'Bearbeiten' }).click();
  inlineEditForm = inlineEditRow.locator('[data-inline-task-edit-form]');
  await expect(inlineEditForm).toBeVisible();
  await expect(inlineEditForm.locator('[name="title"]')).toBeFocused();
  await inlineEditForm.locator('[name="title"]').fill('E2E Inline Edit Canceled');
  await inlineEditForm.getByRole('button', { name: 'Abbrechen' }).click();
  await expect(inlineEditForm).toBeHidden();
  await expect(inlineEditRow).toContainText('E2E Inline Created');
  await expect(inlineEditRow).not.toContainText('E2E Inline Edit Canceled');

  await inlineEditRow.getByRole('button', { name: 'Bearbeiten' }).click();
  inlineEditForm = inlineEditRow.locator('[data-inline-task-edit-form]');
  await inlineEditForm.locator('[name="title"]').fill('E2E Inline Edited');
  await inlineEditForm.locator('[name="description"]').fill('edited inline through browser');
  await inlineEditForm.locator('[name="due_date"]').fill('2099-06-12');
  await inlineEditForm.locator('[name="priority"]').selectOption('5');
  await inlineEditForm.locator('[name="labels"]').fill('browser, inline');
  await inlineEditForm.getByRole('button', { name: 'Speichern' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first()).toBeVisible();
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toContainText('edited inline through browser');
  await expect(inlineEditRow).toContainText('Fällig 2099-06-12');
  await expect(inlineEditRow).toContainText('P2');
  await expect(inlineEditRow).toContainText('browser');
  await expect(inlineEditRow).toContainText('inline');
  await ensureBrowserCSRFCookie(page);
  await expect(inlineEditRow.getByRole('button', { name: 'Favorit setzen' })).toHaveAttribute('aria-pressed', 'false');
  await inlineEditRow.getByRole('button', { name: 'Favorit setzen' }).click();
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow.getByRole('button', { name: 'Favorit entfernen' })).toHaveAttribute('aria-pressed', 'true');
  await gotoApp(page, '/favorites');
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toBeVisible();
  await inlineEditRow.getByRole('button', { name: 'Favorit entfernen' }).click();
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' })).toHaveCount(0);
  await gotoApp(page, '/search?q=%23Work');
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow.getByRole('button', { name: 'Favorit setzen' })).toHaveAttribute('aria-pressed', 'false');

  let detailDialog = inlineEditRow.locator('[data-task-detail-dialog]');
  await expect(detailDialog).toBeHidden();
  await inlineEditRow.getByRole('button', { name: 'Details' }).click();
  await expect(detailDialog).toBeVisible();
  await expect(detailDialog.locator('[name="title"]')).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(detailDialog).toBeHidden();

  await inlineEditRow.getByRole('button', { name: 'Details' }).click();
  detailDialog = inlineEditRow.locator('[data-task-detail-dialog]');
  await detailDialog.locator('[name="title"]').fill('E2E Panel Edited');
  await detailDialog.locator('[name="description"]').fill('edited through task detail panel');
  await detailDialog.locator('[name="due_date"]').fill('2099-06-11');
  await detailDialog.locator('[name="priority"]').selectOption('1');
  await detailDialog.locator('[name="labels"]').fill('panel, browser');
  await detailDialog.locator('[name="repeat_freq"]').selectOption('DAILY');
  await detailDialog.getByRole('button', { name: 'Speichern' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Panel Edited' }).first()).toBeVisible();
  let detailRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Panel Edited' }).first();
  await expect(detailRow).toContainText('edited through task detail panel');
  await expect(detailRow).toContainText('Fällig 2099-06-11');
  await expect(detailRow).toContainText('P1');
  await expect(detailRow).toContainText('panel');
  await expect(detailRow).toContainText('browser');

  const panelTaskID = await detailRow.getAttribute('data-task-id');
  await ensureBrowserCSRFCookie(page);
  await detailRow.getByRole('button', { name: 'Unteraufgabe hinzufügen' }).click();
  const subtaskForm = detailRow.locator('[data-subtask-create-form]');
  await expect(subtaskForm).toBeVisible();
  const subtaskTitle = subtaskForm.locator('[data-inline-task-create-title]');
  await expect(subtaskTitle).toBeFocused();
  await subtaskTitle.fill('E2E Browser Subtask');
  await subtaskTitle.press('Enter');
  const subtaskRow = page.locator('[data-task-id].caldo-task-row-subtask').filter({ hasText: 'E2E Browser Subtask' }).first();
  await expect(subtaskRow).toBeVisible();
  await expect(subtaskRow).toContainText('Unteraufgabe von E2E Panel Edited');
  await expect(subtaskRow.locator('.caldo-task-checkbox[aria-label="Aufgabe erledigen"]')).toBeVisible();
  await expect(subtaskRow.getByRole('button', { name: 'Unteraufgabe hinzufügen' })).toHaveCount(0);
  detailRow = page.locator(`[data-task-id="${panelTaskID}"]`);
  await expect(detailRow).toContainText('1 Unteraufgabe');
  let completeDialog = detailRow.locator('[data-task-complete-dialog]');
  await expect(completeDialog).toBeHidden();
  await detailRow.locator('[data-task-complete-open]').first().click();
  await expect(completeDialog).toBeVisible();
  await expect(completeDialog).toContainText('1 offene Unteraufgabe');
  await expect(completeDialog.getByRole('button', { name: 'Nur Elternaufgabe erledigen' })).toBeVisible();
  await expect(completeDialog.getByRole('button', { name: 'Aufgabe und 1 offene Unteraufgabe erledigen' })).toBeVisible();
  await completeDialog.getByRole('button', { name: 'Abbrechen', exact: true }).click();
  await expect(completeDialog).toBeHidden();

  await page.setViewportSize(mobileViewport);
  await detailRow.getByRole('button', { name: 'Details' }).click();
  detailDialog = detailRow.locator('[data-task-detail-dialog]');
  await expect(detailDialog).toBeVisible();
  const panelBox = await detailDialog.boundingBox();
  expect(panelBox.x).toBeGreaterThanOrEqual(0);
  expect(panelBox.y).toBeGreaterThanOrEqual(0);
  expect(panelBox.x + panelBox.width).toBeLessThanOrEqual(mobileViewport.width);
  expect(panelBox.y + panelBox.height).toBeLessThanOrEqual(mobileViewport.height);
  await detailDialog.getByRole('button', { name: 'Details schließen' }).click();
  await expect(detailDialog).toBeHidden();
  await page.setViewportSize(desktopViewport);

  await gotoApp(page, '/search?q=E2E%20Panel%20Edited');
  detailRow = page.locator(`[data-task-id="${panelTaskID}"]`);
  const parentDeleteDialog = detailRow.locator('[data-task-delete-dialog]');
  await detailRow.locator('[data-task-delete-open]').click();
  await expect(parentDeleteDialog).toBeVisible();
  await expect(parentDeleteDialog).toContainText('1 Unteraufgabe');
  await parentDeleteDialog.getByRole('button', { name: 'Aufgabe und Unteraufgaben löschen' }).click();
  await waitForNoSearchResult(page, 'E2E Panel Edited');
  await gotoApp(page, '/search?q=E2E%20Browser%20Subtask');
  await expectNoSearchResult(page, 'E2E Browser Subtask');

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

  await gotoApp(page, '/search?q=E2E%20Local%20Edited');
  await ensureBrowserCSRFCookie(page);
  let deleteRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Local Edited' }).first();
  let deleteDialog = deleteRow.locator('[data-task-delete-dialog]');
  await expect(deleteDialog).toBeHidden();
  await deleteRow.locator('[data-task-delete-open]').click();
  await expect(deleteDialog).toBeVisible();
  await expect(deleteDialog.locator('[data-task-delete-cancel]')).toBeFocused();
  await deleteDialog.locator('[data-task-delete-cancel]').click();
  await expect(deleteDialog).toBeHidden();
  await expect(deleteRow).toBeVisible();

  await deleteRow.locator('[data-task-delete-open]').click();
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Endgültig löschen' }).click();
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Local Edited' })).toHaveCount(0);
  const undoNotifications = page.locator('#notifications');
  await expect(undoNotifications.getByRole('button', { name: 'Rückgängig' })).toBeVisible();
  await expect(undoNotifications).toContainText('Noch');
  await page.reload({ waitUntil: 'domcontentloaded' });
  await expect(undoNotifications.getByRole('button', { name: 'Rückgängig' })).toBeVisible();
  await expect(undoNotifications).toContainText('Noch');
  await page.screenshot({ path: 'test-results/e2e/undo-available.png', fullPage: true });
  const secondPage = await page.context().newPage();
  await gotoApp(secondPage, '/search?q=E2E%20Local%20Edited');
  await expect(secondPage.locator('#notifications').getByRole('button', { name: 'Rückgängig' })).toHaveCount(0);
  await secondPage.close();
  await undoNotifications.getByRole('button', { name: 'Rückgängig' }).click();
  await expect(undoNotifications).toContainText('Rückgängig ausgeführt.');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Local Edited' }).first()).toBeVisible();

  deleteRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Local Edited' }).first();
  deleteDialog = deleteRow.locator('[data-task-delete-dialog]');
  await deleteRow.locator('[data-task-delete-open]').click();
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Endgültig löschen' }).click();
  await waitForNoSearchResult(page, 'E2E Local Edited');
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
