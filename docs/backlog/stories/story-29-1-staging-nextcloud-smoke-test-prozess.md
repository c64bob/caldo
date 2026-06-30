# Story 29.1 — Staging-Nextcloud-Smoke-Test-Prozess

## Name
Story 29.1 — Staging-Nextcloud-Smoke-Test-Prozess

## Status
Offen

## Ziel
Ein wiederholbarer Staging-Smoke-Test gibt vor Releases Sicherheit mit einem echten Nextcloud-CalDAV-Server.

## Eingangszustand
Fake-CalDAV-Tests und einzelne Staging-Tests existieren, aber kein verbindlicher Release-Smoke-Prozess.

## Ausgangszustand
Vor Releases kann derselbe Staging-Flow mit synthetischen Daten ausgefuehrt und als Release-Evidenz festgehalten werden.

## Akzeptanzkriterien
* Der Smoke-Test deckt Setup, Import, manuellen Sync, Remote-Anlage, Remote-Aenderung, Remote-Loeschung, lokalen Dirty-vs-Remote-Konflikt und Konfliktloesung ab.
* Testdaten sind synthetisch und enthalten keine privaten Aufgaben oder Zugangsdaten.
* Ergebnisnotizen enthalten Datum, Build oder Commit, Servertyp und pass/fail-Status.
* Gefundene Fehler werden als GitHub Issues erfasst und einem Release-Milestone zugeordnet.

---
