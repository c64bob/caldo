# Story 8.3 — Projekt löschen

## Name
Story 8.3 — Projekt löschen

## Ziel
Ein Projekt kann nach starker Bestätigung endgültig gelöscht werden.

## Eingangszustand
Ein Projekt mit oder ohne Tasks existiert.

## Ausgangszustand
Remote-Kalender, lokales Projekt und zugehörige lokale Tasks sind entfernt.

## Akzeptanzkriterien
* Bestätigung zeigt Projektname und Anzahl betroffener Tasks.
* Starke Bestätigung ist erforderlich.
* CalDAV-Kalender wird gelöscht.
* Es werden keine einzelnen Task-DELETEs für Projektlöschung gesendet.
* Lokales Projekt und lokale Tasks werden nach Remote-Erfolg gelöscht.
* FTS-Einträge werden entfernt.
* War es das Default-Projekt, wird `default_project_id=NULL`.
* Neue Task-Erstellung ist danach blockiert, bis ein neues Default-Projekt gewählt ist.

---
