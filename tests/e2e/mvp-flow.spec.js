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

const desktopViewport = { width: 1440, height: 1000 };
const tabletViewport = { width: 834, height: 1112 };
const mobileViewport = { width: 390, height: 844 };

test('MVP setup, sync, write-through, and conflict flow works in a browser session', async ({ page }, testInfo) => {
  fs.mkdirSync(browserBaselineDir(testInfo), { recursive: true });
  const state = readState();
  let response;
  const browserErrors = [];
  page.on('console', (message) => {
    if (message.type() === 'error') {
      browserErrors.push(message.text());
    }
  });
  page.on('pageerror', (error) => {
    browserErrors.push(error.message);
  });

  const health = await page.request.get(appURL('/health'));
  expect(health.status()).toBe(200);

  await gotoApp(page, '/');
  await expect(page).toHaveURL(/\/setup$/);
  await expect(page.getByRole('heading', { name: 'CalDAV einrichten' }).first()).toBeVisible();
  await captureBaselineSet(page, testInfo, 'setup');
  await page.setViewportSize(desktopViewport);

  await ensureBrowserCSRFCookie(page);
  await page.locator('[name="caldav_url"]').fill(state.stageBaseURL);
  await page.locator('[name="caldav_username"]').fill(state.stageUsername);
  await page.locator('[name="caldav_password"]').fill(state.stagePassword);
  await page.getByRole('button', { name: 'Verbindung testen' }).click();
  await expect(page).toHaveURL(/\/setup\/calendars$/);
  await expect(page.getByRole('heading', { name: 'Kalender auswählen' }).first()).toBeVisible();
  await expect(page.getByText('Work')).toBeVisible();

  await ensureBrowserCSRFCookie(page);
  await page.locator('[name="new_default_project_name"]').fill('E2E Setup Inbox');
  await page.getByRole('button', { name: 'Weiter zum Import' }).click();
  await expect.poll(async () => {
    if (await page.locator('[data-setup-import]').count()) {
      return 'import';
    }
    return new URL(page.url()).pathname === '/' ? 'complete' : 'pending';
  }, { timeout: 30_000 }).not.toBe('pending');
  await expect(page).toHaveURL(/\/$/, { timeout: 30_000 });
  await expect.poll(async () => {
    const remoteState = await stageState();
    return remoteState.calendars.some((calendar) => calendar.display_name === 'E2E Setup Inbox');
  }).toBe(true);

  await gotoApp(page, '/search?q=Stage');
  await expectCurrentView(page, 'Suche');
  await expect(page.locator('[data-search-results]').filter({ hasText: 'Stage Seed Task' })).toBeVisible();
  await exerciseKeyboardShortcuts(page);
  await exerciseQuickAddOverlay(page);
  await exerciseThemeToggle(page);
  await exerciseSSESyncStatus(page);
  await gotoApp(page, '/search?q=Stage');
  await expect(page.locator('[data-search-results]').filter({ hasText: 'Stage Seed Task' })).toBeVisible();

  await gotoApp(page, '/projects');
  await captureBaselineSet(page, testInfo, 'inbox-equivalent-default-project');
  await page.setViewportSize(desktopViewport);
  const projectCreateForm = page.locator('[data-project-create-form]');
  await expect(projectCreateForm).toBeVisible();
  await ensureBrowserCSRFCookie(page);
  await projectCreateForm.locator('[name="display_name"]').fill('E2E Empty Project');
  await projectCreateForm.getByRole('button', { name: 'Projekt anlegen' }).click();
  await expect(page.locator('[data-project-create-success]')).toContainText('projekt wurde angelegt');
  await expect(page.locator('[data-navigation-overview]').filter({ hasText: 'E2E Empty Project' })).toBeVisible();
  await expect.poll(async () => {
    const remoteState = await stageState();
    return remoteState.calendars.some((calendar) => calendar.display_name === 'E2E Empty Project');
  }).toBe(true);
  const emptyProjectNav = page.locator('.caldo-sidebar [data-nav-projects] a').filter({ hasText: 'E2E Empty Project' }).first();
  await emptyProjectNav.scrollIntoViewIfNeeded();
  await expect(emptyProjectNav).toBeVisible();
  await expect(emptyProjectNav.locator('.caldo-nav-count')).toHaveText('0');
  await emptyProjectNav.click();
  await expect(page).toHaveURL(/\/projects\/[^/]+$/);
  await expectCurrentView(page, 'E2E Empty Project');
  await expect(page.locator('.caldo-sidebar [data-nav-projects] a[aria-current="page"]').filter({ hasText: 'E2E Empty Project' })).toBeVisible();
  await expect(page.getByText('Keine offenen Aufgaben in diesem Projekt.')).toBeVisible();
  await gotoApp(page, '/search?q=%23Work');
  const searchSaveFilterForm = page.locator('[data-search-save-filter-form]');
  await expect(searchSaveFilterForm).toBeVisible();
  await searchSaveFilterForm.locator('[data-search-save-filter-name]').fill('E2E Work Filter');
  await searchSaveFilterForm.getByRole('button', { name: 'Filter anlegen' }).click();
  await expect(page.locator('[data-saved-filter-list]').filter({ hasText: 'E2E Work Filter' })).toBeVisible();
  await expect(page.locator('[data-nav-user-filters] a').filter({ hasText: 'E2E Work Filter' })).toHaveCount(2);
  await gotoApp(page, '/filters');
  const invalidFilterCreateForm = page.locator('[data-saved-filter-create-form]');
  await invalidFilterCreateForm.locator('[name="name"]').fill('E2E Broken Filter');
  await invalidFilterCreateForm.locator('[name="query"]').fill('today AND (');
  await invalidFilterCreateForm.locator('[name="favorite"]').check();
  await invalidFilterCreateForm.getByRole('button', { name: 'Filter anlegen' }).click();
  await expect(page.locator('[data-saved-filter-create-error]')).toContainText('filterquery ist ungültig');
  await expect(page.locator('[data-saved-filter-list]').filter({ hasText: 'E2E Broken Filter' })).toHaveCount(0);
  await expect(page.locator('[data-nav-user-filters] a').filter({ hasText: 'E2E Broken Filter' })).toHaveCount(0);
  await gotoApp(page, '/today');
  await captureBaselineSet(page, testInfo, 'today');
  await gotoApp(page, '/upcoming');
  await captureBaselineSet(page, testInfo, 'upcoming');
  await gotoApp(page, '/search?q=Stage');
  await captureBaselineSet(page, testInfo, 'search');
  await gotoApp(page, '/quick-add');
  await captureBaselineSet(page, testInfo, 'quick-add');
  await gotoApp(page, '/settings');
  await captureBaselineSet(page, testInfo, 'settings');
  await exerciseTabletCoreViews(page);
  await gotoApp(page, '/search?q=Stage');

  await page.setViewportSize(mobileViewport);
  await expect(page.getByRole('button', { name: 'Navigation öffnen' })).toBeVisible();
  await page.getByRole('button', { name: 'Navigation öffnen' }).click();
  await expect(page.getByRole('navigation', { name: 'Mobile Hauptnavigation' })).toBeVisible();
  await expect(page.locator('[data-mobile-nav-dialog] nav[aria-label="Mobile Hauptnavigation"] a[href="/today"]')).toBeVisible();
  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/search-mobile.png`, fullPage: true, caret: 'initial' });
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
  await inlineTitle.fill('E2E Inline Preserved');
  await exerciseWriteStatusForFailedInlineCreate(page, mainInlineCreate, inlineTitle);
  await inlineTitle.fill('E2E Inline Created');
  await inlineTitle.press('Enter');
  await expect.poll(async () => page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Created' }).count()).toBe(1);
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Created' }).first()).toBeVisible();
  await expect(page.locator('[data-write-status]')).toContainText('Gespeichert');

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
  await inlineEditForm.locator('[name="description"]').fill('edited inline through browser https://example.com/browser');
  await inlineEditForm.locator('[name="due_date"]').fill('2099-06-12');
  await inlineEditForm.locator('[name="priority"]').selectOption('5');
  await inlineEditForm.locator('[name="labels"]').fill('browser, inline');
  await expect(inlineEditForm.locator('[data-task-labels-input]')).toHaveValue('browser, inline');
  await inlineEditForm.getByRole('button', { name: 'Speichern' }).focus();
  await page.keyboard.press('Enter');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first()).toBeVisible();
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toContainText('edited inline through browser');
  await expect(inlineEditRow.locator('.caldo-task-description-link[href="https://example.com/browser"]')).toHaveAttribute('target', '_blank');
  await expect(inlineEditRow.locator('.caldo-task-description-link[href="https://example.com/browser"]')).toHaveAttribute('rel', 'noopener noreferrer');
  await expect(inlineEditRow).toContainText('Fällig 2099-06-12');
  await expect(inlineEditRow).toContainText('P2');
  await expect(inlineEditRow).toContainText('browser');
  await expect(inlineEditRow).toContainText('inline');
  await openLabelDetail(page, 'browser');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first()).toBeVisible();
  await gotoApp(page, '/search?q=%23Work');
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
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

  await dragTaskRowToProject(page, inlineEditRow, 'E2E Empty Project');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' })).toHaveCount(0);
  await gotoApp(page, '/search?q=%23E2E%20Empty%20Project');
  let movedProjectRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(movedProjectRow).toBeVisible();
  await expect(movedProjectRow).toContainText('E2E Empty Project');
  await dragTaskRowToProject(page, movedProjectRow, 'Work');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' })).toHaveCount(0);
  await gotoApp(page, '/search?q=%23Work');
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toBeVisible();

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
  await expect(detailDialog.locator('[data-task-labels-input]')).toHaveValue('panel, browser');
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
  await openLabelDetail(page, 'inline');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Panel Edited' })).toHaveCount(0);
  await expect(page.getByText('Keine Aufgaben mit diesem Label.')).toBeVisible();
  await openLabelDetail(page, 'browser');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Panel Edited' }).first()).toBeVisible();
  await gotoApp(page, '/search?q=%23Work');
  detailRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Panel Edited' }).first();
  await expect(detailRow).toBeVisible();

  const panelTaskID = await detailRow.getAttribute('data-task-id');
  await ensureBrowserCSRFCookie(page);
  await expect(detailRow.getByRole('button', { name: 'Unteraufgabe hinzufügen' })).toHaveCount(0);
  await openTaskRowEdit(detailRow);
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
  await expect(subtaskRow.locator('.caldo-task-checkbox-label[aria-label="Aufgabe erledigen"]')).toBeVisible();
  await expect(subtaskRow.getByRole('checkbox', { name: 'Aufgabe erledigen' })).toBeAttached();
  await expect(subtaskRow.getByRole('button', { name: 'Unteraufgabe hinzufügen' })).toHaveCount(0);
  detailRow = page.locator(`[data-task-id="${panelTaskID}"]`);
  await expect(detailRow).toContainText('1 Unteraufgabe');
  await exerciseTabletTaskActions(page, panelTaskID);
  detailRow = page.locator(`[data-task-id="${panelTaskID}"]`);
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
  await expect(detailRow.getByRole('button', { name: 'Löschen' })).toHaveCount(0);
  await openTaskRowEdit(detailRow);
  await detailRow.getByRole('button', { name: 'Löschen' }).click();
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
  response = await appFormRequest(page, 'PATCH', `/tasks/${createdID}`, {
    expected_version: String(version),
    title: 'E2E Focus Refreshed',
    description: 'refreshed after browser focus',
    status: 'needs-action'
  }, { tabID: 'e2e-background-refresh' });
  expect(response.status()).toBe(200);
  await page.waitForTimeout(600);
  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  let focusRow = page.locator(`[data-task-id="${createdID}"]`);
  await expect(focusRow).toContainText('E2E Focus Refreshed');
  await expect(focusRow).toContainText('refreshed after browser focus');

  await focusRow.getByRole('button', { name: 'Bearbeiten' }).click();
  let focusEditForm = focusRow.locator('[data-inline-task-edit-form]');
  await focusEditForm.locator('[name="title"]').fill('Unsaved local focus edit');
  version = await taskVersion(page, createdID);
  response = await appFormRequest(page, 'PATCH', `/tasks/${createdID}`, {
    expected_version: String(version),
    title: 'E2E Focus Dirty Remote',
    description: 'remote changed while local form was dirty',
    status: 'needs-action'
  }, { tabID: 'e2e-background-dirty' });
  expect(response.status()).toBe(200);
  await page.waitForTimeout(600);
  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(focusEditForm.locator('[name="title"]')).toHaveValue('Unsaved local focus edit');
  await expect(focusRow).toContainText('Aufgabe wurde in einem anderen Tab geändert');
  await focusEditForm.getByRole('button', { name: 'Abbrechen' }).click();
  await page.waitForTimeout(600);
  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(focusRow).toContainText('E2E Focus Dirty Remote');
  await expect(focusRow).toContainText('remote changed while local form was dirty');
  const currentLocalTitle = 'E2E Focus Dirty Remote';

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

  await gotoApp(page, `/search?q=${encodeURIComponent(currentLocalTitle)}`);
  await ensureBrowserCSRFCookie(page);
  let deleteRow = page.locator('[data-task-id]').filter({ hasText: currentLocalTitle }).first();
  let deleteDialog = deleteRow.locator('[data-task-delete-dialog]');
  await expect(deleteDialog).toBeHidden();
  await expect(deleteRow.getByRole('button', { name: 'Löschen' })).toHaveCount(0);
  await openTaskRowEdit(deleteRow);
  await deleteRow.getByRole('button', { name: 'Löschen' }).click();
  await expect(deleteDialog).toBeVisible();
  await expect(deleteDialog.locator('[data-task-delete-cancel]')).toBeFocused();
  await deleteDialog.locator('[data-task-delete-cancel]').click();
  await expect(deleteDialog).toBeHidden();
  await expect(deleteRow).toBeVisible();

  await deleteRow.getByRole('button', { name: 'Löschen' }).click();
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Endgültig löschen' }).click();
  await expect(page.locator('[data-task-id]').filter({ hasText: currentLocalTitle })).toHaveCount(0);
  const undoNotifications = page.locator('#notifications');
  await expect(undoNotifications.getByRole('button', { name: 'Rückgängig' })).toBeVisible();
  await expect(undoNotifications).toContainText('Noch');
  await page.reload({ waitUntil: 'domcontentloaded' });
  await expect(undoNotifications.getByRole('button', { name: 'Rückgängig' })).toBeVisible();
  await expect(undoNotifications).toContainText('Noch');
  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/undo-available.png`, fullPage: true, caret: 'initial' });
  const secondPage = await page.context().newPage();
  await gotoApp(secondPage, `/search?q=${encodeURIComponent(currentLocalTitle)}`);
  await expect(secondPage.locator('#notifications').getByRole('button', { name: 'Rückgängig' })).toHaveCount(0);
  await secondPage.close();
  await undoNotifications.getByRole('button', { name: 'Rückgängig' }).click();
  await expect(undoNotifications).toContainText('Rückgängig ausgeführt.');
  await expect(page.locator('[data-task-id]').filter({ hasText: currentLocalTitle }).first()).toBeVisible();

  deleteRow = page.locator('[data-task-id]').filter({ hasText: currentLocalTitle }).first();
  deleteDialog = deleteRow.locator('[data-task-delete-dialog]');
  await openTaskRowEdit(deleteRow);
  await deleteRow.getByRole('button', { name: 'Löschen' }).click();
  await expect(deleteDialog).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Endgültig löschen' }).click();
  await waitForNoSearchResult(page, currentLocalTitle);
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
  await expect(page.locator('[data-conflict-list-summary]')).toContainText('offener Konflikt');
  await expect(page.locator('[data-conflict-list-row]').first()).toContainText('Feldkonflikt');
  await expect(page.locator('[data-conflict-next-action]').first()).toContainText('Felder vergleichen und Zielversion wählen');
  const conflictLink = page.locator('[data-conflict-list] a').first();
  await expect(conflictLink).toBeVisible();
  await captureBaselineSet(page, testInfo, 'conflicts');
  const conflictHref = await conflictLink.getAttribute('href');
  expect(conflictHref).toMatch(/^\/conflicts\//);

  await gotoApp(page, conflictHref);
  await expectCurrentView(page, 'Konfliktdetail');
  await expect(page.locator('[data-conflict-comparison]').getByText('E2E Remote Conflict Edit', { exact: true })).toBeVisible();
  await expect(page.locator('[data-conflict-field="project"]')).toHaveAttribute('data-conflict-row-state', 'unchanged');
  await expect(page.locator('[data-conflict-field="title"]')).toHaveAttribute('data-conflict-row-state', 'changed');
  await page.setViewportSize(mobileViewport);
  await expectNoHorizontalOverflow(page);
  await expect(page.locator('[data-conflict-comparison]')).toBeVisible();
  await expect(page.locator('[data-conflict-field="project"]')).toBeVisible();
  await page.setViewportSize(desktopViewport);
  const conflictSplitPreview = page.locator('[data-conflict-split-preview]');
  await expect(conflictSplitPreview).toContainText('Nach dem Split existieren zwei Aufgaben');
  await expect(conflictSplitPreview.locator('[data-conflict-split-target="local"]')).toContainText('E2E Local Conflict Edit');
  await expect(conflictSplitPreview.locator('[data-conflict-split-target="remote"]')).toContainText('E2E Remote Conflict Edit');
  await expect(conflictSplitPreview.locator('[data-conflict-split-submit]')).toBeDisabled();
  await conflictSplitPreview.locator('[data-conflict-split-confirm]').check();
  await expect(conflictSplitPreview.locator('[data-conflict-split-submit]')).toBeEnabled();
  await conflictSplitPreview.locator('[data-conflict-split-confirm]').uncheck();
  const conflictManualForm = page.locator('[data-conflict-manual-form]');
  await expect(conflictManualForm.locator('[data-conflict-field-source="title"] [data-conflict-source-option="local"]')).toContainText('E2E Local Conflict Edit');
  await expect(conflictManualForm.locator('[data-conflict-field-source="title"] [data-conflict-source-option="remote"]')).toContainText('E2E Remote Conflict Edit');
  await expect(conflictManualForm.locator('[name="title_source"][value="remote"]')).toBeChecked();
  await expect(conflictManualForm.locator('[data-conflict-field-source="title"] [data-conflict-source-option="manual"]')).toBeVisible();
  const conflictPreviewTitle = conflictManualForm.locator('[data-conflict-preview-value="title"]');
  await expect(conflictPreviewTitle).toContainText('E2E Remote Conflict Edit');
  await conflictManualForm.locator('[name="title_source"][value="local"]').check();
  await expect(conflictPreviewTitle).toContainText('E2E Local Conflict Edit');
  await conflictManualForm.locator('[name="title_source"][value="manual"]').check();
  await conflictManualForm.locator('[name="title_manual"]').fill('E2E Manual Conflict Resolution');
  await expect(conflictPreviewTitle).toContainText('E2E Manual Conflict Resolution');

  await Promise.all([
    page.waitForURL(/\/conflicts$/),
    conflictManualForm.getByRole('button', { name: 'Ausgewählte Felder speichern' }).click()
  ]);

  await gotoApp(page, '/conflicts');
  await expect(page.getByText('Keine ungelösten Konflikte')).toBeVisible();
  await expectSearchResult(page, 'E2E Manual Conflict Resolution');
  expect(browserErrors.filter((message) => !expectedBrowserConsoleError(message, testInfo.project.name))).toEqual([]);
});

