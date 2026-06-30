# Story 30.5 — CTag-/ETag-Strategie implementieren

## Name
Story 30.5 — CTag-/ETag-Strategie implementieren

## Status
Offen

## Ziel
CTag- und ETag-Informationen werden genutzt, um unnoetige Full-Scans zu vermeiden.

## Eingangszustand
CTag- und ETag-Faehigkeiten werden gespeichert, aber nicht als vollwertige inkrementelle Strategie genutzt.

## Ausgangszustand
Unveraenderte Kalender oder Aufgaben koennen uebersprungen werden, ohne lokale oder entfernte Aenderungen zu verlieren.

## Akzeptanzkriterien
* Unveraenderte CTag-/ETag-Zustaende fuehren zu nachvollziehbar weniger Remote-Arbeit.
* Geaenderte Aufgaben werden korrekt aktualisiert, geloescht oder als Konflikt markiert.
* Fehlerhafte oder fehlende CTag-/ETag-Daten fuehren zu sicherem Full-Scan-Fallback.
* Tests decken unveraenderte, geaenderte, geloeschte und konfliktbehaftete Remote-Zustaende ab.

---
