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
const narrowMobileViewport = { width: 320, height: 720 };

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
  await exerciseTaskListDisplayPreferences(page);
  await exerciseKeyboardShortcuts(page);
  await exerciseQuickAddOverlay(page);
  await exerciseKeyboardFocusAccessibility(page, testInfo);
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
  const filterCreateError = page.locator('[data-saved-filter-create-error]');
  await expect(filterCreateError).toContainText('filterquery ist ungültig');
  await expect(filterCreateError).toHaveAttribute('role', 'alert');
  const filterCreateErrorID = await filterCreateError.getAttribute('id');
  expect(filterCreateErrorID).toBeTruthy();
  await expect(page.locator('[data-saved-filter-create-form]')).toHaveAttribute('aria-describedby', new RegExp(filterCreateErrorID));
  await expect(page.locator('[data-saved-filter-create-form] [name="name"]')).toHaveAttribute('aria-describedby', new RegExp(filterCreateErrorID));
  await expect(page.locator('[data-saved-filter-create-form] [name="name"]')).toHaveAttribute('aria-invalid', 'true');
  await expectVisibleFormErrorsAssociated(page);
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
  await exerciseTabletCoreViews(page, testInfo);
  await gotoApp(page, '/search?q=Stage');

  await exerciseMobileNavigationAndSettings(page, testInfo);

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
  let inlineEditForm = inlineEditRow.locator('[data-inline-task-edit-form][data-inline-task-edit-kind="title"]');
  await expect(inlineEditForm).toBeHidden();
  inlineEditForm = await openTaskRowEdit(inlineEditRow);
  await expect(inlineEditForm).toBeVisible();
  await expect(inlineEditForm.locator('[name="title"]')).toBeFocused();
  await inlineEditForm.locator('[name="title"]').fill('E2E Inline Edit Canceled');
  await inlineEditForm.locator('[name="title"]').press('Escape');
  await expect(inlineEditForm).toBeHidden();
  await expect(inlineEditRow).toContainText('E2E Inline Created');
  await expect(inlineEditRow).not.toContainText('E2E Inline Edit Canceled');

  inlineEditForm = await openTaskRowEdit(inlineEditRow);
  await inlineEditForm.locator('[name="title"]').fill('E2E Inline Edited');
  await inlineEditForm.locator('[name="title"]').press('Enter');
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first()).toBeVisible();
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await inlineEditRow.locator('.caldo-date-dropdown [data-date-hidden-input]').evaluate((input) => {
    input.value = '2099-06-12';
  });
  await inlineEditRow.locator('.caldo-date-dropdown form').evaluate((form) => form.requestSubmit());
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first()).toBeVisible();
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toContainText('Fällig 12.06.2099');

  let priorityEditForm = await openTaskRowEdit(inlineEditRow, 'priority');
  await expect(priorityEditForm.locator('[name="priority"]')).toBeFocused();
  await priorityEditForm.locator('[name="priority"]').press('Escape');
  await expect(priorityEditForm).toBeHidden();
  await expect(inlineEditRow).toContainText('Keine Priorität');
  priorityEditForm = await openTaskRowEdit(inlineEditRow, 'priority');
  await priorityEditForm.locator('[name="priority"]').selectOption('5');
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toContainText('P2 Mittel');

  let labelsEditForm = await openTaskRowEdit(inlineEditRow, 'labels');
  await expect(labelsEditForm.locator('[name="labels"]')).toBeFocused();
  await labelsEditForm.locator('[name="labels"]').fill('failed inline label');
  await exerciseWriteStatusForFailedInlineMetadata(page, inlineEditRow, labelsEditForm.locator('[name="labels"]'));
  await labelsEditForm.locator('[name="labels"]').fill('browser, inline');
  await labelsEditForm.locator('[name="labels"]').press('Enter');
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toContainText('browser');
  await expect(inlineEditRow).toContainText('inline');

  let projectEditForm = await openTaskRowEdit(inlineEditRow, 'project');
  await expect(projectEditForm.locator('[name="project_id"]')).toBeFocused();
  await projectEditForm.locator('[name="project_id"]').selectOption({ label: 'E2E Empty Project' });
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' })).toHaveCount(0);
  await gotoApp(page, '/search?q=%23E2E%20Empty%20Project');
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toBeVisible();
  await expect(inlineEditRow).toContainText('E2E Empty Project');
  projectEditForm = await openTaskRowEdit(inlineEditRow, 'project');
  await projectEditForm.locator('[name="project_id"]').selectOption({ label: 'Work' });
  await expect(page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' })).toHaveCount(0);
  await gotoApp(page, '/search?q=%23Work');
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toBeVisible();

  await inlineEditRow.getByRole('button', { name: 'Details' }).click();
  const inlineDetailDialog = inlineEditRow.locator('[data-task-detail-dialog]');
  await inlineDetailDialog.locator('[name="description"]').fill('edited inline through browser https://example.com/browser');
  await inlineDetailDialog.getByRole('button', { name: 'Speichern' }).focus();
  await page.keyboard.press('Enter');
  inlineEditRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Inline Edited' }).first();
  await expect(inlineEditRow).toContainText('edited inline through browser');
  await expect(inlineEditRow.locator('.caldo-task-description-link[href="https://example.com/browser"]')).toHaveAttribute('target', '_blank');
  await expect(inlineEditRow.locator('.caldo-task-description-link[href="https://example.com/browser"]')).toHaveAttribute('rel', 'noopener noreferrer');
  await expect(inlineEditRow).toContainText('Fällig 12.06.2099');
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
  await expect(detailDialog.locator('[data-task-detail-title]')).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(detailDialog).toBeHidden();

  await inlineEditRow.getByRole('button', { name: 'Details' }).click();
  detailDialog = inlineEditRow.locator('[data-task-detail-dialog]');
  await detailDialog.locator('[data-task-detail-title]').fill('E2E Panel Edited');
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
  await expect(detailRow).toContainText('Fällig 11.06.2099');
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
  await detailRow.getByRole('button', { name: 'Details' }).click();
  let actionDetailDialog = detailRow.locator('[data-task-detail-dialog]');
  await expect(actionDetailDialog).toBeVisible();
  await expectSubtaskSectionAligned(actionDetailDialog);
  await actionDetailDialog.getByRole('button', { name: 'Unteraufgabe hinzufügen' }).click();
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
  await exerciseMobileTaskEditing(page, testInfo, panelTaskID);
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
  await expectSubtaskSectionAligned(detailDialog);
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
  await detailRow.getByRole('button', { name: 'Details' }).click();
  actionDetailDialog = detailRow.locator('[data-task-detail-dialog]');
  await expect(actionDetailDialog).toBeVisible();
  await actionDetailDialog.getByRole('button', { name: 'Löschen' }).click();
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

  let focusEditForm = await openTaskRowEdit(focusRow);
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
  await focusEditForm.locator('[name="title"]').press('Escape');
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
  await deleteRow.getByRole('button', { name: 'Details' }).click();
  let deleteDetailDialog = deleteRow.locator('[data-task-detail-dialog]');
  await expect(deleteDetailDialog).toBeVisible();
  await deleteDetailDialog.getByRole('button', { name: 'Löschen' }).click();
  await expect(deleteDialog).toBeVisible();
  await expect(deleteDialog.locator('[data-task-delete-cancel]')).toBeFocused();
  await deleteDialog.locator('[data-task-delete-cancel]').click();
  await expect(deleteDialog).toBeHidden();
  await expect(deleteRow).toBeVisible();

  await deleteRow.getByRole('button', { name: 'Details' }).click();
  deleteDetailDialog = deleteRow.locator('[data-task-detail-dialog]');
  await expect(deleteDetailDialog).toBeVisible();
  await deleteDetailDialog.getByRole('button', { name: 'Löschen' }).click();
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
  await deleteRow.getByRole('button', { name: 'Details' }).click();
  deleteDetailDialog = deleteRow.locator('[data-task-detail-dialog]');
  await expect(deleteDetailDialog).toBeVisible();
  await deleteDetailDialog.getByRole('button', { name: 'Löschen' }).click();
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
  await exerciseTabletConflictViews(page, testInfo, conflictHref);

  await gotoApp(page, conflictHref);
  await expectCurrentView(page, 'Konfliktdetail');
  await expect(page.locator('[data-conflict-comparison]').getByText('E2E Remote Conflict Edit', { exact: true })).toBeVisible();
  await expect(page.locator('[data-conflict-field="project"]')).toHaveAttribute('data-conflict-row-state', 'unchanged');
  await expect(page.locator('[data-conflict-field="title"]')).toHaveAttribute('data-conflict-row-state', 'changed');
  await page.setViewportSize(mobileViewport);
  await expectNoHorizontalOverflow(page);
  await expect(page.locator('[data-conflict-comparison]')).toBeVisible();
  await expect(page.locator('[data-conflict-field="project"]')).toBeVisible();
  await exerciseMobileConflictResolution(page, testInfo);
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
  expect(unexpectedBrowserConsoleErrors(browserErrors, testInfo.project.name)).toEqual([]);
});

