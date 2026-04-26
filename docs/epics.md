# Epic 1 — Betriebsfundament und Startfähigkeit

## Story 1.0 — Projekt-Scaffold und Repository-Grundlage

**Ziel:**
Das Repository hat eine vollständige, buildbare Grundstruktur – Go-Modul, Verzeichnisbaum, Build-Tooling, Deployment-Konfiguration und CI/CD-Workflows – sodass alle späteren Stories auf einem stabilen Fundament aufbauen.

**Eingangszustand:**
Das Repository enthält nur `docs/` und `.git/`.

**Ausgangszustand:**
`go build ./...` und `docker build .` laufen fehlerfrei durch. GitHub Actions führen CI, Security-Scans und Release-Workflows aus.

**Akzeptanzkriterien:**

* Verzeichnisstruktur:

```
cmd/caldo/main.go
internal/
  config/
  db/
  caldav/
  sync/
  handler/
  middleware/
  scheduler/
  crypto/
  migrations/
  parser/
  query/
  model/
web/
  static/
  templates/
docs/
  arch.md
  prd.md
  epics.md
Makefile
Dockerfile
docker-compose.yml
.github/
  workflows/
  dependabot.yml
.gitignore
```

* `go.mod` enthält ausschließlich die in arch.md Abschnitt 2.1 festgelegten Abhängigkeiten.
* `cmd/caldo/main.go` kompiliert ohne Fehler; `func main()` ist vorhanden und leer.
* `go build ./...` läuft fehlerfrei durch.
* `go vet ./...` erzeugt keine Ausgabe.
* `Makefile` enthält die Targets `build`, `dev`, `tailwind`, `templ`, `test`, `lint` und `docker-build`.
* `Dockerfile` ist mehrstufig: Builder-Stage kompiliert das Binary, Runtime-Stage enthält nur Binary und `web/static/`.
* `docker-compose.yml` enthält einen `caldo`-Service mit konfigurierbaren Umgebungsvariablen, einem benannten Volume für die SQLite-Datenbank und einem `healthcheck` gegen `GET /health`.
* `.gitignore` schließt kompilierte Binaries, `*.db`, `*.db-wal`, `*.db-shm`, `web/static/app.*.css`, `web/static/app.*.js` und `web/static/manifest.json` aus. Generierte `*_templ.go`-Dateien werden eingecheckt.

**GitHub Actions – CI-Workflow** (`.github/workflows/ci.yml`, Trigger: Push und Pull Request auf `main`):

* `go vet ./...` muss fehlerfrei laufen.
* `go test ./... -race` muss fehlerfrei laufen.
* `templ generate` darf keine uncommitteten Änderungen erzeugen (Diff-Check).
* Tailwind-Build darf keine uncommitteten Änderungen erzeugen.
* CI schlägt fehl, wenn generierte Dateien nicht aktuell eingecheckt sind.

**GitHub Actions – Security-Workflow** (`.github/workflows/security.yml`, Trigger: Push auf `main`, wöchentlicher Cron):

* `govulncheck ./...` prüft bekannte Go-Schwachstellen in Abhängigkeiten.
* `gosec ./...` prüft sicherheitsrelevante Code-Muster.
* Trivy scannt das fertig gebaute Docker-Image auf bekannte CVEs (HIGH und CRITICAL).
* Scan-Ergebnisse werden als SARIF in den GitHub Security-Tab hochgeladen.
* Der Security-Workflow verhindert keinen Merge; er ist informativ.

**GitHub Actions – Release-Workflow** (`.github/workflows/release.yml`, Trigger: Git-Tag `v*`):

* Das Binary wird für `linux/amd64` und `linux/arm64` gebaut.
* Das Docker-Image wird für beide Architekturen gebaut und in GitHub Container Registry (`ghcr.io`) gepusht.
* Image-Tags: exakter Versions-Tag sowie `latest`.
* Ein GitHub Release wird automatisch mit dem Image-Digest und einer generierten Changelog-Sektion aus Commits seit dem letzten Tag erstellt.

**Dependabot** (`.github/dependabot.yml`):

* Go-Module werden wöchentlich geprüft.
* GitHub Actions werden wöchentlich geprüft.
* Automatische Pull Requests werden als Draft erstellt.

---

## Story 1.1 — App-Konfiguration validieren

**Ziel:**
Caldo startet nur mit gültiger Minimal-Konfiguration.

**Eingangszustand:**
Es gibt keine validierte Laufzeitkonfiguration.

**Ausgangszustand:**
`BASE_URL`, `ENCRYPTION_KEY`, `PROXY_USER_HEADER` und optionale Defaults sind geprüft.

**Akzeptanzkriterien:**

* Fehlendes `BASE_URL` verhindert den Start.
* `BASE_URL` ohne `https://` verhindert den Start.
* Fehlender `PROXY_USER_HEADER` verhindert den Start.
* Fehlender, nicht Base64-kodierter oder nicht exakt 32 Byte langer `ENCRYPTION_KEY` verhindert den Start.
* Optionale Werte wie `LOG_LEVEL`, `PORT` und `DB_PATH` erhalten dokumentierte Defaults.
* Startfehler werden strukturiert und ohne Secrets geloggt.

---

## Story 1.2 — Startup-Lock und Single-Process-Betrieb

**Ziel:**
Es läuft höchstens ein Caldo-Prozess pro Datenverzeichnis.

**Eingangszustand:**
Mehrere Prozesse könnten dieselbe SQLite-Datei verwenden.

**Ausgangszustand:**
Ein Advisory Startup-Lock verhindert parallele Prozesse.

**Akzeptanzkriterien:**

* Vor DB-Migrationen wird ein Startup-Lock erworben.
* Ein zweiter Prozess mit demselben Datenpfad startet nicht.
* Der Lock bleibt bis zum Prozessende gehalten.
* Ein nicht erwerbbarer Lock führt zu hartem Startabbruch.
* Es gibt keinen Cluster- oder Distributed-Lock-Mechanismus.

---

## Story 1.3 — SQLite öffnen und robust konfigurieren

**Ziel:**
SQLite ist als lokale MVP-Datenbank stabil vorbereitet.

**Eingangszustand:**
Es existiert keine initialisierte DB-Verbindung.

**Ausgangszustand:**
SQLite läuft mit WAL, sinnvollen PRAGMAs und einem kontrollierten Schreibpfad.

**Akzeptanzkriterien:**

* SQLite wird am konfigurierten Pfad geöffnet.
* `journal_mode=WAL` ist gesetzt.
* `synchronous=NORMAL` ist gesetzt.
* `busy_timeout=5000` ist gesetzt.
* Die DB nutzt maximal eine offene Verbindung.
* Alle späteren DB-Writes können über einen globalen Write-Mutex laufen.

---

## Story 1.4 — Automatische Migrationen mit Backup

**Ziel:**
Schemaänderungen laufen beim Start sicher und nachvollziehbar.

**Eingangszustand:**
Es gibt kein verlässliches Migrationssystem.

**Ausgangszustand:**
Migrationen werden eingebettet, versioniert, geprüft und automatisch ausgeführt.

**Akzeptanzkriterien:**

* Die Migrationstabelle speichert Version, Name, Zeitpunkt und Checksum.
* Bereits angewendete Migrationen werden auf Checksum-Abweichung geprüft.
* Ausstehende Migrationen werden automatisch beim Start ausgeführt.
* Vor der ersten ausstehenden Migration wird ein SQLite-Backup erstellt.
* Eine Migration läuft jeweils in einer Transaktion.
* Fehlgeschlagene Migrationen verhindern den normalen App-Start.
* Migrationen loggen keine Task-Inhalte, Credentials oder Tokens.

---

## Story 1.5 — Healthcheck bereitstellen

**Ziel:**
Deployments können erkennen, ob der Prozess läuft.

**Eingangszustand:**
Es gibt keinen Liveness-Endpunkt.

**Ausgangszustand:**
`GET /health` ist ohne Auth erreichbar.

**Akzeptanzkriterien:**

* `GET /health` antwortet ohne Reverse-Proxy-Auth.
* Der Healthcheck prüft nur Prozess-Liveness.
* Der Healthcheck prüft nicht CalDAV.
* Der Healthcheck prüft nicht vollständige DB-Integrität.
* Bei fehlgeschlagenem Start ist der Healthcheck nicht verfügbar.

---

## Story 1.6 — Strukturiertes sicheres Logging

**Ziel:**
Betriebslogs sind nützlich, aber frei von sensiblen Inhalten.

**Eingangszustand:**
Es gibt keine zentrale Logging-Policy.

**Ausgangszustand:**
Logs sind strukturiert, korrelierbar und zentral maskiert.

**Akzeptanzkriterien:**

* Production-Logs sind JSON.
* Development-Logs sind lesbarer Text.
* Jeder HTTP-Request erhält eine `request_id`.
* Jeder Sync-Lauf erhält eine `sync_run_id`.
* Task-Titel, Beschreibungen, Raw-VTODO, Credentials, Tokens, Session-IDs und Auth-Header-Werte werden nie geloggt.
* Maskierung erfolgt zentral.
* Fehlertypen werden ohne nutzdatenhaltige Messages geloggt.

---

## Story 1.7 — Graceful Shutdown

**Ziel:**
Der Prozess beendet sich bei SIGTERM und SIGINT kontrolliert, ohne laufende CalDAV-Operationen abrupt abzubrechen.