function expectedBrowserConsoleError(message, projectName) {
  if (
    projectName === 'webkit' &&
    message === "Refused to apply a stylesheet because its hash, its nonce, or 'unsafe-inline' does not appear in the style-src directive of the Content Security Policy."
  ) {
    return true;
  }

  return [
    'Failed to load resource: the server responded with a status of 400 (Bad Request)',
    'Response Status Error Code 400 from /tasks',
    'Failed to load resource: the server responded with a status of 502 (Bad Gateway)',
    'Response Status Error Code 502 from /tasks/'
  ].includes(message);
}

async function dragTaskRowToProject(page, row, projectName) {
  const target = page.locator('.caldo-sidebar [data-nav-projects] [data-project-drop-target]').filter({ hasText: projectName }).first();
  await expect(row).toBeVisible();
  await expect(row).toHaveAttribute('draggable', 'true');
  await expect(target).toBeVisible();
  await row.dragTo(target);
}

async function expectCurrentView(page, title) {
  await expect(page.locator('.caldo-topbar-heading')).toHaveText(title);
}

async function openLabelDetail(page, labelName) {
  await gotoApp(page, '/labels');
  const link = page.locator('[data-navigation-overview] a[href^="/labels/"]').filter({ hasText: labelName }).first();
  await expect(link).toBeVisible();
  await link.click();
  await expectCurrentView(page, labelName);
  await expect(page).toHaveURL(/\/labels\/[^/]+$/);
}

