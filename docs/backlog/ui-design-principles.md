# Caldo UI Design Principles

Stand: 2026-06-08

Dieses Dokument legt die visuelle Richtung fuer die UI-Stories ab Epic 23 fest. Es beschreibt Produkt- und Designentscheidungen, keine technische Implementierung.

## Zielbild

Caldo ist eine produktive Aufgaben-App fuer wiederholte taegliche Nutzung. Die Oberflaeche soll ruhig, schnell erfassbar und Todoist-nah wirken. Sie ist keine Landing Page, keine Administrationskonsole und kein dekoratives Dashboard.

Die UI priorisiert:

* schnelles Erfassen und Abarbeiten von Aufgaben
* klare Navigation zwischen Heute, Demnaechst, Projekten, Labels, Filtern, Favoriten, Suche, Konflikten und Einstellungen
* sichtbaren Sync-, Write-, Fehler- und Konfliktstatus
* hohe Informationsdichte ohne visuelle Enge
* konsistente Tastatur-, Maus- und Fokusbedienung

## Todoist-Nahe Richtung

Caldo soll sich auch visuell an Todoist orientieren. Das bedeutet vertraute Muster bei Shell, Sidebar, Aufgabenlisten, Schnellanlage, Detailansichten, Metadaten und Aktionszustaenden.

Todoist-nahe Muster fuer Caldo:

* linke Sidebar mit Systemlisten, Projekten, Labels und Filtern
* kompakte Aufgabenlisten als primaere Arbeitsflaeche
* Aufgabenzeilen mit Checkbox links, Titel als Hauptinhalt und Metadaten darunter oder rechts
* dezente Hover-Aktionen, die Listen nicht dauerhaft ueberladen
* globale Schnellanlage als zentrale Aktion
* klare Dialoge fuer Quick Add, Unteraufgabenentscheidungen und Konfliktloesung
* rote bis korallrote Akzentfarbe fuer Primaeraktionen, aktive Zustaende und wichtige Hinweise
* ruhige neutrale Flaechen, feine Trennlinien und wenige Schatten

Caldo verwendet keine Todoist-Logos, keine fremden Assets, keine kopierten Texte und keine pixelgenaue Nachbildung. Ziel ist vertraute Produktqualitaet und Bedienlogik, nicht Markenimitation.

## Desktop-First

Die naechste UI-Phase ist desktop-first. Desktop-Browser sind der primaere Entwurfs- und QA-Kontext.

Desktop:

* Sidebar und Hauptbereich sind gleichzeitig sichtbar.
* Aufgabenlisten sind die Hauptansicht, nicht Kartenraster.
* Quick Add, Suche, manueller Sync und Einstellungen sind ohne Umweg erreichbar.
* Detail- und Konfliktansichten duerfen seitlich oder als fokussierter Dialog erscheinen, solange die Aufgabenliste nicht instabil springt.

Tablet:

* Tablet bleibt vorerst ein Qualitaetsziel, aber nicht der erste Gestaltungsanker.
* Layouts duerfen vereinfachen, muessen aber Kernnavigation, Aufgabenliste und Dialoge ohne Ueberlappung zeigen.

Mobile:

* Mobile ist fuer diese UI-Phase dokumentiert, aber nicht aktiv priorisiert.
* Mobile Breiten sollen nicht unbenutzbar brechen, muessen aber nicht den vollen Todoist-nahen Bedienkomfort erreichen.

## Dichte

Caldo verwendet Todoist-nahe Dichte. Aufgabenlisten sollen viele Eintraege scanbar zeigen und wiederholte Arbeit nicht durch uebergrosse Flaechen bremsen.

Richtlinien:

* Aufgabenzeilen sind kompakt, aber mit ausreichender Klick- und Fokusflaeche.
* Sekundaere Metadaten bleiben klein und visuell zurueckhaltend.
* Projekt-, Label-, Favorit- und Faelligkeitsinformationen sind sichtbar, ohne den Titel zu verdraengen.
* Leere Zustaende sind knapp und handlungsorientiert.
* Einstellungen und Konflikte duerfen mehr Raum nutzen, bleiben aber arbeitsfokussiert.
* Karten werden nur fuer einzelne wiederholte Objekte, Dialoge oder klar abgegrenzte Werkzeuge verwendet.

## Farbe Und Kontrast

Die visuelle Basis ist neutral. Die Akzentfarbe ist Todoist-nah rot bis korallrot und wird sparsam eingesetzt.

Akzentfarbe wird verwendet fuer:

* primaere Aktionen
* aktive Navigationszustaende
* Fokus auf wichtige Entscheidungen
* kritische Sync-, Fehler- oder Konflikthinweise, wenn fachlich passend

Neutrale Farben tragen den Grossteil der UI:

* helle Flaechen fuer Inhalt
* leicht abgesetzte Sidebar
* feine Linien fuer Trennung
* klare Textkontraste fuer Titel, Metadaten und deaktivierte Zustaende

Dark Mode ist Pflicht. Er folgt derselben Hierarchie: neutraler Grund, dezente Trennung, rote Akzente, klare Kontraste.

## Typografie

