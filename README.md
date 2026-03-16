# Caldo — Self-Hosted Task Manager with CalDAV Sync

> A focused, privacy-first task manager inspired by Toodledo. Built with Go and HTMX. Syncs tasks via CalDAV — works with Nextcloud, Radicale, or any compliant CalDAV server.

## Why Caldo?

Most task managers are either too simple, too complex, or rely on proprietary cloud sync.
Caldo gives you a dense, productivity-focused UI in the style of Toodledo — with full CalDAV
sync and zero vendor lock-in. Your tasks stay on your infrastructure.

**Works out of the box with [Tasks.org](https://tasks.org) on Android.**

## Features

- **Toodledo-inspired UI** — compact list view, inline editing, priority colors, folder sidebar
- **CalDAV Sync** — bidirectional sync via VTODO standard (RFC 5545)
- **Thin Architecture** — no proprietary task database; CalDAV server is the single source of truth
- **Reverse Proxy Auth** — no built-in user management; delegates identity to your reverse proxy (`X-Forwarded-User`)
- **Lightweight** — single Go binary, HTMX frontend, SQLite for preferences only
- **Self-Hosted** — Docker-ready, minimal resource footprint
- **Open Source** — AGPL-3.0

## Tech Stack

| Layer | Technology |
|---|---|
| Backend | [Go](https://go.dev/) |
| Frontend | [HTMX](https://htmx.org/) + Go HTML Templates |
| CalDAV Client | [go-webdav](https://github.com/emersion/go-webdav) |
| Persistence | SQLite (preferences, DAV credentials, sync state) |
| Auth | Reverse Proxy (`X-Forwarded-User` header) |
| Deployment | Docker + docker-compose |

## Quick Start (Docker mit bestehender Nextcloud)

1. Docker-Config anpassen:

```bash
cp configs/docker/config.docker.yaml configs/docker/config.local.yaml
# server_url auf deine Nextcloud-CalDAV-URL setzen
```

2. Stack starten:

```bash
docker compose -f deployments/docker-compose.yml up -d --build
```

3. Im Browser öffnen: `http://localhost:8080`

4. Reverse-Proxy-Identität für Tests mitgeben (erforderlich in Phase 1), z. B. Header
   `X-Forwarded-User: alice@example.com`.

Danach unter `/settings` den DAV-Account speichern; der Zugang wird beim Speichern
über `PROPFIND` streng als WebDAV validiert (HTTP 207 erwartet) und verschlüsselt persistiert.
Die eingetragene Server-URL muss dabei auf den konfigurierten CalDAV-Host zeigen.

Die Aufgabenansicht ist unter `/tasks` verfügbar und lädt Listen/Task-Tabelle in einer
dichten HTMX-Struktur (Sidebar + Tabellen-Partial).

## CalDAV Compatibility

Tested against:

- Nextcloud (Caldav Provider, Tasks app)
- Android Sync via [Tasks.org](https://tasks.org) (direct CalDAV)

## Configuration

```yaml
# config.yaml
server:
  port: 8080
  auth_header: "X-Forwarded-User"   # Header set by your reverse proxy

caldav:
  server_url: "https://nextcloud.example.com"
  default_list: "Tasks"

security:
  encryption_key_file: "/run/secrets/caldo_key"  # For encrypted credential storage

database:
  path: "/data/caldo.db"
```

## CI / Build- und Release-Automatisierung

GitHub Actions übernimmt Build, Packaging und Release

## Roadmap

**v1.0** (current)
- [x] CalDAV CRUD for tasks (Create/Update/Delete inkl. ETag-Konflikterkennung)
- [x] Folder / list sidebar (Read-first)
- [ ] Inline editing (title, priority, due date, tags)
- [ ] Docker deployment
- [ ] Keyboard shortcuts
- [ ] Saved filters
- [ ] Recurrence rules (RRULE)
- [ ] Subtasks via `RELATED-TO`
- [ ] Hotlist view
- [ ] Quick-add with natural language parsing

## Contributing

Pull requests welcome. Please open an issue first for significant changes.
All contributions must be licensed under AGPL-3.0.

## License

[GNU Affero General Public License v3.0](LICENSE) — see [LICENSE](LICENSE) for details.

In short: you can use, modify and distribute Caldo freely, but modifications must be
released under the same license — including if you run it as a network service.

## Architekturplanung

Siehe den initialen Analyse- und Umsetzungsplan: [docs/architecture-plan-v1.md](docs/architecture-plan-v1.md).