function unexpectedBrowserConsoleErrors(messages, projectName) {
  const hasWebKitSyncStatusAccessError = projectName === 'webkit' && messages.some(isWebKitSyncStatusAccessError);
  return messages.filter((message) => {
    if (expectedBrowserConsoleError(message, projectName)) return false;
    if (hasWebKitSyncStatusAccessError && ['htmx:afterRequest', 'htmx:sendError'].includes(message)) return false;
    return true;
  });
}

function expectedBrowserConsoleError(message, projectName) {
  if (
    projectName === 'webkit' &&
    message === "Refused to apply a stylesheet because its hash, its nonce, or 'unsafe-inline' does not appear in the style-src directive of the Content Security Policy."
  ) {
    return true;
  }
  if (projectName === 'webkit' && isWebKitSyncStatusAccessError(message)) {
    return true;
  }
  if (/^Response Status Error Code 502 from \/tasks\/[^/]+$/.test(message)) {
    return true;
  }
  if (projectName === 'webkit' && isWebKitQuickAddSuggestionsAccessError(message)) {
    return true;
  }

  return [
    'Failed to load resource: the server responded with a status of 400 (Bad Request)',
    'Response Status Error Code 400 from /tasks',
    'Failed to load resource: the server responded with a status of 502 (Bad Gateway)',
    'Response Status Error Code 502 from /tasks/'
  ].includes(message);
}

function isWebKitSyncStatusAccessError(message) {
  return /\/127\.0\.0\.1:\d+\/sync\/status due to access control checks\.$/.test(message);
}

