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


### Sync (Phase 5)

Caldo speichert Sync-Metadaten pro Collection und unterstützt zwei Modi:

- `webdav-sync` (bevorzugt)
- `etag-fallback` (Fallback, wenn kein Sync-Token nutzbar ist)

Manuelle Synchronisierung ist über `POST /api/sync/now` möglich (authentifiziert via Proxy-Header). Der Endpoint nutzt ausschließlich die authentifizierte Principal-Identität (kein Principal-Override per Request).
Optional kann ein Hintergrundjob aktiviert werden:

```yaml
sync:
  enabled: false
  interval_seconds: 300
  default_principal: "alice@example.com"
```

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

Für Container/ungewöhnliche Arbeitsverzeichnisse kann das Template-Verzeichnis optional per Environment `CALDO_TEMPLATE_DIR` gesetzt werden.

## CI / Build- und Release-Automatisierung

GitHub Actions übernimmt Build, Packaging und Release

## Roadmap

Die UI-Roadmap orientiert sich am Umsetzungsplan in `docs/ui-plan-v1.md` und ist hier mit dem aktuellen Stand konsolidiert.

### v1.0 (aktuell) — nach UI, Backend, Funktionalität

#### UI (gemäß `docs/ui-plan-v1.md`)
- [x] **Phase 1:** Toodledo-ähnliches Grundlayout mit Header, Sidebar (220px) und flexiblem Hauptbereich ist umgesetzt.
- [x] **Phase 1:** Sidebar-Sektionen in Toodledo-Reihenfolge (Hotlist/Main/Folders/Contexts/Goals/Priority/Due Date/Tags) sind navigierbar und der aktive View wird hervorgehoben.
- [x] **Phase 1:** Sidebar-Collapse per Button sowie responsive Ausblendung auf schmalen Screens sind umgesetzt.
- [x] **Phase 1:** Task-Tabelle nutzt das definierte Default-Spalten-Set inkl. Zebra-Striping, Hover-Highlight und Priority-Farbindikator.
- [ ] **Phase 2:** Vollständiges Inline-Editing der Task-Felder (Summary, Priority, Due, Status, Tags, Percent, Star/Done) inkl. Expand-Panel.
- [ ] **Phase 2:** Smart-Add/Quick-Add mit Toodledo-Syntax und Natural-Language-Datum.
- [ ] **Phase 3:** Keyboard-Shortcuts inkl. `?`-Overlay.
- [ ] **Phase 5:** Filter-Panel, Saved Filters und Hotlist-Ansicht.

#### Backend
- [x] CalDAV CRUD für Tasks (Create/Update/Delete mit ETag-/If-Match-Konfliktbehandlung).
- [x] Sync-State pro Collection (WebDAV-Sync bevorzugt, ETag-Fallback als Rückfallebene).
- [x] Manuelle Synchronisation via `POST /api/sync/now`.
- [x] Verschlüsselte Persistenz von DAV-Credentials (AES-256-GCM).
- [x] Docker-Build + `docker-compose` Setup für Self-Hosting.
- [x] Erweiterte Konflikt-/Fehlertexte für 412, Auth, TLS und Server-Unerreichbarkeit.

#### Funktionalität
- [x] Thin-Client-Prinzip umgesetzt: CalDAV bleibt Single Source of Truth; SQLite speichert nur Präferenzen, verschlüsselte Credentials und Sync-Metadaten.
- [ ] Persistente Nutzerpräferenzen für UI-Ansichten/Sortierung/Spalten (über die Basispräferenzen hinaus).
- [ ] Multi-Level-Sortierung, Batch-Edit und Import/Export (vgl. spätere UI-Plan-Phasen).

### v2+ (mögliche Erweiterungen / bewusst out of scope für v1.0)

- [ ] RRULE/Recurrence-Unterstützung.
- [ ] Subtasks über `RELATED-TO` inkl. Reordering/Progress-Anzeige.
- [ ] Komplexe Feld-Merge-Strategien bei Konflikten (statt Reload/Overwrite).
- [ ] Visuelle Politur/UX-Ausbau (Animationen, erweitertes A11y-Finetuning, Dark Mode).

## Contributing

Pull requests welcome. Please open an issue first for significant changes.
All contributions must be licensed under AGPL-3.0.

## License

[GNU Affero General Public License v3.0](LICENSE) — see [LICENSE](LICENSE) for details.

In short: you can use, modify and distribute Caldo freely, but modifications must be
released under the same license — including if you run it as a network service.

## Architekturplanung

Siehe den initialen Analyse- und Umsetzungsplan: [docs/architecture-plan-v1.md](docs/architecture-plan-v1.md).
