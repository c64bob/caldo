# Story 5.4 — Initialimport ausführen

## Name
Story 5.4 — Initialimport ausführen

## Ziel
Bestehende VTODOs werden vor Nutzung importiert.

## Eingangszustand
Kalenderauswahl und Default-Projekt sind abgeschlossen.

## Ausgangszustand
Alle ausgewählten Kalender sind initial importiert.

## Akzeptanzkriterien
* Initialimport läuft über alle ausgewählten Kalender.
* Import verwendet Full-Scan-Modus.
* Importierte VTODOs werden als `synced` übernommen.
* `base_vtodo = raw_vtodo`.
* Es wird keine Konfliktbehandlung ausgeführt.
* Normalisierte Kernfelder werden aufgebaut.
* FTS-Indexdaten werden vorbereitet.
* Fortschritt wird über Setup-SSE gemeldet.
* Setup-SSE sendet keine normalen Task-/Sync-Events.

---
