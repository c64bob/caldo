# Caldo — Self-Hosted Task Manager with CalDAV Sync

> A focused, privacy-first task manager inspired by Toodledo. Built with Go and HTMX. Syncs tasks via CalDAV — works with Nextcloud, Radicale, or any compliant CalDAV server.

## Why Caldo?

Most task managers are either too simple, too complex, or rely on proprietary cloud sync.
Caldo gives you a dense, productivity-focused UI in the style of Toodledo — with full CalDAV
sync and zero vendor lock-in. Your tasks stay on your infrastructure.

## Features

- **Toodledo-inspired UI** — compact list view, inline editing, priority colors, folder sidebar
- **HTMX Partial Mutations** — task updates render partials instead of full-page redirects
- **CalDAV Sync** — bidirectional sync via VTODO standard (RFC 5545)
- **Thin Architecture** — no proprietary task database; CalDAV server is the single source of truth
- **Reverse Proxy Auth** — no built-in user management; delegates identity to your reverse proxy (`X-Forwarded-User`)
- **Lightweight** — single Go binary, HTMX frontend, SQLite for preferences only
- **Configurable UI Preferences** — default view, sync interval hint, and column visibility in Settings
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

1. Stack per Environment konfigurieren:

Passe in `deployments/docker-compose.yml` die `CALDO_*`-Variablen (mindestens `CALDO_CALDAV_SERVER_URL` und `CALDO_MASTER_KEY`) an deine Umgebung an.

2. Stack starten:

```bash
docker compose -f deployments/docker-compose.yml up -d --build
```

3. Im Browser öffnen: `http://localhost:8080`

4. Reverse-Proxy-Identität für Tests mitgeben, z. B. Header
   `X-Forwarded-User: alice@example.com`.

Danach unter `/settings` den DAV-Account speichern; der Zugang wird beim Speichern
über `PROPFIND` streng als WebDAV validiert (HTTP 207 erwartet) und verschlüsselt persistiert.
Die eingetragene Server-URL muss dabei auf den konfigurierten CalDAV-Host zeigen.

Über `/settings` können außerdem eine neue CalDAV-Liste (Collection) angelegt sowie
UI-Präferenzen (Default-View, sichtbare Spalten, Sync-Intervall-Hinweis) gespeichert werden.

Die Aufgabenansicht ist unter `/tasks` verfügbar und lädt Listen/Task-Tabelle in einer
dichten HTMX-Struktur (Sidebar + Tabellen-Partial).

### Troubleshooting: `Tasks konnten nicht geladen werden` (HTTP 502)

- Caldo mappt CalDAV-/Netzwerk-/TLS-/Auth-Fehler beim Task-Laden auf HTTP `502` mit
  nutzerfreundlichen Meldungen.
- Zusätzlich schreibt der Server jetzt Diagnosezeilen nach Stdout/Stderr, z. B.:
  `task load failed scope=tasks.page principal=alice@example.com list=tasks err=...`
- Im Docker-Setup findest du diese Logs über:

```bash
docker logs caldo-app
docker compose -f deployments/docker-compose.yml logs -f caldo
```

Typische Ursachen:
- Falsche `CALDO_CALDAV_SERVER_URL` (für Nextcloud typischerweise
  `https://<host>/remote.php/dav/calendars/<user>/tasks/`)
- Ungültige DAV-Credentials/App-Passwort
- TLS-Zertifikats-/Truststore-Probleme
- Netzwerk-/DNS-Erreichbarkeit zwischen Caldo und Nextcloud


### Sync

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


### Environment Overrides (inkl. Persistenz nach config.yaml)

Alle Felder aus `config.yaml` können über Environment-Variablen gesetzt werden. Beim Start lädt Caldo zuerst die YAML-Datei und überschreibt danach die gesetzten `CALDO_*`-Variablen. Falls mindestens ein Override vorhanden ist, schreibt Caldo die effektiven Werte zurück in die konfigurierte Datei (`CALDO_CONFIG`).

> Wichtig für Docker: Die Config-Datei muss im Container **schreibbar** sein, wenn Overrides persistent in die Datei zurückgeschrieben werden sollen.