**Eingangszustand:**
Der HTTP-Server läuft, aber ein SIGTERM beendet den Prozess sofort ohne Rücksicht auf laufende Requests oder Sync-Läufe.

**Ausgangszustand:**
SIGTERM und SIGINT lösen eine geordnete Shutdown-Sequenz aus, nach deren Abschluss der Prozess sauber endet.

**Akzeptanzkriterien:**

* Signal-Handler registriert sich für `SIGTERM` und `SIGINT`.
* Bei Signal-Empfang nimmt der HTTP-Server sofort keine neuen Verbindungen mehr an.
* Laufende HTTP-Requests werden mit einem Timeout von maximal 30 Sekunden abgewartet.
* Der Scheduler wird angewiesen, keine neuen Jobs mehr zu starten.
* Ein laufender Sync-Job darf bis zu 30 Sekunden zur Fertigstellung nutzen.
* Alle CalDAV-Operationen verwenden `context.Context`, sodass sie auf Timeout reagieren.
* Nach Ablauf des Timeouts werden verbleibende Operationen kontextbasiert abgebrochen.
* Der Prozess endet mit Exit-Code 0 nach geordnetem Shutdown.
* Der Prozess endet mit Exit-Code 1, wenn der Shutdown-Timeout überschritten wurde.
* Kein laufender CalDAV-Write wird durch den Shutdown stumm abgebrochen ohne Log-Eintrag.
* Der Shutdown-Ablauf wird strukturiert geloggt (Start, Scheduler gestoppt, HTTP-Server gestoppt, Prozess beendet); keine Task-Inhalte werden dabei geloggt.

---

# Epic 2 — HTTP-Grundgerüst, Sicherheit und Assets

## Story 2.1 — Middleware-Stack etablieren

**Ziel:**
Alle Requests laufen durch eine konsistente Sicherheits- und Fehlerbehandlung.

**Eingangszustand:**
Es gibt keinen kanonischen Request-Pfad.

**Ausgangszustand:**
Middleware-Reihenfolge ist umgesetzt und stabil.

**Akzeptanzkriterien:**

* `request_id` ist die erste Middleware.
* `recovery` ist die zweite Middleware.
* `safe_logging` läuft vor fachlichen Handlern.
* Security-Header werden für alle relevanten Antworten gesetzt.
* Panics führen zu sicherem 500 ohne interne Details.
* `/health` bleibt von Auth und CSRF ausgenommen.

---

## Story 2.2 — Reverse-Proxy-Authentifizierung

**Ziel:**
Caldo nutzt ausschließlich vorgelagerte Authentifizierung.

**Eingangszustand:**
Requests werden nicht authentifiziert.

**Ausgangszustand:**
Alle normalen App-Routen verlangen den konfigurierten Auth-Header.

**Akzeptanzkriterien:**

* Der Headername kommt aus `PROXY_USER_HEADER`.
* Requests ohne gültigen Header erhalten `403 Forbidden`.
* Es gibt keinen lokalen Login.
* Es gibt keinen Login-Redirect.
* Es gibt keine Rollen.
* Der Headerwert wird nicht geloggt.

---

## Story 2.3 — Session, Tab-ID und CSRF-Grundlage

**Ziel:**
Mutierende UI-Aktionen sind sicher und tab-spezifisch nachvollziehbar.

**Eingangszustand:**
Es gibt keine Session- oder CSRF-Struktur.

**Ausgangszustand:**
Session-Cookie, CSRF-Token und Tab-ID-Konzept sind fachlich nutzbar.

**Akzeptanzkriterien:**

* `session_id` wird als `HttpOnly`, `Secure`, `SameSite=Strict` Session-Cookie gesetzt.
* CSRF schützt alle mutierenden Methoden.
* CSRF verwendet Double-Submit-Cookie mit HMAC-Validierung.
* Ungültiger oder fehlender CSRF-Token ergibt `403`.
* HTMX-Requests können `X-CSRF-Token` und `X-Tab-ID` senden.
* Undo-Identität kann über `(session_id, tab_id)` abgebildet werden.

---

## Story 2.4 — Server-rendered UI-Asset-Grundlage

**Ziel:**
Die Weboberfläche kann ohne Runtime-CDN ausgeliefert werden.

**Eingangszustand:**
Es gibt keine definierte Asset-Auslieferung.

**Ausgangszustand:**
Statische Assets werden lokal, versioniert und CSP-kompatibel ausgeliefert.

**Akzeptanzkriterien:**

* `/static/*` liefert lokale Assets aus.
* Es wird kein Runtime-CDN verwendet.
* CSS- und JS-Dateien nutzen dateinamenbasiertes Cache-Busting.
* `manifest.json` wird beim Start geladen.
* Fehlendes Manifest führt zu hartem Startabbruch.
* Statische Assets erhalten langfristige Cache-Header.
* CSP erlaubt keine inline Scripts und kein `'unsafe-inline'`.

---

## Story 2.5 — Templ-Grundgerüst und Base-Layout

**Ziel:**
Alle späteren Handler können serverseitig gerenderte HTML-Responses zurückgeben, die auf einem konsistenten Base-Layout basieren, HTMX und Alpine.js nutzen und eine gültige Content Security Policy einhalten.

**Eingangszustand:**
Es gibt keine Templ-Templates und kein definiertes HTML-Grundgerüst.

**Ausgangszustand:**
Ein lauffähiges Base-Layout ist vorhanden, das aus `manifest.json` die gehashten Asset-Pfade liest, HTMX und Alpine.js lokal einbindet und eine CSP-konforme Struktur vorgibt.

**Akzeptanzkriterien:**

* `templ generate` erzeugt aus allen `.templ`-Dateien valide `_templ.go`-Dateien.
* Das Base-Layout ist als Templ-Komponente `BaseLayout(title string, content templ.Component)` implementiert.
* Das Base-Layout erzeugt valides HTML5 mit `<!DOCTYPE html>`, `<html lang="de">`, `<head>` und `<body>`.
* HTMX wird aus `manifest.json` unter `/static/htmx.<hash>.min.js` eingebunden; kein CDN.
* Die HTMX-SSE-Extension wird aus `/static/htmx-sse.<hash>.js` eingebunden; kein CDN.
* Alpine.js wird aus `/static/alpine.<hash>.min.js` eingebunden; kein CDN.
* `app.js` wird aus `/static/app.<hash>.js` eingebunden; kein CDN.
* `app.css` wird aus `/static/app.<hash>.css` eingebunden.
* Alle Asset-Pfade werden ausschließlich über die beim Start geladene `manifest.json` aufgelöst; keine hartcodierten Hashes im Template.
* Der `Content-Security-Policy`-Header erlaubt `script-src 'self'`; kein `'unsafe-inline'` und kein `'unsafe-eval'`.
* Der `Content-Security-Policy`-Header erlaubt `style-src 'self'`; kein `'unsafe-inline'`.
* Das Layout enthält ein `<meta name="csrf-token">` mit dem aktuellen CSRF-Token, damit `app.js` ihn für HTMX-Requests auslesen kann.
* Das Layout enthält ein `<div id="notifications">` als Ziel für HTMX-Out-of-Band-Updates von Benachrichtigungen.
* Das Layout enthält ein Navigationselement mit Platzhaltern für Systemfilter und Projektliste; konkrete Inhalte kommen in späteren Stories.
* Dark-Mode-Toggle ist als Button mit `data-theme-toggle` im Layout vorhanden; Alpine.js steuert die Klasse auf `<html>`.
* Die Systempräferenz `prefers-color-scheme` wird beim ersten Laden ausgewertet.
* Ein Handler in `cmd/caldo/main.go` liefert eine Beispiel-Route `GET /` mit dem Base-Layout und einer leeren Inhaltskomponente; `go test ./...` läuft fehlerfrei durch.

---

# Epic 3 — Datenmodell und Repository-Basis

## Story 3.1 — Settings-Singleton persistieren

**Ziel:**
Setup-, CalDAV-, Sync- und UI-Einstellungen haben eine zentrale Persistenz.

**Eingangszustand:**
Es gibt keine persistierten Einstellungen.

**Ausgangszustand:**
Eine Singleton-Settings-Zeile steuert Setup und Normalbetrieb.

**Akzeptanzkriterien:**

* Es existiert genau eine Settings-Zeile mit `id='default'`.
* `setup_complete` startet bei `false`.
* `setup_step` startet bei `caldav`.
* Sync-Intervall defaultet auf 15 Minuten.
* UI-Sprache defaultet auf Deutsch.
* Dark Mode defaultet auf Systempräferenz.
* `default_project_id` darf vor Setup-Abschluss leer sein.

---

## Story 3.2 — Projekte als CalDAV-Kalender modellieren

**Ziel:**
CalDAV-Kalender können lokal als Projekte verwaltet werden.

**Eingangszustand:**
Es gibt keine Projektpersistenz.

**Ausgangszustand:**
Projekte speichern Kalenderbezug, Sync-Metadaten und Versionen.

**Akzeptanzkriterien:**

* Projekt enthält Kalender-HREF und Anzeigename.
* Projekt enthält `ctag`, `sync_token` und `sync_strategy`.
* `sync_strategy` unterstützt `webdav_sync`, `ctag`, `fullscan`.
* Projekt enthält `server_version`.
* Ein Projekt kann als Default-Projekt markiert werden.
* Projektänderungen können per Optimistic Locking abgesichert werden.

---

## Story 3.3 — Aufgabenmodell persistieren

**Ziel:**
VTODO-Aufgaben können lokal vollständig genug gespeichert werden.

