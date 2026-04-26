# Story 11.3 — Fokus-Refresh

## Name
Story 11.3 — Fokus-Refresh

## Ziel
Lange offene Tabs aktualisieren veraltete Fragmente beim Zurückkehren.

## Eingangszustand
Ein Tab kann lange mit alten Versionen offen bleiben.

## Ausgangszustand
Der Tab kann bekannte Task-Versionen gegen den Server prüfen.

## Akzeptanzkriterien
* `GET /api/tasks/versions` nimmt bekannte Task-IDs entgegen.
* Response enthält aktuelle Versionen.
* Der Client lädt nur veraltete Fragmente nach.
* Offene Formulare ohne lokale Änderungen dürfen aktualisiert werden.
* Offene Formulare mit lokalen Änderungen werden nicht überschrieben.
* Bei lokalen Änderungen wird ein Hinweis angezeigt.

---