function isWebKitQuickAddSuggestionsAccessError(message) {
  return /\/127\.0\.0\.1:\d+\/quick-add\/suggestions due to access control checks\.$/.test(message);
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

async function exerciseKeyboardFocusAccessibility(page, testInfo) {
  await page.setViewportSize(desktopViewport);
  await gotoApp(page, '/search?q=Stage');
  await expectIconButtonsHaveAccessibleNames(page);

  await expectVisibleFocusIndicator(page.locator('.caldo-topbar a[href="/search"]').first());
  await expectVisibleFocusIndicator(page.locator('.caldo-topbar [data-quick-add-open]').first());
  await expectVisibleFocusIndicator(page.locator('.caldo-sidebar [data-nav-system-filters] a[href="/today"]').first());

  const stageRow = page.locator('[data-task-id]').filter({ hasText: 'Stage Seed Task' }).first();
  await expect(stageRow).toBeVisible();
  await expectVisibleFocusIndicator(stageRow.locator('[data-inline-task-edit-open][data-inline-task-edit-focus="title"]').first());
  await expectVisibleFocusIndicator(stageRow.getByRole('button', { name: 'Details' }));
  await expectVisibleFocusIndicator(stageRow.getByRole('button', { name: /Favorit/ }).first());

  const quickAddTrigger = page.locator('.caldo-topbar [data-quick-add-open]').first();
  await quickAddTrigger.focus();
  await page.keyboard.press('Enter');
  const quickAddDialog = page.locator('[data-quick-add-overlay]');
  await expect(quickAddDialog).toBeVisible();
  await expect(quickAddDialog.locator('[data-quick-add-overlay-input]')).toBeFocused();
  await expectFocusWithinDialog(quickAddDialog);
  await page.keyboard.press('Shift+Tab');
  await expectFocusWithinDialog(quickAddDialog);
  await page.keyboard.press('Tab');
  await expectFocusWithinDialog(quickAddDialog);
  await page.keyboard.press('Escape');
  await expect(quickAddDialog).toBeHidden();
  await expect(quickAddTrigger).toBeFocused();

  await gotoApp(page, '/search?q=Stage');
  const detailRow = page.locator('[data-task-id]').filter({ hasText: 'Stage Seed Task' }).first();
  const detailButton = detailRow.getByRole('button', { name: 'Details' });
  await detailButton.focus();
  await page.keyboard.press('Enter');
  const detailDialog = detailRow.locator('[data-task-detail-dialog]');
  await expect(detailDialog).toBeVisible();
  await expect(detailDialog.locator('[data-task-detail-title]')).toBeFocused();
  await expectIconButtonsHaveAccessibleNames(page);
  await expectFocusWithinDialog(detailDialog);
  await page.keyboard.press('Shift+Tab');
  await expectFocusWithinDialog(detailDialog);
  await page.keyboard.press('Escape');
  await expect(detailDialog).toBeHidden();
  await expect(detailButton).toBeFocused();

  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/accessibility-focus-dialogs.png`, fullPage: true, caret: 'initial' });
}

async function exerciseTaskListDisplayPreferences(page) {
  await page.setViewportSize(mobileViewport);
  await ensureBrowserCSRFCookie(page);

  const display = page.locator('[data-task-display]');
  const trigger = display.locator('summary');
  await expect(trigger).toBeVisible();
  await trigger.focus();
  await page.keyboard.press('Enter');

  const form = display.locator('[data-task-display-form]');
  await expect(form).toBeVisible();
  await page.locator('[data-live-search-input]').fill('Stage Seed');
  await expect(form.locator('[name="search_query"]')).toHaveValue('Stage Seed');
  await form.locator('[name="group_by"]').selectOption('priority');
  await form.locator('[name="sort_by"]').selectOption('name');
  await expect(form.locator('[data-task-display-order-field]')).toBeVisible();
  await form.locator('[name="sort_order"]').selectOption('desc');
  await expectElementHorizontallyWithinViewport(form, mobileViewport);

  await form.getByRole('button', { name: 'Anwenden' }).click();
  await expect(page).toHaveURL(/\/search\?q=Stage\+Seed$/);
  await expect(page.locator('[data-task-display] summary')).toContainText('Priorität');
  await expect(page.locator('[data-task-display] summary')).toContainText('Name');
  await expect(page.locator('[data-task-display] summary')).toContainText('Absteigend');

  await page.reload({ waitUntil: 'domcontentloaded' });
  await page.locator('[data-task-display] summary').click();
  await expect(page.locator('[data-task-display-form] [name="group_by"]')).toHaveValue('priority');
  await expect(page.locator('[data-task-display-form] [name="sort_by"]')).toHaveValue('name');
  await expect(page.locator('[data-task-display-form] [name="sort_order"]')).toHaveValue('desc');

  await page.getByRole('button', { name: 'Zurücksetzen' }).click();
  await expect(page).toHaveURL(/\/search\?q=Stage\+Seed$/);
  await expect(page.locator('[data-task-display] summary span')).toHaveCount(0);
  await page.setViewportSize(desktopViewport);
}

async function exerciseMobileNavigationAndSettings(page, testInfo) {
  await page.setViewportSize(narrowMobileViewport);
  await gotoApp(page, '/search?q=Stage');
  const searchInput = page.locator('#global-search');
  await expect(searchInput).toBeVisible();
  await searchInput.focus();

  let mobileNavTrigger = page.getByRole('button', { name: 'Navigation öffnen' });
  await expect(mobileNavTrigger).toBeVisible();
  await mobileNavTrigger.click();
  await expect(mobileNavTrigger).toHaveAttribute('aria-expanded', 'true');
  const mobileNavDialog = page.locator('[data-mobile-nav-dialog]');
  await expect(mobileNavDialog).toBeVisible();
  await expect(page.getByRole('navigation', { name: 'Mobile Hauptnavigation' })).toBeVisible();
  await expect(page.locator('[data-mobile-nav-close]')).toBeFocused();
  await expect(page.locator('[data-mobile-nav-dialog] nav[aria-label="Mobile Hauptnavigation"] a[href="/today"]')).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/search-mobile.png`, fullPage: true, caret: 'initial' });

  await page.keyboard.press('Escape');
  await expect(mobileNavTrigger).toHaveAttribute('aria-expanded', 'false');
  await expect(mobileNavTrigger).toBeFocused();
  await expect(searchInput).toBeVisible();
  await expectNoHorizontalOverflow(page);

  await mobileNavTrigger.click();
  await expect(mobileNavDialog).toBeVisible();
  await page.locator('[data-mobile-nav-dialog] a[href="/settings"]').click();
  await expect(page).toHaveURL(/\/settings$/);
  await expectCurrentView(page, 'Einstellungen');
  mobileNavTrigger = page.getByRole('button', { name: 'Navigation öffnen' });
  await expect(mobileNavTrigger).toHaveAttribute('aria-expanded', 'false');
  await expect(page.locator('.caldo-topbar a[href="/search"]')).toBeVisible();
  await expect(page.locator('.caldo-topbar a[href="/quick-add"]')).toBeVisible();
  await exerciseMobileSettings(page, testInfo);

  await mobileNavTrigger.click();
  await expect(mobileNavDialog).toBeVisible();
  await page.locator('[data-mobile-nav-dialog] [data-quick-add-open]').click();
  await expect(mobileNavDialog).toBeHidden();
  const quickAddOverlay = page.locator('[data-quick-add-overlay]');
  await expect(quickAddOverlay).toBeVisible();
  await expect(quickAddOverlay.locator('[data-quick-add-overlay-input]')).toBeFocused();
  await expectElementWithinViewport(quickAddOverlay, narrowMobileViewport);
  await expectNoHorizontalOverflow(page);
  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/mobile-quick-add-from-nav.png`, fullPage: true, caret: 'initial' });
  await quickAddOverlay.getByRole('button', { name: 'Schließen' }).click();
  await expect(quickAddOverlay).toBeHidden();
  await page.setViewportSize(desktopViewport);
}

async function exerciseMobileSettings(page, testInfo) {
  await page.setViewportSize(narrowMobileViewport);
  await expectCurrentView(page, 'Einstellungen');
  await expect(page.locator('[data-settings-caldav]')).toBeVisible();
  await expectNoHorizontalOverflow(page);

  const caldavURL = page.locator('#settings_caldav_url');
  await caldavURL.scrollIntoViewIfNeeded();
  await expect(caldavURL).toBeVisible();
  await caldavURL.focus();
  await expectElementHorizontallyWithinViewport(caldavURL, narrowMobileViewport);
  await expect(page.locator('[data-settings-caldav] button[name="caldav_action"][value="test"]')).toBeVisible();
  await expect(page.locator('[data-settings-caldav] button[name="caldav_action"][value="save"]')).toBeVisible();

  const calendarSettings = page.locator('[data-settings-calendars]');
  await calendarSettings.scrollIntoViewIfNeeded();
  await expect(calendarSettings).toBeVisible();
  await expectNoHorizontalOverflow(page);

  const uiSettings = page.locator('#ui-settings');
  await uiSettings.scrollIntoViewIfNeeded();
  await expect(uiSettings).toBeVisible();
  await expect(uiSettings.locator('select[name="dark_mode"]')).toBeVisible();
  await expect(uiSettings.getByRole('button', { name: 'UI-Einstellungen speichern' })).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/mobile-settings.png`, fullPage: true, caret: 'initial' });
}

