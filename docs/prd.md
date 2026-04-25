# Caldo – Product Requirements Document

**Status:** Final MVP v1  
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
- Automatische SQLite-Schema-Migrationen beim App-Start.
- Erster-Start-Setup-Wizard für CalDAV, Kalenderauswahl, Default-Projekt und Initialimport.
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
- Undo für die letzte Undo-fähige Aktion.
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

### 6.2.1 SQLite-Schema-Migrationen

- SQLite-Schema-Migrationen müssen automatisch bei App-Start ausgeführt werden.
- Migrationen laufen immer automatisch und können nicht per Environment-Variable deaktiviert werden.
- Unterstützt werden nur Vorwärtsmigrationen.
- Downgrade- oder Rollback-Migrationen sind nicht Bestandteil des MVP.
- Vor der Ausführung von Migrationen muss automatisch ein Backup der SQLite-Datenbankdatei erstellt werden.
- Migrationen müssen transaktional ausgeführt werden, soweit SQLite dies für die jeweilige Änderung unterstützt.
- Eine fehlgeschlagene Migration darf nicht zu Datenverlust führen.
- Schlägt eine Migration fehl, darf die App nicht normal starten.
- Bei fehlgeschlagener Migration muss der App-Start hart abbrechen und eine klare Fehlermeldung in den Logs ausgeben.
- Die Weboberfläche darf bei fehlgeschlagener Migration nicht verfügbar sein.
- Eine fehlgeschlagene Migration darf keine teilweise migrierte, inkonsistente Datenbank als nutzbaren Zustand hinterlassen.
- Der Migrationsstatus beziehungsweise die aktuelle Schema-Version muss lokal nachvollziehbar gespeichert werden.

Akzeptanzkriterien:

- Beim Start mit veraltetem Schema wird die Migration automatisch ausgeführt.
- Vor der Migration wird ein Datenbank-Backup erstellt.
- Nach erfolgreicher Migration startet die App normal.
- Bei fehlerhafter Migration startet die App nicht normal.
- Bei fehlerhafter Migration bleiben ursprüngliche Daten über das Backup wiederherstellbar.
- Migrationen werden in den strukturierten Logs ohne sensible Nutzdaten protokolliert.

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

### 7.4.1 Erster-Start-Flow und Onboarding

Wenn Caldo ohne vollständige Erstkonfiguration startet, wird der authentifizierte Nutzer nach erfolgreicher Reverse-Proxy-Authentifizierung automatisch in einen Setup-Wizard geleitet.

Die normale Todo-UI ist bis zum erfolgreichen Abschluss des Setup-Wizards gesperrt und nicht nutzbar.

Der Setup-Wizard muss mindestens folgende Schritte enthalten:

1. Systemcheck:
   - `BASE_URL` ist gesetzt.
   - `ENCRYPTION_KEY` ist gesetzt.
   - `PROXY_USER_HEADER` ist gesetzt.
   - Reverse-Proxy-Auth-Header wird erkannt.
   - Die HTTPS-Anforderung wird gemäß Abschnitt 6.5 geprüft.
2. CalDAV-Konfiguration:
   - CalDAV-URL eingeben.
   - Benutzername eingeben.
   - Passwort oder App-Passwort eingeben.
3. Verbindungstest:
   - Die CalDAV-Verbindung wird getestet.
   - Bei Fehlschlag bleibt der Nutzer im Wizard.
   - Eine unvollständige oder ungültige CalDAV-Konfiguration darf nicht als abgeschlossen gespeichert werden.
4. Kalenderauswahl:
   - Vorhandene CalDAV-Kalender werden geladen.
   - Der Nutzer wählt die zu synchronisierenden Kalender aus.
5. Default-Projekt:
   - Der Nutzer wählt einen bestehenden Kalender als Default-Projekt.
   - Alternativ kann der Nutzer ein neues Projekt beziehungsweise einen neuen CalDAV-Kalender anlegen.
6. Initialimport:
   - Der initiale CalDAV-Import wird im Wizard gestartet.
   - Nach erfolgreichem Initialimport wird der Nutzer in die normale App-UI weitergeleitet.

