# Story 2.1 — Middleware-Stack etablieren

## Name
Story 2.1 — Middleware-Stack etablieren

## Ziel
Alle Requests laufen durch eine konsistente Sicherheits- und Fehlerbehandlung.

## Eingangszustand
Es gibt keinen kanonischen Request-Pfad.

## Ausgangszustand
Middleware-Reihenfolge ist umgesetzt und stabil.

## Akzeptanzkriterien
* `request_id` ist die erste Middleware.
* `recovery` ist die zweite Middleware.
* `safe_logging` läuft vor fachlichen Handlern.
* Security-Header werden für alle relevanten Antworten gesetzt.
* Panics führen zu sicherem 500 ohne interne Details.
* `/health` bleibt von Auth und CSRF ausgenommen.

---
