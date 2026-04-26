# Story 18.2 — Komplexe RRULEs erhalten

## Name
Story 18.2 — Komplexe RRULEs erhalten

## Ziel
Nicht unterstützte Wiederholungen gehen nicht verloren.

## Eingangszustand
Komplexe RRULEs könnten versehentlich überschrieben werden.

## Ausgangszustand
Komplexe RRULEs sind read-only sichtbar und bleiben erhalten.

## Akzeptanzkriterien
* Komplexe RRULEs werden erkannt.
* UI zeigt Badge „Komplexe Wiederholung – wird erhalten, kann nicht bearbeitet werden“.
* Wiederholungseditor ist deaktiviert.
* Andere Kernfelder bleiben bearbeitbar.
* Bearbeitung anderer Felder erhält RRULE unverändert.
* Erledigen verändert RRULE nicht.
* Es wird keine nächste Instanz lokal erzeugt.

---
