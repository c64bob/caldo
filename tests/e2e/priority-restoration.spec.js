const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');

test('restored priority controls are reconciled from server-selected options', async ({ page }) => {
  await page.setContent(`
    <ul>
      <li data-task-id="task-high">
        <select class="caldo-task-priority-select caldo-task-priority-p1" data-inline-task-priority-select aria-label="High priority">
          <option value="">Keine Priorität</option>
          <option value="1" selected>P1 Hoch</option>
          <option value="5">P2 Mittel</option>
          <option value="9">P3 Niedrig</option>
        </select>
      </li>
      <li data-task-id="task-low">
        <select class="caldo-task-priority-select caldo-task-priority-p3" data-inline-task-priority-select aria-label="Low priority">
          <option value="">Keine Priorität</option>
          <option value="1">P1 Hoch</option>
          <option value="5">P2 Mittel</option>
          <option value="9" selected>P3 Niedrig</option>
        </select>
      </li>
    </ul>
  `);
  await page.addScriptTag({ path: appScript });

  const highPriority = page.getByRole('combobox', { name: 'High priority' });
  const lowPriority = page.getByRole('combobox', { name: 'Low priority' });

  await highPriority.evaluate((control) => { control.value = '5'; });
  await lowPriority.evaluate((control) => { control.value = '1'; });
  await expect(highPriority).toHaveValue('5');
  await expect(lowPriority).toHaveValue('1');

  await page.evaluate(() => window.dispatchEvent(new PageTransitionEvent('pageshow')));

  await expect(highPriority).toHaveValue('1');
  await expect(highPriority).toHaveClass(/caldo-task-priority-p1/);
  await expect(highPriority).not.toHaveClass(/caldo-task-priority-p2/);
  await expect(lowPriority).toHaveValue('9');
  await expect(lowPriority).toHaveClass(/caldo-task-priority-p3/);
  await expect(lowPriority).not.toHaveClass(/caldo-task-priority-p1/);
});
