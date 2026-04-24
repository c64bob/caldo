# Caldo – Product Requirements Document

**Status:** Entwurf  
**Produktname:** Caldo  
**Dokumenttyp:** Product Requirements Document (PRD)  
**Zielsprache:** Deutsch  
**Zielpfad im Repository:** `docs/prd.md`

---

## 1. Überblick

Caldo ist eine selbst gehostete Web-App für Todo-Management. Die App richtet sich primär an technisch versierte Self-Hoster und ist zunächst für die Nutzung durch eine Einzelperson konzipiert.

CalDAV ist die führende Datenquelle. Caldo speichert Aufgaben lokal in SQLite, synchronisiert sie aber mit genau einem CalDAV-Account, typischerweise Nextcloud Tasks/VTODO. Mehrere CalDAV-Kalender werden als Projekte dargestellt.

Die Bedienung soll sich stark an Todoist orientieren, insbesondere hinsichtlich Navigation, Schnellanlage, Projekten, Labels, Filtern, Fälligkeitsdaten, Favoriten, Tastaturkürzeln und Ansichten wie Heute und Demnächst.

---

## 2. Ziele

### 2.1 Produktziele

- Eine Todoist-nahe Web-App für Self-Hosting bereitstellen.
- CalDAV/VTODO als führende Datenquelle verwenden.
- Nextcloud-kompatible Aufgabenverwaltung ermöglichen.
- Aufgaben, Projekte, Labels, Filter und Fälligkeitsdaten komfortabel verwalten.
- Robuste Synchronisation mit manueller Konfliktauflösung bereitstellen.
- Datenverlust vermeiden, ohne die Bedienung unnötig komplex zu machen.
- Deployment als Go-Binary und Docker-Container ermöglichen.
- Single-User-Betrieb hinter Reverse-Proxy-Authentifizierung unterstützen.

### 2.2 Nicht-Ziele

Folgende Punkte sind für Version 1 ausdrücklich nicht vorgesehen:

- Multi-User-Betrieb.
- Rollen- oder Rechtemodell.
- Team-Kollaboration.
- Echte Browser-Offline-Funktionalität oder PWA-Modus.
- Projektarchivierung.
- Vollständige mobile Optimierung.
- Board-/Kanban-Ansicht.
- Produktivitätsstatistiken oder Karma-System.
- Aufgaben-Vorlagen.
- Sichtbares Aktivitätslog.
- Papierkorb.
- Lokale Backup-/Exportfunktion.
- Themes außer Dark Mode.
- Vollständiges Erinnerungsmanagement in der UI.
- Aktive Verwaltung von Datei-Anhängen.
- Komplexe Wiederholungsausnahmen einzelner Instanzen im MVP.

---

## 3. Zielgruppe

### 3.1 Primäre Zielgruppe

- Technisch versierte Self-Hoster.
- Einzelpersonen, die eine eigene Todo-App betreiben wollen.
- Nutzer, die Nextcloud/CalDAV bereits verwenden.
- Nutzer, die Todoist-ähnliche Bedienung mit selbst gehosteter Infrastruktur wünschen.

### 3.2 Nutzerannahmen

- Der Nutzer betreibt Caldo hinter einem Reverse Proxy.
- Der Nutzer hat einen CalDAV-Server, z. B. Nextcloud.
- Der Nutzer akzeptiert Environment-basierte Serverkonfiguration.
- Der Nutzer erwartet keine Teamfunktionen.
- Der Nutzer legt Wert auf Datensicherheit und nachvollziehbare Konfliktlösung.

---

## 4. Produktprinzipien

1. **CalDAV ist führend.**  
   CalDAV ist die maßgebliche Datenquelle. Änderungen gelten erst nach erfolgreichem CalDAV-Write als gespeichert.

2. **Keine stillen Datenverluste.**  
   Konflikte werden erkannt und müssen manuell lösbar sein.

3. **Todoist-nahe Bedienung.**  
   Navigation, Schnellanlage, Filter, Labels, Projekte und Tastaturkürzel sollen sich vertraut anfühlen.

4. **Self-Hosting zuerst.**  
   Go-Binary, SQLite und Docker sind zentrale Betriebsanforderungen.

5. **Single-User bewusst einfach halten.**  
   Keine Rollen, keine Mandanten, keine Teams.

6. **Robust bei temporären Ausfällen.**  
   Die App startet auch, wenn CalDAV temporär nicht erreichbar ist, zeigt aber Fehler klar an.

---

## 5. MVP-Scope

### 5.1 Must-have für Version 1

- Single-User-Web-App.
- Go-Binary.
- Docker-Container inklusive Docker-Compose-Referenzdeployment.
- SQLite als lokale Datenbank.
- CalDAV-Verbindung zu genau einem Account.
- Unterstützung mehrerer CalDAV-Kalender.
- Projekt = CalDAV-Kalender.
- Import bestehender VTODOs.
- Bidirektionaler CalDAV-Sync.
- Sofortiges Schreiben lokaler Änderungen zu CalDAV.
- Manueller Sync.
- Periodischer Sync.
- Konfigurierbares Sync-Intervall, Default 15 Minuten.
- Konflikterkennung und manuelle Konfliktauflösung.
- Todoist-nahe UI mit serverseitigem Rendering und gezieltem JavaScript.
- Inbox-ähnliches Verhalten über konfigurierbares Default-Projekt.
- Projekte.
- Aufgaben.
- Unteraufgaben mit genau einer Ebene.
- Labels.
- Filter.
- Favoriten.
- Fälligkeitsdaten.
- Natürliche Datumseingabe Deutsch/Englisch.
- Wiederkehrende Aufgaben für definierte MVP-Muster.
- Prioritäten high/medium/low.
- Notizen/Beschreibungen.
- Suche.
- Heute-Ansicht.
- Demnächst-Ansicht.
- Überfällige Aufgaben.
- Tastaturkürzel.
- Schnell hinzufügen.
- Dark Mode.
- Einstellungen-Seite.
- Reverse-Proxy-Authentifizierung über konfigurierbaren Header.
- Verschlüsselte Speicherung von CalDAV-Zugangsdaten in SQLite.
- Strukturierte Logs mit Maskierung sensibler Daten.
- Healthcheck-Endpunkt.
- HTTPS-only-Betrieb.