| YAML-Pfad | Environment-Variable |
|---|---|
| `server.port` | `CALDO_SERVER_PORT` |
| `server.auth_header` | `CALDO_SERVER_AUTH_HEADER` |
| `caldav.server_url` | `CALDO_CALDAV_SERVER_URL` |
| `caldav.default_list` | `CALDO_CALDAV_DEFAULT_LIST` |
| `security.encryption_key_file` | `CALDO_SECURITY_ENCRYPTION_KEY_FILE` |
| `database.path` | `CALDO_DATABASE_PATH` |
| `sync.enabled` | `CALDO_SYNC_ENABLED` |
| `sync.interval_seconds` | `CALDO_SYNC_INTERVAL_SECONDS` |
| `sync.default_principal` | `CALDO_SYNC_DEFAULT_PRINCIPAL` |

Beispiel für ein vollständig Environment-gesteuertes Deployment:

```yaml
services:
  caldo:
    environment:
      CALDO_CONFIG: /app/configs/docker/config.docker.yaml
      CALDO_MASTER_KEY: "<32-byte-base64-key>"
      CALDO_SERVER_PORT: "8080"
      CALDO_SERVER_AUTH_HEADER: "X-Forwarded-User"
      CALDO_CALDAV_SERVER_URL: "https://nextcloud.example.com/remote.php/dav/calendars/alice/tasks/"
      CALDO_CALDAV_DEFAULT_LIST: "Tasks"
      CALDO_SECURITY_ENCRYPTION_KEY_FILE: "/run/secrets/caldo_key"
      CALDO_DATABASE_PATH: "/data/caldo.db"
      CALDO_SYNC_ENABLED: "false"
      CALDO_SYNC_INTERVAL_SECONDS: "300"
      CALDO_SYNC_DEFAULT_PRINCIPAL: "alice@example.com"
```


## Roadmap

Die UI-Roadmap orientiert sich am Umsetzungsplan in `docs/ui-plan-v1.md` und ist hier mit dem aktuellen Stand konsolidiert.

### v1.0 (aktuell) — nach UI, Backend, Funktionalität

#### UI (gemäß `docs/ui-plan-v1.md`)
- [x] **Phase 1:** Toodledo-ähnliches Grundlayout mit Header, Sidebar (220px) und flexiblem Hauptbereich ist umgesetzt.
- [x] **Phase 1:** Sidebar-Sektionen in Toodledo-Reihenfolge (Hotlist/Main/Folders/Contexts/Goals/Priority/Due Date/Tags) sind navigierbar und der aktive View wird hervorgehoben.
- [x] **Phase 1:** Sidebar-Collapse per Button sowie responsive Ausblendung auf schmalen Screens sind umgesetzt.
- [x] **Phase 1:** Task-Tabelle nutzt das definierte Default-Spalten-Set inkl. Zebra-Striping, Hover-Highlight und Priority-Farbindikator.
- [x] **Phase 2:** Vollständiges Inline-Editing der Task-Felder (Summary, Priority, Due, Status, Tags, Percent, Star/Done) inkl. Expand-Panel.
- [x] **Phase 2:** Smart-Add/Quick-Add mit Toodledo-Syntax und Natural-Language-Datum.
- [x] **Phase 3:** Keyboard-Shortcuts inkl. `?`-Overlay.
- [x] **Phase 4:** Subtasks, Folders, Contexts, Goals.
- [ ] **Phase 5:** Filter-Panel, Saved Filters und Hotlist-Ansicht.
- [ ] **Phase 6:** Sortierung, Batch-Edit, Import/Export.
- [ ] **Phase 7:** Visuelle Politur & UX-Details.

#### Backend
- [x] CalDAV CRUD für Tasks (Create/Update/Delete mit ETag-/If-Match-Konfliktbehandlung).
- [x] Sync-State pro Collection (WebDAV-Sync bevorzugt, ETag-Fallback als Rückfallebene).
- [x] Manuelle Synchronisation via `POST /api/sync/now`.
- [x] Verschlüsselte Persistenz von DAV-Credentials (AES-256-GCM).
- [x] Docker-Build + `docker-compose` Setup für Self-Hosting.
- [x] Erweiterte Konflikt-/Fehlertexte für 412, Auth, TLS und Server-Unerreichbarkeit.

#### Funktionalität
- [x] Thin-Client-Prinzip umgesetzt: CalDAV bleibt Single Source of Truth; SQLite speichert nur Präferenzen, verschlüsselte Credentials und Sync-Metadaten.
- [x] Persistente Nutzerpräferenzen für Default-View und Spalten-Sichtbarkeit.
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
