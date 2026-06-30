# Backlog- und Issue-Tracking

## Ziel
Fehlende Arbeit soll entweder als Story, GitHub Issue oder Release-Milestone sichtbar sein. Es gibt keine dauerhafte informelle Liste ausserhalb des Repositories oder GitHub.

## Stories
Stories beschreiben geplante Produkt-, QA-, Betriebs- oder Engineering-Arbeit mit Akzeptanzkriterien.

Eine Story ist passend, wenn:
- das Ziel vorab planbar ist
- mehrere Aenderungen oder Pruefschritte zusammengehoeren
- die Arbeit Teil einer Epic oder eines Release-Ziels ist
- die Akzeptanzkriterien vor der Umsetzung klar formuliert werden koennen

Stories bleiben Planungsdokumente. Sie enthalten keine Code-Skizzen, keine Architekturentscheidungen ausserhalb von `docs/arch.md` und keine privaten Testdaten.

## GitHub Issues
GitHub Issues erfassen konkrete Findings, Defekte, Release-Blocker und kleine, einzeln abarbeitbare Folgearbeiten.

Ein Issue ist passend, wenn:
- ein Fehler in Staging, Produktion, CI oder Browser-QA beobachtet wurde
- ein konkreter Reproduktionsweg existiert oder noch gesammelt werden muss
- eine Story zwar geplant ist, aber ein einzelner Defekt separat verfolgt werden soll
- das Finding eine Release-Entscheidung beeinflusst

Issues enthalten keine CalDAV-Zugangsdaten, keine Aufgabeninhalte, keine Sessionwerte, keine CSRF-Tokens und keine privaten Screenshots.

Standardlabels fuer produktionsnahe Arbeit:
- `production-readiness`: Arbeit oder Finding vor produktiver Freigabe.
- `sync-maturity`: CalDAV-Sync-Haertung, Provider-Kompatibilitaet oder inkrementelle Strategien.
- `staging-finding`: Finding aus Staging- oder Real-Server-Validierung.
- `release-blocker`: blockiert den aktuellen Release-Milestone.

## Milestones
Milestones gruppieren Issues und PRs fuer konkrete Release-Ziele.

Der Standard-Milestone fuer produktionsnahe Arbeit ist `v1.0 production readiness`.

Ein Issue gehoert in diesen Milestone, wenn es:
- eine produktive Freigabe blockiert
- Staging- oder Real-Server-Vertrauen betrifft
- Backup, Restore, Migration, Security, Performance oder Sync-Maturity betrifft
- als offene Einschraenkung in Release-Notizen erscheinen muesste

## Arbeitsregel
Vor Beginn einer neuen Arbeit gilt:

1. Geplante Arbeit wird als Story in `docs/backlog/stories/` beschrieben.
2. Beobachtete Fehler werden als GitHub Issue erfasst.
3. Release-relevante Issues werden dem Release-Milestone zugeordnet.
4. PRs verweisen auf die Story oder das Issue, das sie abschliessen.
5. Nach Merge wird der Story-Status oder das Issue aktualisiert.