async function exerciseTabletCoreViews(page) {
  await page.setViewportSize(tabletViewport);
  const coreViews = [
    { pathname: '/today', heading: 'Heute' },
    { pathname: '/upcoming', heading: 'Demnächst' },
    { pathname: '/projects', heading: 'Projekte' },
    { pathname: '/search?q=Stage', heading: 'Suche', content: 'Stage Seed Task' },
    { pathname: '/settings', heading: 'Einstellungen' }
  ];

  for (const view of coreViews) {
    await gotoApp(page, view.pathname);
    await expectCurrentView(page, view.heading);
    await expect(page.locator('main .caldo-page-title')).toHaveCount(0);
    if (view.content) {
      await expect(page.locator('main').filter({ hasText: view.content })).toBeVisible();
    }
    await expect(page.getByRole('navigation', { name: 'Hauptnavigation' })).toBeVisible();
    await expect(page.locator('.caldo-topbar a[href="/search"]')).toBeVisible();
    await expect(page.locator('.caldo-topbar a[href="/quick-add"]')).toBeVisible();
    await expect(page.locator('.caldo-topbar [data-theme-toggle]')).toBeVisible();
    await expect(page.locator('.caldo-topbar #sync-status')).toBeVisible();
    await expectNoHorizontalOverflow(page);
  }
}

