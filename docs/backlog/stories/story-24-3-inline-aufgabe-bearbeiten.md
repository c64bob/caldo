# Story 24.3 — Inline-Aufgabe bearbeiten

## Name
Story 24.3 — Inline-Aufgabe bearbeiten

## Status
Umgesetzt

## Ziel
Haeufige Aufgabenfelder koennen direkt in der Liste bearbeitet werden.

## Eingangszustand
Edit-Routen existieren, aber die sichtbare Bearbeitung in der Aufgabenliste ist nicht ausreichend ausgearbeitet.

## Ausgangszustand
Titel, Beschreibung, Faelligkeit, Prioritaet, Projekt und Labels koennen in der Liste sicher geaendert werden.

## Akzeptanzkriterien
* Inline-Editoren speichern und verwerfen eindeutig.
* Fehlgeschlagene Writes lassen die lokale Aufgabe nicht erfolgreich aussehen.
* Gleichzeitige Remote-Aenderungen fuehren zu sichtbaren Konfliktzustaenden.
* Die Bearbeitung bleibt auch bei langen Texten layoutstabil.

---
