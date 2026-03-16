# Caldo v1.0 — Architektur-Analyse und Umsetzungsplan

Dieses Dokument fasst die aktuellen Anforderungen zusammen und übersetzt sie in eine konkrete, umsetzbare Struktur für das Go-Projekt — **ohne Implementierungscode**.

## 1) Anforderungen verstanden (Kurz-Analyse)

### Produkt- und Architekturkern
- Caldo ist ein **selbst gehosteter Task-Manager** mit Fokus auf produktive, dichte Listenansicht (Toodledo-Stil).
- Das Backend ist **Thin-by-Design**:
  - CalDAV-Server (z. B. Nextcloud) ist **Single Source of Truth** für Aufgaben.
  - Keine eigene domänenspezifische Task-Datenbank in Caldo.
  - SQLite dient nur für App-spezifische Metadaten (Prefs, Credentials, Sync-Zustand).

### Technische Leitplanken
- Go-Backend + HTMX + Go Templates.
- CalDAV/WebDAV über `github.com/emersion/go-webdav`.
- Authentifizierung über Reverse-Proxy-Identität (`X-Forwarded-User`), keine eigene Nutzerverwaltung.
- VTODO-Felder für v1.0: `SUMMARY`, `DESCRIPTION`, `DTSTART`, `DUE`, `STATUS`, `PRIORITY`, `CATEGORIES`, `PERCENT-COMPLETE`, `VALARM`, `UID`, `ETAG`.
- Konfliktbehandlung über ETag.
- Delta-Sync bevorzugt via WebDAV-Sync (RFC 6578), mit Fallback-Strategie nötig.
- RRULE explizit nicht Teil von v1.0.

### Sicherheitsanforderung
- App-Passwörter verschlüsselt in SQLite (AES-256-GCM).
- Master-Key aus Env-Variable oder Secret-Datei.
- Keine geheimnisbehafteten Daten im Klartext loggen.

---

## 2) Vorgeschlagene Go-Projektstruktur

```text
caldo/
├─ cmd/
│  └─ caldo/
│     └─ main.go
├─ internal/
│  ├─ app/
│  │  ├─ app.go
│  │  ├─ config.go
│  │  └─ routes.go
│  ├─ http/
│  │  ├─ middleware/
│  │  │  ├─ request_id.go
│  │  │  ├─ logging.go
│  │  │  ├─ recover.go
│  │  │  ├─ security_headers.go
│  │  │  └─ proxy_auth.go
│  │  ├─ handlers/
│  │  │  ├─ page_tasks.go
│  │  │  ├─ htmx_tasks_list.go
│  │  │  ├─ htmx_task_row.go
│  │  │  ├─ htmx_sidebar_lists.go
│  │  │  ├─ api_task_create.go
│  │  │  ├─ api_task_update.go
│  │  │  ├─ api_task_delete.go
│  │  │  ├─ settings_dav_account.go
│  │  │  └─ health.go
│  │  ├─ dto/
│  │  │  ├─ task_form.go
│  │  │  └─ settings_form.go
│  │  └─ render/
│  │     ├─ templates.go
│  │     └─ viewmodel.go
│  ├─ caldav/
│  │  ├─ client.go
│  │  ├─ discovery.go
│  │  ├─ collections.go
│  │  ├─ tasks_repo.go
│  │  ├─ sync.go
│  │  ├─ etag.go
│  │  ├─ map_ical.go
│  │  ├─ alarm.go
│  │  └─ errors.go
│  ├─ domain/
│  │  ├─ task.go
│  │  ├─ filter.go
│  │  ├─ principal.go
│  │  └─ list.go
│  ├─ service/
│  │  ├─ task_service.go
│  │  ├─ sync_service.go
│  │  ├─ settings_service.go
│  │  └─ preferences_service.go
│  ├─ store/
│  │  ├─ sqlite/
│  │  │  ├─ db.go
│  │  │  ├─ migrations/
│  │  │  │  ├─ 0001_init.sql
│  │  │  │  └─ 0002_indexes.sql
│  │  │  ├─ principals_repo.go
│  │  │  ├─ preferences_repo.go
│  │  │  ├─ dav_accounts_repo.go
│  │  │  └─ sync_state_repo.go
│  │  └─ tx.go
│  ├─ security/
│  │  ├─ crypto.go
│  │  ├─ key_provider.go
│  │  └─ redact.go
│  ├─ jobs/
│  │  ├─ scheduler.go
│  │  └─ sync_job.go
│  └─ observability/
│     ├─ logger.go
│     └─ metrics.go
├─ web/
│  ├─ templates/
│  │  ├─ layout.gohtml
│  │  ├─ pages/tasks.gohtml
│  │  ├─ partials/task_row.gohtml
│  │  ├─ partials/task_table.gohtml
│  │  ├─ partials/sidebar_lists.gohtml
│  │  └─ partials/flash.gohtml
│  ├─ static/
│  │  ├─ css/app.css
│  │  └─ js/app.js
│  └─ htmx/
│     └─ attributes.md
├─ configs/
│  ├─ config.example.yaml
│  └─ docker/
│     └─ config.docker.yaml
├─ deployments/
│  ├─ Dockerfile
│  └─ docker-compose.yml
├─ docs/
│  ├─ architecture-plan-v1.md
│  ├─ caldav-interop-notes.md
│  └─ adr/
│     ├─ 0001-thin-backend-boundary.md
│     ├─ 0002-sync-strategy.md
│     └─ 0003-timezone-and-date-model.md
├─ test/
│  ├─ integration/
│  │  ├─ nextcloud/
│  │  └─ radicale/
│  └─ fixtures/
│     └─ ical/
├─ go.mod
├─ go.sum
└─ README.md
```

