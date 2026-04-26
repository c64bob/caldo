# Story 7.3 — Aufgaben erledigen und wieder öffnen

## Name
Story 7.3 — Aufgaben erledigen und wieder öffnen

## Ziel
Aufgabenstatus wird CalDAV-kompatibel geändert.

## Eingangszustand
Eine offene oder erledigte Aufgabe existiert.

## Ausgangszustand
Der Status ist lokal und remote konsistent.

## Akzeptanzkriterien
* Erledigen setzt VTODO-Completed/Status.
* Wieder öffnen entfernt oder aktualisiert Completed/Status konsistent.
* `expected_version` wird geprüft.
* Änderung ist Undo-fähig.
* Erledigte Aufgaben sind standardmäßig ausgeblendet.
* Bei CalDAV-Fehler wird der Nutzer informiert.
* Wiederkehrende Aufgaben behalten ihre RRULE unverändert.

---