**Eingangszustand:**
Es gibt keine lokale Aufgabenpersistenz.

**Ausgangszustand:**
Tasks speichern normalisierte Felder, Raw-VTODO, Sync-Status und Versionen.

**Akzeptanzkriterien:**

* Task enthält Projektbezug, UID, HREF und ETag.
* Task enthält `server_version`.
* Task enthält Titel, Beschreibung, Status, Fälligkeit, Priorität und RRULE.
* Task enthält `raw_vtodo`.
* Task kann `base_vtodo` speichern.
* Task enthält `sync_status`.
* Task kann Parent-Bezug für Unteraufgaben speichern.
* Denormalisierte Suchfelder für Projekt- und Labelnamen sind vorhanden.

---

## Story 3.4 — Labels, Task-Labels und Favoriten-Grundlage

**Ziel:**
Labels und Favoriten können lokal und VTODO-kompatibel abgebildet werden.

**Eingangszustand:**
Es gibt keine Labelstruktur.

**Ausgangszustand:**
Labels sind lokal gespeichert und Aufgaben zuordenbar.

**Akzeptanzkriterien:**

* Labels haben eindeutige Namen.
* Tasks können mehrere Labels haben.
* Neue Labels können automatisch entstehen.
* Labels sind als VTODO-Categories abbildbar.
* `STARRED` ist als reservierte Kategorie für Favoriten vorgesehen.
* Labeldaten können für Suche und Filter denormalisiert werden.

---

## Story 3.5 — Undo- und Konflikttabellen anlegen

**Ziel:**
Spätere Undo- und Konfliktlogik hat eine sichere Datenbasis.

**Eingangszustand:**
Vorversionen werden nicht persistiert.

**Ausgangszustand:**
Undo-Snapshots und Konflikte sind als eigene Entitäten speicherbar.

**Akzeptanzkriterien:**

* Undo-Snapshot speichert Session, Tab, Task, Aktion, VTODO-Snapshot und Ablaufzeit.
* Pro `(session_id, tab_id)` gibt es maximal einen Undo-Snapshot.
* Konflikte speichern Base-, lokale, Remote- und gelöste VTODO-Versionen.
* Konflikte können ungelöst oder gelöst sein.
* Gelöste Konflikte können später bereinigt werden.
* Konflikte sind nicht nur ein Statusfeld auf Tasks.

---

# Epic 4 — Secret Handling und CalDAV-Verbindungsbasis

## Story 4.1 — CalDAV-Credentials verschlüsselt speichern

**Ziel:**
Zugangsdaten werden niemals im Klartext persistiert.

**Eingangszustand:**
Es gibt keine Secret-Speicherung.

**Ausgangszustand:**
CalDAV-Passwörter werden mit AES-256-GCM gespeichert.

**Akzeptanzkriterien:**

* Der Schlüssel stammt aus `ENCRYPTION_KEY`.
* Speicherformat enthält Version, Nonce und Ciphertext.
* Das Passwort wird nicht im Klartext gespeichert.
* Credentials werden nicht geloggt.
* Formal ungültiger Key verhindert den Start.
* Formal gültiger, aber falscher Key verhindert nicht den Start, macht CalDAV aber unavailable.

---

## Story 4.2 — CalDAV-Verbindungstest

**Ziel:**
CalDAV-Konfiguration wird nur nach echtem Verbindungstest akzeptiert.

**Eingangszustand:**
Es gibt keine geprüfte CalDAV-Verbindung.

**Ausgangszustand:**
CalDAV-URL und Credentials können getestet und Capability-Daten gespeichert werden.

**Akzeptanzkriterien:**

* Der Test nutzt einen echten CalDAV/WebDAV-Request.
* Der Test erkennt globale Account-/Server-Capabilities.
* WebDAV-Sync-, CTag-, ETag- und Fullscan-Fähigkeiten werden gespeichert.
* Fehlschlag zeigt einen sicheren Fehler ohne Secrets.
* Fehlschlag markiert die Konfiguration nicht als erfolgreich.
* Timeouts werden eingehalten.

---

# Epic 5 — Setup-Wizard und Initialimport

## Story 5.1 — Setup-Gate für Erststart

**Ziel:**
Die normale App ist vor abgeschlossenem Setup nicht erreichbar.

**Eingangszustand:**
Unkonfigurierte Installationen könnten normale Routen verwenden.

**Ausgangszustand:**
`setup_complete=false` blockiert Normalbetrieb hart.

**Akzeptanzkriterien:**

* Bei `setup_complete=false` sind nur Setup-Routen und `/health` erreichbar.
* Andere Routen leiten nach `/setup`.
* Setup-Routen laufen durch Proxy-Auth.
* Mutierende Setup-Routen laufen durch CSRF.
* Der Wizard-Zustand liegt serverseitig in Settings.
* Normalbetrieb und Setup teilen DB und Router, sind aber durch Gate getrennt.

---

## Story 5.2 — Setup-Schritt CalDAV

**Ziel:**
Der Nutzer kann CalDAV-Zugangsdaten im Erststart erfassen und prüfen.

**Eingangszustand:**
Setup steht auf Schritt `caldav`.

**Ausgangszustand:**
Bei erfolgreichem Test sind Credentials verschlüsselt gespeichert und der Wizard geht zu Kalendern.

**Akzeptanzkriterien:**

* CalDAV-URL, Benutzername und Passwort/App-Passwort können eingegeben werden.
* Credentials werden sofort verschlüsselt gespeichert.
* Credentials werden nicht im Browser oder in Session-State gehalten.
* Ein echter Verbindungstest wird ausgeführt.
* Capabilities werden gespeichert.
* Bei Erfolg wird `setup_step='calendars'`.
* Bei Fehler bleibt `setup_step='caldav'`.

---

## Story 5.3 — Setup-Schritt Kalenderauswahl und Default-Projekt

**Ziel:**
Der Nutzer wählt zu synchronisierende Kalender und ein Default-Projekt.

**Eingangszustand:**
CalDAV-Verbindung ist erfolgreich getestet.

**Ausgangszustand:**
Ausgewählte Kalender sind als Projekte gespeichert, ein Default-Projekt ist gesetzt.

**Akzeptanzkriterien:**

* Verfügbare CalDAV-Kalender werden geladen.
* Der Nutzer kann mehrere Kalender auswählen.
* Der Nutzer kann ein Default-Projekt wählen.
* Optional kann ein neues Default-Projekt angelegt werden.
* Ohne Default-Projekt ist Fortfahren nicht möglich.
* Für ausgewählte Kalender wird initial eine Sync-Strategie gesetzt.
* Bei Erfolg wird `setup_step='import'`.

---

## Story 5.4 — Initialimport ausführen

**Ziel:**
Bestehende VTODOs werden vor Nutzung importiert.

**Eingangszustand:**
Kalenderauswahl und Default-Projekt sind abgeschlossen.

**Ausgangszustand:**
Alle ausgewählten Kalender sind initial importiert.

**Akzeptanzkriterien:**

* Initialimport läuft über alle ausgewählten Kalender.
* Import verwendet Full-Scan-Modus.
* Importierte VTODOs werden als `synced` übernommen.
* `base_vtodo = raw_vtodo`.
* Es wird keine Konfliktbehandlung ausgeführt.
* Normalisierte Kernfelder werden aufgebaut.
* FTS-Indexdaten werden vorbereitet.
* Fortschritt wird über Setup-SSE gemeldet.
* Setup-SSE sendet keine normalen Task-/Sync-Events.

---

## Story 5.5 — Setup abschließen und Scheduler aktivieren

**Ziel:**
Nach erfolgreichem Initialimport wechselt Caldo ohne Neustart in den Normalbetrieb.

**Eingangszustand:**
Initialimport ist erfolgreich abgeschlossen.

**Ausgangszustand:**
`setup_complete=true`, normale Routen sind erreichbar, Scheduler läuft.

**Akzeptanzkriterien:**

* Kalenderauswahl, Default-Projekt und Importerfolg werden geprüft.
* `setup_step='complete'` wird gesetzt.
* `setup_complete=true` wird gesetzt.
* Nach Commit lässt das Setup-Gate normale Routen zu.
* Der Scheduler wird gestartet.
* Ein Scheduler-Startfehler rollt Setup nicht zurück.
* Der Nutzer wird zur normalen App-UI weitergeleitet.

---

# Epic 6 — VTODO-Roundtrip und CalDAV-Write-Basis

## Story 6.1 — VTODO-Felder extrahieren

**Ziel:**
Caldo kann VTODOs lesen und bekannte Felder normalisieren.

**Eingangszustand:**
Raw-VTODOs sind nicht fachlich auswertbar.

**Ausgangszustand:**
Bekannte Felder sind aus Raw-VTODOs extrahierbar.

**Akzeptanzkriterien:**

* Titel wird extrahiert.
* Beschreibung wird extrahiert.
* Fälligkeit mit und ohne Uhrzeit wird extrahiert.
* Status und Completed werden extrahiert.
* Priorität wird extrahiert.
* Kategorien werden extrahiert.
* RRULE wird als Rohstring extrahiert.
* Parent-Referenzen werden extrahiert.
* Unbekannte Properties bleiben im Raw-VTODO erhalten.

---

## Story 6.2 — VTODO-Patching ohne Datenverlust

**Ziel:**
Bekannte Feldänderungen zerstören keine unbekannten VTODO-Inhalte.

**Eingangszustand:**
Änderungen könnten Raw-VTODO vollständig neu serialisieren.

