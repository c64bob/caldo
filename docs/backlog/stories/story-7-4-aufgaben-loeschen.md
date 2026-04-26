# Story 7.4 — Aufgaben löschen

## Name
Story 7.4 — Aufgaben löschen

## Ziel
Aufgaben können endgültig gelöscht werden.

## Eingangszustand
Eine Aufgabe existiert lokal und remote.

## Ausgangszustand
Die Aufgabe ist nach erfolgreichem CalDAV-Delete lokal entfernt.

## Akzeptanzkriterien
* Vor Löschen erscheint eine Bestätigung.
* `expected_version` wird geprüft.
* Vor Löschen wird ein Undo-Snapshot erstellt.
* CalDAV-DELETE läuft synchron.
* Lokale Task-Zeile wird erst nach erfolgreichem DELETE entfernt.
* `404 Not Found` beim DELETE gilt als Erfolg.
* Es gibt keinen Papierkorb.
* Löschkonflikte können später erkannt werden.

---
