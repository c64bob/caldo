# Story 6.1 — VTODO-Felder extrahieren

## Name
Story 6.1 — VTODO-Felder extrahieren

## Ziel
Caldo kann VTODOs lesen und bekannte Felder normalisieren.

## Eingangszustand
Raw-VTODOs sind nicht fachlich auswertbar.

## Ausgangszustand
Bekannte Felder sind aus Raw-VTODOs extrahierbar.

## Akzeptanzkriterien
* Titel wird extrahiert.
* Beschreibung wird extrahiert.
* Fälligkeit mit und ohne Uhrzeit wird extrahiert.
* Status und Completed werden extrahiert.
* Priorität wird extrahiert.
* Kategorien werden extrahiert.
* RRULE wird als Rohstring extrahiert.
* Parent-Referenzen werden extrahiert.
* Unbekannte Properties bleiben im Raw-VTODO erhalten.

---
