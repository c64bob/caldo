# Story 9.1 — FTS5-Suchindex aufbauen

## Name
Story 9.1 — FTS5-Suchindex aufbauen

## Ziel
Globale Suche ist performant und konsistent.

## Eingangszustand
Tasks können nicht freitextbasiert durchsucht werden.

## Ausgangszustand
FTS5 indexiert aktive Aufgabenfelder.

## Akzeptanzkriterien
* FTS5 indexiert Titel, Beschreibung, Labelnamen und Projektnamen.
* Trigger halten strukturelle Konsistenz bei Insert, Update und Delete.
* Go-Layer pflegt denormalisierte Suchfelder.
* Erledigte Aufgaben werden standardmäßig ausgeschlossen.
* Undo-Snapshots, Konfliktversionen und Historie werden nicht indexiert.
* Umlaut-/Diakritik- und Prefix-Suche sind getestet.

---
