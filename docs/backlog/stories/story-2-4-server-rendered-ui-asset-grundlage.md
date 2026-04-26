# Story 2.4 — Server-rendered UI-Asset-Grundlage

## Name
Story 2.4 — Server-rendered UI-Asset-Grundlage

## Ziel
Die Weboberfläche kann ohne Runtime-CDN ausgeliefert werden.

## Eingangszustand
Es gibt keine definierte Asset-Auslieferung.

## Ausgangszustand
Statische Assets werden lokal, versioniert und CSP-kompatibel ausgeliefert.

## Akzeptanzkriterien
* `/static/*` liefert lokale Assets aus.
* Es wird kein Runtime-CDN verwendet.
* CSS- und JS-Dateien nutzen dateinamenbasiertes Cache-Busting.
* `manifest.json` wird beim Start geladen.
* Fehlendes Manifest führt zu hartem Startabbruch.
* Statische Assets erhalten langfristige Cache-Header.
* CSP erlaubt keine inline Scripts und kein `'unsafe-inline'`.

---
