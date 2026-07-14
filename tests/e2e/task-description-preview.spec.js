const path = require('node:path');
const { test, expect } = require('@playwright/test');

const appScript = path.resolve(__dirname, '../../web/assets/app.js');

test.beforeEach(async ({ page }) => {
  await page.setContent(`
    <ul>
      <li data-task-id="task-description">
        <div class="caldo-task-description">
          First line<br><strong>Strong line</strong> with <em>emphasis</em>,
          <code>code</code>, and
          <a class="caldo-task-description-link" href="https://example.com/spec" target="_blank" rel="noopener noreferrer">safe link</a>
        </div>
      </li>
    </ul>
    <form data-conflict-manual-form>
      <fieldset data-conflict-field-source="description">
        <label data-conflict-source-option="local" data-conflict-option-display="Local raw">
          <input type="radio" name="description_source" value="local">
          <span data-conflict-description-rendered>Local <em>emphasis</em></span>
        </label>
        <label data-conflict-source-option="remote" data-conflict-option-display="Remote raw">
          <input type="radio" name="description_source" value="remote" checked>
          <span data-conflict-description-rendered>Remote <strong>strong</strong><br>second line</span>
        </label>
        <label data-conflict-source-option="manual">
          <input type="radio" name="description_source" value="manual">
          <textarea data-conflict-manual-input="description" data-conflict-manual-empty="Keine Beschreibung">Manual **raw**\\ntext</textarea>
        </label>
      </fieldset>
      <section data-conflict-manual-preview>
        <dd data-conflict-preview-value="description"></dd>
      </section>
    </form>
  `);
  await page.addScriptTag({ path: appScript });
});

test('hover preview preserves safe server-rendered line breaks and Markdown', async ({ page }) => {
  const row = page.locator('[data-task-id="task-description"]');
  await row.hover();

  const preview = row.locator('.caldo-task-hover-preview-description');
  await expect(preview).toBeVisible();
  await expect(preview.locator('br')).toHaveCount(1);
  await expect(preview.locator('strong')).toHaveText('Strong line');
  await expect(preview.locator('em')).toHaveText('emphasis');
  await expect(preview.locator('code')).toHaveText('code');
  await expect(preview.locator('a')).toHaveAttribute('href', 'https://example.com/spec');
  await expect(preview.locator('a')).toHaveAttribute('rel', 'noopener noreferrer');
});

test('conflict preview preserves rendered persisted descriptions and raw manual edits', async ({ page }) => {
  const preview = page.locator('[data-conflict-preview-value="description"]');
  await expect(preview).toContainText('Remote strong');
  await expect(preview.locator('strong')).toHaveText('strong');
  await expect(preview.locator('br')).toHaveCount(1);

  await page.locator('input[name="description_source"][value="local"]').check();
  await expect(preview.locator('em')).toHaveText('emphasis');
  await expect(preview.locator('strong')).toHaveCount(0);

  const manual = page.locator('[data-conflict-manual-input="description"]');
  await manual.fill('Changed **raw**\\nvalue');
  await expect(preview).toHaveText('Changed **raw**\\nvalue');
  await expect(preview.locator('strong')).toHaveCount(0);
});
