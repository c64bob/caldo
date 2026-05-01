# Caldo

Caldo ist eine selbst gehostete, Todoist-nahe Todo-Web-App für Einzelpersonen mit **CalDAV/VTODO als führender Datenquelle**. Die Anwendung ist auf technikaffine Self-Hoster ausgelegt, läuft als einzelner Go-Prozess und synchronisiert Aufgaben bidirektional mit einem CalDAV-Account (z. B. Nextcloud Tasks).  

> Kurz gesagt: **lokale Kontrolle + vertraute Aufgabenverwaltung + robuste CalDAV-Synchronisation ohne SaaS-Abhängigkeit**.

## Zielsetzung

Die Produktziele für den MVP sind:

- Todoist-ähnliche Bedienung im Self-Hosting-Kontext bereitstellen
- Aufgaben, Projekte, Labels, Filter und Fälligkeitsdaten effizient verwalten
- Datenintegrität in den Mittelpunkt stellen (keine stillen Datenverluste)
- Konflikte bei Synchronisationsproblemen erkennbar und manuell lösbar machen
- Betrieb als Go-Binary und Docker-Container ermöglichen

## Motivation

Viele Nutzer möchten die Bedienqualität moderner Todo-Apps, aber mit eigener Infrastruktur und ohne Vendor-Lock-in. Caldo adressiert genau diese Lücke:

- **Datensouveränität:** Betrieb auf eigener Infrastruktur
- **Interoperabilität:** CalDAV/VTODO statt proprietärem Datensilo
- **Pragmatische Architektur:** Single-User, SQLite, klarer Fokus auf Robustheit
- **Vertraute UX:** bekannte Navigations- und Interaktionsmuster

## Projektstatus

**Status:** MVP in aktiver Implementierung.

- Scope und Anforderungen: `docs/prd.md`
- Architektur und Invarianten: `docs/arch.md`
- Geplante Arbeitspakete: `docs/backlog/`

---

## README-Überblick

Die folgenden Abschnitte dokumentieren den aktuellen Stand von Caldo für Betrieb, Entwicklung und Fehlersuche.

### 1) Features

- Aufgabenverwaltung mit Todoist-naher UX (Erstellen, Bearbeiten, Abschließen, Löschen, Priorisierung)
- Projekt- und Label-Organisation inklusive Favoriten
- Standardansichten wie *Heute*, *Demnächst* und *Überfällig*
- CalDAV-Synchronisation als führende Datenquelle (manuell und periodisch)
- Konflikterkennung bei konkurrierenden Änderungen mit gezielter Konfliktbehandlung
- Setup-Wizard für Erstkonfiguration, Kalenderauswahl und Initialimport

### 2) Architektur auf einen Blick

- **Web/UI:** Go-HTTP-Server mit Chi, serverseitigen Templ-Views, HTMX für Interaktionen und Alpine.js für lokalen UI-State
- **Sync:** CalDAV/WebDAV-Anbindung mit `emersion/go-webdav`, VTODO-Verarbeitung über `emersion/go-ical` plus Roundtrip-Schicht
- **Persistenz:** SQLite (`modernc.org/sqlite`) im WAL-Modus
- **Hintergrundarbeit:** In-Prozess-Scheduler (kein externer Job-Runner)
- **Datenfluss:** UI-Aktion → validierter Write-Pfad (mit Versionsprüfung) → CalDAV-Write → lokaler Persistenz-Commit → SSE-Event
- **Invarianten:** CalDAV bleibt führend; unbekannte VTODO-Felder müssen erhalten bleiben; Writes laufen über einen einzelnen synchronisierten Write-Pfad

### 3) Voraussetzungen

- Go `1.24+` (für lokalen Build)
- Docker Engine + Docker Compose Plugin (für Referenzdeployment)
- Reverse Proxy mit vorgeschalteter Authentifizierung und TLS-Terminierung
- Gültige HTTPS-Basis-URL für die Instanz (`BASE_URL`)
- 32-Byte-Schlüssel als Base64 für `ENCRYPTION_KEY`

### 4) Installation

#### A) Als Go-Binary

```bash
make build
BASE_URL="https://todos.example.com" ENCRYPTION_KEY="<base64-32-byte-key>" PROXY_USER_HEADER="X-Authentik-Username" DB_PATH="./caldo.db" ./bin/caldo
```

#### B) Mit Docker Compose (Referenzdeployment)

```bash
docker compose up -d --build
```

Die Referenzkonfiguration in `docker-compose.yml` setzt auf:

- lokalen Port-Bind auf `127.0.0.1:8080:8080`
- persistentes Volume `caldo_data` für `/data`
- Healthcheck auf `GET /health`
- Restart-Policy `on-failure:3` (kein `unless-stopped`)

### 5) Konfiguration (Environment)

Pflichtvariablen:

| Variable | Pflicht | Beschreibung |
|---|---:|---|
| `BASE_URL` | Ja | Externe HTTPS-Basis-URL der Instanz (muss mit `https://` beginnen). |
| `ENCRYPTION_KEY` | Ja | Base64-Schlüssel, der auf exakt 32 Byte decodiert. |
| `PROXY_USER_HEADER` | Ja | Headername, über den der Reverse Proxy den Benutzer übergibt. |

Optionale Variablen:

| Variable | Pflicht | Beschreibung |
|---|---:|---|
| `LOG_LEVEL` | Nein | Loglevel (Default: `info`). |
| `PORT` | Nein | HTTP-Port im Container/Prozess (Default: `8080`). |
| `DB_PATH` | Nein | Pfad zur SQLite-Datei (Default: `/data/caldo.db`). |

