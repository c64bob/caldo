# AGENTS.md — Caldo

## Source of Truth

Use these documents as the authoritative source for all product and architecture decisions:

- `docs/prd.md` — product requirements, user-facing behavior, UI decisions
- `docs/arch.md` — technical architecture, invariants, data model, startup sequence
- `docs/backlog/` — epics and stories defining implementation scope

Do not introduce behavior that contradicts these documents.
If a story and arch.md conflict, arch.md takes precedence. Report the conflict instead of resolving it silently.

---

## Scope

Implement exactly what the current story requests. Nothing more.

- Do not implement neighboring stories unless explicitly instructed.
- Do not add database columns, routes, or UI elements that belong to a future story.
- Do not refactor code outside the story's scope, even if you notice improvements.
- If completing the story requires a decision not covered by arch.md or the story itself, stop and ask.

---

## Tech Stack

Use only the libraries and tools listed in arch.md Section 2. The following are the confirmed choices:

| Area | Decision |
|---|---|
| Language | Go |
| HTTP router | Chi |
| Templates | Templ |
| Database | SQLite via `modernc.org/sqlite` |
| Logging | `log/slog` |
| CalDAV / WebDAV | `emersion/go-webdav` |
| iCalendar parsing | `emersion/go-ical` + custom VTODO roundtrip layer |
| Migrations | custom embedded migration system |
| Scheduler | goroutine within the Go process |
| Server-driven UI | HTMX |
| Local UI state | Alpine.js |
| CSS | Tailwind CSS |
| Keyboard shortcuts | Vanilla JS |

**Not allowed — do not introduce under any circumstances:**

- Echo, Gin, or any HTTP framework other than Chi
- goose, golang-migrate, or any external migration library
- zap, zerolog, or any logging library other than `log/slog`
- React, Vue, Svelte, or any JS application framework
- Any CDN at runtime — all assets must be served from the local server
- Redis, Memcached, or any external cache
- Cron, Celery, Sidekiq, or any external job runner
- npm, Vite, esbuild, or Webpack — Tailwind CSS is the only build tool
- `localStorage` or `sessionStorage` — use in-memory state only

---

## Hard Rules

These invariants from arch.md must never be violated:

### CalDAV

- CalDAV is the leading data source.
- A local change is only considered saved after a successful CalDAV write.
- Unknown VTODO properties and extensions must be preserved on roundtrip.
- `VALARM`, `ATTACH`, and complex `RRULE` values must survive any patch operation.
- `412 Precondition Failed` responses must trigger conflict handling, not a retry.
- `DELETE` returning `404` must be treated as success.

### Data Integrity

- No silent data loss. Unknown fields, complex RRULEs, and conflict versions must be preserved as long as technically possible.
- All mutating task operations must check `expected_version` before writing.
- `server_version` must never be used as a CalDAV ETag.
- `etag` must never be used as a UI version counter.
- Undo snapshots and the triggering write must be committed in the same DB transaction.

### Privacy and Logging

Never log the following — not in debug, not in error output, not in any structured field:

- Task titles
- Task descriptions
- Raw VTODO content
- CalDAV credentials (URL, username, password, app password)
- Encryption keys or derived key material
- Session IDs
- Proxy auth header values
- CSRF tokens

Log error types and codes. Never log the user-supplied message content associated with an error.

### Database

- SQLite operates in WAL mode (`journal_mode=WAL`).
- There is exactly one write path. All writes go through a single write mutex.
- Remote fetching and VTODO parsing happen outside the write mutex.
- DB mutations are applied in chunks under the write mutex.
- A backup is created before the first pending migration in every startup.
- Applied migrations must never be modified. A checksum mismatch is a hard startup abort.

### Security

- No local login. Authentication is delegated entirely to the reverse proxy.
- The proxy auth header name comes from `PROXY_USER_HEADER`.
- All mutating routes are protected by CSRF (Double-Submit-Cookie with HMAC validation).
- `GET /health` is exempt from auth and CSRF.
- No runtime CDN. All JS and CSS assets are served locally.
- CSP must not include `'unsafe-inline'` or `'unsafe-eval'` for scripts.

