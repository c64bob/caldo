# Caldo Architektur (`arch.md`)

**Status:** Entwurf  
**Produkt:** Caldo  
**Dokumenttyp:** Technisches Architekturdokument  
**Zielpfad:** `docs/arch.md`  
**Quelle:** PRD `docs/prd.md`  
**Geltungsbereich:** MVP / Version 1

---

## 1. Zweck und Architekturprinzipien

Caldo ist eine selbst gehostete Single-User-Web-App für Todo-Management mit CalDAV/VTODO als führender Datenquelle. Dieses Dokument beschreibt die technische Architektur, die aus dem PRD und den bestätigten Architekturentscheidungen abgeleitet ist.

Die Architektur folgt diesen Prinzipien:

1. **CalDAV ist führend.** Lokale Änderungen gelten fachlich erst nach erfolgreichem CalDAV-Write als gespeichert.
2. **Keine stillen Datenverluste.** Unbekannte VTODO-Felder, komplexe RRULEs und Konfliktversionen werden erhalten, solange dies technisch möglich ist.
3. **Serverseitiges Rendering zuerst.** Die UI wird serverseitig gerendert und nur gezielt mit JavaScript ergänzt.
4. **Single-User bewusst einfach halten.** Kein Rollenmodell, keine Mandantenfähigkeit, kein verteilter Betrieb.
5. **SQLite bewusst robust betreiben.** WAL, ein Schreibpfad, automatische Migrationen und Backups vor Migrationen.
6. **Task-Inhalte bleiben privat.** Task-Titel, Beschreibungen und Raw-VTODO-Inhalte werden niemals geloggt.
7. **Implementierungsentscheidungen sind explizit.** Dieses Dokument beschreibt Invarianten, die bei der Implementierung nicht stillschweigend geändert werden dürfen.

---

## 2. Technologiestack

### 2.1 Backend

| Bereich | Entscheidung |
|---|---|
| Sprache | Go |
| HTTP-Router | Chi |
| Templates | Templ |
| Datenbank | SQLite |
| SQLite-Treiber | `modernc.org/sqlite` bevorzugt |
| Logging | `log/slog` |
| CalDAV/WebDAV | `emersion/go-webdav` |
| iCalendar Parsing | `emersion/go-ical` plus eigene VTODO-Roundtrip-Schicht |
| Migrationen | eigenes eingebettetes Migrationssystem |
| Scheduler | Goroutine im Go-Prozess |

Nicht verwendet:

- Echo
- Gin
- goose
- golang-migrate
- zap
- zerolog
- Redis
- externer Job-Runner
- Cron
- Browser-Sync-Polling als primärer Scheduler

### 2.2 Frontend

| Bereich | Entscheidung |
|---|---|
| Rendering | serverseitig |
| Server-driven UI-Updates | HTMX |
| Lokale UI-Zustände | Alpine.js |
| Global Keyboard Shortcuts | Vanilla JS |
| `beforeunload` bei laufenden Writes | Vanilla JS |
| CSS | Tailwind CSS |
| Asset-Build | nur Tailwind CSS |
| Laufzeit-CDN | nicht erlaubt |

Nicht verwendet:

- Vite
- esbuild
- Webpack
- React/Vue als App-Framework
- CDN zur Laufzeit

---

## 3. Laufzeit- und Prozessmodell

Caldo läuft als einzelner Go-Prozess.

### 3.1 Single-Process-Invariante

Caldo ist für genau einen aktiven Prozess pro Datenverzeichnis ausgelegt.

Ein OS-Advisory-Startup-Lock verhindert parallele Prozesse:

- Lock-Datei: `<dbPath>.startup.lock`
- Lock wird vor Migrationen erworben.
- Lock bleibt bis zum Prozessende gehalten.
- Ein zweiter Prozess bricht beim Start ab.

Es gibt keinen distributed Lock, keinen Cluster-Betrieb und keine Multi-Instance-Architektur.

### 3.2 Graceful Shutdown

Bei `SIGTERM` oder `SIGINT`:

1. HTTP-Server nimmt keine neuen Requests mehr an.
2. Scheduler stoppt neue Jobs.
3. Laufender Sync darf bis zu 30 Sekunden abschließen.
4. CalDAV-Operationen verwenden Contexts, damit Timeouts respektiert werden.
5. Prozess beendet sich kontrolliert.

---

## 4. Konfiguration und Startvalidierung

Die Serverkonfiguration erfolgt über Environment-Variablen.

Pflichtvariablen:

- `BASE_URL`
- `ENCRYPTION_KEY`
- `PROXY_USER_HEADER`

Optionale Variablen:

- `LOG_LEVEL`, Default `info`
- weitere Laufzeitparameter, soweit später dokumentiert

### 4.1 HTTPS-Prüfung

Caldo prüft HTTPS ausschließlich über `BASE_URL`.

Regel:

- `BASE_URL` muss mit `https://` beginnen.
- Die App prüft nicht, ob sie selbst TLS terminiert.
- Interner HTTP-Traffic zwischen Reverse Proxy und Caldo ist erlaubt.
- Bei ungültigem `BASE_URL` startet die App nicht.

### 4.2 Startabbruch

Harter Startabbruch erfolgt bei:

- fehlendem `BASE_URL`
- `BASE_URL` ohne `https://`
- fehlendem `PROXY_USER_HEADER`
- fehlendem oder formal ungültigem `ENCRYPTION_KEY`
- Migrationsfehler
- Checksum-Abweichung bereits angewendeter Migrationen
- nicht erwerbbarem Startup-Lock

---

### 4.3 Kanonische Startup-Sequenz

`cmd/caldo/main.go` führt den Prozessstart strikt in dieser Reihenfolge aus. Diese Reihenfolge ist eine Architektur-Invariante.

1. **Environment-Variablen laden und validieren (`config.Load`).**
   - `BASE_URL` fehlt oder beginnt nicht mit `https://` → `os.Exit(1)`.
   - `ENCRYPTION_KEY` fehlt, ist kein gültiges Base64 oder decodiert nicht auf exakt 32 Bytes → `os.Exit(1)`.
   - `PROXY_USER_HEADER` fehlt → `os.Exit(1)`.
2. **Startup-Lock erwerben (`syscall.Flock`).**
   - Lock nicht erwerbbar → `os.Exit(1)`.
   - Der Lock bleibt bis zum Prozessende gehalten.
3. **SQLite öffnen und PRAGMAs setzen.**
4. **Migrationssystem ausführen.**
   - Checksum-Abweichung → `os.Exit(1)`.
   - Migrationsfehler → `os.Exit(1)`.
   - Backup immer vor der ersten ausstehenden Migration.
5. **Scheduler initialisieren, aber noch nicht starten.**
6. **Setup-Status prüfen.**
   - `settings.setup_complete == false`: HTTP-Server ausschließlich mit Setup-Wizard-Routen und `GET /health` starten; normaler Betrieb ist blockiert.
   - `settings.setup_complete == true`: weiter mit Schritt 7.
7. **CalDAV-Credentials laden und entschlüsseln.**
   - Entschlüsselung fehlgeschlagen: App startet, CalDAV ist nicht verfügbar, UI zeigt Fehler; kein `os.Exit(1)`.
8. **Scheduler starten.**
9. **HTTP-Server mit allen normalen Routen starten.**
10. **Signal-Handler registrieren.**
    - `SIGTERM` und `SIGINT` lösen Graceful Shutdown aus.

Schritt 6 ist ein harter Gate: Vor abgeschlossenem Setup darf kein normaler HTTP-Traffic verarbeitet werden. Setup-Wizard und Normalbetrieb teilen dieselbe SQLite-Datenbank, verwenden aber unterschiedliche Route-Sets.