**Ausgangszustand:**
Nur explizit geänderte bekannte Felder werden gepatcht.

**Akzeptanzkriterien:**

* Unbekannte Properties bleiben erhalten.
* `VALARM` bleibt erhalten.
* `ATTACH` bleibt erhalten.
* RRULE wird nur bei expliziter Wiederholungsänderung verändert.
* Raw-VTODO ist Roundtrip-Quelle.
* Tests decken unbekannte Properties, VALARM, ATTACH und RRULE-Erhalt ab.

---

## Story 6.3 — CalDAV-Operationen mit Timeout und Retry-Policy

**Ziel:**
CalDAV-Zugriffe sind robust und kontrolliert.

**Eingangszustand:**
Remote-Operationen haben keine einheitliche Fehlerpolitik.

**Ausgangszustand:**
CalDAV-Operationen folgen festen Timeouts, Retry-Regeln und Backoff.

**Akzeptanzkriterien:**

* PROPFIND, REPORT, GET, PUT, DELETE, MKCALENDAR und Full-Scan haben definierte Timeouts.
* Sichere idempotente Operationen werden bis maximal 3 Versuche wiederholt.
* PUT Create wird nicht blind wiederholt.
* PUT Update mit `If-Match` darf wiederholt werden.
* `412 Precondition Failed` führt nicht zu Retry, sondern Konfliktbehandlung.
* DELETE mit `404` gilt als Erfolg.
* Backoff nutzt Jitter.

---

# Epic 7 — Aufgaben-Kernfunktionen mit Write-Through

## Story 7.1 — Aufgaben erstellen

**Ziel:**
Neue Aufgaben werden im Default- oder gewählten Projekt erstellt und sofort zu CalDAV geschrieben.

**Eingangszustand:**
Der Nutzer hat mindestens ein Projekt und ein Default-Projekt.

**Ausgangszustand:**
Eine neue Aufgabe existiert lokal und remote.

**Akzeptanzkriterien:**

* Neue Aufgabe benötigt Titel und Projekt.
* Ohne explizites Projekt wird das Default-Projekt verwendet.
* Wenn kein gültiges Default-Projekt existiert, wird Erstellung blockiert.
* Task wird lokal als `pending` vorbereitet.
* CalDAV-Create läuft synchron.
* Erst nach erfolgreichem CalDAV-Write gilt die Aufgabe als gespeichert.
* Bei Erfolg werden HREF, ETag, `sync_status=synced` und Version gespeichert.
* Bei Fehler sieht der Nutzer eine Fehlermeldung; keine stille Speicherung.

---

## Story 7.2 — Aufgaben bearbeiten

**Ziel:**
Kernfelder einer Aufgabe können geändert werden.

**Eingangszustand:**
Eine synchronisierte Aufgabe existiert.

**Ausgangszustand:**
Die Änderung ist lokal versioniert und remote gespeichert.

**Akzeptanzkriterien:**

* Bearbeitbar sind Titel, Beschreibung, Fälligkeit, Priorität, Status, Projekt und Labels.
* Request enthält `expected_version`.
* Bei Versionsabweichung wird nicht gespeichert.
* Vor Änderung wird ein Undo-Snapshot erstellt, sofern die Aktion Undo-fähig ist.
* Änderung wird als `pending` versioniert.
* CalDAV-Write läuft synchron.
* Bei Erfolg wird neuer ETag gespeichert und `sync_status=synced`.
* Bei Fehler bleibt der Fehler sichtbar und die Änderung gilt nicht fachlich gespeichert.

---

## Story 7.3 — Aufgaben erledigen und wieder öffnen

**Ziel:**
Aufgabenstatus wird CalDAV-kompatibel geändert.

**Eingangszustand:**
Eine offene oder erledigte Aufgabe existiert.

**Ausgangszustand:**
Der Status ist lokal und remote konsistent.

**Akzeptanzkriterien:**

* Erledigen setzt VTODO-Completed/Status.
* Wieder öffnen entfernt oder aktualisiert Completed/Status konsistent.
* `expected_version` wird geprüft.
* Änderung ist Undo-fähig.
* Erledigte Aufgaben sind standardmäßig ausgeblendet.
* Bei CalDAV-Fehler wird der Nutzer informiert.
* Wiederkehrende Aufgaben behalten ihre RRULE unverändert.

---

## Story 7.4 — Aufgaben löschen

**Ziel:**
Aufgaben können endgültig gelöscht werden.

**Eingangszustand:**
Eine Aufgabe existiert lokal und remote.

**Ausgangszustand:**
Die Aufgabe ist nach erfolgreichem CalDAV-Delete lokal entfernt.

**Akzeptanzkriterien:**

* Vor Löschen erscheint eine Bestätigung.
* `expected_version` wird geprüft.
* Vor Löschen wird ein Undo-Snapshot erstellt.
* CalDAV-DELETE läuft synchron.
* Lokale Task-Zeile wird erst nach erfolgreichem DELETE entfernt.
* `404 Not Found` beim DELETE gilt als Erfolg.
* Es gibt keinen Papierkorb.
* Löschkonflikte können später erkannt werden.

---

## Story 7.5 — Elternaufgabe mit offenen Unteraufgaben erledigen

**Ziel:**
Der Nutzer entscheidet explizit, wie offene Unteraufgaben behandelt werden.

**Eingangszustand:**
Eine Elternaufgabe hat offene direkte Unteraufgaben.

**Ausgangszustand:**
Die gewählte Aktion ist zu CalDAV geschrieben oder abgebrochen.

**Akzeptanzkriterien:**

* Beim Erledigen erscheint ein Dialog.
* Option 1: nur Elternaufgabe erledigen.
* Option 2: Elternaufgabe und offene Unteraufgaben erledigen.
* Option 3: abbrechen.
* Die gewählte Änderung prüft Versionen.
* Jede betroffene Task wird zu CalDAV geschrieben.
* Keine Unteraufgaben werden stillschweigend miterledigt.

---

# Epic 8 — Projekte, Kalender und Default-Projekt im Normalbetrieb

## Story 8.1 — Projekt anlegen

**Ziel:**
Ein neues Projekt erstellt einen neuen CalDAV-Kalender.

**Eingangszustand:**
Der Nutzer ist im Normalbetrieb.

**Ausgangszustand:**
Ein neues Projekt existiert lokal und remote.

**Akzeptanzkriterien:**

* Projektanlage nutzt CalDAV-Kalenderanlage.
* Lokales Projekt wird erst nach erfolgreicher Remote-Anlage gespeichert.
* Projekt kann leer sein.
* Fehler werden sichtbar angezeigt.
* Es gibt kein optimistisches UI-Update vor Remote-Erfolg.

---

## Story 8.2 — Projekt umbenennen

**Ziel:**
Projektname und CalDAV-Kalendername bleiben konsistent.

**Eingangszustand:**
Ein Projekt existiert.

**Ausgangszustand:**
Remote-Kalender und lokales Projekt sind umbenannt.

**Akzeptanzkriterien:**

* Request enthält `expected_version`.
* Remote-Kalender wird zuerst umbenannt.
* Lokales Projekt wird erst nach Remote-Erfolg aktualisiert.
* Denormalisierte `project_name`-Felder betroffener Tasks werden aktualisiert.
* Suchindex wird für betroffene Tasks aktualisiert.
* Fehler werden ohne lokale Teilumbenennung angezeigt.

---

## Story 8.3 — Projekt löschen

**Ziel:**
Ein Projekt kann nach starker Bestätigung endgültig gelöscht werden.

**Eingangszustand:**
Ein Projekt mit oder ohne Tasks existiert.

**Ausgangszustand:**
Remote-Kalender, lokales Projekt und zugehörige lokale Tasks sind entfernt.

**Akzeptanzkriterien:**

* Bestätigung zeigt Projektname und Anzahl betroffener Tasks.
* Starke Bestätigung ist erforderlich.
* CalDAV-Kalender wird gelöscht.
* Es werden keine einzelnen Task-DELETEs für Projektlöschung gesendet.
* Lokales Projekt und lokale Tasks werden nach Remote-Erfolg gelöscht.
* FTS-Einträge werden entfernt.
* War es das Default-Projekt, wird `default_project_id=NULL`.
* Neue Task-Erstellung ist danach blockiert, bis ein neues Default-Projekt gewählt ist.

---

## Story 8.4 — Remote gelöschte Kalender bereinigen

**Ziel:**
Remote-Kalenderlöschung wird autoritativ übernommen.

**Eingangszustand:**
Ein lokal bekanntes Projekt existiert remote nicht mehr.

**Ausgangszustand:**
Das lokale Projekt und abhängige Daten sind bereinigt.

**Akzeptanzkriterien:**

* Remote-Kalenderlöschung erzeugt keinen Projektkonflikt.
* Lokales Projekt wird gelöscht.
* Zugehörige Tasks werden gelöscht.
* FTS-Einträge werden gelöscht.
* Undo-Snapshots für betroffene Tasks werden gelöscht.
* Bei pending Tasks wird eine einmalige Warnung angezeigt.
* War es das Default-Projekt, muss der Nutzer ein neues Default-Projekt wählen.

---

# Epic 9 — Suche, FTS und Basisansichten

## Story 9.1 — FTS5-Suchindex aufbauen

**Ziel:**
Globale Suche ist performant und konsistent.

**Eingangszustand:**
Tasks können nicht freitextbasiert durchsucht werden.