### 5.2 Should-have

- Tablet-Layout brauchbar, aber nicht priorisiert.
- Safari-Unterstützung.
- Drag-and-drop zwischen Projekten.
- Anzeige von Datei-Anhängen ohne aktive Verwaltung.
- Erhaltung bestehender Erinnerungen.
- Erhaltung unbekannter VTODO-Felder.
- Internationalisierungsfähige Architektur.

### 5.3 Could-have

- Kalenderansicht.
- Listenansicht als separate Spezialansicht.
- Unterprojekte.
- Erweiterte Erinnerungsverwaltung.
- Aktive Anhangsverwaltung.
- Komplexe Wiederholungsausnahmen.
- Weitere Sprachen.
- Mobile-Optimierung.

### 5.4 Won’t-have in Version 1

- Multi-User.
- Rollen.
- Team-Sharing.
- PWA.
- Browser-Offline-Queue.
- Kanban.
- Aktivitätslog.
- Papierkorb.
- Projektarchivierung.
- Produktivitätsstatistiken.
- Aufgaben-Vorlagen.
- Lokale Backup-/Exportfunktion.

---

## 6. Architektur- und Betriebsanforderungen

### 6.1 Deployment

Caldo muss als folgende Artefakte bereitgestellt werden:

- Go-Binary.
- Docker-Container.
- Docker-Compose-Referenzdeployment.

### 6.2 Datenbank

- SQLite ist die einzige Datenbank für Version 1.
- SQLite speichert:
  - lokale Aufgabenrepräsentationen,
  - Sync-Metadaten,
  - verschlüsselte CalDAV-Zugangsdaten,
  - Einstellungen,
  - gespeicherte Filter,
  - Favoriten,
  - Konfliktmetadaten,
  - technische Konfliktversionen.

### 6.3 Konfiguration

Die Serverkonfiguration erfolgt über Environment-Variablen.

Pflichtvariablen:

- `BASE_URL`
- `ENCRYPTION_KEY`
- `PROXY_USER_HEADER`

CalDAV-Zugangsdaten werden nicht dauerhaft über Environment-Variablen bereitgestellt, sondern ausschließlich verschlüsselt in SQLite gespeichert.

### 6.4 Verschlüsselung

- CalDAV-Zugangsdaten müssen verschlüsselt in SQLite gespeichert werden.
- Der Verschlüsselungsschlüssel wird über `ENCRYPTION_KEY` bereitgestellt.
- Ohne gesetzten Verschlüsselungsschlüssel darf die App nicht starten.

### 6.5 HTTPS

