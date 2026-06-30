# Story 26.2 — Projektverwaltung in der UI

## Name
Story 26.2 — Projektverwaltung in der UI

## Status
Umgesetzt

## Ziel
Projektanlage, Umbenennung und Loeschung werden sichtbar und sicher bedienbar.

## Eingangszustand
Backend-Routen existieren, aber Verwaltungsansichten sind nicht vollstaendig ausgearbeitet.

## Ausgangszustand
Nutzer koennen Projekte erstellen, umbenennen und loeschen, waehrend Remote-Write-through erhalten bleibt.

## Akzeptanzkriterien
* Projektanlage schreibt erst nach erfolgreicher Remote-Anlage lokal erfolgreich.
* Umbenennen zeigt Erfolg oder Fehler sichtbar an.
* Loeschen macht Folgen fuer lokale Aufgaben klar.
* Fehlerhafte Remote-Operationen hinterlassen keinen stillen lokalen Erfolg.

---
