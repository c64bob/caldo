# Story 27.3 — Feldweise Konfliktloesung

## Name
Story 27.3 — Feldweise Konfliktloesung

## Status
Umgesetzt

## Ziel
Nutzer koennen pro Feld zwischen lokaler, entfernter oder manueller Version waehlen.

## Eingangszustand
Backend-nahe Konfliktloesung existiert teilweise, aber die UI fuer Feldwahl fehlt.

## Ausgangszustand
Eine gemischte Loesung kann kontrolliert erstellt, remote geschrieben und lokal als geloest markiert werden.

## Akzeptanzkriterien
* Jedes konfliktfaehige Feld bietet lokale, entfernte und manuelle Auswahl, soweit fachlich sinnvoll.
* Die entstehende Vorschau ist vor dem Speichern sichtbar.
* Ein erneuter Versionskonflikt erzeugt keinen Retry und bleibt sichtbar.
* Nicht betroffene unbekannte Daten bleiben erhalten.

---