### Single Process

- Caldo is designed for exactly one active process per data directory.
- The startup lock (`<dbPath>.startup.lock`) must be acquired before migrations.
- A second process on the same data path must abort on startup.

---

## Startup Sequence

The startup sequence in `cmd/caldo/main.go` is an architecture invariant (arch.md Section 4.3).
Do not reorder steps. The canonical sequence is:

1. Load and validate environment variables
2. Acquire startup lock
3. Open SQLite and set PRAGMAs
4. Run migrations
5. Initialize scheduler (do not start yet)
6. Check setup status — gate normal routes if `setup_complete = false`
7. Load and decrypt CalDAV credentials (if setup complete)
8. Start scheduler (if setup complete)
9. Register signal handlers
10. Start HTTP server

---

## Code Conventions

- All packages live under `internal/`. Only `cmd/caldo/main.go` is outside.
- Package names are short, lowercase, single words: `config`, `db`, `caldav`, `sync`, `handler`, `middleware`, `scheduler`, `crypto`, `migrations`, `parser`, `query`, `model`.
- Exported functions have Go doc comments.
- Error messages are lowercase and do not end with punctuation (Go convention).
- Errors are wrapped with `fmt.Errorf("context: %w", err)` to preserve the chain.
- No `init()` functions outside of test files.
- No global mutable state outside of the explicitly defined write mutex and SSE broker.
- Context is threaded through all CalDAV and DB operations — never use `context.Background()` in a request handler.

---

## Testing

- Unit tests that cover pure logic (lexer, parser, VTODO roundtrip, date parsing, crypto) must have zero dependencies on HTTP or DB.
- Integration tests that touch SQLite use a temporary in-memory database (`file::memory:?cache=shared`).
- Tests must not make real CalDAV network requests. Use interface mocks or test doubles.
- `go test ./... -race` must pass without data race warnings.
- `go vet ./...` must produce no output.

---

## Templ

- All HTML output goes through Templ components. No `html/template` directly.
- Run `templ generate` after modifying any `.templ` file.
- Generated `*_templ.go` files are committed to the repository.
- The CI workflow verifies that generated files are up to date.

---

## Asset Pipeline

- Asset filenames include a content hash: `app.<hash>.css`, `htmx.<hash>.min.js`.
- All asset paths are resolved through `web/static/manifest.json` loaded at startup.
- If `manifest.json` is missing at startup, the process must abort with `os.Exit(1)`.
- Do not hardcode asset hashes in templates or Go code.

---

## SSE

- There is exactly one SSE endpoint for normal operation: `GET /events`.
- The setup wizard uses a separate SSE endpoint: `GET /setup/import/events`.
- Setup SSE must not emit normal task or sync events.
- Normal SSE must not emit setup or import events.
- SSE events are sent only after a DB commit, never speculatively.

---

## Backlog Files

Backlog files in `docs/backlog/` are planning documents, not implementation guides.

- Do not add implementation details, code snippets, or architecture notes to story files.
- Do not change the `Status` field of a story unless explicitly asked.
- Each story contains only:

  - Name
  - Ziel
  - Eingangszustand
  - Ausgangszustand
  - Akzeptanzkriterien

---

## When in Doubt

If a story is ambiguous, arch.md does not cover the case, or implementing the story would require violating an invariant:

**Stop. Ask. Do not invent a solution.**

Prefer a clarifying question over a plausible-but-wrong implementation.

## Codex Environment Constraint

In Codex task environments, Docker CLI is typically unavailable.

- Do **not** attempt to install Docker or run local Docker commands in Codex.
- Validate Docker-related changes by static inspection of `Dockerfile` / Compose files.
- Ensure Docker image builds are enforced in CI workflows instead of local Codex execution.