Nach abgeschlossenem Onboarding wird der Wizard nicht als eigener Wizard erneut geöffnet. Spätere Änderungen erfolgen über die normalen Einstellungen.

Akzeptanzkriterien:

- Eine frische Installation ohne CalDAV-Konfiguration zeigt nach Authentifizierung den Setup-Wizard.
- Die normale Todo-UI ist vor Abschluss des Wizards nicht erreichbar.
- Bei fehlgeschlagenem CalDAV-Verbindungstest bleibt der Nutzer im Wizard.
- Ohne gewähltes oder neu angelegtes Default-Projekt kann der Wizard nicht abgeschlossen werden.
- Der Initialimport wird im Wizard gestartet.
- Nach erfolgreichem Initialimport wird die normale UI angezeigt.
- Spätere Konfigurationsänderungen erfolgen über die Einstellungen, nicht über den Erststart-Wizard.

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
- Wenn eine Elternaufgabe offene Unteraufgaben enthält, darf sie nicht stillschweigend inklusive Unteraufgaben erledigt werden.
- Beim Erledigen einer Elternaufgabe mit offenen Unteraufgaben muss ein Todoist-ähnlicher Dialog erscheinen.
- Der Dialog muss den Nutzer explizit wählen lassen:
  - nur die Elternaufgabe erledigen,
  - Elternaufgabe und alle offenen Unteraufgaben erledigen,
  - Aktion abbrechen.
- Die gewählte Aktion muss wie jede andere Änderung sofort zu CalDAV geschrieben werden.

### 8.6 Löschen

- Das Löschen einer Aufgabe ist endgültig.
- Vor dem Löschen einer Aufgabe reicht eine normale Bestätigung.
- Es gibt keinen Papierkorb.
- Gelöschte Aufgaben können bei Konflikten dennoch zu Konfliktfällen führen.

### 8.7 Undo

Eine einfache Undo-Funktion für die letzte Undo-fähige Aktion ist Pflicht.

Undo ist kein rein lokaler UI-Rollback. Da lokale Änderungen erst nach erfolgreichem CalDAV-Write als gespeichert gelten, muss auch ein Undo als neue fachliche Gegenänderung behandelt werden.

Undo muss mindestens folgende Aktionen unterstützen:

- Aufgabe erledigen.
- Aufgabe bearbeiten.
- Aufgabe löschen.
- Projektwechsel.
- Labeländerung.

Anforderungen:

- Undo gilt nur für die letzte Undo-fähige Aktion pro Browser-Session.
- Für das MVP bedeutet Browser-Session im Undo-Kontext: der jeweilige Browser-Tab beziehungsweise dessen serverseitig zugeordnete UI-Session.
- Undo-Snapshots sind tab-lokal.
- Änderungen in anderen Tabs invalidieren den Undo-Snapshot des aktuellen Tabs nicht automatisch, können aber beim Ausführen des Undos einen Konflikt erzeugen.
- Undo bleibt nach einem Seitenreload innerhalb derselben Browser-Session verfügbar.
- Undo ist maximal 5 Minuten verfügbar oder bis zur nächsten Undo-fähigen Aktion, je nachdem, was zuerst eintritt.
- Für Undo muss ein serverseitiger Undo-Snapshot der vorherigen Aufgabenfassung vorgehalten werden.
- Das Ausführen von Undo erzeugt eine neue Änderung, die sofort zu CalDAV geschrieben wird.
- Ein Undo gilt erst nach erfolgreichem CalDAV-Write als abgeschlossen.
- Wenn der CalDAV-Write des Undos fehlschlägt, wird ein Fehler angezeigt und der aktuelle Zustand bleibt bestehen.
- Es gibt keine ausstehende Undo-Queue.
- Wenn die Aufgabe zwischen ursprünglicher Änderung und Undo remote verändert wurde, muss ein Konflikt erzeugt werden.
- Undo für Löschen bedeutet, dass die Aufgabe aus dem Undo-Snapshot neu erstellt wird.
- Löschen wird sofort zu CalDAV geschrieben und nicht verzögert ausgeführt.
- Undo darf keine stillen lokalen Zustände erzeugen, die nicht mit CalDAV abgeglichen sind.

Akzeptanzkriterien:

