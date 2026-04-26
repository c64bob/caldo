# Story 14.2 — Projekt-, Label- und Prioritätstokens

## Name
Story 14.2 — Projekt-, Label- und Prioritätstokens

## Ziel
Todoist-nahe Schnellsyntax ist nutzbar.

## Eingangszustand
Quick Add erkennt nur Freitext.

## Ausgangszustand
`#`, `@` und `!`-Tokens werden erkannt und aufgelöst.

## Akzeptanzkriterien
* `#Projekt` setzt das Projekt.
* Unbekanntes Projekt wird nicht still ignoriert.
* UI zeigt Projektvorschlag oder Anlageoption.
* Neues Projekt erzeugt einen CalDAV-Kalender.
* `@Label` setzt Labels.
* Neues Label wird automatisch angelegt.
* `!high`, `!medium`, `!low`, `!1`, `!2`, `!3` werden erkannt.
* Gemeinsame Tokenregeln divergieren nicht von Suche/Filter.

---
