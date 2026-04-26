# Story 7.2 — Aufgaben bearbeiten

## Name
Story 7.2 — Aufgaben bearbeiten

## Ziel
Kernfelder einer Aufgabe können geändert werden.

## Eingangszustand
Eine synchronisierte Aufgabe existiert.

## Ausgangszustand
Die Änderung ist lokal versioniert und remote gespeichert.

## Akzeptanzkriterien
* Bearbeitbar sind Titel, Beschreibung, Fälligkeit, Priorität, Status, Projekt und Labels.
* Request enthält `expected_version`.
* Bei Versionsabweichung wird nicht gespeichert.
* Vor Änderung wird ein Undo-Snapshot erstellt, sofern die Aktion Undo-fähig ist.
* Änderung wird als `pending` versioniert.
* CalDAV-Write läuft synchron.
* Bei Erfolg wird neuer ETag gespeichert und `sync_status=synced`.
* Bei Fehler bleibt der Fehler sichtbar und die Änderung gilt nicht fachlich gespeichert.

---
