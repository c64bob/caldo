const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');
const htmxScript = path.resolve(__dirname, '../../web/static/htmx.449317a.min.js');

test('detail completion sends exactly one direct completion request', async ({ page }) => {
  let requestCount = 0;
  const pageErrors = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  await page.route('**/tasks/task-1/complete', async (route) => {
    requestCount += 1;
    await route.fulfill({ status: 200, contentType: 'text/html', body: '' });
  });
  await setTaskDetailCompletionFixture(page);

  const detail = await openTaskDetail(page);
  await detail.getByRole('button', { name: 'Erledigt' }).click();

  await expect.poll(() => requestCount).toBe(1);
  await expect(page.locator('[data-task-detail-dialog]')).toHaveCount(0);
  await page.waitForTimeout(100);
  expect(requestCount).toBe(1);
  expect(pageErrors).toEqual([]);
});

for (const failure of [
  { status: 409, message: 'Aufgabe konnte nicht erledigt werden. Aufgabe prüfen.' },
  { status: 502, message: 'Aufgabe konnte nicht auf dem CalDAV-Server erledigt werden.' }
]) {
  test(`detail completion keeps the pane open after status ${failure.status}`, async ({ page }) => {
    let requestCount = 0;
    await page.route('**/tasks/task-1/complete', async (route) => {
      requestCount += 1;
      await route.fulfill({ status: failure.status, contentType: 'text/plain', body: 'write failed' });
    });
    await setTaskDetailCompletionFixture(page);

    const detail = await openTaskDetail(page);
    await detail.getByRole('button', { name: 'Erledigt' }).click();

    await expect.poll(() => requestCount).toBe(1);
    await expect(detail).toBeVisible();
    await expect(detail.locator('[data-task-detail-error]')).toHaveText(failure.message);
    await expect.poll(() => browserWouldBlockUnload(page)).toBe(false);
  });
}

test('dirty task details block completion without saving or discarding fields', async ({ page }) => {
  let requestCount = 0;
  await page.route('**/tasks/task-1/complete', async (route) => {
    requestCount += 1;
    await route.fulfill({ status: 200, contentType: 'text/html', body: '' });
  });
  await setTaskDetailCompletionFixture(page);

  const detail = await openTaskDetail(page);
  const title = detail.locator('[name="title"]');
  await title.fill('Unsaved title');
  await detail.getByRole('button', { name: 'Erledigt' }).click();

  await expect.poll(() => requestCount).toBe(0);
  await expect(title).toHaveValue('Unsaved title');
  await expect(detail.locator('[data-task-detail-error]')).toHaveText('Ungespeicherte Änderungen zuerst speichern oder mit Schließen verwerfen.');

  await detail.getByRole('button', { name: 'Schließen' }).click();
  const reopened = await openTaskDetail(page);
  await expect(reopened.locator('[name="title"]')).toHaveValue('Persisted title');
  await reopened.getByRole('button', { name: 'Erledigt' }).click();
  await expect.poll(() => requestCount).toBe(1);
});

test('parent detail completion opens the existing subtask decision dialog', async ({ page }) => {
  let requestCount = 0;
  await page.route('**/tasks/task-1/complete', async (route) => {
    requestCount += 1;
    await route.fulfill({ status: 200, contentType: 'text/html', body: '' });
  });
  await setTaskDetailCompletionFixture(page, { needsDecision: true });

  const detail = await openTaskDetail(page);
  await detail.getByRole('button', { name: 'Erledigt' }).click();

  await expect(detail).toBeHidden();
  const decision = page.locator('[data-task-complete-dialog]');
  await expect(decision).toBeVisible();
  await expect(decision.locator('[data-task-complete-cancel]')).toBeFocused();
  expect(requestCount).toBe(0);
});

async function setTaskDetailCompletionFixture(page, options = {}) {
  await page.route('http://caldo.test/', async (route) => {
    await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
  });
  await page.goto('http://caldo.test/');
  const decisionControl = options.needsDecision
    ? '<button type="button" data-task-complete-open aria-controls="task-complete-task-1" aria-expanded="false">Complete decision</button>'
    : '<input type="checkbox" data-task-completion-checkbox data-task-completion-state-control aria-label="Aufgabe erledigen">';
  await page.setContent(`
    <meta name="csrf-token" content="test-token">
    <div id="write-status" data-write-status></div>
    <div id="notifications"></div>
    <ul>
      <li class="caldo-task-row" data-task-id="task-1" data-task-status="needs-action">
        <div class="caldo-task-row-grid">
          <form method="post" action="/tasks/task-1/complete" hx-post="/tasks/task-1/complete" hx-swap="none" data-refresh-after-write="true" data-task-action-form>
            <input type="hidden" name="expected_version" value="3">
            ${decisionControl}
          </form>
          <button type="button" data-task-detail-open aria-controls="task-detail-task-1" aria-expanded="false">Details</button>
        </div>
        <p data-task-action-error role="alert" hidden></p>
        <dialog id="task-detail-task-1" data-task-detail-dialog>
          <p data-task-detail-error role="alert" aria-live="polite" hidden></p>
          <form data-task-detail-form>
            <input type="hidden" name="expected_version" value="3">
            <input type="text" name="title" value="Persisted title" data-task-detail-title>
            <button type="button" data-task-detail-complete>Erledigt</button>
            <button type="button" data-task-detail-close>Schließen</button>
          </form>
        </dialog>
        <dialog id="task-complete-task-1" data-task-complete-dialog>
          <p data-task-complete-error role="alert" hidden></p>
          <button type="button" data-task-complete-close data-task-complete-cancel>Abbrechen</button>
        </dialog>
      </li>
    </ul>
  `);
  await page.addScriptTag({ path: htmxScript });
  await page.addScriptTag({ path: appScript });
  await page.evaluate(() => window.htmx.process(document.body));
}

async function openTaskDetail(page) {
  await page.getByRole('button', { name: 'Details' }).click();
  const detail = page.locator('[data-task-detail-dialog]');
  await expect(detail).toBeVisible();
  return detail;
}

async function browserWouldBlockUnload(page) {
  return page.evaluate(() => {
    const event = new Event('beforeunload', { cancelable: true });
    const dispatched = window.dispatchEvent(event);
    return !dispatched || event.defaultPrevented;
  });
}
