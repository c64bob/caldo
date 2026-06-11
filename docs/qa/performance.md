# Performance QA

Dieses Dokument macht die Performance-Ziele aus `docs/prd.md` Abschnitt 23 als wiederholbare QA-Checks pruefbar. Die Checks sind fuer Release- oder Pre-Release-QA gedacht, nicht als harte lokale Entwicklerpflicht bei jeder Aenderung.

## Grundregeln

- Nur synthetische Daten verwenden. Keine privaten Aufgaben, echten CalDAV-Zugaenge oder produktiven Nextcloud-Instanzen.
- Gegen lokal gebaute Binaries messen, nicht gegen `go run`, damit Build-Zeit nicht in die Messung faellt.
- Docker ist fuer diese lokale Messung nicht erforderlich.
- Vor jedem Lauf Commit-SHA, Betriebssystem, CPU/RAM, Go-Version, Browser-Version und Datenmenge notieren.
- Jeden Messpunkt mindestens fuenfmal wiederholen. Ein Lauf gilt nur als bestanden, wenn alle verwertbaren Wiederholungen den Zielwert einhalten.
- Offensichtlich durch externe Last gestoerte Laeufe duerfen verworfen werden, muessen aber im Ergebnisprotokoll erwaehnt werden.
- Ergebnisse als lokale QA-Artefakte unter `test-results/performance/<yyyy-mm-dd>/` ablegen; diese Artefakte werden nicht committed.

## Referenzdaten

### 400-Aufgaben-Datensatz

Der 400-Aufgaben-Datensatz ist die PRD-Referenz fuer realistische Nutzung. Er besteht aus:

- einem CalDAV-Kalender/Projekt `Work`
- 400 aktiven VTODOs mit stabilen UIDs `perf-0001` bis `perf-0400`
- kurzen synthetischen Titeln wie `Perf Task 0001`
- gemischten Faelligkeiten: ca. 25 Prozent heute/ueberfaellig, 25 Prozent zukuenftig, 50 Prozent ohne Datum
- optionalen synthetischen Labels, aber keinen Anhaengen

Dieser Datensatz wird fuer Initialimport und inkrementellen Sync verwendet.

### 10.000-Aufgaben-Datensatz

Der 10.000-Aufgaben-Datensatz ist die PRD-Grenzgroesse fuer lokale UI-Ladezeit. Er besteht aus:

- mindestens einem CalDAV-Kalender/Projekt
- 10.000 lokal gespeicherten Tasks
- ca. 80 Prozent aktiven und 20 Prozent erledigten Tasks
- gemischten Faelligkeiten und Labels

Die Daten sollen ueber den normalen Importpfad oder eine dokumentierte Testfixture erzeugt werden, damit VTODO-Parsing, lokale Persistenz, FTS und Navigationszaehler realistisch befuellt sind. Fuer die reine Ladezeitmessung darf die Vorbereitung laenger dauern; fuer 10.000 Aufgaben gibt die PRD keine harte Sync-Dauer vor.

## Messpunkte

| Messpunkt | Zielwert | Referenzdaten | Start | Ende | Bestanden wenn |
|---|---:|---|---|---|---|
| Prozessstart ohne Migrationen | max. 5 s | bereits migrierte, setup-complete DB | Prozessstart des Caldo-Binary | `GET /health` liefert 200 | jeder verwertbare Lauf <= 5 s |
| Erste UI-Ansicht bei grosser Aufgabenmenge | max. 2 s | 10.000 lokale Tasks | Browser-Navigation zu `/today` nach frischem Prozessstart | `domcontentloaded` fuer `/today` | jeder verwertbare Lauf <= 2 s |
| Inkrementeller Sync ohne groessere Aenderungen | max. 10 s | 400 Remote-Tasks, Setup abgeschlossen | manueller Sync-Trigger | Sync-Status meldet abgeschlossenen Lauf per SSE/UI | jeder verwertbare Lauf <= 10 s |
| Initialimport | max. 30 s | 400 Remote-Tasks | `POST /setup/import` angenommen | Setup-Import meldet abgeschlossen und `POST /setup/complete` ist moeglich | jeder verwertbare Lauf <= 30 s |