Die Typografie ist funktional und zurueckhaltend. Sie unterstuetzt schnelles Scannen, nicht Editorial- oder Marketingwirkung.

Richtlinien:

* Systemnahe Sans-Serif-Schrift.
* Aufgaben- und Navigationslabels stehen im Vordergrund.
* Seitentitel sind klar, aber nicht hero-artig.
* Metadaten, Fehlerdetails und Hilfetexte sind kleiner und ruhiger.
* Text darf in Buttons, Chips, Aufgabenzeilen und Dialogen nicht ueberlaufen.

## Aufgabenzeilen

Die Aufgabenzeile ist die wichtigste Komponente der normalen App.

Visuelle Prioritaet:

1. Completion-Zustand und Titel
2. Faelligkeit, Projekt, Labels, Prioritaet und Favorit
3. Sync-, Fehler-, Konflikt- und Wiederholungsstatus
4. Hover- oder Fokusaktionen fuer Bearbeiten, Loeschen und weitere Optionen

Aufgabenzeilen sollen:

* links eine klare Completion-Kontrolle zeigen
* Titel und relevante Metadaten ohne Ueberladung anzeigen
* erledigte Aufgaben sichtbar anders darstellen, wenn sie eingeblendet sind
* laufende Writes und Fehler direkt an der betroffenen Aufgabe erkennbar machen
* lange Titel stabil kuerzen oder umbrechen
* mit Tastaturfokus genauso eindeutig wirken wie mit Hover

## Sidebar Und Navigation

Die Sidebar ist Orientierung und Arbeitsstartpunkt. Sie bleibt ruhig, wiedererkennbar und scanbar.

Prioritaet:

1. Systemlisten: Inbox, Heute, Demnaechst, Suche, Favoriten
2. Projekte
3. Labels und Filter
4. Konflikte und Einstellungen

Navigationszustaende:

* aktive Eintraege sind klar, aber nicht laut
* Zaehler sind kompakt und nur sichtbar, wenn fachlich korrekt
* lange Namen verursachen keine Layoutspruenge
* leere Gruppen wirken nicht wie Fehler
* Konflikte duerfen sichtbarer priorisiert werden als normale Zaehler

## Dialoge Und Panels

Dialoge und Panels sind Arbeitsflaechen fuer Entscheidungen, nicht dekorative Karten.

Quick Add:

* fokussiert auf Eingabe und Vorschau
* Todoist-nah kompakt
* mit Projekt-, Label-, Datums-, Prioritaets- und Wiederholungsfeedback

Aufgabendetail:

* geeignet fuer Beschreibung, Wiederholung, Anhaenge, Labels und groessere Bearbeitungen
* kompakt genug, um den Aufgabenlisten-Kontext nicht zu verlieren

Konflikte:

* Unterschiede muessen feldweise verstaendlich sein
* rohe VTODO-Daten sind nicht die primaere Entscheidungsflaeche
* lokale, entfernte und manuelle Auswahl muessen visuell eindeutig getrennt sein

Einstellungen:

* ruhig und formularorientiert
* klare Gruppen fuer CalDAV, Sync, UI und Sicherheit
* sensible Werte werden nie unnoetig sichtbar gemacht

## Statusmeldungen

Caldo darf nie so wirken, als sei eine lokale Aenderung fachlich gespeichert, bevor der CalDAV-Write erfolgreich war.

Statusprioritaet:

1. laufende Writes
2. Fehler
3. Konflikte
4. veraltete Tabs oder stale Versionen
5. Sync-Erfolg und letzte Aktualisierung

Statusmeldungen sind knapp, sichtbar und ohne private Aufgabeninhalte. Fehler nennen Aktion und Fehlerklasse, nicht Task-Beschreibung, Credentials oder Rohdaten.

## Interaktion

Die UI bleibt serverseitig gerendert und wird gezielt interaktiv ergaenzt.

Interaktionsprinzipien:

* Tastaturpfade sind fuer neue Aufgabe, Suche, Ansichtenwechsel und Hilfe Pflicht.
* Fokuszustaende sind sichtbar und stabil.
* Mutierende Aktionen zeigen pending, success oder error eindeutig.
* Mehrere Tabs duerfen neue Daten nicht still ueberschreiben.
* Kein dauerhafter lokaler Browserzustand fuer fachliche Daten.
* Hover-Aktionen haben eine Tastatur- oder sichtbare Alternative.

## UI-Review-Checkliste

Eine spaetere UI-Story sollte diese Fragen positiv beantworten:

* Wirkt die Ansicht wie eine produktive Aufgaben-App statt wie eine Marketing- oder Admin-Seite?
* Ist die Todoist-nahe Bedien- und Visualsprache erkennbar?
* Bleibt die rote Akzentfarbe sparsam und handlungsbezogen?
* Ist Desktop klar priorisiert?
* Ist die Informationsdichte Todoist-nah kompakt?
* Sind Aufgabenzeilen, Sidebar, Dialoge und Statusmeldungen klar priorisiert?
* Sind laufende Writes, Fehler und Konflikte sichtbar?
* Bleiben Dark Mode, strikte CSP, lokale Assets und serverseitiges Rendering kompatibel?
