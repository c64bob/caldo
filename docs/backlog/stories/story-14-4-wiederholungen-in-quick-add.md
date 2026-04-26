# Story 14.4 — Wiederholungen in Quick Add

## Name
Story 14.4 — Wiederholungen in Quick Add

## Ziel
Wiederkehrende Aufgaben können direkt über natürliche Eingabe erstellt werden.

## Eingangszustand
Quick Add erzeugt nur Einzelaufgaben.

## Ausgangszustand
MVP-Wiederholungsmuster erzeugen RRULEs.

## Akzeptanzkriterien
* `jeden Montag` und `every monday` werden erkannt.
* `täglich/daily`, `wöchentlich/weekly`, `monatlich/monthly`, `jährlich/yearly` werden erkannt.
* `werktags/weekdays` wird erkannt.
* `alle X Tage/Wochen/Monate` wird erkannt.
* Erkannte Wiederholung wird nicht nachträglich abgelehnt.
* RRULE wird beim Speichern in VTODO geschrieben.
* Nicht unterstützte komplexe Muster bleiben Freitext oder werden klar nicht als Wiederholung behandelt.

---
