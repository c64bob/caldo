# Dependency, License, and Update Review

This runbook makes dependency, license, and update risk review repeatable before releases. It is a release QA gate, not a normal PR test.

## Scope

Review every dependency source used to build, test, package, or release Caldo:

- Go modules in `go.mod` and `go.sum`.
- Templ as both a Go module and generated-code tool.
- Tailwind CSS standalone binary pins and SHA-256 checks.
- npm dev-only dependencies in `package.json` and `package-lock.json`.
- Playwright browsers and `@playwright/test` used for browser QA.
- GitHub Actions and container/base-image scanner findings when they affect release confidence.

This review complements:

- Release and rollback checklist: `docs/qa/release-rollback.md`
- Security, privacy, and logging audit: `docs/qa/security-privacy-logging-audit.md`
- Playwright QA: `docs/qa/playwright.md`

## Privacy Rules

This review uses only repository metadata, package metadata, CI/security workflow results, and synthetic QA artifacts. Do not include private data, CalDAV credentials, task content, raw VTODO, session values, CSRF tokens, encryption keys, or private screenshots in review notes.

Store detailed local evidence under:

```text
test-results/dependencies/<version>-review.md
```

Do not commit local dependency reports unless they are explicitly sanitized and intended as docs.

## Required Inputs

| Input | Required | Notes |
|---|---:|---|
| Commit SHA or release tag | Yes | Exact candidate under review. |
| `go.mod` and `go.sum` diff | Yes | Record whether Go dependencies changed since the previous release. |
| `package.json` and `package-lock.json` diff | Yes | Record whether npm/Playwright dependencies changed. |
| Templ pin review | Yes | Compare `go.mod`, `Makefile`, `Dockerfile`, CI, and release workflow pins. |
| Tailwind pin review | Yes | Compare `TAILWIND_VERSION` and `TAILWIND_SHA256` in Dockerfile, CI, and release workflow. |
| Security workflow result | Yes | Review govulncheck, gosec, and Trivy findings. |
| npm audit result | Yes | Review dev-only npm dependency findings. |
| License review | Yes | Record changed dependency licenses and exceptions. |
| Update decisions | Yes | Link required update or exception issues. |

## Inventory Commands

Run from the repository root. Commands that contact registries may need normal development network access.

```bash
mkdir -p test-results/dependencies
go list -m all > test-results/dependencies/go-modules.txt
go list -m -u all > test-results/dependencies/go-modules-updates.txt
go mod verify
npm ci
npm ls --all --omit=optional > test-results/dependencies/npm-tree.txt
npm outdated --long > test-results/dependencies/npm-outdated.txt || true
npm audit --audit-level=high > test-results/dependencies/npm-audit.txt || true
```

For release-candidate review, also inspect:

```bash
git diff -- go.mod go.sum package.json package-lock.json Dockerfile .github/workflows/ci.yml .github/workflows/release.yml
```

Do not paste raw command output into issues if it contains private paths or local machine details. Summarize package name, version, severity, license, and decision.

## Review Checklist

| Area | Check | Pass condition |
|---|---|---|
| Go modules | `go.mod`/`go.sum` are intentional, tidy, verified, and reviewed for changed direct and indirect dependencies. | No unexplained module drift; `go mod verify` passes. |
| Go security | govulncheck output is reviewed. | No untriaged release-relevant vulnerability remains. |
| Templ | Templ version is consistent across `go.mod`, `Makefile`, `Dockerfile`, CI, and release workflow. | All pins match or mismatch is documented with an issue. |
| Tailwind | Tailwind version and SHA-256 are consistent across `Dockerfile`, CI, and release workflow. | Version/SHA match and checksum verification remains present. |
| npm lockfile | `package-lock.json` matches `package.json` and install is reproducible with `npm ci`. | `npm ci` succeeds; lockfile drift is intentional. |
| Playwright | `@playwright/test`, `playwright`, and `playwright-core` versions are reviewed as dev-only QA dependencies. | Playwright remains dev-only and does not enter runtime assets or container runtime image. |
| npm security | `npm audit --audit-level=high` output is reviewed. | No untriaged high or critical npm finding remains. |
| License review | New or changed dependencies have license metadata reviewed. | Permissive licenses are recorded; unknown, proprietary, copyleft, or network-copyleft licenses have explicit issue links and release decisions. |
| Update review | Outdated direct dependencies and Dependabot PRs are reviewed. | Required updates are merged or tracked; accepted deferrals have issue links. |
| Runtime boundary | Dev-only npm/Playwright tooling does not create production runtime assets beyond local QA artifacts. | Container/runtime image remains Go binary plus `web/static/` assets built by the approved pipeline. |
| Secrets and privacy | Review artifacts contain only dependency metadata. | No secrets or private data are present. |

## License Review Guidance

For each new or changed dependency, record:

- Package/module name.
- Previous version and candidate version.
- Direct or indirect dependency.
- Runtime, build-time, QA-only, or GitHub Actions scope.
- License from package metadata or upstream license file.
- Risk decision: accepted, needs issue, or blocks release.

Treat these as release-relevant risks until reviewed:

- Missing or unknown license.
- Proprietary or source-unavailable dependency.
- Copyleft or network-copyleft license in runtime code or assets.
- Dependency that changes licensing between versions.
- New dependency that ships executable tooling in CI or release workflows.

The project license is `MIT`; dependency review must not silently introduce incompatible obligations.

## Update Decision Rules

| Situation | Decision |
|---|---|
| Security fix available for release-relevant dependency | Update before release or create a `release-blocker` issue with explicit owner and mitigation. |
| High/critical vulnerability in runtime dependency | Block release unless a documented non-applicability decision exists. |
| High/critical vulnerability in dev-only dependency | Triage before release; block if browser QA or release artifacts can be affected. |
| Minor/patch update with low risk | Merge if CI and browser QA pass, or defer with a tracked issue. |
| Major update or tooling update | Review changelog, run full CI and browser QA, and create issue/PR with rollback note. |
| License exception needed | Create an issue with milestone `v1.0 production readiness` before release. |

## Issue Tracking

Create GitHub issues for required updates, exceptions, or unknowns. Assign milestone `v1.0 production readiness` and label `production-readiness`. Add:

- `release-blocker` for current-release blockers.
- `dependencies` for dependency update work.
- `sync-maturity` only when a dependency issue specifically affects CalDAV sync behavior.

Issues must not contain private data or raw local audit output. Include package name, version, scope, severity/license category, impact, decision, and next action.

## Result Template

Store one local report per release candidate:

```text
test-results/dependencies/<version>-review.md
```

Template:

```markdown
# Dependency, License, and Update Review

- Date:
- Timezone:
- Commit/tag/image:
- Previous release:
- Go dependency changes:
- npm/Playwright dependency changes:
- Templ pin:
- Tailwind version/SHA:
- Security workflow result:
- npm audit result:
- Overall result: pass/fail/blocked

| Area | Result | Evidence |
|---|---|---|
| Go modules | pass/fail/blocked | |
| Go security | pass/fail/blocked | |
| Templ | pass/fail/blocked | |
| Tailwind | pass/fail/blocked | |
| npm lockfile | pass/fail/blocked | |
| Playwright | pass/fail/blocked | |
| npm security | pass/fail/blocked | |
| License review | pass/fail/blocked | |
| Update review | pass/fail/blocked | |
| Runtime boundary | pass/fail/blocked | |
| Secrets and privacy | pass/fail/blocked | |

Required updates:
-

Accepted deferrals:
-

License exceptions:
-

Issues created or updated:
-
```
