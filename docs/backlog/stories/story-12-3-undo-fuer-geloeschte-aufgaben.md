# Story 12.3 — Undo für gelöschte Aufgaben

## Name
Story 12.3 — Undo für gelöschte Aufgaben

## Ziel
Eine gelöschte Aufgabe kann aus Snapshot neu erstellt werden.

## Eingangszustand
Eine Aufgabe wurde erfolgreich gelöscht und ein Snapshot existiert.

## Ausgangszustand
Die Aufgabe wird als neue CalDAV-Ressource wiederhergestellt.

## Akzeptanzkriterien
* Undo rekonstruiert Task aus Snapshot.
* VTODO-UID bleibt erhalten, sofern kein Split/Konflikt nötig ist.
* Es wird eine neue CalDAV-Ressource erstellt.
* Bei erfolgreichem Write wird lokale Task-Zeile neu gespeichert.
* Bei Fehler wird der Nutzer informiert.
* Bei zwischenzeitlicher Remote-Änderung entsteht ein Konflikt.

---
