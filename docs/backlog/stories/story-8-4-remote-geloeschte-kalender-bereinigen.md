# Story 8.4 — Remote gelöschte Kalender bereinigen

## Name
Story 8.4 — Remote gelöschte Kalender bereinigen

## Ziel
Remote-Kalenderlöschung wird autoritativ übernommen.

## Eingangszustand
Ein lokal bekanntes Projekt existiert remote nicht mehr.

## Ausgangszustand
Das lokale Projekt und abhängige Daten sind bereinigt.

## Akzeptanzkriterien
* Remote-Kalenderlöschung erzeugt keinen Projektkonflikt.
* Lokales Projekt wird gelöscht.
* Zugehörige Tasks werden gelöscht.
* FTS-Einträge werden gelöscht.
* Undo-Snapshots für betroffene Tasks werden gelöscht.
* Bei pending Tasks wird eine einmalige Warnung angezeigt.
* War es das Default-Projekt, muss der Nutzer ein neues Default-Projekt wählen.

---
