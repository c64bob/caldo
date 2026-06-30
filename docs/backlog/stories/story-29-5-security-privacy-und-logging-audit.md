# Story 29.5 — Security-, Privacy- und Logging-Audit

## Name
Story 29.5 — Security-, Privacy- und Logging-Audit

## Status
Offen

## Ziel
Vor produktiver Freigabe werden Sicherheits-, Datenschutz- und Logging-Invarianten gezielt geprueft.

## Eingangszustand
Architekturregeln und Tests existieren, aber kein gebuendelter Auditprozess fuer Releases.

## Ausgangszustand
Release-Kandidaten haben eine nachvollziehbare Pruefung fuer sensible Daten, Auth, CSRF, CSP und Credential-Schutz.

## Akzeptanzkriterien
* Logs und Fehlerausgaben werden auf verbotene Inhalte geprueft.
* Reverse-Proxy-Auth, CSRF, CSP und Secret-Handling sind Teil der Auditliste.
* CalDAV-Zugangsdaten, Aufgabeninhalte, VTODO-Rohdaten und Tokens erscheinen nicht in Audit-Artefakten.
* Blockierende Findings werden als GitHub Issues mit Release-Milestone erfasst.

---
