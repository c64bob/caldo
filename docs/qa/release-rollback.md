# Release and Rollback Checklist

This checklist is the release decision record for Caldo. Use it for every release candidate before publishing a tag or treating a built image/binary as production-ready.

## Scope

The checklist bundles release evidence, known risk, and rollback decisions. It does not replace the detailed QA runbooks:

- Nextcloud staging smoke: `docs/qa/nextcloud.md`
- Backup, restore, and migration drill: `docs/qa/backup-restore.md`
- CalDAV compatibility matrix: `docs/qa/caldav-compatibility.md`
- Conflict edge-case matrix: `docs/qa/conflict-edge-cases.md`
- Security, privacy, and logging audit: `docs/qa/security-privacy-logging-audit.md`
- Dependency, license, and update review: `docs/qa/dependency-license-update-review.md`
- Browser QA: `docs/qa/playwright.md`
- Performance measurement points: `docs/qa/performance.md`

Release evidence may live locally under `test-results/`. Do not commit private run artifacts, credentials, task content, session values, CSRF tokens, encryption keys, raw VTODO content, or private screenshots.

## Required Release Inputs

Record these values before making a release decision:

| Input | Required | Notes |
|---|---:|---|
| Commit SHA | Yes | Exact commit that will be tagged or deployed. |
| Version | Yes | Release tag such as `v0.9.1`. |
| Binary artifacts | Yes | Release workflow builds Linux amd64 and arm64 binaries. |
| Container image | Yes for container users | Record `ghcr.io/<owner>/caldo:<version>` and the image digest from the release workflow. |
| CI result | Yes | Main or release-candidate commit must have passing CI. |
| Release workflow result | Yes when tag is published | Tag workflow must complete successfully before announcing artifacts. |
| Security/privacy/logging audit | Yes | Required release audit must pass or block release. |
| Dependency/license/update review | Yes | Required dependency review must pass or have tracked issues for all accepted deferrals. |
| Staging smoke | Yes | Required Nextcloud smoke must pass or be explicitly non-applicable for a non-product reason. |
| Backup/restore decision | Yes | Required before releases with migrations, persistence changes, sync conflict changes, or deployment changes. |
| Known risks | Yes | Link GitHub issues or state `none known`. |
| Open blockers | Yes | No open `release-blocker` issue may remain for the target release. |
| Release notes | Yes | Include changes, migration notes, rollback notes, and known limitations. |

## Pre-Tag Checklist

| Check | Pass condition | Result |
|---|---|---|
| Target commit selected | Commit SHA is recorded and comes from `main`. | pass/fail/blocked |
| Version selected | Version tag is recorded and not already used. | pass/fail/blocked |
| CI passed | Go vet, race tests, templ generation check, asset manifest check, binary build, browser QA, and Docker image build passed for the commit. | pass/fail/blocked |
| Security workflow reviewed | Latest scheduled or push security workflow has no unresolved release-blocking finding. | pass/fail/blocked |
| Security/privacy/logging audit completed | `docs/qa/security-privacy-logging-audit.md` result is `pass` with no unresolved blocking finding. | pass/fail/blocked |
| Dependency/license/update review completed | `docs/qa/dependency-license-update-review.md` result is `pass` with no unresolved blocking finding. | pass/fail/blocked |
| Staging smoke completed | `docs/qa/nextcloud.md` result is `pass` or only has non-product `not applicable` rows. | pass/fail/blocked |
| Compatibility matrix updated | `docs/qa/caldav-compatibility.md` reflects the latest real-server smoke outcome. | pass/fail/blocked |
| Migration risk classified | Release is marked as `no migration`, `migration on copied/staging DB passed`, or `migration blocked`. | pass/fail/blocked |
| Backup/restore drill completed when required | `docs/qa/backup-restore.md` result supports the release decision. | pass/fail/blocked/not applicable |
| Known risks reviewed | All release-relevant issues are linked and classified. | pass/fail/blocked |
| Open blockers cleared | No unresolved `release-blocker` issue remains for this release. | pass/fail/blocked |
| Release notes drafted | Notes include changes, known limitations, migration notes, and rollback notes. | pass/fail/blocked |

Do not tag the release while any required row is `fail` or product-caused `blocked`.

## Migration Decision

Before release, inspect whether the candidate contains new or changed migration files under `internal/migrations/sql/`.

Use this decision rule:

| Situation | Required decision |
|---|---|
| No new or changed migrations | Record `no migration`; normal rollback can reuse the current DB unless another persistence change says otherwise. |
| New migration exists | Run the backup/restore and migration drill on a copied or staging DB before release. |
| Migration drill passes | Record backup file shape, restored DB path, and candidate build identifier in the local release report. |
| Migration drill fails | Block release and create a `release-blocker` issue. |
| Production already migrated and rollback is needed | Roll back app binary/image first only if the previous version is compatible with the migrated DB; otherwise restore the pre-migration backup and then start the previous app version. |

