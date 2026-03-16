# AGENTS.md

Diese Datei definiert Arbeitsregeln für Agenten im Repository `caldo`.

## Zielbild
- Caldo ist ein **Thin CalDAV Client**.
- **Single Source of Truth** für Tasks bleibt der CalDAV-Server (z. B. Nextcloud), nicht SQLite.
- SQLite speichert nur:
  - Präferenzen
  - verschlüsselte DAV-Credentials
  - Sync-Metadaten

## Allgemeine Regeln
1. **Keine eigene Task-Datenbank einführen** (keine Duplikation des CalDAV-Datenmodells).
2. Änderungen klein, nachvollziehbar und fokussiert halten.
3. Bei Architektur-/Verhaltensänderungen immer Doku mitpflegen (`README.md` oder `docs/`).
4. Secrets niemals loggen oder im Klartext speichern.
5. Neue Konfiguration immer mit sinnvollen Defaults und Doku ergänzen.

## Backend (Go)
- Bevorzugte Struktur:
  - `cmd/` für Startpunkte
  - `internal/` für App-Logik
  - `internal/caldav` für DAV-Protokollzugriffe
  - `internal/store/sqlite` nur für Metadaten
- Business-Logik nicht in HTTP-Handlern verstecken; Handler sollen dünn bleiben.
- Fehler mit Kontext zurückgeben, aber ohne sensitive Daten.

## CalDAV/Sync-Regeln
- ETag/If-Match für updates/deletes nutzen.
- `412 Precondition Failed` als normaler Konfliktfall behandeln (kein stilles Überschreiben).
- WebDAV-Sync nutzen, wenn verfügbar; sonst ETag-basierter Fallback pro Collection.
- Sync-State **pro Collection** verwalten.

## Security
- DAV-Passwörter ausschließlich verschlüsselt (AES-256-GCM) persistieren.
- Master-Key aus Environment oder Secret-File laden.
- Logging redaction aktiv halten (Passwörter, Tokens, Auth-Header).

## Frontend (HTMX + Templates)
- Dichte, tabellarische Task-Ansicht priorisieren.
- HTMX-Responses als Partials halten (keine unnötigen Full-Page Reloads).
- Bei Konflikten/Fehlern klare, nutzerverständliche Meldungen anzeigen.

## Tests & Verifikation
- Nach Änderungen mindestens ausführen:
  - relevante Unit-/Integrationstests (falls vorhanden)
  - schnelle statische Checks (z. B. `go test ./...` sobald Go-Code vorhanden ist)
- Bei reinen Doku-Änderungen: Konsistenzprüfung der betroffenen Dateien.

## Pull Requests
- PR-Beschreibung soll enthalten:
  - Motivation
  - Was wurde geändert
  - Wie wurde validiert
- Wenn bewusst etwas **nicht** umgesetzt wurde, kurz begründen.

## Out of Scope für v1.0
- RRULE/Recurrence-Implementierung
- Komplexe Feld-Merge-Strategien bei Konflikten (stattdessen Reload/Overwrite)
