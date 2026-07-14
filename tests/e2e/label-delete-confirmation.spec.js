const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');
const htmxScript = path.resolve(__dirname, '../../web/static/htmx.449317a.min.js');

test('label delete cancellation paths make no request and return focus', async ({ page }) => {
  let deleteRequests = 0;
  await page.route('http://caldo.test/labels/label-office', async (route) => {
    deleteRequests += 1;
    await route.fulfill({ status: 204, body: '' });
  });
  await setLabelDeleteFixture(page);

  const trigger = page.getByRole('button', { name: 'Label löschen' });
  const dialog = page.locator('[data-label-delete-dialog]');
  const cancel = dialog.getByRole('button', { name: 'Nein' });

  await trigger.click();
  await expect(dialog).toBeVisible();
  await expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await expect(cancel).toBeFocused();
  await expect(dialog).toHaveAttribute('aria-labelledby', 'label-delete-title-label-office');
  await expect(dialog).toHaveAttribute('aria-describedby', 'label-delete-description-label-office');
  await cancel.click();
  await expect(dialog).toBeHidden();
  await expect(trigger).toBeFocused();

  await trigger.click();
  await page.keyboard.press('Escape');
  await expect(dialog).toBeHidden();
  await expect(trigger).toBeFocused();

  await trigger.click();
  await dialog.evaluate((element) => element.click());
  await expect(dialog).toBeHidden();
  await expect(trigger).toBeFocused();
  expect(deleteRequests).toBe(0);
});

test('label delete confirmation sends one explicit request under repeated activation', async ({ page }) => {
  const requestGate = deferred();
  let deleteRequests = 0;
  await page.route('http://caldo.test/labels/label-office', async (route) => {
    deleteRequests += 1;
    await requestGate.promise;
    await route.fulfill({ status: 204, body: '' });
  });
  await setLabelDeleteFixture(page);

  await page.getByRole('button', { name: 'Label löschen' }).click();
  const confirm = page.getByRole('button', { name: 'Ja, löschen' });
  await confirm.evaluate((button) => {
    button.click();
    button.click();
  });

  await expect.poll(() => deleteRequests).toBe(1);
  requestGate.resolve();
  await expect.poll(() => deleteRequests).toBe(1);
});

test('a failed label deletion reopens the dialog with safe initial focus', async ({ page }) => {
  await setLabelDeleteFixture(page, true);

  const dialog = page.locator('[data-label-delete-dialog]');
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole('alert')).toHaveText('Label konnte nicht vollständig gelöscht werden.');
  await expect(dialog.getByRole('button', { name: 'Nein' })).toBeFocused();
  await expect(dialog).not.toHaveAttribute('data-label-delete-reopen', '');
});

async function setLabelDeleteFixture(page, reopen = false) {
  await page.route('http://caldo.test/', async (route) => {
    await route.fulfill({ status: 200, contentType: 'text/html', body: '<!doctype html><html><body></body></html>' });
  });
  await page.goto('http://caldo.test/');
  await page.setContent(`
    <meta name="csrf-token" content="test-token">
    <div id="write-status" data-write-status></div>
    <div id="notifications"></div>
    <ul>
      <li class="caldo-list-row">
        <button
          type="button"
          data-label-delete-open
          aria-controls="label-delete-dialog-label-office"
          aria-haspopup="dialog"
          aria-expanded="false"
        >Label löschen</button>
        <dialog
          id="label-delete-dialog-label-office"
          data-label-delete-dialog
          ${reopen ? 'data-label-delete-reopen' : ''}
          aria-labelledby="label-delete-title-label-office"
          aria-describedby="label-delete-description-label-office"
        >
          <h2 id="label-delete-title-label-office">Label löschen</h2>
          <p id="label-delete-description-label-office">Möchtest du das Label „Büro“ wirklich löschen? Es wird aus 2 Aufgaben entfernt.</p>
          ${reopen ? '<p role="alert">Label konnte nicht vollständig gelöscht werden.</p>' : ''}
          <form
            method="post"
            action="/labels/label-office"
            hx-delete="/labels/label-office"
            hx-swap="none"
            hx-sync="this:drop"
            hx-disabled-elt="find button"
            data-label-delete-form
          >
            <input type="hidden" name="confirmed" value="true">
            <button type="button" data-label-delete-close data-label-delete-cancel>Nein</button>
            <button type="submit">Ja, löschen</button>
          </form>
        </dialog>
      </li>
    </ul>
  `);
  await page.addScriptTag({ path: htmxScript });
  await page.addScriptTag({ path: appScript });
  await page.evaluate(() => window.htmx.process(document.body));
}

function deferred() {
  let resolve;
  const promise = new Promise((done) => {
    resolve = done;
  });
  return { promise, resolve };
}
