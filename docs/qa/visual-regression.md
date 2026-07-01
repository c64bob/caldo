# Visual Regression QA

This process makes UI screenshots reviewable before merge. It uses only synthetic Playwright/Staging-CalDAV data and local artifacts under `test-results/`; do not commit screenshots or upload private screenshots from real accounts.

## When To Run

Run the visual review for pull requests that change:

- Templ templates or generated UI HTML
- Tailwind input CSS or static UI assets
- `web/assets/app.js` behavior that affects layout, dialogs, overlays, focus, sync status, Quick Add, search, settings, conflicts, or task rows
- Text that changes layout density or navigation labels

For non-UI backend-only changes, normal `npm run test:e2e` evidence is enough unless the change alters visible state, error messages, or routing.

## Required Screenshots

`npm run test:e2e:visual` captures and verifies the Chromium baseline set:

- Setup
- Inbox-equivalent/default project overview
- Today
- Upcoming
- Search
- Quick Add
- Conflicts
- Settings

Each view is captured at desktop, tablet, and mobile widths, for 24 required screenshots total. The screenshots are written to `test-results/e2e/chromium/baselines/` and a manifest is written to `test-results/e2e/chromium/visual-review-manifest.json`.

For Safari/WebKit-sensitive UI work, also run:

```bash
npm run test:e2e:visual:webkit
```

That creates the same manifest under `test-results/e2e/webkit/`.

## Local Review Loop

1. Run `npm run test:e2e:visual`.
2. Open the screenshots in `test-results/e2e/chromium/baselines/`.
3. Check the required views for layout shifts, clipped text, horizontal overflow, broken density, missing focus affordances, hidden actions, or unintended color/theme changes.
4. If the PR intentionally changes design, note the expected screenshot differences in the PR body.
5. If the differences are unintentional, treat them as regressions and fix before merge.

To compare two local runs, keep the previous manifest and pass it to the manifest checker after a new run:

```bash
node tests/e2e/visual-review.js --browser chromium --compare path/to/previous-visual-review-manifest.json
```

The comparison reports added, missing, and changed screenshots by filename, dimensions, byte size, and SHA-256 prefix. A changed hash is not automatically bad; it means the reviewer must classify the change as expected design change or regression.

## Review Classification

Expected design changes:

- Match the active story or PR scope.
- Preserve the required viewport coverage.
- Do not introduce private data into artifacts.
- Are called out in the PR with affected screenshot names.

Regressions:

- Contradict the story scope or UI principles.
- Hide or clip primary actions.
- Add horizontal page overflow or unreadable text.
- Break density, spacing, focus states, dialogs, task rows, navigation, sync status, Quick Add, search, conflicts, or settings.
- Appear in one viewport/browser unintentionally.

PR reviewers should ask for a fix when a screenshot change is not explicitly expected and justified.
