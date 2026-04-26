# Story 15.2 — Gespeicherte Filter verwalten

## Name
Story 15.2 — Gespeicherte Filter verwalten

## Ziel
Der Nutzer kann eigene Aufgabenansichten speichern.

## Eingangszustand
Filterqueries sind nicht persistierbar.

## Ausgangszustand
Filter können angelegt, geändert, gelöscht und favorisiert werden.

## Akzeptanzkriterien
* Filter haben Name und Query.
* Filter werden lokal gespeichert.
* Filter werden nicht zu CalDAV synchronisiert.
* Filter können favorisiert werden.
* Filteränderungen nutzen `server_version`.
* Syntaxfehler gespeicherter Queries führen zur Laufzeit zu leerer Ergebnisliste, nicht zu hartem Fehler.
* Favorisierte Filter erscheinen in der Navigation.

---
