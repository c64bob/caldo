# Nextcloud QA Runbook

This runbook verifies Caldo against a real Nextcloud CalDAV account. It is a release/staging QA gate, not a normal PR test.

## Scope

Use this runbook to check that Caldo interoperates with Nextcloud for setup, sync, write-through task operations, subtasks, favorites, recurrence preservation, attachments, conflict handling, and settings.

Do not convert this runbook into a committed Playwright suite. Normal automated browser QA must keep using the local staging CalDAV server.

## Release Smoke Gate

For every release candidate, run the minimal staging smoke below against a dedicated Nextcloud test account before treating the build as production-ready. The run is a release evidence artifact, not a committed fixture.

A release smoke run passes only when all required rows in the result report are `pass` or explicitly `not applicable` with a non-product reason. Any `fail` or product-caused `blocked` result must become a GitHub issue before release decision.

Store one local report per run:

```text
test-results/nextcloud/<yyyy-mm-dd>-staging-smoke.md
```

The report must record:

- Date and timezone of the run
- Commit SHA, release tag, or container image digest
- Server type, for example `Nextcloud`, and server version when available without exposing private infrastructure details
- Browser and OS used for manual verification
- Redacted endpoint shape, not the full CalDAV URL
- Overall pass/fail status

If a finding is product-relevant, create a GitHub issue with sanitized reproduction steps. Assign the `v1.0 production readiness` milestone and add `staging-finding` plus either `production-readiness` or `sync-maturity`. Add `release-blocker` when the finding blocks the current release.

## Safety Rules

- Use a dedicated Nextcloud test account.
- Use disposable calendars and synthetic tasks only.
- Do not use a personal or production task calendar.
- Do not commit artifacts from `test-results/`.
- Do not record credentials, raw VTODO content, session cookies, CSRF tokens, private task text, or full CalDAV URLs in QA notes.
- Record only pass/fail, timestamps, sanitized endpoint shape, client/browser versions, and non-sensitive observations.
- Delete the disposable calendar or all synthetic tasks after the run.

## Prerequisites

- A reachable Nextcloud instance with the Tasks app enabled.
- A dedicated test user and app password.
- A clean local Caldo data directory.
- Local dependencies already used by normal development: Go, Node/npm for optional screenshot review, and the built static assets in `web/static/`.

The local `.env` may contain these QA-only values:

```text
nextcloud_url=<redacted CalDAV calendar collection endpoint>
nextcloud_user=<redacted test user>
nextcloud_password=<redacted app password>
```

`nextcloud_url` must be the CalDAV calendar collection URL that accepts `PROPFIND`, not the normal Nextcloud web root. For a typical Nextcloud installation the shape is:

```text
https://<nextcloud-host>/remote.php/dav/calendars/<test-user-id>/
```

If a QA secret store only provides the Nextcloud web root, derive the CalDAV URL for the run and record that derivation in the sanitized result without writing the full URL.

Caldo itself still requires its normal runtime env:

```bash
BASE_URL=https://caldo.nextcloud-qa.local
PROXY_USER_HEADER=X-Forwarded-User
ENCRYPTION_KEY=<32 bytes base64>
DB_PATH=.tmp/nextcloud-qa/caldo.db
PORT=19080
```

`BASE_URL` must use `https://` even when the local QA browser reaches Caldo over `http://127.0.0.1:<PORT>`.

## Local Startup

1. Build the binary:

```bash
go build -o .tmp/nextcloud-qa/caldo ./cmd/caldo
```

2. Start Caldo with an isolated database and a proxy-auth test header name:

```bash
mkdir -p .tmp/nextcloud-qa test-results/nextcloud
BASE_URL=https://caldo.nextcloud-qa.local \
PROXY_USER_HEADER=X-Forwarded-User \
ENCRYPTION_KEY=<32-byte-base64-key> \
DB_PATH=.tmp/nextcloud-qa/caldo.db \
PORT=19080 \
.tmp/nextcloud-qa/caldo
```

3. Open the browser with the configured proxy-auth header. For manual QA this can be done through a local reverse proxy, a browser extension, or a temporary debugging browser context.

4. Visit:

```text
http://127.0.0.1:19080/
```

## Test Data

Create or reserve one disposable Nextcloud calendar named:

```text
Caldo QA <YYYY-MM-DD>
```

Use synthetic task names such as:

```text
Caldo QA create
Caldo QA edit
Caldo QA parent
Caldo QA remote
```

## Checklist

### Minimal Staging Smoke

This is the required Story 29.1 release smoke. The broader sections below can be run when the release touches those areas or when additional confidence is needed.

