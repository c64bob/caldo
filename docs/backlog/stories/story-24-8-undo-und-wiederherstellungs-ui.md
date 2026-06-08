# Story 24.8 — Undo- und Wiederherstellungs-UI

## Name
Story 24.8 — Undo- und Wiederherstellungs-UI

## Status
Offen

## Ziel
Rueckgaengig-Funktionen werden fuer Nutzer sichtbar und zeitlich nachvollziehbar.

## Eingangszustand
Undo-Snapshots und Routen existieren, aber die UI fuehrt den Nutzer nicht konsequent durch Wiederherstellung.

## Ausgangszustand
Nach unterstuetzten Mutationen erscheint eine klare Undo-Moeglichkeit mit Ablaufverhalten.

## Akzeptanzkriterien
* Undo erscheint nur fuer Aktionen, fuer die ein gueltiger Snapshot existiert.
* Ablauf und Ergebnis einer Undo-Aktion werden sichtbar kommuniziert.
* Fehlgeschlagene Undo-Versuche zeigen sichtbare Fehler.
* Mehrere Browser-Tabs koennen den Zustand ohne stille Ueberschreibung aktualisieren.

---
