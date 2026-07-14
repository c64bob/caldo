const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');
const htmxScript = path.resolve(__dirname, '../../web/static/htmx.449317a.min.js');

test('direct completion sends one request without an HTMX exception', async ({ page }) => {
  const pageErrors = [];
  let requestCount = 0;
  page.on('pageerror', (error) => pageErrors.push(error.message));
  await page.route('**/tasks/task-1/complete', async (route) => {
    requestCount += 1;
    await new Promise((resolve) => setTimeout(resolve, 100));
    await route.fulfill({ status: 200, contentType: 'text/html', body: '' });
  });
  await setCompletionFixture(page);

  await page.locator('[data-task-action-form]').evaluate((form) => form.requestSubmit());

  await expect.poll(() => requestCount).toBe(1);
  await expect(page.locator('[data-task-completion-checkbox]')).toBeEnabled();
  await expect.poll(() => browserWouldBlockUnload(page)).toBe(false);
  expect(pageErrors).toEqual([]);
});

test('failed direct completion restores the checkbox and pending-write state', async ({ page }) => {
  const pageErrors = [];
  page.on('pageerror', (error) => pageErrors.push(error.message));
  await page.route('**/tasks/task-1/complete', async (route) => {
    await route.fulfill({ status: 502, contentType: 'text/plain', body: 'caldav write failed' });
  });
  await setCompletionFixture(page);

  await page.locator('[data-task-action-form]').evaluate((form) => form.requestSubmit());

  await expect(page.locator('[data-task-completion-checkbox]')).toBeEnabled();
  await expect(page.locator('[data-task-action-error]')).toBeVisible();
  await expect.poll(() => browserWouldBlockUnload(page)).toBe(false);
  expect(pageErrors).toEqual([]);
});

test('restored checkbox state is reconciled from the persisted task status', async ({ page }) => {
  await page.setContent(`
    <ul>
      <li data-task-id="task-next" data-task-status="needs-action">
        <input type="checkbox" checked data-task-completion-state-control aria-label="Next task">
      </li>
      <li data-task-id="task-completed" data-task-status="completed">
        <input type="checkbox" data-task-completion-state-control aria-label="Completed task">
      </li>
    </ul>
  `);

  await page.addScriptTag({ path: appScript });

  const nextTask = page.getByRole('checkbox', { name: 'Next task' });
  const completedTask = page.getByRole('checkbox', { name: 'Completed task' });
  await expect(nextTask).not.toBeChecked();
  await expect(completedTask).toBeChecked();

  await nextTask.evaluate((control) => { control.checked = true; });
  await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pageshow')));
  await expect(nextTask).not.toBeChecked();
});

async function setCompletionFixture(page) {
  await page.route('http://caldo.test/', async (route) => {
    await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
  });
  await page.goto('http://caldo.test/');
  await page.setContent(`
    <meta name="csrf-token" content="test-token">
    <div id="write-status" data-write-status></div>
    <div id="notifications"></div>
    <ul>
      <li data-task-id="task-1" data-task-status="needs-action">
        <form
          method="post"
          action="/tasks/task-1/complete"
          hx-post="/tasks/task-1/complete"
          hx-swap="none"
          hx-disabled-elt="find input[data-task-completion-checkbox]"
          data-task-action-form
        >
          <input type="hidden" name="expected_version" value="1">
          <label>
            <input type="checkbox" data-task-completion-checkbox data-task-completion-state-control aria-label="Aufgabe erledigen">
          </label>
        </form>
        <p data-task-action-error role="alert" hidden></p>
      </li>
    </ul>
  `);
  await page.addScriptTag({ path: htmxScript });
  await page.addScriptTag({ path: appScript });
  await page.evaluate(() => window.htmx.process(document.body));
}

async function browserWouldBlockUnload(page) {
  return page.evaluate(() => {
    const event = new Event('beforeunload', { cancelable: true });
    const dispatched = window.dispatchEvent(event);
    return !dispatched || event.defaultPrevented;
  });
}
