# Operational Health and Debugging Guide

This guide helps operators classify Caldo health, startup, sync, scheduler, and configuration issues without exposing private task data or CalDAV secrets.

## Scope

Use this guide after deployment, during release smoke checks, or when investigating a running instance. It complements:

- Deployment and operations runbook: `docs/operations/deployment.md`
- Release and rollback checklist: `docs/qa/release-rollback.md`
- Backup, restore, and migration drill: `docs/qa/backup-restore.md`
- Security, privacy, and logging audit: `docs/qa/security-privacy-logging-audit.md`

Do not copy task titles, task descriptions, raw VTODO, CalDAV credentials, full private CalDAV URLs, `ENCRYPTION_KEY`, session cookies, CSRF tokens, proxy-auth values, or private screenshots into issues or shared notes.

## First Checks

Start every incident with these safe facts:

| Check | How | Meaning |
|---|---|---|
| Process health | `GET /health` on the internal or proxied URL. | HTTP process is running. |
| Startup logs | Search for `startup_failed` and root cause fields. | Startup failed before HTTP if `/health` is unavailable. |
| Normal route through proxy | Open the public URL with reverse-proxy auth. | Proxy, setup gate, session, CSRF bootstrap, and UI assets are broadly working. |
| Sync status | Check the UI sync badge or `GET /sync/status` through an authenticated session. | Manual or scheduled sync state is visible. |
| Recent request logs | Filter by `request_id`, path, status, and duration. | Confirms HTTP route behavior without query strings or task content. |

`/health` is liveness-only. It does not prove setup is complete, CalDAV is reachable, credentials decrypt, migrations are safe after startup, or sync succeeded.

## Safe Log Fields

Logs should be useful without exposing private data. Safe fields include:

- `time`
- `level`
- `msg`
- `request_id`
- `method`
- `path` without query string
- `status`
- `duration_ms`
- `error_type`
- `root_cause_type`
- `root_cause_errno`
- `root_cause_path`
- safe error `code` values such as `missing`, `must_use_https`, `invalid_base64`, `invalid_length`, `not_configured`, `decrypt_failed`, `sync_failed`, or `sync_unavailable`

If a log line contains a forbidden value, treat that as a security/privacy finding and use `docs/qa/security-privacy-logging-audit.md`.

## Startup Classification

| Symptom | Likely class | Local configuration or product bug? | Safe evidence |
|---|---|---|---|
| `/health` unavailable and `startup_failed` names `BASE_URL` | Runtime config invalid. | Local configuration. | `field`, `code`. |
| `/health` unavailable and `startup_failed` names `ENCRYPTION_KEY` | Secret missing, invalid Base64, or wrong length. | Local configuration or secret management. | `field`, `code`; never key value. |
| `/health` unavailable and `startup_failed` names `PROXY_USER_HEADER` | Proxy header name missing. | Local configuration. | `field`, `code`. |
| `/health` unavailable with `root_cause_path` under `web/static/manifest.json` | Static assets missing or unreadable. | Deployment packaging/configuration. | sanitized path and errno. |
| `/health` unavailable with `root_cause_path` under `DB_PATH` | Data path missing, permission denied, DB open failure, or migration issue. | Local storage/configuration unless migration checksum or SQL failure points to product issue. | sanitized path, errno, migration version if present. |
| Startup lock cannot be acquired | Another Caldo process uses the same DB path or lock path is not writable. | Local process/configuration. | sanitized DB path shape and process count. |
| Migration checksum mismatch | Applied migration no longer matches embedded migration. | Product/release artifact issue unless local files were modified. | migration version and checksum mismatch class, not DB content. |
| CalDAV credentials unavailable warning after startup | Stored credentials cannot be loaded or decrypted. | Usually key mismatch or corrupted settings; possible product bug if reproducible with correct key. | `msg=caldav_credentials_unavailable`, `code`. |

Use `docs/qa/backup-restore.md` before experimenting with migrated or production databases.

## Setup and Proxy Classification

| Symptom | Likely class | Local configuration or product bug? | Safe evidence |
|---|---|---|---|
| `/health` works but normal routes return `403` | Reverse proxy auth header is missing before Caldo. | Local proxy configuration. | Header name, not header value; HTTP status. |
| Public URL loops or shows setup unexpectedly | Setup incomplete or wrong/restored DB. | Local DB/configuration unless setup state changed unexpectedly. | `settings.setup_complete` state if inspected locally, route status, no task data. |
| Static assets return errors | Asset files missing, manifest mismatch, or wrong working directory for binary mode. | Deployment packaging. | asset filename shape, status code, path. |
| Mutating routes fail with `403` | CSRF token missing/invalid or proxy/cookie issue. | Usually proxy/session/client configuration; product bug if reproducible in normal UI flow. | route path, method, status, request ID. |

Caldo has no local login. Authentication must be solved at the reverse proxy before requests reach normal app routes.

## Sync Status

The sync badge shows:

- `Status: idle`
- `Status: running`
- `Status: error`
- `Letzter Sync: <local browser-rendered time>` or `nie`

Manual sync uses `POST /sync/manual`. While state is `running`, the UI polls `GET /sync/status` and also receives SSE updates through `GET /events`.

