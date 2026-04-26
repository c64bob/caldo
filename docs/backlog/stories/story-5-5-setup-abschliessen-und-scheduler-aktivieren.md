# Story 5.5 — Setup abschließen und Scheduler aktivieren

## Name
Story 5.5 — Setup abschließen und Scheduler aktivieren

## Ziel
Nach erfolgreichem Initialimport wechselt Caldo ohne Neustart in den Normalbetrieb.

## Eingangszustand
Initialimport ist erfolgreich abgeschlossen.

## Ausgangszustand
`setup_complete=true`, normale Routen sind erreichbar, Scheduler läuft.

## Akzeptanzkriterien
* Kalenderauswahl, Default-Projekt und Importerfolg werden geprüft.
* `setup_step='complete'` wird gesetzt.
* `setup_complete=true` wird gesetzt.
* Nach Commit lässt das Setup-Gate normale Routen zu.
* Der Scheduler wird gestartet.
* Ein Scheduler-Startfehler rollt Setup nicht zurück.
* Der Nutzer wird zur normalen App-UI weitergeleitet.

---
