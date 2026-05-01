# Story 21.3 — Drag-and-drop zwischen Projekten

## Name
Story 21.3 — Drag-and-drop zwischen Projekten

## Ziel
Aufgaben können per Drag-and-drop zwischen Projekten verschoben werden.

## Eingangszustand
Projektwechsel ist möglich, aber kein Drag-and-drop-Flow ist explizit im Story-Backlog abgebildet.

## Ausgangszustand
Projektwechsel per Drag-and-drop ist nutzbar und folgt denselben Datenkonsistenzregeln wie andere Mutationen.

## Akzeptanzkriterien
* Eine Aufgabe kann aus einer Projektliste in ein anderes Projekt verschoben werden.
* Der Projektwechsel wird wie jede Mutation sofort zu CalDAV geschrieben.
* Die Änderung gilt erst nach erfolgreichem CalDAV-Write als gespeichert.
* Bei fehlgeschlagenem Write wird ein sichtbarer Fehler angezeigt und kein stiller lokaler Endzustand erzeugt.

---