**Ausgangszustand:**
FTS5 indexiert aktive Aufgabenfelder.

**Akzeptanzkriterien:**

* FTS5 indexiert Titel, Beschreibung, Labelnamen und Projektnamen.
* Trigger halten strukturelle Konsistenz bei Insert, Update und Delete.
* Go-Layer pflegt denormalisierte Suchfelder.
* Erledigte Aufgaben werden standardmäßig ausgeschlossen.
* Undo-Snapshots, Konfliktversionen und Historie werden nicht indexiert.
* Umlaut-/Diakritik- und Prefix-Suche sind getestet.

---

## Story 9.2 — Globale Suche

**Ziel:**
Der Nutzer findet aktive Aufgaben schnell.

**Eingangszustand:**
Es gibt keine globale Suchfunktion.

**Ausgangszustand:**
Suche findet aktive Aufgaben über Text, Projekt- und Labeltokens.

**Akzeptanzkriterien:**

* Suche ist als globale Freitextsuche erkennbar.
* Titel und Beschreibung werden durchsucht.
* Label- und Projektnamen werden durchsucht.
* `#Projekt` schränkt auf Projekt ein.
* `@Label` schränkt auf Label ein.
* Erledigte Aufgaben werden standardmäßig nicht durchsucht.
* Konfliktversionen und Undo-Snapshots werden nicht durchsucht.
* Suche ist per Tastaturkürzel erreichbar.

---

## Story 9.3 — Heute-, Demnächst- und Überfällig-Ansichten

**Ziel:**
Die wichtigsten Systemansichten sind verfügbar.

**Eingangszustand:**
Tasks sind nur projektbezogen sichtbar.

**Ausgangszustand:**
Heute, Demnächst und Überfällig zeigen passende aktive Aufgaben.

**Akzeptanzkriterien:**

* Heute zeigt Aufgaben mit heutigem Fälligkeitsdatum.
* Heute zeigt zusätzlich überfällige Aufgaben.
* Demnächst nutzt den konfigurierten Zeitraum.
* Default für Demnächst ist 7 Tage.
* Aufgaben ohne Fälligkeit erscheinen nicht automatisch in Heute oder Demnächst.
* Erledigte Aufgaben sind standardmäßig ausgeblendet.
* Einstellung `show_completed` wird berücksichtigt.

---

# Epic 10 — Sync Engine und Scheduler

## Story 10.1 — Sync Engine mit Fallback-Strategien

**Ziel:**
Remote-Änderungen werden robust aus CalDAV importiert.

**Eingangszustand:**
Es gibt keinen normalen Sync-Lauf.

**Ausgangszustand:**
Sync nutzt WebDAV Sync, CTag/ETag oder Full-Scan je Projekt.

**Akzeptanzkriterien:**

* Pro Projekt wird die aktuelle Sync-Strategie gelesen.
* WebDAV Sync wird bevorzugt.
* Bei Nichtunterstützung fällt Sync auf CTag/ETag zurück.
* Bei weiterer Unzuverlässigkeit fällt Sync auf Full-Scan zurück.
* Effektive Strategie wird pro Projekt gespeichert.
* Remote-Fetching und Parsing passieren außerhalb des Write-Mutex.
* DB-Mutationen passieren in Chunks mit Write-Mutex.
* Importierte Remote-Änderungen erhöhen `server_version`.

---

## Story 10.2 — Manueller Sync

**Ziel:**
Der Nutzer kann jederzeit einen Full-Sync starten.

**Eingangszustand:**
Remote-Änderungen kommen nur über Initialimport oder Writes herein.

**Ausgangszustand:**
Ein manueller Sync kann gestartet und überwacht werden.

**Akzeptanzkriterien:**

* Es gibt einen sichtbaren manuellen Sync-Zugriff.
* Ein laufender Sync verhindert parallele Full-Syncs.
* Bei laufendem Sync wird kein zweiter Lauf queued.
* UI zeigt aktuellen Sync-Status.
* UI zeigt letzten erfolgreichen Sync-Zeitpunkt.
* Abschluss oder Fehler wird sichtbar gemeldet.
* SSE kann Sync-Status verteilen.

---

## Story 10.3 — Periodischer Scheduler

**Ziel:**
Remote-Änderungen werden serverseitig regelmäßig abgeholt.

**Eingangszustand:**
Es gibt keinen periodischen Sync.

**Ausgangszustand:**
Scheduler führt Full-Syncs im konfigurierten Intervall aus.

**Akzeptanzkriterien:**

* Scheduler startet erst nach `setup_complete=true`.
* Default-Intervall ist 15 Minuten.
* Intervalländerungen starten den Ticker kontrolliert neu.
* Scheduler läuft im Go-Prozess.
* Kein Browser-Polling dient als Scheduler.
* Kein Cron, Redis oder externer Job-Runner wird benötigt.
* Scheduler startet keinen neuen Sync, solange einer aktiv ist.

---

## Story 10.4 — Sync-Cleanup-Jobs

**Ziel:**
Kurzlebige technische Daten werden automatisch bereinigt.

**Eingangszustand:**
Undo-Snapshots und gelöste Konflikte bleiben unbegrenzt liegen.

**Ausgangszustand:**
Cleanup läuft regelmäßig im Sync-/Scheduler-Kontext.

**Akzeptanzkriterien:**

* Abgelaufene Undo-Snapshots werden bei Sync-Läufen gelöscht.
* Gelöste Konflikte älter als 7 Tage werden täglich gelöscht.
* Ungelöste Konflikte werden nie automatisch gelöscht.
* Cleanup läuft über den globalen Write-Mutex.
* Cleanup loggt keine Task-Inhalte.

---

# Epic 11 — Optimistic Locking, SSE und Mehr-Tab-Verhalten

## Story 11.1 — Optimistic Locking für mutierende Requests

**Ziel:**
Veraltete Tabs überschreiben keine neueren Daten.

**Eingangszustand:**
Mutierende Requests könnten stale Daten speichern.

**Ausgangszustand:**
Alle relevanten Mutationen prüfen `expected_version`.

**Akzeptanzkriterien:**

* Task-mutierende Requests enthalten immer `expected_version`.
* Projekt- und Filteränderungen nutzen ebenfalls Versionen.
* Bei Versionsgleichheit darf verarbeitet werden.
* Bei Versionsabweichung wird nicht gespeichert.
* Nutzer erhält Konflikt- oder Aktualisierungshinweis.
* `etag` wird nie als UI-Version genutzt.
* `server_version` wird nie als CalDAV-ETag genutzt.

---

## Story 11.2 — Globaler SSE-Endpunkt

**Ziel:**
Offene Tabs werden über relevante Änderungen informiert.

**Eingangszustand:**
Mehrere Tabs erfahren nichts voneinander.

**Ausgangszustand:**
`GET /events` verteilt Task-, Projekt-, Sync- und Konflikt-Events.

**Akzeptanzkriterien:**

* Es gibt genau einen normalen SSE-Endpunkt.
* Jede Verbindung hat eine `connection_id`.
* Events enthalten Typ, Ressource, Version und Origin-Connection.
* Events werden nach DB-Commit gesendet.
* Die auslösende Verbindung erhält ihr Ergebnis primär über die HTTP-Response.
* Andere Verbindungen erhalten Broadcasts.
* Setup-SSE und Normalbetrieb-SSE sind getrennt.

---

## Story 11.3 — Fokus-Refresh

**Ziel:**
Lange offene Tabs aktualisieren veraltete Fragmente beim Zurückkehren.

**Eingangszustand:**
Ein Tab kann lange mit alten Versionen offen bleiben.

**Ausgangszustand:**
Der Tab kann bekannte Task-Versionen gegen den Server prüfen.

**Akzeptanzkriterien:**

* `GET /api/tasks/versions` nimmt bekannte Task-IDs entgegen.
* Response enthält aktuelle Versionen.
* Der Client lädt nur veraltete Fragmente nach.
* Offene Formulare ohne lokale Änderungen dürfen aktualisiert werden.
* Offene Formulare mit lokalen Änderungen werden nicht überschrieben.
* Bei lokalen Änderungen wird ein Hinweis angezeigt.

---

# Epic 12 — Undo

## Story 12.1 — Undo-Snapshot bei Undo-fähigen Aktionen

**Ziel:**
Die letzte Undo-fähige Aktion pro Tab kann rückgängig gemacht werden.

**Eingangszustand:**
Vorherige Task-Zustände werden nicht gespeichert.

**Ausgangszustand:**
Vor Änderung wird ein tab-lokaler Snapshot gespeichert.

**Akzeptanzkriterien:**

* Snapshot und ursprüngliche Änderung liegen in derselben DB-Transaktion.
* Pro `(session_id, tab_id)` existiert maximal ein Snapshot.
* Neuer Snapshot ersetzt vorherigen Snapshot desselben Tabs.
* Snapshot enthält Raw-VTODO, normalisierte Felder und `etag_at_snapshot`.
* Snapshot läuft nach 5 Minuten ab.
* Reload im selben Tab erhält Undo-Verfügbarkeit.

---

## Story 12.2 — Undo ausführen

**Ziel:**
Undo ist eine neue fachliche Gegenänderung mit CalDAV-Write.

**Eingangszustand:**
Ein gültiger Undo-Snapshot existiert.

**Ausgangszustand:**
Der vorherige Zustand ist wiederhergestellt oder ein Fehler/Konflikt ist sichtbar.

**Akzeptanzkriterien:**

