# Story 1.2 — Startup-Lock und Single-Process-Betrieb

## Name
Story 1.2 — Startup-Lock und Single-Process-Betrieb

## Ziel
Es läuft höchstens ein Caldo-Prozess pro Datenverzeichnis.

## Eingangszustand
Mehrere Prozesse könnten dieselbe SQLite-Datei verwenden.

## Ausgangszustand
Ein Advisory Startup-Lock verhindert parallele Prozesse.

## Akzeptanzkriterien
* Vor DB-Migrationen wird ein Startup-Lock erworben.
* Ein zweiter Prozess mit demselben Datenpfad startet nicht.
* Der Lock bleibt bis zum Prozessende gehalten.
* Ein nicht erwerbbarer Lock führt zu hartem Startabbruch.
* Es gibt keinen Cluster- oder Distributed-Lock-Mechanismus.

---
