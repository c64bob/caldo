# Story 6.2 — VTODO-Patching ohne Datenverlust

## Name
Story 6.2 — VTODO-Patching ohne Datenverlust

## Ziel
Bekannte Feldänderungen zerstören keine unbekannten VTODO-Inhalte.

## Eingangszustand
Änderungen könnten Raw-VTODO vollständig neu serialisieren.

## Ausgangszustand
Nur explizit geänderte bekannte Felder werden gepatcht.

## Akzeptanzkriterien
* Unbekannte Properties bleiben erhalten.
* `VALARM` bleibt erhalten.
* `ATTACH` bleibt erhalten.
* RRULE wird nur bei expliziter Wiederholungsänderung verändert.
* Raw-VTODO ist Roundtrip-Quelle.
* Tests decken unbekannte Properties, VALARM, ATTACH und RRULE-Erhalt ab.

---
