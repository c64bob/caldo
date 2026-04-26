# Story 13.1 — Konflikte bei Remote-Sync erkennen

## Name
Story 13.1 — Konflikte bei Remote-Sync erkennen

## Ziel
Lokale und Remote-Änderungen werden verlustfrei verglichen.

## Eingangszustand
Remote-Import könnte lokale Änderungen überschreiben.

## Ausgangszustand
Konflikte werden als eigene Entitäten erzeugt.

## Akzeptanzkriterien
* `base_vtodo`, `local_vtodo` und `remote_vtodo` werden berücksichtigt.
* Bei fehlender Base ist Auto-Merge deaktiviert.
* Feldbasierter Auto-Merge wird nur bei konfliktfreien Änderungen ausgeführt.
* Bei echtem Feldkonflikt entsteht ein Konfliktdatensatz.
* `tasks.sync_status=conflict` blockiert die betroffene Aufgabe.
* Andere Aufgaben synchronisieren weiter.

---
