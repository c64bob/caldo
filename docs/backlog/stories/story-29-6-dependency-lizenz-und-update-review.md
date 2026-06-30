# Story 29.6 — Dependency-, Lizenz- und Update-Review

## Name
Story 29.6 — Dependency-, Lizenz- und Update-Review

## Status
Offen

## Ziel
Abhaengigkeiten, Lizenzen und Update-Risiken sind vor Releases bekannt.

## Eingangszustand
CI baut und testet die vorhandenen Abhaengigkeiten, aber Release-Reviews sind nicht explizit geplant.

## Ausgangszustand
Vor Releases gibt es eine knappe, wiederholbare Pruefung fuer Abhaengigkeiten, Lizenzen und bekannte Sicherheitsmeldungen.

## Akzeptanzkriterien
* Go-, Templ-, Tailwind-, npm- und Playwright-Abhaengigkeiten werden in den Review einbezogen.
* Lizenz- und Sicherheitsrisiken werden bewertet und dokumentiert.
* Notwendige Updates oder Ausnahmen werden als Issues nachverfolgbar.
* Der Review nutzt keine privaten Daten und enthaelt keine Secrets.

---
