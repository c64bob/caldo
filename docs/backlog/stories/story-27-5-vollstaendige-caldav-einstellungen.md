# Story 27.5 — Vollstaendige CalDAV-Einstellungen

## Name
Story 27.5 — Vollstaendige CalDAV-Einstellungen

## Status
Offen

## Ziel
CalDAV-Verbindung und Sync-Verhalten koennen nach Setup sicher verwaltet werden.

## Eingangszustand
Settings existieren teilweise, aber CalDAV-Verbindungsverwaltung ist nicht vollstaendig nutzbar.

## Ausgangszustand
Nutzer koennen Verbindung pruefen, Credentials ersetzen und Sync-Einstellungen anpassen, ohne Setup neu zu starten.

## Akzeptanzkriterien
* Credentials werden nie im Klartext angezeigt.
* Verbindungstest zeigt Erfolg oder Fehlerklasse ohne sensible Details.
* Aenderungen speichern neue Credentials verschluesselt.
* Nach geaenderten Credentials wird der Sync-Zustand nachvollziehbar dargestellt.

---
