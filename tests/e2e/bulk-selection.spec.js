const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');
const staticManifest = require('../../web/static/manifest.json');
const appStyle = path.resolve(__dirname, '../../web/static', staticManifest['app.css']);

test('bulk selection is reachable and reports partial completion failures', async ({ page }) => {
  await page.setContent(`
    <div data-toast-container></div>
    <ul>
      ${taskRow('task-1', 'Erste Aufgabe')}
      ${taskRow('task-2', 'Zweite Aufgabe')}
    </ul>
  `);
  await page.evaluate(() => {
    window.fetch = (url) => Promise.resolve({
      ok: String(url).endsWith('/task-1/complete'),
      status: String(url).endsWith('/task-1/complete') ? 200 : 409,
      text: () => Promise.resolve('')
    });
  });
  await page.addScriptTag({ path: appScript });

  const firstRow = page.locator('[data-task-id="task-1"]');
  const secondRow = page.locator('[data-task-id="task-2"]');
  const firstControl = firstRow.getByRole('checkbox', { name: /Mehrfachbearbeitung/ });
  const secondControl = secondRow.getByRole('checkbox', { name: /Mehrfachbearbeitung/ });

  await firstControl.click();
  await expect(firstControl).toBeChecked();
  await expect(firstRow).toHaveClass(/caldo-task-row-selected/);
  await expect(page.getByRole('region', { name: 'Mehrfachbearbeitung' })).toContainText('1 Aufgabe ausgewählt');
  await firstControl.click();
  await expect(firstControl).not.toBeChecked();
  await expect(firstRow).not.toHaveClass(/caldo-task-row-selected/);
  await expect(page.getByRole('region', { name: 'Mehrfachbearbeitung' })).toHaveCount(0);

  await firstRow.getByRole('textbox', { name: 'Erste Aufgabe' }).click({ modifiers: ['Control'] });
  await expect(firstControl).toBeChecked();
  await expect(firstRow.locator('[data-inline-task-edit-form]')).toBeVisible();

  await secondControl.focus();
  await page.keyboard.press('Shift+Enter');
  await expect(firstControl).toBeChecked();
  await expect(secondControl).toBeChecked();
  const bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await expect(bulkBar).toContainText('2 Aufgaben ausgewählt');

  await page.keyboard.press('Enter');
  await expect(secondControl).not.toBeChecked();
  await expect(bulkBar).toContainText('1 Aufgabe ausgewählt');
  await page.keyboard.press('Space');
  await expect(secondControl).toBeChecked();

  await bulkBar.getByRole('button', { name: 'Erledigen' }).click();
  await expect(firstRow).toHaveCount(0);
  await expect(secondRow).toBeVisible();
  await expect(secondControl).toBeChecked();
  await expect(bulkBar).toContainText('1 Aufgabe ausgewählt');
  await expect(page.locator('[data-toast-container]')).toContainText('1 Aufgabe erledigt, 1 Aufgabe konnte nicht erledigt werden.');
  await expect(page.locator('[data-toast-container]')).not.toContainText('2 Aufgaben erledigt.');
});