| Flow | Steps | Expected |
|---|---|---|
| Setup and import | Start from a clean Caldo DB, submit the dedicated Nextcloud CalDAV credentials, select only the disposable calendar, complete initial import. | Connection succeeds, no sensitive value appears in UI/logs, normal app opens, selected calendar appears as a project. |
| Manual sync | Trigger manual sync from Caldo after setup. | Sync completes visibly and does not remain stuck in running state after refresh. |
| Remote create | Create a synthetic task directly in Nextcloud, then trigger manual sync in Caldo. | New remote task appears locally without duplicate rows. |
| Remote update | Edit the same remote task directly in Nextcloud, then trigger manual sync in Caldo. | Local clean task updates to the remote value. |
| Remote delete | Delete the same remote task directly in Nextcloud, then trigger manual sync in Caldo. | Local clean task is removed or a documented conflict appears if local state was changed. |
| Local dirty vs remote changed | Open a Caldo edit form, change a field without saving, change the same task in Nextcloud, then sync or focus-refresh Caldo. | Dirty local form is not silently overwritten; remote change becomes visible as warning or conflict path. |
| Conflict resolution | Create or use a visible conflict, open conflict detail, resolve using a selected result, then refresh/sync. | Resolution writes to CalDAV, conflict is marked resolved, selected result remains after refresh. |

### 1. Setup And Initial Import

- Open Caldo with the proxy-auth header.
- Enter the Nextcloud CalDAV URL, test user, and app password in the setup wizard.
- Submit the CalDAV step.
- Expected: connection succeeds and no sensitive value appears in the UI or logs.
- Select only the disposable calendar.
- Complete initial import.
- Expected: the normal app opens and the selected calendar appears as a project.

### 2. Task Write-Through

- Create a task in the disposable project.
- Edit title, description, due date, priority, and labels.
- Complete and reopen the task.
- Delete the task.
- Expected: each operation succeeds in Caldo and is visible in Nextcloud after refresh.

### 3. Remote Sync

- Create a task directly in Nextcloud in the disposable calendar.
- Trigger manual sync in Caldo.
- Expected: the remote task appears in Caldo.
- Edit that task in Nextcloud.
- Trigger manual sync in Caldo.
- Expected: the edited values appear in Caldo without duplicate tasks.
- Delete that task in Nextcloud.
- Trigger manual sync in Caldo.
- Expected: the task is removed locally unless a local conflict is expected.

### 4. Subtasks

- In Caldo, create a parent task and one direct subtask.
- Refresh Nextcloud.
- Expected: Nextcloud shows the child as a subtask or preserves the parent relationship.
- In Nextcloud, create another direct subtask under the same parent.
- Trigger manual sync in Caldo.
- Expected: Caldo shows the Nextcloud-created child as an indented direct subtask.

### 5. Favorites And Labels

- Mark a task as favorite in Caldo.
- Refresh Nextcloud.
- Expected: the task has the `STARRED` category or the Nextcloud equivalent representation.
- Remove favorite in Caldo.
- Expected: the normal labels remain and the favorite marker is removed.

### 6. Attachments And Recurrence Preservation

- In Nextcloud, create a task with an external attachment URL if the UI supports it.
- In Caldo, edit a normal field such as the title.
- Expected: the attachment survives the edit.
- In Nextcloud, create or edit a recurring task.
- In Caldo, edit a normal field without changing recurrence.
- Expected: the RRULE survives the edit. If Nextcloud changes completion behavior for recurring tasks, record the observed server behavior.

### 7. Conflict Handling

- Open a task edit form in Caldo and change a field without saving.
- Change the same task in Nextcloud and sync or focus-refresh Caldo.
- Expected: Caldo warns about the remote change without overwriting the dirty local form.
- Save a stale local change if possible.
- Expected: `412 Precondition Failed` or equivalent stale-write handling creates a visible conflict path rather than silently retrying.

### 8. Settings

- Open Settings.
- Run the CalDAV connection test.
- Save the same credentials again.
- Change the default project to the disposable calendar if needed.
- Trigger manual sync.
- Expected: settings save without losing setup state; sync completes.

### 9. Cleanup

- Delete the disposable calendar in Nextcloud, or delete all synthetic QA tasks.
- Remove `.tmp/nextcloud-qa/` if the local DB is no longer needed.
- Keep only the sanitized QA result under `test-results/nextcloud/`.

## Result Template

Store one local report per run:

```text
test-results/nextcloud/<yyyy-mm-dd>-manual.md
```

Template:

```markdown
# Nextcloud Staging Smoke Result

- Date:
- Timezone:
- Commit/tag/image:
- Caldo version/branch:
- Server type/version:
- Nextcloud endpoint shape: redacted
- Nextcloud account: dedicated test account, redacted
- Browser:
- OS:
- Disposable calendar:
- Overall result: pass/fail/blocked

| Area | Result | Notes |
|---|---|---|
| Setup and import | pass/fail/blocked | |
| Manual sync | pass/fail/blocked | |
| Remote create | pass/fail/blocked | |
| Remote update | pass/fail/blocked | |
| Remote delete | pass/fail/blocked | |
| Local dirty vs remote changed | pass/fail/blocked | |
| Conflict resolution | pass/fail/blocked | |
| Task write-through | pass/fail/blocked/not applicable | |
| Subtasks | pass/fail/blocked/not applicable | |
| Favorites and labels | pass/fail/blocked/not applicable | |
| Attachments and recurrence preservation | pass/fail/blocked/not applicable | |
| Settings | pass/fail/blocked/not applicable | |
| Cleanup | pass/fail/blocked | |

## Observations

-

## Follow-Ups

- GitHub issue:
- Milestone:
- Labels:
```
