# Story 17.1 — Unteraufgaben importieren und anzeigen

## Name
Story 17.1 — Unteraufgaben importieren und anzeigen

## Ziel
Eine Ebene Unteraufgaben wird aus CalDAV sichtbar.

## Eingangszustand
Parent-Referenzen werden nicht ausgewertet.

## Ausgangszustand
Direkte Unteraufgaben erscheinen eingerückt unter Elternaufgaben.

## Akzeptanzkriterien
* `RELATED-TO;RELTYPE=PARENT` wird als Parent erkannt.
* `RELATED-TO` ohne RELTYPE wird Nextcloud-kompatibel als Parent interpretiert.
* Genau eine Ebene wird dargestellt.
* Tiefere Verschachtelungen werden als Wurzelaufgaben importiert.
* Raw-VTODO tieferer Aufgaben bleibt unverändert.
* Keine Warnung oder Badge für Tiefe 2+ ist erforderlich.

---
