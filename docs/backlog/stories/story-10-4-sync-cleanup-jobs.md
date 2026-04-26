# Story 10.4 — Sync-Cleanup-Jobs

## Name
Story 10.4 — Sync-Cleanup-Jobs

## Ziel
Kurzlebige technische Daten werden automatisch bereinigt.

## Eingangszustand
Undo-Snapshots und gelöste Konflikte bleiben unbegrenzt liegen.

## Ausgangszustand
Cleanup läuft regelmäßig im Sync-/Scheduler-Kontext.

## Akzeptanzkriterien
* Abgelaufene Undo-Snapshots werden bei Sync-Läufen gelöscht.
* Gelöste Konflikte älter als 7 Tage werden täglich gelöscht.
* Ungelöste Konflikte werden nie automatisch gelöscht.
* Cleanup läuft über den globalen Write-Mutex.
* Cleanup loggt keine Task-Inhalte.

---
