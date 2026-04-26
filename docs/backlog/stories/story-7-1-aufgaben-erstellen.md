# Story 7.1 — Aufgaben erstellen

## Name
Story 7.1 — Aufgaben erstellen

## Ziel
Neue Aufgaben werden im Default- oder gewählten Projekt erstellt und sofort zu CalDAV geschrieben.

## Eingangszustand
Der Nutzer hat mindestens ein Projekt und ein Default-Projekt.

## Ausgangszustand
Eine neue Aufgabe existiert lokal und remote.

## Akzeptanzkriterien
* Neue Aufgabe benötigt Titel und Projekt.
* Ohne explizites Projekt wird das Default-Projekt verwendet.
* Wenn kein gültiges Default-Projekt existiert, wird Erstellung blockiert.
* Task wird lokal als `pending` vorbereitet.
* CalDAV-Create läuft synchron.
* Erst nach erfolgreichem CalDAV-Write gilt die Aufgabe als gespeichert.
* Bei Erfolg werden HREF, ETag, `sync_status=synced` und Version gespeichert.
* Bei Fehler sieht der Nutzer eine Fehlermeldung; keine stille Speicherung.

---