async function exerciseTabletCoreViews(page, testInfo) {
  await page.setViewportSize(tabletViewport);
  await gotoApp(page, '/projects');
  const defaultProjectHref = await sidebarProjectHref(page, 'E2E Setup Inbox');
  const emptyProjectHref = await sidebarProjectHref(page, 'E2E Empty Project');
  const coreViews = [
    { name: 'inbox-equivalent-default-project', pathname: defaultProjectHref, heading: 'E2E Setup Inbox' },
    { pathname: '/today', heading: 'Heute' },
    { name: 'project-detail', pathname: emptyProjectHref, heading: 'E2E Empty Project', content: 'Keine offenen Aufgaben in diesem Projekt.' },
    { pathname: '/search?q=Stage', heading: 'Suche', content: 'Stage Seed Task' },
    { pathname: '/quick-add', heading: 'Quick Add', content: 'Aufgabe' },
    { pathname: '/conflicts', heading: 'Konflikte', content: 'Keine ungelösten Konflikte' },
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
    await expectTabletTouchTargets(page);
    await page.screenshot({
      path: `${browserArtifactDir(testInfo)}/tablet-${view.name ?? slugifyArtifactName(view.heading)}.png`,
      fullPage: true,
      caret: 'initial'
    });
  }
}

async function exerciseTabletConflictViews(page, testInfo, conflictHref) {
  await page.setViewportSize(tabletViewport);
  await gotoApp(page, '/conflicts');
  await expectCurrentView(page, 'Konflikte');
  await expect(page.locator('[data-conflict-list-row]').first()).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await expectTabletTouchTargets(page);
  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/tablet-conflicts-open.png`, fullPage: true, caret: 'initial' });

  await gotoApp(page, conflictHref);
  await expectCurrentView(page, 'Konfliktdetail');
  await expect(page.locator('[data-conflict-comparison]')).toBeVisible();
  await expect(page.locator('[data-conflict-resolution]')).toBeVisible();
  await expectNoHorizontalOverflow(page);
  await expectTabletTouchTargets(page);
  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/tablet-conflict-detail.png`, fullPage: true, caret: 'initial' });
  await page.setViewportSize(desktopViewport);
}

