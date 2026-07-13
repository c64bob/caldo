const path = require('node:path');
const { test, expect } = require('@playwright/test');
const manifest = require('../../web/static/manifest.json');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');
const htmxScript = path.resolve(__dirname, '../../web/static/htmx.449317a.min.js');
const appStyles = path.resolve(__dirname, '../../web/static', manifest['app.css']);
const title = 'Alpha Bravo Charlie Delta Echo Foxtrot';

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
            </div>
          </div>
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