Downgrade or rollback migrations are not part of the MVP. A database rollback means restoring a known-good backup.

## Publish Checklist

1. Confirm the release decision is `ship`.
2. Create the version tag from the recorded commit.
3. Wait for `.github/workflows/release.yml` to complete.
4. Record the GitHub release URL.
5. Record binary artifact names.
6. Record the container image tag and digest.
7. Confirm the release notes match the shipped artifacts.
8. Announce only after artifacts and image digest are available.

The release workflow is tag-triggered for `v*` tags. It builds Linux amd64 and arm64 binaries, pushes the GHCR image for the tag and `latest`, and creates the GitHub release.

## Binary Rollback

Use this when Caldo is deployed as a local binary.

1. Stop the running Caldo process.
2. Preserve the current data directory and logs for diagnosis.
3. Check whether the failed release ran migrations.
4. If no incompatible migration ran, replace the binary with the previous known-good version and keep the same environment and `DB_PATH`.
5. If an incompatible migration may have run, restore the pre-migration database backup first, then start the previous binary.
6. Start Caldo.
7. Verify `GET /health`.
8. Open a normal route through the reverse proxy auth path.
9. Trigger or observe sync only after the restored app state is coherent.
10. Record the rollback result and create follow-up issues for the failed release.

Do not delete the failed database state until the failure has enough evidence for debugging.

## Container Rollback

Use this when Caldo is deployed from GHCR or another container registry.

1. Stop or pause the running container.
2. Preserve the mounted data volume and logs for diagnosis.
3. Pin the deployment to the previous known-good image tag or digest. Prefer a digest over `latest`.
4. Check whether the failed release ran migrations.
5. If no incompatible migration ran, start the previous image against the same data volume.
6. If an incompatible migration may have run, restore the pre-migration database backup into the data volume first, then start the previous image.
7. Verify `GET /health`.
8. Verify normal routes through the reverse proxy auth path.
9. Disable automatic re-upgrade to the failed image until the release-blocking issue is resolved.
10. Record the rollback result and create follow-up issues for the failed release.

Rollback is complete only when the previous image is pinned, the healthcheck is stable, and the UI reaches the expected setup or normal app state.

## Release Notes

Release notes must be ready before publishing and include:

- Version and commit SHA.
- Binary artifacts and container image digest.
- User-visible changes.
- CalDAV or sync behavior changes.
- Migration notes and whether a backup/restore drill was required.
- Known limitations with GitHub issue links.
- Fixed release blockers.
- Rollback note for binary and container operators.

Do not include private server names, CalDAV URLs, task content, credentials, logs with sensitive values, or screenshots from private accounts.

## Decision Rules

| Decision | Meaning |
|---|---|
| `ship` | All required checks pass, artifacts are known, and no release blocker remains. |
| `hold` | Checks are incomplete or non-blocking risk needs an explicit owner before release. |
| `block` | A required check failed, a migration/restore drill failed, CI failed, security/privacy/logging audit failed, dependency/license/update review failed, staging smoke failed, or a release blocker remains open. |
| `rollback` | A published or deployed release has caused a production-impacting issue and operators should move back to the previous known-good binary/image, restoring DB backup if required. |

## Result Template

Store one local report per release candidate:

```text
test-results/release/<version>-checklist.md
```

Template:

```markdown
# Caldo Release Checklist

- Date:
- Timezone:
- Version:
- Commit SHA:
- GitHub release URL:
- Binary artifacts:
- Container image:
- Image digest:
- CI result:
- Release workflow result:
- Security workflow reviewed:
- Security/privacy/logging audit result:
- Dependency/license/update review result:
- Staging smoke result:
- Compatibility matrix updated:
- Backup/restore drill result:
- Migration decision:
- Known risks:
- Open blockers:
- Release decision: ship/hold/block
- Rollback owner:

| Check | Result | Evidence |
|---|---|---|
| Target commit selected | pass/fail/blocked | |
| Version selected | pass/fail/blocked | |
| CI passed | pass/fail/blocked | |
| Security workflow reviewed | pass/fail/blocked | |
| Security/privacy/logging audit completed | pass/fail/blocked | |
| Dependency/license/update review completed | pass/fail/blocked | |
| Staging smoke completed | pass/fail/blocked | |
| Compatibility matrix updated | pass/fail/blocked | |
| Migration risk classified | pass/fail/blocked | |
| Backup/restore drill completed when required | pass/fail/blocked/not applicable | |
| Known risks reviewed | pass/fail/blocked | |
| Open blockers cleared | pass/fail/blocked | |
| Release notes drafted | pass/fail/blocked | |

Release notes summary:

Known limitations:

Rollback notes:

Issues created or updated:
-
```