async function exerciseTabletTaskActions(page, panelTaskID) {
  await page.setViewportSize(tabletViewport);
  await gotoApp(page, '/search?q=E2E%20Panel%20Edited');
  await expectNoHorizontalOverflow(page);
  await ensureBrowserCSRFCookie(page);

  let row = page.locator(`[data-task-id="${panelTaskID}"]`);
  await expect(row).toBeVisible();
  await row.getByRole('button', { name: 'Details' }).click();
  let detailDialog = row.locator('[data-task-detail-dialog]');
  await expect(detailDialog).toBeVisible();
  await expectElementWithinViewport(detailDialog, tabletViewport);
  await detailDialog.locator('[name="description"]').fill('edited through tablet detail panel');
  await detailDialog.getByRole('button', { name: 'Speichern' }).click();
  row = page.locator(`[data-task-id="${panelTaskID}"]`);
  await expect(row).toContainText('edited through tablet detail panel');
  await expectNoHorizontalOverflow(page);

  row = page.locator(`[data-task-id="${panelTaskID}"]`);
  const completeDialog = row.locator('[data-task-complete-dialog]');
  await row.locator('[data-task-complete-open]').first().click();
  await expect(completeDialog).toBeVisible();
  await expectElementWithinViewport(completeDialog, tabletViewport);
  await expect(completeDialog.getByRole('button', { name: 'Nur Elternaufgabe erledigen' })).toBeVisible();
  await expect(completeDialog.getByRole('button', { name: 'Aufgabe und 1 offene Unteraufgabe erledigen' })).toBeVisible();
  await completeDialog.getByRole('button', { name: 'Abbrechen', exact: true }).click();
  await expect(completeDialog).toBeHidden();

  row = page.locator(`[data-task-id="${panelTaskID}"]`);
  const deleteDialog = row.locator('[data-task-delete-dialog]');
  await expect(row.getByRole('button', { name: 'Löschen' })).toHaveCount(0);
  await openTaskRowEdit(row);
  await row.getByRole('button', { name: 'Löschen' }).click();
  await expect(deleteDialog).toBeVisible();
  await expectElementWithinViewport(deleteDialog, tabletViewport);
  await expect(deleteDialog.getByRole('button', { name: 'Aufgabe und Unteraufgaben löschen' })).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Abbrechen', exact: true }).click();
  await expect(deleteDialog).toBeHidden();
  await closeTaskRowEdit(row);
  await page.setViewportSize(desktopViewport);
}

