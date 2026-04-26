# Story 3.3 — Aufgabenmodell persistieren

## Name
Story 3.3 — Aufgabenmodell persistieren

## Ziel
VTODO-Aufgaben können lokal vollständig genug gespeichert werden.

## Eingangszustand
Es gibt keine lokale Aufgabenpersistenz.

## Ausgangszustand
Tasks speichern normalisierte Felder, Raw-VTODO, Sync-Status und Versionen.

## Akzeptanzkriterien
* Task enthält Projektbezug, UID, HREF und ETag.
* Task enthält `server_version`.
* Task enthält Titel, Beschreibung, Status, Fälligkeit, Priorität und RRULE.
* Task enthält `raw_vtodo`.
* Task kann `base_vtodo` speichern.
* Task enthält `sync_status`.
* Task kann Parent-Bezug für Unteraufgaben speichern.
* Denormalisierte Suchfelder für Projekt- und Labelnamen sind vorhanden.

---
