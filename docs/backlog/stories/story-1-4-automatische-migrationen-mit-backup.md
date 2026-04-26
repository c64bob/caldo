# Story 1.4 — Automatische Migrationen mit Backup

## Name
Story 1.4 — Automatische Migrationen mit Backup

## Ziel
Schemaänderungen laufen beim Start sicher und nachvollziehbar.

## Eingangszustand
Es gibt kein verlässliches Migrationssystem.

## Ausgangszustand
Migrationen werden eingebettet, versioniert, geprüft und automatisch ausgeführt.

## Akzeptanzkriterien
* Die Migrationstabelle speichert Version, Name, Zeitpunkt und Checksum.
* Bereits angewendete Migrationen werden auf Checksum-Abweichung geprüft.
* Ausstehende Migrationen werden automatisch beim Start ausgeführt.
* Vor der ersten ausstehenden Migration wird ein SQLite-Backup erstellt.
* Eine Migration läuft jeweils in einer Transaktion.
* Fehlgeschlagene Migrationen verhindern den normalen App-Start.
* Migrationen loggen keine Task-Inhalte, Credentials oder Tokens.

---
