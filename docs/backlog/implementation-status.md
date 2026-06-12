# Implementation Status Audit

Stand: 2026-06-12

Dieses Dokument sammelt die Evidenz zum Status der bestehenden Stories. Die Story-Dateien selbst enthalten nur den knappen Status, damit sie Planungsdokumente bleiben.

## Zusammenfassung

- Bestehende Stories: 89 `Umgesetzt`, 0 `Teilweise umgesetzt`, 0 `Offen`.
- Neue Planungsstories: 25 `Umgesetzt`, 14 `Offen`.
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
| 2.3 | Umgesetzt | `internal/handler/session.go`, Router-Wiring, CSRF-Middleware, `web/assets/app.js` und Task-Undo-/Mutationshandler setzen ein sicheres `session_id`-Cookie, liefern CSRF-Token per Double-Submit-HMAC, senden `X-CSRF-Token`/`X-Tab-ID` fuer HTMX/Fetch und speichern Undo-Snapshots ueber `(session_id, tab_id)`. |
| 2.4 | Umgesetzt | Lokale Asset-Auslieferung, Manifest und CSP sind in Router/Layout/Asset-Tests abgedeckt. |
| 2.5 | Umgesetzt | `internal/view/layout.templ`, lokale Manifest-Assets, CSP-/Router-Tests und `web/assets/app.js` liefern BaseLayout mit CSRF-Meta, Notifications-Ziel, Navigationsplatzhaltern, lokalem HTMX/HTMX-SSE/Alpine/app/css und einem CSP-kompatiblen `button[data-theme-toggle]`, der Systempraeferenz sowie Light-/Dark-Klassen auf `<html>` steuert. |
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
| 7.2 | Umgesetzt | Write-through-Edit fuer Kernfelder ist ueber Inline-/Detailformulare und Completion/Reopen-Aktionen abgedeckt; Handler-Tests pruefen Version, Undo, Pending, CalDAV-Erfolg, Projektwechsel und Fehlerstatus. |
| 7.3 | Umgesetzt | Complete/Reopen schreiben VTODO-Status, `completed_at`, Undo-Snapshot, ETag und Sync-Status ueber den Write-through-Pfad; sichtbare Aktionen, Fehleranzeige, Standard-Ausblendung erledigter Aufgaben und RRULE-Erhalt sind getestet. |
| 7.4 | Umgesetzt | Delete-Write-through prueft `expected_version` vor Bestaetigungs-/CalDAV-Schritten, nutzt den sichtbaren Bestaetigungsdialog, erstellt Undo-Snapshots, entfernt lokale Zeilen erst nach erfolgreichem DELETE und behandelt CalDAV-404 als Erfolg. |
| 7.5 | Umgesetzt | `internal/view/task_rows.templ`, `internal/view/task_rows.go`, `web/assets/app.js` und `internal/handler/tasks_complete.go` zeigen beim Erledigen von Elternaufgaben mit offenen direkten Unteraufgaben einen Entscheidungsdialog fuer Elternaufgabe, offene Unteraufgaben oder Abbruch; die bestehenden Handler pruefen Versionen und schreiben jede betroffene Task zu CalDAV. |
| 8.1 | Umgesetzt | `internal/view/navigation_pages.templ` und `internal/handler/projects_create.go` stellen auf der Projektseite eine produktive Anlage bereit; der Handler legt zuerst per CalDAV `MKCALENDAR` an, speichert danach lokal, rendert Fehler sichtbar ohne optimistischen Listeneintrag und Router-E2E prueft die Fake-CalDAV-Kalendersichtbarkeit. |
| 8.2 | Umgesetzt | Die Projektseite rendert Umbenennen-Formulare mit `expected_version`; `internal/handler/projects_rename.go` benennt zuerst den CalDAV-Kalender um, aktualisiert danach lokales Projekt, denormalisierte `project_name`-Felder und FTS-Suche, und zeigt Fehler ohne lokale Teilumbenennung sichtbar an. |
| 8.3 | Umgesetzt | Die Projektseite zeigt pro Projekt eine starke Loeschbestaetigung mit Projektname, betroffener Task-Anzahl und `expected_version`; `internal/handler/projects_delete.go` loescht zuerst den CalDAV-Kalender, entfernt danach lokales Projekt und Tasks, zeigt Fehler ohne lokale Teilloeschung und Router-E2E prueft Kalender-DELETE ohne einzelne Task-DELETEs sowie Default-Projekt-Blockade. |
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
| 11.3 | Umgesetzt | `GET /api/tasks/versions` bleibt als Versionsabgleich erhalten; `GET /tasks/{taskID}` rendert aktuelle TaskRow-Fragmente, `web/assets/app.js` prueft sichtbare Task-Versionen bei Fokus/Sichtbarkeit, ersetzt nur veraltete saubere Zeilen und warnt bei offenen Formularen mit lokalen Aenderungen ohne diese zu ueberschreiben. |
| 12.1 | Umgesetzt | Undo-Snapshot-Modell und DB-Transaktionen sind implementiert. |
| 12.2 | Umgesetzt | Undo-Route kann Snapshots wiederherstellen. |
| 12.3 | Umgesetzt | Geloeschte Aufgaben koennen ueber Undo-Snapshots wiederhergestellt werden. |
| 13.1 | Umgesetzt | Full-Scan-Sync erkennt lokale/remote Divergenz und legt Konflikte an. |
| 13.2 | Umgesetzt | Remote-Delete gegen lokal veraenderte Aufgaben erzeugt Konflikte statt stiller Loeschung. |
| 13.3 | Umgesetzt | Hauptnavigation, globale ungelöste Konfliktliste und Detailroute sind vorhanden; die Detailansicht zeigt Base-, lokale und Remote-Versionen als fachliche Feldvergleichstabelle, geloeschte Seiten als fehlend und Task-Zeilen mit bekanntem offenem Konflikt verlinken direkt zur Konfliktansicht. |
| 13.4 | Umgesetzt | Konfliktdetails rendern lokale, Remote-, Split- und manuelle Resolve-Formulare; die manuelle Form waehlt Base/Lokal/Remote fuer Titel, Beschreibung, Faelligkeit, Prioritaet, Labels, Status und Unteraufgaben sowie das Zielprojekt, schreibt die geloeste VTODO zu CalDAV, aktualisiert `resolved_at`/`resolution` und laesst Konflikte bei Write-Fehlern ungeloest. |
| 13.5 | Umgesetzt | Die Konfliktdetailseite zeigt einen Split-Flow mit Vorschau der lokalen Aufgabe und der neu anzulegenden Remote-Version; `ResolveConflict` schreibt die Remote-Version mit neuer UID zu CalDAV, entfernt Parent-Links, laesst die lokale UID bestehen, speichert beide Tasks im selben Projekt, markiert `resolution=split` erst nach erfolgreicher Persistenz und laesst den Konflikt bei Write-/Persistenzfehlern ungeloest. |
| 14.1 | Umgesetzt | Quick Add Route, Preview und Save-Flow sind vorhanden. |
| 14.2 | Umgesetzt | Quick Add nutzt gemeinsame `#`/`@`-Token-Grenzen mit der Filter-Suche, loest `#Projekt` auf, zeigt unbekannte Projekte als explizite CalDAV-Kalender-Anlageoption, erstellt neue Projekte vor dem Task-Write ueber CalDAV, uebergibt `@Label`-Tokens an die automatische Label-Anlage und erkennt `!high`, `!medium`, `!low`, `!1`, `!2` und `!3` in Preview und Speichern. |
| 14.3 | Umgesetzt | Natuerliche Datumsangaben werden geparst und getestet. |
| 14.4 | Umgesetzt | Recurrence Preview ist im Quick-Add-Parser vorhanden. |
| 15.1a | Umgesetzt | `internal/query` enthaelt Lexer und Token-Definitionen fuer Filterqueries. |
| 15.1b | Umgesetzt | `internal/query` enthaelt Parser und AST fuer Filterqueries. |
| 15.1c | Umgesetzt | Filterqueries werden in SQL/DB-Suchlogik kompiliert und getestet. |
| 15.2 | Umgesetzt | Gespeicherte Filter haben lokale CRUD-Routen und Templ-Verwaltungs-UI fuer Name, Query und Favorit; Aendern und Loeschen nutzen `server_version`, Filter werden nicht zu CalDAV synchronisiert, favorisierte Filter erscheinen in der Sidebar-Navigation und `/filters/{filter_id}` rendert gespeicherte Queries als Aufgabenansicht, wobei Syntaxfehler zu einer leeren Ergebnisliste fuehren. |
| 15.3 | Umgesetzt | Systemfilter-Routen und Smart-List-Views sind vorhanden. |
| 16.1 | Umgesetzt | Aufgabenzeilen und Detailpanel bieten Labelbearbeitung; `POST /tasks/{task_id}/labels` und der allgemeine Update-Pfad schreiben Labels als VTODO-`CATEGORIES` zu CalDAV, legen neue Labels lokal an, aktualisieren `task_labels`/`label_names`, erstellen Undo-Snapshots, pruefen `expected_version` und aktualisierte Labels sind in Suche und gespeicherten Filtern wirksam. |
| 16.2 | Umgesetzt | `STARRED`-Mapping, Favorit-Route und Favoriten-View sind umgesetzt. |
| 17.1 | Umgesetzt | `RELATED-TO;RELTYPE=PARENT` und Nextcloud-kompatibles `RELATED-TO` werden importiert, genau eine Ebene wird ueber `parent_id` dargestellt, tiefere Ebenen bleiben Root-Aufgaben mit unveraendertem Raw-VTODO und Unteraufgaben erscheinen eingerueckt mit Parent-Metadaten. |
| 17.2 | Umgesetzt | Direkte Unteraufgaben werden ausschliesslich ueber `Unteraufgabe hinzufuegen` erstellt, Quick Add lehnt Parent-Felder ab, Subtasks schreiben `RELATED-TO;RELTYPE=PARENT` sofort zu CalDAV, verschachtelte Subtasks sind blockiert und Handler-/Router-E2E-Tests decken die Fake-CalDAV-Sichtbarkeit ab. |
| 17.3 | Umgesetzt | `internal/view/task_rows.templ`, `web/assets/app.js`, `internal/handler/tasks_delete.go` und `internal/handler/tasks_delete_test.go` zeigen die Anzahl direkter Unteraufgaben im Loeschdialog, senden die explizite `delete_all`-Aktion und loeschen Elternaufgabe sowie direkte Unteraufgaben einzeln mit Undo-Snapshot fuer die bestaetigte Loeschaktion. |
| 18.1 | Umgesetzt | Das Detailpanel rendert einen RRULE-Editor fuer MVP-Muster mit taeglich, woechentlich, monatlich, jaehrlich, werktags, Wochentag, Intervall sowie Ende nie/bis Datum/nach Anzahl; `internal/handler/tasks_update.go` ersetzt RRULEs nur bei expliziter Wiederholungsbearbeitung, waehrend andere Feldspeicherungen RRULEs unveraendert lassen. |
| 18.2 | Umgesetzt | Komplexe RRULEs werden beim Patchen erhalten. |
| 18.3 | Umgesetzt | `model.ParseVTODOFields` extrahiert `ATTACH`-Properties, Aufgabenzeilen und Detailpanel zeigen externe Anhaenge read-only als sichere Links sowie Inline-/Binary-Anhaenge als vorhandenen Anhang; `PatchVTODO` und der Task-Update-Handler erhalten `ATTACH` und unbekannte VTODO-Properties bei normalen Feldbearbeitungen. |
| 19.1 | Umgesetzt | Die App-Shell rendert Desktop- und Mobile-Hauptnavigation mit Heute, Demnaechst, Projekte, Labels, Filter, Favoriten, Suche, Konflikte und Einstellungen; aktive Ansichten erhalten die visuelle aktive Klasse und `aria-current="page"`. |
| 19.2 | Umgesetzt | `web/assets/app.js` bietet lokale CSP-kompatible Shortcuts fuer neue Aufgabe, Suche, Hauptansichtenwechsel und Hilfe, ignoriert aktive Eingabefelder und der Hilfedialog dokumentiert die verfuegbaren Kuerzel; Browser-QA deckt die Shortcut-Pfade ab. |
| 19.3 | Umgesetzt | Der globale HTMX-/Fetch-Write-Tracker zeigt Pending-, Erfolgs- und Fehlerstatus sichtbar an, nutzt `beforeunload` bei laufenden Writes, erhaelt fehlgeschlagene Formularwerte ohne Browser-Offline-Queue und Browser-QA deckt Pending, Fehler, Reload-Erfolg und Navigation mit laufendem Write ab. |
| 19.4 | Umgesetzt | `/settings` rendert CalDAV-URL, Benutzername, Passwort-/App-Passwort-Aenderung mit Verbindungstest, Kalender-/Projektmapping, Default-Projekt, Sync-Intervall, manuellen Sync, erledigte Aufgaben, Demnaechst-Zeitraum, Sprache, Dark Mode sowie Reverse-Proxy-/HTTPS-Status; `/settings/caldav` und `/settings/calendars` speichern die Normalbetriebskonfiguration ohne Setup-Wizard. |
| 19.5 | Umgesetzt | `internal/view/ui.go`, `internal/view/layout.templ`, `internal/view/quick_add.templ`, `internal/handler/navigation.go` und `internal/handler/quick_add.go` wenden gespeicherte UI-Sprache und Dark-Mode-Praeferenz auf Shell, Settings und Quick Add an; `light`/`dark` ueberschreiben die Systempraeferenz, `system` folgt `prefers-color-scheme`, und Quick Add nutzt die gespeicherte Sprache fuer natuerliche Eingabe. |
| 20.1 | Umgesetzt | Go-Build ist ueber Makefile/CI vorhanden. |
| 20.2 | Umgesetzt | Dockerfile und statische Dockerfile-Tests sind vorhanden; Build wird in CI-Kontext abgesichert. |
| 20.3 | Umgesetzt | Compose-Referenzdateien sind vorhanden und statisch pruefbar. |
| 21.1 | Umgesetzt | `playwright.config.js`, `package.json`, CI-Browser-QA und `docs/qa/playwright.md` verankern denselben MVP-Smoke fuer Chromium und Playwright WebKit; die Browserlaeufe nutzen jeweils frische lokale Caldo-/Staging-CalDAV-Instanzen und decken Setup, Navigation, Suche, Tastaturkuerzel, Write-through-Aufgabenaktionen, Fokus-Refresh, Undo, Sync/SSE-Statuspfade und Konflikte ab. |
| 21.2 | Umgesetzt | `tests/e2e/mvp-flow.spec.js` und `docs/qa/playwright.md` pruefen/dokumentieren Tablet-QA bei 834x1112 fuer Heute, Demnaechst, Projekte, Suche und Einstellungen inklusive horizontalem Overflow, erreichbarer Navigation/Topbar-Aktionen sowie Task-Detail-, Erledigen- und Loeschdialogen. |
| 21.3 | Umgesetzt | `internal/view/task_rows.templ`, `internal/view/layout.templ`, `internal/view/navigation_pages.templ`, `web/assets/app.js` und `internal/handler/tasks_update.go` markieren verschiebbare Aufgaben und Projektziele, senden `POST /tasks/{taskID}/move` mit `project_id`, `expected_version`, CSRF und Tab-ID und nutzen den bestehenden CalDAV-Write-through-Pfad mit sichtbarer Fehleranzeige ohne optimistische finale DOM-Verschiebung. |
| 21.4 | Umgesetzt | `internal/handler/search.go` prueft Suchanfragen per Filter-Lexer/-Parser/-Compiler auf eindeutige gespeicherte-Filter-Syntax; `internal/view/search.templ` bietet nur dann ein `POST /filters`-Formular mit uebernommener Query und Name an, und Browser-QA speichert einen Filter aus der Suche heraus. |
| 21.5 | Umgesetzt | `internal/view/task_rows.templ`, `internal/view/task_rows.go`, `web/static/tailwind.input.css` und Browser-/View-Tests rendern `http`-/`https`-URLs in Aufgabenbeschreibungen als sichere Links mit `rel="noopener noreferrer"`, waehrend Text ohne URL und die zugrunde liegenden Beschreibungstexte in Editierfeldern unveraendert bleiben. |
| 21.6 | Umgesetzt | `docs/qa/performance.md` dokumentiert wiederholbare QA-Messpunkte mit PRD-Zielwerten fuer Startzeit ohne Migrationen, erste UI-Ansicht mit 10.000 lokalen Tasks, Initialimport mit 400 Remote-Tasks und inkrementellen Sync mit 400 Tasks; `docs/qa/playwright.md` verweist auf den separaten Performance-QA-Prozess. |
| 22.1 | Umgesetzt | `internal/view/conflicts.templ`, `internal/view/conflicts.go`, `web/static/tailwind.input.css` und Konflikt-View-/Handler-Tests zeigen je Konfliktfeld explizite Base-/Lokal-/Remote-Quellen mit den zugehoerigen Werten als Radio-Auswahl und behalten den bestehenden `ResolveConflict`-Pfad fuer feldweise Aufloesung bei. |
| 22.2 | Umgesetzt | `/settings` rendert CalDAV-URL, Benutzername, Passwort-/App-Passwort, separaten Verbindungstest, Speichern nach erfolgreichem Test, Kalenderauswahl und Default-Projekt; `internal/handler/settings_update.go`, `internal/view/settings.go`, `internal/view/settings_test.go` und `internal/handler/settings_update_test.go` decken Test-only, Save und Default-Validierung ab. |
| 22.3 | Umgesetzt | `internal/handler/quick_add.go`, `internal/view/quick_add.templ`, `internal/handler/tasks_create.go` und zugehoerige Tests zeigen bei unbekanntem `#Projekt` eine auswählbare Projektvorschlagsliste, bieten direktes CalDAV-Projektanlegen an und nutzen die Auswahl beim aktuellen Quick-Add-Speichern. |