- Nach einer Undo-fähigen Aktion wird eine Undo-Möglichkeit angezeigt.
- Undo verschwindet nach 5 Minuten oder nach der nächsten Undo-fähigen Aktion.
- Nach Reload derselben Browser-Session ist Undo weiterhin verfügbar, sofern Zeitlimit und Aktionslimit nicht überschritten sind.
- Ein erfolgreicher Undo ist in CalDAV sichtbar.
- Ein fehlgeschlagener Undo zeigt eine Fehlermeldung und verändert den gespeicherten Zustand nicht stillschweigend.
- Ein Undo nach zwischenzeitlicher Remote-Änderung erzeugt einen Konflikt.
- Eine gelöschte Aufgabe kann per Undo aus dem Snapshot neu erstellt werden.

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

### 17.1 Grundsatz

Die Suche ist Pflicht und ist als globale Freitextsuche zu verstehen.

Die globale Suche und die Filter-Query-Syntax sind getrennte UI-Konzepte:

- Die globale Suche dient dem schnellen Finden aktiver Aufgaben.
- Die Filter-Query-Syntax dient gespeicherten, systematischen Aufgabenansichten.
- Beide Funktionen dürfen intern dieselbe Such- oder Query-Engine nutzen.
- In der UI müssen sie als unterschiedliche Eingaben beziehungsweise Nutzungskontexte erkennbar bleiben.

### 17.2 Umfang

Die globale Suche muss aktive Aufgaben finden über:

- Titel.
- Beschreibung.
- Label.
- Projektname.

Standardmäßig durchsucht die globale Suche keine erledigten Aufgaben.

Standardmäßig durchsucht die globale Suche nicht:

- erledigte Aufgaben,
- Konfliktversionen,
- technische Undo-Snapshots,
- historische Versionen.

### 17.3 Strukturierte Tokens in der Suche

Die globale Suche muss einfache strukturierte Tokens erkennen, ohne die vollständige Filter-Query-Syntax ersetzen zu müssen.

Mindestens zu erkennen:

- `#Projekt`
- `@Label`

Diese Tokens dürfen verwendet werden, um Freitextsuche und einfache Einschränkungen zu kombinieren.

Beispiel:

- `rechnung #Finanzen`
- `arzt @wichtig`

### 17.4 Verhältnis zu gespeicherten Filtern

- Gespeicherte Filter verwenden die Filter-Query-Syntax aus Abschnitt 16.
- Aus einer Suche heraus darf ein gespeicherter Filter erstellt werden, wenn die Suchanfrage eindeutig in eine gültige Filter-Query übersetzt werden kann.
- Wenn eine Suchanfrage nicht eindeutig als Filter-Query interpretierbar ist, darf sie nicht stillschweigend als gespeicherter Filter übernommen werden.

### 17.5 Tastaturzugriff

Die Suche muss per Tastaturkürzel aufrufbar sein.

Akzeptanzkriterien:

- Die globale Suche ist als Freitextsuche nutzbar.
- Filter-Query und globale Suche sind in der UI unterscheidbar.
- Die Suche findet aktive Aufgaben über Titel, Beschreibung, Label und Projektname.
- Erledigte Aufgaben werden standardmäßig nicht durchsucht.
- Historische Konfliktversionen und Undo-Snapshots werden nicht durchsucht.
- Einfache Tokens wie `#Projekt` und `@Label` werden erkannt.
- Aus einer Suche kann ein Filter erstellt werden, sofern die Eingabe eindeutig in eine gültige Filter-Query überführbar ist.

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

Mehrere offene Browser-Tabs desselben Nutzers müssen unterstützt werden.

Caldo verwendet dafür ein Modell aus serverseitiger Versionsprüfung, Fokus-Refresh und Server-Sent Events (SSE).

Anforderungen:

