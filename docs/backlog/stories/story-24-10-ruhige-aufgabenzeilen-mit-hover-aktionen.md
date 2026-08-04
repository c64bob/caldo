# Story 24.10 — Ruhige Aufgabenzeilen mit Hover-Aktionen

## Name
Story 24.10 — Ruhige Aufgabenzeilen mit Hover-Aktionen

## Status
Umgesetzt

## Ziel
Aufgabenzeilen wirken beim Scannen ruhig, weil reine Bedien-Affordanzen ohne Informationswert erst bei Hover oder Fokus erscheinen.

## Eingangszustand
Jede Aufgabenzeile zeigt dauerhaft die Mehrfachauswahl-Checkbox, leere Meta-Editoren ("Labels", "Keine Priorität") und die Auswahl-Markierung der Priorität. Die Liste liest sich dadurch wie ein Formular statt wie eine scanbare Aufgabenliste.

## Ausgangszustand
Aufgabenzeilen zeigen dauerhaft nur Inhalte mit Informationswert. Bedienelemente ohne gesetzten Wert erscheinen bei Hover, Tastaturfokus oder aktiver Mehrfachauswahl, ohne dass sich das Layout der Zeile verschiebt.

## Akzeptanzkriterien
* Die Mehrfachauswahl-Checkbox ist nur bei Hover, Fokus innerhalb der Zeile oder aktiver Mehrfachauswahl sichtbar.
* Bei aktiver Mehrfachauswahl sind die Checkboxen aller Zeilen sichtbar.
* Leere Meta-Editoren (Labels ohne Labels, Priorität ohne Priorität) erscheinen erst bei Hover oder Fokus innerhalb der Zeile.
* Gesetzte Werte (Priorität, Labels, Projekt, Fälligkeit) bleiben dauerhaft sichtbar.
* Das Ein- und Ausblenden verursacht keine Layoutverschiebung in der Zeile.
* Auf Geräten ohne Hover bleiben alle Bedienelemente dauerhaft sichtbar.
* Tastaturbedienung und bestehende Inline-Bearbeitung bleiben unverändert funktionsfähig.

---
