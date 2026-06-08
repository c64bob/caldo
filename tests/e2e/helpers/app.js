const { expect } = require('@playwright/test');
const { appURL, readState } = require('./state');

async function gotoApp(page, pathname) {
  await page.goto(appURL(pathname), { waitUntil: 'domcontentloaded' });
}

async function appFormRequest(page, method, pathname, fields = {}, options = {}) {
  const token = await currentCSRFToken(page);
  const state = readState();
  const headers = {
    [state.proxyUserHeader]: 'e2e-user',
    'X-CSRF-Token': token,
    Cookie: `caldo_csrf=${token}`,
    'Content-Type': 'application/x-www-form-urlencoded'
  };
  if (options.tabID) {
    headers['X-Tab-ID'] = options.tabID;
  }

  return page.request.fetch(appURL(pathname), {
    method,
    headers,
    data: new URLSearchParams(fields).toString(),
    maxRedirects: options.maxRedirects ?? 0,
    failOnStatusCode: false
  });
}

async function currentCSRFToken(page) {
  const token = await page.locator('meta[name="csrf-token"]').getAttribute('content');
  if (token) {
    return token;
  }

  const cookies = await page.context().cookies(appURL('/'));
  const csrfCookie = cookies.find((cookie) => cookie.name === 'caldo_csrf');
  if (csrfCookie && csrfCookie.value) {
    return csrfCookie.value;
  }

  const state = readState();
  const response = await page.request.get(appURL('/setup/'), {
    headers: { [state.proxyUserHeader]: 'e2e-user' },
    failOnStatusCode: false
  });
  const headerToken = response.headers()['x-csrf-token'];
  if (headerToken) {
    return headerToken;
  }

  throw new Error('csrf token not found on current page or browser context');
}

async function taskIDFromSearch(page, title) {
  await gotoApp(page, `/search?q=${encodeURIComponent(title)}`);
  const row = page.locator('[data-task-id]').filter({ hasText: title }).first();
  await expect(row).toBeVisible();
  return row.getAttribute('data-task-id');
}

async function expectSearchResult(page, title) {
  await gotoApp(page, `/search?q=${encodeURIComponent(title)}`);
  await expect(page.locator('[data-search-results]').filter({ hasText: title })).toBeVisible();
}

async function expectNoSearchResult(page, title) {
  await gotoApp(page, `/search?q=${encodeURIComponent(title)}`);
  await expect(page.locator('[data-search-results]').filter({ hasText: title })).toHaveCount(0);
}

async function waitForSearchResult(page, title) {
  await expect.poll(async () => searchHasResults(page, title)).toBe(true);
  await expectSearchResult(page, title);
}

async function waitForNoSearchResult(page, title) {
  await expect.poll(async () => searchHasResults(page, title)).toBe(false);
  await expectNoSearchResult(page, title);
}

async function taskVersion(page, taskID) {
  const response = await page.request.get(appURL(`/api/tasks/versions?ids=${encodeURIComponent(taskID)}`), {
    headers: { [readState().proxyUserHeader]: 'e2e-user' },
    failOnStatusCode: false
  });
  expect(response.status()).toBe(200);
  const payload = await response.json();
  const task = payload.tasks.find((item) => item.task_id === taskID);
  if (!task) {
    throw new Error(`task version not found for ${taskID}`);
  }
  return task.server_version;
}

async function manualSync(page) {
  const response = await appFormRequest(page, 'POST', '/sync/manual');
  expect(response.status()).toBe(200);
}

async function searchHasResults(page, title) {
  const response = await page.request.get(appURL(`/search?q=${encodeURIComponent(title)}`), {
    headers: { [readState().proxyUserHeader]: 'e2e-user' },
    failOnStatusCode: false
  });
  expect(response.status()).toBe(200);
  const html = await response.text();
  return html.includes('data-search-results');
}

module.exports = {
  appFormRequest,
  expectNoSearchResult,
  expectSearchResult,
  gotoApp,
  manualSync,
  taskIDFromSearch,
  taskVersion,
  waitForNoSearchResult,
  waitForSearchResult
};