- Es gibt keine Browser-Offline-Queue.
- Es gibt kein pessimistisches Locking beim Bearbeiten.
- Aufgaben werden beim Bearbeiten nicht exklusiv durch einen Tab gesperrt.
- Jeder schreibende Request muss die dem Browser bekannte Version der Aufgabe beziehungsweise Ressource enthalten.
- Wenn die Server-Version von der bekannten Browser-Version abweicht, darf die Änderung nicht stillschweigend überschrieben werden.
- Bei Fokuswechsel zurück in einen Tab muss die Ansicht aktualisiert beziehungsweise auf veraltete Daten geprüft werden.
- SSE soll offene Tabs über relevante Änderungen informieren.
- Wenn ein anderer Tab dieselbe Aufgabe geändert hat, während ein Tab keine lokalen ungespeicherten Formularänderungen hat, darf die Ansicht automatisch aktualisiert werden.
- Wenn ein Tab lokale ungespeicherte Formularänderungen hat, darf die Ansicht nicht automatisch überschrieben werden.
- Wenn beim Speichern eine Versionsabweichung erkannt wird, muss ein Konflikt oder ein Aktualisierungshinweis erzeugt werden.
- Mehr-Tab-Konflikte werden nach denselben Grundprinzipien behandelt wie CalDAV-Konflikte.
- Wenn ein Mehr-Tab-Fall als echter Konflikt klassifiziert wird, muss er in der globalen Konfliktansicht erscheinen; reine Aktualisierungshinweise ohne widersprüchliche lokale Änderung müssen nicht in der globalen Konfliktansicht erscheinen.

Akzeptanzkriterien:

- Zwei geöffnete Tabs zeigen nach einer Änderung in einem Tab zeitnah konsistente Daten oder einen Aktualisierungshinweis.
- Ein Tab überschreibt keine neuere Server-Version stillschweigend.
- Ein Tab mit lokalen ungespeicherten Änderungen wird nicht automatisch überschrieben.
- Bei Speichern auf Basis einer veralteten Version wird ein Konflikt oder ein klarer Aktualisierungshinweis angezeigt.
- Es gibt keine Bearbeitungssperre nur aufgrund eines geöffneten Tabs.
- Fokus-Refresh aktualisiert lange offene Tabs beim Zurückkehren.

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

Die Einstellungen dienen nach abgeschlossenem Erststart-Wizard zur späteren Änderung der Konfiguration. Der Erststart selbst erfolgt über den Setup-Wizard gemäß Abschnitt 7.4.1.

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
- UI-Sprache-Umschaltung Deutsch/Englisch; diese Auswahl beeinflusst auch die natürliche Eingabe.
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
- Ausstehende Writes, die beim Schließen des Browsers noch nicht erfolgreich abgeschlossen sind, werden beim nächsten Öffnen nicht automatisch nachgesendet.
- Da Änderungen erst nach erfolgreichem CalDAV-Write als gespeichert gelten, muss die UI bei noch laufenden Writes sichtbar machen, dass der Speichervorgang noch nicht abgeschlossen ist.
- Beim Verlassen oder Schließen einer Seite mit laufendem Write soll der Browser, soweit technisch möglich, eine Warnung anzeigen.


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
- Echte Mehr-Tab-Konflikte erscheinen ebenfalls in der globalen Konfliktansicht.
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
- Bei laufenden Migrationen ist die Startzeit nicht hart begrenzt, solange die Migration aktiv Fortschritt zeigt und kein Timeout oder Fehler ausgelöst wird.
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

Die App muss eine UI-Sprache-Umschaltung zwischen Deutsch und Englisch bereitstellen. Die gewählte Sprache beeinflusst auch die natürliche Eingabe, insbesondere Datumsausdrücke, Wiederholungen und Schnellsyntax-Hilfetexte.

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

**Als Nutzer möchte ich beim ersten Start durch einen Setup-Wizard geführt werden, damit Caldo erst nach vollständiger CalDAV-Konfiguration nutzbar ist.**

Akzeptanzkriterien:

- Eine unkonfigurierte Installation leitet nach erfolgreicher Reverse-Proxy-Authentifizierung in den Setup-Wizard.
- Die normale Todo-UI ist bis zum Abschluss des Wizards gesperrt.
- CalDAV-Konfiguration, Verbindungstest, Kalenderauswahl, Default-Projekt und Initialimport sind Teil des Wizards.
- Bei fehlgeschlagenem Verbindungstest bleibt der Nutzer im Wizard.
- Nach erfolgreichem Initialimport wird die normale UI geöffnet.
- Spätere Änderungen erfolgen über Einstellungen.

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

