# Story 13.4 — Konflikt manuell lösen

## Name
Story 13.4 — Konflikt manuell lösen

## Ziel
Der Nutzer kann Konflikte ohne Datenverlust auflösen.

## Eingangszustand
Ein ungelöster Konflikt existiert.

## Ausgangszustand
Eine gewählte Lösung ist lokal und remote gespeichert.

## Akzeptanzkriterien
* Lokale Version übernehmen ist möglich.
* Remote-Version übernehmen ist möglich.
* Felder manuell auswählen ist möglich.
* Beide Versionen behalten ist möglich.
* Mindestens Titel, Beschreibung, Fälligkeit, Priorität, Labels, Projekt, Status und Unteraufgaben sind feldweise lösbar.
* Lösung wird zu CalDAV geschrieben.
* Konflikt erhält `resolved_at` und `resolution`.
* Bei Write-Fehler bleibt Konflikt ungelöst.

---
