# Story 13.5 — Beide Versionen behalten

## Name
Story 13.5 — Beide Versionen behalten

## Ziel
Widersprüchliche Versionen können als separate Aufgaben erhalten bleiben.

## Eingangszustand
Ein Konflikt mit lokaler und Remote-Version existiert.

## Ausgangszustand
Beide Versionen existieren als eigenständige Aufgaben.

## Akzeptanzkriterien
* Remote-Version wird als neue Task mit neuer UID zu CalDAV geschrieben.
* Lokale Version behält ihre UID.
* Beide Tasks liegen im selben Projekt.
* Es wird keine Parent-Verknüpfung zwischen beiden erzeugt.
* Konflikt wird mit `resolution=split` markiert.
* Bei Teilfehlern wird kein still inkonsistenter Zustand als gelöst markiert.

---
