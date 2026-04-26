# Story 17.3 — Elternaufgabe mit Unteraufgaben löschen

## Name
Story 17.3 — Elternaufgabe mit Unteraufgaben löschen

## Ziel
Löschen einer Elternaufgabe behandelt direkte Unteraufgaben explizit.

## Eingangszustand
Eine Elternaufgabe hat direkte Unteraufgaben.

## Ausgangszustand
Elternaufgabe und direkte Unteraufgaben sind nach Bestätigung gelöscht.

## Akzeptanzkriterien
* Löschdialog zeigt Anzahl direkter Unteraufgaben.
* Elternaufgabe und direkte Unteraufgaben werden gelöscht.
* Jede Task wird einzeln zu CalDAV gelöscht.
* Es gibt keinen Batch-Delete für einzelne Tasks.
* Undo-Snapshots werden für relevante Löschaktion erstellt.
* Fehler werden sichtbar und ohne stillen Datenverlust behandelt.

---
