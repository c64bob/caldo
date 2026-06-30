# Epic 30 — CalDAV Sync Maturity

## Ziel
CalDAV-Synchronisation wird ueber echte Server, groessere Datenmengen und inkrementelle Strategien hinweg produktionsreif.

## Rahmen
- Die Epic beschreibt Sync-Haertung nach dem Full-Scan-MVP.
- CalDAV bleibt fuehrende Datenquelle.
- Bestehende Datenverlust-, Konflikt- und Logging-Invarianten bleiben unveraendert.
- Konkrete Umsetzung erfolgt erst in den jeweils ausgewaehlten Stories.

## Enthaltene Stories
- Story 30.1: Real-Server-Kompatibilitaetsmatrix
- Story 30.2: Langlauf-Sync-Validierung
- Story 30.3: Grosse Kalender und dokumentierte Grenzwerte
- Story 30.4: WebDAV-Sync-Strategie implementieren
- Story 30.5: CTag-/ETag-Strategie implementieren
- Story 30.6: Konflikt-Edge-Case-Testmatrix