## Pruefablauf

### 1. Binaries bauen

```bash
mkdir -p .tmp/perf test-results/performance
go build -o .tmp/perf/caldo ./cmd/caldo
go build -o .tmp/perf/stagecaldav ./cmd/stagecaldav
```

### 2. Lokale Stage-CalDAV-Instanz vorbereiten

Eine lokale `cmd/stagecaldav`-Instanz mit Admin-Token starten und den 400-Aufgaben-Datensatz ueber die Admin-API erzeugen. Fuer 10.000-Aufgaben-Messungen denselben Ablauf mit 10.000 synthetischen Tasks verwenden.

Die Stage-Admin-API ist nur fuer lokale QA gedacht. Ergebnisprotokolle duerfen den Admin-Token, CalDAV-Passwoerter oder rohe VTODO-Inhalte nicht enthalten.

### 3. Setup-DB erzeugen

Caldo mit isoliertem `DB_PATH` starten, den normalen Setup-Wizard gegen Stage-CalDAV durchlaufen lassen und den Initialimport abschliessen. Diese DB ist danach die Basis fuer:

- Startzeitmessung ohne ausstehende Migrationen
- erste UI-Ansicht bei 10.000 lokalen Tasks
- inkrementellen Sync nach abgeschlossenem Setup

Vor Startzeitmessungen muss die DB bereits migriert sein. Wenn eine neue Migration ansteht, ist der Lauf kein gueltiger Startzeitlauf fuer Story 21.6.

### 4. Startzeit messen

Caldo beenden, denselben `DB_PATH` erneut mit dem gebauten Binary starten und die Zeit von Prozessstart bis erfolgreichem `GET /health` messen. Die Weboberflaeche muss ohne initial erfolgreichen CalDAV-Sync erreichbar werden.

Ziel: jeder verwertbare Lauf <= 5 Sekunden.

### 5. Erste UI-Ansicht messen

Mit dem 10.000-Aufgaben-Datensatz Caldo frisch starten, im Browser eine neue Session mit Proxy-Auth-Testheader oeffnen und `/today` laden. Gemessen wird die Zeit von Navigation-Start bis `domcontentloaded`.

Ziel: jeder verwertbare Lauf <= 2 Sekunden.

### 6. Initialimport messen

Mit leerer Caldo-DB und Stage-CalDAV mit 400 Remote-Tasks den Setup-Wizard bis zum Import-Schritt durchlaufen. Gemessen wird von angenommenem `POST /setup/import` bis der Import abgeschlossen ist und `POST /setup/complete` erfolgreich in den Normalmodus wechseln kann.

Ziel: jeder verwertbare Lauf <= 30 Sekunden.

### 7. Inkrementellen Sync messen

Mit abgeschlossener Setup-DB und 400 synchronisierten Tasks einen manuellen Sync ausloesen. Vor dem Lauf keine groesseren Remote-Aenderungen erzeugen; null oder eine synthetische Remote-Aenderung ist fuer diesen Check zulaessig. Gemessen wird bis der normale Sync-Status per SSE oder UI als abgeschlossen sichtbar ist.

Ziel: jeder verwertbare Lauf <= 10 Sekunden.

## Ergebnisprotokoll

Pro Messlauf eine kurze Markdown- oder JSON-Datei unter `test-results/performance/<yyyy-mm-dd>/` ablegen:

```text
commit:
os/cpu/ram:
go version:
browser/version:
dataset: 400 | 10000
check:
runs:
  - seconds:
  - seconds:
  - seconds:
  - seconds:
  - seconds:
pass: true | false
notes:
```

Wenn ein Zielwert verfehlt wird, ist das eine Performance-Regressionsindikation. Zielwerte werden nur ueber eine PRD-Aenderung angepasst, nicht im QA-Protokoll.
