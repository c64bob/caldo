# Story 1.1 — App-Konfiguration validieren

## Name
Story 1.1 — App-Konfiguration validieren

## Ziel
Caldo startet nur mit gültiger Minimal-Konfiguration.

## Eingangszustand
Es gibt keine validierte Laufzeitkonfiguration.

## Ausgangszustand
`BASE_URL`, `ENCRYPTION_KEY`, `PROXY_USER_HEADER` und optionale Defaults sind geprüft.

## Akzeptanzkriterien
* Fehlendes `BASE_URL` verhindert den Start.
* `BASE_URL` ohne `https://` verhindert den Start.
* Fehlender `PROXY_USER_HEADER` verhindert den Start.
* Fehlender, nicht Base64-kodierter oder nicht exakt 32 Byte langer `ENCRYPTION_KEY` verhindert den Start.
* Optionale Werte wie `LOG_LEVEL`, `PORT` und `DB_PATH` erhalten dokumentierte Defaults.
* Startfehler werden strukturiert und ohne Secrets geloggt.

---
