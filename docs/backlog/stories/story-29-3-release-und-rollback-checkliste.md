# Story 29.3 — Release- und Rollback-Checkliste

## Name
Story 29.3 — Release- und Rollback-Checkliste

## Status
Offen

## Ziel
Releases folgen einer klaren Checkliste inklusive Rollback-Entscheidung.

## Eingangszustand
CI und Container-Builds existieren, aber Release- und Rollback-Kriterien sind nicht gebuendelt.

## Ausgangszustand
Jede Freigabe kann anhand einer knappen Checkliste bewertet, veroeffentlicht oder gestoppt werden.

## Akzeptanzkriterien
* Die Checkliste nennt Commit, Version, Image, CI-Ergebnis, Staging-Smoke, bekannte Risiken und offene Blocker.
* Rollback fuer Binary- und Containerbetrieb ist beschrieben.
* Datenbankmigrationen werden mit Backup-/Restore-Entscheidung beruecksichtigt.
* Release-Notizen und bekannte Einschraenkungen sind vor der Freigabe festgehalten.

---
