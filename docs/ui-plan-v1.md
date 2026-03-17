## Phase 1 — Strukturelles Layout & Navigation (Fundament)

**Ziel:** Das UI-Skelett sieht aus wie Toodledo — Sidebar, Hauptbereich, Header.

**Layout:**
- Zweispaltig: Sidebar 220px fest links, Hauptbereich flexibel rechts
- Header: App-Logo, globale Suchbox, Account-Icon — keine überladene Toolbar
- Sidebar-Sektionen: Hotlist, Main, Folders, Contexts, Goals, Priority-View, Due-Date-View, Tags — exakt Toodledo-Reihenfolge [toodledo](https://www.toodledo.com/products.php)
- Aktiver View wird in Sidebar hervorgehoben
- Sidebar collapsible via Button (nicht per Keyboard in Phase 1)
- Responsive: Sidebar blendet auf schmalen Screens aus

**Task-Tabelle Grundstruktur:**
- Spalten konfigurierbar, Default-Set: ☆ Star | Checkbox | Task-Name | Folder | Context | Due Date | Priority
- Spaltenbreiten: Name nimmt verfügbaren Restplatz, restliche Spalten kompakt/fix
- Zebra-Striping in gedämpftem Grau, hover-Highlight
- Priority-Farbe als linker Rand-Indikator (1px farbiger Balken): Top=Rot, High=Orange, Medium=Blau, Low=Schwarz, Negative=Grau

**Erreichtes nach Phase 1:**
Caldo sieht strukturell wie Toodledo aus. Alle Views sind navigierbar. Noch kein Inline-Editing, noch keine Shortcuts.

***

## Phase 2 — Task-Zeile: Felder, Inline-Editing, Smart Add

**Ziel:** Jede Task-Zeile ist vollständig bedienbar ohne Seitennavigation.

**Task-Zeile Felder:**
- `SUMMARY` — editierbar inline per Klick, Enter speichert, Escape bricht ab
- `PRIORITY` — Dropdown (Top / High / Medium / Low / Negative) mit Farbvorschau
- `DUE` / `DTSTART` — Datepicker mit Natural Language: `today`, `tomorrow`, `mon`, `1 week`, `next month` → wird zu ISO-Datum aufgelöst
- `STATUS` — Dropdown: None / Next Action / Active / Planning / Delegated / Waiting / Hold / Someday / Cancelled / Reference
- `CATEGORIES` (Tags) — Chip-Input: Tippen, Enter fügt hinzu, × entfernt
- `PERCENT-COMPLETE` — Schieberegler 0–100% oder Direkteingabe
- ☆ Star — Toggle per Klick, HTMX-Partial Update
- Checkbox — Task als erledigt markieren, `STATUS=COMPLETED` in CalDAV

**Expand-Zeile (Detail-Panel):**
Klick auf Expand-Pfeil öffnet unter der Task-Zeile ein inline Panel (kein Modal) mit:
- `DESCRIPTION` / Notes — Textarea mit Markdown-Preview-Toggle
- `VALARM` — Reminder-Zeit konfigurieren
- Subtasks-Liste (Phase 4)
- Metadaten: UID, letztes Sync-Datum

**Smart Add (Toodledo-Syntax):**
Quick-Add-Eingabefeld oben in der Tabelle — Parsing-Regeln: [toodledo](https://www.toodledo.com/features.php)
```
"Arzt /folder:Privat /context:@Telefon /due:friday !high #Gesundheit"
```
Slash-Prefixes für Folder/Context, `!`-Prefix für Priority, `#`-Prefix für Tag, natural language für Datum.

**Erreichtes nach Phase 2:**
Vollständige Task-CRUD-Operationen im UI ohne Mausklick auf Menüpunkte. Smart Add ermöglicht schnelle Eingabe.

***

## Phase 3 — Keyboard Shortcuts

**Ziel:** Alle häufigen Aktionen ohne Maus erreichbar. Toodledo-kompatible Shortcuts als Basis.

**Navigation (View-Wechsel):** [toodledo](https://www.toodledo.com/forums/3/1874/0/keyboard-shortcuts.html)

| Key | Aktion |
|---|---|
| `m` | Main View |
| `o` | Folder View |
| `c` | Context View |
| `d` | Due-Date View |
| `g` | Goal View |
| `p` | Priority View |
| `h` | Hotlist |
| `e` | Search View |
| `1`–`9` | Tabs innerhalb des aktiven Views |

**Aktionen:**

| Key | Aktion |
|---|---|
| `n` | Fokus auf Quick-Add-Feld (neue Task) |
| `f` | Fokus auf globale Suchbox |
| `s` | Alle offenen Inline-Edits speichern |
| `Escape` | Aktiven Inline-Edit abbrechen |
| `Enter` | Inline-Edit bestätigen |
| `Tab` | Nächstes Feld in Inline-Edit |
| `Shift+Tab` | Vorheriges Feld |
| `↑` / `↓` | Zwischen Tasks navigieren |
| `Space` | Ausgewählte Task als erledigt markieren |
| `x` | Ausgewählte Task für Batch-Edit markieren |
| `Del` | Ausgewählte Task löschen (mit Confirm) |
| `?` | Keyboard-Shortcut-Overlay anzeigen (Gmail-Stil) |

**Implementation-Detail:** Shortcuts via `document.addEventListener('keydown')` in einem kleinen Vanilla-JS-Modul (~80 Zeilen). Deaktivierung wenn Fokus in Inputfeld — `event.target.tagName !== 'INPUT'` check. HTMX-kompatibel, kein Framework nötig.

**Erreichtes nach Phase 3:**
Kompletter Workflow ohne Maus möglich. `?`-Overlay als eingebaute Hilfe.

***

## Phase 4 — Subtasks, Folders, Contexts, Goals

**Ziel:** Die GTD-Hierarchie von Toodledo vollständig abbilden.

**Subtasks:**
- Expansion via Expand-Pfeil in Zeile, Subtasks werden eingerückt darunter angezeigt
- Subtask hinzufügen via Button oder `Tab`-Shortcut in der Parent-Zeile
- Subtasks werden via `RELATED-TO: PARENT` in CalDAV persistiert
- Drag-and-drop Reordering (via Sortable.js, ~5kB) [toodledo](https://www.toodledo.com/forums/5/4965/0/use-subtasks-for-projects-already-defined-tasks-subtasks.html)
- Parent-Task zeigt Fortschrittsanzeige: `(3/5)` abgeschlossene Subtasks

**Folders:**
- Sidebar-Sektion listet alle CalDAV-Collections als Folder
- Folder-Farbe konfigurierbar (gespeichert in SQLite preferences)
- Tasks per Drag-and-drop in anderen Folder verschieben
- "Alle Tasks"-View zeigt folder-übergreifend

**Contexts & Goals:**
- Contexts: frei definierbare @-Präfix-Labels (z.B. `@Home`, `@Work`, `@Telefon`)
- Goals: hierarchische Struktur (Lifetime → Long-term → Short-term), gespeichert in SQLite da CalDAV kein Goal-Konzept kennt — via `X-CALDO-GOAL` Property in VTODO
- Goal-Chain: abgeschlossene Tasks pro Goal werden gezählt und angezeigt [toodledo](https://www.toodledo.com/features.php)

**Erreichtes nach Phase 4:**
Vollständige GTD-Struktur. Folders/Contexts/Goals funktionieren wie in Toodledo.

***

## Phase 5 — Hotlist, Filter, Saved Searches

**Ziel:** Die zwei mächtigsten Toodledo-Features — Hotlist-Algorithmus und Saved Searches.

**Hotlist-Algorithmus:** [toodledo](https://www.toodledo.com/features.php)
Die Hotlist zeigt Tasks die Toodledo intern als "am wichtigsten jetzt" bewertet. Implementierung als server-seitige Score-Funktion:

```go
func hotlistScore(t Task) float64 {
    score := 0.0
    if t.Star { score += 3.0 }
    // Priority: Top=4, High=3, Medium=2, Low=1, Negative=0
    score += float64(priorityWeight(t.Priority))
    // Fälligkeit: überfällig=5, heute=4, morgen=3, diese Woche=2
    score += dueDateWeight(t.Due)
    // Status: Next Action = Bonus
    if t.Status == "Next Action" { score += 1.0 }
    return score
}
```

Hotlist-Konfiguration: User kann Schwellwerte in Preferences anpassen — welcher Priority-Level, wie viele Tage im Voraus, ob Status eingerechnet wird.

**Filter-Panel:**
Ausklappbar über der Tabelle. Filter-Kriterien kombinierbar: [toodledo](https://www.toodledo.com/features.php)
- Priority (Mehrfachauswahl)
- Due Date Range
- Folder / Context / Goal
- Status (Mehrfachauswahl)
- Tags
- Star ja/nein
- Text-Suche in Summary + Description

**Saved Searches / Saved Filters:**
- Filter-Kombinationen speichern → erscheint als eigener Eintrag in Sidebar
- Gespeichert in SQLite (`saved_filters` Tabelle)
- Direkt-URL für jeden gespeicherten Filter: `/filter/[slug]`

**Erreichtes nach Phase 5:**
Caldo ist in der täglichen Nutzung mit Toodledo gleichwertig. Hotlist ist das wichtigste Feature für GTD-Power-User.

***

## Phase 6 — Sortierung, Batch-Edit, Import/Export

**Ziel:** Power-User-Features für große Task-Bestände.

**Multi-Level-Sort:**
Bis zu 3 Sortierkriterien kombinierbar — Toodledo-Kernfeature. Klick auf Spaltenheader sortiert primär, `Shift+Klick` fügt sekundäres Kriterium hinzu. Gespeichert als User-Präferenz. [toodledo](https://www.toodledo.com/features.php)

**Batch-Edit:** [apps.apple](https://apps.apple.com/ca/app/toodledo-next-generation-app/id1615853505)
- Tasks markieren via `x`-Shortcut oder Checkbox in erster Spalte
- Batch-Toolbar erscheint oben: Priority setzen, Folder verschieben, Status ändern, Tags hinzufügen, löschen
- Selektion: `Shift+Klick` für Range, `Ctrl+A` für alle

**Import / Export:**
- Export: alle Tasks als iCalendar-Datei (VTODO) — direkt aus CalDAV, kein Konvertierungsschritt
- Import: iCalendar-Datei hochladen → Tasks in ausgewählte Collection schreiben
- CSV-Export für Spreadsheet-Nutzung (konvertiert VTODO-Felder)

**Erreichtes nach Phase 6:**
Vollständiger Datenimport aus Toodledo oder anderen Apps. Batch-Operationen machen große Umstrukturierungen praktikabel.

***

## Phase 7 — Visuelle Politur & UX-Details

**Ziel:** Die letzten ~20% die den Unterschied zwischen "funktioniert" und "fühlt sich an wie Toodledo" machen.

**Visuelle Details:**
- Überfällige Tasks: Datum in Rot, leichter Rot-Hintergrund in der Zeile
- Heute fällige Tasks: Datum in Orange
- Erledigte Tasks: durchgestrichen, ausgegraut, in separate "Completed"-Sektion
- Priority-Farb-Schema pixelgenau zu Toodledo: Top=`#d9534f`, High=`#f0ad4e`, Medium=`#5bc0de`, Low=`#5cb85c`, Negative=`#999`
- Inline-Edit-Felder: erscheinen exakt an der Position des geklickten Texts, kein Layout-Shift

**Micro-Interactions:**
- Star-Toggle: kurze CSS-Transition (scale 1→1.3→1, 150ms)
- Task erledigt: Checkbox-Animation + Strike-through slide
- Neue Task via Smart Add: erscheint mit fade-in in der richtigen Position
- Sync-Indicator: kleines Spinner-Icon im Header während CalDAV-Request

**Accessibility:**
- `aria-label` auf allen Icons
- Keyboard-Focus-Ring sichtbar (nicht via `outline: none` unterdrückt)
- Screen-Reader-freundliche Tabellen-Struktur

**Dark Mode:**
- CSS-Variables für alle Farben, `prefers-color-scheme: dark` Media Query
- Toodledo hat kein Dark Mode — das ist ein direktes Upgrade

**Erreichtes nach Phase 7:**
Caldo ist visuell poliert und in der täglichen Nutzung nicht von Toodledo unterscheidbar — mit Dark Mode als echtem Mehrwert.

