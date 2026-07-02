# Large Calendar Limits QA

This runbook defines the repeatable release checks for large CalDAV calendars. Use it when a change touches sync, task queries, navigation counts, search, task rows, conflict handling, or release readiness.

## Safety Rules

- Use only synthetic tasks, labels, projects, and conflicts.
- Use Stage-CalDAV for automated local checks and a dedicated disposable staging account for real-server confidence.
- Do not use private calendars, production accounts, raw VTODO content, credentials, session cookies, CSRF tokens, full private CalDAV URLs, or private screenshots in reports.
- Store detailed artifacts under `test-results/performance/<yyyy-mm-dd>/` or `test-results/large-calendars/<yyyy-mm-dd>/`; these files are local QA artifacts and are not committed.
- Product failures become GitHub issues with sanitized steps, expected/actual behavior, commit or image, server type/version, dataset size, measured time, and local artifact path.

## Documented Test Sizes

| Scenario | Tasks | Projects | Labels | Completed | Subtasks | Conflicts | Purpose | Required before |
|---|---:|---:|---:|---:|---:|---:|---|---|
| Baseline UI and sync | 400 | 8 | 24 | 0 | 0 | 0 | PRD realistic-use target and default automated run | sync/UI PRs when performance is relevant |
| Mixed realistic calendar | 400 | 8 | 24 | 80 | 40 | 1 | realistic active/completed mix with one dirty-vs-remote conflict | release candidates |
| Medium calendar | 1,000 | 12 | 40 | 200 | 100 | 2 | early warning for query, search, and full-scan growth | sync/UI query changes |
| Large calendar | 2,500 | 25 | 100 | 500 | 250 | 5 | release confidence for large personal calendars | release candidates |
| Boundary calendar | 10,000 | 25 | 250 | 2,000 | 1,000 | 10 | PRD local storage/UI boundary; sync must stay robust but has no hard duration promise | production-readiness review |

The 400-task baseline is the only size with hard PRD import and incremental-sync duration targets. Larger sizes still record the same measurements and are evaluated as pass/fail against diagnostic targets in the local result file.

## Measured Flows

Record these flows separately. Do not merge initial import, full-scan sync, and UI timings into one number.

| Flow | Start | End | Target |
|---|---|---|---|
| Initial import | click `Weiter zum Import` in setup | normal home route is loaded | 400 tasks <= 30 s; larger sizes record measured time |
| Clean Full-Scan manual sync | click `Jetzt synchronisieren` after setup | `/sync/status` returns `idle` and `Letzter Sync` is not `nie` | 400 tasks <= 10 s; larger sizes record measured time |
| Full-Scan with conflicts | manual sync after synthetic dirty-local-vs-remote changes | conflict list contains the expected rows and sync is idle | diagnostic target <= 10 s unless overridden |
| Navigation | open `/today`, `/upcoming`, `/projects`, `/labels`, and conflict list when conflicts exist | target view is visible | <= 2 s |
| Search | title search, `#Project` search, `@label` search, and live search typing | expected result is visible | <= 2 s |
| Task editing | write-through PATCH of one active task | HTTP write succeeds and edited task is searchable | diagnostic target <= 2 s |

## Automated Stage-CalDAV Check

The opt-in Playwright performance test creates synthetic calendars in the local Stage-CalDAV server, drives setup/import through the browser, measures the flows above, and writes JSON results to `test-results/performance/<yyyy-mm-dd>/playwright-chromium.json`.

Default baseline:

```bash
npm run test:e2e:performance
```

Mixed realistic release check:

```bash
CALDO_E2E_PERF_COMPLETED_TASKS=80 \
CALDO_E2E_PERF_SUBTASKS=40 \
CALDO_E2E_PERF_CONFLICTS=1 \
npm run test:e2e:performance
```

Large local diagnostic run:

```bash
CALDO_E2E_PERF_TASKS=2500 \
CALDO_E2E_PERF_PROJECTS=25 \
CALDO_E2E_PERF_LABELS=100 \
CALDO_E2E_PERF_COMPLETED_TASKS=500 \
CALDO_E2E_PERF_SUBTASKS=250 \
CALDO_E2E_PERF_CONFLICTS=5 \
CALDO_E2E_PERF_TIMEOUT_MS=1800000 \
npm run test:e2e:performance
```

Boundary run:

```bash
CALDO_E2E_PERF_TASKS=10000 \
CALDO_E2E_PERF_PROJECTS=25 \
CALDO_E2E_PERF_LABELS=250 \
CALDO_E2E_PERF_COMPLETED_TASKS=2000 \
CALDO_E2E_PERF_SUBTASKS=1000 \
CALDO_E2E_PERF_CONFLICTS=10 \
CALDO_E2E_PERF_TIMEOUT_MS=7200000 \
npm run test:e2e:performance
```

Target overrides are for diagnosis only and must be mentioned in the result:

```bash
CALDO_E2E_PERF_SEARCH_MS=3000 npm run test:e2e:performance
```

## Real-Server Staging Check

For release candidates, repeat at least the mixed realistic and large calendar checks against a disposable Nextcloud or other CalDAV staging account. The automated Stage-CalDAV JSON is still useful as a baseline, but real-server results must also state:

- server type and version
- auth method shape, without credentials
- endpoint shape, redacted
- network location shape, for example local LAN, VPS, or CI runner
- Caldo commit/tag/image
- browser and OS
- dataset row from the table above

If the real server cannot be bulk-seeded safely, use fewer conflicts but keep the same task/project/label/completed/subtask counts whenever possible. Document deviations in the result.

## Result Template

Store one Markdown summary per large-calendar run:

```text
test-results/large-calendars/<yyyy-mm-dd>/<server>-<scenario>.md
```

Template:

```markdown
# Large Calendar Result

- Date:
- Timezone:
- Commit/tag/image:
- Server type/version:
- Endpoint shape: redacted
- Caldo build:
- Browser/version:
- OS/CPU/RAM:
- Dataset: baseline | mixed | medium | large | boundary
- Tasks/projects/labels/completed/subtasks/conflicts:
- Overall result: pass/fail/blocked

| Flow | Target | Measured | Result | Issue |
|---|---:|---:|---|---|
| Initial import | | | pass/fail/n/a | |
| Clean Full-Scan manual sync | | | pass/fail/n/a | |
| Full-Scan with conflicts | | | pass/fail/n/a | |
| Navigation `/today` | | | pass/fail/n/a | |
| Navigation `/upcoming` | | | pass/fail/n/a | |
| Navigation `/projects` | | | pass/fail/n/a | |
| Navigation `/labels` | | | pass/fail/n/a | |
| Search title/project/label | | | pass/fail/n/a | |
| Live search | | | pass/fail/n/a | |
| Task editing | | | pass/fail/n/a | |

## Findings

-

## Cleanup

- Disposable remote calendars removed: yes/no
- Local DB retained for debugging: yes/no
```

## Failure Handling

Create a GitHub issue when a target is exceeded, sync does not return to idle, conflicts are missing, duplicate tasks appear, edits are lost, or the UI becomes unusable at a documented size.

Use labels `performance` and `sync-maturity`. Add `release-blocker` when the failure affects a release candidate. The issue should include sanitized reproduction steps, dataset size, measured times, expected target, server type/version, Caldo commit/tag/image, and the local artifact path. Do not upload private task data or credentials.
