# Story 18.3 — Anhänge und unbekannte Felder anzeigen/erhalten

## Name
Story 18.3 — Anhänge und unbekannte Felder anzeigen/erhalten

## Ziel
Nicht aktiv unterstützte VTODO-Inhalte bleiben erhalten und teils sichtbar.

## Eingangszustand
Anhänge und unbekannte Properties sind nur Raw-Daten.

## Ausgangszustand
Anhänge werden read-only angezeigt, unbekannte Felder bleiben erhalten.

## Akzeptanzkriterien
* `ATTACH`-Properties bleiben bei Bearbeitung erhalten.
* Externe ATTACH-URLs werden als Links angezeigt.
* Externe Links öffnen mit `rel="noopener noreferrer"`.
* Inline-/Binary-Anhänge werden als vorhanden angezeigt, aber nicht gerendert.
* Keine Upload-, Entfernen- oder Bearbeiten-Funktion für Anhänge.
* Unbekannte Properties werden nicht entfernt.

---