async function exerciseMobileConflictResolution(page, testInfo) {
  await page.setViewportSize(narrowMobileViewport);
  await expectCurrentView(page, 'Konfliktdetail');
  await expect(page.locator('[data-conflict-comparison]')).toBeVisible();
  await expect(page.locator('[data-conflict-resolution]')).toBeVisible();
  await expectNoHorizontalOverflow(page);

  const manualForm = page.locator('[data-conflict-manual-form]');
  await manualForm.scrollIntoViewIfNeeded();
  await expect(manualForm).toBeVisible();
  await manualForm.locator('[name="title_source"][value="manual"]').check();
  await manualForm.locator('[name="title_manual"]').fill('E2E Mobile Manual Preview');
  await expect(manualForm.locator('[data-conflict-preview-value="title"]')).toContainText('E2E Mobile Manual Preview');
  await expectNoHorizontalOverflow(page);
  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/mobile-conflict-resolution.png`, fullPage: true, caret: 'initial' });

  await manualForm.locator('[name="title_source"][value="remote"]').check();
  await expect(manualForm.locator('[data-conflict-preview-value="title"]')).toContainText('E2E Remote Conflict Edit');
}

async function sidebarProjectHref(page, projectName) {
  const link = page.locator('.caldo-sidebar [data-nav-projects] a').filter({ hasText: projectName }).first();
  await link.scrollIntoViewIfNeeded();
  await expect(link).toBeVisible();
  const href = await link.getAttribute('href');
  expect(href, `project link for ${projectName} should have href`).toBeTruthy();
  return href;
}

async function expectTabletTouchTargets(page) {
  const targets = [
    page.locator('.caldo-sidebar [data-nav-system-filters] a[href="/today"]').first(),
    page.locator('.caldo-topbar a[href="/search"]').first(),
    page.locator('.caldo-topbar [data-quick-add-open]').first(),
    page.locator('.caldo-topbar [data-theme-toggle]').first()
  ];

  for (const target of targets) {
    await expectTouchTargetAtLeast(target, 32);
  }
}

async function expectSubtaskSectionAligned(detailDialog) {
  const titleBox = await detailDialog.locator('[data-task-detail-title]').boundingBox();
  const subtaskHeadingBox = await detailDialog.locator('section[aria-label="Unteraufgaben"] h3').boundingBox();
  expect(titleBox).not.toBeNull();
  expect(subtaskHeadingBox).not.toBeNull();
  expect(Math.abs(subtaskHeadingBox.x - titleBox.x)).toBeLessThanOrEqual(1);
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
  await expectSubtaskSectionAligned(detailDialog);
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
  await row.getByRole('button', { name: 'Details' }).click();
  detailDialog = row.locator('[data-task-detail-dialog]');
  await expect(detailDialog).toBeVisible();
  await detailDialog.getByRole('button', { name: 'Löschen' }).click();
  await expect(deleteDialog).toBeVisible();
  await expectElementWithinViewport(deleteDialog, tabletViewport);
  await expect(deleteDialog.getByRole('button', { name: 'Aufgabe und Unteraufgaben löschen' })).toBeVisible();
  await deleteDialog.getByRole('button', { name: 'Abbrechen', exact: true }).click();
  await expect(deleteDialog).toBeHidden();
  await page.setViewportSize(desktopViewport);
}

async function exerciseMobileTaskEditing(page, testInfo, panelTaskID) {
  await page.setViewportSize(narrowMobileViewport);
  await gotoApp(page, '/search?q=E2E%20Panel%20Edited');
  await expectCurrentView(page, 'Suche');
  await expectNoHorizontalOverflow(page);
  await ensureBrowserCSRFCookie(page);

  let row = page.locator(`[data-task-id="${panelTaskID}"]`);
  await expect(row).toBeVisible();
  await expect(row.locator('.caldo-task-meta-line')).toBeVisible();
  await expectNoHorizontalOverflow(page);

  await openTaskRowEdit(row);
  const inlineTitle = row.locator('[data-inline-task-edit-title]');
  await expect(inlineTitle).toBeFocused();
  await inlineTitle.fill('E2E Mobile Unsaved Draft');
  await expectNoHorizontalOverflow(page);
  await page.screenshot({ path: `${browserArtifactDir(testInfo)}/mobile-task-inline-edit.png`, fullPage: true, caret: 'initial' });
  await inlineTitle.press('Escape');
  await expect(row.locator('[data-inline-task-edit-form][data-inline-task-edit-kind="title"]').first()).toBeHidden();
  await expect(row).toContainText('E2E Panel Edited');

  row = page.locator(`[data-task-id="${panelTaskID}"]`);
  await row.getByRole('button', { name: 'Details' }).click();
  const detailDialog = row.locator('[data-task-detail-dialog]');
  await expect(detailDialog).toBeVisible();
  await expectElementWithinViewport(detailDialog, narrowMobileViewport);
  await detailDialog.locator('[data-task-detail-title]').fill('E2E Mobile Detail Draft');
  await expectNoHorizontalOverflow(page);
  await detailDialog.getByRole('button', { name: 'Details schließen' }).click();
  await expect(detailDialog).toBeHidden();
  await expect(row).toContainText('E2E Panel Edited');
  await page.setViewportSize(desktopViewport);
}

async function openTaskRowEdit(row, focusTarget = 'title') {
  await row.locator(`[data-inline-task-edit-open][data-inline-task-edit-focus="${focusTarget}"]`).first().click();
  const form = row.locator(`[data-inline-task-edit-form][data-inline-task-edit-kind="${focusTarget}"]`).first();
  await expect(form).toBeVisible();
  return form;
}

async function closeTaskRowEdit(row) {
  const titleInput = row.locator('[data-inline-task-edit-title]').first();
  await titleInput.press('Escape');
  await expect(row.locator('[data-inline-task-edit-form][data-inline-task-edit-kind="title"]').first()).toBeHidden();
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
  await expect(page.locator('.caldo-sidebar [data-nav-system-filters] a[href="/settings"]')).toHaveCount(0);
  await expect(page.locator('.caldo-sidebar [data-nav-settings] a[href="/settings"]')).toBeVisible();
  await expect(page.locator('.caldo-sidebar [data-nav-settings] a[href="/settings"]')).toHaveAttribute('aria-current', 'page');
  await page.keyboard.press('g');
  await page.keyboard.press('t');
  await expect(page).toHaveURL(/\/today$/);
  await page.keyboard.press('g');
  await page.keyboard.press('u');
  await expect(page).toHaveURL(/\/upcoming$/);
  await page.keyboard.press('g');
  await page.keyboard.press('p');
  await expect(page).toHaveURL(/\/projects$/);

  await gotoApp(page, '/today');
  await ensureBrowserCSRFCookie(page);
  const syncShortcutResponse = page.waitForResponse(response =>
    response.url().endsWith('/sync/manual') && response.request().method() === 'POST'
  );
  await page.keyboard.press('r');
  expect((await syncShortcutResponse).status()).toBe(200);

  const helpReturnTarget = page.locator('.caldo-sidebar [data-nav-system-filters] a[href="/today"]').first();
  await helpReturnTarget.focus();
  await page.keyboard.press('Shift+/');
  const helpDialog = page.locator('[data-shortcut-help-dialog]');
  await expect(helpDialog).toBeVisible();
  await expect(helpDialog).toHaveAttribute('aria-labelledby', 'shortcut-help-title');
  await expect(helpDialog.getByRole('button', { name: 'Schließen' })).toBeFocused();
  await page.keyboard.press('Tab');
  await expect(helpDialog.getByRole('button', { name: 'Schließen' })).toBeFocused();
  await expect(helpDialog).toContainText('Tastaturkürzel');
  await expect(helpDialog).not.toContainText('Mehrfachbearbeitung ist nicht verfügbar');
  await expect(helpDialog).toContainText('Jetzt synchronisieren');
  await expect(helpDialog).toContainText('G');
  await page.keyboard.press('Escape');
  await expect(helpDialog).toBeHidden();
  await expect(helpReturnTarget).toBeFocused();
}

async function exerciseQuickAddOverlay(page) {
  await gotoApp(page, '/search?q=Stage');
  await ensureBrowserCSRFCookie(page);
  const searchURL = page.url();
  const overlay = page.locator('[data-quick-add-overlay]');
  const input = overlay.locator('[data-quick-add-overlay-input]');
  const previewForm = overlay.locator('[data-quick-add-overlay-form]');
  const shortcutReturnTarget = page.locator('.caldo-topbar [data-theme-toggle]').first();

  await shortcutReturnTarget.focus();
  await expect(shortcutReturnTarget).toBeFocused();
  await page.keyboard.press('n');
  await expect(overlay).toBeVisible();
  await expect(input).toBeFocused();
  await expect(page).toHaveURL(searchURL);

  await input.fill('E2E Live Date morgen');
  await input.evaluate((element) => element.setSelectionRange(5, 5));
  let liveDateChip = overlay.locator('[data-quick-add-chips] [data-quick-add-date-chip]');
  await expect(liveDateChip).toBeVisible();
  await expect(liveDateChip).toContainText('morgen');
  await expect(liveDateChip).toContainText(/\d{4}-\d{2}-\d{2}/);
  await expect(input).toBeFocused();
  await expect.poll(async () => input.evaluate((element) => element.selectionStart)).toBe(5);
  const liveSaveForm = overlay.locator('[data-quick-add-overlay-save-form]');
  await expect(liveSaveForm.locator('input[name="title"]')).toHaveValue('E2E Live Date');

  await input.fill('E2E Live Weekday Mittwoch');
  await input.evaluate((element) => element.setSelectionRange(4, 4));
  liveDateChip = overlay.locator('[data-quick-add-chips] [data-quick-add-date-chip]');
  await expect(liveDateChip).toContainText('Mittwoch');
  await expect(liveDateChip.locator('[data-quick-add-date-warning]')).toBeVisible();
  await expect(input).toBeFocused();
  await expect.poll(async () => input.evaluate((element) => element.selectionStart)).toBe(4);

  await input.fill('E2E Overlay Canceled');
  await page.keyboard.press('Escape');
  await expect(overlay).toBeHidden();
  await expect(shortcutReturnTarget).toBeFocused();
  await expect(page).toHaveURL(searchURL);

  await page.keyboard.press('n');
  await expect(input).toBeFocused();
  await page.keyboard.type('n');
  await expect(input).toHaveValue('n');
  await input.fill('E2E Overlay Failed');
  await page.keyboard.press('Enter');
  let saveForm = overlay.locator('[data-quick-add-overlay-save-form]');
  await expect(saveForm).toBeVisible();
  // Input-first: corrections hidden, reveal via toggle
  await saveForm.locator('[data-quick-add-corrections-toggle]').click();
  await expect(saveForm.locator('[data-quick-add-corrections]')).toBeVisible();
  await expect(saveForm.locator('input[name="title"]')).toBeFocused();
  await saveForm.locator('input[name="title"]').fill('');
  await page.keyboard.press('Control+Enter');
  await expect(overlay).toBeVisible();
  await expect(overlay.locator('[data-quick-add-overlay-error]')).toContainText('Aufgabe konnte nicht gespeichert werden.');
  await expect(overlay.locator('[data-quick-add-overlay-error]')).toHaveAttribute('role', 'alert');
  await expect(saveForm).toHaveAttribute('aria-describedby', /quick-add-overlay-error/);
  await expect(saveForm.locator('input[name="title"]')).toHaveAttribute('aria-describedby', /quick-add-overlay-error/);
  await expect(saveForm.locator('input[name="title"]')).toHaveAttribute('aria-invalid', 'true');
  await expectVisibleFormErrorsAssociated(page);
  await expect(input).toHaveValue('E2E Overlay Failed');
  await expect(page).toHaveURL(searchURL);

  await input.fill('E2E Overlay Chips #Work @urgent morgen wöchentlich !2');
  await page.keyboard.press('Enter');
  saveForm = overlay.locator('[data-quick-add-overlay-save-form]');
  await expect(overlay.locator('[data-quick-add-chips]')).toContainText('Work');
  await expect(overlay.locator('[data-quick-add-chips]')).toContainText('urgent');
  await expect(overlay.locator('[data-quick-add-chips]')).toContainText('Wöchentlich');
  await expect(overlay.locator('[data-quick-add-chips]')).toContainText('P2 Mittel');

  // Input-first: correction fields hidden by default, toggle via "Bearbeiten"
  const corrections = saveForm.locator('[data-quick-add-corrections]');
  await expect(corrections).toBeHidden();
  const editToggle = saveForm.locator('[data-quick-add-corrections-toggle]');
  await expect(editToggle).toBeVisible();
  await editToggle.click();
  await expect(corrections).toBeVisible();
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
  await expect(shortcutReturnTarget).toBeFocused();
  await expect(page).toHaveURL(searchURL);
  await waitForSearchResult(page, 'E2E Overlay Corrected');
  const correctedRow = page.locator('[data-task-id]').filter({ hasText: 'E2E Overlay Corrected' }).first();
  await expect(correctedRow).toContainText('reviewed');
  await expect(correctedRow).toContainText('P3');
  await expect(correctedRow).toContainText('Wöchentlich');
  await expect(correctedRow).toContainText('Fällig 30.06.2099');

  await page.locator('.caldo-topbar [data-quick-add-open]').click();
  await input.fill('E2E Overlay Suggested #Work');
  await previewForm.getByRole('button', { name: 'Vorschau' }).click();
  saveForm = overlay.locator('[data-quick-add-overlay-save-form]');
  // Input-first: suggestions are inside corrections, reveal first
  await saveForm.locator('[data-quick-add-corrections-toggle]').click();
  await expect(saveForm.locator('[data-quick-add-corrections]')).toBeVisible();
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
  await expect(toggle).toHaveAttribute('aria-label', 'Darstellung: System');
  await expect(toggle).toHaveAttribute('title', 'Darstellung: System');

  await toggle.click();
  await expect(root).toHaveAttribute('data-theme-mode', 'dark');
  await expect.poll(async () => (await root.getAttribute('class')) || '').toBe('dark');
  await expect(toggle).toHaveAttribute('aria-label', 'Darstellung: Dunkel');

  await toggle.click();
  await expect(root).toHaveAttribute('data-theme-mode', 'light');
  await expect.poll(async () => (await root.getAttribute('class')) || '').toBe('light');
  await expect(toggle).toHaveAttribute('aria-label', 'Darstellung: Hell');

  await toggle.click();
  await expect(root).toHaveAttribute('data-theme-mode', 'system');
  await expect(root).toHaveAttribute('data-theme-effective', /^(dark|light)$/);
  await expect.poll(async () => (await root.getAttribute('class')) || '').toBe('');
  await expect(toggle).toHaveAttribute('aria-label', 'Darstellung: System');
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
    await syncStatus.getByRole('button', { name: /Jetzt synchronisieren/ }).click();
    const event = await eventPromise;
    expect(event).toMatchObject({ type: 'sync', resource: 'sync_status' });
    await expect.poll(async () => {
      const response = await page.request.get(appURL('/sync/status'), {
        headers: { [state.proxyUserHeader]: 'e2e-user' },
        failOnStatusCode: false
      });
      if (response.status() !== 200) return '';
      return response.text();
    }, { timeout: 30_000 }).toMatch(/data-sync-state="idle"[\s\S]*Letzter erfolgreicher Sync: (?!nie)/);
    await expect.poll(async () => await syncStatus.getByRole('button', { name: /Jetzt synchronisieren/ }).getAttribute('aria-label') || '').toMatch(/Status: bereit[\s\S]*Letzter erfolgreicher Sync: (?!nie)/);
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

async function exerciseWriteStatusForFailedInlineMetadata(page, taskRow, input) {
  const routePattern = '**/tasks/*';
  const routeHandler = async (route) => {
    const requestURL = new URL(route.request().url());
    if (route.request().method() !== 'PATCH' || !/^\/tasks\/[^/]+$/.test(requestURL.pathname)) {
      await route.continue();
      return;
    }
    await route.fulfill({
      status: 502,
      headers: { 'content-type': 'text/plain; charset=utf-8' },
      body: 'failed to update task on caldav server'
    });
  };

  await page.route(routePattern, routeHandler);
  try {
    await input.press('Enter');
    await expect(taskRow.locator('[data-inline-task-edit-error]')).toBeVisible();
    await expect(taskRow.locator('[data-inline-task-edit-error]')).toContainText('Aufgabe konnte nicht gespeichert werden.');
    await expect(input).toHaveValue('failed inline label');
    await expect(page.locator('[data-write-status]')).toContainText('Speichern fehlgeschlagen');
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

function slugifyArtifactName(value) {
  return value
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[^\w]+/g, '-')
    .replace(/^-+|-+$/g, '');
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

async function expectElementHorizontallyWithinViewport(locator, viewport) {
  const box = await locator.boundingBox();
  if (!box) {
    throw new Error('expected visible element to have a bounding box');
  }
  expect(box.x).toBeGreaterThanOrEqual(-1);
  expect(box.x + box.width).toBeLessThanOrEqual(viewport.width + 1);
}

async function expectIconButtonsHaveAccessibleNames(page) {
  const missingNames = await page.locator('button.caldo-icon-button').evaluateAll((buttons) => (
    buttons
      .filter((button) => !String(button.getAttribute('aria-label') || '').trim())
      .map((button) => button.outerHTML)
  ));
  expect(missingNames, 'icon buttons must have explicit aria-label values').toEqual([]);
}

async function expectVisibleFocusIndicator(locator) {
  await expect(locator).toBeVisible();
  await locator.focus();
  await expect(locator).toBeFocused();
  const focusStyle = await locator.evaluate((element) => {
    const style = window.getComputedStyle(element);
    const outlineWidth = Number.parseFloat(style.outlineWidth || '0') || 0;
    const hasOutline = style.outlineStyle !== 'none' && outlineWidth > 0;
    const hasShadow = Boolean(style.boxShadow && style.boxShadow !== 'none');
    return {
      hasIndicator: hasOutline || hasShadow,
      outlineStyle: style.outlineStyle,
      outlineWidth: style.outlineWidth,
      boxShadow: style.boxShadow
    };
  });
  expect(
    focusStyle.hasIndicator,
    `expected visible focus indicator, got outline=${focusStyle.outlineStyle} ${focusStyle.outlineWidth} box-shadow=${focusStyle.boxShadow}`
  ).toBe(true);
}

async function expectFocusWithinDialog(dialog) {
  const containsFocus = await dialog.evaluate((element) => element.contains(document.activeElement));
  expect(containsFocus, 'open dialog must contain keyboard focus').toBe(true);
}

async function expectVisibleFormErrorsAssociated(page) {
  const errors = page.locator('.caldo-alert-error[role="alert"]:not([hidden])');
  const count = await errors.count();
  expect(count, 'expected at least one visible form error').toBeGreaterThan(0);
  for (let index = 0; index < count; index += 1) {
    const association = await errors.nth(index).evaluate((error) => {
      const id = String(error.id || '').trim();
      const describedByIncludesID = (element) => String(element.getAttribute('aria-describedby') || '')
        .split(/\s+/)
        .includes(id);
      const forms = Array.from(document.querySelectorAll('form'))
        .filter((form) => describedByIncludesID(form));
      const controls = forms.flatMap((form) => (
        Array.from(form.querySelectorAll('input:not([type="hidden"]), select, textarea'))
      ));
      return {
        id,
        formDescribed: Boolean(id && forms.length > 0),
        controlDescribed: Boolean(id && controls.some((control) => describedByIncludesID(control))),
        controlInvalid: Boolean(controls.some((control) => control.getAttribute('aria-invalid') === 'true'))
      };
    });
    expect(association.id, 'visible form error must have an id').toBeTruthy();
    expect(association.formDescribed, 'form must reference visible error via aria-describedby').toBe(true);
    expect(association.controlDescribed, 'a form control must reference visible error via aria-describedby').toBe(true);
    expect(association.controlInvalid, 'a form control must expose aria-invalid=true').toBe(true);
  }
}

async function expectTouchTargetAtLeast(locator, minimumSize) {
  await expect(locator).toBeVisible();
  const box = await locator.boundingBox();
  if (!box) {
    throw new Error('expected visible touch target to have a bounding box');
  }
  expect(box.width).toBeGreaterThanOrEqual(minimumSize);
  expect(box.height).toBeGreaterThanOrEqual(minimumSize);
}
