# Story 13.2 — Löschkonflikte erkennen

## Name
Story 13.2 — Löschkonflikte erkennen

## Ziel
Edit/Delete- und Delete/Edit-Fälle werden explizit behandelt.

## Eingangszustand
Eine Seite hat geändert, die andere gelöscht.

## Ausgangszustand
Der Nutzer kann über Wiederherstellung oder Löschung entscheiden.

## Akzeptanzkriterien
* Lokal geändert, remote gelöscht erzeugt `edit_delete`.
* Lokal gelöscht, remote geändert erzeugt `delete_edit`.
* Fehlende Seite wird als `NULL`-VTODO gespeichert.
* Die Konfliktansicht bietet passende Optionen.
* Es gibt keinen stillen Datenverlust.
* Nicht betroffene Tasks bleiben synchronisierbar.

---
