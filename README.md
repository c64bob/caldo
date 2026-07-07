# Caldo

Caldo ist eine selbst gehostete Todo-Web-App für Einzelpersonen mit **CalDAV/VTODO als führender Datenquelle**. Die App ist auf einen einzelnen Go-Prozess ausgelegt und synchronisiert Aufgaben bidirektional mit einem CalDAV-Account, zum Beispiel Nextcloud Tasks.

## Kurzüberblick

- Todoist-nahe Bedienung für Self-Hosting
- Aufgaben, Projekte, Labels, Filter und Fälligkeitsdaten
- CalDAV als führende Datenquelle, keine stillen Datenverluste
- Konflikte sichtbar machen und manuell auflösen
- Betrieb als Go-Binary oder mit Docker Compose

## Projektstatus

Der MVP ist umgesetzt und wird weiter gepflegt. Die maßgeblichen Referenzen sind:

- `docs/prd.md` — Produktanforderungen
- `docs/arch.md` — Architektur und Invarianten
- `docs/backlog/` — historische Backlog- und Planungsdokumente

## Voraussetzungen

- Go 1.26+
- Docker Engine + Docker Compose Plugin für das Referenzdeployment
- Reverse Proxy mit vorgeschalteter Authentifizierung und TLS-Terminierung
- HTTPS-Basis-URL für die Instanz (`BASE_URL`)
- 32-Byte-Schlüssel als Base64 für `ENCRYPTION_KEY`

## Start als Go-Binary

```bash
make build
BASE_URL="https://todos.example.com" ENCRYPTION_KEY="<base64-32-byte-key>" PROXY_USER_HEADER="X-Authentik-Username" DB_PATH="./caldo.db" ./bin/caldo
```

## Start mit Docker Compose

```bash
docker compose up -d --build
```

Die Referenzkonfiguration bindet lokal an `127.0.0.1:8080`, nutzt das Volume `caldo_data` für `/data` und prüft `GET /health`.

## Konfiguration

Pflichtvariablen:

| Variable | Beschreibung |
|---|---|
| `BASE_URL` | Externe HTTPS-Basis-URL der Instanz |
| `ENCRYPTION_KEY` | Base64-Schlüssel, der auf exakt 32 Byte decodiert |
| `PROXY_USER_HEADER` | Headername, über den der Reverse Proxy den Benutzer übergibt |

Optionale Variablen:

| Variable | Default | Beschreibung |
|---|---:|---|
| `LOG_LEVEL` | `info` | Loglevel |
| `PORT` | `8080` | HTTP-Port |
| `DB_PATH` | `/data/caldo.db` | Pfad zur SQLite-Datei |

Beispiel:

```dotenv
BASE_URL=https://todos.example.com
ENCRYPTION_KEY=<base64-encoded-32-byte-key>
PROXY_USER_HEADER=X-Authentik-Username
LOG_LEVEL=info
PORT=8080
DB_PATH=/data/caldo.db
```

## Entwicklung

```bash
make build
go test ./...
go test ./... -race
go vet ./...
```

Browser-QA und lokale Staging-Workflows sind in `docs/qa/` dokumentiert. Für Setup, Browsertests, Release-Checks, Backup/Restore und Betrieb sind die entsprechenden Runbooks dort verlinkt.

## Projektstruktur

- `cmd/caldo/` — Programmstart und Startup-Sequenz
- `internal/` — Anwendungslogik
- `web/` — statische Assets und Manifest
- `docs/` — Produkt-, Architektur- und Operationsdokumentation

## Hinweise

- Nach Änderungen an `.templ`-Dateien `templ generate` ausführen und die generierten `*_templ.go`-Dateien committen.
- Migrationen werden über das eingebettete Migrationssystem verwaltet; bereits angewendete Migrationen dürfen nicht geändert werden.
- Alle maßgeblichen Anforderungen und Invarianten stehen in `docs/prd.md` und `docs/arch.md`.