### 4.4 Setup-Wizard-Architektur

Der Setup-Wizard ist der einmalige Erststart-Ablauf für frische Installationen. Sein Abschluss wird dauerhaft mit `settings.setup_complete = true` persistiert. Der Wizard ist ein eigener Betriebsmodus des HTTP-Servers, nicht nur eine UI-Seite.

#### 4.4.1 Persistierter Wizard-Zustand

Der Wizard-Fortschritt wird serverseitig in `settings` gespeichert, nicht im Browser. Dadurch überlebt der Setup-Zustand Reloads, Browser-Neustarts und Prozess-Restarts.

```text
settings.setup_complete: boolean
settings.setup_step: 'caldav' | 'calendars' | 'import' | 'complete'
```

Bedeutung:

| `setup_step` | Bedeutung |
|---|---|
| `caldav` | Schritt 1: CalDAV-URL und Credentials erfassen und testen |
| `calendars` | Schritt 2: verfügbare Kalender laden, Auswahl und Default-Projekt festlegen |
| `import` | Schritt 3: Initialimport ausführen |
| `complete` | Setup abgeschlossen; `setup_complete` wird auf `true` gesetzt |

`setup_complete=true` ist der maßgebliche Gate-Wert. `setup_step='complete'` allein reicht nicht aus.

#### 4.4.2 Routing im Setup-Modus

Wenn `settings.setup_complete == false`, registriert der HTTP-Server ausschließlich folgende Routen:

| Methode | Pfad | Zweck |
|---|---|---|
| `GET` | `/setup` | Wizard-Einstieg und Rendern des aktuellen Setup-Schritts |
| `POST` | `/setup/caldav` | CalDAV-URL und Credentials testen |
| `GET` | `/setup/calendars` | verfügbare Kalender nach erfolgreichem Connect laden |
| `POST` | `/setup/calendars` | Kalenderauswahl und Default-Projekt speichern |
| `POST` | `/setup/import` | Initialimport starten; Fortschritt per SSE |
| `POST` | `/setup/complete` | Setup abschließen und `setup_complete=true` setzen |
| `GET` | `/health` | Liveness-Healthcheck; immer erreichbar |

Alle anderen Routen antworten im Setup-Modus mit:

```text
302 Location: /setup
```

Die Setup-Routen laufen durch die Sicherheitsmiddleware, soweit anwendbar: Proxy-Auth ist erforderlich, CSRF schützt mutierende Setup-Routen, und `GET /health` bleibt als Liveness-Endpunkt erreichbar.

#### 4.4.3 CalDAV-Test im Setup

`POST /setup/caldav` führt einen echten CalDAV-Test aus:

1. CalDAV-URL, Benutzername und Passwort/App-Passwort entgegennehmen.
2. Credentials sofort mit AES-256-GCM verschlüsseln und in SQLite speichern.
3. Credentials nicht im Browser, nicht in Cookies und nicht in serverseitigem Session-State halten.
4. `PROPFIND` gegen den CalDAV-Server ausführen.
5. Server-Capability erkennen und speichern:
   - WebDAV-Sync,
   - CTag/ETag-Fallback,
   - Full-Scan-Fallback.
6. Bei Erfolg `settings.setup_step = 'calendars'` setzen.
7. Bei Fehlschlag im Schritt `caldav` bleiben und Fehler ohne Secrets anzeigen.

#### 4.4.4 Kalenderauswahl und Default-Projekt

`GET /setup/calendars` lädt die verfügbaren Kalender über die gespeicherten, entschlüsselten CalDAV-Credentials.

`POST /setup/calendars` speichert:

- ausgewählte Kalender,
- lokales Projekt-Mapping,
- Default-Kalender beziehungsweise Default-Projekt,
- optional ein neu angelegtes Default-Projekt.

Ohne Default-Projekt darf der Wizard nicht in den Import-Schritt wechseln. Nach erfolgreicher Kalenderauswahl gilt:

```text
settings.setup_step = 'import'
```

#### 4.4.5 Initialimport

Der Initialimport läuft über `POST /setup/import`.

Architektonisch ist er ein Full-Scan-Sync über alle ausgewählten Kalender, aber mit abweichender Konfliktsemantik:

- Es gibt noch keine lokalen Benutzeränderungen.
- Es gibt keine lokalen Base-Versionen mit abweichender Local-Version.
- Importierte VTODOs werden als `synced` übernommen.
- Es wird keine Konfliktbehandlung ausgeführt.
- `raw_vtodo`, `base_vtodo`, normalisierte Kernfelder, `etag`, Kalenderzuordnung und FTS5-Index werden aufgebaut.

Der Import verwendet dieselben CalDAV-, VTODO- und DB-Komponenten wie der normale Full-Scan-Fallback, aber einen expliziten `initial_import`-Modus in der Sync-Schicht. Fortschritt wird über SSE gemeldet. Im Setup-Modus dürfen SSE-Events nur Setup-/Import-Fortschritt transportieren, keine normalen Task-/Sync-Events.

#### 4.4.6 Setup-Abschluss

Nach erfolgreichem Initialimport führt `POST /setup/complete` aus:

1. Prüfen, dass Kalenderauswahl, Default-Projekt und Initialimport erfolgreich abgeschlossen wurden.
2. `settings.setup_step = 'complete'` setzen.
3. `settings.setup_complete = true` setzen.
4. Transaktion committen.
5. Scheduler starten oder den initialisierten Scheduler aktivieren.
6. Auf die normale App-UI weiterleiten.

Ab diesem Zeitpunkt wird beim nächsten Start die normale Startup-Sequenz ab Schritt 7 fortgesetzt und der HTTP-Server mit allen normalen Routen gestartet.

#### 4.4.7 Setup-Invarianten

1. Wizard-Zustand liegt serverseitig in `settings`, nicht im Browser.
2. `setup_complete=false` blockiert normalen App-Betrieb hart.
3. Im Setup-Modus werden nur Setup-Routen und `GET /health` registriert.
4. Alle anderen Routen leiten mit `302` nach `/setup`.
5. Credentials werden nach `POST /setup/caldav` sofort verschlüsselt gespeichert.
6. `POST /setup/caldav` erkennt und speichert die CalDAV-Capability.
7. Initialimport ist ein Full-Scan ohne Konfliktbehandlung.
8. Importierte VTODOs werden als `synced` übernommen.
9. Scheduler startet erst nach `setup_complete=true`.
10. Setup-Wizard und Normalbetrieb teilen dieselbe SQLite-Datenbank, aber nicht dasselbe Route-Set.


---
## 5. Authentifizierung, Sessions und CSRF

### 5.1 Reverse-Proxy-Authentifizierung

Caldo hat kein lokales Login.

Regeln:

- Der Headername kommt aus `PROXY_USER_HEADER`.
- Requests ohne gültigen Proxy-Auth-Header werden mit `403 Forbidden` abgelehnt.
- Es gibt keinen Redirect auf Login.
- Es gibt keinen lokalen Notfall-Login.
- Es gibt keine Rollen.

Der Headerwert darf niemals geloggt werden.

### 5.2 Session-Cookie

Caldo setzt ein Session-Cookie für UI-Kontinuität, Undo-Zuordnung und Tab-Verhalten.

Cookie:

| Name | Eigenschaften |
|---|---|
| `session_id` | `HttpOnly`, `Secure`, `SameSite=Strict`, Session-Cookie |

`session_id` ist nicht die eigentliche Authentifizierung. Die Authentifizierung erfolgt weiterhin ausschließlich über den Reverse-Proxy-Header.

### 5.3 Tab-ID

Jeder Browser-Tab bekommt eine eigene `tab_id`.

Regeln:

