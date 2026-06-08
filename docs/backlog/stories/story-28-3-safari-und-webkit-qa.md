# Story 28.3 — Safari- und WebKit-QA

## Name
Story 28.3 — Safari- und WebKit-QA

## Status
Offen

## Ziel
Safari-nahe Browserunterschiede werden vor Releases sichtbar.

## Eingangszustand
CI prueft vor allem Chromium; WebKit/Safari ist nur teilweise dokumentiert.

## Ausgangszustand
Kritische MVP-Flows koennen regelmaessig gegen WebKit ausgefuehrt werden.

## Akzeptanzkriterien
* Der QA-Prozess beschreibt, wann WebKit lokal oder automatisiert ausgefuehrt wird.
* Setup, Quick Add, task write-through, manual sync und Konfliktloesung sind abgedeckt.
* Fehlende WebKit-Voraussetzungen fuehren zu klaren Setup-Hinweisen.
* WebKit-Tests nutzen weiterhin nur Fake- oder Staging-CalDAV.

---
