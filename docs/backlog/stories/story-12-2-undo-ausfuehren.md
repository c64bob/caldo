# Story 12.2 — Undo ausführen

## Name
Story 12.2 — Undo ausführen

## Ziel
Undo ist eine neue fachliche Gegenänderung mit CalDAV-Write.

## Eingangszustand
Ein gültiger Undo-Snapshot existiert.

## Ausgangszustand
Der vorherige Zustand ist wiederhergestellt oder ein Fehler/Konflikt ist sichtbar.

## Akzeptanzkriterien
* Snapshot wird anhand von Session und Tab geladen.
* Abgelaufener Snapshot kann nicht verwendet werden.
* Aktuelle Task wird mit `etag_at_snapshot` verglichen.
* Bei abweichendem ETag wird ein Konflikt erzeugt.
* Zielzustand wird als `pending` gespeichert.
* CalDAV-Write läuft synchron.
* Erst nach erfolgreichem Write wird der Snapshot gelöscht.
* Bei Write-Fehler bleibt der Snapshot erhalten, sofern nicht abgelaufen.

---
