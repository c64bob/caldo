# Story 1.3 — SQLite öffnen und robust konfigurieren

## Name
Story 1.3 — SQLite öffnen und robust konfigurieren

## Ziel
SQLite ist als lokale MVP-Datenbank stabil vorbereitet.

## Eingangszustand
Es existiert keine initialisierte DB-Verbindung.

## Ausgangszustand
SQLite läuft mit WAL, sinnvollen PRAGMAs und einem kontrollierten Schreibpfad.

## Akzeptanzkriterien
* SQLite wird am konfigurierten Pfad geöffnet.
* `journal_mode=WAL` ist gesetzt.
* `synchronous=NORMAL` ist gesetzt.
* `busy_timeout=5000` ist gesetzt.
* Die DB nutzt maximal eine offene Verbindung.
* Alle späteren DB-Writes können über einen globalen Write-Mutex laufen.

---
