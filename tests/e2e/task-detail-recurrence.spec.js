const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');

test.beforeEach(async ({ page }) => {
  await page.setContent(`
    <ul>
      <li data-task-id="task-recurrence">
        <button type="button" data-task-detail-open aria-controls="task-detail-recurrence" aria-expanded="false">Details</button>
        <dialog id="task-detail-recurrence" data-task-detail-dialog>
          <form data-task-detail-form>
            <input type="text" name="title" value="Recurring task" data-task-detail-title>
            <details data-task-recurrence-section>
              <summary>Wiederholung <span>Wöchentlich</span></summary>
              <input type="hidden" name="repeat_update" value="1" disabled data-task-recurrence-update>
              <select name="repeat_freq" data-task-recurrence-control>
                <option value="NONE">Keine</option>
                <option value="WEEKLY" selected>Wöchentlich</option>
                <option value="DAILY">Täglich</option>
              </select>
            </details>
            <button type="button" data-task-detail-close>Schließen</button>
          </form>
        </dialog>
      </li>
    </ul>
  `);
  await page.addScriptTag({ path: appScript });
});

test('recurrence disclosure supports pointer and keyboard and resets on reopen', async ({ page }) => {
  const detail = await openTaskDetail(page);
  const recurrence = detail.locator('[data-task-recurrence-section]');
  const summary = recurrence.locator('summary');

  await expect(recurrence).not.toHaveAttribute('open', '');
  await expect(detail.locator('[name="repeat_freq"]')).toBeHidden();
  await summary.click();
  await expect(recurrence).toHaveAttribute('open', '');
  await expect(detail.locator('[name="repeat_freq"]')).toBeVisible();

  await summary.focus();
  await page.keyboard.press('Enter');
  await expect(recurrence).not.toHaveAttribute('open', '');
  await page.keyboard.press('Space');
  await expect(recurrence).toHaveAttribute('open', '');

  await detail.getByRole('button', { name: 'Schließen' }).click();
  await expect(detail).toBeHidden();
  const reopened = await openTaskDetail(page);
  await expect(reopened.locator('[data-task-recurrence-section]')).not.toHaveAttribute('open', '');
});

test('collapsing recurrence keeps edits and the explicit update marker', async ({ page }) => {
  const detail = await openTaskDetail(page);
  const recurrence = detail.locator('[data-task-recurrence-section]');
  const frequency = detail.locator('[name="repeat_freq"]');
  const marker = detail.locator('[data-task-recurrence-update]');

  await recurrence.locator('summary').click();
  await expect(marker).toBeDisabled();
  await frequency.selectOption('DAILY');
  await expect(marker).toBeEnabled();
  await recurrence.locator('summary').click();

  await expect(recurrence).not.toHaveAttribute('open', '');
  await expect(frequency).toHaveValue('DAILY');
  await expect(marker).toBeEnabled();
});

async function openTaskDetail(page) {
  await page.getByRole('button', { name: 'Details' }).click();
  const detail = page.locator('[data-task-detail-dialog]');
  await expect(detail).toBeVisible();
  return detail;
}