- `tab_id` wird im Browser mit `crypto.randomUUID()` erzeugt.
- Speicherung in `sessionStorage`.
- Nicht in `localStorage`.
- Überlebt Reloads im selben Tab.
- Ein neuer Tab bekommt eine neue `tab_id`.
- Alle HTMX-Requests senden `X-Tab-ID`.

Die Undo-Identität ist immer:

```text
(session_id, tab_id)
```

### 5.4 CSRF-Schutz

CSRF-Middleware schützt alle mutierenden Methoden:

- `POST`
- `PUT`
- `PATCH`
- `DELETE`

Verfahren:

- Double-Submit-Cookie
- HMAC-validiert
- Token wird in Cookie und Request übermittelt

Cookies:

| Cookie | Eigenschaften |
|---|---|
| `session_id` | `HttpOnly`, `Secure`, `SameSite=Strict` |
| `csrf_token` | nicht `HttpOnly`, `Secure`, `SameSite=Strict` |

HTMX-Requests senden:

- `X-CSRF-Token`
- `X-Tab-ID`

Fehlender oder ungültiger CSRF-Token ergibt `403 Forbidden`.

---

## 6. SQLite-Betrieb

### 6.1 PRAGMAs und Connection Pool

SQLite wird mit folgenden Einstellungen betrieben:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
```

Go-SQL-Pool:

```text
MaxOpenConns = 1
```

Begründung:

- WAL verbessert Read/Write-Nebenläufigkeit.
- `NORMAL` balanciert Performance und Crash-Sicherheit.
- `busy_timeout=5000` verhindert sofortige Fehler bei kurzer Lock-Contention.
- `MaxOpenConns=1` reduziert SQLite-Locking-Komplexität.

### 6.2 Globaler Write-Mutex

Alle schreibenden DB-Operationen laufen über einen globalen Write-Mutex.

Gilt für:

- HTTP-Write-Handler
- Sync-Importe
- CalDAV-Write-Statusupdates
- Cleanup-Jobs
- Reindex-Operationen
- Projekt-/Labeländerungen
- Undo-Ausführung
- Konfliktauflösung

### 6.3 Transaktionsregeln

Invarianten:

1. Keine nested Transactions.
2. Der Write-Mutex darf nicht nested gehalten werden.
3. Schreibfunktionen nehmen entweder eine bestehende Transaktion entgegen oder besitzen selbst Mutex und Transaktion.
4. Lange Syncs halten den Mutex nicht dauerhaft.
5. Remote-Fetching und Parsing passieren außerhalb des Mutex.
6. DB-Mutationen während Sync erfolgen in Chunks.

---

## 7. Migrationssystem

Caldo verwendet ein eigenes eingebettetes Migrationssystem.

### 7.1 Migrationstabelle

```sql
CREATE TABLE IF NOT EXISTS schema_migrations (
  version     INTEGER PRIMARY KEY,
  name        TEXT NOT NULL,
  applied_at  DATETIME NOT NULL,
  checksum    TEXT NOT NULL
);
```

`checksum` ist SHA-256 über das Migration-SQL.

### 7.2 Datei- und Embed-Struktur

```text
internal/migrations/
  migrations.go
  001_initial.sql
  002_add_conflicts.sql
  003_add_undo_snapshots.sql
  ...
```

Migrationen werden in das Binary eingebettet. Es gibt kein externes Migrationsverzeichnis zur Laufzeit.

### 7.3 Ablauf beim App-Start

1. Startup-Lock erwerben.
2. SQLite öffnen.
3. `schema_migrations` sicherstellen.
4. Angewendete Migrationen laden.
5. Checksums validieren.
6. Ausstehende Migrationen bestimmen.
7. Wenn keine ausstehenden Migrationen existieren: normal starten.
8. Wenn ausstehende Migrationen existieren: Backup erstellen.
9. Ausstehende Migrationen der Reihe nach anwenden.
10. Bei Fehler: strukturierter Logeintrag und `os.Exit(1)`.

### 7.4 Backup vor Migration

Backup-Regel:

- immer vor der ersten ausstehenden Migration
- nie nach Start einer Migration
- im selben Verzeichnis wie die SQLite-Datei
- kein separates Backup-Verzeichnis

Namensschema:

```text
caldo_backup_pre_migration_<version>_<timestamp>.db
```

`<version>` ist die höchste ausstehende Migration.

### 7.5 Migrations-Invarianten

1. Eine Migration = eine Transaktion.
2. DDL und DML werden nie in derselben Migration gemischt.
3. Backup immer vor der ersten ausstehenden Migration.
4. Bereits angewendete Migrationen dürfen nicht nachträglich verändert werden.
5. Checksum-Abweichung führt zum Startabbruch.
6. Fehler führen zu `os.Exit(1)`, nicht zu `panic`.
7. Das Docker-Compose-Referenzdeployment verwendet `restart: on-failure:3`, nicht `unless-stopped`.

---

## 8. Verschlüsselung von CalDAV-Credentials

### 8.1 Key-Format

`ENCRYPTION_KEY` ist:

- Base64-kodiert
- exakt 32 Bytes nach Decoding
- direkter AES-256-Schlüssel
- keine Passphrase
- keine KDF

Ungültige Formen führen zum Startabbruch.

### 8.2 Algorithmus

Caldo verwendet:

```text
AES-256-GCM
```

Jeder verschlüsselte Wert enthält:

- Formatversion
- zufällige Nonce
- Ciphertext inklusive Auth-Tag

Bevorzugtes Speicherformat:

```text
v1:<base64_nonce>:<base64_ciphertext>
```

### 8.3 Key-Rotation

Key-Rotation ist nicht Bestandteil des MVP.

### 8.4 Falscher, aber formal gültiger Key

Wenn der Key formal gültig ist, aber vorhandene Credentials nicht entschlüsselt werden können:

- App startet.
- CalDAV-Verbindung ist nicht verfügbar.
- UI zeigt eine klare Fehlermeldung.
- Logs enthalten nur einen sicheren Fehlertyp.
- Nutzer kann CalDAV-Credentials neu eingeben.

Es erfolgt kein harter Startabbruch.

---

## 9. Datenmodell

Dieses Kapitel beschreibt die wichtigsten Tabellen und Konzepte. Feldlisten sind implementierungsleitend, aber nicht als vollständige finale Migration zu verstehen.

### 9.1 Tasks

Wichtige Task-Felder:

```sql
CREATE TABLE tasks (
  id                 TEXT PRIMARY KEY,
  project_id         TEXT NOT NULL,
  uid                TEXT NOT NULL,
  href               TEXT,
  etag               TEXT,
  server_version     INTEGER NOT NULL DEFAULT 1,

  title              TEXT NOT NULL,
  description        TEXT,
  status             TEXT NOT NULL,
  completed_at       DATETIME,
  due_date           DATE,
  due_at             DATETIME,
  priority           INTEGER,
  rrule              TEXT,

  parent_id          TEXT,
  raw_vtodo          TEXT NOT NULL,
  base_vtodo         TEXT,

  label_names        TEXT,
  project_name       TEXT,

  sync_status        TEXT NOT NULL,
  created_at         DATETIME NOT NULL,
  updated_at         DATETIME NOT NULL,

  FOREIGN KEY(project_id) REFERENCES projects(id),
  FOREIGN KEY(parent_id) REFERENCES tasks(id)
);
```

Hinweise:

- `raw_vtodo` enthält die aktuelle lokale VTODO-Rohrepräsentation.
- `base_vtodo` ist der letzte bekannte gemeinsame Zustand vor lokaler Änderung.
- `label_names` und `project_name` sind denormalisierte Suchfelder.
- `etag` ist Remote-Zustand vom CalDAV-Server.
- `server_version` ist lokaler Caldo-Zustand für Optimistic Locking.

### 9.2 Projects

```sql
CREATE TABLE projects (
  id              TEXT PRIMARY KEY,
  calendar_href   TEXT NOT NULL,
  display_name    TEXT NOT NULL,
  ctag            TEXT,
  sync_token      TEXT,
  server_version  INTEGER NOT NULL DEFAULT 1,
  is_default      BOOLEAN NOT NULL DEFAULT FALSE,
  created_at      DATETIME NOT NULL,
  updated_at      DATETIME NOT NULL
);
```

### 9.3 Labels

```sql
CREATE TABLE labels (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL UNIQUE,
  created_at  DATETIME NOT NULL
);

