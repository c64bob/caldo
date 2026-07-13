const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');
const htmxScript = path.resolve(__dirname, '../../web/static/htmx.449317a.min.js');

for (const action of ['apply', 'reset']) {
  test(`display preference ${action} redirects without unload protection`, async ({ page }) => {
    const requestGate = deferred();
    let requestStarted = false;
    const dialogs = [];
    page.on('dialog', async (dialog) => {
      dialogs.push(dialog.type());
      await dialog.dismiss();
    });
    await page.route('http://caldo.test/task-view-preferences', async (route) => {
      requestStarted = true;
      await requestGate.promise;
      await route.fulfill({
        status: 204,
        headers: { 'HX-Redirect': '/upcoming' },
        body: ''
      });
    });
    await page.route('http://caldo.test/upcoming', async (route) => {
      await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><title>Upcoming</title>' });
    });
    await setFixture(page, `
      <form
        method="post"
        action="/task-view-preferences"
        hx-post="/task-view-preferences"
        hx-swap="none"
        hx-disabled-elt="find button"
        data-task-display-form
      >
        <input type="hidden" name="view_kind" value="today">
        <input type="hidden" name="sort_by" value="name">
        <input type="hidden" name="sort_order" value="asc">
        <input type="hidden" name="group_by" value="none">
        <button type="submit" name="action" value="${action === 'reset' ? 'reset' : ''}">${action}</button>
      </form>
    `);

    await page.getByRole('button', { name: action }).click();
    await expect.poll(() => requestStarted).toBe(true);
    await expect.poll(() => browserWouldBlockUnload(page)).toBe(false);

    requestGate.resolve();
    await expect(page).toHaveURL('http://caldo.test/upcoming');
    expect(dialogs).toEqual([]);
  });
}

for (const responseStatus of [204, 502]) {
  test(`task mutation status ${responseStatus} clears unload protection before navigation`, async ({ page }) => {
    const requestGate = deferred();
    let requestStarted = false;
    const dialogs = [];
    page.on('dialog', async (dialog) => {
      dialogs.push(dialog.type());
      await dialog.dismiss();
    });
    await page.route('http://caldo.test/tasks/task-1', async (route) => {
      requestStarted = true;
      await requestGate.promise;
      await route.fulfill({ status: responseStatus, contentType: 'text/plain', body: responseStatus >= 400 ? 'write failed' : '' });
    });
    await page.route('http://caldo.test/upcoming', async (route) => {
      await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><title>Upcoming</title>' });
    });
    await setFixture(page, taskMutationForm('task-1'));

    await page.getByRole('button', { name: 'save task-1' }).click();
    await expect.poll(() => requestStarted).toBe(true);
    await expect.poll(() => browserWouldBlockUnload(page)).toBe(true);

    requestGate.resolve();
    await expect.poll(() => browserWouldBlockUnload(page)).toBe(false);
    await page.getByRole('link', { name: 'Upcoming' }).click();
    await expect(page).toHaveURL('http://caldo.test/upcoming');
    expect(dialogs).toEqual([]);
  });
}

test('concurrent task writes remain protected until every request finishes', async ({ page }) => {
  const firstGate = deferred();
  const secondGate = deferred();
  let startedRequests = 0;
  await page.route('http://caldo.test/tasks/task-1', async (route) => {
    startedRequests += 1;
    await firstGate.promise;
    await route.fulfill({ status: 204, body: '' });
  });
  await page.route('http://caldo.test/tasks/task-2', async (route) => {
    startedRequests += 1;
    await secondGate.promise;
    await route.fulfill({ status: 204, body: '' });
  });
  await setFixture(page, taskMutationForm('task-1') + taskMutationForm('task-2'));

  await page.getByRole('button', { name: 'save task-1' }).click();
  await page.getByRole('button', { name: 'save task-2' }).click();
  await expect.poll(() => startedRequests).toBe(2);
  await expect.poll(() => browserWouldBlockUnload(page)).toBe(true);

  const firstResponse = page.waitForResponse('http://caldo.test/tasks/task-1');
  firstGate.resolve();
  await firstResponse;
  await expect.poll(() => browserWouldBlockUnload(page)).toBe(true);

  secondGate.resolve();
  await expect.poll(() => browserWouldBlockUnload(page)).toBe(false);
});

test('aborted task mutation clears unload protection', async ({ page }) => {
  const requestGate = deferred();
  let requestStarted = false;
  await page.route('http://caldo.test/tasks/task-1', async (route) => {
    requestStarted = true;
    await requestGate.promise;
    await route.abort('failed');
  });
  await setFixture(page, taskMutationForm('task-1'));

  await page.getByRole('button', { name: 'save task-1' }).click();
  await expect.poll(() => requestStarted).toBe(true);
  await expect.poll(() => browserWouldBlockUnload(page)).toBe(true);

  requestGate.resolve();
  await expect.poll(() => browserWouldBlockUnload(page)).toBe(false);
});

async function setFixture(page, body) {
  await page.route('http://caldo.test/', async (route) => {
    await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
  });
  await page.goto('http://caldo.test/');
  await page.setContent(`
    <meta name="csrf-token" content="test-token">
    <div id="write-status" data-write-status></div>
    <div id="notifications"></div>
    <a href="/upcoming">Upcoming</a>
    ${body}
  `);
  await page.addScriptTag({ path: htmxScript });
  await page.addScriptTag({ path: appScript });
  await page.evaluate(() => window.htmx.process(document.body));
}

function taskMutationForm(taskID) {
  return `
    <div data-task-id="${taskID}">
      <form
        method="post"
        action="/tasks/${taskID}"
        hx-patch="/tasks/${taskID}"
        hx-swap="none"
        hx-disabled-elt="find button"
        data-task-action-form
      >
        <input type="hidden" name="expected_version" value="1">
        <button type="submit">save ${taskID}</button>
      </form>
      <p data-task-action-error role="alert" hidden></p>
    </div>
  `;
}

function deferred() {
  let resolve;
  const promise = new Promise((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

async function browserWouldBlockUnload(page) {
  return page.evaluate(() => {
    const event = new Event('beforeunload', { cancelable: true });
    const dispatched = window.dispatchEvent(event);
    return !dispatched || event.defaultPrevented;
  });
}