### Package-Grenzen (wichtig)
- `internal/caldav`: Nur Protokoll/Client-seitige DAV-Operationen + iCalendar-Mapping.
- `internal/domain`: Stabiler interner Typensatz (`Task`, `List`, `Principal`), ohne Infrastrukturabhängigkeit.
- `internal/service`: Use-Case-Schicht (Validierung, Konflikt-Logik, Orchestrierung).
- `internal/store/sqlite`: Ausschließlich Metadatenpersistenz (keine Taskdaten).
- `internal/http`: Handler + HTMX-Responses + Template-Rendering.

Damit bleibt die zentrale Trennung erhalten: **Task-Inhalt kommt immer aus CalDAV**.

---

## 3) Architekturentscheidungen für v1.0 (festgelegt)

Die folgenden Punkte sind auf Basis der Review-Kommentare konkret entschieden.

1. **Robuste Auto-Discovery (Nextcloud v33) + sinnvolle Fallbacks**
   - Primärpfad:
     1) Start von konfigurierter Server-URL.
     2) `PROPFIND` auf `current-user-principal`.
     3) `PROPFIND` auf `calendar-home-set`.
     4) Kalender-Collections mit `resourcetype` + `supported-calendar-component-set` filtern (`VTODO`).
   - Fallbacks:
     - Wenn `current-user-principal` fehlt: Well-known-Redirects (`/.well-known/caldav`) folgen.
     - Wenn `calendar-home-set` fehlt: heuristische Suche unter typischen Nextcloud-Pfaden (nur innerhalb Host-Basis-URL).
     - Wenn Komponentenset nicht geliefert wird: read-only Probe (`REPORT calendar-query`) und nur Collections mit VTODO-Treffern freischalten.
   - Failure-Mode: Discovery-Fehler mit klarer Ursache + „Server-URL prüfen / App-Passwort prüfen“ statt generischer Fehlermeldung.

2. **Pull-only Background Sync vs Request-driven Lazy Sync (Vor-/Nachteile)**
   - Pull-only Background Sync
     - Vorteile: konsistentes UI, weniger Wartezeit pro User-Interaktion, gute Basis für Multi-Tab.
     - Nachteile: Dauerlast auf DAV-Server, auch bei Inaktivität; komplexerer Scheduler.
   - Request-driven Lazy Sync
     - Vorteile: sehr ressourcenschonend, einfache Implementierung am Anfang.
     - Nachteile: erste UI-Interaktion oft langsam/stale, inkonsistent bei parallelen Sessions.
   - **Entscheidung v1.0**: Hybrid.
     - Kurzer Background-Intervall (z. B. 60–120s) + „sync on demand“ bei kritischen Aktionen (Seitenaufruf nach Idle, manuelles Refresh, nach 412-Konflikt).

3. **Konfliktstrategie bei ETag-Mismatch (412)**
   - **Entscheidung v1.0**: Standard ist „Reload & Retry“ mit Nutzerhinweis; zusätzlich optionale **Overwrite**-Aktion im Konfliktdialog.
   - „Merge“ auf Feldebene wird **nicht** in v1.0 umgesetzt (zu fehleranfällig ohne vollständige Änderungs-Historie).
   - UX:
     - Inline-Hinweis: „Aufgabe wurde extern geändert“.
     - Buttons: „Neu laden“ (default) und „Meine Änderung überschreiben“ (explizit).

