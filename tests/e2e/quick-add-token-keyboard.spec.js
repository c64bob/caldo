const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');

test('Quick Add project and label suggestions support input-owned keyboard selection', async ({ page }) => {
  await page.setContent(`
    <dialog open data-quick-add-overlay data-quick-add-root>
      <form data-quick-add-overlay-form onsubmit="event.preventDefault()">
        <div>
          <input
            data-quick-add-input
            data-quick-add-overlay-input
            role="combobox"
            aria-autocomplete="list"
            aria-expanded="false"
            aria-controls="quick-add-token-suggestions"
          >
        </div>
      </form>
    </dialog>
  `);
  await page.evaluate(() => {
    window.fetch = () => Promise.resolve({
      ok: true,
      json: () => Promise.resolve({
        projects: [
          { id: 'work', name: 'Work' },
          { id: 'workshop', name: 'Workshop' },
          { id: 'personal', name: 'Personal' }
        ],
        labels: [
          { name: 'urgent' },
          { name: 'reviewed' },
          { name: 'home' }
        ]
      })
    });
  });
  await page.addScriptTag({ path: appScript });

  const input = page.getByRole('combobox');
  await input.focus();
  await input.fill('Plan #w');
  const listbox = page.getByRole('listbox');
  await expect(listbox).toBeVisible();
  await expect(input).toBeFocused();
  await expect(input).toHaveAttribute('aria-expanded', 'true');
  const work = listbox.locator('[role="option"][data-quick-add-token-name="Work"]');
  await expect(work).toHaveAttribute('aria-selected', 'true');
  await expect(work).toHaveClass(/caldo-quick-add-token-item-active/);

  await input.press('ArrowDown');
  const workshop = listbox.locator('[role="option"][data-quick-add-token-name="Workshop"]');
  await expect(workshop).toHaveAttribute('aria-selected', 'true');
  await expect(workshop).toHaveClass(/caldo-quick-add-token-item-active/);
  await expect(input).toHaveAttribute('aria-activedescendant', await workshop.getAttribute('id'));
  await expect(input).toBeFocused();
  await input.press('Enter');
  await expect(input).toHaveValue('Plan #Workshop ');
  await expect(listbox).toBeHidden();

  await input.fill('Plan @');
  await expect(listbox).toBeVisible();
  await input.press('End');
  await expect(listbox.getByRole('option', { name: '@home' })).toHaveAttribute('aria-selected', 'true');
  await input.press('Home');
  await expect(listbox.getByRole('option', { name: '@urgent' })).toHaveAttribute('aria-selected', 'true');
  await input.press('ArrowUp');
  await input.press('Enter');
  await expect(input).toHaveValue('Plan @home ');

  await input.fill('Plan #');
  await expect(listbox).toBeVisible();
  await input.press('Escape');
  await expect(listbox).toBeHidden();
  await expect(input).toBeFocused();

  await input.fill('Plan @rev');
  await expect(listbox).toBeVisible();
  await listbox.getByRole('option', { name: '@reviewed' }).click();
  await expect(input).toHaveValue('Plan @reviewed ');
  await expect(input).toBeFocused();
});
