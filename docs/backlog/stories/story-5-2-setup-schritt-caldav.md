# Story 5.2 — Setup-Schritt CalDAV

## Name
Story 5.2 — Setup-Schritt CalDAV

## Ziel
Der Nutzer kann CalDAV-Zugangsdaten im Erststart erfassen und prüfen.

## Eingangszustand
Setup steht auf Schritt `caldav`.

## Ausgangszustand
Bei erfolgreichem Test sind Credentials verschlüsselt gespeichert und der Wizard geht zu Kalendern.

## Akzeptanzkriterien
* CalDAV-URL, Benutzername und Passwort/App-Passwort können eingegeben werden.
* Credentials werden sofort verschlüsselt gespeichert.
* Credentials werden nicht im Browser oder in Session-State gehalten.
* Ein echter Verbindungstest wird ausgeführt.
* Capabilities werden gespeichert.
* Bei Erfolg wird `setup_step='calendars'`.
* Bei Fehler bleibt `setup_step='caldav'`.

---
