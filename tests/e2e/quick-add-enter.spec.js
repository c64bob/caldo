const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');
const htmxScript = path.resolve(__dirname, '../../web/static/htmx.449317a.min.js');

for (const surface of ['overlay', 'page']) {
  test(`Quick Add ${surface} creates the latest input once with Enter`, async ({ page }) => {
    let createCount = 0;
    let createdTitle = '';

    await page.route('http://caldo.test/quick-add/preview', async (route) => {
      const values = new URLSearchParams(route.request().postData() || '');
      const text = (values.get('text') || '').trim();
      await new Promise((resolve) => setTimeout(resolve, 100));
      await route.fulfill({
        status: 200,
        contentType: 'text/html',
        body: quickAddPreviewFixture(surface, text)
      });
    });
    await page.route('http://caldo.test/tasks', async (route) => {
      createCount += 1;
      const values = new URLSearchParams(route.request().postData() || '');
      createdTitle = values.get('title') || '';
      await route.fulfill({ status: 201, contentType: 'text/plain', body: 'task created' });
    });
    await page.route('http://caldo.test/', async (route) => {
      await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
    });

    await page.goto('http://caldo.test/');
    await page.setContent(quickAddRootFixture(surface));
    await page.addScriptTag({ path: htmxScript });
    await page.addScriptTag({ path: appScript });
    await page.evaluate(() => window.htmx.process(document.body));

    const input = page.locator('[data-quick-add-input]');
    await input.fill('Latest task');
    await input.press('Enter');
    await input.press('Enter');

    await expect.poll(() => createCount).toBe(1);
    expect(createdTitle).toBe('Latest task');
    if (surface === 'overlay') {
      await expect(page.locator('[data-quick-add-overlay]')).toBeHidden();
    } else {
      await expect(input).not.toHaveAttribute('aria-busy', 'true');
    }
  });
}

function quickAddRootFixture(surface) {
  const rootStart = surface === 'overlay'
    ? '<dialog open data-quick-add-overlay data-quick-add-root>'
    : '<section data-quick-add-root>';
  const rootEnd = surface === 'overlay' ? '</dialog>' : '</section>';
  const target = surface === 'overlay' ? 'quick-add-overlay-preview' : 'quick-add-preview';
  return `
    ${rootStart}
      <form class="caldo-quick-add-form" action="/quick-add/preview" hx-post="/quick-add/preview" hx-target="#${target}" hx-swap="outerHTML" hx-sync="this:replace">
        <input name="text" data-quick-add-input aria-label="Task">
        <button type="submit">Preview</button>
      </form>
      <div id="${target}"></div>
    ${rootEnd}
  `;
}

function quickAddPreviewFixture(surface, text) {
  const target = surface === 'overlay' ? 'quick-add-overlay-preview' : 'quick-add-preview';
  const overlayAttribute = surface === 'overlay' ? 'data-quick-add-overlay-save-form' : '';
  return `
    <section id="${target}" data-quick-add-preview data-quick-add-source-text="${text}">
      <form action="/tasks" hx-post="/tasks" hx-swap="none" data-quick-add-save-form ${overlayAttribute}>
        <input name="title" value="${text}">
        <button type="submit">Save</button>
      </form>
    </section>
  `;
}
