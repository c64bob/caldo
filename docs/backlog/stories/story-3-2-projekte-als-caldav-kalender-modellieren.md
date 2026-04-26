# Story 3.2 — Projekte als CalDAV-Kalender modellieren

## Name
Story 3.2 — Projekte als CalDAV-Kalender modellieren

## Ziel
CalDAV-Kalender können lokal als Projekte verwaltet werden.

## Eingangszustand
Es gibt keine Projektpersistenz.

## Ausgangszustand
Projekte speichern Kalenderbezug, Sync-Metadaten und Versionen.

## Akzeptanzkriterien
* Projekt enthält Kalender-HREF und Anzeigename.
* Projekt enthält `ctag`, `sync_token` und `sync_strategy`.
* `sync_strategy` unterstützt `webdav_sync`, `ctag`, `fullscan`.
* Projekt enthält `server_version`.
* Ein Projekt kann als Default-Projekt markiert werden.
* Projektänderungen können per Optimistic Locking abgesichert werden.

---
