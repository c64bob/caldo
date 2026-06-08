# Story 23.6 — Leere, ladende und fehlerhafte Zustaende

## Name
Story 23.6 — Leere, ladende und fehlerhafte Zustaende

## Status
Offen

## Ziel
Alle Kernansichten zeigen klare Zustaende, wenn keine Daten, noch ladende Daten oder Fehler vorliegen.

## Eingangszustand
Viele Ansichten wirken bei leeren Daten oder Fehlern wie Platzhalter.

## Ausgangszustand
Nutzer erhalten knappe, handlungsorientierte Rueckmeldungen ohne sensible Inhalte.

## Akzeptanzkriterien
* Leere Aufgabenlisten, Projekte, Labels, Filter, Suche und Konflikte haben eigene Empty States.
* Ladezustaende verhindern Layoutspruenge und doppelte Mutationen.
* Fehlerzustaende nennen Aktion und Fehlerklasse, aber keine privaten Inhalte.
* Retry-Aktionen sind nur dort sichtbar, wo ein Retry fachlich erlaubt ist.

---
