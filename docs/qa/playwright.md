# Playwright Browser-QA

Playwright und npm sind in Caldo nur als Entwicklungs- und QA-Werkzeuge erlaubt. Sie bauen keine Runtime-Assets, werden nicht in das Go-Binary eingebettet und sind nicht Teil des Produktions-Containers.

## Lokale Installation

Node und npm müssen lokal vorhanden sein. Die npm-Abhängigkeiten werden ohne Root installiert:

```bash
npm ci
```

Unter Linux benötigen Playwright-Browser einmalig Systempakete. Dieses Kommando ist der Root-Schritt:

```bash
sudo npx playwright install-deps chromium
```

Der Browser-Download läuft wieder als normaler Nutzer:

```bash
npx playwright install chromium
```

Optionaler WebKit-Check unter Linux:

```bash
sudo npx playwright install-deps webkit
npx playwright install webkit
```

Linux-WebKit ist kein echter Safari-Ersatz. Reale Safari-Freigabe braucht macOS mit Safari oder Safari Technology Preview.

## Tests Ausführen

```bash
npm run test:e2e
npm run test:e2e:headed
npm run test:e2e:ui
```

Alternativ:

```bash
make e2e
make e2e-headed
```

## CI

Der CI-Workflow führt Browser-QA in einem separaten Job aus. Der Job läuft nach den normalen Go-/Build-Checks, installiert Node über `actions/setup-node`, führt `npm ci` aus und installiert Chromium inklusive Linux-Systemabhängigkeiten mit:

```bash
npx playwright install --with-deps chromium
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
- lokale Task-Write-Through-Aktionen über HTTP-Routen
- manueller Full-Scan-Sync
- Remote Create/Update/Delete über Staging-CalDAV
- Dirty-Local-vs-Remote-Changed-Konflikt
- Konfliktdetail und Remote-Auflösung
- einfacher mobiler Screenshot der Suchansicht

Bei Fehlschlägen liegen Artefakte unter `test-results/` und der HTML-Report unter `playwright-report/`. Temporäre Serverlogs liegen während des Laufs unter `.playwright/caldo-e2e/`; mit `CALDO_E2E_KEEP_ARTIFACTS=1` bleibt auch das temporäre Datenverzeichnis erhalten.

## UI-Review

Die Todoist-nahe UI sollte in Browser-QA regelmäßig gegen diese Punkte geprüft werden:

- ruhige linke Navigation, dichte Task-Zeilen und klare Scanbarkeit
- Quick-Add, Suche, Sync-Status und Konfliktansicht ohne Layoutsprünge
- mobile Breiten ohne Textüberlauf oder unbedienbare Controls
- sichtbare Pending/Error/Saved-Zustände bei Writes
- Tastaturpfade für Suche, Navigation und neue Aufgabe

Playwright-Screenshots sind der Feedback-Loop: Ansicht öffnen, Desktop und Mobile prüfen, CSS/Templ anpassen, erneut ausführen.

## Visuelle Baselines

`npm run test:e2e` erzeugt Review-Screenshots unter `test-results/e2e/baselines/`. Das Verzeichnis ist ein lokales QA-Artefakt und wird nicht committed.

Die Baselines erfassen Desktop und Mobile für:

- Setup
- Inbox-Äquivalent über die Default-Projekt-Übersicht
- Heute
- Suche
- Quick Add
- Konflikte
- Einstellungen

Caldo hat laut `arch.md` und PRD keine separate globale Inbox unabhängig von CalDAV. Für UI-Reviews steht deshalb `inbox-equivalent-default-project-*` für die Default-Projekt-/Projektübersicht.

Die E2E-Umgebung nutzt lokale Staging-Daten, keine echten CalDAV-Konten und keine privaten Aufgabeninhalte. Bei visuellen Änderungen: Test ausführen, Screenshots im Baseline-Verzeichnis prüfen, UI anpassen und den Test erneut ausführen.