async function openTaskRowEdit(row) {
  await row.getByRole('button', { name: 'Bearbeiten' }).click();
  await expect(row.locator('[data-inline-task-edit-form]')).toBeVisible();
  await expect(row.locator('[data-inline-task-edit-extra]')).toBeVisible();
}

async function closeTaskRowEdit(row) {
  await row.getByRole('button', { name: 'Abbrechen' }).click();
  await expect(row.locator('[data-inline-task-edit-form]')).toBeHidden();
  await expect(row.getByRole('button', { name: 'Details' })).toBeVisible();
}

async function exerciseKeyboardShortcuts(page) {
  await gotoApp(page, '/today');
  await page.keyboard.press('n');
  const quickAddOverlay = page.locator('[data-quick-add-overlay]');
  await expect(quickAddOverlay).toBeVisible();
  await expect(page).toHaveURL(/\/today$/);
  await expect(quickAddOverlay.locator('[data-quick-add-overlay-input]')).toBeFocused();
  await quickAddOverlay.getByRole('button', { name: 'Schließen' }).click();
  await expect(quickAddOverlay).toBeHidden();

  await gotoApp(page, '/today');
  await page.keyboard.press('s');
  await expect(page).toHaveURL(/\/search$/);
  await expectCurrentView(page, 'Suche');

  const searchInput = page.locator('#global-search');
  await expect(searchInput).toBeFocused();
  await searchInput.fill('Stage');
  await expect(page.locator('[data-search-results]').filter({ hasText: 'Stage Seed Task' })).toBeVisible();
  await expect(page).toHaveURL(/\/search$/);
  await searchInput.fill('typing');
  await page.keyboard.press('n');
  await expect(page).toHaveURL(/\/search$/);
  await expect(searchInput).toHaveValue('typingn');

  await gotoApp(page, '/settings');
  await page.keyboard.press('g');
  await page.keyboard.press('t');
  await expect(page).toHaveURL(/\/today$/);
  await page.keyboard.press('g');
  await page.keyboard.press('u');
  await expect(page).toHaveURL(/\/upcoming$/);
  await page.keyboard.press('g');
  await page.keyboard.press('p');
  await expect(page).toHaveURL(/\/projects$/);

  await page.keyboard.press('Shift+/');
  const helpDialog = page.locator('[data-shortcut-help-dialog]');
  await expect(helpDialog).toBeVisible();
  await expect(helpDialog).toContainText('Tastaturkürzel');
  await expect(helpDialog).toContainText('G');
  await helpDialog.getByRole('button', { name: 'Schließen' }).click();
  await expect(helpDialog).toBeHidden();
}

