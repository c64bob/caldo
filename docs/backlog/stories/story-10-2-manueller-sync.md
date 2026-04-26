# Story 10.2 — Manueller Sync

## Name
Story 10.2 — Manueller Sync

## Ziel
Der Nutzer kann jederzeit einen Full-Sync starten.

## Eingangszustand
Remote-Änderungen kommen nur über Initialimport oder Writes herein.

## Ausgangszustand
Ein manueller Sync kann gestartet und überwacht werden.

## Akzeptanzkriterien
* Es gibt einen sichtbaren manuellen Sync-Zugriff.
* Ein laufender Sync verhindert parallele Full-Syncs.
* Bei laufendem Sync wird kein zweiter Lauf queued.
* UI zeigt aktuellen Sync-Status.
* UI zeigt letzten erfolgreichen Sync-Zeitpunkt.
* Abschluss oder Fehler wird sichtbar gemeldet.
* SSE kann Sync-Status verteilen.

---
