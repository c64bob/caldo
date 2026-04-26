# Story 19.3 — Laufende Writes sichtbar machen

## Name
Story 19.3 — Laufende Writes sichtbar machen

## Ziel
Der Nutzer versteht, wann Änderungen noch nicht gespeichert sind.

## Eingangszustand
Writes könnten unbemerkt laufen oder abbrechen.

## Ausgangszustand
UI zeigt laufende und fehlgeschlagene Writes klar an.

## Akzeptanzkriterien
* Während eines Writes ist ein Speichern-/Pending-Zustand sichtbar.
* Bei erfolgreichem Write wird der gespeicherte Zustand angezeigt.
* Bei Fehler wird eine sichtbare Fehlermeldung angezeigt.
* Formularinhalte bleiben nach Möglichkeit erhalten.
* Beim Schließen/Navigieren mit laufendem Write wird `beforeunload` genutzt, soweit Browser es erlauben.
* Es gibt keine Browser-Offline-Queue.

---
