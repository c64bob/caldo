# Story 5.3 — Setup-Schritt Kalenderauswahl und Default-Projekt

## Name
Story 5.3 — Setup-Schritt Kalenderauswahl und Default-Projekt

## Ziel
Der Nutzer wählt zu synchronisierende Kalender und ein Default-Projekt.

## Eingangszustand
CalDAV-Verbindung ist erfolgreich getestet.

## Ausgangszustand
Ausgewählte Kalender sind als Projekte gespeichert, ein Default-Projekt ist gesetzt.

## Akzeptanzkriterien
* Verfügbare CalDAV-Kalender werden geladen.
* Der Nutzer kann mehrere Kalender auswählen.
* Der Nutzer kann ein Default-Projekt wählen.
* Optional kann ein neues Default-Projekt angelegt werden.
* Ohne Default-Projekt ist Fortfahren nicht möglich.
* Für ausgewählte Kalender wird initial eine Sync-Strategie gesetzt.
* Bei Erfolg wird `setup_step='import'`.

---