async function exerciseQuickAddOverlay(page) {
  await gotoApp(page, '/search?q=Stage');
  await ensureBrowserCSRFCookie(page);
  const searchURL = page.url();
  const overlay = page.locator('[data-quick-add-overlay]');
  const input = overlay.locator('[data-quick-add-overlay-input]');
  const previewForm = overlay.locator('[data-quick-add-overlay-form]');
  const searchShortcutTarget = page.locator('.caldo-topbar a[href="/search"]').first();

  await searchShortcutTarget.focus();
  await page.keyboard.press('n');
  await expect(overlay).toBeVisible();
  await expect(input).toBeFocused();
  await expect(page).toHaveURL(searchURL);
  await input.fill('E2E Overlay Canceled');
  await page.keyboard.press('Escape');
  await expect(overlay).toBeHidden();
  await expect(searchShortcutTarget).toBeFocused();
  await expect(page).toHaveURL(searchURL);

  await page.keyboard.press('n');
  await expect(input).toBeFocused();
  await page.keyboard.type('n');
  await expect(input).toHaveValue('n');
  await input.fill('E2E Overlay Failed');
  await page.keyboard.press('Enter');
  let saveForm = overlay.locator('[data-quick-add-overlay-save-form]');
  await expect(saveForm).toBeVisible();
  await expect(saveForm.locator('input[name="title"]')).toBeFocused();
  await saveForm.locator('input[name="title"]').fill('');
  await page.keyboard.press('Control+Enter');
  await expect(overlay).toBeVisible();
  await expect(overlay.locator('[data-quick-add-overlay-error]')).toContainText('Aufgabe konnte nicht gespeichert werden.');
  await expect(input).toHaveValue('E2E Overlay Failed');
  await expect(page).toHaveURL(searchURL);

  await input.fill('E2E Overlay Chips #Work @urgent morgen wöchentlich !2');
  await page.keyboard.press('Enter');
  saveForm = overlay.locator('[data-quick-add-overlay-save-form]');
  await expect(overlay.locator('[data-quick-add-chips]')).toContainText('Work');
  await expect(overlay.locator('[data-quick-add-chips]')).toContainText('urgent');
  await expect(overlay.locator('[data-quick-add-chips]')).toContainText('Wöchentlich');
  await expect(overlay.locator('[data-quick-add-chips]')).toContainText('P2 Mittel');
  await expect(saveForm.locator('input[name="title"]')).toHaveValue('E2E Overlay Chips');
  await expect(saveForm.locator('input[name="title"]')).toBeFocused();
  await expect(saveForm.locator('input[name="labels"]')).toHaveValue('urgent');
  await expect(saveForm.locator('input[name="due_date"]')).toHaveValue(/\d{4}-\d{2}-\d{2}/);
  await expect(saveForm.locator('input[name="recurrence"]')).toHaveValue('FREQ=WEEKLY');
  await expect(overlay.locator('[data-quick-add-date-resolution]')).toContainText('morgen');
  const firstChip = overlay.locator('[data-quick-add-chips] button').first();
  await firstChip.focus();
  await page.keyboard.press('ArrowRight');
  await expect(overlay.locator('[data-quick-add-chips] button').nth(1)).toBeFocused();
  const dateChip = overlay.locator(`[data-quick-add-chips] button[data-quick-add-clear="[name='due_date']"]`);
  await dateChip.focus();
  await page.keyboard.press('Enter');
  await expect(saveForm.locator('input[name="due_date"]')).toHaveValue('');
  await expect(dateChip).toBeHidden();
  await saveForm.locator('input[name="due_date"]').fill('2099-06-30');
  await expect(saveForm.locator('select[name="priority"]')).toHaveValue('medium');
  await expect(overlay.locator('[data-quick-add-chips]')).toContainText('Neu');
  const priorityChip = overlay.locator(`[data-quick-add-chips] button[data-quick-add-clear="[name='priority']"]`);
  await priorityChip.click();
  await expect(saveForm.locator('select[name="priority"]')).toHaveValue('');
  await expect(priorityChip).toBeHidden();
  const recurrenceInput = saveForm.locator('input[name="recurrence"]');
  await recurrenceInput.fill('COUNT=2');
  await saveForm.getByRole('button', { name: 'Speichern' }).click();
  await expect(overlay).toBeVisible();
  await expect(overlay.locator('[data-quick-add-recurrence-error]')).toBeVisible();
  await expect(overlay.locator('[data-quick-add-recurrence-error]')).toContainText('Wiederholung prüfen');
  await recurrenceInput.fill('FREQ=WEEKLY');
  await expect(overlay.locator('[data-quick-add-recurrence-error]')).toBeHidden();
  await saveForm.locator('input[name="title"]').fill('E2E Overlay Corrected');
  await saveForm.locator('input[name="labels"]').fill('reviewed');
  await saveForm.locator('select[name="priority"]').selectOption('low');
  await saveForm.locator('input[name="labels"]').focus();
  await page.keyboard.press('Control+Enter');
  await expect(overlay).toBeHidden();
  await expect(searchShortcutTarget).toBeFocused();
  await expect(page).toHaveURL(searchURL);
  await waitForSearchResult(page, 'E2E Overlay Corrected');
  const correctedRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Overlay Corrected' }).first();
  await expect(correctedRow).toContainText('reviewed');
  await expect(correctedRow).toContainText('P3');
  await expect(correctedRow).toContainText('Wöchentlich');
  await expect(correctedRow).toContainText('Fällig 2099-06-30');

  await page.locator('.caldo-topbar [data-quick-add-open]').click();
  await input.fill('E2E Overlay Suggested #Work');
  await previewForm.getByRole('button', { name: 'Vorschau' }).click();
  saveForm = overlay.locator('[data-quick-add-overlay-save-form]');
  const reviewedSuggestion = overlay.locator('[data-quick-add-label-suggestions] button[data-quick-add-append-label="reviewed"]');
  await expect(reviewedSuggestion).toBeVisible();
  await reviewedSuggestion.click();
  await expect(saveForm.locator('input[name="labels"]')).toHaveValue('reviewed');
  await saveForm.getByRole('button', { name: 'Speichern' }).click();
  await expect(overlay).toBeHidden();
  await waitForSearchResult(page, 'E2E Overlay Suggested');
  const suggestedRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Overlay Suggested' }).first();
  await expect(suggestedRow).toContainText('reviewed');

  await page.setViewportSize(mobileViewport);
  await gotoApp(page, '/search?q=Stage');
  await page.locator('.caldo-topbar [data-quick-add-open]').click();
  await expect(overlay).toBeVisible();
  await expectElementWithinViewport(overlay, mobileViewport);
  await overlay.getByRole('button', { name: 'Schließen' }).click();
  await expect(overlay).toBeHidden();
  await page.setViewportSize(desktopViewport);
}

