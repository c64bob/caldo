# Playwright Browser-QA

Playwright und npm sind in Caldo nur als Entwicklungs- und QA-Werkzeuge erlaubt. Sie bauen keine Runtime-Assets, werden nicht in das Go-Binary eingebettet und sind nicht Teil des Produktions-Containers.

## Lokale Installation

Node und npm müssen lokal vorhanden sein. Die npm-Abhängigkeiten werden ohne Root installiert:

```bash
npm ci
```

Unter Linux benötigen Playwright-Browser einmalig Systempakete. Dieses Kommando ist der Root-Schritt für den normalen Chromium-Smoke:

```bash
sudo npx playwright install-deps chromium
```

Der Browser-Download läuft wieder als normaler Nutzer:

```bash
npx playwright install chromium
```

Für den Safari-nahen WebKit-Smoke werden zusätzlich WebKit-Systempakete und der WebKit-Browser benötigt:

```bash
sudo npx playwright install-deps webkit
npx playwright install webkit
```

Alternativ lädt der gemeinsame Browser-Installationsbefehl beide Browser:

```bash
npm run playwright:install
```

Linux-WebKit ist kein echter Safari-Ersatz. Die Story-21.1-Prüfung nutzt Playwright WebKit als CI-nahe Safari-Approximation; reale Safari-Freigabe braucht macOS mit Safari oder Safari Technology Preview.

## Tests Ausführen

```bash
npm run test:e2e
npm run test:e2e:webkit
npm run test:e2e:ci
npm run test:e2e:headed
npm run test:e2e:webkit:headed
npm run test:e2e:ui
```

Alternativ:

```bash
make e2e
make e2e-webkit
make e2e-ci
make e2e-headed
```

`npm run test:e2e` läuft gegen Chromium. `npm run test:e2e:webkit` läuft denselben MVP-Smoke gegen Playwright WebKit. `npm run test:e2e:ci` führt beide Browser nacheinander aus; dadurch bekommt jeder Browser ein frisches temporäres Caldo/Staging-CalDAV-Setup.

## CI

Der CI-Workflow führt Browser-QA in einem separaten Job aus. Der Job läuft nach den normalen Go-/Build-Checks, installiert Node über `actions/setup-node`, führt `npm ci` aus und installiert Chromium und WebKit inklusive Linux-Systemabhängigkeiten mit:

```bash
npx playwright install --with-deps chromium webkit
```

Anschließend läuft:

```bash
npm run test:e2e:ci
```

Bei Fehlschlägen werden Playwright-Report, Test-Artefakte und lokale Caldo/Staging-CalDAV-Logs als Workflow-Artefakt hochgeladen.

Der Playwright-Global-Setup startet automatisch:

- `cmd/stagecaldav` mit In-Memory-Testdaten
- `cmd/caldo` mit temporärer SQLite-Datenbank
- Reverse-Proxy-Auth-Testheader für Browser-Requests

Die Tests verwenden keine echten CalDAV-Zugänge. Remote-Änderungen laufen über die token-geschützte Admin-API der lokalen Staging-CalDAV-Testinstanz.

## MVP-Smoke

Der erste Browser-Smoke deckt ab:

- Setup-Gate, CalDAV-Konfiguration, Kalenderauswahl und Initialimport
- globale Navigation, Tastaturkürzel und Theme-Toggle
- Aufgaben erstellen, inline und im Detailpanel bearbeiten, erledigen, wieder öffnen und löschen
- Suche sowie Fokus-Refresh bei Remote-Änderungen
- tabbezogene Undo-Anzeige
- lokale Task-Write-Through-Aktionen über HTTP-Routen
- SSE-/Sync-Status-Pfade über die lokale Browser-Session
- Tablet-Layout für Heute, Demnächst, Projekte, Suche und Einstellungen bei 834x1112 ohne horizontales Seiten-Overflow
- Tablet-Erreichbarkeit von Navigation, Suche, Quick Add, Sync, Theme sowie Task-Detail-, Erledigen- und Löschdialogen
- manueller Full-Scan-Sync
- Remote Create/Update/Delete über Staging-CalDAV
- Dirty-Local-vs-Remote-Changed-Konflikt
- Konfliktdetail und Remote-Auflösung
- einfacher mobiler Screenshot der Suchansicht

Der Smoke ist für Chromium und Playwright WebKit identisch. Er nutzt ausschließlich die lokale Staging-CalDAV-Testinstanz und keine echten CalDAV-Zugänge.

Bei Fehlschlägen liegen Artefakte unter `test-results/` und der HTML-Report unter `playwright-report/`. Browser-spezifische Review-Screenshots liegen unter `test-results/e2e/chromium/` oder `test-results/e2e/webkit/`. Temporäre Serverlogs liegen während des Laufs unter `.playwright/caldo-e2e/`; mit `CALDO_E2E_KEEP_ARTIFACTS=1` bleibt auch das temporäre Datenverzeichnis erhalten.

Performance-Zielwerte werden nicht im normalen Browser-Smoke bewertet. Die wiederholbaren Messpunkte fuer Startzeit, erste UI-Ansicht, Initialimport und inkrementellen Sync stehen in `docs/qa/performance.md`.

## UI-Review

Die Todoist-nahe UI sollte in Browser-QA regelmäßig gegen diese Punkte geprüft werden:

- ruhige linke Navigation, dichte Task-Zeilen und klare Scanbarkeit
- Quick-Add, Suche, Sync-Status und Konfliktansicht ohne Layoutsprünge
- mobile Breiten ohne Textüberlauf oder unbedienbare Controls
- Tablet-Breite 834x1112: Heute, Demnächst, Projekte, Suche und Einstellungen ohne horizontales Scrollen prüfen
- Tablet-Breite 834x1112: Hauptnavigation, Suche, Quick Add, Sync, Theme und Task-Dialogaktionen bleiben sichtbar erreichbar
- sichtbare Pending/Error/Saved-Zustände bei Writes
- Tastaturpfade für Suche, Navigation und neue Aufgabe

Playwright-Screenshots sind der Feedback-Loop: Ansicht öffnen, Desktop, Tablet und Mobile prüfen, CSS/Templ anpassen, erneut ausführen.

## Visuelle Baselines

`npm run test:e2e` erzeugt Review-Screenshots unter `test-results/e2e/chromium/baselines/`. `npm run test:e2e:webkit` erzeugt die WebKit-Variante unter `test-results/e2e/webkit/baselines/`. Diese Verzeichnisse sind lokale QA-Artefakte und werden nicht committed.

Die Baselines erfassen Desktop, Tablet und Mobile für:

- Setup
- Inbox-Äquivalent über die Default-Projekt-Übersicht
- Heute
- Demnächst
- Suche
- Quick Add
- Konflikte
- Einstellungen

Caldo hat laut `arch.md` und PRD keine separate globale Inbox unabhängig von CalDAV. Für UI-Reviews steht deshalb `inbox-equivalent-default-project-*` für die Default-Projekt-/Projektübersicht.

Die E2E-Umgebung nutzt lokale Staging-Daten, keine echten CalDAV-Konten und keine privaten Aufgabeninhalte. Bei visuellen Änderungen: Test ausführen, Screenshots im Baseline-Verzeichnis prüfen, UI anpassen und den Test erneut ausführen.
