# Story 2.2 — Reverse-Proxy-Authentifizierung

## Name
Story 2.2 — Reverse-Proxy-Authentifizierung

## Ziel
Caldo nutzt ausschließlich vorgelagerte Authentifizierung.

## Eingangszustand
Requests werden nicht authentifiziert.

## Ausgangszustand
Alle normalen App-Routen verlangen den konfigurierten Auth-Header.

## Akzeptanzkriterien
* Der Headername kommt aus `PROXY_USER_HEADER`.
* Requests ohne gültigen Header erhalten `403 Forbidden`.
* Es gibt keinen lokalen Login.
* Es gibt keinen Login-Redirect.
* Es gibt keine Rollen.
* Der Headerwert wird nicht geloggt.

---