* Snapshot wird anhand von Session und Tab geladen.
* Abgelaufener Snapshot kann nicht verwendet werden.
* Aktuelle Task wird mit `etag_at_snapshot` verglichen.
* Bei abweichendem ETag wird ein Konflikt erzeugt.
* Zielzustand wird als `pending` gespeichert.
* CalDAV-Write läuft synchron.
* Erst nach erfolgreichem Write wird der Snapshot gelöscht.
* Bei Write-Fehler bleibt der Snapshot erhalten, sofern nicht abgelaufen.

---

## Story 12.3 — Undo für gelöschte Aufgaben

**Ziel:**
Eine gelöschte Aufgabe kann aus Snapshot neu erstellt werden.

**Eingangszustand:**
Eine Aufgabe wurde erfolgreich gelöscht und ein Snapshot existiert.

**Ausgangszustand:**
Die Aufgabe wird als neue CalDAV-Ressource wiederhergestellt.

**Akzeptanzkriterien:**

* Undo rekonstruiert Task aus Snapshot.
* VTODO-UID bleibt erhalten, sofern kein Split/Konflikt nötig ist.
* Es wird eine neue CalDAV-Ressource erstellt.
* Bei erfolgreichem Write wird lokale Task-Zeile neu gespeichert.
* Bei Fehler wird der Nutzer informiert.
* Bei zwischenzeitlicher Remote-Änderung entsteht ein Konflikt.

---

# Epic 13 — Konflikte und manuelle Auflösung

## Story 13.1 — Konflikte bei Remote-Sync erkennen

**Ziel:**
Lokale und Remote-Änderungen werden verlustfrei verglichen.

**Eingangszustand:**
Remote-Import könnte lokale Änderungen überschreiben.

**Ausgangszustand:**
Konflikte werden als eigene Entitäten erzeugt.

**Akzeptanzkriterien:**

* `base_vtodo`, `local_vtodo` und `remote_vtodo` werden berücksichtigt.
* Bei fehlender Base ist Auto-Merge deaktiviert.
* Feldbasierter Auto-Merge wird nur bei konfliktfreien Änderungen ausgeführt.
* Bei echtem Feldkonflikt entsteht ein Konfliktdatensatz.
* `tasks.sync_status=conflict` blockiert die betroffene Aufgabe.
* Andere Aufgaben synchronisieren weiter.

---

## Story 13.2 — Löschkonflikte erkennen

**Ziel:**
Edit/Delete- und Delete/Edit-Fälle werden explizit behandelt.

**Eingangszustand:**
Eine Seite hat geändert, die andere gelöscht.

**Ausgangszustand:**
Der Nutzer kann über Wiederherstellung oder Löschung entscheiden.

**Akzeptanzkriterien:**

* Lokal geändert, remote gelöscht erzeugt `edit_delete`.
* Lokal gelöscht, remote geändert erzeugt `delete_edit`.
* Fehlende Seite wird als `NULL`-VTODO gespeichert.
* Die Konfliktansicht bietet passende Optionen.
* Es gibt keinen stillen Datenverlust.
* Nicht betroffene Tasks bleiben synchronisierbar.

---

## Story 13.3 — Globale Konfliktansicht

**Ziel:**
Alle ungelösten Konflikte sind zentral auffindbar.

**Eingangszustand:**
Konflikte wären nur indirekt sichtbar.

**Ausgangszustand:**
Es gibt eine globale Konfliktliste und Detailansichten.

**Akzeptanzkriterien:**

* Hauptnavigation enthält Konflikte.
* Globale Ansicht zeigt ungelöste Konflikte.
* Konfliktdetail zeigt lokale, remote und ggf. Base-Information fachlich verständlich.
* Konfliktbehaftete Aufgaben öffnen direkt in Konfliktansicht.
* Gelöste Konflikte verschwinden aus der aktiven Liste.
* Ungelöste Konflikte werden nicht automatisch gelöscht.

---

## Story 13.4 — Konflikt manuell lösen

**Ziel:**
Der Nutzer kann Konflikte ohne Datenverlust auflösen.

**Eingangszustand:**
Ein ungelöster Konflikt existiert.

**Ausgangszustand:**
Eine gewählte Lösung ist lokal und remote gespeichert.

**Akzeptanzkriterien:**

* Lokale Version übernehmen ist möglich.
* Remote-Version übernehmen ist möglich.
* Felder manuell auswählen ist möglich.
* Beide Versionen behalten ist möglich.
* Mindestens Titel, Beschreibung, Fälligkeit, Priorität, Labels, Projekt, Status und Unteraufgaben sind feldweise lösbar.
* Lösung wird zu CalDAV geschrieben.
* Konflikt erhält `resolved_at` und `resolution`.
* Bei Write-Fehler bleibt Konflikt ungelöst.

---

## Story 13.5 — Beide Versionen behalten

**Ziel:**
Widersprüchliche Versionen können als separate Aufgaben erhalten bleiben.

**Eingangszustand:**
Ein Konflikt mit lokaler und Remote-Version existiert.

**Ausgangszustand:**
Beide Versionen existieren als eigenständige Aufgaben.

**Akzeptanzkriterien:**

* Remote-Version wird als neue Task mit neuer UID zu CalDAV geschrieben.
* Lokale Version behält ihre UID.
* Beide Tasks liegen im selben Projekt.
* Es wird keine Parent-Verknüpfung zwischen beiden erzeugt.
* Konflikt wird mit `resolution=split` markiert.
* Bei Teilfehlern wird kein still inkonsistenter Zustand als gelöst markiert.

---

# Epic 14 — Quick Add und natürliche Eingabe

## Story 14.1 — Quick-Add-Grundfunktion

**Ziel:**
Der Nutzer kann Aufgaben reibungsarm erfassen.

**Eingangszustand:**
Aufgaben werden nur über vollständige Formulare erstellt.

**Ausgangszustand:**
Schnellanlage erzeugt einen Aufgabenentwurf und kann speichern.

**Akzeptanzkriterien:**

* Quick Add ist per Tastaturkürzel erreichbar.
* Freitext wird als Titel erkannt.
* Default-Projekt wird verwendet, wenn kein Projekt angegeben ist.
* Vorschau zeigt erkannten Titel, Projekt, Labels, Datum, Wiederholung und Priorität.
* Speichern schreibt sofort zu CalDAV.
* Fehler verhindern stille Speicherung.

---

## Story 14.2 — Projekt-, Label- und Prioritätstokens

**Ziel:**
Todoist-nahe Schnellsyntax ist nutzbar.

**Eingangszustand:**
Quick Add erkennt nur Freitext.

**Ausgangszustand:**
`#`, `@` und `!`-Tokens werden erkannt und aufgelöst.

**Akzeptanzkriterien:**

* `#Projekt` setzt das Projekt.
* Unbekanntes Projekt wird nicht still ignoriert.
* UI zeigt Projektvorschlag oder Anlageoption.
* Neues Projekt erzeugt einen CalDAV-Kalender.
* `@Label` setzt Labels.
* Neues Label wird automatisch angelegt.
* `!high`, `!medium`, `!low`, `!1`, `!2`, `!3` werden erkannt.
* Gemeinsame Tokenregeln divergieren nicht von Suche/Filter.

---

## Story 14.3 — Natürliche Datumseingabe Deutsch/Englisch

**Ziel:**
Fälligkeitsdaten können natürlich eingegeben werden.

**Eingangszustand:**
Datum muss manuell gesetzt werden.

**Ausgangszustand:**
Deutsch- und Englischmuster werden erkannt.

**Akzeptanzkriterien:**

* `heute`, `morgen`, `übermorgen` werden erkannt.
* `today`, `tomorrow` werden erkannt.
* `nächsten Montag` und `next monday` werden erkannt.
* `in 3 Tagen` und `in 3 days` werden erkannt.
* Deutsche und englische Wochentage werden erkannt.
* Unbekannte Tokens bleiben Teil des Titels.
* Unbekannte Tokens erzeugen keine Fehlermeldung.
* Parser-Tests laufen ohne HTTP und ohne DB.

---

## Story 14.4 — Wiederholungen in Quick Add

**Ziel:**
Wiederkehrende Aufgaben können direkt über natürliche Eingabe erstellt werden.

**Eingangszustand:**
Quick Add erzeugt nur Einzelaufgaben.

**Ausgangszustand:**
MVP-Wiederholungsmuster erzeugen RRULEs.

**Akzeptanzkriterien:**

* `jeden Montag` und `every monday` werden erkannt.
* `täglich/daily`, `wöchentlich/weekly`, `monatlich/monthly`, `jährlich/yearly` werden erkannt.
* `werktags/weekdays` wird erkannt.
* `alle X Tage/Wochen/Monate` wird erkannt.
* Erkannte Wiederholung wird nicht nachträglich abgelehnt.
* RRULE wird beim Speichern in VTODO geschrieben.
* Nicht unterstützte komplexe Muster bleiben Freitext oder werden klar nicht als Wiederholung behandelt.

---

# Epic 15 — Filter und gespeicherte Ansichten

### Story 15.1a — Filter-Lexer und Token-Definitionen

**Ziel:**
Filterausdrücke können in eine Folge typisierter Tokens zerlegt werden.

**Eingangszustand:**
Es gibt keine Tokenisierung für Filterqueries.

**Ausgangszustand:**
Ein Lexer nimmt einen Filterstring entgegen und gibt eine Tokenliste zurück.

**Akzeptanzkriterien:**

