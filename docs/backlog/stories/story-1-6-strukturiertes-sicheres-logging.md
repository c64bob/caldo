# Story 1.6 — Strukturiertes sicheres Logging

## Name
Story 1.6 — Strukturiertes sicheres Logging

## Ziel
Betriebslogs sind nützlich, aber frei von sensiblen Inhalten.

## Eingangszustand
Es gibt keine zentrale Logging-Policy.

## Ausgangszustand
Logs sind strukturiert, korrelierbar und zentral maskiert.

## Akzeptanzkriterien
* Production-Logs sind JSON.
* Development-Logs sind lesbarer Text.
* Jeder HTTP-Request erhält eine `request_id`.
* Jeder Sync-Lauf erhält eine `sync_run_id`.
* Task-Titel, Beschreibungen, Raw-VTODO, Credentials, Tokens, Session-IDs und Auth-Header-Werte werden nie geloggt.
* Maskierung erfolgt zentral.
* Fehlertypen werden ohne nutzdatenhaltige Messages geloggt.

---
