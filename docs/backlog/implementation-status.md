# Implementation Status Audit

Stand: 2026-06-08

Dieses Dokument sammelt die Evidenz zum Status der bestehenden Stories. Die Story-Dateien selbst enthalten nur den knappen Status, damit sie Planungsdokumente bleiben.

## Zusammenfassung

- Bestehende Stories: 54 `Umgesetzt`, 32 `Teilweise umgesetzt`, 3 `Offen`.
- Neue Planungsstories: 2 `Umgesetzt`, 37 `Offen`.
- Neue Epics: 23 UI-Grundsystem, 24 Aufgabenliste, 25 Schnellanlage, 26 Navigation/Projekte/Filter/Labels, 27 Konflikte/Einstellungen, 28 Responsive QA/Accessibility/Performance.

## Legende

- `Umgesetzt`: Akzeptanzkriterien sind im aktuellen Codepfad im Wesentlichen abgedeckt.
- `Teilweise umgesetzt`: wesentliche Backend-, Daten- oder UI-Teile existieren, aber mindestens ein fachlich sichtbarer Teil fehlt.
- `Offen`: fuer die Story wurde keine belastbare Umsetzung gefunden.

## Bestehende Stories

| Story | Status | Evidenzort |
|---|---|---|
| 1.0 | Umgesetzt | Repository, Makefile, Dockerfile, Compose-Dateien und CI-/Security-/Release-Workflows sind vorhanden; CI enthaelt Browser-QA. |
| 1.1 | Umgesetzt | `internal/config/config.go` validiert Runtime-Konfiguration inklusive Tests. |
| 1.2 | Umgesetzt | `internal/lock` und Startup-Wiring erzwingen den Single-Process-Lock. |
| 1.3 | Umgesetzt | `internal/db/sqlite.go` setzt SQLite-PRAGMAs inklusive WAL. |
| 1.4 | Umgesetzt | `internal/migrations` prueft Checksums, Backups und angewandte Migrationen. |
| 1.5 | Umgesetzt | `internal/handler/health.go` stellt den authfreien Health-Endpunkt bereit. |
| 1.6 | Umgesetzt | `internal/logging` und Request-Middleware nutzen `log/slog` ohne sensible Inhalte. |
| 1.7 | Umgesetzt | `internal/shutdown` und `cmd/caldo/main.go` behandeln Signale und geordnetes Beenden. |
| 2.1 | Umgesetzt | `internal/handler/router.go` und Middleware registrieren Chi-Routen, statische Assets und Request-Kontext. |
| 2.2 | Umgesetzt | Reverse-Proxy-Auth-Middleware liest den konfigurierten Header und laesst `/health` frei. |
| 2.3 | Teilweise umgesetzt | CSRF- und Mutationsschutz existieren; Tab-/Session-Verhalten ist noch nicht durchgaengig in der sichtbaren UI integriert. |
| 2.4 | Umgesetzt | Lokale Asset-Auslieferung, Manifest und CSP sind in Router/Layout/Asset-Tests abgedeckt. |
| 2.5 | Teilweise umgesetzt | Templ-Layout und lokale Assets existieren; CSRF-/Tab-Metadaten und Layout-Inhalte sind nicht durchgaengig fuer alle Normalrouten abgeschlossen. |
| 3.1 | Umgesetzt | `internal/db/settings.go` und Migrationen verwalten das Settings-Singleton. |
| 3.2 | Umgesetzt | Projekt-/Kalender-Tabellen und DB-Funktionen sind vorhanden und getestet. |
| 3.3 | Umgesetzt | Task-Tabelle und Repository-Funktionen decken lokale Aufgabenpersistenz ab. |
| 3.4 | Umgesetzt | Labels und `STARRED`-Favoriten sind im Modell und in DB-Constraints umgesetzt. |
| 3.5 | Umgesetzt | Undo- und Konflikttabellen existieren mit Tests fuer Zustandswechsel. |
| 4.1 | Umgesetzt | `internal/crypto/credentials.go` und DB-Credential-Funktionen speichern CalDAV-Zugangsdaten verschluesselt. |
| 4.2 | Umgesetzt | CalDAV-Verbindungstest ist in Client/Setup-Flows vorhanden und mit Fake/Staging-CalDAV abgedeckt. |
| 5.1 | Umgesetzt | Router-Gating trennt Setup- und Normalmodus; Router-E2E prueft Redirects. |
| 5.2 | Umgesetzt | Setup-Credential-Formular, Test und verschluesselte Speicherung sind umgesetzt. |
| 5.3 | Umgesetzt | Kalenderauswahl und Default-Kalender-Anlage sind im Setup-Fluss umgesetzt. |
| 5.4 | Umgesetzt | Initialer Import schreibt Projekte und Aufgaben; Router- und Browser-E2E pruefen den Flow. |
| 5.5 | Umgesetzt | Setup-Completion oeffnet Normalrouten und startet den Scheduler erst danach. |
| 6.1 | Umgesetzt | `internal/model/vtodo.go` extrahiert Kernfelder und bewahrt unbekannte VTODO-Daten. |
| 6.2 | Umgesetzt | Patch-Logik bewahrt unbekannte VTODO-Daten, VALARM, ATTACH und komplexe RRULEs. |
| 6.3 | Umgesetzt | `internal/caldav/todos.go` und Retry-Tests behandeln PUT/DELETE, 412 und 404 gemaess Architektur. |
| 7.1 | Umgesetzt | Task-Create schreibt CalDAV vor lokaler Persistenz; Router-E2E deckt den Flow ab. |
| 7.2 | Teilweise umgesetzt | Write-through-Edit-Route existiert; sichtbare, vollstaendige Bearbeitungs-UI bleibt UI-Restarbeit. |
| 7.3 | Teilweise umgesetzt | Complete/Reopen-Routen existieren; sichtbare UI-Aktionen und Fehlerzustaende sind noch auszubauen. |
| 7.4 | Teilweise umgesetzt | Delete-Write-through und 404-Erfolg sind umgesetzt; Loesch-UI/Undo-Fuehrung bleibt Restarbeit. |
| 7.5 | Teilweise umgesetzt | Backend behandelt offene Unteraufgaben mit expliziter Aktion; sichtbarer Entscheidungsdialog fehlt. |
| 8.1 | Teilweise umgesetzt | Projektanlage ist backendseitig vorhanden; produktive UI-Verwaltung bleibt offen. |
| 8.2 | Teilweise umgesetzt | Projektumbenennung ist backendseitig vorhanden; produktive UI-Verwaltung bleibt offen. |
| 8.3 | Teilweise umgesetzt | Projektloeschung ist backendseitig vorhanden; produktive UI-Verwaltung bleibt offen. |
| 8.4 | Umgesetzt | `internal/db/projects_remote_cleanup.go` und Full-Scan-Sync bereinigen remote geloeschte Kalender. |
| 9.1 | Umgesetzt | FTS-Suchindex und Tests existieren in `internal/db/tasks_search.go` und `tasks_fts_test.go`. |
| 9.2 | Umgesetzt | Suchroute und Ergebnisrendering sind in Handlern und Tests vorhanden. |
| 9.3 | Umgesetzt | Heute-, Demnaechst-, Ueberfaellig- und Ohne-Datum-Views sind implementiert. |
| 10.1 | Umgesetzt | `internal/sync/engine.go` waehlt Strategien und faellt auf Full-Scan zurueck. |
| 10.2 | Umgesetzt | Manual-Sync-Route ist mit echtem Runner verdrahtet und in Router-E2E abgedeckt. |
| 10.3 | Umgesetzt | Scheduler startet nur bei abgeschlossenem Setup und nutzt den Sync-Runner. |
| 10.4 | Umgesetzt | `internal/db/sync_cleanup.go` und Scheduler-Tests loeschen abgelaufene Undo-Snapshots und alte geloeste Konflikte. |
| 11.1 | Umgesetzt | Mutierende Task-/Projekt-/Filterpfade pruefen erwartete Versionen. |
| 11.2 | Umgesetzt | SSE-Broker und Events existieren fuer Normal- und Setup-Flows getrennt. |
| 11.3 | Teilweise umgesetzt | Staleness-Erkennung ist serverseitig vorhanden; Fokus-/Tab-Refresh ist in der sichtbaren UI noch nicht vollstaendig. |
| 12.1 | Umgesetzt | Undo-Snapshot-Modell und DB-Transaktionen sind implementiert. |
| 12.2 | Umgesetzt | Undo-Route kann Snapshots wiederherstellen. |
| 12.3 | Umgesetzt | Geloeschte Aufgaben koennen ueber Undo-Snapshots wiederhergestellt werden. |
| 13.1 | Umgesetzt | Full-Scan-Sync erkennt lokale/remote Divergenz und legt Konflikte an. |
| 13.2 | Umgesetzt | Remote-Delete gegen lokal veraenderte Aufgaben erzeugt Konflikte statt stiller Loeschung. |
| 13.3 | Teilweise umgesetzt | Konfliktliste und Detailroute existieren; fachlich klare Vergleichs-UI bleibt offen. |
| 13.4 | Teilweise umgesetzt | Resolve-Handler fuer lokal/remote/manuell existiert; feldweise UI ist unvollstaendig. |
| 13.5 | Teilweise umgesetzt | Split-Resolution existiert im Handler; sichtbarer Split-Flow bleibt offen. |
| 14.1 | Umgesetzt | Quick Add Route, Preview und Save-Flow sind vorhanden. |
| 14.2 | Teilweise umgesetzt | Projekt/Label/Prioritaet werden erkannt; Vorschlags- und Erstell-UI ist unvollstaendig. |
| 14.3 | Umgesetzt | Natuerliche Datumsangaben werden geparst und getestet. |
| 14.4 | Umgesetzt | Recurrence Preview ist im Quick-Add-Parser vorhanden. |
| 15.1a | Umgesetzt | `internal/query` enthaelt Lexer und Token-Definitionen fuer Filterqueries. |
| 15.1b | Umgesetzt | `internal/query` enthaelt Parser und AST fuer Filterqueries. |
| 15.1c | Umgesetzt | Filterqueries werden in SQL/DB-Suchlogik kompiliert und getestet. |
| 15.2 | Teilweise umgesetzt | Saved-Filter-Datenbankfunktionen existieren; Verwaltungs-UI/Routen bleiben offen. |
| 15.3 | Umgesetzt | Systemfilter-Routen und Smart-List-Views sind vorhanden. |
| 16.1 | Teilweise umgesetzt | Task-Labels sind im Modell/Update-Pfad vorhanden; eigene Label-Bearbeitungs-UI ist offen. |
| 16.2 | Umgesetzt | `STARRED`-Mapping, Favorit-Route und Favoriten-View sind umgesetzt. |
| 17.1 | Teilweise umgesetzt | RELATED-TO/Parent-Child-Import ist implementiert; produktive hierarchische UI bleibt offen. |
| 17.2 | Teilweise umgesetzt | Subtask-Routen existieren; sichtbare Erstellung ist noch auszubauen. |
| 17.3 | Teilweise umgesetzt | Backend loescht direkte Unteraufgaben mit expliziter Aktion; sichtbarer Loeschdialog fehlt. |
| 18.1 | Teilweise umgesetzt | RRULE-Preservation und Update-Pfade existieren; dedicated Recurrence-Editor fehlt. |
| 18.2 | Umgesetzt | Komplexe RRULEs werden beim Patchen erhalten. |
| 18.3 | Teilweise umgesetzt | ATTACH und unbekannte Felder werden erhalten und teilweise angezeigt; vollstaendige Anzeige/Klickbarkeit bleibt offen. |
| 19.1 | Teilweise umgesetzt | Hauptnavigation existiert; Todoist-nahe UI-Qualitaet und vollstaendige Zielseiten fehlen. |
| 19.2 | Teilweise umgesetzt | Shortcuts und Hilfe existieren teilweise; vollstaendige Tastaturfuehrung bleibt offen. |
| 19.3 | Teilweise umgesetzt | Write-Status-Indikatoren existieren; alle Mutationsorte sind noch nicht konsistent abgedeckt. |
| 19.4 | Teilweise umgesetzt | Settings-Seite existiert; CalDAV- und Projektverwaltung sind nicht vollstaendig. |
| 19.5 | Teilweise umgesetzt | Theme-/Sprachpraeferenzen existieren teilweise; vollstaendige Lokalisierung/Politur fehlt. |
| 20.1 | Umgesetzt | Go-Build ist ueber Makefile/CI vorhanden. |
| 20.2 | Umgesetzt | Dockerfile und statische Dockerfile-Tests sind vorhanden; Build wird in CI-Kontext abgesichert. |
| 20.3 | Umgesetzt | Compose-Referenzdateien sind vorhanden und statisch pruefbar. |
| 21.1 | Teilweise umgesetzt | Playwright/Chromium E2E und CI existieren; WebKit/Safari ist noch nicht fest verankert. |
| 21.2 | Teilweise umgesetzt | Mobile Smoke-Abdeckung existiert; Tablet-Layout ist noch nicht systematisch fertig. |
| 21.3 | Offen | Keine produktive Drag-and-Drop-Verschiebung oder QA-Abdeckung gefunden. |
| 21.4 | Offen | Suche-zu-gespeichertem-Filter-Flow ist noch nicht umgesetzt. |
| 21.5 | Offen | Beschreibungstext-Linkifizierung ist nicht als fertiger UI-Flow vorhanden. |
| 21.6 | Teilweise umgesetzt | QA-Dokumentation und Browser-CI existieren; Performance-Messungen bleiben offen. |
| 22.1 | Teilweise umgesetzt | Konflikt-Resolve-Backend existiert; feldweise Konflikt-UI bleibt offen. |
| 22.2 | Teilweise umgesetzt | Settings-Grundlage existiert; vollstaendige CalDAV-Management-UI bleibt offen. |
| 22.3 | Teilweise umgesetzt | Quick Add erkennt unbekannte Projekte; Vorschlags-/Anlagefluss ist noch nicht fertig. |

## Neue Planungsstories

Die neuen Epics 23 bis 28 brechen die verbleibende UI- und QA-Arbeit in kleinere Stories auf.

| Story | Status | Evidenzort |
|---|---|---|
| 23.1 | Umgesetzt | `docs/backlog/ui-design-principles.md` dokumentiert die visuelle Todoist-nahe Richtung, Desktop-First-Scope, Dichte, Akzentfarbe und Prioritaeten fuer Aufgabenzeilen, Sidebar, Dialoge und Statusmeldungen. |
| 23.2 | Umgesetzt | `web/static/tailwind.input.css` definiert Caldo-Farb-, Oberflaechen-, Border-, Fokus- und Status-Tokens sowie Basisklassen fuer Buttons, Eingaben, Dialoge, Listenzeilen, Badges und Menues; bestehende Views verwenden diese Klassen. |
