# Story 29.2 — Backup-, Restore- und Migrationsdrill

## Name
Story 29.2 — Backup-, Restore- und Migrationsdrill

## Status
Offen

## Ziel
Betreiber koennen vor und nach Migrationen nachvollziehbar sichern und wiederherstellen.

## Eingangszustand
Automatische Migrationsbackups existieren, aber Restore und Migrationsdrill sind nicht als Release-Kriterium dokumentiert.

## Ausgangszustand
Ein Restore aus Backup und ein Migrationslauf auf einer Kopie koennen vor Releases geprueft werden.

## Akzeptanzkriterien
* Der Drill beschreibt Sicherung, Restore und Start mit wiederhergestellter Datenbank.
* Migrationen werden auf einer Kopie oder Staging-Datenbank geprueft, nicht direkt auf privaten Produktionsdaten.
* Ein fehlgeschlagener Restore oder Migrationsdrill blockiert die Freigabe.
* Betreiber koennen erkennen, welche Dateien fuer Backup und Restore relevant sind.

---