test('bulk due-date and priority actions patch every selected task with its current version', async ({ page }) => {
  await setBulkTaskPage(page, [
    taskRow('task-1', 'Erste Aufgabe', { version: 3, labels: 'home, urgent' }),
    taskRow('task-2', 'Zweite Aufgabe', { version: 7, labels: 'work' })
  ]);
  await installMetadataFetch(page);
  await page.addScriptTag({ path: appScript });

  await selectAllTasks(page);
  let bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await bulkBar.locator('[data-bulk-action-menu]').filter({ hasText: 'Fälligkeit' }).locator('summary').click();
  await bulkBar.getByRole('button', { name: 'Heute', exact: true }).click();
  const today = await page.evaluate(() => {
    const now = new Date();
    return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}-${String(now.getDate()).padStart(2, '0')}`;
  });
  await expect.poll(() => metadataRequestParams(page, 0)).toMatchObject({ expected_version: '3', due_date: today });
  await expect.poll(() => metadataRequestParams(page, 1)).toMatchObject({ expected_version: '7', due_date: today });
  await expect(page.locator('[data-toast-container]')).toContainText('2 Aufgaben aktualisiert.');
  await expect(bulkBar).toHaveCount(0);

  await selectAllTasks(page);
  bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await bulkBar.locator('[data-bulk-action-menu]').filter({ hasText: 'Fälligkeit' }).locator('summary').click();
  await bulkBar.locator('[data-bulk-due-custom]').fill('2030-04-15');
  await bulkBar.getByRole('button', { name: 'Anwenden' }).click();
  await expect.poll(() => metadataRequestParams(page, 2)).toMatchObject({ due_date: '2030-04-15' });
  await expect.poll(() => metadataRequestParams(page, 3)).toMatchObject({ due_date: '2030-04-15' });
  await expect(bulkBar).toHaveCount(0);

  await selectAllTasks(page);
  bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await bulkBar.locator('[data-bulk-action-menu]').filter({ hasText: 'Fälligkeit' }).locator('summary').click();
  await bulkBar.getByRole('button', { name: 'Kein Datum' }).click();
  await expect.poll(() => metadataRequestParams(page, 4)).toMatchObject({ due_date: '' });
  await expect.poll(() => metadataRequestParams(page, 5)).toMatchObject({ due_date: '' });
  await expect(bulkBar).toHaveCount(0);

  await selectAllTasks(page);
  bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await bulkBar.locator('[data-bulk-action-menu]').filter({ hasText: 'Priorität' }).locator('summary').click();
  await bulkBar.getByRole('button', { name: 'P1 Hoch' }).click();
  await expect.poll(() => metadataRequestParams(page, 6)).toMatchObject({ priority: '1' });
  await expect.poll(() => metadataRequestParams(page, 7)).toMatchObject({ priority: '1' });
  await expect(bulkBar).toHaveCount(0);

  await selectAllTasks(page);
  bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await bulkBar.locator('[data-bulk-action-menu]').filter({ hasText: 'Priorität' }).locator('summary').click();
  await bulkBar.getByRole('button', { name: 'Keine Priorität' }).click();
  await expect.poll(() => metadataRequestParams(page, 8)).toMatchObject({ priority: '' });
  await expect.poll(() => metadataRequestParams(page, 9)).toMatchObject({ priority: '' });
  await expect(bulkBar).toHaveCount(0);
});

test('bulk label actions add and remove without replacing unrelated labels', async ({ page }) => {
  await setBulkTaskPage(page, [
    taskRow('task-1', 'Erste Aufgabe', { version: 2, labels: 'home, urgent' }),
    taskRow('task-2', 'Zweite Aufgabe', { version: 4, labels: 'work, urgent' })
  ]);
  await installMetadataFetch(page);
  await page.addScriptTag({ path: appScript });

  await selectAllTasks(page);
  let bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await bulkBar.locator('[data-bulk-label-action] summary').click();
  await bulkBar.locator('[data-bulk-label-input]').fill('shared, HOME');
  await bulkBar.getByRole('button', { name: 'Hinzufügen' }).click();
  await expect.poll(() => metadataRequestParams(page, 0)).toMatchObject({ labels: 'home,urgent,shared' });
  await expect.poll(() => metadataRequestParams(page, 1)).toMatchObject({ labels: 'work,urgent,shared,HOME' });
  await expect(bulkBar).toHaveCount(0);

  await selectAllTasks(page);
  bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await bulkBar.locator('[data-bulk-label-action] summary').click();
  await bulkBar.locator('[data-bulk-label-input]').fill('urgent');
  await bulkBar.getByRole('button', { name: 'Entfernen' }).click();
  await expect.poll(() => metadataRequestParams(page, 2)).toMatchObject({ labels: 'home' });
  await expect.poll(() => metadataRequestParams(page, 3)).toMatchObject({ labels: 'work' });
  await expect(bulkBar).toHaveCount(0);

  await selectAllTasks(page);
  bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await bulkBar.locator('[data-bulk-label-action] summary').click();
  await bulkBar.locator('[data-bulk-label-input]').fill('STARRED');
  await bulkBar.getByRole('button', { name: 'Hinzufügen' }).click();
  await expect(page.locator('[data-toast-container]')).toContainText('STARRED ist reserviert');
  await expect.poll(() => page.evaluate(() => window.__bulkRequests.length)).toBe(4);
  await expect(page.getByRole('checkbox', { name: /Mehrfachbearbeitung/ }).first()).toBeChecked();
});

test('bulk metadata keeps failed tasks selected and prevents overlapping writes', async ({ page }) => {
  await setBulkTaskPage(page, [
    taskRow('task-1', 'Erste Aufgabe', { version: 5, labels: 'home' }),
    taskRow('task-2', 'Zweite Aufgabe', { version: 9, labels: 'work' }),
    taskRow('task-3', 'Dritte Aufgabe', { version: 12, labels: 'shared' })
  ]);
  await page.evaluate(() => {
    window.__bulkRequests = [];
    window.__bulkResolvers = [];
    window.fetch = (url, options = {}) => {
      const method = String(options.method || 'GET').toUpperCase();
      if (method === 'PATCH') {
        window.__bulkRequests.push({ url: String(url), method, body: String(options.body || '') });
        return new Promise((resolve) => window.__bulkResolvers.push(resolve));
      }
      return Promise.resolve({
        ok: true,
        status: 200,
        text: () => Promise.resolve('<div class="caldo-content"><section class="caldo-page"><div class="caldo-state">Keine Aufgaben</div></section></div>')
      });
    };
  });
  await page.addScriptTag({ path: appScript });

  await selectAllTasks(page);
  let bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await bulkBar.locator('[data-bulk-action-menu]').filter({ hasText: 'Priorität' }).locator('summary').click();
  await bulkBar.getByRole('button', { name: 'P2 Mittel' }).click();
  await expect(bulkBar).toHaveAttribute('data-bulk-running', '');
  await expect(bulkBar.locator('[data-bulk-complete]')).toBeDisabled();
  await expect.poll(() => page.evaluate(() => window.__bulkRequests.length)).toBe(3);
  await bulkBar.locator('[data-bulk-complete]').click({ force: true });
  await expect.poll(() => page.evaluate(() => window.__bulkRequests.length)).toBe(3);

  await page.evaluate(() => {
    window.__bulkResolvers[0]({ ok: true, status: 200, text: () => Promise.resolve('') });
    window.__bulkResolvers[1]({ ok: false, status: 409, text: () => Promise.resolve('') });
    window.__bulkResolvers[2]({ ok: false, status: 502, text: () => Promise.resolve('') });
  });
  bulkBar = page.getByRole('region', { name: 'Mehrfachbearbeitung' });
  await expect(bulkBar).toContainText('2 Aufgaben ausgewählt');
  await expect(page.locator('[data-task-id="task-1"]')).toHaveCount(0);
  await expect(page.locator('[data-task-id="task-2"] [data-bulk-select-control]')).toBeChecked();
  await expect(page.locator('[data-task-id="task-3"] [data-bulk-select-control]')).toBeChecked();
  await expect(page.locator('[data-toast-container]')).toContainText('1 Aufgabe aktualisiert, 2 Aufgaben konnten nicht aktualisiert werden.');
  await expect(page.locator('[data-toast-container]')).not.toContainText('3 Aufgaben aktualisiert.');
});

test('bulk action bar and menus stay within narrow mobile and tablet viewports', async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 720 });
  await setBulkTaskPage(page, [taskRow('task-1', 'Erste Aufgabe', { version: 1, labels: 'home' })]);
  await page.addStyleTag({ path: appStyle });
  await page.addScriptTag({ path: appScript });
  await page.getByRole('checkbox', { name: /Mehrfachbearbeitung/ }).click();

  const dueDetails = page.locator('[data-bulk-action-menu]').filter({ hasText: 'Fälligkeit' });
  const priorityDetails = page.locator('[data-bulk-action-menu]').filter({ hasText: 'Priorität' });
  await dueDetails.locator('summary').focus();
  await page.keyboard.press('Enter');
  await expect(dueDetails).toHaveAttribute('open', '');
  await priorityDetails.locator('summary').focus();
  await page.keyboard.press('Enter');
  await expect(priorityDetails).toHaveAttribute('open', '');
  await expect(dueDetails).not.toHaveAttribute('open', '');
  await page.keyboard.press('Enter');

  for (const summary of ['Fälligkeit', 'Priorität', 'Labels']) {
    const details = page.locator('[data-bulk-action-menu]').filter({ hasText: summary });
    await details.locator('summary').click();
    await expectElementWithinWidth(details.locator('.caldo-bulk-action-menu'), 320);
    await details.locator('summary').click();
  }
  await expectNoDocumentOverflow(page, 320);

  await page.setViewportSize({ width: 834, height: 1112 });
  const labelDetails = page.locator('[data-bulk-label-action]');
  await labelDetails.locator('summary').click();
  await expectElementWithinWidth(labelDetails.locator('.caldo-bulk-action-menu'), 834);
  await expectNoDocumentOverflow(page, 834);
});

async function setBulkTaskPage(page, rows) {
  await page.setContent(`
    <div data-toast-container></div>
    <div id="write-status" class="hidden"></div>
    <div class="caldo-content">
      <section class="caldo-page">
        <ul class="caldo-list caldo-task-list" data-date-view-results>${rows.join('')}</ul>
      </section>
    </div>
  `);
}

async function installMetadataFetch(page) {
  await page.evaluate(() => {
    window.__bulkRequests = [];
    window.fetch = (url, options = {}) => {
      const method = String(options.method || 'GET').toUpperCase();
      if (method === 'PATCH') {
        window.__bulkRequests.push({ url: String(url), method, body: String(options.body || '') });
        return Promise.resolve({ ok: true, status: 200, text: () => Promise.resolve('') });
      }
      return Promise.resolve({ ok: true, status: 200, text: () => Promise.resolve(document.documentElement.outerHTML) });
    };
  });
}

async function selectAllTasks(page) {
  const controls = page.getByRole('checkbox', { name: /Mehrfachbearbeitung/ });
  const count = await controls.count();
  for (let index = 0; index < count; index += 1) {
    await controls.nth(index).click();
  }
}

async function metadataRequestParams(page, index) {
  return page.evaluate((requestIndex) => {
    const request = window.__bulkRequests[requestIndex];
    if (!request) return null;
    return Object.fromEntries(new URLSearchParams(request.body));
  }, index);
}

async function expectElementWithinWidth(locator, width) {
  const box = await locator.boundingBox();
  expect(box).not.toBeNull();
  expect(box.x).toBeGreaterThanOrEqual(-1);
  expect(box.x + box.width).toBeLessThanOrEqual(width + 1);
}

async function expectNoDocumentOverflow(page, width) {
  const scrollWidth = await page.evaluate(() => Math.max(document.documentElement.scrollWidth, document.body.scrollWidth));
  expect(scrollWidth).toBeLessThanOrEqual(width + 1);
}

function taskRow(id, title, options = {}) {
  const version = options.version || 1;
  const labels = options.labels || '';
  return `
    <li data-task-id="${id}" data-task-status="needs-action" data-server-version="${version}">
      <form action="/tasks/${id}/complete" data-task-action-form>
        <input type="hidden" name="expected_version" value="${version}">
      </form>
      <form data-inline-task-edit-form data-inline-task-edit-persistent>
        <input name="title" data-inline-task-edit-title aria-label="${title}" value="${title}">
      </form>
      <input type="text" value="${labels}" data-task-labels-input>
      <label>
        <input type="checkbox" data-bulk-select-control aria-label="Für Mehrfachbearbeitung auswählen: ${title}">
      </label>
    </li>
  `;
}
