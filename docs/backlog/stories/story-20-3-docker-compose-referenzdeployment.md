# Story 20.3 — Docker-Compose-Referenzdeployment

## Name
Story 20.3 — Docker-Compose-Referenzdeployment

## Ziel
Self-Hoster können Caldo nachvollziehbar starten.

## Eingangszustand
Es gibt keine Referenzkonfiguration.

## Ausgangszustand
Docker Compose beschreibt Standardbetrieb hinter Reverse Proxy.

## Akzeptanzkriterien
* Compose nutzt Volume für `/data`.
* Pflicht-Environment-Variablen sind dokumentiert.
* Port wird lokal gebunden.
* Healthcheck ruft `/health` auf.
* Restart-Policy ist `on-failure:3`.
* `unless-stopped` wird nicht verwendet.
* Dokumentation erklärt, dass `BASE_URL` auch hinter internem HTTP-Proxy `https://` enthalten muss.

---
