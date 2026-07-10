const path = require('node:path');
const { test, expect } = require('@playwright/test');

test('focus refresh preserves a form dirtied while its fragment is in flight', async ({ page }) => {
  await page.setContent(`
    <meta name="csrf-token" content="test-token">
    <ul>
      <li data-task-id="task-1" data-server-version="1">
        <div data-inline-task-edit-scope data-inline-task-edit-focus="title">
          <button type="button" data-inline-task-edit-open data-inline-task-edit-focus="title" aria-expanded="false">Old title</button>
          <form data-inline-task-edit-form data-inline-task-edit-kind="title" hidden>
            <input type="hidden" name="expected_version" value="1">
            <input type="text" name="title" value="Old title" data-inline-task-edit-title>
          </form>
        </div>
        <p data-inline-task-edit-error hidden></p>
        <p data-task-action-error hidden></p>
      </li>
    </ul>
  `);

  await page.evaluate(() => {
    window.__fragmentStarted = false;
    window.__resolveFragment = null;
    window.fetch = (input) => {
      const url = String(input);
      if (url.startsWith('/api/tasks/versions')) {
        return Promise.resolve(new Response(JSON.stringify({
          tasks: [{ task_id: 'task-1', server_version: 2 }]
        }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' }
        }));
      }
      if (url === '/tasks/task-1') {
        window.__fragmentStarted = true;
        return new Promise((resolve) => {
          window.__resolveFragment = () => resolve(new Response(
            '<li data-task-id="task-1" data-server-version="2"><p>Remote title</p></li>',
            { status: 200, headers: { 'Content-Type': 'text/html' } }
          ));
        });
      }
      return Promise.resolve(new Response('', { status: 200 }));
    };
  });
  await page.addScriptTag({ path: path.resolve(__dirname, '../../web/assets/app.js') });

  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect.poll(() => page.evaluate(() => window.__fragmentStarted)).toBe(true);

  const row = page.locator('[data-task-id="task-1"]');
  await row.locator('[data-inline-task-edit-open]').click();
  const input = row.locator('[data-inline-task-edit-title]');
  await input.fill('Unsaved draft');
  await page.evaluate(() => window.__resolveFragment());

  await expect(input).toHaveValue('Unsaved draft');
  await expect(row).toHaveAttribute('data-stale-local-changes', 'true');
  await expect(row.locator('[data-inline-task-edit-error]')).toContainText('Aufgabe wurde in einem anderen Tab geändert');
  await expect(row).not.toContainText('Remote title');
});