async function exerciseThemeToggle(page) {
  await gotoApp(page, '/today');
  const root = page.locator('html');
  const toggle = page.locator('[data-theme-toggle]');

  await expect(root).toHaveAttribute('data-theme-mode', 'system');
  await expect(toggle).toContainText('Darstellung: System');

  await toggle.click();
  await expect(root).toHaveAttribute('data-theme-mode', 'dark');
  await expect.poll(async () => (await root.getAttribute('class')) || '').toBe('dark');
  await expect(toggle).toContainText('Darstellung: Dunkel');

  await toggle.click();
  await expect(root).toHaveAttribute('data-theme-mode', 'light');
  await expect.poll(async () => (await root.getAttribute('class')) || '').toBe('light');
  await expect(toggle).toContainText('Darstellung: Hell');

  await toggle.click();
  await expect(root).toHaveAttribute('data-theme-mode', 'system');
  await expect(root).toHaveAttribute('data-theme-effective', /^(dark|light)$/);
  await expect.poll(async () => (await root.getAttribute('class')) || '').toBe('');
  await expect(toggle).toContainText('Darstellung: System');
}

async function exerciseSSESyncStatus(page) {
  const state = readState();
  const eventsRoute = '**/events';
  await page.route(eventsRoute, async (route) => {
    await route.continue({
      headers: {
        ...route.request().headers(),
        [state.proxyUserHeader]: 'e2e-user'
      }
    });
  });

  await gotoApp(page, '/today');
  try {
    const syncStatus = page.locator('.caldo-topbar #sync-status');
    await expect(syncStatus).toBeVisible();
    const eventPromise = page.evaluate(() => new Promise((resolve, reject) => {
      window.__caldoSSEConnected = false;
      const source = new EventSource('/events');
      const timeout = window.setTimeout(() => {
        source.close();
        reject(new Error('timed out waiting for sync SSE event'));
      }, 10_000);
      source.addEventListener('connected', () => {
        window.__caldoSSEConnected = true;
      });
      source.addEventListener('app-event', (event) => {
        window.clearTimeout(timeout);
        source.close();
        resolve(JSON.parse(event.data));
      });
      source.onerror = () => {
        window.clearTimeout(timeout);
        source.close();
        reject(new Error('SSE connection failed'));
      };
    }));

    await expect.poll(() => page.evaluate(() => window.__caldoSSEConnected === true)).toBe(true);
    await ensureBrowserCSRFCookie(page);
    await syncStatus.getByRole('button', { name: 'Jetzt synchronisieren' }).click();
    const event = await eventPromise;
    expect(event).toMatchObject({ type: 'sync', resource: 'sync_status' });
    await expect.poll(async () => {
      const response = await page.request.get(appURL('/sync/status'), {
        headers: { [state.proxyUserHeader]: 'e2e-user' },
        failOnStatusCode: false
      });
      if (response.status() !== 200) return '';
      return response.text();
    }, { timeout: 30_000 }).toMatch(/Status: idle[\s\S]*Letzter Sync: (?!nie)/);
    await expect.poll(async () => (await syncStatus.textContent()) || '').toMatch(/Status: idle[\s\S]*Letzter Sync: (?!nie)/);
    await expect(page.locator('[data-write-status]')).not.toContainText('Gespeichert');
  } finally {
    await page.unroute(eventsRoute);
  }
}

