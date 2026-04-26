# Story 4.2 — CalDAV-Verbindungstest

## Name
Story 4.2 — CalDAV-Verbindungstest

## Ziel
CalDAV-Konfiguration wird nur nach echtem Verbindungstest akzeptiert.

## Eingangszustand
Es gibt keine geprüfte CalDAV-Verbindung.

## Ausgangszustand
CalDAV-URL und Credentials können getestet und Capability-Daten gespeichert werden.

## Akzeptanzkriterien
* Der Test nutzt einen echten CalDAV/WebDAV-Request.
* Der Test erkennt globale Account-/Server-Capabilities.
* WebDAV-Sync-, CTag-, ETag- und Fullscan-Fähigkeiten werden gespeichert.
* Fehlschlag zeigt einen sicheren Fehler ohne Secrets.
* Fehlschlag markiert die Konfiguration nicht als erfolgreich.
* Timeouts werden eingehalten.

---