> Wichtiger Hinweis: `BASE_URL` muss immer eine `https://`-URL sein – auch dann, wenn der Reverse Proxy intern per HTTP an Caldo weiterleitet.

Beispiel `.env`:

```dotenv
BASE_URL=https://todos.example.com
ENCRYPTION_KEY=<base64-encoded-32-byte-key>
PROXY_USER_HEADER=X-Authentik-Username
LOG_LEVEL=info
PORT=8080
DB_PATH=/data/caldo.db
```

### 6) Nutzung

1. Anwendung starten und Setup-Wizard aufrufen.
2. CalDAV-Zugangsdaten hinterlegen und Verbindung prüfen.
3. Kalender für den Import auswählen und Standardprojekt festlegen.
4. Initialimport ausführen und Ergebnis im UI prüfen.
5. Erste manuelle Synchronisation starten und Statusmeldungen kontrollieren.

### 7) Sicherheit & Datenschutz

- Authentifizierung erfolgt ausschließlich über den vorgeschalteten Reverse Proxy (kein lokales Login).
- Alle mutierenden Routen sind durch CSRF (Double-Submit-Cookie mit HMAC-Validierung) geschützt.
- `GET /health` ist bewusst von Auth/CSRF ausgenommen.
- Sensible Inhalte (z. B. Aufgabeninhalt, Credentials, Tokens, Schlüsselmaterial) dürfen nicht geloggt werden.
- Assets werden lokal ausgeliefert; Laufzeit-CDNs sind nicht vorgesehen.

### 8) Entwicklung

```bash
make build
go test ./...
go test ./... -race
go vet ./...
```

Projektstruktur:

- `cmd/caldo/` – Programmstart und Startup-Sequenz
- `internal/` – Anwendungslogik (z. B. Handler, Sync, DB, Scheduler, Parser)
- `web/` – statische Assets und Manifest
- `docs/` – PRD, Architektur und Backlog

Hinweise:

- Nach Änderungen an `.templ`-Dateien `templ generate` ausführen und generierte `*_templ.go`-Dateien committen.
- Migrationen werden über das eingebettete Migrationssystem verwaltet; bereits angewendete Migrationen dürfen nicht geändert werden.

### 9) Tests & Qualitätssicherung

Verbindliche Basis-Checks:

```bash
go test ./...
go test ./... -race
go vet ./...
```

Teststrategie:

- Unit-Tests für reine Logik (z. B. Parsing, Roundtrip, Kryptografie)
- Integrationsnahe Tests für SQLite-Verhalten über temporäre In-Memory-Datenbank
- Keine echten CalDAV-Netzwerkzugriffe in Tests; stattdessen Mocks/Test Doubles

### 10) Betrieb (Operations)

- **Healthcheck:** `GET /health` für Liveness/Readiness auf HTTP-Ebene
- **Single-Process-Modell:** Eine aktive Instanz pro Datenverzeichnis
- **SQLite-Betrieb:** WAL-Modus, ein synchronisierter Write-Pfad
- **Migrationen:** Backup vor erster ausstehender Migration; Prüfsummenabweichungen führen zum Startup-Abbruch
- **Monitoring-Basis:** HTTP-Status, Fehlercodes und Sync-Status beobachten (ohne sensible Nutzdaten)

### 11) Troubleshooting

- **Start schlägt fehl:** Pflicht-ENVs (`BASE_URL`, `ENCRYPTION_KEY`, `PROXY_USER_HEADER`) prüfen.
- **Setup blockiert:** Reverse-Proxy-Auth-Header und HTTPS-Termination verifizieren.
- **CalDAV-Probleme:** Erreichbarkeit, Credentials und Kalenderberechtigungen kontrollieren.
- **Konflikte bei Änderungen:** Konfliktstatus im UI prüfen und bewusst manuell auflösen statt blind zu überschreiben.
- **Asset-Fehler beim Start:** Vorhandensein von `web/static/manifest.json` sicherstellen.

### 12) Roadmap

- Fertigstellung des MVP-Umfangs gemäß `docs/backlog/`
- Weitere Härtung von Synchronisations- und Konfliktfällen
- Ausbau von Testabdeckung und operativer Dokumentation
- Post-MVP-Themen werden nach MVP-Abschluss priorisiert

### 13) FAQ

- **Ist Multi-User geplant?**
  Nein. Caldo ist für Single-User-Betrieb auf eigener Infrastruktur ausgelegt.
- **Warum kein lokales Login?**
  Sicherheit und Identität werden vollständig an den Reverse Proxy delegiert.
- **Welche CalDAV-Server sind unterstützt?**
  Caldo setzt auf offene CalDAV/WebDAV-Standards; praktische Kompatibilität hängt von Serververhalten und VTODO-Unterstützung ab.

### 14) Lizenz

Die Lizenz ist in der Datei `LICENSE` im Repository geregelt.

### 15) Beitrag leisten (Contributing)

- Vor Änderungen bitte PRD (`docs/prd.md`) und Architektur (`docs/arch.md`) lesen.
- Änderungen eng am Backlog-Umfang halten und Invarianten nicht verletzen.
- Für Pull Requests:
  - kleine, klar abgegrenzte Änderungen einreichen,
  - Tests/Checks dokumentieren,
  - bei Architekturkonflikten nicht raten, sondern offen adressieren.

---

## Dokumente im Repository

- Produktanforderungen: `docs/prd.md`
- Technische Architektur: `docs/arch.md`
- Backlog (Epics/Stories): `docs/backlog/`

## Hinweis

Dieses README beschreibt den aktuellen Projektstand. Maßgeblich für Anforderungen und Architektur bleiben `docs/prd.md` und `docs/arch.md`.
