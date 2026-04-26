# Story 11.1 — Optimistic Locking für mutierende Requests

## Name
Story 11.1 — Optimistic Locking für mutierende Requests

## Ziel
Veraltete Tabs überschreiben keine neueren Daten.

## Eingangszustand
Mutierende Requests könnten stale Daten speichern.

## Ausgangszustand
Alle relevanten Mutationen prüfen `expected_version`.

## Akzeptanzkriterien
* Task-mutierende Requests enthalten immer `expected_version`.
* Projekt- und Filteränderungen nutzen ebenfalls Versionen.
* Bei Versionsgleichheit darf verarbeitet werden.
* Bei Versionsabweichung wird nicht gespeichert.
* Nutzer erhält Konflikt- oder Aktualisierungshinweis.
* `etag` wird nie als UI-Version genutzt.
* `server_version` wird nie als CalDAV-ETag genutzt.

---