* Erkannte Token-Typen: `TODAY`, `OVERDUE`, `UPCOMING`, `NO_DATE`, `COMPLETED`, `PRIORITY`, `TEXT`, `PROJECT` (`#`-Prefix), `LABEL` (`@`-Prefix), `BEFORE`, `AFTER`, `AND`, `OR`, `NOT`, `LPAREN`, `RPAREN`, `COLON`, `STRING`, `EOF`.
* Schlüsselwörter sind case-insensitiv: `today`, `TODAY`, `Today` erzeugen denselben Token-Typ.
* Whitespace zwischen Tokens wird übersprungen.
* Unbekannte Zeichenfolgen werden als `STRING`-Token behandelt, nicht als Fehler.
* Der Lexer ist zustandslos und hat keine Abhängigkeit zu Datenbank oder HTTP.
* Unit-Tests laufen ohne DB und ohne HTTP.
* Tests decken alle Token-Typen, Groß-/Kleinschreibung, Sonderzeichen in Strings und leere Eingabe ab.

---

### Story 15.1b — Filter-Parser und AST

**Ziel:**
Eine Tokenliste wird zu einem Syntaxbaum (AST) geparst, der die Operatorprioritäten korrekt abbildet.

**Eingangszustand:**
Story 15.1a ist abgeschlossen; der Lexer liefert Tokenlisten.

**Ausgangszustand:**
Ein rekursiv-deszendenter Parser baut aus der Tokenliste einen AST.

**Akzeptanzkriterien:**

* Der AST unterscheidet folgende Node-Typen: `AndNode`, `OrNode`, `NotNode`, `FilterNode` (Blatt mit Operator und Wert).
* Operatorpriorität: `NOT` bindet stärker als `AND`, `AND` stärker als `OR`.
* Klammerung mit `(` und `)` überschreibt Priorität korrekt.
* `today` ohne weitere Operatoren erzeugt einen validen Einzel-Node-AST.
* Fehlende schließende Klammer ergibt einen `ParseError`; kein Panic.
* Unbekannte Token an unerwarteter Stelle ergeben einen `ParseError`.
* Der Parser hat keine Abhängigkeit zu Datenbank oder HTTP.
* Unit-Tests laufen ohne DB und ohne HTTP.
* Tests decken alle Knotentypen, Prioritätsfälle, Klammerung und Fehlerfälle ab.

---

### Story 15.1c — AST-zu-SQL-Compiler

**Ziel:**
Ein AST wird in eine parametrisierte SQL-WHERE-Klausel übersetzt, die gegen die `tasks`-Tabelle ausgeführt werden kann.

**Eingangszustand:**
Story 15.1b ist abgeschlossen; der Parser liefert ASTs.

**Ausgangszustand:**
Der Compiler erzeugt aus einem AST einen SQL-Fragment-String und eine Parameterliste.

**Akzeptanzkriterien:**

* `TODAY` erzeugt `due_date = ?` mit dem heutigen Datum als Parameter.
* `OVERDUE` erzeugt `due_date < ?` mit dem heutigen Datum als Parameter.
* `UPCOMING` erzeugt `due_date BETWEEN ? AND ?` mit dem konfigurierten Vorschauzeitraum.
* `NO_DATE` erzeugt `due_date IS NULL`.
* `PROJECT` erzeugt einen Vergleich auf `project_name` (denormalisiertes Feld).
* `LABEL` erzeugt einen Vergleich auf `label_names` (denormalisiertes Feld).
* `PRIORITY` erzeugt einen Vergleich auf das `priority`-Feld.
* `COMPLETED` erzeugt einen Filter auf `sync_status` und `completed_at`.
* `TEXT` erzeugt eine FTS5-Subquery gegen den Suchindex.
* `BEFORE` und `AFTER` erzeugen datumbasierte `due_date`-Vergleiche.
* `AND`, `OR`, `NOT` werden korrekt in SQL-Konstrukte übersetzt.
* SQL wird ausschließlich parametrisiert erzeugt; keine String-Interpolation von Nutzerwerten.
* Unbekannte Projekt- oder Labelnamen ergeben keine leere WHERE-Klausel, sondern `1=0` (leere Ergebnismenge).
* Unbekannte AST-Node-Typen ergeben einen `CompileError`.
* Der Compiler hat keine Abhängigkeit zu Datenbank oder HTTP.
* Unit-Tests laufen ohne DB und ohne HTTP.
* Tests decken alle Filtertypen, logische Verknüpfungen, Parametrisierung und Fehlerfälle ab.

---

## Story 15.2 — Gespeicherte Filter verwalten

**Ziel:**
Der Nutzer kann eigene Aufgabenansichten speichern.

**Eingangszustand:**
Filterqueries sind nicht persistierbar.

**Ausgangszustand:**
Filter können angelegt, geändert, gelöscht und favorisiert werden.

**Akzeptanzkriterien:**

* Filter haben Name und Query.
* Filter werden lokal gespeichert.
* Filter werden nicht zu CalDAV synchronisiert.
* Filter können favorisiert werden.
* Filteränderungen nutzen `server_version`.
* Syntaxfehler gespeicherter Queries führen zur Laufzeit zu leerer Ergebnisliste, nicht zu hartem Fehler.
* Favorisierte Filter erscheinen in der Navigation.

---

## Story 15.3 — Systemfilter bereitstellen

**Ziel:**
Pflichtansichten sind ohne manuelle Filteranlage verfügbar.

**Eingangszustand:**
Nur manuelle Navigation existiert.

**Ausgangszustand:**
Systemfilter decken zentrale Aufgabenlisten ab.

**Akzeptanzkriterien:**

* Heute ist verfügbar.
* Demnächst ist verfügbar.
* Überfällig ist verfügbar.
* Favoriten ist verfügbar.
* Aufgaben ohne Datum ist verfügbar.
* Erledigte Aufgaben ist verfügbar, wenn sichtbar geschaltet.
* Konflikte ist verfügbar.
* Systemfilter sind von gespeicherten Nutzerfiltern unterscheidbar.

---

# Epic 16 — Labels, Favoriten und Kategorien

## Story 16.1 — Labels in Aufgaben bearbeiten

**Ziel:**
Aufgaben können projektübergreifend organisiert werden.

**Eingangszustand:**
Labels sind nur als Datenmodell vorhanden.

**Ausgangszustand:**
Labels können in UI und VTODO geändert werden.

**Akzeptanzkriterien:**

* Nutzer kann Labels an einer Aufgabe setzen und entfernen.
* Neue Labels werden automatisch lokal angelegt.
* Labels werden als VTODO `CATEGORIES` geschrieben.
* Labeländerung ist Undo-fähig.
* Labeländerung prüft `expected_version`.
* Suche und Filter berücksichtigen aktualisierte Labels.

---

## Story 16.2 — Favoriten über STARRED

**Ziel:**
Favoriten sind lokal sichtbar und CalDAV-kompatibel synchronisiert.

**Eingangszustand:**
Es gibt keine Favoritenfunktion.

**Ausgangszustand:**
Favorit entspricht Kategorie `STARRED`.

**Akzeptanzkriterien:**

* `STARRED` aus CalDAV wird als Favorit importiert.
* Favorit setzen schreibt `STARRED` in VTODO-Categories.
* Favorit entfernen entfernt nur die Favoritenbedeutung.
* Andere Kategorien bleiben erhalten.
* Favoritenansicht zeigt favorisierte aktive Aufgaben.
* Favoritenstatus ist per Optimistic Locking geschützt.

---

# Epic 17 — Unteraufgaben

## Story 17.1 — Unteraufgaben importieren und anzeigen

**Ziel:**
Eine Ebene Unteraufgaben wird aus CalDAV sichtbar.

**Eingangszustand:**
Parent-Referenzen werden nicht ausgewertet.

**Ausgangszustand:**
Direkte Unteraufgaben erscheinen eingerückt unter Elternaufgaben.

**Akzeptanzkriterien:**

* `RELATED-TO;RELTYPE=PARENT` wird als Parent erkannt.
* `RELATED-TO` ohne RELTYPE wird Nextcloud-kompatibel als Parent interpretiert.
* Genau eine Ebene wird dargestellt.
* Tiefere Verschachtelungen werden als Wurzelaufgaben importiert.
* Raw-VTODO tieferer Aufgaben bleibt unverändert.
* Keine Warnung oder Badge für Tiefe 2+ ist erforderlich.

---

## Story 17.2 — Unteraufgabe erstellen

**Ziel:**
Der Nutzer kann direkte Unteraufgaben anlegen.

**Eingangszustand:**
Neue Tasks sind nur Wurzelaufgaben.

**Ausgangszustand:**
Eine Unteraufgabe ist lokal und in Nextcloud als solche sichtbar.

**Akzeptanzkriterien:**

* Unteraufgaben werden nur über „Unteraufgabe hinzufügen“ erstellt.
* Quick Add erstellt keine Unteraufgaben.
* Unteraufgaben erhalten Parent-Referenz im VTODO.
* Unteraufgaben können selbst keine Unteraufgaben haben.
* Entsprechende UI-Aktion ist deaktiviert.
* Erstellung schreibt sofort zu CalDAV.
* Nextcloud-Integrationstest bestätigt Sichtbarkeit.

---

## Story 17.3 — Elternaufgabe mit Unteraufgaben löschen

**Ziel:**
Löschen einer Elternaufgabe behandelt direkte Unteraufgaben explizit.

**Eingangszustand:**
Eine Elternaufgabe hat direkte Unteraufgaben.

