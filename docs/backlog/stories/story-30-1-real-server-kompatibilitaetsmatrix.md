# Story 30.1 — Real-Server-Kompatibilitaetsmatrix

## Name
Story 30.1 — Real-Server-Kompatibilitaetsmatrix

## Status
Umgesetzt

## Ziel
Caldo dokumentiert, welche CalDAV-Server und Faehigkeiten regelmaessig geprueft werden.

## Eingangszustand
Fake-CalDAV und einzelne Nextcloud-Tests existieren, aber keine gepflegte Kompatibilitaetsmatrix.

## Ausgangszustand
Release-Entscheidungen koennen auf dokumentierte Ergebnisse mit echten Servern gestuetzt werden.

## Akzeptanzkriterien
* Die Matrix nennt Servertyp, Version, Auth-Methode, erkannte CalDAV-Faehigkeiten und getestete Kernflows.
* Ergebnisse verwenden synthetische Testdaten und enthalten keine Zugangsdaten.
* Bekannte Einschraenkungen sind sichtbar und mit Issues verlinkbar.
* Neue Server oder Versionen koennen ohne Codeaenderung ergaenzt werden.

---
