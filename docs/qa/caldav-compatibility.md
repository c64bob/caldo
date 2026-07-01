# CalDAV Compatibility Matrix

This matrix records sanitized real-server compatibility evidence for Caldo release decisions. It complements fake-CalDAV automation and the Nextcloud staging smoke runbook.

## Safety Rules

- Use dedicated test accounts and disposable calendars only.
- Use synthetic tasks only.
- Do not record credentials, raw VTODO content, session cookies, CSRF tokens, private task text, or full CalDAV URLs.
- Record endpoint shape only, for example `https://<host>/remote.php/dav/calendars/<user>/`.
- Link product findings to GitHub issues instead of storing private logs or screenshots in this document.
- Keep detailed local run artifacts under `test-results/`; commit only sanitized summary rows here.

## Required Core Flows

Each supported-server row must summarize these flows:

- setup/import
- manual sync
- remote create
- remote update
- remote delete
- local dirty vs remote changed
- conflict resolution

Optional rows may also include task write-through, subtasks, favorites and labels, attachments, recurrence preservation, and settings when those areas changed in the release.

## Capability Fields

Record capabilities as `yes`, `no`, `partial`, `unknown`, or `not tested`.

| Field | Meaning |
|---|---|
| `calendar-home-set` | Server exposes a discoverable calendar home. |
| `calendar-query-vtodo` | Server supports VTODO listing through CalDAV calendar-query. |
| `mkcalendar` | Server supports calendar creation for project creation tests. |
| `put-if-match` | Server honors ETag preconditions for task updates. |
| `delete-404-success` | Server returns or tolerates 404 for already-deleted task cleanup. |
| `getctag` | Server exposes CalendarServer CTag or equivalent collection change marker. |
| `webdav-sync` | Server supports WebDAV Sync collection reports. |
| `fullscan` | Server can be synchronized through the Full-Scan fallback. |

## Current Matrix

Add one sanitized row per tested server version and auth method. If the same server version is tested with multiple auth methods, use separate rows.

| Server | Version | Auth method | Endpoint shape | Capabilities | Core flows | Last run | Build | Result | Known limitations / issues |
|---|---|---|---|---|---|---|---|---|---|
| Nextcloud | pending first recorded run | app password | `https://<host>/remote.php/dav/calendars/<user>/` | pending | pending | pending | pending | pending | none recorded |

## Row Template

```markdown
| Server | Version | Auth method | Endpoint shape | Capabilities | Core flows | Last run | Build | Result | Known limitations / issues |
|---|---|---|---|---|---|---|---|---|---|
| Nextcloud | 00.0.0 | app password | `https://<host>/remote.php/dav/calendars/<user>/` | calendar-home-set=yes; calendar-query-vtodo=yes; mkcalendar=yes; put-if-match=yes; delete-404-success=yes; getctag=yes; webdav-sync=unknown; fullscan=yes | setup/import=pass; manual sync=pass; remote create=pass; remote update=pass; remote delete=pass; dirty-vs-remote=pass; conflict resolution=pass | 2026-07-01 | commit/tag/image | pass | #123 |
```

## Adding A Server Or Version

1. Run the relevant real-server QA process with synthetic data.
2. Copy only sanitized summary data into the matrix.
3. Use a separate row for each server version and auth method.
4. Put every limitation or failure in `Known limitations / issues` with a GitHub issue link.
5. Add `staging-finding`, `production-readiness`, `sync-maturity`, or `release-blocker` labels to issues when appropriate.

No code change is required to add a server, version, auth method, or limitation. Update this document in the same PR as the release evidence update, or in a dedicated docs PR.
