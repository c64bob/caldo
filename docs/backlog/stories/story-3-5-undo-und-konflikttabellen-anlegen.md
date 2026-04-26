# Story 3.5 — Undo- und Konflikttabellen anlegen

## Name
Story 3.5 — Undo- und Konflikttabellen anlegen

## Ziel
Spätere Undo- und Konfliktlogik hat eine sichere Datenbasis.

## Eingangszustand
Vorversionen werden nicht persistiert.

## Ausgangszustand
Undo-Snapshots und Konflikte sind als eigene Entitäten speicherbar.

## Akzeptanzkriterien
* Undo-Snapshot speichert Session, Tab, Task, Aktion, VTODO-Snapshot und Ablaufzeit.
* Pro `(session_id, tab_id)` gibt es maximal einen Undo-Snapshot.
* Konflikte speichern Base-, lokale, Remote- und gelöste VTODO-Versionen.
* Konflikte können ungelöst oder gelöst sein.
* Gelöste Konflikte können später bereinigt werden.
* Konflikte sind nicht nur ein Statusfeld auf Tasks.

---
