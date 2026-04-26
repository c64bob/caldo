# Story 14.3 — Natürliche Datumseingabe Deutsch/Englisch

## Name
Story 14.3 — Natürliche Datumseingabe Deutsch/Englisch

## Ziel
Fälligkeitsdaten können natürlich eingegeben werden.

## Eingangszustand
Datum muss manuell gesetzt werden.

## Ausgangszustand
Deutsch- und Englischmuster werden erkannt.

## Akzeptanzkriterien
* `heute`, `morgen`, `übermorgen` werden erkannt.
* `today`, `tomorrow` werden erkannt.
* `nächsten Montag` und `next monday` werden erkannt.
* `in 3 Tagen` und `in 3 days` werden erkannt.
* Deutsche und englische Wochentage werden erkannt.
* Unbekannte Tokens bleiben Teil des Titels.
* Unbekannte Tokens erzeugen keine Fehlermeldung.
* Parser-Tests laufen ohne HTTP und ohne DB.

---
