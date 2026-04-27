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

## Empfohlene README-Abschnitte

Die folgenden Abschnitte sollten im weiteren Projektverlauf ergänzt bzw. konkretisiert werden.

### 1) Features

- [ ] Kernfunktionen (Aufgaben, Projekte, Labels, Filter, Favoriten)
- [ ] Ansichten (Heute, Demnächst, Überfällig)
- [ ] Sync-Funktionen (manuell, periodisch, Konfliktlösung)
- [ ] Setup-Wizard und Erstimport

### 2) Architektur auf einen Blick

- [ ] Komponentenübersicht (Web, Sync, Scheduler, DB)
- [ ] Datenfluss lokal ↔ CalDAV
- [ ] Invarianten (CalDAV führend, Write-Pfad, Konfliktregeln)

### 3) Voraussetzungen

- [ ] Go-Version: `TODO`
- [ ] Optional: Docker / Docker Compose Versionen: `TODO`
- [ ] Reverse-Proxy-Anforderungen (Auth-Header): `TODO`
- [ ] Unterstützte Browser (MVP): `TODO`

### 4) Installation

#### A) Als Go-Binary

```bash
# TODO: konkrete Build- und Startbefehle ergänzen
```

#### B) Mit Docker

```bash
# TODO: konkrete docker compose Befehle ergänzen
```

### 5) Konfiguration (Environment)

| Variable | Pflicht | Beschreibung |
|---|---:|---|
| `BASE_URL` | Ja | Externe HTTPS-Basis-URL der Instanz |
| `ENCRYPTION_KEY` | Ja | Base64-Schlüssel (32 Byte nach Decoding) |
| `PROXY_USER_HEADER` | Ja | Headername für Reverse-Proxy-Auth |
| `LOG_LEVEL` | Nein | Loglevel (Default: `info`) |
| `TODO` | `TODO` | Weitere optionale Parameter ergänzen |

> Beispiel-`.env`: `TODO`

### 6) Nutzung

- [ ] Setup-Wizard durchlaufen
- [ ] CalDAV-Zugangsdaten hinterlegen
- [ ] Kalenderauswahl / Default-Projekt konfigurieren
- [ ] Erste Synchronisation prüfen

### 7) Sicherheit & Datenschutz

- [ ] Sicherheitsmodell (Reverse-Proxy-Auth, CSRF, HTTPS)
- [ ] Logging-Grenzen (keine sensiblen Inhalte in Logs)
- [ ] Geheimnisverwaltung (ENCRYPTION_KEY, Credentials)

### 8) Entwicklung

```bash
# TODO: lokale Entwickler-Shortcuts eintragen (make targets)
```

- [ ] Projektstruktur (`cmd/`, `internal/`, `web/`, `docs/`) erläutern
- [ ] Vorgehen bei Schema-Migrationen dokumentieren
- [ ] Hinweise zu Templ/Tailwind-Generierung ergänzen

### 9) Tests & Qualitätssicherung

```bash
# TODO: verbindliche Checks dokumentieren
# z. B. go test ./... -race
#      go vet ./...
```

- [ ] Teststrategie (Unit vs. Integration) kurz beschreiben
- [ ] CI-Checks und Qualitäts-Gates dokumentieren

### 10) Betrieb (Operations)

- [ ] Healthcheck-Endpunkt und Monitoring-Hinweise
- [ ] Backup/Restore-Konzept für SQLite (`TODO`)
- [ ] Update-Strategie inkl. Migrationen (`TODO`)
- [ ] Logging/Observability (`TODO`)

### 11) Troubleshooting

- [ ] Häufige Startfehler (z. B. ungültige ENVs)
- [ ] CalDAV-Verbindungsprobleme
- [ ] Konfliktfälle und empfohlene Auflösung

### 12) Roadmap

- [ ] MVP-Restumfang aus Backlog zusammenfassen
- [ ] Geplante Post-MVP-Themen als Stichpunkte (`TODO`)

### 13) FAQ

- [ ] „Ist Multi-User geplant?“
- [ ] „Welche CalDAV-Server sind getestet?“ (`TODO`)
- [ ] „Warum kein lokales Login?“

### 14) Lizenz

- [ ] Lizenztyp ergänzen (`TODO`)
- [ ] Copyright-/Attribution-Hinweise (`TODO`)

### 15) Beitrag leisten (Contributing)

- [ ] Contribution-Prozess (`TODO`)
- [ ] Coding-Standards / Commit-Konventionen (`TODO`)
- [ ] Review- und PR-Erwartungen (`TODO`)

---

## Dokumente im Repository

- Produktanforderungen: `docs/prd.md`
- Technische Architektur: `docs/arch.md`
- Backlog (Epics/Stories): `docs/backlog/`

## Hinweis

Dieses README enthält bewusst Platzhalter, damit die noch offenen Implementierungsdetails im Projektverlauf sauber nachgezogen werden können, ohne Architekturentscheidungen vorwegzunehmen.
