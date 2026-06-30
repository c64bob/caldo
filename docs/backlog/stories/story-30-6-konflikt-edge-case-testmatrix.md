# Story 30.6 — Konflikt-Edge-Case-Testmatrix

## Name
Story 30.6 — Konflikt-Edge-Case-Testmatrix

## Status
Offen

## Ziel
Konflikte werden fuer komplexe CalDAV- und VTODO-Faelle systematisch validiert.

## Eingangszustand
Kernkonflikte sind getestet, aber komplexe VTODO-Felder und Provider-Eigenheiten sind nicht als Matrix geplant.

## Ausgangszustand
Release-Tests koennen kritische Konfliktfaelle gezielt auswaehlen und bewerten.

## Akzeptanzkriterien
* Die Matrix umfasst 412, Remote-Delete gegen lokale Aenderung, gleichzeitige Feldupdates und wiederholte Konfliktloesung.
* Unteraufgaben, Labels, Wiederholungen, Alarme, Anhaenge und unbekannte VTODO-Felder sind beruecksichtigt.
* Jede Matrix-Zeile beschreibt erwartetes Ergebnis und erlaubte sichtbare Fehlerzustaende.
* Gefundene Abweichungen werden als Issues mit nicht privaten Reproduktionsschritten erfasst.

---
