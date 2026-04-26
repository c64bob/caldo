# Story 8.2 — Projekt umbenennen

## Name
Story 8.2 — Projekt umbenennen

## Ziel
Projektname und CalDAV-Kalendername bleiben konsistent.

## Eingangszustand
Ein Projekt existiert.

## Ausgangszustand
Remote-Kalender und lokales Projekt sind umbenannt.

## Akzeptanzkriterien
* Request enthält `expected_version`.
* Remote-Kalender wird zuerst umbenannt.
* Lokales Projekt wird erst nach Remote-Erfolg aktualisiert.
* Denormalisierte `project_name`-Felder betroffener Tasks werden aktualisiert.
* Suchindex wird für betroffene Tasks aktualisiert.
* Fehler werden ohne lokale Teilumbenennung angezeigt.

---
