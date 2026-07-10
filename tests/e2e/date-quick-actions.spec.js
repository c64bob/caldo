const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');

const timezoneCases = [
  {
    timezoneId: 'America/Los_Angeles',
    now: '2026-07-11T06:30:00Z',
    expected: {
      '0': '2026-07-10',
      '1': '2026-07-11',
      'next-monday': '2026-07-13',
      'next-weekend': '2026-07-11'
    }
  },
  {
    timezoneId: 'Pacific/Auckland',
    now: '2026-07-10T12:30:00Z',
    expected: {
      '0': '2026-07-11',
      '1': '2026-07-12',
      'next-monday': '2026-07-13',
      'next-weekend': '2026-07-18'
    }
  }
];

test('due-date quick actions use the browser local calendar day', async ({ browser }) => {
  for (const timezoneCase of timezoneCases) {
    const context = await browser.newContext({ timezoneId: timezoneCase.timezoneId });
    const page = await context.newPage();
    await page.clock.install({ time: new Date(timezoneCase.now) });
    await page.setContent(`
      <div class="caldo-date-dropdown">
        <button type="button" data-date-dropdown-trigger aria-expanded="true">Datum</button>
        <div class="caldo-date-dropdown-menu">
          <button type="button" data-date-dropdown-action data-date-days="0">Heute</button>
          <button type="button" data-date-dropdown-action data-date-days="1">Morgen</button>
          <button type="button" data-date-dropdown-action data-date-days="next-monday">Nächster Montag</button>
          <button type="button" data-date-dropdown-action data-date-days="next-weekend">Nächstes Wochenende</button>
        </div>
        <form onsubmit="event.preventDefault(); this.dataset.submitted = 'true'">
          <input data-date-hidden-input name="due_date">
        </form>
      </div>
    `);
    await page.addScriptTag({ path: appScript });

    for (const [action, expectedDate] of Object.entries(timezoneCase.expected)) {
      await page.locator('.caldo-date-dropdown-menu').evaluate((menu) => { menu.hidden = false; });
      await page.locator(`[data-date-days="${action}"]`).click();
      await expect(page.locator('[data-date-hidden-input]')).toHaveValue(expectedDate);
      await expect(page.locator('form')).toHaveAttribute('data-submitted', 'true');
    }

    await context.close();
  }
});
