const path = require('node:path');
const { test, expect } = require('@playwright/test');
const manifest = require('../../web/static/manifest.json');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');
const htmxScript = path.resolve(__dirname, '../../web/static/htmx.449317a.min.js');
const appStyles = path.resolve(__dirname, '../../web/static', manifest['app.css']);
const title = 'Alpha Bravo Charlie Delta Echo Foxtrot';
const detailDescription = 'First detail line with enough text for selection\nSecond detail line for another caret target';

test.beforeEach(async ({ page }) => {
  await page.setViewportSize({ width: 800, height: 600 });
  await page.route('http://caldo.test/', async (route) => {
    await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
  });
  await page.goto('http://caldo.test/');
  await page.setContent(`
    <meta name="csrf-token" content="test-token">
    <div id="write-status" data-write-status></div>
    <main style="width: 320px">
      <ul>
        <li
          class="caldo-task-row"
          data-task-id="task-1"
          data-server-version="1"
          data-task-project-id="project-1"
          data-task-move-path="/tasks/task-1/move"
          data-task-drag-move
          draggable="true"
          tabindex="0"
        >
          <div class="caldo-task-row-grid">
            <div aria-hidden="true"></div>
            <div class="caldo-task-content">
              <div class="caldo-task-title-line">
                <div class="caldo-task-title-edit-shell" data-inline-task-edit-scope data-inline-task-edit-focus="title">
                  <form
                    method="post"
                    action="/tasks/task-1"
                    hx-patch="/tasks/task-1"
                    hx-swap="none"
                    data-inline-task-edit-form
                    data-inline-task-edit-kind="title"
                    data-inline-task-edit-persistent
                    aria-describedby="task-edit-error-task-1"
                  >
                    <input type="hidden" name="expected_version" value="1">
                    <input
                      type="text"
                      name="title"
                      draggable="false"
                      class="caldo-task-title caldo-task-edit-title"
                      value="${title}"
                      data-inline-task-edit-title
                      aria-label="Titel bearbeiten: ${title}"
                      aria-describedby="task-edit-error-task-1"
                    >
                    <button type="submit" class="sr-only">Speichern</button>
                  </form>
                </div>
              </div>
              <p id="task-edit-error-task-1" data-inline-task-edit-error role="alert" hidden></p>
              <button
                type="button"
                data-task-detail-open
                aria-controls="task-detail-task-1"
                aria-expanded="false"
              >Details</button>
            </div>
          </div>
          <dialog id="task-detail-task-1" data-task-detail-dialog>
            <form data-task-detail-form>
              <input data-task-detail-title name="title" type="text" value="${title}">
              <textarea name="description">${detailDescription}</textarea>
              <input name="labels" type="text" value="alpha, bravo, charlie">
              <input name="due_date" type="date" value="2099-06-12">
              <select name="priority">
                <option value="0">Keine</option>
                <option value="1" selected>Hoch</option>
              </select>
              <button type="button" data-task-detail-close>Schließen</button>
            </form>
          </dialog>
        </li>
      </ul>
    </main>
  `);
  await page.addStyleTag({ path: appStyles });
  await page.addScriptTag({ path: htmxScript });
  await page.addScriptTag({ path: appScript });
  await page.evaluate(() => window.htmx.process(document.body));
});

test('persistent title input uses native mouse and keyboard selection', async ({ page }) => {
  const input = page.locator('[data-inline-task-edit-title]');
  const box = await input.boundingBox();
  expect(box).not.toBeNull();

  await input.click({ position: { x: box.width * 0.7, y: box.height / 2 } });
  const firstOffset = await selectionStart(input);
  await input.click({ position: { x: box.width * 0.2, y: box.height / 2 } });
  const secondOffset = await selectionStart(input);
  await input.click({ position: { x: box.width * 0.48, y: box.height / 2 } });
  const thirdOffset = await selectionStart(input);

  expect(secondOffset).toBeLessThan(firstOffset);
  expect(thirdOffset).toBeGreaterThan(secondOffset);

  await page.mouse.move(box.x + box.width * 0.15, box.y + box.height / 2);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width * 0.62, box.y + box.height / 2, { steps: 8 });
  await page.mouse.up();
  const draggedSelection = await input.evaluate((element) => ({
    start: element.selectionStart,
    end: element.selectionEnd
  }));
  expect(draggedSelection.end).toBeGreaterThan(draggedSelection.start);

  await input.press('Home');
  await input.press('Shift+End');
  await expect.poll(() => input.evaluate((element) => element.selectionEnd - element.selectionStart)).toBe(title.length);
  await input.press('ArrowLeft');
  await input.press('Control+A');
  await expect.poll(() => input.evaluate((element) => element.selectionEnd - element.selectionStart)).toBe(title.length);
});