**Als Nutzer möchte ich beim Erledigen einer Elternaufgabe mit offenen Unteraufgaben gefragt werden, was passieren soll, damit ich nicht versehentlich Unteraufgaben miterledige oder offen lasse.**

Akzeptanzkriterien:

- Bei offenen Unteraufgaben erscheint ein Dialog.
- Der Nutzer kann nur die Elternaufgabe erledigen.
- Der Nutzer kann Elternaufgabe und offene Unteraufgaben erledigen.
- Der Nutzer kann die Aktion abbrechen.
- Die gewählte Aktion wird sofort zu CalDAV geschrieben.

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

### 25.6 Undo, Suche und Mehr-Tab-Nutzung

**Als Nutzer möchte ich die letzte Änderung rückgängig machen, damit ich versehentliche Aktionen korrigieren kann.**

Akzeptanzkriterien:

- Undo ist für die letzte Undo-fähige Aktion pro Browser-Session verfügbar.
- Undo bleibt nach Reload innerhalb derselben Browser-Session für bis zu 5 Minuten verfügbar.
- Undo wird als neue Änderung sofort zu CalDAV geschrieben.
- Bei fehlgeschlagenem Undo-Write wird ein Fehler angezeigt.
- Bei zwischenzeitlicher Remote-Änderung erzeugt Undo einen Konflikt.
- Undo für Löschen erstellt die Aufgabe aus dem Snapshot neu.

**Als Nutzer möchte ich eine globale Suche nutzen, damit ich aktive Aufgaben schnell finde.**

Akzeptanzkriterien:

- Die globale Suche ist von gespeicherten Filtern unterscheidbar.
- Die Suche durchsucht standardmäßig nur aktive Aufgaben.
- Einfache Tokens wie `#Projekt` und `@Label` werden erkannt.
- Aus einer Suche kann ein Filter erstellt werden, wenn die Eingabe eindeutig in eine Filter-Query überführbar ist.

**Als Nutzer möchte ich Caldo in mehreren Tabs nutzen können, damit ich parallel arbeiten kann, ohne Daten stillschweigend zu überschreiben.**

Akzeptanzkriterien:

- Änderungen in einem Tab werden über SSE oder Fokus-Refresh in anderen Tabs sichtbar.
- Schreibvorgänge prüfen serverseitige Versionen.
- Veraltete Tabs überschreiben keine neueren Daten stillschweigend.
- Bei Versionskonflikten wird ein Konflikt oder Aktualisierungshinweis angezeigt.
- Es gibt keine pessimistischen Bearbeitungssperren.

### 25.7 Sync

**Als Nutzer möchte ich manuell und automatisch synchronisieren, damit Änderungen aus Nextcloud und Caldo aktuell bleiben.**

Akzeptanzkriterien:

- Manueller Sync ist verfügbar.
- Periodischer Sync ist verfügbar.
- Default-Intervall ist 15 Minuten.
- Intervall ist konfigurierbar.
- Letzter erfolgreicher Sync-Zeitpunkt wird angezeigt.
- Sync-Fehler werden sichtbar angezeigt.

### 25.8 Wiederholungen

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
- Der Erststart-Setup-Wizard vor produktiver Nutzung abgeschlossen werden muss.
- Undo für die letzte Undo-fähige Aktion verfügbar ist und als CalDAV-Write ausgeführt wird.
- Mehrere Tabs ohne stilles Überschreiben neuerer Daten unterstützt werden.
- Beim Erledigen einer Elternaufgabe mit offenen Unteraufgaben erscheint ein Dialog zur Auswahl des gewünschten Verhaltens.
- Die UI-Sprache kann zwischen Deutsch und Englisch umgeschaltet werden und beeinflusst die natürliche Eingabe entsprechend.

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
- SQLite-Schema-Migrationen beim App-Start automatisch ausgeführt werden.
- Vor Migrationen ein SQLite-Backup erstellt wird.
- Eine fehlgeschlagene Migration den normalen App-Start verhindert und keine Daten stillschweigend beschädigt.

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

### 27.6 Migrationen

Fehlerhafte SQLite-Schema-Migrationen können die lokale Datenbank beschädigen oder den Start verhindern.

Gegenmaßnahme:

