# Story 16.1 — Labels in Aufgaben bearbeiten

## Name
Story 16.1 — Labels in Aufgaben bearbeiten

## Ziel
Aufgaben können projektübergreifend organisiert werden.

## Eingangszustand
Labels sind nur als Datenmodell vorhanden.

## Ausgangszustand
Labels können in UI und VTODO geändert werden.

## Akzeptanzkriterien
* Nutzer kann Labels an einer Aufgabe setzen und entfernen.
* Neue Labels werden automatisch lokal angelegt.
* Labels werden als VTODO `CATEGORIES` geschrieben.
* Labeländerung ist Undo-fähig.
* Labeländerung prüft `expected_version`.
* Suche und Filter berücksichtigen aktualisierte Labels.

---
