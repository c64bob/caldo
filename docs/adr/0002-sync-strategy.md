# ADR 0002: Sync Strategy

## Status
Accepted

## Kontext
Phase 5 verlangt robuste Synchronisation ohne eigene Task-Datenbank. Der CalDAV-Server bleibt
Single Source of Truth; lokal werden nur Sync-Metadaten pro Collection gehalten.

## Entscheidung
- Caldo führt Sync pro Collection aus.
- Primärmodus ist `webdav-sync`.
- Wenn kein verwendbarer Sync-Token vorliegt, wird automatisch auf `etag-fallback` gewechselt.
- Persistiert werden nur `sync_token`, ETag-Digest, Anzahl Ressourcen, Modus und Fehlerzeitpunkt.

## Konsequenzen
- Keine Duplikation des Task-Modells in SQLite.
- Manuelle Synchronisierung (`POST /api/sync/now`) und optionaler Hintergrundjob sind möglich.
- Konfliktauflösung bleibt weiter über ETag/If-Match in CRUD-Operationen.
