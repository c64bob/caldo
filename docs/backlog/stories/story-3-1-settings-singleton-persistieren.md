# Story 3.1 — Settings-Singleton persistieren

## Name
Story 3.1 — Settings-Singleton persistieren

## Ziel
Setup-, CalDAV-, Sync- und UI-Einstellungen haben eine zentrale Persistenz.

## Eingangszustand
Es gibt keine persistierten Einstellungen.

## Ausgangszustand
Eine Singleton-Settings-Zeile steuert Setup und Normalbetrieb.

## Akzeptanzkriterien
* Es existiert genau eine Settings-Zeile mit `id='default'`.
* `setup_complete` startet bei `false`.
* `setup_step` startet bei `caldav`.
* Sync-Intervall defaultet auf 15 Minuten.
* UI-Sprache defaultet auf Deutsch.
* Dark Mode defaultet auf Systempräferenz.
* `default_project_id` darf vor Setup-Abschluss leer sein.

---