CREATE TABLE task_labels (
  task_id   TEXT NOT NULL,
  label_id  TEXT NOT NULL,
  PRIMARY KEY (task_id, label_id)
);
```

### 9.4 Settings

Die `settings`-Tabelle enthält lokale App-Einstellungen:

- Sync-Intervall
- Default-Projekt
- UI-Sprache
- Demnächst-Zeitraum
- erledigte Aufgaben anzeigen
- Dark Mode
- verschlüsselte CalDAV-Credentials
- CalDAV-URL und Benutzername, soweit nicht als Secret behandelt
- Setup-Status: `setup_complete`, `setup_step`
- gespeicherte CalDAV-Capabilities für Sync-Strategiewahl

### 9.5 Undo-Snapshots

```sql
CREATE TABLE undo_snapshots (
  id                TEXT PRIMARY KEY,
  session_id        TEXT NOT NULL,
  tab_id            TEXT NOT NULL,
  task_id           TEXT NOT NULL,
  action_type       TEXT NOT NULL,
  snapshot_vtodo    TEXT NOT NULL,
  snapshot_fields   TEXT NOT NULL,
  etag_at_snapshot  TEXT,
  created_at        DATETIME NOT NULL,
  expires_at        DATETIME NOT NULL,

  UNIQUE(session_id, tab_id)
);