**Ausgangszustand:**
Elternaufgabe und direkte Unteraufgaben sind nach Bestätigung gelöscht.

**Akzeptanzkriterien:**

* Löschdialog zeigt Anzahl direkter Unteraufgaben.
* Elternaufgabe und direkte Unteraufgaben werden gelöscht.
* Jede Task wird einzeln zu CalDAV gelöscht.
* Es gibt keinen Batch-Delete für einzelne Tasks.
* Undo-Snapshots werden für relevante Löschaktion erstellt.
* Fehler werden sichtbar und ohne stillen Datenverlust behandelt.

---

# Epic 18 — Wiederkehrende Aufgaben und VTODO-Erhalt

## Story 18.1 — Wiederholungseditor für MVP-Muster

**Ziel:**
Der Nutzer kann einfache Wiederholungen bearbeiten.

**Eingangszustand:**
RRULE wird nur importiert oder über Quick Add erzeugt.

**Ausgangszustand:**
MVP-Wiederholungsmuster sind in der UI bearbeitbar.

**Akzeptanzkriterien:**

* Täglich, wöchentlich, monatlich, jährlich sind bearbeitbar.
* Werktags ist bearbeitbar.
* Alle X Tage/Wochen/Monate sind bearbeitbar.
* Bestimmter Wochentag ist bearbeitbar.
* Ende nie, bis Datum und nach N Wiederholungen sind bearbeitbar.
* Änderung ersetzt RRULE explizit.
* Andere Feldänderungen verändern RRULE nicht.

---

## Story 18.2 — Komplexe RRULEs erhalten

**Ziel:**
Nicht unterstützte Wiederholungen gehen nicht verloren.

**Eingangszustand:**
Komplexe RRULEs könnten versehentlich überschrieben werden.

**Ausgangszustand:**
Komplexe RRULEs sind read-only sichtbar und bleiben erhalten.

**Akzeptanzkriterien:**

* Komplexe RRULEs werden erkannt.
* UI zeigt Badge „Komplexe Wiederholung – wird erhalten, kann nicht bearbeitet werden“.
* Wiederholungseditor ist deaktiviert.
* Andere Kernfelder bleiben bearbeitbar.
* Bearbeitung anderer Felder erhält RRULE unverändert.
* Erledigen verändert RRULE nicht.
* Es wird keine nächste Instanz lokal erzeugt.

---

## Story 18.3 — Anhänge und unbekannte Felder anzeigen/erhalten

**Ziel:**
Nicht aktiv unterstützte VTODO-Inhalte bleiben erhalten und teils sichtbar.

**Eingangszustand:**
Anhänge und unbekannte Properties sind nur Raw-Daten.

**Ausgangszustand:**
Anhänge werden read-only angezeigt, unbekannte Felder bleiben erhalten.

**Akzeptanzkriterien:**

* `ATTACH`-Properties bleiben bei Bearbeitung erhalten.
* Externe ATTACH-URLs werden als Links angezeigt.
* Externe Links öffnen mit `rel="noopener noreferrer"`.
* Inline-/Binary-Anhänge werden als vorhanden angezeigt, aber nicht gerendert.
* Keine Upload-, Entfernen- oder Bearbeiten-Funktion für Anhänge.
* Unbekannte Properties werden nicht entfernt.

---

# Epic 19 — Todoist-nahe UI, Navigation und Einstellungen

## Story 19.1 — Hauptnavigation

**Ziel:**
Die App bietet die MVP-Navigationsstruktur.

**Eingangszustand:**
Es gibt keine vollständige App-Navigation.

**Ausgangszustand:**
Alle Pflichtbereiche sind erreichbar.

**Akzeptanzkriterien:**

* Navigation enthält Heute.
* Navigation enthält Demnächst.
* Navigation enthält Projekte.
* Navigation enthält Labels.
* Navigation enthält Filter.
* Navigation enthält Favoriten.
* Navigation enthält Suche.
* Navigation enthält Konflikte.
* Navigation enthält Einstellungen.
* Aktive Ansicht ist erkennbar.

---

## Story 19.2 — Tastaturkürzel und Hilfe

**Ziel:**
Power-User können zentrale Aktionen per Tastatur ausführen.

**Eingangszustand:**
Alle Aktionen erfordern Mausnavigation.

**Ausgangszustand:**
Zentrale Tastaturkürzel und Hilfedialog sind vorhanden.

**Akzeptanzkriterien:**

* Neue Aufgabe ist per Shortcut erreichbar.
* Suche ist per Shortcut erreichbar.
* Ansichtenwechsel ist per Shortcut möglich.
* Tastaturhilfe ist verfügbar.
* Shortcuts kollidieren nicht mit aktiven Eingabefeldern.
* JavaScript bleibt CSP-kompatibel und lokal ausgeliefert.

---

## Story 19.3 — Laufende Writes sichtbar machen

**Ziel:**
Der Nutzer versteht, wann Änderungen noch nicht gespeichert sind.

**Eingangszustand:**
Writes könnten unbemerkt laufen oder abbrechen.

**Ausgangszustand:**
UI zeigt laufende und fehlgeschlagene Writes klar an.

**Akzeptanzkriterien:**

* Während eines Writes ist ein Speichern-/Pending-Zustand sichtbar.
* Bei erfolgreichem Write wird der gespeicherte Zustand angezeigt.
* Bei Fehler wird eine sichtbare Fehlermeldung angezeigt.
* Formularinhalte bleiben nach Möglichkeit erhalten.
* Beim Schließen/Navigieren mit laufendem Write wird `beforeunload` genutzt, soweit Browser es erlauben.
* Es gibt keine Browser-Offline-Queue.

---

## Story 19.4 — Einstellungen nach Setup

**Ziel:**
Konfiguration kann nach dem Erststart im Normalbetrieb geändert werden.

**Eingangszustand:**
Änderungen wären nur im Setup möglich.

**Ausgangszustand:**
Einstellungen decken CalDAV, Kalender, Sync, UI und Sicherheitsstatus ab.

**Akzeptanzkriterien:**

* CalDAV-URL, Benutzername und Passwort/App-Passwort können geändert werden.
* Speichern testet die Verbindung.
* Kalenderauswahl und Projektmapping können geändert werden.
* Default-Projekt kann geändert werden.
* Sync-Intervall kann geändert werden.
* Manueller Sync ist erreichbar.
* Erledigte Aufgaben anzeigen/ausblenden ist konfigurierbar.
* Demnächst-Zeitraum ist konfigurierbar.
* UI-Sprache Deutsch/Englisch ist konfigurierbar.
* Dark Mode ist konfigurierbar.
* Reverse-Proxy-Header- und HTTPS-Status werden angezeigt.

---

## Story 19.5 — Dark Mode und UI-Sprache

**Ziel:**
MVP-Anforderungen für Darstellung und Sprache sind erfüllt.

**Eingangszustand:**
UI hat nur eine feste Darstellung und Sprache.

**Ausgangszustand:**
Deutsch/Englisch und Dark Mode sind nutzbar.

**Akzeptanzkriterien:**

* UI-Sprache kann zwischen Deutsch und Englisch wechseln.
* UI-Sprache beeinflusst natürliche Eingabe.
* Dark Mode kann auf hell, dunkel oder System gesetzt werden.
* `system` folgt Browser-Präferenz.
* Weitere Themes sind nicht enthalten.
* UI-Texte sind nicht hart verstreut.

---

# Epic 20 — Deployment und Referenzbetrieb

## Story 20.1 — Go-Binary bauen

**Ziel:**
Caldo ist als einzelnes Go-Binary lieferbar.

**Eingangszustand:**
Es gibt kein baubares Release-Artefakt.

**Ausgangszustand:**
Ein Binary enthält Serverlogik, Templates und Migrationen.

**Akzeptanzkriterien:**

* Build erzeugt ein lauffähiges Caldo-Binary.
* Migrationen sind eingebettet.
* Templates sind generiert/eingebunden.
* Assets unter `web/static` werden separat bereitgestellt.
* Build-Reihenfolge folgt Templates, Tailwind, Go-Build.

---

## Story 20.2 — Docker-Image

**Ziel:**
Caldo kann als Container betrieben werden.

**Eingangszustand:**
Es gibt kein Container-Artefakt.

**Ausgangszustand:**
Ein Runtime-Image enthält Binary und statische Assets.

**Akzeptanzkriterien:**

* Multi-Stage-Build ist möglich.
* Runtime-Image enthält keine Go-Toolchain.
* Runtime läuft als Non-root-User.
* `/data` ist persistenter Datenpfad.
* Port `8080` ist einziger Listener.
* Healthcheck-fähiges Tool ist im Image vorhanden.

---

## Story 20.3 — Docker-Compose-Referenzdeployment

**Ziel:**
Self-Hoster können Caldo nachvollziehbar starten.

**Eingangszustand:**
Es gibt keine Referenzkonfiguration.

**Ausgangszustand:**
Docker Compose beschreibt Standardbetrieb hinter Reverse Proxy.

**Akzeptanzkriterien:**

* Compose nutzt Volume für `/data`.
* Pflicht-Environment-Variablen sind dokumentiert.
* Port wird lokal gebunden.
* Healthcheck ruft `/health` auf.
* Restart-Policy ist `on-failure:3`.
* `unless-stopped` wird nicht verwendet.
* Dokumentation erklärt, dass `BASE_URL` auch hinter internem HTTP-Proxy `https://` enthalten muss.

---
