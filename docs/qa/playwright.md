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

## Safari-/WebKit-QA-Gate

WebKit-QA ist der Safari-nahe Browser-Gate für Caldo. Er nutzt denselben MVP-Smoke wie Chromium und läuft ausschließlich gegen die lokale Staging-CalDAV-Testinstanz aus dem Playwright-Global-Setup. Für diesen automatisierten Gate dürfen keine echten CalDAV-Zugänge, privaten Aufgaben oder Nextcloud-Staging-Secrets verwendet werden.

Automatisiert:

- Jeder Pull Request und jeder Push auf `main` durchläuft den CI-Job `browser-qa`.
- Der Job installiert Chromium und WebKit inklusive Linux-Systemabhängigkeiten mit `npx playwright install --with-deps chromium webkit`.
- Danach läuft `npm run test:e2e:ci`, also Chromium und WebKit nacheinander mit jeweils frischer temporärer Caldo-/Staging-CalDAV-Instanz.
- Vor jedem Release Candidate muss der CI-Status dieses Jobs für den Ziel-Commit `pass` sein oder als Release-Blocker behandelt werden.

Lokal:

- Führe `npm run test:e2e:webkit` aus, wenn Änderungen Templ, CSS, `web/assets/app.js`, HTMX-/SSE-Flows, Dialoge, Fokusverhalten, Quick Add, Sync-Status, Konflikte oder Settings berühren.
- Führe `npm run test:e2e:webkit` vor einem Release lokal aus, wenn CI nicht verfügbar ist oder ein WebKit-/Safari-naher Befund überprüft werden muss.
- Nutze `npm run test:e2e:webkit:headed` nur zur lokalen Diagnose. Headed-Läufe sind keine zusätzlichen Commit-Artefakte.
- Reale Safari-Prüfung auf macOS ist zusätzliche manuelle Release-Sicherheit, ersetzt aber nicht den automatisierten WebKit-Smoke im CI-nahen Prozess.

Der WebKit-Smoke deckt die kritischen MVP-Flows ab:

- Setup-Gate, CalDAV-Konfiguration, Kalenderauswahl und Initialimport.
- Quick Add inklusive Overlay, Preview, Korrekturfeldern, Token-Chips, Tastaturpfad und Speichern.
- Task Write-through: Erstellen, Inline-Bearbeitung, Detailpanel-Bearbeitung, Erledigen, Wiederöffnen, Favorisieren, Verschieben, Unteraufgabe und Löschen mit Undo.
- Manueller Sync, SSE-/Sync-Status, Remote Create/Update/Delete über die lokale Staging-CalDAV-Admin-API.
- Dirty-Local-vs-Remote-Changed-Konflikt, Konfliktliste, Konfliktdetail, manuelle Vorschau und Konfliktauflösung.
- Desktop-, Tablet- und schmale Mobile-Breiten inklusive Screenshots, Overflow-Prüfungen und Dialog-/Panel-Containment.

WebKit-Befunde werden wie normale Browser-QA-Befunde behandelt: Produktrelevante Fehler bekommen ein GitHub-Issue mit sanitisierten Schritten, Browsername `webkit`, Betriebssystem, Commit-SHA, relevanter Route und nicht-sensiblen Artefaktpfaden. Keine privaten Screenshots, CalDAV-Zugänge, Session-/CSRF-Werte oder Task-Inhalte hochladen.

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

## WebKit Setup-Hinweise

Wenn WebKit lokal nicht startet und Playwright eine Meldung wie `Host system is missing dependencies to run browsers` mit fehlenden Bibliotheken wie `libgtk-4.so.1`, `libicudata.so`, `libgstreamer`, `libsecret-1.so.0` oder `libwoff2dec.so` zeigt, ist das ein lokales Host-Setup-Problem, kein Caldo-Testfehler.

Führe dann aus:

```bash
sudo npx playwright install-deps webkit
npx playwright install webkit
```

Wenn sowohl Chromium als auch WebKit fehlen oder eine frische QA-Maschine eingerichtet wird:

```bash
sudo npx playwright install-deps chromium webkit
npx playwright install chromium webkit
```

In CI wird dieser Schritt durch `npx playwright install --with-deps chromium webkit` erledigt. In Codex-Umgebungen werden fehlende Systembibliotheken nicht per Docker oder Paketmanager nachinstalliert; solche lokalen WebKit-Startfehler werden im PR als Umgebungslimit dokumentiert.

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
- Quick Add mit Overlay, Preview, Korrektur und Speichern
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

## Tastatur, Fokus und Accessibility

Der MVP-Smoke prüft repräsentative Keyboard- und Accessibility-Invarianten ohne zusätzliche Runtime- oder npm-Abhängigkeiten:

- Globale Aktionen, Navigation und Aufgabenaktionen haben sichtbare Fokusindikatoren.
- Quick-Add-Overlay, Hilfedialog, Mobile-Navigation und Task-Detaildialog halten Fokus im offenen Dialog.
- Dialoge geben Fokus beim Schließen an den auslösenden Button oder Link zurück.
- Formfehler haben `role="alert"`, eine stabile `id`, `aria-describedby` am Formular und am betroffenen Control sowie `aria-invalid="true"` am fehlerhaften Control.
- Icon-Buttons verwenden explizite `aria-label`-Werte statt nur sichtbarer Symbole.

Bei neuen interaktiven Elementen muss entweder eine bestehende Caldo-Komponentenklasse mit Fokuszustand verwendet oder eine passende `:focus-visible`-Regel ergänzt werden. Neue Dialoge müssen dieselbe Fokusfalle und Fokus-Rückgabe wie die bestehenden Dialoge erfüllen.

## Performance

`npm run test:e2e:performance` ist ein opt-in Chromium-Lauf fuer Story 28.5. Er erzeugt synthetische Aufgaben, Projekte und Labels in der lokalen Stage-CalDAV-Instanz, durchlaeuft Setup/Import und misst Navigation, Suche, Live-Suche und Manual-Sync-Responsiveness. Die Ergebnisdateien liegen unter `test-results/performance/` und werden nicht committed.

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
