const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');

test('D opens a keyboard-navigable due-date menu', async ({ page }) => {
  await page.setContent(`
    <div data-task-id="task-1">
      <div class="caldo-date-dropdown">
        <form onsubmit="event.preventDefault(); this.dataset.submitted = 'true'">
          <input name="due_date" data-date-hidden-input>
        </form>
        <button type="button" class="caldo-task-date-trigger" data-date-dropdown-trigger aria-expanded="false" aria-haspopup="menu" aria-controls="date-menu">Datum</button>
        <div id="date-menu" class="caldo-date-dropdown-menu" role="menu" hidden>
          <button type="button" role="menuitem" tabindex="-1" data-date-dropdown-action data-date-days="0">Heute</button>
          <button type="button" role="menuitem" tabindex="-1" data-date-dropdown-action data-date-days="1">Morgen</button>
          <button type="button" role="menuitem" tabindex="-1" data-date-dropdown-action data-date-days="2">Übermorgen</button>
          <button type="button" role="menuitem" tabindex="-1" data-date-dropdown-action data-date-days="next-monday">Nächster Montag</button>
          <button type="button" role="menuitem" tabindex="-1" data-date-dropdown-action data-date-days="next-weekend">Nächstes Wochenende</button>
          <label role="menuitem" tabindex="-1">Benutzerdefiniertes Datum<input type="date" data-date-custom-input></label>
          <button type="button" role="menuitem" tabindex="-1" data-date-dropdown-action data-date-days="clear">Kein Datum</button>
        </div>
      </div>
    </div>
  `);
  await page.addScriptTag({ path: appScript });

  const trigger = page.locator('[data-date-dropdown-trigger]');
  const menu = page.getByRole('menu');
  await trigger.focus();
  await page.keyboard.press('d');
  await expect(menu).toBeVisible();
  await expect(menu.getByRole('menuitem', { name: 'Heute' })).toBeFocused();

  await page.keyboard.press('ArrowDown');
  await expect(menu.getByRole('menuitem', { name: 'Morgen', exact: true })).toBeFocused();
  await page.keyboard.press('End');
  await expect(menu.getByRole('menuitem', { name: 'Kein Datum' })).toBeFocused();
  await page.keyboard.press('Home');
  await page.keyboard.press('ArrowDown');
  await page.keyboard.press('ArrowDown');
  await expect(menu.getByRole('menuitem', { name: 'Übermorgen' })).toBeFocused();
  await page.keyboard.press('Enter');

  await expect(menu).toBeHidden();
  await expect(trigger).toBeFocused();
  await expect(page.locator('form')).toHaveAttribute('data-submitted', 'true');
  await expect(page.locator('[data-date-hidden-input]')).not.toHaveValue('');

  await page.keyboard.press('d');
  await expect(menu.getByRole('menuitem', { name: 'Heute' })).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(menu).toBeHidden();
  await expect(trigger).toBeFocused();
});
