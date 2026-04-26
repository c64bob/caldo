# Story 15.1a — Filter-Lexer und Token-Definitionen

## Name
Story 15.1a — Filter-Lexer und Token-Definitionen

## Ziel
Filterausdrücke können in eine Folge typisierter Tokens zerlegt werden.

## Eingangszustand
Es gibt keine Tokenisierung für Filterqueries.

## Ausgangszustand
Ein Lexer nimmt einen Filterstring entgegen und gibt eine Tokenliste zurück.

## Akzeptanzkriterien
* Erkannte Token-Typen: `TODAY`, `OVERDUE`, `UPCOMING`, `NO_DATE`, `COMPLETED`, `PRIORITY`, `TEXT`, `PROJECT` (`#`-Prefix), `LABEL` (`@`-Prefix), `BEFORE`, `AFTER`, `AND`, `OR`, `NOT`, `LPAREN`, `RPAREN`, `COLON`, `STRING`, `EOF`.
* Schlüsselwörter sind case-insensitiv: `today`, `TODAY`, `Today` erzeugen denselben Token-Typ.
* Whitespace zwischen Tokens wird übersprungen.
* Unbekannte Zeichenfolgen werden als `STRING`-Token behandelt, nicht als Fehler.
* Der Lexer ist zustandslos und hat keine Abhängigkeit zu Datenbank oder HTTP.
* Unit-Tests laufen ohne DB und ohne HTTP.
* Tests decken alle Token-Typen, Groß-/Kleinschreibung, Sonderzeichen in Strings und leere Eingabe ab.

---