test('task detail controls preserve native pointer behavior inside a draggable row', async ({ page }) => {
  const row = page.locator('[data-task-id="task-1"]');
  const detail = await openTaskDetail(row);
  const detailTitle = detail.locator('[data-task-detail-title]');
  const description = detail.locator('[name="description"]');

  await expect(row).toHaveAttribute('draggable', 'true');
  await expect(detailTitle).toHaveJSProperty('selectionStart', 0);

  const titleOffsets = await clickAtFractions(detailTitle, [0.75, 0.2, 0.5]);
  expect(titleOffsets[1]).toBeLessThan(titleOffsets[0]);
  expect(titleOffsets[2]).toBeGreaterThan(titleOffsets[1]);
  await expect(row).toHaveAttribute('draggable', 'true');

  const descriptionOffsets = await clickAtFractions(description, [0.7, 0.15]);
  expect(descriptionOffsets[1]).toBeLessThan(descriptionOffsets[0]);

  const descriptionBox = await description.boundingBox();
  expect(descriptionBox).not.toBeNull();
  await page.mouse.move(descriptionBox.x + descriptionBox.width * 0.12, descriptionBox.y + 12);
  await page.mouse.down();
  await expect(row).toHaveAttribute('draggable', 'false');
  await page.mouse.move(descriptionBox.x + descriptionBox.width * 0.62, descriptionBox.y + 12, { steps: 8 });
  await page.mouse.up();
  await expect(row).toHaveAttribute('draggable', 'true');
  const draggedSelection = await description.evaluate((element) => ({
    start: element.selectionStart,
    end: element.selectionEnd
  }));
  expect(draggedSelection.end).toBeGreaterThan(draggedSelection.start);

  await description.press('Home');
  await description.press('Shift+End');
  await expect.poll(() => description.evaluate((element) => element.selectionEnd - element.selectionStart)).toBeGreaterThan(0);

  for (const control of [detail.locator('[name="labels"]'), detail.locator('[name="due_date"]'), detail.locator('[name="priority"]')]) {
    const box = await control.boundingBox();
    expect(box).not.toBeNull();
    await page.mouse.move(box.x + 5, box.y + box.height / 2);
    await page.mouse.down();
    await expect(row).toHaveAttribute('draggable', 'false');
    await page.mouse.up();
    await expect(row).toHaveAttribute('draggable', 'true');
  }
});

test('closing or cancelling detail interaction restores row dragging', async ({ page }) => {
  const row = page.locator('[data-task-id="task-1"]');
  let detail = await openTaskDetail(row);
  const description = detail.locator('[name="description"]');
  const box = await description.boundingBox();
  expect(box).not.toBeNull();

  await page.mouse.move(box.x + 10, box.y + 10);
  await page.mouse.down();
  await expect(row).toHaveAttribute('draggable', 'false');
  await detail.evaluate((dialog) => dialog.close());
  await expect(row).toHaveAttribute('draggable', 'true');
  await page.mouse.up();

  detail = await openTaskDetail(row);
  await page.mouse.move(box.x + 12, box.y + 12);
  await page.mouse.down();
  await expect(row).toHaveAttribute('draggable', 'false');
  await description.dispatchEvent('pointercancel');
  await expect(row).toHaveAttribute('draggable', 'true');
  await page.mouse.up();
});

test('Escape restores the original title without hiding the field', async ({ page }) => {
  const row = page.locator('[data-task-id="task-1"]');
  const form = row.locator('[data-inline-task-edit-form]');
  const input = form.locator('[data-inline-task-edit-title]');

  await input.fill('Unsaved draft');
  await input.press('Escape');

  await expect(input).toHaveValue(title);
  await expect(form).toBeVisible();
  await expect(row).toBeFocused();
});

test('failed save keeps the title editable with native caret behavior', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  await page.route('http://caldo.test/tasks/task-1', async (route) => {
    await route.fulfill({ status: 502, contentType: 'text/plain', body: 'write failed' });
  });

  const input = page.locator('[data-inline-task-edit-title]');
  await input.fill('Failed title');
  await input.press('Enter');

  await expect(page.locator('[data-inline-task-edit-error]')).toContainText('Aufgabe konnte nicht gespeichert werden.');
  await expect(input).toHaveValue('Failed title');
  await expect(input).toBeFocused();
  await expect(input).toHaveAttribute('aria-invalid', 'true');

  const box = await input.boundingBox();
  expect(box).not.toBeNull();
  await input.click({ position: { x: box.width * 0.1, y: box.height / 2 } });
  const offset = await selectionStart(input);
  expect(offset).toBeGreaterThan(0);
  expect(offset).toBeLessThan('Failed title'.length);
  expect(pageErrors).toEqual([]);
});

async function selectionStart(input) {
  return input.evaluate((element) => element.selectionStart);
}

async function clickAtFractions(control, fractions) {
  const box = await control.boundingBox();
  expect(box).not.toBeNull();
  const offsets = [];
  for (const fraction of fractions) {
    await control.click({ position: { x: box.width * fraction, y: Math.min(box.height / 2, 12) } });
    offsets.push(await selectionStart(control));
  }
  return offsets;
}

async function openTaskDetail(row) {
  await row.locator('[data-task-detail-open]').click();
  const detail = row.locator('[data-task-detail-dialog]');
  await expect(detail).toBeVisible();
  return detail;
}
