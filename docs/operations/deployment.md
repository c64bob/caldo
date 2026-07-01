# Deployment and Operations Runbook

This runbook explains how to install, start, update, and triage Caldo in self-hosted operation. It is operator-facing and follows the architecture invariants in `docs/arch.md`.

## Scope

Caldo runs as exactly one active process per data directory. It listens on one internal HTTP port and expects a reverse proxy to provide TLS and authentication.

Use the related release and QA runbooks for release decisions and evidence:

- Release and rollback checklist: `docs/qa/release-rollback.md`
- Backup, restore, and migration drill: `docs/qa/backup-restore.md`
- Nextcloud staging smoke: `docs/qa/nextcloud.md`
- CalDAV compatibility matrix: `docs/qa/caldav-compatibility.md`

Do not record private task content, CalDAV credentials, encryption keys, session IDs, CSRF tokens, proxy auth header values, or raw VTODO content in operational notes.

## Deployment Modes

| Mode | Use when | Runtime artifact | Data location |
|---|---|---|---|
| Container | You want the reference deployment shape and static assets packaged with the app. | OCI image with `/app/caldo` and `/app/web/static`. | Persistent volume mounted at `/data`. |
| Binary | You manage the process directly with systemd or another supervisor. | `caldo` binary plus matching `web/static/` directory from the same build. | `DB_PATH`, default `/data/caldo.db`. |

Container and binary deployments use the same environment variables and the same startup rules. Do not run both against the same `DB_PATH`.

## Runtime Configuration

Required environment variables:

| Variable | Required | Meaning |
|---|---:|---|
| `BASE_URL` | Yes | External HTTPS URL of the Caldo instance. Must start with `https://`. |
| `ENCRYPTION_KEY` | Yes | Base64 value that decodes to exactly 32 bytes. |
| `PROXY_USER_HEADER` | Yes | Header name set by the reverse proxy to identify the authenticated user. |

Optional environment variables:

| Variable | Default | Meaning |
|---|---|---|
| `LOG_LEVEL` | `info` | Structured log level. |
| `PORT` | `8080` | Internal HTTP listener port. |
| `DB_PATH` | `/data/caldo.db` | SQLite database path. |

`BASE_URL` must be an external `https://` URL even when the reverse proxy talks to Caldo over internal HTTP.

Generate `ENCRYPTION_KEY` in the operator secret store and keep it stable for the lifetime of the database. If the key is lost or changed, existing encrypted CalDAV credentials cannot be decrypted.

## Reverse Proxy Requirements

The reverse proxy is responsible for:

- TLS termination for the public URL.
- Authentication before requests reach Caldo.
- Setting the configured `PROXY_USER_HEADER` to a non-empty authenticated user value.
- Removing or overwriting any incoming client-supplied header with the same name.
- Forwarding normal HTTP requests to Caldo's internal `PORT`.
- Keeping SSE connections open for `/events`; a read timeout around 3600 seconds is recommended.

Caldo has no local login. If the proxy auth header is missing, normal app routes are not usable. `GET /health` remains unauthenticated.

## Secrets

Keep these values outside committed files and release evidence:

- `ENCRYPTION_KEY`
- CalDAV server URL, username, password, or app password
- Reverse-proxy-auth header values
- Session cookies and CSRF tokens

CalDAV credentials are entered through setup or settings and stored encrypted in SQLite. They are not supplied as runtime environment variables.

## Data Directory

For the default container deployment, `/data` is persistent and contains:

| Path | Purpose | Backup relevance |
|---|---|---|
| `/data/caldo.db` | Main SQLite database. | Required. |
| `/data/caldo.db-wal` | SQLite WAL sidecar when present. | Include for file-level backups when present. |
| `/data/caldo.db-shm` | SQLite shared-memory sidecar when present. | Include for file-level backups when present. |
| `/data/caldo.db.startup.lock` | Single-process lock. | Runtime-only; do not restore as data. |
| `/data/caldo.db.backup-*` | Automatic pre-migration backups. | Standalone restore source. |

Run backup and restore drills with `docs/qa/backup-restore.md`.

## Container Operation

Use the reference shape from `docker-compose.yml` unless your platform needs a different service manager.

Minimum container requirements:

- Image contains `/app/caldo` and `/app/web/static`.
- Persistent volume mounted at `/data`.
- Internal listener exposed on `PORT`, default `8080`.
- Healthcheck calls `http://localhost:8080/health` inside the container.
- Restart policy is bounded, for example `on-failure:3`, so migration failures do not restart forever.

Example operational flow:

1. Set required environment variables in the container platform secret/config mechanism.
2. Mount a persistent data volume at `/data`.
3. Start exactly one Caldo container for that data volume.
4. Verify the container healthcheck.
5. Open the public URL through the reverse proxy.
6. Complete setup or verify normal routes.

For updates:

1. Read `docs/qa/release-rollback.md`.
2. If the release has migrations or persistence risk, run `docs/qa/backup-restore.md` on a copy or staging DB first.
3. Pull or pin the target image tag or digest.
4. Stop the current container.
5. Start the new container against the same persistent volume.
6. Verify `/health` and normal routes.
7. Keep the previous image tag or digest available for rollback.

