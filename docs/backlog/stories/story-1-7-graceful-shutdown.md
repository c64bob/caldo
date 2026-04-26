# Story 1.7 — Graceful Shutdown

## Name
Story 1.7 — Graceful Shutdown

## Ziel
Der Prozess beendet sich bei SIGTERM und SIGINT kontrolliert, ohne laufende CalDAV-Operationen abrupt abzubrechen.

## Eingangszustand
Der HTTP-Server läuft, aber ein SIGTERM beendet den Prozess sofort ohne Rücksicht auf laufende Requests oder Sync-Läufe.

## Ausgangszustand
SIGTERM und SIGINT lösen eine geordnete Shutdown-Sequenz aus, nach deren Abschluss der Prozess sauber endet.

## Akzeptanzkriterien
* Signal-Handler registriert sich für `SIGTERM` und `SIGINT`.
* Bei Signal-Empfang nimmt der HTTP-Server sofort keine neuen Verbindungen mehr an.
* Laufende HTTP-Requests werden mit einem Timeout von maximal 30 Sekunden abgewartet.
* Der Scheduler wird angewiesen, keine neuen Jobs mehr zu starten.
* Ein laufender Sync-Job darf bis zu 30 Sekunden zur Fertigstellung nutzen.
* Alle CalDAV-Operationen verwenden `context.Context`, sodass sie auf Timeout reagieren.
* Nach Ablauf des Timeouts werden verbleibende Operationen kontextbasiert abgebrochen.
* Der Prozess endet mit Exit-Code 0 nach geordnetem Shutdown.
* Der Prozess endet mit Exit-Code 1, wenn der Shutdown-Timeout überschritten wurde.
* Kein laufender CalDAV-Write wird durch den Shutdown stumm abgebrochen ohne Log-Eintrag.
* Der Shutdown-Ablauf wird strukturiert geloggt (Start, Scheduler gestoppt, HTTP-Server gestoppt, Prozess beendet); keine Task-Inhalte werden dabei geloggt.

---
