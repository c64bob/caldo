const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');
const title = 'Alpha Bravo Charlie';

test('inline title editing preserves the clicked text position', async ({ page }) => {
  await page.setContent(`
    <style>
      button, input { box-sizing: border-box; font: 16px/24px monospace; }
      button { display: block; width: max-content; margin: 40px; padding: 0; }
      input { width: 240px; }
    </style>
    <div data-task-id="task-1">
      <div data-inline-task-edit-scope data-inline-task-edit-focus="title">
        <button type="button" data-inline-task-edit-open data-inline-task-edit-focus="title" aria-expanded="false">${title}</button>
        <form data-inline-task-edit-form data-inline-task-edit-kind="title" hidden onsubmit="event.preventDefault()">
          <input type="text" name="title" value="${title}" data-inline-task-edit-title>
        </form>
      </div>
      <p data-inline-task-edit-error hidden></p>
    </div>
  `);
  await page.addScriptTag({ path: appScript });

  const trigger = page.locator('[data-inline-task-edit-open]');
  const triggerBox = await trigger.boundingBox();
  expect(triggerBox).not.toBeNull();
  await trigger.click({ position: { x: triggerBox.width * 0.48, y: triggerBox.height / 2 } });

  const input = page.locator('[data-inline-task-edit-title]');
  await expect(input).toBeVisible();
  await expect(input).toBeFocused();
  const initialOffset = await input.evaluate((element) => element.selectionStart);
  expect(initialOffset).toBeGreaterThan(3);
  expect(initialOffset).toBeLessThan(title.length - 3);

  const inputBox = await input.boundingBox();
  expect(inputBox).not.toBeNull();
  await input.click({ position: { x: inputBox.width * 0.2, y: inputBox.height / 2 } });
  const movedOffset = await input.evaluate((element) => element.selectionStart);
  expect(movedOffset).toBeLessThan(initialOffset);

  await input.press('Escape');
  await expect(trigger).toBeVisible();
  await trigger.press('Enter');
  await expect(input).toBeFocused();
});