4. **Zeitmodell pro Task konfigurierbar (DATE vs DATE-TIME)**
   - **Entscheidung v1.0**: pro Task expliziter Modus `due_kind = date | datetime`.
   - Mapping:
     - `date` ⇒ `DUE;VALUE=DATE:YYYYMMDD`.
     - `datetime` ⇒ `DUE:...` mit TZ/UTC gemäß Server-/User-Preference.
   - UI: Toggle „Ganztägig (ohne Uhrzeit)“ je Task.

5. **`STATUS` / `PERCENT-COMPLETE` Semantik**
   - **Entscheidung v1.0**:
     - Erlaubte Statuswerte: `NEEDS-ACTION`, `IN-PROCESS`, `COMPLETED`, `CANCELLED`.
     - UI-Shortcut „Done“ setzt `STATUS=COMPLETED` und `PERCENT-COMPLETE=100`.
     - Bei Reopen: `STATUS=NEEDS-ACTION`, `PERCENT-COMPLETE` auf letzten manuellen Wert oder 0.
   - Validierung: Prozent nur 0–100, serverseitig geklemmt/validiert.

6. **Reminder-Modell (VALARM)**
   - **Entscheidung v1.0**: genau **ein** Reminder pro Task.
   - Unterstützt:
     - relative Trigger (z. B. `TRIGGER:-PT15M`),
     - absolute Trigger (Datum/Zeit).
   - Nicht unterstützt in v1.0: mehrere parallele Alarme; bei Import mit mehreren Alarmen wird der erste unterstützte übernommen, Rest verworfen (mit Log-Hinweis).

7. **default_list-Verhalten**
   - **Entscheidung v1.0**: fixer initialer Name, falls Nutzer nichts setzt: `Tasks`.
   - Falls `Tasks` nicht existiert:
     - erste verfügbare VTODO-Collection als temporärer Fallback,
     - Hinweis in Settings, bis Nutzer explizit eine Standardliste gewählt hat.

8. **Credential-Lifecycle**
   - **Entscheidung v1.0**: Validierung **beim Speichern und beim ersten Sync** (beides).
   - Fehlerbehandlung:
     - Auth-Fehler markieren Account als „reconnect required“.
     - Bestehende UI bleibt lesbar mit letztem bekannten Zustand, aber Schreibaktionen gesperrt bis Re-Auth.

9. **Proxy-Header-Fehlerfälle (`X-Forwarded-User`)**
   - **Entscheidung v1.0**:
     - fehlend ⇒ `401` mit klarer Meldung („Authentifizierungsheader fehlt“),
     - mehrfach/mehrdeutig ⇒ `400` mit klarer Meldung („Mehrdeutiger Benutzerheader“).
   - Logging mit Request-ID, aber ohne sensible Header-Dumps.

10. **Fehlertoleranz gegenüber Server-Besonderheiten (Vorschlag)**
    - **Entscheidung v1.0**: „liberal in reading, strict in writing“.
      - Lesen: toleranter Parser für unbekannte Properties/Parameter, diese als Raw-Block erhalten wenn möglich.
      - Schreiben: nur definierte v1.0-Felder konsistent und RFC-konform ausgeben.
    - Ziel: hohe Interop, ohne instabile Spezialfälle in der UI freizuschalten.

11. **Sync-State-Granularität**
    - **Entscheidung v1.0**: genau ein `sync_token` **pro Collection**.
    - Begründung: unterschiedliche Änderungsraten/Capabilities pro Liste sauber abbildbar.

12. **Observability-MVP (Vorschlag)**
    - **Entscheidung v1.0**: minimale Pflichtmetriken + strukturierte Logs.
    - Pflichtmetriken:
      - Sync-Latenz (p50/p95),
      - Anzahl synchronisierter Elemente (created/updated/deleted),
      - Fehlerquote pro Fehlerklasse (auth, netzwerk, 412, parse),
      - Discovery-Erfolgsrate,
      - Conflict-Rate (`412`).
    - Logs:
      - korrelierbar per Request-ID/Principal-ID,
      - redacted secrets,
      - klare Fehlercodes für UI-Mapping.

---

## 4) Konkrete technische Risiken (go-webdav + Nextcloud)

1. **RFC-Interpretationsunterschiede zwischen Servern**
   - Nextcloud, Radicale, Baikal verhalten sich bei PROPFIND/REPORT-Details teils unterschiedlich.
   - Risiko: Features laufen auf Nextcloud, aber brechen bei sekundären Zielen.

2. **WebDAV-Sync nicht überall oder inkonsistent**
   - Falls `sync-collection` fehlt/abweicht, braucht es robusten Fallback.
   - Risiko: hohes Netzlastprofil bei großen Listen im Fallback-Modus.

3. **ETag-Semantik/Quoting**
   - Unterschiede bei starken/schwachen ETags oder Quoting können If-Match-Updates brechen.