CREATE INDEX idx_undo_session_tab ON undo_snapshots(session_id, tab_id);
CREATE INDEX idx_undo_expires ON undo_snapshots(expires_at);
```

### 9.6 Conflicts

```sql
CREATE TABLE conflicts (
  id              TEXT PRIMARY KEY,
  task_id         TEXT,
  project_id      TEXT REFERENCES projects(id),
  conflict_type   TEXT NOT NULL,
  created_at      DATETIME NOT NULL,
  resolved_at     DATETIME,
  resolution      TEXT,

  base_vtodo      TEXT,
  local_vtodo     TEXT,
  remote_vtodo    TEXT,
  resolved_vtodo  TEXT
);
```

`local_vtodo` und `remote_vtodo` sind nullable, weil Löschkonflikte jeweils eine Seite ohne VTODO enthalten können.

---

## 10. VTODO-Roundtrip-Architektur

### 10.1 Grundsatz

Caldo verwendet `emersion/go-ical` zum Lesen und Extrahieren bekannter Felder, aber nicht als alleinige Schreibschicht.

Grund:

- Beim reinen Serialisieren über Parser-Bibliotheken können unbekannte Properties verloren gehen.
- Das PRD verlangt Erhalt unbekannter VTODO-Felder.

### 10.2 Eigene VTODO-Schicht

Caldo implementiert eine eigene VTODO-Roundtrip-Schicht.

Aufgaben:

1. `raw_vtodo` unverändert speichern.
2. bekannte Felder extrahieren:
   - Titel
   - Beschreibung
   - Fälligkeit
   - Startdatum
   - Status
   - Completed
   - Prozent abgeschlossen
   - Priorität
   - Kategorien
   - RRULE
   - RELATED-TO
3. normalisierte Felder aktualisieren.
4. Beim Schreiben nur explizit geänderte bekannte Felder patchen.
5. Unbekannte Properties unverändert erhalten.

### 10.3 Patcher-Invarianten

1. Der Patcher darf unbekannte Properties nicht löschen.
2. Der Patcher darf RRULE nur ändern, wenn Wiederholung explizit bearbeitet wurde.
3. Der Patcher darf Erinnerungen und Anhänge nicht entfernen.
4. Der Patcher muss Raw-VTODO als Roundtrip-Quelle behandeln.
5. Der Patcher muss Tests für unbekannte Properties, VALARMs, ATTACH und RRULE-Erhalt haben.

---

## 11. CalDAV- und Sync-Architektur

### 11.1 CalDAV-Ziel

Caldo ist CalDAV-serverneutral, mit Nextcloud als primärem Testfall.

Pflicht:

- Integrationstest gegen echten Nextcloud-CalDAV-Endpunkt.
- Keine Nextcloud-spezifische harte Kopplung in Kernlogik.

### 11.2 Sync-Stufen

Sync arbeitet in drei Stufen:

1. **Primär:** WebDAV Sync
2. **Sekundär:** CTag + ETag-Vergleich
3. **Tertiär:** Full-Scan

### 11.3 Sync-Trigger

| Trigger | Verhalten |
|---|---|
| Sofort-nach-Write | synchron im HTTP-Handler, nur betroffene Ressource |
| Manuell | Full-Sync, HTTP-Response wartet nicht, Abschluss per SSE |
| Periodisch | Full-Sync aller Kalender durch Scheduler |

### 11.4 Sync-Locking

Die `SyncEngine` besitzt einen Mutex:

- verhindert parallele Full-Sync-Läufe
- keine Sync-Queue
- Trigger während laufendem Sync melden `already running` oder kehren ohne neuen Lauf zurück

Sofort-nach-Write für eine einzelne Ressource ist der höchste Prioritätsfall und läuft synchron, damit die UI korrekt anzeigen kann, ob die Änderung fachlich gespeichert ist.

### 11.5 Write-Through-Statusmodell

Für Task-Änderungen gilt:

1. HTTP-Request enthält `expected_version`.
2. Server prüft Optimistic Locking.
3. Vorheriger Zustand wird ggf. als Undo-Snapshot gespeichert.
4. Lokale Änderung wird als `pending` versioniert.
5. CalDAV-Write wird synchron ausgeführt.
6. Bei Erfolg:
   - neuer ETag wird gespeichert
   - `sync_status = synced`
   - `server_version` wird erneut erhöht
   - Response zeigt gespeicherten Zustand
   - SSE wird nach DB-Commit gesendet
7. Bei Fehler:
   - UI zeigt Fehler
   - `sync_status = error`
   - kein zusätzliches `server_version`-Increment für den Fehlerstatus
   - Änderung gilt nicht als fachlich gespeichert

### 11.6 Remote-Sync

Remote-Änderungen werden in Chunks verarbeitet:

1. Remote-Fetch außerhalb des Write-Mutex.
2. Parsing außerhalb des Write-Mutex.
3. DB-Mutations-Chunk mit Write-Mutex und Transaktion.
4. `server_version` wird bei importierten Änderungen erhöht.
5. `etag`, `ctag` und/oder `sync_token` werden aktualisiert.
6. SSE-Events werden nach Commit gesendet.

---

## 12. Scheduler

### 12.1 Grundsatz

Der periodische Sync läuft serverseitig im Go-Prozess, unabhängig von offenen Browser-Tabs.

### 12.2 Intervall

- Default: 15 Minuten
- Wert kommt aus `settings`
- Änderung des Intervalls startet Scheduler/Ticker kontrolliert neu

### 12.3 Cleanup-Jobs

Bei jedem Sync-Lauf:

```sql
DELETE FROM undo_snapshots WHERE expires_at < CURRENT_TIMESTAMP;
```

Täglich:

- gelöste Konflikte älter als 7 Tage löschen
- ungelöste Konflikte niemals automatisch löschen

### 12.4 Scheduler-Invarianten

1. Kein Browser-Polling als Sync-Scheduler.
2. Kein externer Job-Runner.
3. Kein Cron.
4. Kein Redis.
5. Alle Jobs laufen im einen Caldo-Prozess.
6. Startup-Lock verhindert parallele Scheduler-Prozesse.

---

## 13. Optimistic Locking, SSE und Fokus-Refresh

### 13.1 `server_version` vs. `etag`

`server_version` und `etag` sind strikt getrennt.

| Konzept | Bedeutung | Quelle |
|---|---|---|
| `server_version` | lokaler Caldo-Zustand für Optimistic Locking, SSE und Fokus-Refresh | Caldo |
| `etag` | Remote-Zustand der CalDAV-Ressource | CalDAV-Server |

Regeln:

| Ereignis | `server_version` | `etag` |
|---|---:|---|
| Nutzer ändert Aufgabe lokal | +1 | unverändert |
| CalDAV-Write erfolgreich | +1 | neuer Wert |
| CalDAV-Write fehlgeschlagen | unverändert | unverändert |
| Remote-Sync holt neue Version | +1 | neuer Wert |

### 13.2 Schreibende Requests

Jeder schreibende Request muss `expected_version` enthalten.

Regel:

```text
expected_version == current server_version  → Änderung darf verarbeitet werden
expected_version != current server_version  → keine Änderung, Outdated-/Conflict-Response
```

Es gibt keine Ausnahmen.

### 13.3 SSE

Ein globaler SSE-Endpunkt reicht für Single-User-Betrieb.

Jede SSE-Verbindung bekommt eine `connection_id`.

Event-Felder:

| Feld | Bedeutung |
|---|---|
| `type` | `task_updated`, `task_deleted`, `project_updated`, `sync_complete`, `conflict_created` |
| `resource_id` | Task- oder Projekt-ID |
| `version` | neue lokale Version |
| `origin_connection_id` | auslösende SSE-Verbindung |

Broadcast-Regel:

- Events gehen an alle offenen Connections außer die auslösende.
- Die auslösende Connection erhält ihr Ergebnis synchron über die HTTP-Response.
- SSE-Broadcast erfolgt immer nach erfolgreichem DB-Commit.

### 13.4 Verhalten anderer Tabs

Wenn ein anderer Tab ein Event erhält:

- kein offenes Formular für Ressource: Fragment per HTMX aktualisieren
- offenes Formular ohne lokale Änderungen: Fragment darf aktualisiert werden
- offenes Formular mit lokalen ungespeicherten Änderungen: nicht überschreiben, Hinweis anzeigen
- Submit mit veralteter Version: Server lehnt Änderung ab

### 13.5 Fokus-Refresh

Beim Zurückkehren in einen Tab:

```text
GET /api/tasks/versions?ids=...
```

Der Tab vergleicht bekannte Versionen mit Server-Versionen und lädt nur veraltete Fragmente per HTMX neu.

### 13.6 Invarianten

1. `server_version` wird nur im DB-Write-Pfad inkrementiert.
2. Jeder mutierende Request enthält `expected_version`.
3. SSE-Broadcast erfolgt nach DB-Commit.
4. `etag` wird niemals als UI-Version verwendet.
5. `server_version` wird niemals als CalDAV-ETag verwendet.

---

## 14. Undo

### 14.1 Umfang

Undo unterstützt die letzte Undo-fähige Aktion pro Tab:

- Aufgabe erledigen
- Aufgabe bearbeiten
- Aufgabe löschen
- Projektwechsel
- Labeländerung

Gültigkeit:

- maximal 5 Minuten
- oder bis zur nächsten Undo-fähigen Aktion
- Reload im selben Tab erhält Undo
- neuer Tab hat eigenes Undo

### 14.2 Snapshot-Erstellung

Snapshot und Änderung liegen in derselben DB-Transaktion.

Ablauf:

1. `(session_id, tab_id)` bestimmen.
2. Task-Zustand vor Änderung lesen.
3. Undo-Snapshot per UPSERT speichern.
4. Änderung ausführen.
5. `server_version` erhöhen.
6. `sync_status = pending`.
7. Transaktion committen.
8. CalDAV-Write synchron ausführen.

### 14.3 Snapshot-Inhalt

Ein Undo-Snapshot enthält:

- `raw_vtodo` vor der Änderung
- normalisierte Felder vor der Änderung
- `etag_at_snapshot`
- Aktionstyp
- Ablaufzeit

`etag_at_snapshot` ist erforderlich, um Remote-Änderungen zwischen Änderung und Undo zu erkennen.

### 14.4 Undo-Ausführung

Ablauf:

1. Snapshot für `(session_id, tab_id)` laden.
2. Ablaufzeit prüfen.
3. Aktuelle Task laden.
4. Wenn `etag` seit Snapshot abweicht: Konflikt erzeugen.
5. Sonst Zielzustand aus Snapshot herstellen.
6. Zustand als `pending` speichern.
7. CalDAV-Write synchron ausführen.
8. Nur bei erfolgreichem CalDAV-Write:
   - neuen ETag speichern
   - `sync_status = synced`
   - Snapshot löschen
   - SSE senden

Bei fehlgeschlagenem CalDAV-Write:

- Fehler anzeigen
- Snapshot bleibt erhalten, sofern nicht abgelaufen
- Undo gilt nicht als abgeschlossen

### 14.5 Invarianten

1. Snapshot und ursprüngliche Änderung sind immer in derselben DB-Transaktion.
2. `UNIQUE(session_id, tab_id)` erzwingt genau einen Snapshot pro Tab.
3. `etag_at_snapshot` wird gespeichert.
4. Snapshot wird erst nach erfolgreichem Undo-CalDAV-Write gelöscht.

---

## 15. Konfliktmodell

### 15.1 Grundsatz

Konflikte sind eigene DB-Entitäten, kein reines Statusfeld auf Tasks.

`tasks.sync_status = conflict` ist nur ein operativer Zustand. Die Konfliktversionen und der Lebenszyklus liegen in `conflicts`.

### 15.2 Drei-Wege-Modell

Ein Konflikt kann enthalten:

- `base_vtodo`: letzter bekannter gemeinsamer Zustand
- `local_vtodo`: lokale Version
- `remote_vtodo`: Remote-Version
- `resolved_vtodo`: Ergebnis der Auflösung

`base_vtodo` wird explizit gespeichert. Es ist der `raw_vtodo` zum Zeitpunkt des letzten erfolgreichen Syncs beziehungsweise der Zustand vor lokaler Änderung.

### 15.3 Fehlende Base

`base_vtodo` kann in Ausnahmefällen `NULL` sein.

Regel:

- Wenn `base_vtodo IS NULL`, ist Auto-Merge deaktiviert.
- Auch scheinbar triviale Unterschiede werden nicht automatisch gemerged.
- Manuelle Konfliktauflösung ist erforderlich.

### 15.4 Auto-Merge

Auto-Merge arbeitet feldbasiert auf geparsten Kernfeldern, nicht auf Raw-Text-Diffs.

Für jedes bekannte Feld:

```text
local == base   → remote übernehmen
remote == base  → local behalten
local == remote → Wert übernehmen
sonst           → echter Feldkonflikt
```

Wenn alle Felder konfliktfrei auflösbar sind:

- Auto-Merge
- kein `conflicts`-Eintrag

Wenn mindestens ein Feld konfliktbehaftet ist:

- `conflicts`-Eintrag erstellen
- `tasks.sync_status = conflict`

### 15.5 Labels / CATEGORIES

Labels sind Sets.

Auto-Merge darf Sets vereinigen, wenn keine Entfernung widersprüchlich ist.

Echter Konflikt entsteht, wenn eine Seite ein Label entfernt und die andere Seite dasselbe Label hinzufügt oder in widersprüchlicher Weise verändert.

### 15.6 Löschkonflikte

#### `edit_delete`

Lokal geändert, remote gelöscht:

- `task_id` zeigt auf lokale Task
- `local_vtodo` vorhanden
- `remote_vtodo = NULL`
- Optionen:
  - lokale Version zu CalDAV schreiben
  - lokal löschen

#### `delete_edit`

Lokal gelöscht, remote geändert:

- `task_id = NULL`
- `local_vtodo = NULL`
- `remote_vtodo` vorhanden
- Optionen:
  - remote Version lokal importieren
  - remote endgültig löschen

### 15.7 Beide Versionen behalten

„Beide behalten“ bedeutet:

1. Remote-Version wird als neue eigenständige Task mit neuer UID in CalDAV geschrieben.
2. Lokale Version behält ihre UID und wird normal zu CalDAV geschrieben.
3. Beide Tasks landen im selben Projekt.
4. Es gibt keine Parent-Referenz oder technische Verbindung zwischen beiden.
5. Konflikt erhält `resolution = split`.

### 15.8 Lebenszyklus

```text
created → visible in UI → resolved → cleanup nach 7 Tagen
```

Regeln:

- ungelöste Konflikte werden nie automatisch gelöscht
- gelöste Konflikte werden nach 7 Tagen gelöscht
- Konfliktversionen werden nur für Konflikte gespeichert

---

## 16. Projekte und CalDAV-Kalender

### 16.1 Mapping

Ein Projekt entspricht einem CalDAV-Kalender.

### 16.2 Write-Through

Projektoperationen sind Write-Through-Operationen gegen CalDAV.

| Operation | Remote-Operation | Lokaler Update |
|---|---|---|
| Projekt anlegen | Kalender anlegen | erst nach Erfolg |
| Projekt umbenennen | Kalender umbenennen | erst nach Erfolg |
| Projekt löschen | Kalender löschen | erst nach Erfolg |

Es gibt kein optimistisches UI-Update bei Projektoperationen.

### 16.3 Projekt löschen

Löschen eines Projekts:

- löscht den CalDAV-Kalender
- sendet keine einzelnen Task-DELETEs
- entfernt danach lokales Projekt
- entfernt lokale Tasks des Projekts
- entfernt FTS5-Einträge

Bestätigungsdialog zeigt:

- Projektname
- Anzahl betroffener Tasks
- starke Bestätigung, z. B. Eingabe des Projektnamens

### 16.4 Remote gelöschte Kalender

Remote gelöschte Kalender sind autoritativ.

Regeln:

- lokales Cleanup
- kein Konflikt
- lokales Projekt verschwindet
- zugehörige lokale Tasks verschwinden
- FTS5-Einträge werden gelöscht

Dies ist eine bewusste Architekturentscheidung: Remote-Kalenderlöschung wird nicht als Projektkonflikt behandelt.

### 16.5 Projektumbenennung und Suche

Bei Projektumbenennung:

- CalDAV-Kalendername wird zuerst geändert.
- Nach Erfolg wird lokales Projekt aktualisiert.
- `project_name` in denormalisierten Task-Suchfeldern wird aktualisiert.
- FTS5-Index wird synchron neu indiziert.

---

## 17. Unteraufgaben

### 17.1 Parent-Referenz

Caldo schreibt Parent-Beziehungen als:

```text
RELATED-TO;RELTYPE=PARENT:<uid>
```

Beim Lesen wird auch `RELATED-TO` ohne `RELTYPE` als Parent interpretiert, um Nextcloud-kompatibel zu sein.

### 17.2 Unterstützte Tiefe

Caldo stellt genau eine Ebene Unteraufgaben dar.

| Tiefe | Verhalten |
|---:|---|
| 0 | Wurzelaufgabe |
| 1 | Unteraufgabe |
| 2+ | als Wurzelaufgabe importieren |

Für Tiefe 2+ gilt:

- `parent_id = NULL`
- Anzeige als eigenständige Aufgabe
- `raw_vtodo` bleibt unverändert
- `RELATED-TO` bleibt im Rohtext erhalten
- keine Warnung, kein Badge

### 17.3 UI

- Unteraufgaben werden eingerückt angezeigt.
- Unteraufgaben werden nur über „Unteraufgabe hinzufügen“ erstellt.
- Keine Unteraufgabenerstellung über Quick Add.
- Unteraufgaben können selbst keine Unteraufgaben haben.
- Die UI-Aktion ist bei Unteraufgaben deaktiviert.

### 17.4 Löschen einer Elternaufgabe

Beim Löschen einer Elternaufgabe:

- Dialog zeigt Anzahl der Unteraufgaben.
- Elternaufgabe und direkte Unteraufgaben werden gelöscht.
- Jede Task wird einzeln zu CalDAV gelöscht.
- Es gibt keinen Batch-Delete für einzelne Tasks.

### 17.5 Integrationstests

Pflichttests gegen lokalen Nextcloud-Container:

1. Unteraufgabe in Caldo anlegen → in Nextcloud als Unteraufgabe sichtbar.
2. Unteraufgabe in Nextcloud anlegen → in Caldo als Unteraufgabe sichtbar.

---

## 18. Wiederkehrende Aufgaben

### 18.1 Grundsatz

Caldo erhält komplexe RRULEs, bearbeitet aber nur MVP-Muster.

### 18.2 Speicherung

- `tasks.rrule` speichert RRULE als Rohstring.
- Keine Normalisierung.
- Keine Zerlegung.
- Beim Write-Back wird RRULE unverändert in `raw_vtodo` eingesetzt.
- RRULE wird nur ersetzt, wenn der Nutzer Wiederholung explizit bearbeitet.

### 18.3 Erledigen wiederkehrender Aufgaben

Beim Erledigen:

- `STATUS:COMPLETED` setzen
- `COMPLETED:` setzen
- RRULE nicht ändern
- keine nächste Instanz lokal erzeugen
- nächste Instanz kommt vom CalDAV-Server beim nächsten Sync

Nicht im MVP:

- `THISANDFUTURE`
- `EXDATE`-Management
- lokale Folgeinstanz-Erzeugung
- komplexe Ausnahmebehandlung

### 18.4 Bearbeitbare Muster

Der Wiederholungs-Editor unterstützt nur MVP-Muster:

- täglich
- wöchentlich
- monatlich
- jährlich
- werktags
- alle X Tage
- alle X Wochen
- alle X Monate
- bestimmter Wochentag
- Ende nie
- Ende bis Datum
- Ende nach N Wiederholungen

### 18.5 Komplexe RRULEs

Komplexe RRULEs enthalten z. B.:

```text
BYDAY=1MO,3MO
BYSETPOS=-1
BYMONTHDAY=15,30
EXDATE=...
```

UI-Verhalten:

- Read-only-Badge: „Komplexe Wiederholung – wird erhalten, kann nicht bearbeitet werden“
- Wiederholungs-Editor deaktiviert
- andere Kernfelder bleiben bearbeitbar

### 18.6 Invarianten

1. RRULE niemals als Nebeneffekt anderer Feldänderungen modifizieren.
2. Erledigen lässt RRULE unangetastet.
3. Komplexe RRULEs werden erhalten und nicht bearbeitet.
4. Nächste Instanz wird nicht lokal erzeugt.
5. RRULE-Parsing dient nur Anzeige und Editor-Entscheidung.

---

## 19. Suche mit SQLite FTS5

### 19.1 Grundsatz

Globale Freitextsuche verwendet SQLite FTS5.

Kein `LIKE`-Fallback im MVP.

### 19.2 FTS5-Schema

```sql
CREATE VIRTUAL TABLE tasks_fts USING fts5(
  title,
  description,
  label_names,
  project_name,
  content=tasks,
  content_rowid=rowid,
  tokenize='unicode61 remove_diacritics 1'
);
```

`tasks` enthält die denormalisierten Spalten `label_names` und `project_name`, damit externe Content-Table-Nutzung konsistent bleibt.

### 19.3 Index-Pflege

- Trigger pflegen strukturelle Konsistenz bei INSERT, UPDATE, DELETE.
- Go-Layer setzt `label_names` und `project_name`.
- Label- und Projektumbenennungen lösen explizites Reindexing betroffener Tasks aus.
- Konfliktversionen, Undo-Snapshots und historische Versionen werden nie indiziert.

### 19.4 Suchverhalten

Freitext:

```text
tasks_fts MATCH ?
```

Mit Prefix-Syntax:

```text
rech → rech*
```

Projektfilter:

```text
#Finanzen → project_id = ?
```

Labelfilter:

```text
@wichtig → EXISTS/JOIN über task_labels
```

Kombination:

```text
rechnung #Finanzen → FTS5 MATCH rechnung* AND project_id = ?
```

### 19.5 Standardausschlüsse

Standardmäßig:

```sql
status != 'completed'
```

Ausnahme nur bei explizitem `completed:true`.

Nicht im MVP:

- Fuzzy-Suche
- Tippfehlertoleranz
- Relevanzranking als Produktfeature

### 19.6 Tests

Pflicht:

- FTS5-Integrationstest gegen echte SQLite-Test-DB.
- Umlaut-/Diakritiktest.
- Prefix-Test.
- erledigte Aufgaben erscheinen standardmäßig nicht.

---

## 20. Filter-Query-Engine

### 20.1 Architektur

Filter und globale Suche nutzen eine AST-basierte Query-Engine.

Schichten:

```text
Lexer → Parser → AST → SQL-Compiler
```

Package-Struktur:

```text
internal/query/
  lexer.go
  parser.go
  ast.go
  compiler.go
  query_test.go
