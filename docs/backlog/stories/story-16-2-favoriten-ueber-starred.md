# Story 16.2 — Favoriten über STARRED

## Name
Story 16.2 — Favoriten über STARRED

## Ziel
Favoriten sind lokal sichtbar und CalDAV-kompatibel synchronisiert.

## Eingangszustand
Es gibt keine Favoritenfunktion.

## Ausgangszustand
Favorit entspricht Kategorie `STARRED`.

## Akzeptanzkriterien
* `STARRED` aus CalDAV wird als Favorit importiert.
* Favorit setzen schreibt `STARRED` in VTODO-Categories.
* Favorit entfernen entfernt nur die Favoritenbedeutung.
* Andere Kategorien bleiben erhalten.
* Favoritenansicht zeigt favorisierte aktive Aufgaben.
* Favoritenstatus ist per Optimistic Locking geschützt.

---
