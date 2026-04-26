# Story 15.1c — AST-zu-SQL-Compiler

## Name
Story 15.1c — AST-zu-SQL-Compiler

## Ziel
Ein AST wird in eine parametrisierte SQL-WHERE-Klausel übersetzt, die gegen die `tasks`-Tabelle ausgeführt werden kann.

## Eingangszustand
Story 15.1b ist abgeschlossen; der Parser liefert ASTs.

## Ausgangszustand
Der Compiler erzeugt aus einem AST einen SQL-Fragment-String und eine Parameterliste.

## Akzeptanzkriterien
* `TODAY` erzeugt `due_date = ?` mit dem heutigen Datum als Parameter.
* `OVERDUE` erzeugt `due_date < ?` mit dem heutigen Datum als Parameter.
* `UPCOMING` erzeugt `due_date BETWEEN ? AND ?` mit dem konfigurierten Vorschauzeitraum.
* `NO_DATE` erzeugt `due_date IS NULL`.
* `PROJECT` erzeugt einen Vergleich auf `project_name` (denormalisiertes Feld).
* `LABEL` erzeugt einen Vergleich auf `label_names` (denormalisiertes Feld).
* `PRIORITY` erzeugt einen Vergleich auf das `priority`-Feld.
* `COMPLETED` erzeugt einen Filter auf `sync_status` und `completed_at`.
* `TEXT` erzeugt eine FTS5-Subquery gegen den Suchindex.
* `BEFORE` und `AFTER` erzeugen datumbasierte `due_date`-Vergleiche.
* `AND`, `OR`, `NOT` werden korrekt in SQL-Konstrukte übersetzt.
* SQL wird ausschließlich parametrisiert erzeugt; keine String-Interpolation von Nutzerwerten.
* Unbekannte Projekt- oder Labelnamen ergeben keine leere WHERE-Klausel, sondern `1=0` (leere Ergebnismenge).
* Unbekannte AST-Node-Typen ergeben einen `CompileError`.
* Der Compiler hat keine Abhängigkeit zu Datenbank oder HTTP.
* Unit-Tests laufen ohne DB und ohne HTTP.
* Tests decken alle Filtertypen, logische Verknüpfungen, Parametrisierung und Fehlerfälle ab.

---
