# Story 2.3 — Session, Tab-ID und CSRF-Grundlage

## Name
Story 2.3 — Session, Tab-ID und CSRF-Grundlage

## Ziel
Mutierende UI-Aktionen sind sicher und tab-spezifisch nachvollziehbar.

## Eingangszustand
Es gibt keine Session- oder CSRF-Struktur.

## Ausgangszustand
Session-Cookie, CSRF-Token und Tab-ID-Konzept sind fachlich nutzbar.

## Akzeptanzkriterien
* `session_id` wird als `HttpOnly`, `Secure`, `SameSite=Strict` Session-Cookie gesetzt.
* CSRF schützt alle mutierenden Methoden.
* CSRF verwendet Double-Submit-Cookie mit HMAC-Validierung.
* Ungültiger oder fehlender CSRF-Token ergibt `403`.
* HTMX-Requests können `X-CSRF-Token` und `X-Tab-ID` senden.
* Undo-Identität kann über `(session_id, tab_id)` abgebildet werden.

---