Safe sync status evidence:

- sync state: `idle`, `running`, or `error`
- `sync_last_started_at`
- `sync_last_finished_at`
- `sync_last_success_at`
- `sync_last_error_code`
- affected route path and HTTP status
- whether the issue occurs for manual sync, scheduled sync, or both

Do not include task content, raw VTODO, CalDAV URLs, usernames, passwords, app passwords, or remote calendar names from private accounts.

## Scheduler Behavior

The scheduler:

- Is initialized at startup but only starts after setup is complete.
- Uses the configured sync interval, defaulting to 15 minutes.
- Skips a tick when sync is already `running`.
- Marks scheduler sync failures as `sync_failed`.
- Marks missing runner failures as `sync_unavailable`.
- Cleans expired undo snapshots on sync ticks.
- Cleans old resolved conflicts at most once per 24 hours.
- Stops on process shutdown and waits for in-flight work during graceful shutdown.

Common scheduler cases:

| Symptom | Likely class | Local configuration or product bug? | Safe evidence |
|---|---|---|---|
| No scheduled sync before setup completion | Expected behavior. | Neither. | setup status and scheduler expectation. |
| Manual sync works but scheduled sync appears absent | Interval not elapsed, setup incomplete, process restarted, or scheduler stopped with process. | Usually local observation/configuration. | uptime, sync interval, last start/success timestamps. |
| Sync remains `running` after restart | Persisted state may reflect interrupted run. | Product bug if repeatable with normal shutdown/restart. | timestamps, request IDs, whether process was killed. |
| `sync_unavailable` | Runner was not wired or unavailable. | Product/runtime wiring issue. | error code only. |
| `sync_failed` | CalDAV, parsing, DB, or conflict path failed. | Could be remote server, local config, or product bug. | sanitized error class, route, status, server type/version if staging. |

## Local Configuration vs Product Bug

Classify as likely local configuration when:

- Required env vars are missing or invalid.
- `BASE_URL` is not `https://`.
- Reverse proxy does not set or sanitize `PROXY_USER_HEADER`.
- Data directory permissions, volume mounts, or working directory are wrong.
- The wrong `ENCRYPTION_KEY` is used for an existing DB.
- Static assets are missing next to a binary deployment.
- A second process uses the same `DB_PATH`.
- CalDAV credentials, server URL, or app password are invalid.

Classify as possible product bug when:

- CI and release artifacts are current, but a normal documented flow fails with sanitized reproducible steps.
- A migration checksum mismatch appears from an unmodified release artifact.
- The UI shows local success after a failed CalDAV write.
- Sync stays `running` without an interrupted process and reproduces.
- `412 Precondition Failed` is retried as success or fails to create a visible conflict path.
- Unknown VTODO fields, `VALARM`, `ATTACH`, or complex `RRULE` are lost after a normal edit.
- Forbidden data appears in logs or QA artifacts.

When unsure, create a GitHub issue with sanitized evidence and classify it as investigation.

## Safe GitHub Issue Template

Include:

- Caldo version, commit SHA, release tag, or image digest.
- Deployment mode: container or binary.
- OS/platform and architecture.
- Sanitized `BASE_URL` shape, for example `https://<host>`, not the real host if private.
- Reverse proxy product and whether it sets the configured auth header; do not include header values.
- `DB_PATH` shape and whether it is a persistent volume or local path; do not attach the DB.
- Setup state: incomplete or complete.
- Route path, HTTP method, HTTP status, and `request_id`.
- Sync state and safe sync timestamps.
- Safe log fields: `msg`, `error_type`, `root_cause_type`, `root_cause_errno`, `root_cause_path`, `field`, `code`.
- CalDAV server type/version and sanitized endpoint shape when relevant.
- Whether the issue reproduces with synthetic data or staging CalDAV.
- Expected behavior and actual behavior.
- Links to local runbook rows or sanitized QA result summaries.

Do not include:

- Task titles or descriptions.
- Raw VTODO or calendar payloads.
- CalDAV credentials or full private URLs.
- `ENCRYPTION_KEY`.
- Session cookies, CSRF tokens, bearer tokens, or proxy-auth values.
- Private screenshots, browser traces, or logs that contain private data.
- SQLite database files or backups.

Use labels:

- `production-readiness` for production release or operations impact.
- `staging-finding` for real-server/staging findings.
- `sync-maturity` for CalDAV sync behavior.
- `release-blocker` when the current release must stop.

Assign milestone `v1.0 production readiness` when the issue affects production readiness.

## Debugging Flow

1. Check `/health`.
2. If `/health` is unavailable, inspect startup logs using only safe fields.
3. If `/health` works, verify the public route through reverse proxy auth.
4. If setup appears, confirm whether setup should be complete for this DB.
5. Check static assets when UI shell is broken.
6. Check sync badge state and timestamps.
7. Try manual sync with synthetic/staging data if CalDAV behavior is involved.
8. Compare manual and scheduled sync behavior.
9. Classify local configuration vs possible product bug.
10. Create a sanitized GitHub issue when product behavior is unclear or wrong.
