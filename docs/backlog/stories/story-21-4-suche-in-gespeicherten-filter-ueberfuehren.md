# Story 21.4 — Suche in gespeicherten Filter überführen

## Name
Story 21.4 — Suche in gespeicherten Filter überführen

## Ziel
Eindeutig interpretierbare Suchanfragen können als gespeicherte Filter übernommen werden.

## Eingangszustand
Globale Suche und Filterverwaltung existieren getrennt, aber der Übergang Suche → Filter ist nicht explizit abgebildet.

## Ausgangszustand
Nutzer können geeignete Suchabfragen direkt als gespeicherten Filter anlegen.

## Akzeptanzkriterien
* Für Suchanfragen, die eindeutig in die unterstützte Filter-Query überführbar sind, wird „als Filter speichern“ angeboten.
* Der gespeicherte Filter enthält Name und überführte Query.
* Nicht eindeutig überführbare Suchanfragen bieten die Übernahme nicht an.
* Gespeicherte Filter erscheinen anschließend in der regulären Filterverwaltung.

---
