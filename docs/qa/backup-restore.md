# Backup, Restore, and Migration Drill

This runbook is a release/staging QA gate for operator confidence. It verifies that a Caldo database can be backed up, restored to an isolated path, started, and migrated on a copy before a release reaches production.

## Scope

Use this drill before release candidates that include migrations, persistence changes, sync conflict changes, or deployment changes. It is also the baseline disaster-recovery rehearsal for operators.

This is not a normal PR test and it must not use live private production data as the first migration target. Migration validation happens on a copied database or a staging database.

## Safety Rules

- Prefer synthetic or staging data for release drills.
- If a production backup copy is used for a disaster-recovery rehearsal, keep it in an operator-controlled environment and never commit it.
- Do not commit artifacts from `test-results/`.
- Do not record CalDAV credentials, encryption keys, session cookies, CSRF tokens, raw VTODO content, task titles, task descriptions, or full private CalDAV URLs.
- Record only file shapes, pass/fail state, timestamps, build identifiers, and sanitized observations.
- Do not run the first migration trial for a release directly against private production data.

## Relevant Files

| File or value | Required for backup/restore | Notes |
|---|---:|---|
| `<DB_PATH>` | Yes | Main SQLite database. Default is `/data/caldo.db`. |
| `<DB_PATH>-wal` | If present in a file copy | WAL sidecar. Include when copying database files directly. |
| `<DB_PATH>-shm` | If present in a file copy | Shared-memory sidecar. Include when copying database files directly. |
| `<DB_PATH>.backup-*` | Optional restore source | Automatic pre-migration backup created before the first pending migration. It is produced with SQLite `VACUUM INTO` and is a standalone database file. |
| `<DB_PATH>.startup.lock` | No | Runtime lock file. Do not restore it as evidence or backup data. |
| `ENCRYPTION_KEY` | Required runtime secret | Needed to decrypt stored CalDAV credentials after restore. Keep it in the operator secret store, not in drill reports. |
| `BASE_URL`, `PROXY_USER_HEADER`, `PORT`, `DB_PATH` | Required runtime config | Recreate the runtime environment for the restored instance. |

Static assets and the Caldo binary or container image are release artifacts, not database backup artifacts. Restore them by deploying the selected release build.

## Preflight

Record this metadata in the local result report:

- Date and timezone.
- Commit SHA, release tag, or container image digest.
- Source database kind: staging, synthetic, or copied production backup.
- Sanitized source path shape, for example `/data/caldo.db`.
- Candidate build identifier.
- Whether pending migrations are expected.

Confirm that Caldo is the only active process using the source data directory. The standard drill uses a stopped process or a previously created standalone migration backup. Do not copy a live SQLite WAL database without a consistent backup mechanism.

## Backup Drill

1. Stop Caldo for the source data directory, or select an existing standalone `<DB_PATH>.backup-*` migration backup.
2. Create a local drill directory:

```bash
DRILL_DATE=2026-07-01
DRILL_DIR="test-results/backup-restore/$DRILL_DATE"
mkdir -p "$DRILL_DIR"
```

3. For a direct file backup, copy the database and any sidecars that exist:

```bash
DB_PATH=/data/caldo.db
cp "$DB_PATH" "$DRILL_DIR/caldo.db"
cp "$DB_PATH-wal" "$DRILL_DIR/caldo.db-wal" 2>/dev/null || true
cp "$DB_PATH-shm" "$DRILL_DIR/caldo.db-shm" 2>/dev/null || true
```

4. For an automatic migration backup, copy the selected standalone backup file to the drill directory:

```bash
BACKUP_PATH=/data/caldo.db.backup-20260701T090000Z-000000000-deadbeef-00
cp "$BACKUP_PATH" "$DRILL_DIR/caldo.db"
```

5. Record which source was used. Do not record private task data from the database.

## Restore Drill

1. Restore into an isolated path, not the original production path:

```bash
mkdir -p .tmp/restore-drill
DRILL_DIR=test-results/backup-restore/2026-07-01
cp "$DRILL_DIR/caldo.db" .tmp/restore-drill/caldo.db
cp "$DRILL_DIR/caldo.db-wal" .tmp/restore-drill/caldo.db-wal 2>/dev/null || true
cp "$DRILL_DIR/caldo.db-shm" .tmp/restore-drill/caldo.db-shm 2>/dev/null || true
```

2. Start the same released build, or the candidate build when the restore is part of a migration drill, with the restored database path:

```bash
# Export CALDO_RESTORE_DRILL_ENCRYPTION_KEY from the operator secret store before starting.
BASE_URL=https://caldo-restore-drill.local \
PROXY_USER_HEADER=X-Forwarded-User \
ENCRYPTION_KEY="$CALDO_RESTORE_DRILL_ENCRYPTION_KEY" \
DB_PATH=.tmp/restore-drill/caldo.db \
PORT=19081 \
./bin/caldo
```

3. Verify the process starts and `/health` responds:

```bash
curl -fsS http://127.0.0.1:19081/health
```

4. With the configured proxy-auth header, open a normal route and confirm the expected setup state:

- Existing configured instance: normal app routes are available.
- Incomplete setup database: setup wizard is shown.
- Decryption failure after restore: treat as failed restore unless the wrong `ENCRYPTION_KEY` was intentionally used.

## Migration Drill

Run the migration drill against the restored copy or a staging database only.

1. Start the candidate build with `DB_PATH` pointing at the restored copy.
2. If pending migrations exist, confirm startup creates a file matching:

```text
<DB_PATH>.backup-*
```

3. Verify the candidate build starts successfully and `/health` responds.
4. Open a normal route with the proxy-auth header and confirm tasks, projects, settings, and setup state are still coherent at a high level.
5. If startup fails because of a migration error, checksum mismatch, missing backup, or unreadable restored database, the release is blocked.

Never use the original private production database as the first target for a candidate build with pending migrations.

## Release Gate

A release is blocked when any required row is `fail` or product-caused `blocked`:

- Backup files cannot be identified.
- Restore cannot start from the selected backup.
- Restored app cannot answer `/health`.
- Restored setup state is incoherent.
- Pending migration does not create `<DB_PATH>.backup-*`.
- Candidate build fails migration on the restored copy or staging database.
- Migration checksum mismatch occurs.

Create a GitHub issue for product-relevant failures. Assign the `v1.0 production readiness` milestone and the `production-readiness` label. Add `release-blocker` when the finding blocks the current release.

## Result Template

Store one local report per run:

```text
test-results/backup-restore/<yyyy-mm-dd>-drill.md
```

Template:

```markdown
# Backup, Restore, and Migration Drill Result

- Date:
- Timezone:
- Commit/tag/image:
- Candidate build:
- Source database kind: synthetic/staging/copied-production-backup
- Source DB path shape: redacted
- Restored DB path:
- Pending migrations expected: yes/no/unknown
- Overall result: pass/fail/blocked

| Check | Result | Notes |
|---|---|---|
| Relevant files identified | pass/fail/blocked | |
| Backup copied or selected | pass/fail/blocked | |
| Restore path isolated | pass/fail/blocked | |
| Restored process started | pass/fail/blocked | |
| `/health` responded | pass/fail/blocked | |
| Normal or setup route matched expected state | pass/fail/blocked | |
| Migration backup created when pending | pass/fail/blocked/not applicable | |
| Candidate migrations succeeded on copy/staging | pass/fail/blocked/not applicable | |
| Release decision | pass/fail/blocked | |

Issues created:
-
```