```

### 20.2 AST

Node-Typen:

- `AndNode`
- `OrNode`
- `NotNode`
- `LeafNode`

`LeafNode` enthält Operator und Wert.

### 20.3 Operator-Priorität

Ohne Klammern gilt:

```text
NOT > AND > OR
```

Implementierung erfolgt über rekursiven Abstiegsparser.

### 20.4 Blattoperatoren

| Operator | Bedeutung |
|---|---|
| `today` | Fälligkeit heute |
| `overdue` | überfällig und nicht erledigt |
| `upcoming` | Fälligkeit zwischen heute und konfiguriertem Zeitraum |
| `#Projekt` | Projektfilter |
| `@Label` | Labelfilter |
| `priority:high` | hohe Priorität |
| `completed:false` | nicht erledigt |
| `text:foo` | FTS5-Suche |
| `before:date` | Fälligkeit vor Datum |
| `after:date` | Fälligkeit nach Datum |
| `no date` | keine Fälligkeit |

### 20.5 SQL-Compiler-Invarianten

1. Compiler erzeugt immer parametrisierte SQL-Fragmente plus Argumentliste.
2. Keine String-Interpolation mit User-Input.
3. Unbekannte Operatoren erzeugen Compile-Error.
4. Projekt- und Labelnamen werden vor Kompilierung gegen IDs aufgelöst.
5. Unbekannte Projekt-/Labelnamen ergeben leere Ergebnisse, keinen Fehler.
6. `upcoming` nutzt den konfigurierten Zeitraum, Default 7 Tage.