## Neue Planungsstories

Die neuen Epics 23 bis 28 brechen die verbleibende UI- und QA-Arbeit in kleinere Stories auf.

| Story | Status | Evidenzort |
|---|---|---|
| 23.1 | Umgesetzt | `docs/backlog/ui-design-principles.md` dokumentiert die visuelle Todoist-nahe Richtung, Desktop-First-Scope, Dichte, Akzentfarbe und Prioritaeten fuer Aufgabenzeilen, Sidebar, Dialoge und Statusmeldungen. |
| 23.2 | Umgesetzt | `web/static/tailwind.input.css` definiert Caldo-Farb-, Oberflaechen-, Border-, Fokus- und Status-Tokens sowie Basisklassen fuer Buttons, Eingaben, Dialoge, Listenzeilen, Badges und Menues; bestehende Views verwenden diese Klassen. |
| 23.3 | Umgesetzt | `internal/view/layout.templ` und `web/static/tailwind.input.css` definieren eine Desktop-App-Shell mit stabiler Sidebar, sticky Topbar, globalen Quick-Add-/Such-/Sync-Aktionen, begrenztem Inhaltsbereich und nicht ueberlappender Write-Status-Anzeige. |
| 23.4 | Umgesetzt | `internal/view/layout.templ`, `web/static/tailwind.input.css` und `web/assets/app.js` liefern responsive Topbar, Mobile-Navigation per Dialog und kleine Viewports; `tests/e2e/mvp-flow.spec.js` prueft den mobilen Navigationspfad. |
| 23.5 | Umgesetzt | `internal/db/navigation.go`, `internal/handler/navigation.go`, `internal/view/layout.templ` und `internal/view/navigation_pages.templ` laden Zaehler, aktive Navigationszustaende und knappe Empty-Zustaende fuer Systemlisten, Projekte, Labels und Filter. |
| 23.6 | Umgesetzt | `internal/view/states.templ`, `internal/handler/navigation_pages.go`, `internal/view/date_views.templ`, `internal/view/search.templ` und die CSS-State-Klassen rendern leere, fehlerhafte und ladende Grundzustaende ohne sensible Fehlerdetails. |
| 23.7 | Umgesetzt | `tests/e2e/mvp-flow.spec.js` erzeugt reproduzierbare Desktop- und Mobile-Baselines fuer Setup, Projekte, Heute, Suche, Quick Add, Einstellungen und Konflikte; `docs/qa/playwright.md` dokumentiert den lokalen QA-Artefaktpfad. |
| 24.1 | Umgesetzt | `internal/view/task_rows.templ`, `internal/view/task_rows.go` und `web/static/tailwind.input.css` rendern scanbare Aufgabenzeilen mit Checkbox, Titel, Beschreibung, Metadaten, Labels, Anhaengen und Sync-/Konflikt-/Fehlerzustaenden; Handler- und View-Tests decken die Zeilen ab. |
| 24.2 | Umgesetzt | `internal/view/task_rows.templ`, `internal/handler/date_views.go`, `internal/handler/search.go`, `internal/handler/tasks_create.go` und `web/assets/app.js` implementieren Inline-Erstellung mit Projekt-/Datumskontext, Cancel ohne Entwurf, lokaler Fehlermeldung und Tastaturfluss; `tests/e2e/mvp-flow.spec.js` prueft Cancel und Enter-Speichern im Browser. |
| 24.3 | Umgesetzt | `internal/view/task_rows.templ`, `internal/handler/tasks_update.go` und `web/assets/app.js` implementieren Inline-Bearbeitung fuer Titel, Beschreibung, Faelligkeit, Prioritaet, Projekt und Labels mit Fehleranzeige und Tastaturfluss. |
| 24.4 | Umgesetzt | `internal/view/task_rows.templ`, `internal/handler/tasks_update.go`, `web/assets/app.js` und die Task-Zeilenaktionen stellen ein produktives Detailpanel fuer relevante Aufgabenfelder bereit. |
| 24.5 | Umgesetzt | `internal/view/task_rows.templ`, `internal/handler/tasks_complete.go`, `internal/handler/tasks_delete.go` und `web/assets/app.js` zeigen Complete/Reopen/Delete-Aktionen, Pending-/Fehlerzustaende und Undo-Fuehrung im Listenfluss. |
| 24.6 | Umgesetzt | `internal/view/task_rows.templ`, `internal/handler/tasks_update.go`, `internal/handler/tasks_favorite.go` und zugehoerige Tests machen Prioritaet, Labels, Favorit und Faelligkeit in der Aufgabenzeile sichtbar und direkt bearbeitbar. |
| 24.7 | Umgesetzt | `internal/view/task_rows.templ`, `internal/view/task_rows.go`, `internal/handler/tasks_create.go` und Browser-Tests gruppieren Unteraufgaben visuell und ermoeglichen Erstellung direkt an der passenden Elternaufgabe. |
| 24.8 | Umgesetzt | `internal/view/undo.templ`, `internal/handler/tasks_undo.go`, `internal/db/tasks_undo.go` und `web/assets/app.js` liefern tabbezogene Undo-Statusanzeige mit Wiederherstellung, Fehler- und Ablaufzustaenden. |
| 25.1 | Umgesetzt | `internal/view/layout.templ`, `internal/view/quick_add.templ`, `web/assets/app.js` und `web/static/tailwind.input.css` stellen ein globales Quick-Add-Overlay mit sichtbaren Shell-Aktionen, Tastaturstart, Kontextverbleib, responsivem Dialog und sichtbaren Fehlern bei fehlgeschlagenem Speichern bereit; `tests/e2e/mvp-flow.spec.js` prueft den Browserfluss. |
| 25.2 | Umgesetzt | `internal/view/quick_add.templ`, `internal/handler/quick_add.go`, `web/assets/app.js` und `web/static/tailwind.input.css` rendern erkannte Quick-Add-Bestandteile als Chips, stellen editierbare Korrekturfelder fuer Titel, Projekt, Labels, Datum, Wiederholung und Prioritaet bereit, halten unbekannte Eingabe im Titel und speichern die sichtbaren Korrekturwerte; `tests/e2e/mvp-flow.spec.js` prueft Korrektur und Speichern im Browser. |
| 25.3 | Umgesetzt | `internal/handler/quick_add.go`, `internal/db/labels.go`, `internal/view/quick_add.templ`, `web/assets/app.js` und `web/static/tailwind.input.css` speisen Quick Add mit lokalen Projekt- und Labelvorschlaegen, zeigen bestehende und neue Labels unterscheidbar an, lassen Labelvorschlaege per Button in das Korrekturfeld uebernehmen und verlangen bei unbekannten Projekten eine explizite Auswahl zwischen bestehendem Projekt und Neuanlage; Handler-, View-, DB- und Browser-Tests decken den Flow ab. |
| 25.4 | Umgesetzt | `internal/parser/quickadd.go`, `internal/view/quick_add.templ`, `web/static/tailwind.input.css` und `tests/e2e/mvp-flow.spec.js` zeigen natuerliche Datumsangaben als konkrete ISO-Daten mit erkannter Eingabequelle, markieren mehrdeutige Wochentagsangaben mit Korrekturhinweis, erlauben Entfernen und Ersetzen ueber das bestehende Datumskorrekturfeld und pruefen, dass die Preview-Route keine Aufgabe persistiert. |
| 25.5 | Umgesetzt | `internal/view/quick_add.go`, `internal/view/quick_add.templ`, `internal/handler/tasks_create.go`, `web/assets/app.js`, `web/static/tailwind.input.css` und `tests/e2e/mvp-flow.spec.js` zeigen einfache RRULEs in Quick Add als menschenlesbare Chips, behalten komplexe RRULEs raw erhalten, machen Prioritaeten als P1/P2/P3 plus Text sichtbar und blockieren fehlerhafte Wiederholungswerte mit sichtbarem Hinweis vor dem Speichern. |
| 25.6 | Umgesetzt | `web/assets/app.js` und `tests/e2e/mvp-flow.spec.js` ergaenzen den Quick-Add-Overlay-Fluss um Tastaturstart per `N`, erwartete Fokusuebergabe nach Preview, Escape-Abbruch mit Fokus-Rueckkehr, Ctrl/Cmd+Enter-Speichern, Pfeiltasten-Navigation ueber erkannte Tokenchips und Browser-QA fuer den tastaturbasierten Schnellanlagepfad. |
| 26.1 | Umgesetzt | `internal/db/navigation.go`, `internal/handler/navigation_pages.go`, `internal/view/layout.templ`, `web/static/tailwind.input.css` und `tests/e2e/mvp-flow.spec.js` listen Projekte stabil in der Sidebar, zeigen offene Aufgabenzaehler, verlinken auf `/projects/{projectID}`, markieren die aktive Projektansicht und halten lange Projektlisten scrollbar. |
| 26.2 | Umgesetzt | `internal/view/navigation_pages.templ`, `internal/handler/projects_create.go`, `internal/handler/projects_rename.go`, `internal/handler/projects_delete.go` und Projekt-/Router-E2E-Tests zeigen Projektanlage, Umbenennung und Loeschung mit sichtbaren Erfolgs- und Fehlerzustaenden, klarer Loeschfolge fuer lokale Aufgaben und unveraendertem Remote-Write-through vor lokaler Persistenz. |
| 26.3 | Umgesetzt | `internal/db/labels.go`, `internal/db/tasks_views.go`, `internal/handler/navigation_pages.go`, `internal/view/navigation_pages.templ`, `internal/view/task_rows.templ`, `web/static/tailwind.input.css` und `tests/e2e/mvp-flow.spec.js` verlinken Labels auf `/labels/{labelID}`, zeigen Label-Zaehler und passende Aufgaben, nutzen die bestehenden write-through Label-Editoren an Aufgaben und halten lange Labelnamen layoutstabil. |
| 26.4 | Umgesetzt | `internal/handler/saved_filters.go`, `internal/db/navigation.go`, `internal/view/saved_filters.templ` und `tests/e2e/mvp-flow.spec.js` bieten Filteranlage, Umbenennen, Loeschen und Favorisieren mit sichtbarer Queryvalidierung vor dem Speichern; gueltige favorisierte Filter erscheinen in der Navigation, ungueltige neue oder bestehende Favoriten nicht. |
| 26.5 | Umgesetzt | Die Suche uebernimmt valide Filterqueries in ein benanntes Speichern-Formular, validiert die Ueberfuehrbarkeit vor Anzeige, erstellt favorisierte gespeicherte Filter ueber die bestehende Filter-Route und zeigt sie danach in Filterverwaltung und Navigation. |
| 26.6 | Umgesetzt | Projekt-Drop-Ziele in Sidebar und Projektuebersicht sowie draggable Task-Zeilen rufen `POST /tasks/{taskID}/move` auf; Handler- und View-Tests decken Endpunkt, Pflicht-`project_id` und Markup fuer Drag-and-Drop ab. |
| 27.1 | Offen | Konfliktliste existiert, aber nicht als ausgearbeitete Arbeitsansicht. |
| 27.2 | Offen | Konfliktdetail zeigt Basisinformationen, aber keinen produktiven Feldvergleich. |
| 27.3 | Offen | Feldweise Konfliktloesung ist nicht als sichtbarer UI-Fluss umgesetzt. |
| 27.4 | Offen | Split-Konflikt ist backendseitig vorbereitet, aber nicht sichtbar produktiv ausfuehrbar. |
| 27.5 | Offen | CalDAV-Einstellungen sind nicht vollstaendig verwaltbar. |
| 27.6 | Offen | Kalenderauswahl und Default-Projekt nach Setup sind nicht als produktive Settings-UI umgesetzt. |
| 28.1 | Offen | Tablet-Layout ist nicht systematisch fuer alle Kernansichten abgesichert. |
| 28.2 | Offen | Mobile Navigation ist grundlegend vorhanden, aber die eigene Story fuer kleine Breiten ist noch nicht vollstaendig umgesetzt. |
| 28.3 | Offen | Safari-/WebKit-QA ist nicht fest im Prozess verankert. |
| 28.4 | Offen | Tastatur-, Fokus- und Accessibility-Abdeckung ist nicht vollstaendig umgesetzt. |
| 28.5 | Offen | Performance-Szenarien und Messwerte sind nicht dokumentiert oder automatisiert. |
| 28.6 | Offen | Visuelle Regression wird ueber Baseline-Screenshots vorbereitet, aber kein vollstaendiger Review-/Regression-Prozess ist umgesetzt. |
