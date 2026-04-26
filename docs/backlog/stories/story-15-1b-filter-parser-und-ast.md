# Story 15.1b — Filter-Parser und AST

## Name
Story 15.1b — Filter-Parser und AST

## Ziel
Eine Tokenliste wird zu einem Syntaxbaum (AST) geparst, der die Operatorprioritäten korrekt abbildet.

## Eingangszustand
Story 15.1a ist abgeschlossen; der Lexer liefert Tokenlisten.

## Ausgangszustand
Ein rekursiv-deszendenter Parser baut aus der Tokenliste einen AST.

## Akzeptanzkriterien
* Der AST unterscheidet folgende Node-Typen: `AndNode`, `OrNode`, `NotNode`, `FilterNode` (Blatt mit Operator und Wert).
* Operatorpriorität: `NOT` bindet stärker als `AND`, `AND` stärker als `OR`.
* Klammerung mit `(` und `)` überschreibt Priorität korrekt.
* `today` ohne weitere Operatoren erzeugt einen validen Einzel-Node-AST.
* Fehlende schließende Klammer ergibt einen `ParseError`; kein Panic.
* Unbekannte Token an unerwarteter Stelle ergeben einen `ParseError`.
* Der Parser hat keine Abhängigkeit zu Datenbank oder HTTP.
* Unit-Tests laufen ohne DB und ohne HTTP.
* Tests decken alle Knotentypen, Prioritätsfälle, Klammerung und Fehlerfälle ab.

---