async function exerciseWriteStatusForFailedInlineCreate(page, inlineCreateRoot, titleInput) {
  let resolveIntercepted;
  let releaseResponse;
  const requestIntercepted = new Promise((resolve) => {
    resolveIntercepted = resolve;
  });
  const responseGate = new Promise((resolve) => {
    releaseResponse = resolve;
  });
  const routePattern = '**/tasks/';
  const routeHandler = async (route) => {
    if (route.request().method() !== 'POST') {
      await route.continue();
      return;
    }
    resolveIntercepted();
    await responseGate;
    await route.fulfill({
      status: 502,
      headers: { 'content-type': 'text/plain; charset=utf-8' },
      body: 'failed to update task on caldav server'
    });
  };

  await page.route(routePattern, routeHandler);
  try {
    await titleInput.press('Enter');
    await requestIntercepted;
    await expect(page.locator('[data-write-status]')).toContainText('Speichern ...');
    await expect.poll(async () => browserWouldBlockUnload(page)).toBe(true);
    releaseResponse();
    await expect(inlineCreateRoot.locator('[data-inline-task-create-error]')).toBeVisible();
    await expect(inlineCreateRoot.locator('[data-inline-task-create-error]')).toContainText('Aufgabe konnte nicht gespeichert werden.');
    await expect(titleInput).toHaveValue('E2E Inline Preserved');
    await expect(page.locator('[data-write-status]')).toContainText('Speichern fehlgeschlagen');
    await expect.poll(async () => browserWouldBlockUnload(page)).toBe(false);
  } finally {
    await page.unroute(routePattern, routeHandler);
  }
}

async function browserWouldBlockUnload(page) {
  return page.evaluate(() => {
    const event = new Event('beforeunload', { cancelable: true });
    const dispatched = window.dispatchEvent(event);
    return !dispatched || event.defaultPrevented;
  });
}

function browserArtifactDir(testInfo) {
  return `test-results/e2e/${testInfo.project.name}`;
}

function browserBaselineDir(testInfo) {
  return `${browserArtifactDir(testInfo)}/baselines`;
}

async function captureBaselineSet(page, testInfo, name) {
  await captureBaseline(page, testInfo, `${name}-desktop`, desktopViewport);
  await captureBaseline(page, testInfo, `${name}-tablet`, tabletViewport);
  await captureBaseline(page, testInfo, `${name}-mobile`, mobileViewport);
}

async function captureBaseline(page, testInfo, name, viewport) {
  await page.setViewportSize(viewport);
  await page.screenshot({ path: `${browserBaselineDir(testInfo)}/${name}.png`, fullPage: true, caret: 'initial' });
}

async function expectNoHorizontalOverflow(page) {
  const overflow = await page.evaluate(() => {
    const root = document.documentElement;
    const body = document.body;
    const viewportWidth = root.clientWidth;
    return {
      viewportWidth,
      documentScrollWidth: root.scrollWidth,
      bodyScrollWidth: body ? body.scrollWidth : 0
    };
  });

  expect(
    overflow.documentScrollWidth,
    `document scroll width ${overflow.documentScrollWidth} must fit tablet viewport ${overflow.viewportWidth}`
  ).toBeLessThanOrEqual(overflow.viewportWidth + 1);
  expect(
    overflow.bodyScrollWidth,
    `body scroll width ${overflow.bodyScrollWidth} must fit tablet viewport ${overflow.viewportWidth}`
  ).toBeLessThanOrEqual(overflow.viewportWidth + 1);
}

async function expectElementWithinViewport(locator, viewport) {
  const box = await locator.boundingBox();
  if (!box) {
    throw new Error('expected visible element to have a bounding box');
  }
  expect(box.x).toBeGreaterThanOrEqual(-1);
  expect(box.y).toBeGreaterThanOrEqual(-1);
  expect(box.x + box.width).toBeLessThanOrEqual(viewport.width + 1);
  expect(box.y + box.height).toBeLessThanOrEqual(viewport.height + 1);
}
