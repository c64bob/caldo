# Story 30.4 — WebDAV-Sync-Strategie implementieren

## Name
Story 30.4 — WebDAV-Sync-Strategie implementieren

## Status
Offen

## Ziel
Die WebDAV-Sync-Strategie wird als echte inkrementelle Sync-Option umgesetzt.

## Eingangszustand
WebDAV-Sync kann als Faehigkeit erkannt werden, faellt aber im normalen Sync auf Full-Scan zurueck.

## Ausgangszustand
Kalender mit verlaesslichem WebDAV-Sync koennen inkrementell synchronisiert werden und fallen bei Problemen sicher zurueck.

## Akzeptanzkriterien
* Unterstuetzte Kalender nutzen WebDAV-Sync ohne vollstaendigen Kalender-Scan.
* Fallback auf CTag oder Full-Scan verletzt keine Konflikt- oder Datenverlustregeln.
* Sync-Metadaten werden nach erfolgreichem Lauf nachvollziehbar aktualisiert.
* Tests nutzen Fake- oder Staging-CalDAV und keine realen privaten Kalender.

---