### 20.6 Globale Suche als Subset

Globale Suche nutzt dieselbe Pipeline, aber eine eingeschränkte Validierung.

Erlaubt:

- `text:`
- `#`
- `@`

Andere Filteroperatoren werden in der globalen Suche als Freitext behandelt.

### 20.7 Verhältnis zu Quick Add

Quick Add und Filter-Query haben unterschiedliche Zielmodelle, aber keine duplizierte Tokenlogik.

Gemeinsam genutzt werden:

- Token-Erkennung für `#`
- Token-Erkennung für `@`
- Prioritäts- und Datumserkennung, soweit passend
- Resolver-Interfaces für Projekte und Labels

Filter-Query erzeugt AST. Quick Add erzeugt einen Aufgabenentwurf. Beide dürfen nicht divergierende Regeln für gemeinsame Tokens implementieren.

---

## 21. Quick-Add-Parser

### 21.1 Architektur

Quick Add verwendet einen eigenen regelbasierten Parser ohne Pflichtabhängigkeit auf externe Libraries.

Package:

```text
internal/parser
```

Schichten:

```text
Tokenizer → Resolver
```

`olebedev/when` darf evaluiert werden, ist aber keine Pflichtabhängigkeit.

### 21.2 Tokenizer

Erkennt:

- Projekt: `#Name`
- Label: `@Name`
- Priorität: `!high`, `!medium`, `!low`, `!1`, `!2`, `!3`
- Datumsausdrücke
- Wiederholungsausdrücke
- freien Text

### 21.3 Resolver

Resolver wird als Interface injiziert und ruft nicht direkt DB-Code.

Aufgaben:

- Projekte auflösen
- unbekannte Projekte als Vorschlags-/Anlagefall markieren
- Labels auflösen
- unbekannte Labels als neu anzulegen markieren

### 21.4 Unterstützte MVP-Muster

Datum:

- `heute`
- `morgen`
- `übermorgen`
- `today`
- `tomorrow`

Relativ:

- `nächsten Montag`
- `next monday`
- `in 3 Tagen`
- `in 3 days`

Wochentage:

- `montag` bis `sonntag`
- `monday` bis `sunday`

Wiederholung:

- `jeden Montag`
- `every monday`
- `täglich`
- `daily`
- `wöchentlich`
- `weekly`
- `monatlich`
- `monthly`
- `jährlich`
- `yearly`
- `werktags`
- `weekdays`
- `alle X Tage`
- `alle X Wochen`
- `alle X Monate`

### 21.5 Mehrdeutigkeiten

Regeln:

- Datum vor Freitext.
- `montag` wird als Datum interpretiert.
- Bei `3.4` bevorzugt deutsche UI-Sprache deutsches Format.
- Sonst ISO-orientierte Interpretation.
- Unbekannte Tokens bleiben Teil des Titels.
- Unbekannte Tokens erzeugen keine Fehlermeldung.

### 21.6 UI-Vorschau

Quick Add zeigt live unter dem Eingabefeld:

- erkannter Titel
- Projekt
- Labels
- Datum
- Wiederholung
- Priorität

### 21.7 Nicht im MVP

- `jeden zweiten Dienstag im Monat`
- `jeden Montag um 9 Uhr`
- natürlichsprachliche Prioritäten wie `dringend`

### 21.8 Tests

- jedes MVP-Muster hat Unit-Tests
- Tests laufen ohne DB
- Tests laufen ohne HTTP
- Resolver wird gefaked

---

## 22. Projekt-, Label- und Favoriten-Mapping

### 22.1 Projekte

Projekt = CalDAV-Kalender.

- Projektanlage legt Kalender an.
- Projektumbenennung benennt Kalender um.
- Projektlöschung löscht Kalender.

### 22.2 Labels

Labels werden als VTODO `CATEGORIES` gespeichert.

- neue Labels werden automatisch angelegt
- Labels werden lokal normalisiert gespeichert
- VTODO-Categories bleiben maßgebliche Sync-Repräsentation

### 22.3 Favoriten

Favoriten werden über Kategorie `STARRED` modelliert.

Regeln:

- `STARRED` in CalDAV wird als Favorit importiert.
- Favorit in Caldo wird als `STARRED` geschrieben.
- `STARRED` ist eine reservierte Kategorie mit UI-Sonderbedeutung.

