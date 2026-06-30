# Story 30.2 — Langlauf-Sync-Validierung

## Name
Story 30.2 — Langlauf-Sync-Validierung

## Status
Offen

## Ziel
Sync bleibt ueber laengere Laufzeit stabil und nachvollziehbar.

## Eingangszustand
Einzelne Sync- und Browserflows sind getestet, aber keine laengere Staging-Validierung ist definiert.

## Ausgangszustand
Periodischer Sync, manueller Sync und externe Aenderungen koennen ueber einen laengeren Zeitraum beobachtet werden.

## Akzeptanzkriterien
* Die Validierung umfasst wiederholte lokale und Remote-Aenderungen ueber mehrere Sync-Zyklen.
* Doppelte Aufgaben, verlorene Aenderungen, haengende Sync-Zustaende und unerwartete Konflikte werden gezielt geprueft.
* Der Lauf nutzt synthetische Daten und dokumentiert Server, Build, Dauer und Ergebnis.
* Fehler werden als Issues mit reproduzierbaren, nicht privaten Schritten erfasst.

---
