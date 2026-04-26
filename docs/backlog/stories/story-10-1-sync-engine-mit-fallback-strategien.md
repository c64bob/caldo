# Story 10.1 — Sync Engine mit Fallback-Strategien

## Name
Story 10.1 — Sync Engine mit Fallback-Strategien

## Ziel
Remote-Änderungen werden robust aus CalDAV importiert.

## Eingangszustand
Es gibt keinen normalen Sync-Lauf.

## Ausgangszustand
Sync nutzt WebDAV Sync, CTag/ETag oder Full-Scan je Projekt.

## Akzeptanzkriterien
* Pro Projekt wird die aktuelle Sync-Strategie gelesen.
* WebDAV Sync wird bevorzugt.
* Bei Nichtunterstützung fällt Sync auf CTag/ETag zurück.
* Bei weiterer Unzuverlässigkeit fällt Sync auf Full-Scan zurück.
* Effektive Strategie wird pro Projekt gespeichert.
* Remote-Fetching und Parsing passieren außerhalb des Write-Mutex.
* DB-Mutationen passieren in Chunks mit Write-Mutex.
* Importierte Remote-Änderungen erhöhen `server_version`.

---
