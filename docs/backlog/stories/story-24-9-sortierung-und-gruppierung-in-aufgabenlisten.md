# Story 24.9 — Sortierung und Gruppierung in Aufgabenlisten

## Name
Story 24.9 — Sortierung und Gruppierung in Aufgabenlisten

## Status
Umgesetzt

## Ziel
Aufgabenlisten koennen pro Ansicht Todoist-nah sortiert und gruppiert werden, ohne die gespeicherten CalDAV-Aufgabendaten zu veraendern.

## Eingangszustand
Die Reihenfolge von Aufgaben ist je Ansicht fest vorgegeben. Nutzer koennen weder Sortierkriterium und Sortierrichtung noch eine Gruppierung auswaehlen.

## Ausgangszustand
Jede Aufgabenlistenansicht bietet eine kompakte Anzeigeauswahl fuer Gruppierung, Sortierung und Sortierrichtung. Die Auswahl bleibt pro Ansicht erhalten, respektiert die Aufgabenhierarchie und kann auf den Standard der Ansicht zurueckgesetzt werden.

## Akzeptanzkriterien
* Aufgabenlisten bieten oben rechts eine dezente Anzeigeauswahl mit "Gruppieren nach", "Sortieren nach", "Reihenfolge" und "Zuruecksetzen".
* Als Sortierung stehen Standard, Faelligkeitsdatum, Prioritaet, Name und Hinzufuegedatum zur Auswahl.
* Fuer eine vom Standard abweichende Sortierung kann zwischen aufsteigender und absteigender Reihenfolge gewechselt werden.
* Die Sortierrichtung wirkt nur auf das primaere Sortierkriterium; weitere Kriterien sorgen fuer eine stabile und reproduzierbare Reihenfolge.
* Die Standardsortierung behaelt das bisherige, zur jeweiligen Ansicht passende Verhalten bei.
* Als Gruppierung stehen Keine, Projekt, Faelligkeitsdatum, Hinzufuegedatum und Prioritaet zur Auswahl.
* Innerhalb jeder Gruppe wird die ausgewaehlte Sortierung angewendet.
* Die Gruppierung nach Faelligkeitsdatum unterscheidet mindestens Ueberfaellig, Heute, Morgen, spaetere Daten und Kein Datum.
* Die Gruppierung nach Prioritaet unterscheidet P1, P2, P3 und Keine Prioritaet.
* Optionen, die in einer Ansicht keinen sinnvollen Mehrwert haben, werden dort nicht angeboten, insbesondere die Projektgruppierung innerhalb eines einzelnen Projekts.
* Die Auswahl wird getrennt fuer Heute, Demnaechst, jedes Projekt, jedes Label, jeden gespeicherten Filter, Favoriten und Suche gespeichert.
* Hauptaufgaben und ihre Unteraufgaben bleiben zusammen. Die Hauptaufgabe bestimmt Gruppe und Position des Aufgabenverbunds; die relative Reihenfolge ihrer Unteraufgaben wird durch Sortierung oder Gruppierung nicht veraendert.
* Aktive Anzeigeoptionen sind platzsparend erkennbar und koennen mit einer Aktion vollstaendig auf den Ansichtsstandard zurueckgesetzt werden.
* Alle Anzeigeoptionen sind per Maus, Touch und Tastatur bedienbar und funktionieren auf Desktop, Tablet und Mobilgeraeten.
* Sortier- und Gruppierungseinstellungen sind reine lokale Anzeigeeinstellungen und veraendern weder VTODO-Inhalte noch CalDAV-Ressourcen.
* Manuelle Drag-and-Drop-Reihenfolge, Gruppierung nach Label, Deadline und Zustaendigkeit gehoeren nicht zum Umfang dieser Story.
