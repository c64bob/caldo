# Story 11.2 — Globaler SSE-Endpunkt

## Name
Story 11.2 — Globaler SSE-Endpunkt

## Ziel
Offene Tabs werden über relevante Änderungen informiert.

## Eingangszustand
Mehrere Tabs erfahren nichts voneinander.

## Ausgangszustand
`GET /events` verteilt Task-, Projekt-, Sync- und Konflikt-Events.

## Akzeptanzkriterien
* Es gibt genau einen normalen SSE-Endpunkt.
* Jede Verbindung hat eine `connection_id`.
* Events enthalten Typ, Ressource, Version und Origin-Connection.
* Events werden nach DB-Commit gesendet.
* Die auslösende Verbindung erhält ihr Ergebnis primär über die HTTP-Response.
* Andere Verbindungen erhalten Broadcasts.
* Setup-SSE und Normalbetrieb-SSE sind getrennt.

---