4. **iCalendar-Feldvarianten**
   - `DUE`/`DTSTART` als `DATE` vs `DATE-TIME`, optionale TZIDs.
   - Risiko: falsche Sortierung, Off-by-one-Tage, inkonsistente Darstellung.

5. **VALARM-Komplexität**
   - Manche Clients erzeugen mehrere oder exotische Alarme.
   - Risiko: Datenverlust bei zu engem Mapping, wenn nur Teilmenge unterstützt wird.

6. **Unicode/Zeilenfaltung in ICS**
   - RFC-konforme line folding/unfolding und Encoding müssen korrekt sein.
   - Risiko: beschädigte Descriptions/Tags bei Roundtrip.

7. **Fehleroberfläche von go-webdav**
   - Bibliothek liefert evtl. generische Fehler; erschwert präzise UI-Fehlermeldungen und Retry-Strategien.

8. **Race Conditions bei parallelem Inline-Editing**
   - HTMX erzeugt potenziell viele kleine PATCH-ähnliche Writes.
   - Risiko: häufige 412-Konflikte, „flackernde“ UI.

9. **Nextcloud-spezifische Eigenschaften**
   - Zusätzliche Properties oder Erwartungen können Mapping beeinflussen (z. B. Displayname/Color auf Collection-Ebene).

10. **Performance bei großen Collections**
    - Initiale Vollsynchronisierung über REPORT kann teuer sein.
    - Risiko: lange Ladezeiten, Timeouts, schlechter Ersteindruck.

11. **Credential-/Transportfehler**
    - Abgelaufene App-Passwörter, TLS/Cert-Probleme.
    - Risiko: schwer unterscheidbare Auth- vs. Netzfehler.

12. **Delete-Semantik und Tombstones**
    - Bei Delta-Sync müssen Löschungen zuverlässig erkannt werden.
    - Risiko: „Zombie-Tasks“ in UI, wenn Tombstones falsch verarbeitet werden.

---

## 5) Empfohlene Implementierungsreihenfolge

### Phase 0 — Projektfundament
1. Basisprojekt (`cmd`, `internal`, `web`) + Konfigurationsloader.
2. SQLite-Anbindung + Migrationen für `principals`, `preferences`, `dav_accounts`, `sync_state`.
3. Security-Bausteine: Key-Loading (env/file), AES-256-GCM Encrypt/Decrypt, Log-Redaction.

### Phase 1 — Auth- und Settings-MVP
4. Reverse-Proxy-Auth-Middleware (`X-Forwarded-User` zwingend).
5. Settings-UI + Speicherung DAV-Account (verschlüsselt).
6. Connectivity-Check (Credential-Test gegen DAV-Server).

### Phase 2 — CalDAV-Leseweg (Read First)
7. Discovery: Principal + Home Set + Task-Collections.
8. Task-Listing (REPORT/PROPFIND) inkl. Mapping auf Domain-`Task`.
9. HTMX-Listenseite (Sidebar + Tabelle, sortier-/filterbar basic).

### Phase 3 — CRUD mit Konfliktsicherheit
10. Create Task (VTODO schreiben).
11. Update Task (If-Match mit ETag, 412-Handling im UI).
12. Delete Task (If-Match/sauberes Fehlerhandling).

### Phase 4 — Sync und Robustheit
13. Sync-Service mit WebDAV-Sync (wenn verfügbar).
14. ETag-basierter Fallback-Sync (wenn kein sync-token).
15. Hintergrund-Job + manuelles „Jetzt synchronisieren“.

### Phase 5 — UX-Feinschliff + Härtung
16. Inline-Editing für Priorität, Due Date, Tags, Status, Notes.
17. Verbesserte Fehlertexte (Auth, TLS, 412, Server-unreachable).
18. Interop-Tests gegen Nextcloud (Pflicht), Radicale/Baikal (Best Effort).
19. Container-Härtung + docker-compose Produktivprofil.

### Phase 6 — Release-Vorbereitung v1.0
20. Dokumentation (Admin-Guide, Proxy-Beispiele, Secret-Handling).
21. Monitoring-Basics (Sync-Erfolg/Fehler, Latenz).
22. Lizenz-/Compliance-Check (AGPL-Hinweise, Third-Party Notices).

---

## Empfohlene Definition of Done für v1.0
- Task-CRUD funktioniert stabil gegen Nextcloud.
- ETag-Konflikte sind korrekt und verständlich im UI sichtbar.
- Sync läuft mit WebDAV-Sync oder sauberem Fallback.
- Keine unverschlüsselten DAV-Credentials in DB/Logs.
- Reverse-Proxy-Auth ist verpflichtend und sicher dokumentiert.
