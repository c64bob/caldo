# Story 12.1 — Undo-Snapshot bei Undo-fähigen Aktionen

## Name
Story 12.1 — Undo-Snapshot bei Undo-fähigen Aktionen

## Ziel
Die letzte Undo-fähige Aktion pro Tab kann rückgängig gemacht werden.

## Eingangszustand
Vorherige Task-Zustände werden nicht gespeichert.

## Ausgangszustand
Vor Änderung wird ein tab-lokaler Snapshot gespeichert.

## Akzeptanzkriterien
* Snapshot und ursprüngliche Änderung liegen in derselben DB-Transaktion.
* Pro `(session_id, tab_id)` existiert maximal ein Snapshot.
* Neuer Snapshot ersetzt vorherigen Snapshot desselben Tabs.
* Snapshot enthält Raw-VTODO, normalisierte Felder und `etag_at_snapshot`.
* Snapshot läuft nach 5 Minuten ab.
* Reload im selben Tab erhält Undo-Verfügbarkeit.

---
