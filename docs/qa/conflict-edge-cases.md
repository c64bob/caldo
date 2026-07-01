# Conflict Edge-Case Matrix

This matrix defines release QA scenarios for CalDAV and VTODO conflict behavior. It is a planning and validation artifact; automated tests may cover individual rows, but real-server QA can also select rows from this matrix.

## Safety Rules

- Use dedicated test accounts, disposable calendars, and synthetic tasks only.
- Do not record task titles from private data, raw VTODO content, credentials, session cookies, CSRF tokens, or full CalDAV URLs.
- Use synthetic labels, URLs, and attachment names such as `Caldo QA label` or `https://example.invalid/caldo-qa`.
- Store detailed local run notes under `test-results/`; commit only sanitized matrix updates or issue links.
- Product deviations become GitHub issues with non-private reproduction steps.

## Result Values

Use these values when recording a run:

- `pass`: expected behavior observed.
- `fail`: product behavior violates the expected result.
- `blocked`: environment or setup prevented the scenario from running.
- `not applicable`: the server or UI cannot create the specific fixture, and that limitation is not a Caldo product failure.

Allowed visible error states are acceptable only when they preserve data, keep the conflict unresolved, and do not hide the failure.

## Required Matrix

| ID | Scenario | Setup | Trigger | Expected result | Allowed visible error states |
|---|---|---|---|---|---|
| CE-001 | 412 stale write | Create a task, load its edit form in Caldo, then change the same task remotely so the CalDAV ETag changes. | Save the stale Caldo form. | Caldo does not retry the stale PUT as success; a visible conflict or stale-update path appears, and no local silent success is shown. | Write error banner; unresolved conflict; task remains blocked until refreshed or resolved. |
| CE-002 | Remote delete against local dirty task | Create a task, open a Caldo edit form and change a field without saving, then delete the task remotely. | Trigger manual sync or focus-refresh. | Caldo does not silently discard local dirty input; a delete conflict or clear stale/deleted warning is visible. | Unresolved conflict; stale warning; save blocked until user refreshes or resolves. |
| CE-003 | Remote delete against locally clean task | Create and sync a task with no local dirty state. | Delete the task remotely and trigger manual sync. | Clean local task is removed without creating an unnecessary conflict. | Sync error state if server request fails; task remains until a later successful sync. |
| CE-004 | Same field changed locally and remotely | Start from a synced task, change title locally and change title remotely to a different value. | Trigger sync or stale write. | Conflict stores base, local, and remote values and shows the title as conflicting. | Unresolved conflict; write failure visible; no silent overwrite. |
| CE-005 | Different fields changed locally and remotely | Start from a synced task, change local due date and remote description. | Trigger sync or conflict resolution. | Non-overlapping field changes can be preserved through manual resolution without losing either side. | Conflict remains unresolved if automatic merge is not possible. |
| CE-006 | Repeated conflict resolution after remote changed again | Create a visible conflict, open conflict detail, then change the remote task again before resolving. | Submit a resolution from the older conflict view. | Caldo does not mark the conflict resolved after a stale write; a follow-up conflict or visible write failure remains. | 412 conflict remains unresolved; visible write error; no retry loop. |
| CE-007 | Resolve local version | Create a conflict with distinct local and remote values. | Resolve by choosing local result. | Local result writes to CalDAV, task is no longer blocked, conflict has resolved metadata, and chosen fields remain after refresh/sync. | Conflict remains unresolved if CalDAV write fails. |
| CE-008 | Resolve remote version | Create a conflict with distinct local and remote values. | Resolve by choosing remote result. | Remote result becomes local state, conflict is resolved, and remote VTODO data is not degraded. | Conflict remains unresolved if persistence fails. |
| CE-009 | Split resolution | Create a conflict where both versions should be kept. | Resolve with split. | Original and new task are preserved as separate tasks, parent links are not corrupted, and conflict is marked resolved only after successful persistence. | Conflict remains unresolved; best-effort cleanup documented if a partial remote write fails. |
| CE-010 | Labels / CATEGORIES conflict | Start with one synced label, remove it locally and add a different label remotely. | Trigger sync or stale write. | Conflict UI shows label differences; selected resolution writes correct VTODO `CATEGORIES` without losing unrelated labels. | Unresolved conflict; visible write failure. |
| CE-011 | Subtask relationship conflict | Create parent and direct child, then change or remove the parent relationship remotely while editing the child locally. | Trigger sync or conflict resolution. | Caldo preserves valid direct subtask relationships or exposes the relationship conflict; no deeper nesting is silently created. | Child becomes root with raw relationship preserved where required; unresolved conflict. |
| CE-012 | Recurrence preserved during conflict | Create a recurring task with RRULE, then create a normal field conflict. | Resolve conflict without editing recurrence. | RRULE survives the conflict and resolution unchanged unless recurrence was explicitly selected for change. | Conflict remains unresolved if recurrence cannot be safely written. |
| CE-013 | Complex RRULE preserved | Create a task with a complex RRULE that Caldo treats as read-only. | Edit or resolve a conflict on a normal field. | Complex RRULE remains in the VTODO and is not simplified or dropped. | Unresolved conflict; visible unsupported-recurrence warning. |
| CE-014 | VALARM preserved | Create a task with a VALARM remotely. | Resolve a conflict on title, description, due date, labels, or priority. | VALARM survives VTODO patching and conflict resolution. | Conflict remains unresolved if preservation cannot be guaranteed. |
| CE-015 | ATTACH preserved | Create a task with an external ATTACH URL or server-supported attachment property. | Resolve a conflict on a normal field. | ATTACH survives VTODO patching and conflict resolution; Caldo does not inline or remove it. | Conflict remains unresolved if attachment preservation cannot be guaranteed. |
| CE-016 | Unknown VTODO fields preserved | Create a task with synthetic unknown properties such as `X-CALDO-QA:preserve`. | Resolve a conflict on known fields. | Unknown properties remain present in the stored remote VTODO after resolution. | Conflict remains unresolved if unknown fields cannot be preserved. |
| CE-017 | Undo after remote change | Create an undo-capable local change, then change the task remotely before undo. | Run undo. | Undo does not overwrite the remote change silently; conflict or visible stale path is created. | Undo failure message; unresolved conflict. |
| CE-018 | Remote delete after conflict exists | Create a conflict, then delete the remote resource before resolving. | Open conflict detail and attempt a resolution. | Caldo keeps the conflict visible and does not claim success unless selected resolution can be written safely. | Write failure; unresolved conflict; split path may be offered if technically possible. |

## Recording A Run

Use this local report path:

```text
test-results/conflicts/<yyyy-mm-dd>-edge-cases.md
```

Template:

```markdown
# Conflict Edge-Case Result

- Date:
- Timezone:
- Commit/tag/image:
- Server type/version:
- Endpoint shape: redacted
- Browser:
- OS:
- Overall result: pass/fail/blocked

| ID | Result | Notes | Issue |
|---|---|---|---|
| CE-001 | pass/fail/blocked/not applicable | | |
| CE-002 | pass/fail/blocked/not applicable | | |

## Follow-Ups

- GitHub issue:
- Milestone:
- Labels:
```

## Issue Rules

When a row fails because of product behavior:

1. Create a GitHub issue with sanitized reproduction steps.
2. Do not include raw VTODO, task content, credentials, cookies, CSRF tokens, or full CalDAV URLs.
3. Assign the `v1.0 production readiness` milestone.
4. Add `sync-maturity`.
5. Add `staging-finding` for real-server findings.
6. Add `release-blocker` if the failure risks data loss, silent overwrite, unresolved stale writes, or hidden conflict state.