---

## 23. UI-Architektur

### 23.1 Rendering

- Seiten und Fragmente werden mit Templ serverseitig gerendert.
- HTMX lädt Fragmente für Interaktionen nach.
- Alpine.js hält lokale Zustände:
  - offene Formulare
  - unsaved state
  - outdated banner
  - kleine UI-Toggles
- Vanilla JS nur für:
  - globale Tastaturkürzel
  - `beforeunload` bei laufenden Writes
  - HTMX-Header-Konfiguration für `X-Tab-ID` und `X-CSRF-Token`

### 23.2 Kein Runtime-CDN

Alle Assets werden lokal ausgeliefert.

### 23.3 Laufende Writes

Die UI muss laufende Writes sichtbar machen.

Bei laufendem Write und Navigation/Tab-Schließen:

- `beforeunload`-Warnung, soweit Browser dies erlaubt
- keine Offline-Queue
- keine automatische Nachsendung beim nächsten Öffnen

### 23.4 Konflikt- und Outdated-UI

Bei veralteter Version:

- nicht still überschreiben
- Hinweis oder Konfliktansicht anzeigen
- Formular mit lokalen Änderungen nicht automatisch ersetzen

---

## 24. Logging und Datenschutz

### 24.1 Library und Formate

Caldo nutzt `log/slog`.

Formate:

- Production: JSON
- Development: Text

Log-Level über:

```text
LOG_LEVEL=info|debug|warn|error
```

Default: `info`.

### 24.2 Niemals loggen

Niemals, auch nicht auf Debug-Level:

- Task-Titel
- Task-Beschreibungen
- `raw_vtodo`
- CalDAV-Passwort
- App-Token
- `ENCRYPTION_KEY`
- `session_id`
- `csrf_token`
- Proxy-Auth-Header-Werte
- Query-Parameter, wenn sie Nutzdaten enthalten könnten

### 24.3 Erlaubte Logdaten

Erlaubt:

- Task-ID
- Project-ID
- ETag
- Sync-Status
- HTTP-Methode
- HTTP-Pfad ohne Query-Parameter
- Fehlertyp ohne nutzdatenhaltige Message
- Sync-Dauer
- Anzahl synchronisierter Tasks
- CalDAV-HTTP-Statuscodes
- technische Run-IDs

### 24.4 Zentrale Maskierung

Maskierung erfolgt zentral über einen `slog.Handler`-Wrapper.

Sensitive Keys liegen zentral als Konstante, nicht verstreut im Code.

Die Maskierung ist zusätzliche Absicherung. Sensible Werte sollen trotzdem gar nicht erst an Log-Aufrufstellen übergeben werden.

### 24.5 Correlation IDs

HTTP:

- jeder Request bekommt `request_id`
- Response-Header: `X-Request-ID`
- alle Request-Logs enthalten `request_id`

Sync:

- jeder Sync-Lauf bekommt `sync_run_id`
- alle Sync-Logs enthalten `sync_run_id`

---

## 25. Healthcheck

Der Healthcheck ist `GET /health` und prüft nur, ob die App läuft.

Er prüft nicht:

- CalDAV-Erreichbarkeit
- Sync-Fähigkeit
- Credentials
- vollständige Systemintegrität

Wenn Migrationen fehlschlagen, startet die Weboberfläche nicht; damit ist auch der Healthcheck nicht verfügbar.

---

## 26. Testing-Strategie

### 26.1 Unit-Tests

Pflichtbereiche:

- VTODO-Feldextraktion
- VTODO-Patching und Erhalt unbekannter Properties
- RRULE-Erkennung
- Quick-Add-Parser
- Query-Lexer
- Query-Parser
- Query-Compiler
- Konflikt-Auto-Merge
- Undo-Snapshot-Logik
- CSRF-Tokenvalidierung
- Secret-Verschlüsselung
- Log-Masking

### 26.2 SQLite-Integrationstests

Pflicht:

- Migrationen
- Checksum-Validierung
- FTS5-Suche
- Prefix-Suche
- Diakritikverhalten
- Default-Ausschluss erledigter Tasks
- Trigger/Reindex
- Undo-Snapshot-Constraints
- Konflikt-Lifecycle

### 26.3 Nextcloud-Integrationstests

Pflicht gegen lokalen Nextcloud-Container:

- CalDAV-Verbindungstest
- Kalenderimport
- Task erstellen/bearbeiten/erledigen/löschen
- Unteraufgabe Caldo → Nextcloud
- Unteraufgabe Nextcloud → Caldo
- `STARRED`-Kategorie
- ETag/CTag-Verhalten
- Wiederkehrende Aufgabe erledigen und Serververhalten beobachten

### 26.4 Kein HTTP/DB in Parser-Unit-Tests

Parser-Tests laufen ohne:

- HTTP
- CalDAV
- echte DB

Resolver werden gefaked.

---

## 27. Zentrale Architektur-Invarianten

Diese Invarianten dürfen in der Implementierung nicht verletzt werden.

### 27.1 Daten und Sync

1. CalDAV ist führend.
2. Lokale Änderungen gelten erst nach erfolgreichem CalDAV-Write als fachlich gespeichert.
3. `raw_vtodo` wird immer erhalten.
4. Unbekannte VTODO-Felder werden nicht gelöscht.
5. RRULE wird nur bei expliziter Wiederholungsänderung ersetzt.
6. `etag` und `server_version` werden niemals vermischt.
7. Remote gelöschte Kalender sind autoritativ und erzeugen keinen Konflikt.

### 27.2 Concurrency

1. Alle DB-Writes laufen über den globalen Write-Mutex.
2. Keine nested Transactions.
3. Jeder mutierende Request enthält `expected_version`.
4. Jeder mutierende HTMX-Request enthält `X-Tab-ID`.
5. SSE-Broadcast erfolgt nach DB-Commit.
6. Syncs werden nicht parallel ausgeführt.

### 27.3 Sicherheit

1. Kein lokaler Login.
2. Fehlender Proxy-Auth-Header ergibt 403.
3. CSRF schützt alle mutierenden Methoden.
4. `session_id` ist `HttpOnly`, `Secure`, `SameSite=Strict`.
5. `csrf_token` ist JS-lesbar, `Secure`, `SameSite=Strict`.
6. `ENCRYPTION_KEY` ist Base64-kodierter 32-Byte-Key.
7. AES-256-GCM ist der einzige Secret-Algorithmus im MVP.

### 27.4 Datenschutz

1. Task-Titel werden niemals geloggt.
2. Task-Beschreibungen werden niemals geloggt.
3. `raw_vtodo` wird niemals geloggt.
4. Credentials, Tokens und Session-Werte werden niemals geloggt.
5. Maskierung erfolgt zentral im Logging-Handler.

### 27.5 Migrationen

1. Migrationen laufen automatisch beim Start.
2. Backup vor erster ausstehender Migration.
3. Checksum-Abweichung führt zu Startabbruch.
4. DDL und DML werden nicht in einer Migration gemischt.
5. Fehler führen zu `os.Exit(1)`.

---

## 28. Bewusst nicht im MVP

Nicht Bestandteil dieser Architektur:

- Multi-User
- Rollenmodell
- lokaler Login
- PWA
- Browser-Offline-Queue
- lokale dauerhafte Write-Queue
- Kanban
- Projektarchivierung
- Papierkorb
- Produktivitätsstatistiken
- Key-Rotation
- distributed Scheduler
- Redis
- komplexe RRULE-Bearbeitung
- EXDATE-Management
- lokale Folgeinstanz-Erzeugung
- Fuzzy-Suche
- Relevanzranking als Produktfeature
- vollständige mobile Optimierung
