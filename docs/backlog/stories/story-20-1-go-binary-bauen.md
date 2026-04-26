# Story 20.1 — Go-Binary bauen

## Name
Story 20.1 — Go-Binary bauen

## Ziel
Caldo ist als einzelnes Go-Binary lieferbar.

## Eingangszustand
Es gibt kein baubares Release-Artefakt.

## Ausgangszustand
Ein Binary enthält Serverlogik, Templates und Migrationen.

## Akzeptanzkriterien
* Build erzeugt ein lauffähiges Caldo-Binary.
* Migrationen sind eingebettet.
* Templates sind generiert/eingebunden.
* Assets unter `web/static` werden separat bereitgestellt.
* Build-Reihenfolge folgt Templates, Tailwind, Go-Build.

---
