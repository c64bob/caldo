const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');

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

function taskRow(id, title) {
  return `
    <li data-task-id="${id}" data-task-status="needs-action">
      <form action="/tasks/${id}/complete" data-task-action-form>
        <input type="hidden" name="expected_version" value="1">
      </form>
      <form data-inline-task-edit-form data-inline-task-edit-persistent>
        <input name="title" data-inline-task-edit-title aria-label="${title}" value="${title}">
      </form>
      <label>
        <input type="checkbox" data-bulk-select-control aria-label="Für Mehrfachbearbeitung auswählen: ${title}">
      </label>
    </li>
  `;
}
