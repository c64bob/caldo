# Story 20.2 — Docker-Image

## Name
Story 20.2 — Docker-Image

## Ziel
Caldo kann als Container betrieben werden.

## Eingangszustand
Es gibt kein Container-Artefakt.

## Ausgangszustand
Ein Runtime-Image enthält Binary und statische Assets.

## Akzeptanzkriterien
* Multi-Stage-Build ist möglich.
* Runtime-Image enthält keine Go-Toolchain.
* Runtime läuft als Non-root-User.
* `/data` ist persistenter Datenpfad.
* Port `8080` ist einziger Listener.
* Healthcheck-fähiges Tool ist im Image vorhanden.

---