- Caldo ist ausschließlich für Betrieb über HTTPS vorgesehen.
- Die App muss erkennen, wenn sie nicht über HTTPS betrieben wird (über BASE_URL mit https://-Präfix), und den Betrieb blockieren oder eine harte Fehlermeldung anzeigen.
- Der Betrieb hinter Reverse Proxy ist der Standardfall.

### 6.6 Authentifizierung

- Caldo unterstützt ausschließlich Single-User-Betrieb.
- Authentifizierung erfolgt über Reverse Proxy.
- Der authentifizierte Nutzer wird über einen konfigurierbaren Header ausgelesen.
- Der Headername wird über `PROXY_USER_HEADER` gesetzt.
- Requests ohne gültigen Auth-Header müssen abgelehnt werden.
- Es gibt keinen lokalen Notfall-Login.
- Es gibt keine Rollen.

### 6.7 Healthcheck

- Ein Healthcheck-Endpunkt ist Pflicht.
- Der Healthcheck prüft nur, ob die App läuft.
- Der Healthcheck prüft nicht CalDAV.
- Der Healthcheck prüft nicht die vollständige Sync-Fähigkeit.

### 6.8 Logging

- Logs müssen strukturiert sein.
- Sensible Daten müssen maskiert werden.
- Zu maskieren sind insbesondere:
  - CalDAV-Passwörter,
  - App-Tokens,
  - Auth-Header,
  - Header-Werte mit Auth-Bezug,
  - CalDAV-URLs mit eingebetteten Zugangsdaten,
  - Task-Titel,
  - Task-Beschreibungen,
  - sonstige Aufgabeninhalte.

---

## 7. CalDAV- und VTODO-Anforderungen

### 7.1 Grundsatz

CalDAV ist die führende Datenquelle. Lokale Änderungen gelten erst nach erfolgreichem Write zu CalDAV als gespeichert.

### 7.2 CalDAV-Account

- Genau ein CalDAV-Account wird unterstützt.
- Der Account wird über eine Konfigurationsmaske in der App eingerichtet.
- Der Nutzer kann CalDAV-URL, Benutzername und Passwort/App-Passwort später über Einstellungen ändern.
- Beim Speichern der CalDAV-Konfiguration muss ein Verbindungstest erfolgen.
- Bei fehlgeschlagenem Verbindungstest darf die Konfiguration nicht als gültig gespeichert werden.

### 7.3 Kalender und Projekte

- Ein CalDAV-Kalender entspricht einem Projekt.
- Mehrere CalDAV-Kalender müssen unterstützt werden.
- Bestehende Kalender können ausgewählt werden.
- Neue Projekte können angelegt werden.
- Das Anlegen eines Projekts legt einen neuen CalDAV-Kalender an.
- Das Umbenennen eines Projekts benennt den CalDAV-Kalender um.
- Das Löschen eines Projekts löscht den CalDAV-Kalender endgültig.
- Vor Projektlöschung ist eine starke Bestätigung erforderlich, z. B. Eingabe des Projektnamens.
- Leere Projekte sind erlaubt.
- Projektarchivierung ist nicht vorgesehen.
- Wenn ein CalDAV-Kalender remote gelöscht wird, verschwindet das lokale Projekt.

### 7.4 Default-Projekt

- Beim ersten Setup muss ein Default-Projekt gewählt werden.
- Alternativ darf ein neues Projekt beziehungsweise ein neuer CalDAV-Kalender angelegt werden.
- Neue oder unzugewiesene Aufgaben werden im Default-Projekt erstellt.
- Es gibt keine separate globale Inbox als eigenes Konzept unabhängig von CalDAV.

### 7.5 VTODO-Felder

Caldo soll folgende VTODO-Felder unterstützen, soweit technisch möglich:

- Titel.
- Beschreibung/Notizen.
- Fälligkeitsdatum.
- Fälligkeitsdatum mit Uhrzeit.
- Startdatum.
- Priorität.
- Status.
- Erledigt/Completed.
- Prozent abgeschlossen.
- Kategorien/Labels.
- Wiederholungsregeln.
- Erinnerungen, mindestens erhaltend.
- Anhänge/Links, mindestens erhaltend beziehungsweise anzeigend.
- Parent-Referenzen für Unteraufgaben.
- Unbekannte Felder.

### 7.6 Erhalt unbekannter Felder

- Unbekannte oder nicht in der UI unterstützte VTODO-Felder müssen erhalten bleiben, soweit technisch möglich.
- Das Bearbeiten eines bekannten Feldes darf nicht dazu führen, dass unbekannte Felder verloren gehen.

---

## 8. Aufgabenmodell

### 8.1 Aufgabe

Eine Aufgabe besteht mindestens aus:

- Titel.
- Projekt.
- Status offen/erledigt.
- Optionalem Fälligkeitsdatum.
- Optionaler Uhrzeit.
- Optionaler Beschreibung.
- Optionaler Priorität.
- Optionalen Labels.
- Optionalen Unteraufgaben.
- Optionaler Wiederholung.
- Optionalem Favoritenstatus.

### 8.2 Fälligkeitsdaten

- Aufgaben unterstützen sowohl reine Datumswerte als auch Datum plus Uhrzeit.
- Standardmäßig wird bei neuen Aufgaben nur ein Datum gesetzt, sofern keine Uhrzeit eingegeben wird.
- Aufgaben ohne Fälligkeitsdatum erscheinen im jeweiligen Projekt, in Labels, Suche und passenden Filtern.
- Aufgaben ohne Fälligkeitsdatum erscheinen nicht automatisch in Heute oder Demnächst.

### 8.3 Heute-Ansicht

- Die Heute-Ansicht zeigt Aufgaben mit heutigem Fälligkeitsdatum.
- Die Heute-Ansicht zeigt auch überfällige Aufgaben.

### 8.4 Demnächst-Ansicht

- Die Demnächst-Ansicht zeigt Aufgaben in einem konfigurierbaren Zeitraum.
- Default-Zeitraum: 7 Tage.

### 8.5 Erledigte Aufgaben

- Erledigen setzt die Aufgabe in CalDAV auf `completed`.
- Erledigte Aufgaben sind standardmäßig ausgeblendet.
- Es gibt eine Einstellung zum Anzeigen oder Ausblenden erledigter Aufgaben.
- Erledigte Aufgaben können in Projekten sichtbar bleiben, wenn die Einstellung dies erlaubt.

### 8.6 Löschen

- Das Löschen einer Aufgabe ist endgültig.
- Vor dem Löschen einer Aufgabe reicht eine normale Bestätigung.
- Es gibt keinen Papierkorb.
- Gelöschte Aufgaben können bei Konflikten dennoch zu Konfliktfällen führen.

### 8.7 Undo

Eine einfache Undo-Funktion für die letzte Änderung ist Pflicht.

Undo muss mindestens folgende Aktionen unterstützen:

- Aufgabe erledigen.
- Aufgabe bearbeiten.
- Aufgabe löschen.
- Projektwechsel.
- Labeländerung.

---

## 9. Unteraufgaben

### 9.1 Umfang

- Unteraufgaben sind Pflicht.
- Es wird genau eine Ebene Unteraufgaben unterstützt.
- Tiefere Verschachtelung ist nicht Bestandteil des MVP.

### 9.2 CalDAV-Abbildung

- Unteraufgaben werden als separate VTODOs mit Parent-Referenz modelliert.
- Unteraufgaben sollen in Nextcloud als Unteraufgaben sichtbar sein, soweit der CalDAV-Server das unterstützt.
- Andere Clients, die Parent-Referenzen ignorieren, sind kein Design-Blocker.

---

## 10. Labels und Kategorien

### 10.1 Labels

- Labels sind Pflicht.
- Labels werden über VTODO `CATEGORIES` synchronisiert.
- Neue Labels werden automatisch angelegt.
- Labels sind in Suche, Filterung, Schnellanlage und Aufgabenbearbeitung nutzbar.

### 10.2 Favoriten

- Favoriten sind Pflicht.
- Favoriten werden synchronisiert.
- Favoriten werden über die VTODO-Kategorie `STARRED` modelliert.
- Ein in Nextcloud gesetzter Stern muss in Caldo sichtbar sein, sofern Nextcloud beziehungsweise der CalDAV-Datenbestand `STARRED` als Kategorie enthält.
- Ein in Caldo gesetzter Stern muss als `STARRED` in CalDAV geschrieben werden.

---

## 11. Prioritäten

### 11.1 Prioritätsstufen

Caldo unterstützt genau drei Prioritätsstufen plus keine Priorität:

- `high`
- `medium`
- `low`
- keine Priorität

### 11.2 CalDAV-Abbildung

Die VTODO-Abbildung lautet:

- `high` → `PRIORITY:1`
- `medium` → `PRIORITY:5`
- `low` → `PRIORITY:9`
- keine Priorität → kein Priority-Wert oder neutraler Zustand

---

## 12. Wiederkehrende Aufgaben

### 12.1 Umfang im MVP

Wiederkehrende Aufgaben sind im MVP funktional zu unterstützen.

Pflichtmuster:

- täglich.
- wöchentlich.
- monatlich.
- jährlich.
- werktags.
- alle X Tage.
- alle X Wochen.
- alle X Monate.
- bestimmter Wochentag, z. B. jeden Montag.

### 12.2 Endbedingungen

Folgende Endbedingungen müssen unterstützt werden:

- nie.
- bis Datum.
- nach N Wiederholungen.

### 12.3 Erledigen wiederkehrender Aufgaben

- Beim Erledigen einer wiederkehrenden Aufgabe soll CalDAV-Standardverhalten verwendet werden.
- Die konkrete technische Umsetzung darf sich am VTODO-/CalDAV-kompatiblen Verhalten orientieren.
- Die App muss vermeiden, Wiederholungsregeln beim Bearbeiten anderer Felder zu zerstören.

### 12.4 Nicht im MVP

- Komplexe Ausnahmen einzelner Wiederholungen sind nicht MVP-Pflicht.
- Spezialfälle wie „jeden letzten Freitag im Monat“ sind nicht MVP-Pflicht, sofern sie nicht vom verwendeten CalDAV-Standardverhalten ohne Zusatzaufwand unterstützt werden.

---

## 13. Erinnerungen

- Erinnerungen sind nice-to-have.
- Bestehende CalDAV-Erinnerungen müssen im MVP erhalten bleiben.
- Erinnerungen werden im MVP nicht in der UI angezeigt.
- Erstellen und Bearbeiten von Erinnerungen ist nicht Bestandteil des MVP.

---

## 14. Anhänge und Links

- Bestehende Anhänge und Links aus VTODOs müssen erhalten bleiben.
- Links in Beschreibungen sollen automatisch klickbar sein.
- Datei-Anhänge werden angezeigt.
- Datei-Anhänge werden im MVP nicht aktiv verwaltet.
- Hochladen, Entfernen oder Bearbeiten von Datei-Anhängen ist nicht Bestandteil des MVP.

---

## 15. Schnell hinzufügen und natürliche Eingabe

### 15.1 Sprachen

Natürliche Eingabe muss unterstützen:

- Deutsch.
- Englisch.

### 15.2 Syntax

Die Schnellanlage soll Todoist-nah sein.

Pflichtsyntax:

- `#Projekt` für Projekt beziehungsweise CalDAV-Kalender.
- `@Label` für Label beziehungsweise VTODO-Kategorie.
- Priorität über `!high`, `!medium`, `!low` oder äquivalente klar dokumentierte Syntax.
- Natürliche Datumsangaben, z. B.:
  - `morgen`
  - `today`
  - `next monday`
  - `jeden Montag`
  - `every Monday`

### 15.3 Projektauflösung

Wenn der Nutzer `#UnbekanntesProjekt` eingibt:

- Die UI zeigt eine Vorschlagsliste.
- Die UI unterstützt das Anlegen eines neuen Projekts.
- Das Anlegen eines neuen Projekts legt einen neuen CalDAV-Kalender an.
- Ein unbekanntes Projekt darf nicht stillschweigend ignoriert werden.

### 15.4 Labelauflösung

Wenn der Nutzer `@NeuesLabel` eingibt:

- Das Label wird automatisch angelegt.
- Das Label wird als VTODO-Kategorie synchronisiert.

### 15.5 Wiederholungen in natürlicher Eingabe

- Eingaben wie `jeden Montag` oder `every Monday` müssen wiederkehrende VTODOs erzeugen.
- Die Eingabe darf nicht lediglich erkannt und anschließend abgelehnt werden.

---

## 16. Filter

### 16.1 Grundsatz

Filter sind Pflicht. Gespeicherte Filter sind lokal und werden nicht über CalDAV synchronisiert.

### 16.2 Speicherung

- Filter werden lokal in SQLite gespeichert.
- Filter können benannt werden.
- Filter können favorisiert werden.

### 16.3 Systemfilter

Systemfilter sind Pflicht. Mindestens erforderlich:

- Heute.
- Demnächst.
- Überfällig.
- Favoriten.
- Aufgaben ohne Datum.
- Erledigte Aufgaben, sofern sichtbar geschaltet.
- Konflikte.

### 16.4 Query-Syntax

Die Filter-Syntax ist Todoist-nah.

Pflichtoperatoren:

- `today`
- `overdue`
- `upcoming`
- `#Projekt`
- `@Label`
- `priority:high`
- `completed:false`
- `text:foo`
- `before:date`
- `after:date`
- `no date`

Boolesche Operatoren:

- `AND`
- `OR`
- `NOT`

Klammern sind im MVP nicht erforderlich.

---

## 17. Suche

### 17.1 Umfang

Die Suche ist Pflicht.

Sie muss Aufgaben finden über:

- Titel.
- Beschreibung.
- Label.
- Projektname.

### 17.2 Tastaturzugriff

Die Suche muss per Tastaturkürzel aufrufbar sein.

---

## 18. UI- und UX-Anforderungen

### 18.1 Allgemeiner Stil

- Die UI soll nah an Todoist angelehnt sein.
- Navigation und Bedienfluss sollen vertraut wirken.

### 18.2 Rendering

- Die UI ist serverseitig gerendert.
- Gezielte JavaScript-Interaktivität ist erlaubt und erwartet.

Zulässige und erwartete JavaScript-Interaktivität:

- Schnell hinzufügen.
- Suchvorschläge.
- Konflikt-Feldauswahl.
- Tastaturkürzel.
- Dynamische Einstellungen.
- Interaktive Filtereingabe.
- Projekt- und Label-Vorschläge.

### 18.3 Navigation

Die Hauptnavigation muss enthalten:

- Heute.
- Demnächst.
- Projekte.
- Labels.
- Filter.
- Favoriten.
- Suche.
- Konflikte.
- Einstellungen.

### 18.4 Dark Mode

- Dark Mode ist Pflicht.
- Weitere Themes sind nicht Bestandteil des MVP.

### 18.5 Browser-Unterstützung

Pflicht:

- Aktueller Firefox.
- Aktueller Chrome/Chromium.

Nice-to-have:

- Safari.

### 18.6 Geräte

- Desktop-Browser sind primär.
- Tablet-Layout ist nice-to-have.
- Mobile-Unterstützung ist nicht nötig.

### 18.7 Mehrere Tabs

- Mehrere offene Browser-Tabs desselben Nutzers müssen unterstützt werden.
- Es gibt keine Browser-Offline-Queue.
- Die App muss vermeiden, dass mehrere Tabs inkonsistente lokale Zustände erzeugen.

---

## 19. Tastaturkürzel

Tastaturkürzel sind Pflicht.

Mindestens erforderlich:

- Neue Aufgabe hinzufügen.
- Suche öffnen.
- Ansichten wechseln.
- Hilfe anzeigen.

Eine Tastaturhilfe muss verfügbar sein.

---

## 20. Einstellungen

Eine Einstellungen-Seite ist Pflicht.

Sie muss mindestens unterstützen:

### 20.1 CalDAV

- CalDAV-URL.
- Benutzername.
- Passwort oder App-Passwort.
- Verbindungstest.
- Kalenderauswahl.
- Default-Projekt.

### 20.2 Sync

- Sync-Intervall.
- Default: 15 Minuten.
- Manueller Sync-Zugriff.

### 20.3 UI

- Erledigte Aufgaben anzeigen/ausblenden.
- Demnächst-Zeitraum.
- Sprache beziehungsweise Sprachverhalten.
- Dark Mode.

### 20.4 Sicherheit

- Anzeige, ob Reverse-Proxy-Header erkannt wird.
- Anzeige, ob HTTPS aktiv ist.

---

## 21. Sync-Anforderungen

### 21.1 Sync-Richtung

- Sync ist bidirektional.
- CalDAV ist führend.
- Lokale Änderungen werden sofort zu CalDAV geschrieben.
- Remote-Änderungen werden über manuellen oder periodischen Sync geholt.

### 21.2 Sync-Trigger

Pflichttrigger:

- Manuell per Button.
- Periodisch.
- Sofort nach lokaler Änderung.

Nicht erforderlich:

- Serverseitiger Sync unabhängig vom offenen Browser.
- Browser-Offline-Sync.

### 21.3 Sync-Intervall

- Das Sync-Intervall ist konfigurierbar.
- Default: 15 Minuten.

### 21.4 Manueller Sync

- Ein Button „Jetzt synchronisieren“ muss sichtbar beziehungsweise leicht erreichbar sein.
- Der Nutzer sieht den letzten erfolgreichen Sync-Zeitpunkt.
- Der Nutzer sieht den aktuellen Sync-Status.

### 21.5 Status pro Aufgabe

Aufgaben müssen Sync-Status anzeigen können:

- synchronisiert.
- wird gespeichert.
- Fehler.
- Konflikt.
- lokal geändert, noch nicht erfolgreich gespeichert, falls temporär nötig.

### 21.6 Verhalten bei Fehlern

- Wenn CalDAV temporär nicht erreichbar ist, startet die App trotzdem.
- Die App zeigt einen Sync-Fehler an.
- Die App gilt nicht als erfolgreich synchronisiert.
- Lokale Änderungen gelten erst nach erfolgreichem CalDAV-Write als gespeichert.
- Bei Write-Fehlern muss der Nutzer den Fehler sehen.
- Formularinhalte sollen nach Möglichkeit erhalten bleiben, damit keine Eingaben verloren gehen.

### 21.7 Browser-Offline

- Es gibt keine echte Browser-Offline-Funktionalität.
- Die App ist keine PWA.
- Offline-Bearbeitung im Browser ist nicht Ziel.
- Es gibt keine dauerhafte lokale Browser-Queue.
- Robustheit bei temporären CalDAV-Ausfällen ist wichtiger als Browser-Offline-Fähigkeit.

---

## 22. Konfliktauflösung

### 22.1 Konfliktdefinition

Ein Konflikt liegt mindestens in folgenden Fällen vor:

- Dieselbe Aufgabe wurde lokal und remote geändert.
- Aufgabe wurde lokal geändert und remote gelöscht.
- Aufgabe wurde lokal gelöscht und remote geändert.
- Dieselbe Aufgabe wurde in mehreren Browser-Tabs widersprüchlich geändert.
- Projekt/Kalender wurde lokal und remote geändert.
- Projekt/Kalender wurde lokal geändert und remote gelöscht.
- Projekt/Kalender wurde lokal gelöscht und remote geändert.

### 22.2 Grundsatz

- Konflikte werden manuell gelöst.
- Datenverlustvermeidung ist wichtig, aber einfache Bedienung soll priorisiert bleiben.
- Automatischer Merge ist erlaubt, wenn sich Änderungen nicht widersprechen und keine Felder überschreiben.

### 22.3 Automatischer Merge

Automatisch gemerged werden dürfen Änderungen, die sich nicht gegenseitig widersprechen, z. B.:

- Unterschiedliche Felder wurden geändert.
- Ein Label wurde hinzugefügt, während ein anderes Feld geändert wurde.
- Beschreibung wurde remote geändert, Priorität lokal geändert.

Automatischer Merge ist nicht erlaubt, wenn:

- dasselbe Feld unterschiedlich geändert wurde.
- eine Seite gelöscht hat und die andere Seite geändert hat.
- die Änderung zu Datenverlust führen könnte.

### 22.4 Konfliktansicht

- Konfliktansicht pro Aufgabe ist Pflicht.
- Beim Öffnen einer konfliktbehafteten Aufgabe wird die Konfliktansicht angezeigt.
- Zusätzlich gibt es eine globale Konfliktansicht.
- Konflikte blockieren die betroffene Aufgabe bis zur Lösung.
- Konflikte blockieren nicht den Sync anderer Aufgaben.

### 22.5 Konfliktlösungsoptionen

Die Konfliktansicht muss unterstützen:

- lokale Version übernehmen.
- Remote-Version übernehmen.
- Felder manuell auswählen.
- beide Versionen als separate Aufgaben behalten.

### 22.6 Manuelle Feldauswahl

Mindestens folgende Felder müssen einzeln auswählbar sein:

- Titel.
- Beschreibung.
- Fälligkeitsdatum.
- Priorität.
- Labels.
- Projekt.
- Status.
- Unteraufgaben.

### 22.7 Konfliktversionen

- Konfliktrelevante Vorversionen werden 7 Tage gespeichert.
- Gelöste Konfliktversionen werden nach 7 Tagen gelöscht. Ungelöste Konflikte bleiben bis zur manuellen Auflösung erhalten.
- Versionen werden nur für Konflikte gespeichert, nicht für jede Änderung.

---

## 23. Performance-Anforderungen

### 23.1 Zielgrößen

Caldo soll ausgelegt sein auf:

- bis zu 10.000 Aufgaben.
- realistische Nutzung mit 200–400 Aufgaben.

### 23.2 Startzeit

- Die App soll nach Prozessstart innerhalb von maximal 5 Sekunden bereit sein, sofern SQLite verfügbar ist und keine Migrationen ausstehen.
- Die Weboberfläche soll ohne initialen erfolgreichen CalDAV-Sync nutzbar starten.
- Die erste UI-Ansicht soll bei bis zu 10.000 lokal gespeicherten Aufgaben innerhalb von maximal 2 Sekunden geladen werden.

### 23.3 Sync-Dauer

- Ein inkrementeller Sync ohne größere Änderungen soll bei 400 Aufgaben innerhalb von 10 Sekunden abgeschlossen sein.
- Ein Erstimport von 400 Aufgaben soll innerhalb von 30 Sekunden abgeschlossen sein, sofern der CalDAV-Server normal antwortet.
- Bei 10.000 Aufgaben muss der Sync robust durchlaufen, aber es gibt keine harte Dauerzusage.

### 23.4 Robustheit

- Sync muss ressourcenschonend arbeiten.
- Sync muss Timeouts verwenden.
- Sync muss Retry mit Backoff verwenden.
- Sync muss Schutz gegen versehentliche Sync-Schleifen enthalten.
- Langsame CalDAV-Server müssen sauber behandelt werden.

---

## 24. Internationalisierung

### 24.1 MVP-Sprachen

Caldo unterstützt im MVP:

- Deutsch.
- Englisch.

### 24.2 Architektur

- Die Architektur soll internationalisierungsfähig sein.
- UI-Texte sollen nicht hart über die Codebasis verstreut werden.
- Weitere Sprachen sollen später ohne grundlegenden Architekturumbau ergänzbar sein.
- Natürliche Eingabe ist separat zu betrachten und muss im MVP Deutsch und Englisch unterstützen.

---

## 25. User Stories

### 25.1 Setup

**Als Nutzer möchte ich Caldo hinter meinem Reverse Proxy betreiben, damit ich keine eigene Benutzerverwaltung in Caldo benötige.**

Akzeptanzkriterien:

- Requests ohne konfigurierten Auth-Header werden abgelehnt.
- Der Headername ist über Environment konfigurierbar.
- Es gibt keinen lokalen Login.

**Als Nutzer möchte ich meine CalDAV-Verbindung in der App konfigurieren, damit ich meine Nextcloud-Aufgaben synchronisieren kann.**

Akzeptanzkriterien:

- CalDAV-URL, Benutzername und Passwort/App-Passwort sind in Einstellungen konfigurierbar.
- Beim Speichern wird die Verbindung getestet.
- Ungültige Konfiguration wird nicht als erfolgreich gespeichert.
- Zugangsdaten werden verschlüsselt in SQLite gespeichert.

**Als Nutzer möchte ich beim Setup ein Default-Projekt auswählen oder neu anlegen, damit neue Aufgaben immer einem CalDAV-Kalender zugeordnet sind.**

Akzeptanzkriterien:

- Bestehende Kalender können als Default-Projekt gewählt werden.
- Ein neuer Kalender kann angelegt werden.
- Ohne Default-Projekt ist die produktive Nutzung nicht abgeschlossen.

### 25.2 Aufgabenverwaltung

**Als Nutzer möchte ich Aufgaben schnell erstellen, damit ich Gedanken ohne Reibung erfassen kann.**

Akzeptanzkriterien:

- Schnell hinzufügen ist per Tastaturkürzel erreichbar.
- Natürliche Datumsangaben Deutsch/Englisch werden erkannt.
- `#Projekt` setzt das Projekt.
- `@Label` setzt Labels.
- Prioritäten können über Schnellsyntax gesetzt werden.

**Als Nutzer möchte ich Aufgaben bearbeiten, damit ich Titel, Beschreibung, Fälligkeit, Projekt, Labels und Priorität ändern kann.**

Akzeptanzkriterien:

- Änderungen werden sofort zu CalDAV geschrieben.
- Die Änderung gilt erst nach erfolgreichem Write als gespeichert.
- Bei Fehlern wird eine sichtbare Fehlermeldung angezeigt.

**Als Nutzer möchte ich Aufgaben erledigen, damit sie in CalDAV als completed markiert werden.**

Akzeptanzkriterien:

- Erledigen setzt den CalDAV-Status auf completed.
- Erledigte Aufgaben sind standardmäßig ausgeblendet.
- Erledigte Aufgaben können per Einstellung angezeigt werden.

**Als Nutzer möchte ich Aufgaben löschen, damit ich nicht mehr benötigte Aufgaben entfernen kann.**

Akzeptanzkriterien:

- Vor dem Löschen erscheint eine Bestätigung.
- Löschen ist endgültig.
- Die Löschung wird zu CalDAV synchronisiert.
- Löschkonflikte werden erkannt.

### 25.3 Projekte

**Als Nutzer möchte ich CalDAV-Kalender als Projekte sehen, damit meine Nextcloud-Struktur erhalten bleibt.**

Akzeptanzkriterien:

- Jeder ausgewählte CalDAV-Kalender erscheint als Projekt.
- Neue Projekte legen CalDAV-Kalender an.
- Umbenennen eines Projekts benennt den Kalender um.
- Löschen eines Projekts löscht den Kalender nach starker Bestätigung.

### 25.4 Labels und Filter

**Als Nutzer möchte ich Labels verwenden, damit ich Aufgaben projektübergreifend organisieren kann.**

Akzeptanzkriterien:

- Labels werden als VTODO-Kategorien gespeichert.
- Neue Labels werden automatisch angelegt.
- Labels sind in Aufgaben, Suche und Filtern nutzbar.

**Als Nutzer möchte ich gespeicherte Filter anlegen, damit ich eigene Aufgabenansichten erstellen kann.**

Akzeptanzkriterien:

- Filter verwenden eine Todoist-nahe Query-Syntax.
- Filter werden lokal gespeichert.
- Filter können favorisiert werden.
- Boolesche Operatoren AND, OR und NOT werden unterstützt.

### 25.5 Konflikte

**Als Nutzer möchte ich Konflikte manuell lösen, damit keine Daten unbemerkt verloren gehen.**

Akzeptanzkriterien:

- Konflikte werden pro Aufgabe angezeigt.
- Es gibt eine globale Konfliktansicht.
- Konfliktbehaftete Aufgaben sind gesperrt.
- Andere Aufgaben synchronisieren weiter.
- Lokale Version, Remote-Version und manuelle Feldauswahl sind möglich.
- Beide Versionen können als separate Aufgaben behalten werden.

### 25.6 Sync

**Als Nutzer möchte ich manuell und automatisch synchronisieren, damit Änderungen aus Nextcloud und Caldo aktuell bleiben.**

Akzeptanzkriterien:

- Manueller Sync ist verfügbar.
- Periodischer Sync ist verfügbar.
- Default-Intervall ist 15 Minuten.
- Intervall ist konfigurierbar.
- Letzter erfolgreicher Sync-Zeitpunkt wird angezeigt.
- Sync-Fehler werden sichtbar angezeigt.

### 25.7 Wiederholungen

**Als Nutzer möchte ich wiederkehrende Aufgaben erstellen, damit regelmäßige Aufgaben automatisch verwaltet werden.**

Akzeptanzkriterien:

- Tägliche, wöchentliche, monatliche und jährliche Wiederholungen sind möglich.
- „Jeden Montag“ und vergleichbare Eingaben werden erkannt.
- Endbedingungen nie, bis Datum und nach N Wiederholungen sind möglich.
- Wiederholungsregeln bleiben bei Bearbeitung anderer Felder erhalten.

---

## 26. Akzeptanzkriterien Gesamt-MVP

### 26.1 CalDAV

Das MVP gilt im Bereich CalDAV als erfüllt, wenn:

- Ein CalDAV-Account eingerichtet werden kann.
- Verbindungstest funktioniert.
- Mehrere Kalender importiert werden.
- Kalender als Projekte erscheinen.
- Bestehende VTODOs importiert werden.
- Aufgaben erstellt, bearbeitet, erledigt und gelöscht werden können.
- Änderungen in Caldo in Nextcloud sichtbar werden.
- Änderungen in Nextcloud nach Sync in Caldo sichtbar werden.
- Unbekannte VTODO-Felder bei Bearbeitung bekannter Felder erhalten bleiben, soweit technisch möglich.
- Konflikte erkannt und manuell lösbar sind.

### 26.2 UI

Das MVP gilt im Bereich UI als erfüllt, wenn:

- Die Hauptnavigation Heute, Demnächst, Projekte, Labels, Filter, Favoriten, Suche, Konflikte und Einstellungen enthält.
- Aufgaben schnell erstellt werden können.
- Tastaturkürzel für neue Aufgabe, Suche, Ansichtenwechsel und Hilfe funktionieren.
- Schnellsyntax für Projekt, Label, Priorität und Datum funktioniert.
- Suche über Titel, Beschreibung, Label und Projektname funktioniert.
- Dark Mode verfügbar ist.
- Erledigte Aufgaben standardmäßig ausgeblendet sind.
- Sync- und Fehlerstatus sichtbar sind.

### 26.3 Betrieb

Das MVP gilt im Bereich Betrieb als erfüllt, wenn:

- Go-Binary gebaut werden kann.
- Docker-Container gebaut werden kann.
- Docker-Compose-Referenzdeployment vorhanden ist.
- SQLite verwendet wird.
- Pflicht-Environment-Variablen validiert werden.
- Ohne `ENCRYPTION_KEY` kein Start erfolgt.
- HTTPS-only-Verhalten durchgesetzt wird.
- Healthcheck verfügbar ist.
- Strukturierte Logs ohne sensible Inhalte erzeugt werden.

---

## 27. Risiken

### 27.1 CalDAV-Kompatibilität

CalDAV-Server und Clients interpretieren VTODO-Felder unterschiedlich. Besonders betroffen:

- Wiederholungen.
- Unteraufgaben.
- Parent-Referenzen.
- Erinnerungen.
- Anhänge.
- Favoriten über `STARRED`.

Gegenmaßnahme:

- Nextcloud als primären Zielserver testen.
- Unbekannte Felder erhalten.
- Nicht unterstützte Fremdclient-Verhalten dokumentieren.

### 27.2 Konfliktkomplexität

Manuelle Konfliktauflösung kann komplex werden, insbesondere bei Unteraufgaben, Wiederholungen und Löschfällen.

Gegenmaßnahme:

- Konfliktansicht aufgabenbezogen halten.
- Automatischen Merge nur für eindeutig konfliktfreie Felder erlauben.
- Globale Konfliktliste bereitstellen.

### 27.3 Wiederholungen

Wiederkehrende VTODOs können komplex sein und unterschiedliche CalDAV-Implementierungen haben.

Gegenmaßnahme:

- MVP-Muster klar begrenzen.
- Komplexe Ausnahmen ausschließen.
- Wiederholungsregeln bei Bearbeitung anderer Felder unverändert erhalten.

### 27.4 Sicherheit

Betrieb hinter Reverse Proxy kann fehleranfällig sein.

Gegenmaßnahme:

- HTTPS-only erzwingen.
- Auth-Header zwingend prüfen.
- Keine lokale Login-Fallback-Logik.
- Sensible Daten in Logs maskieren.

### 27.5 Performance bei großen Datenmengen

10.000 Aufgaben sind deutlich mehr als die realistische Nutzung von 200–400 Aufgaben.

Gegenmaßnahme:

- Lokale Indizes.
- Inkrementeller Sync.
- Batch-Verarbeitung.
- Fortschrittsanzeige bei langen Syncs.
- Timeouts und Backoff.

---

## 28. Offene Annahmen

Folgende Punkte sind Annahmen und sollten während der Umsetzung validiert werden:

1. Nextcloud unterstützt die benötigte Parent-Referenz für Unteraufgaben ausreichend.
2. `STARRED` als VTODO-Kategorie ist für Favoriten akzeptabel.
3. CalDAV-Standardverhalten für wiederkehrende Aufgaben ist ausreichend für das MVP.
4. Serverseitiges Rendering mit gezieltem JavaScript reicht für eine Todoist-nahe UX.
5. HTTPS-Erkennung funktioniert zuverlässig hinter dem vorgesehenen Reverse Proxy.
6. SQLite reicht für bis zu 10.000 Aufgaben bei Single-User-Betrieb aus.
7. Eine 7-tägige Aufbewahrung konfliktrelevanter Versionen ist ausreichend.
8. Firefox und Chrome/Chromium decken die primären Nutzeranforderungen ab.

---

## 29. Priorisierte Umsetzungsempfehlung

### Phase 1: Fundament

- Go-Binary.
- SQLite.
- Docker.
- Environment-Konfiguration.
- Reverse-Proxy-Auth.
- HTTPS-Prüfung.
- Strukturierte Logs.
- Healthcheck.

### Phase 2: CalDAV-Basis

- CalDAV-Konfiguration.
- Verbindungstest.
- Kalenderimport.
- Projektmapping.
- VTODO-Import.
- VTODO-Schreiben.
- Default-Projekt.

### Phase 3: Aufgaben-Kernmodell

- Aufgaben erstellen, bearbeiten, erledigen, löschen.
- Labels.
- Prioritäten.
- Fälligkeitsdaten.
- Beschreibungen.
- Suche.
- Heute/Demnächst/Überfällig.

### Phase 4: Sync und Konflikte

- Manueller Sync.
- Periodischer Sync.
- Sofortiger Write nach Änderung.
- Sync-Status.
- Konflikterkennung.
- Konfliktansicht.
- Globale Konfliktliste.
- Konfliktversionen.

### Phase 5: Todoist-nahe UX

- Schnell hinzufügen.
- Natürliche Eingabe Deutsch/Englisch.
- Tastaturkürzel.
- Filter-Query-Syntax.
- Gespeicherte und favorisierte Filter.
- Dark Mode.

### Phase 6: Erweiterte VTODO-Unterstützung

- Unteraufgaben über Parent-Referenz.
- Wiederholungen.
- Erhalt von Erinnerungen.
- Anzeige von Anhängen.
- Erhalt unbekannter Felder.

---

## 30. Glossar

**CalDAV**  
Protokoll zur Synchronisation von Kalender- und Aufgabeninformationen.

**VTODO**  
iCalendar-Komponente zur Darstellung einer Aufgabe.

**Projekt**  
In Caldo ein CalDAV-Kalender.

**Label**  
Kategorie einer Aufgabe, gespeichert über VTODO `CATEGORIES`.

**Favorit / Star**  
Markierung einer Aufgabe als Favorit, gespeichert über Kategorie `STARRED`.

**Default-Projekt**  
Projekt, in dem neue oder unzugewiesene Aufgaben erstellt werden.

**Konflikt**  
Situation, in der lokale und remote Änderungen nicht automatisch verlustfrei zusammengeführt werden können.

**Sofortiger Write**  
Lokale Änderung wird unmittelbar zu CalDAV geschrieben und gilt erst danach als gespeichert.

---
