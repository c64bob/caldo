# Long-Running Sync QA

This runbook validates that Caldo sync stays stable across repeated local writes, remote writes, remote deletes, manual syncs, and periodic sync windows. Use synthetic data only.

## Safety Rules

- Use the local Stage-CalDAV server or a dedicated staging CalDAV account with disposable calendars.
- Do not use private tasks, personal calendars, production accounts, raw VTODO content, credentials, session cookies, CSRF tokens, or full private CalDAV URLs in reports.
- Store detailed results under `test-results/sync-longrun/<yyyy-mm-dd>/`; these files are local QA artifacts and are not committed.
- Product failures become GitHub issues with sanitized steps, expected/actual behavior, commit or image, server type, duration, and cycle number.

## Automated Stage-CalDAV Check

The opt-in browser check performs repeated manual sync cycles against the local fake CalDAV server:

```bash
npm run test:e2e:sync-longrun
```

Default behavior:

- 3 cycles
- one remote create and update per cycle
- one local create and update per cycle
- remote delete of the previous cycle's remote task after cycle 1
- sync after every mutation phase
- duplicate, lost-change, hanging-sync, and unexpected-conflict checks
- JSON report at `test-results/sync-longrun/<yyyy-mm-dd>/playwright-chromium.json`

Longer local runs can increase cycle count and add a delay between cycles:

```bash
CALDO_E2E_SYNC_LONGRUN_CYCLES=20 \
CALDO_E2E_SYNC_LONGRUN_CYCLE_DELAY_MS=30000 \
npm run test:e2e:sync-longrun
```

The automated check uses Stage-CalDAV and manual sync triggers. It is repeatable and safe for PR verification, but it does not replace release/staging observation on a real server.

## Staging Periodic-Sync Run

For release candidates or sync-related changes, run a longer staging validation against a dedicated test CalDAV account.

Recommended duration:

- Minimum: 60 minutes
- Preferred before sync releases: 2 to 4 hours
- At least 6 periodic scheduler windows, using the configured sync interval

Recommended setup:

- Clean Caldo DB
- Dedicated staging account
- Disposable calendar named `Caldo Longrun <YYYY-MM-DD>`
- 20 to 100 synthetic tasks before start
- Sync interval recorded in the report

Cycle pattern:

1. Create one task in Caldo and confirm it appears on the server.
2. Edit one existing Caldo-created task in Caldo.
3. Create one task directly on the CalDAV server.
4. Edit one server-created task directly on the CalDAV server.
5. Delete one server-created task directly on the CalDAV server.
6. Wait for either manual sync or the next periodic sync window.
7. Refresh Caldo and check the expected local state.
8. Repeat across the full duration.

Mix manual and periodic sync:

- Start with one manual sync to establish baseline health.
- Run at least one manual sync every 3 to 5 cycles.
- Let the scheduler perform most cycles during the observation window.
- Record exact UTC timestamps for each manual sync and each observed periodic sync completion.

## Required Checks

Every run must explicitly check:

- No duplicate tasks for each synthetic UID/title pair.
- Local writes remain visible after subsequent remote syncs.
- Remote creates appear locally.
- Remote updates replace clean local state.
- Remote deletes remove clean local state.
- Sync status returns to `idle` after each manual or observed periodic sync.
- `Letzter Sync` advances after successful sync.
- No unexpected conflicts appear.
- Any expected conflict has a visible conflict row and remains unresolved until manually handled.
- Settings page does not show a stale sync error after successful recovery.

## Result Template

Store one report per run:

```text
test-results/sync-longrun/<yyyy-mm-dd>/<server>-<duration>.md
```

Template:

```markdown
# Long-Running Sync Result

- Date:
- Timezone:
- Commit/tag/image:
- Server type/version:
- Endpoint shape: redacted
- Auth method: redacted test account
- Caldo build:
- Duration:
- Sync interval:
- Cycles completed:
- Manual sync count:
- Periodic sync count:
- Dataset:
- Overall result: pass/fail/blocked

| Cycle | UTC time | Trigger | Local changes | Remote changes | Expected checks | Result | Issue |
|---:|---|---|---|---|---|---|---|
| 1 | | manual | | | no duplicates; no lost changes; idle | pass/fail | |

## Findings

-

## Cleanup

- Disposable tasks/calendar removed: yes/no
- Local DB retained for debugging: yes/no
```

## Failure Handling

Create a GitHub issue for any product-caused failure with:

- sanitized reproduction steps
- server type and version
- commit/tag/image
- cycle number and sync trigger
- expected vs actual result
- whether duplicates, lost changes, hanging sync, or unexpected conflicts occurred
- local artifact path, not uploaded private data

Use labels `sync-maturity` and `staging-finding`. Add `release-blocker` when the issue blocks the release candidate.
