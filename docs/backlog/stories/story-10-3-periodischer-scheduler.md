# Story 10.3 — Periodischer Scheduler

## Name
Story 10.3 — Periodischer Scheduler

## Ziel
Remote-Änderungen werden serverseitig regelmäßig abgeholt.

## Eingangszustand
Es gibt keinen periodischen Sync.

## Ausgangszustand
Scheduler führt Full-Syncs im konfigurierten Intervall aus.

## Akzeptanzkriterien
* Scheduler startet erst nach `setup_complete=true`.
* Default-Intervall ist 15 Minuten.
* Intervalländerungen starten den Ticker kontrolliert neu.
* Scheduler läuft im Go-Prozess.
* Kein Browser-Polling dient als Scheduler.
* Kein Cron, Redis oder externer Job-Runner wird benötigt.
* Scheduler startet keinen neuen Sync, solange einer aktiv ist.

---
