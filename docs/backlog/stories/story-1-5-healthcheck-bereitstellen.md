# Story 1.5 — Healthcheck bereitstellen

## Name
Story 1.5 — Healthcheck bereitstellen

## Ziel
Deployments können erkennen, ob der Prozess läuft.

## Eingangszustand
Es gibt keinen Liveness-Endpunkt.

## Ausgangszustand
`GET /health` ist ohne Auth erreichbar.

## Akzeptanzkriterien
* `GET /health` antwortet ohne Reverse-Proxy-Auth.
* Der Healthcheck prüft nur Prozess-Liveness.
* Der Healthcheck prüft nicht CalDAV.
* Der Healthcheck prüft nicht vollständige DB-Integrität.
* Bei fehlgeschlagenem Start ist der Healthcheck nicht verfügbar.

---
