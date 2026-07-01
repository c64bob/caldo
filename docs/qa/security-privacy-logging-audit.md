# Security, Privacy, and Logging Audit

This runbook verifies release candidates against Caldo's security, privacy, and logging invariants. It is a release QA gate, not a normal PR test.

## Scope

Run this audit before a production release and whenever a release changes authentication, CSRF, CSP, logging, CalDAV credentials, VTODO handling, setup/settings, static assets, or deployment configuration.

This audit complements:

- Release and rollback checklist: `docs/qa/release-rollback.md`
- Deployment and operations runbook: `docs/operations/deployment.md`
- Nextcloud staging smoke: `docs/qa/nextcloud.md`
- Browser QA: `docs/qa/playwright.md`

## Forbidden Audit Content

Audit notes, GitHub issues, release notes, CI artifacts, screenshots, and copied logs must not contain:

- CalDAV server URLs, usernames, passwords, or app passwords.
- `ENCRYPTION_KEY` or derived key material.
- Task titles or descriptions from private data.
- Raw VTODO or calendar payloads.
- Session IDs, CSRF tokens, proxy-auth header values, cookies, or bearer tokens.
- Private screenshots or browser recordings.

Use synthetic data for release evidence. If private production evidence is required for an incident, keep it outside the repository and redact it before creating issues.

## Required Inputs

Record these inputs in the local audit report:

| Input | Required | Notes |
|---|---:|---|
| Commit SHA or release tag | Yes | Exact build under audit. |
| CI result | Yes | Normal CI must pass or known failures must be classified. |
| Security workflow result | Yes | Review govulncheck, gosec, and Trivy outputs. |
| Browser QA result | Yes | Review Playwright logs/artifacts for privacy leaks. |
| Staging smoke result | Yes | Use synthetic data and sanitized notes. |
| Release diff scope | Yes | Note whether auth, CSRF, CSP, logging, credentials, VTODO, setup/settings, or deployment changed. |
| Audit artifact locations | Yes | Local-only paths such as `test-results/security/`; do not commit artifacts. |

## Safe Evidence Collection

Prefer synthetic sentinel values so leakage can be detected without exposing private data. Example sentinel prefixes:

```text
CALDO_AUDIT_TASK_TITLE
CALDO_AUDIT_TASK_DESCRIPTION
CALDO_AUDIT_CALDAV_PASSWORD
CALDO_AUDIT_PROXY_USER
CALDO_AUDIT_CSRF
CALDO_AUDIT_SESSION
```

If a search finds a sentinel or forbidden marker, record only the artifact path, category, and finding status. Do not paste the matching line into issues or reports when it contains sensitive data.

Useful local searches against release artifacts:

```bash
rg -n "CALDO_AUDIT_" test-results/ playwright-report/ .playwright/ 2>/dev/null || true
rg -n "BEGIN:VTODO|BEGIN:VCALENDAR|SUMMARY:|DESCRIPTION:" test-results/ playwright-report/ .playwright/ 2>/dev/null || true
rg -n "session_id|csrf|X-CSRF-Token|X-Forwarded-User|Authorization|Cookie" test-results/ playwright-report/ .playwright/ 2>/dev/null || true
```

These commands may print matches. Treat any match as sensitive until reviewed locally and do not copy raw output into committed files.

## Audit Checklist

| Area | Check | Pass condition |
|---|---|---|
| Logs and error output | Startup, request, sync, setup, settings, CalDAV, Playwright, and staging logs are reviewed for forbidden content. | Logs contain safe error types, codes, IDs, counts, paths without private query data, and status codes only. |
| Error messages | User-visible and HTTP error paths do not echo credentials, raw VTODO, task content, tokens, or private URLs. | Failures use generic messages or sanitized classes. |
| Reverse-proxy auth | Normal routes require the configured proxy user header; `/health` remains exempt. | Missing header cannot reach normal app routes; proxy strips or overwrites client-supplied auth headers. |
| Setup gate | Setup-incomplete instances expose only setup routes and `/health`. | Normal routes redirect or gate to setup until setup is complete. |
| CSRF | Mutating normal and setup routes require valid CSRF. | Missing or invalid CSRF is rejected; `GET /health` is exempt. |
| CSP | Script policy avoids unsafe script execution and runtime CDN. | CSP does not include script `'unsafe-inline'` or `'unsafe-eval'`; assets load from local static files. |
| Secret handling | Runtime secrets are supplied through the operator secret store and not written to reports. | `ENCRYPTION_KEY`, CalDAV passwords, cookies, tokens, and proxy-auth values do not appear in logs or audit artifacts. |
| Credential storage | CalDAV credentials remain protected at rest. | Password/app password is encrypted in SQLite; settings UI does not render stored password values. |
| CalDAV privacy | CalDAV calls and sync errors do not log credentials, URLs with private paths, raw VTODO, titles, or descriptions. | Logs use sanitized error classes, HTTP status codes, counts, IDs, and durations. |
| VTODO preservation privacy | Tests and QA can verify unknown VTODO preservation without committing raw private VTODO. | Any raw VTODO used for QA is synthetic and local-only. |
| Browser artifacts | Screenshots, videos, traces, console logs, and Playwright reports are reviewed. | Artifacts do not contain private screenshots, credentials, task content, tokens, or raw payloads. |
| CI security workflow | govulncheck, gosec, and Trivy results are reviewed. | No unresolved HIGH/CRITICAL or release-relevant finding remains untriaged. |
| Release notes | Notes contain only sanitized limitations and issue links. | No private server names, credentials, task content, tokens, or raw logs. |

## Blocking Findings

Create a GitHub issue when a finding is product-relevant. Assign milestone `v1.0 production readiness` and label `production-readiness`. Add `release-blocker` when the issue blocks the current release.

Findings that block release include:

- Any forbidden content in logs, CI artifacts, browser artifacts, release notes, or issue text.
- Auth bypass for normal routes.
- Missing CSRF protection on mutating routes.
- CSP allowing script `'unsafe-inline'` or `'unsafe-eval'`.
- Runtime CDN usage.
- Plaintext CalDAV password/app password in SQLite or rendered UI.
- `ENCRYPTION_KEY`, session IDs, CSRF tokens, cookies, bearer tokens, or proxy-auth values in logs or artifacts.
- Raw private VTODO, task titles, or task descriptions in logs or artifacts.
- Untriaged security workflow finding that affects the release.

## Result Template

Store one local report per release candidate:

```text
test-results/security/<version>-audit.md
```

Template:

```markdown
# Security, Privacy, and Logging Audit Result

- Date:
- Timezone:
- Commit/tag/image:
- CI result:
- Security workflow result:
- Browser QA result:
- Staging smoke result:
- Release diff scope:
- Artifact paths reviewed:
- Overall result: pass/fail/blocked

| Area | Result | Evidence |
|---|---|---|
| Logs and error output | pass/fail/blocked | |
| Error messages | pass/fail/blocked | |
| Reverse-proxy auth | pass/fail/blocked | |
| Setup gate | pass/fail/blocked | |
| CSRF | pass/fail/blocked | |
| CSP | pass/fail/blocked | |
| Secret handling | pass/fail/blocked | |
| Credential storage | pass/fail/blocked | |
| CalDAV privacy | pass/fail/blocked | |
| VTODO preservation privacy | pass/fail/blocked | |
| Browser artifacts | pass/fail/blocked | |
| CI security workflow | pass/fail/blocked | |
| Release notes | pass/fail/blocked | |

Blocking issues:
-

Non-blocking issues:
-
```
