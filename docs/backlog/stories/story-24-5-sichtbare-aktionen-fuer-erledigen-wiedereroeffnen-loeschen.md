# Story 24.5 — Sichtbare Aktionen fuer Erledigen, Wiedereroeffnen und Loeschen

## Name
Story 24.5 — Sichtbare Aktionen fuer Erledigen, Wiedereroeffnen und Loeschen

## Status
Umgesetzt

## Ziel
Kernaktionen an Aufgaben werden sichtbar, konsistent und fehlertolerant bedienbar.

## Eingangszustand
Die Routen existieren, aber die UI fuer alle Kernaktionen ist nicht ausreichend sichtbar und einheitlich.

## Ausgangszustand
Nutzer koennen Aufgaben erledigen, wiedereroeffnen und loeschen, ohne versteckte HTTP-Flows zu kennen.

## Akzeptanzkriterien
* Completion und Reopen zeigen einen pending Zustand und bestaetigen erst nach Erfolg.
* Loeschen ist gegen versehentliche Ausfuehrung ausreichend abgesichert.
* Ein erfolgreicher Delete erzeugt eine passende Undo-Moeglichkeit.
* Fehler bleiben an der betroffenen Aufgabe oder im globalen Fehlerbereich sichtbar.

---
