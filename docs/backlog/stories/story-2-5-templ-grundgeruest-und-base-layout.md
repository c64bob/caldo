# Story 2.5 — Templ-Grundgerüst und Base-Layout

## Name
Story 2.5 — Templ-Grundgerüst und Base-Layout

## Ziel
Alle späteren Handler können serverseitig gerenderte HTML-Responses zurückgeben, die auf einem konsistenten Base-Layout basieren, HTMX und Alpine.js nutzen und eine gültige Content Security Policy einhalten.

## Eingangszustand
Es gibt keine Templ-Templates und kein definiertes HTML-Grundgerüst.

## Ausgangszustand
Ein lauffähiges Base-Layout ist vorhanden, das aus `manifest.json` die gehashten Asset-Pfade liest, HTMX und Alpine.js lokal einbindet und eine CSP-konforme Struktur vorgibt.

## Akzeptanzkriterien
* `templ generate` erzeugt aus allen `.templ`-Dateien valide `_templ.go`-Dateien.
* Das Base-Layout ist als Templ-Komponente `BaseLayout(title string, content templ.Component)` implementiert.
* Das Base-Layout erzeugt valides HTML5 mit `<!DOCTYPE html>`, `<html lang="de">`, `<head>` und `<body>`.
* HTMX wird aus `manifest.json` unter `/static/htmx.<hash>.min.js` eingebunden; kein CDN.
* Die HTMX-SSE-Extension wird aus `/static/htmx-sse.<hash>.js` eingebunden; kein CDN.
* Alpine.js wird aus `/static/alpine.<hash>.min.js` eingebunden; kein CDN.
* `app.js` wird aus `/static/app.<hash>.js` eingebunden; kein CDN.
* `app.css` wird aus `/static/app.<hash>.css` eingebunden.
* Alle Asset-Pfade werden ausschließlich über die beim Start geladene `manifest.json` aufgelöst; keine hartcodierten Hashes im Template.
* Der `Content-Security-Policy`-Header erlaubt `script-src 'self'`; kein `'unsafe-inline'` und kein `'unsafe-eval'`.
* Der `Content-Security-Policy`-Header erlaubt `style-src 'self'`; kein `'unsafe-inline'`.
* Das Layout enthält ein `<meta name="csrf-token">` mit dem aktuellen CSRF-Token, damit `app.js` ihn für HTMX-Requests auslesen kann.
* Das Layout enthält ein `<div id="notifications">` als Ziel für HTMX-Out-of-Band-Updates von Benachrichtigungen.
* Das Layout enthält ein Navigationselement mit Platzhaltern für Systemfilter und Projektliste; konkrete Inhalte kommen in späteren Stories.
* Dark-Mode-Toggle ist als Button mit `data-theme-toggle` im Layout vorhanden; Alpine.js steuert die Klasse auf `<html>`.
* Die Systempräferenz `prefers-color-scheme` wird beim ersten Laden ausgewertet.
* Ein Handler in `cmd/caldo/main.go` liefert eine Beispiel-Route `GET /` mit dem Base-Layout und einer leeren Inhaltskomponente; `go test ./...` läuft fehlerfrei durch.

---