## Binary Operation

Binary operation requires both the compiled binary and matching static assets:

```text
caldo
web/static/manifest.json
web/static/app.<hash>.css
web/static/*.js
```

The process loads `web/static/manifest.json` relative to its working directory. Start the binary from the directory that contains the matching `web/static/` tree, or deploy the binary and static directory together.

Local build flow:

```bash
make build
BASE_URL=https://todos.example.com \
PROXY_USER_HEADER=X-Forwarded-User \
ENCRYPTION_KEY="$CALDO_ENCRYPTION_KEY" \
DB_PATH=/data/caldo.db \
PORT=8080 \
./bin/caldo
```

For updates:

1. Read `docs/qa/release-rollback.md`.
2. If the release has migrations or persistence risk, run `docs/qa/backup-restore.md` on a copy or staging DB first.
3. Stop the running process.
4. Replace the binary and the matching `web/static/` directory from the same build.
5. Keep the previous binary and static directory available for rollback.
6. Start the process with the same environment and `DB_PATH`.
7. Verify `/health` and normal routes.

## Healthcheck

Endpoint:

```text
GET /health
```

Expected response from a started process:

```json
{ "status": "ok" }
```

`/health` only proves the HTTP process is up. It does not prove CalDAV connectivity, sync success, setup completion, or DB migration health after startup. If startup fails before the HTTP server starts, `/health` is unavailable.

## Startup Effects

The canonical startup order is defined in `docs/arch.md`. Operator-relevant startup effects are:

1. Validates environment variables.
2. Acquires `<DB_PATH>.startup.lock`.
3. Opens SQLite and configures WAL mode.
4. Runs migrations, including a backup before the first pending migration.
5. Initializes but gates the scheduler until setup is complete.
6. Loads setup state.
7. Loads encrypted CalDAV credentials only after setup is complete.
8. Starts the scheduler only after setup is complete.
9. Loads the static asset manifest before serving UI routes.
10. Starts HTTP.

Setup-incomplete instances should expose setup routes and `/health`, while normal routes are gated to setup.

## Common Startup and Configuration Errors

Use sanitized error type, field, code, errno, and path values from logs. Do not paste secrets or task content into tickets.

| Symptom | Likely cause | Safe triage |
|---|---|---|
| `startup_failed` with `field=BASE_URL`, `code=missing` | `BASE_URL` not set. | Set the external URL. Do not paste private hostnames into public issues. |
| `startup_failed` with `field=BASE_URL`, `code=must_use_https` | URL does not start with `https://`. | Use the public HTTPS URL even if internal proxy traffic is HTTP. |
| `startup_failed` with `field=ENCRYPTION_KEY`, `code=missing` | Key not set. | Restore the stable key from the operator secret store. |
| `startup_failed` with `field=ENCRYPTION_KEY`, `code=invalid_base64` | Key is not valid Base64. | Regenerate or correct the secret without logging it. |
| `startup_failed` with `field=ENCRYPTION_KEY`, `code=invalid_length` | Decoded key is not 32 bytes. | Use a Base64-encoded 32-byte key. |
| `startup_failed` with `field=PROXY_USER_HEADER`, `code=missing` | Proxy header name not configured. | Set the header name, not the user value. |
| Startup failure with `root_cause_path` under `web/static/manifest.json` | Static assets missing or unreadable. | Deploy matching `web/static/` next to the binary or use the container image. |
| Startup failure with `root_cause_path` under `DB_PATH` | Data directory missing or permission denied. | Fix ownership and mount path for the Caldo process user. |
| Startup failure acquiring startup lock | Another process uses the same DB path or lock file cannot be created. | Ensure exactly one active process per data directory and check permissions. |
| Startup aborts during migrations | Migration failure or checksum mismatch. | Do not keep restarting blindly; inspect sanitized logs and use the backup/restore drill. |
| `/health` unavailable | Process did not start or port/proxy path is wrong. | Check startup logs, `PORT`, container healthcheck target, and service binding. |
| Normal routes return setup | `settings.setup_complete` is false. | Complete setup wizard or restore the intended DB. |
| Normal routes fail behind proxy | Auth header missing or stripped. | Check proxy auth configuration without logging header values. |
| CalDAV unavailable after setup | Credentials cannot be decrypted or server test fails. | Use settings connection test and check sanitized CalDAV error class. |

## Release, Staging, and Rollback Links

Before production release:

1. Run the release checklist in `docs/qa/release-rollback.md`.
2. Run the Nextcloud staging smoke in `docs/qa/nextcloud.md`.
3. Update `docs/qa/caldav-compatibility.md` with sanitized real-server evidence.
4. Run `docs/qa/backup-restore.md` when migrations, persistence changes, sync conflict changes, or deployment changes are present.
5. Keep rollback artifacts available: previous binary/static tree or previous image digest, plus a known-good DB backup when migrations are involved.

If a deployment fails after release, follow the rollback section in `docs/qa/release-rollback.md`.