- Automatische Backups vor Migrationen.
- Vorwärtsmigrationen.
- Transaktionale Ausführung, soweit SQLite dies unterstützt.
- Harter Startabbruch bei fehlgeschlagener Migration.
- Klare Logs ohne sensible Daten.

### 27.7 Browser-Schließen bei laufendem Write

Wenn der Nutzer den Browser oder Tab schließt, während ein CalDAV-Write noch läuft, kann der Write abbrechen. Da Caldo keine Browser-Offline-Queue bereitstellt, werden solche ausstehenden Writes beim nächsten Öffnen nicht automatisch nachgesendet.

Gegenmaßnahme:

- Änderungen gelten erst nach erfolgreichem CalDAV-Write als gespeichert.
- Laufende Writes müssen in der UI klar sichtbar sein.
- Beim Verlassen oder Schließen einer Seite mit laufendem Write soll, soweit technisch möglich, eine Browser-Warnung angezeigt werden.
- Dieses Verhalten muss in der Produkt- und Nutzungsdokumentation beschrieben werden.

## 28. Offene Annahmen

Folgende Punkte sind Annahmen, bekannte Einschränkungen oder Validierungsaufgaben und sollten während der Umsetzung berücksichtigt werden:

1. Nextcloud unterstützt die benötigte Parent-Referenz für Unteraufgaben ausreichend.
2. `STARRED` als VTODO-Kategorie ist für Favoriten akzeptabel.
3. CalDAV-Standardverhalten für wiederkehrende Aufgaben ist ausreichend für das MVP.
4. Serverseitiges Rendering mit gezieltem JavaScript reicht für eine Todoist-nahe UX.
5. Es ist eine bekannte Einschränkung, dass Deployments hinter HTTPS-Proxy mit `http://` in `BASE_URL` geblockt werden. Die Dokumentation muss explizit beschreiben, dass `BASE_URL` immer `https://` tragen muss, auch wenn der interne Traffic zwischen Reverse Proxy und Caldo unverschlüsselt läuft.
6. SQLite reicht für bis zu 10.000 Aufgaben bei Single-User-Betrieb aus.
7. Eine 7-tägige Aufbewahrung konfliktrelevanter Versionen ist ausreichend.
8. Firefox und Chrome/Chromium decken die primären Nutzeranforderungen ab.

---

## 29. Priorisierte Umsetzungsempfehlung

### Phase 1: Fundament

- Go-Binary.
- SQLite.
- Automatische SQLite-Migrationen.
- Docker.
- Environment-Konfiguration.
- Reverse-Proxy-Auth.
- HTTPS-Prüfung.
- Strukturierte Logs.
- Healthcheck.

### Phase 2: CalDAV-Basis

- CalDAV-Konfiguration.
- Erster-Start-Setup-Wizard.
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
- Trennung von globaler Suche und Filter-Query.
- Heute/Demnächst/Überfällig.

### Phase 4: Sync und Konflikte

- Manueller Sync.
- Periodischer Sync.
- Sofortiger Write nach Änderung.
- Sync-Status.
- Undo mit serverseitigem Snapshot und CalDAV-Write.
- Mehr-Tab-Versionierung, SSE und Fokus-Refresh.
- Konflikterkennung.
- Konfliktansicht.
- Globale Konfliktliste.
- Konfliktversionen.

### Phase 5: Todoist-nahe UX

- Schnell hinzufügen.
- Natürliche Eingabe Deutsch/Englisch.
- UI-Sprache-Umschaltung Deutsch/Englisch.
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

**Setup-Wizard**  
Erster-Start-Ablauf für Systemcheck, CalDAV-Konfiguration, Kalenderauswahl, Default-Projekt und Initialimport.

**Undo-Snapshot**  
Kurzzeitig gespeicherte vorherige Aufgabenfassung, aus der die letzte Undo-fähige Aktion pro Browser-Session rückgängig gemacht werden kann.

**Fokus-Refresh**  
Aktualisierung oder Versionsprüfung eines Browser-Tabs, wenn der Nutzer zu diesem Tab zurückkehrt.

**Server-Sent Events (SSE)**  
Einseitiger Server-zu-Browser-Kanal, über den offene Tabs über relevante Änderungen informiert werden können.

---
