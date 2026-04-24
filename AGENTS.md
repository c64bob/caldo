# AGENTS.md – caldo

## Stack
- Backend: Go (stdlib + `github.com/emersion/go-webdav` für CalDAV)
- Frontend: HTMX + templ
- Datenbank: SQLite (lokale Persistenz)
- Sync: CalDAV (VTODO) → Nextcloud

## Repo-Struktur
- `cmd/server/` – Einstiegspunkt
- `internal/todo/` – Domänenlogik
- `internal/caldav/` – CalDAV-Client
- `internal/db/` – SQLite-Zugriff
- `web/templates/` – templ-Komponenten
- `docs/prd.md` – Product Requirements (Source of Truth)
- `docs/architecture.md` – Architekturentscheidungen
- `docs/stories/` – Feature-Stories

## Conventions
- Kein Feature implementieren, das nicht in docs/prd.md steht
- Keine Dependencies außer: go-webdav, mattn/go-sqlite3, a-h/templ
- Alle Fehler explizit behandeln, kein panic()
- Tests für jedes neue Package

## Build & Run
- `go build ./cmd/server`
- `go test ./...`
- `templ generate` vor dem Build

## Done when
- Code kompiliert fehlerfrei
- Relevante Tests laufen durch
- Kein TODO-Kommentar ohne GitHub Issue-Referenz
