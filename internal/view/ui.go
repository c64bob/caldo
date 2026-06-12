package view

import "context"

const (
	defaultUILanguage = "de"
	defaultDarkMode   = "system"
)

const uiPreferencesKey contextKey = "ui_preferences"

// UIPreferences contains presentation preferences for one rendered page.
type UIPreferences struct {
	Language string
	DarkMode string
}

// WithUIPreferences stores normalized UI preferences in request context.
func WithUIPreferences(ctx context.Context, language string, darkMode string) context.Context {
	return context.WithValue(ctx, uiPreferencesKey, UIPreferences{
		Language: NormalizeUILanguage(language),
		DarkMode: NormalizeDarkMode(darkMode),
	})
}

// CurrentUIPreferences returns normalized UI preferences from context.
func CurrentUIPreferences(ctx context.Context) UIPreferences {
	preferences, ok := ctx.Value(uiPreferencesKey).(UIPreferences)
	if !ok {
		return UIPreferences{Language: defaultUILanguage, DarkMode: defaultDarkMode}
	}
	return UIPreferences{
		Language: NormalizeUILanguage(preferences.Language),
		DarkMode: NormalizeDarkMode(preferences.DarkMode),
	}
}

// NormalizeUILanguage returns a supported UI language or the default.
func NormalizeUILanguage(language string) string {
	if language == "en" {
		return "en"
	}
	return defaultUILanguage
}

// NormalizeDarkMode returns a supported dark-mode preference or the default.
func NormalizeDarkMode(mode string) string {
	switch mode {
	case "light", "dark", "system":
		return mode
	default:
		return defaultDarkMode
	}
}

// UILanguage returns the current HTML language code.
func UILanguage(ctx context.Context) string {
	return CurrentUIPreferences(ctx).Language
}

// UIDarkMode returns the current dark-mode preference.
func UIDarkMode(ctx context.Context) string {
	return CurrentUIPreferences(ctx).DarkMode
}

// ThemeRootClass returns the CSS class needed to override system preference.
func ThemeRootClass(ctx context.Context) string {
	switch UIDarkMode(ctx) {
	case "light":
		return "light"
	case "dark":
		return "dark"
	default:
		return ""
	}
}

// Texts contains user-facing strings for one UI language.
type Texts struct {
	AppSubtitle              string
	CurrentView              string
	OpenNavigation           string
	MainNavigation           string
	MobileMainNavigation     string
	Close                    string
	SystemFilters            string
	Filters                  string
	AllFilters               string
	NoSavedFilters           string
	Labels                   string
	AllLabels                string
	NoLabels                 string
	Projects                 string
	AllProjects              string
	NoProjects               string
	NewTask                  string
	Search                   string
	Settings                 string
	Favorites                string
	Today                    string
	Upcoming                 string
	Overdue                  string
	NoDate                   string
	Completed                string
	Conflicts                string
	ShortcutHelp             string
	ShortcutNewTask          string
	ShortcutSearch           string
	ShortcutToday            string
	ShortcutUpcoming         string
	ShortcutFavorites        string
	ShortcutProjects         string
	ShortcutLabels           string
	ShortcutFilters          string
	ShortcutConflicts        string
	ShortcutSettings         string
	ShortcutOpenHelp         string
	Or                       string
	Then                     string
	OpenItemsAria            string
	ThemeSystem              string
	ThemeLight               string
	ThemeDark                string
	ThemeSettings            string
	QuickAddTitle            string
	QuickAddTaskLabel        string
	QuickAddPreviewAction    string
	QuickAddPreviewPending   string
	QuickAddPreviewHeading   string
	QuickAddFieldTitle       string
	QuickAddFieldProject     string
	QuickAddFieldLabels      string
	QuickAddFieldDate        string
	QuickAddFieldRecurrence  string
	QuickAddFieldPriority    string
	QuickAddDateResolved     string
	QuickAddDateReviewHint   string
	QuickAddRRuleInvalid     string
	QuickAddCreateProject    string
	QuickAddProjectList      string
	QuickAddLabelSuggestions string
	QuickAddSaveAction       string
	QuickAddSavePending      string
	QuickAddDefaultProject   string
	QuickAddCreateNew        string
	QuickAddNewLabel         string
	QuickAddFound            string
	QuickAddDefault          string
	QuickAddClearValue       string
	None                     string
	PriorityHigh             string
	PriorityMedium           string
	PriorityLow              string
	SettingsPageTitle        string
	SettingsCalDAVHelp       string
	SettingsCalDAVURL        string
	SettingsCalDAVUsername   string
	SettingsCalDAVPassword   string
	SettingsPasswordKeep     string
	SettingsPasswordNew      string
	SettingsPasswordKeepHelp string
	SettingsPasswordNewHelp  string
	SettingsCalDAVTest       string
	SettingsCalDAVSubmit     string
	SettingsCalDAVPending    string
	SettingsCalendarsTitle   string
	SettingsCalendarsHelp    string
	SettingsNoCalendars      string
	SettingsNotMapped        string
	SettingsProjectPrefix    string
	SettingsOpenTasks        string
	SettingsUseAsDefault     string
	SettingsSaveCalendars    string
	SettingsCalendarsPending string
	SettingsLocalOnlyTitle   string
	SettingsTasks            string
	SettingsSyncTitle        string
	SettingsIntervalMinutes  string
	SettingsSaveSync         string
	SettingsSyncPending      string
	SettingsManualSync       string
	SettingsManualPending    string
	SettingsUITitle          string
	SettingsShowCompleted    string
	SettingsUpcomingDays     string
	SettingsLanguage         string
	SettingsDarkMode         string
	SettingsSaveUI           string
	SettingsSecurityTitle    string
	SettingsProxyHeader      string
	SettingsDetected         string
	SettingsNotDetected      string
	SettingsHTTPSStatus      string
	SettingsActive           string
	SettingsInconsistent     string
}

var germanTexts = Texts{
	AppSubtitle:              "CalDAV-Aufgaben",
	CurrentView:              "Aktuelle Ansicht",
	OpenNavigation:           "Navigation öffnen",
	MainNavigation:           "Hauptnavigation",
	MobileMainNavigation:     "Mobile Hauptnavigation",
	Close:                    "Schließen",
	SystemFilters:            "Systemfilter",
	Filters:                  "Filter",
	AllFilters:               "Alle Filter",
	NoSavedFilters:           "Keine gespeicherten Filter",
	Labels:                   "Labels",
	AllLabels:                "Alle Labels",
	NoLabels:                 "Keine Labels",
	Projects:                 "Projekte",
	AllProjects:              "Alle Projekte",
	NoProjects:               "Keine Projekte",
	NewTask:                  "Neue Aufgabe",
	Search:                   "Suche",
	Settings:                 "Einstellungen",
	Favorites:                "Favoriten",
	Today:                    "Heute",
	Upcoming:                 "Demnächst",
	Overdue:                  "Überfällig",
	NoDate:                   "Ohne Datum",
	Completed:                "Abgeschlossen",
	Conflicts:                "Konflikte",
	ShortcutHelp:             "Tastaturkürzel",
	ShortcutNewTask:          "Neue Aufgabe",
	ShortcutSearch:           "Suche",
	ShortcutToday:            "Heute",
	ShortcutUpcoming:         "Demnächst",
	ShortcutFavorites:        "Favoriten",
	ShortcutProjects:         "Projekte",
	ShortcutLabels:           "Labels",
	ShortcutFilters:          "Filter",
	ShortcutConflicts:        "Konflikte",
	ShortcutSettings:         "Einstellungen",
	ShortcutOpenHelp:         "Hilfe öffnen",
	Or:                       "oder",
	Then:                     "dann",
	OpenItemsAria:            "offene Einträge",
	ThemeSystem:              "System",
	ThemeLight:               "Hell",
	ThemeDark:                "Dunkel",
	ThemeSettings:            "Darstellung",
	QuickAddTitle:            "Quick Add",
	QuickAddTaskLabel:        "Aufgabe",
	QuickAddPreviewAction:    "Vorschau",
	QuickAddPreviewPending:   "Vorschau wird erstellt ...",
	QuickAddPreviewHeading:   "Vorschau",
	QuickAddFieldTitle:       "Titel",
	QuickAddFieldProject:     "Projekt",
	QuickAddFieldLabels:      "Labels",
	QuickAddFieldDate:        "Datum",
	QuickAddFieldRecurrence:  "Wiederholung",
	QuickAddFieldPriority:    "Priorität",
	QuickAddDateResolved:     "Erkannt",
	QuickAddDateReviewHint:   "Datum prüfen",
	QuickAddRRuleInvalid:     "Wiederholung prüfen. Verwende ein RRULE wie FREQ=WEEKLY oder entferne den Wert.",
	QuickAddCreateProject:    "als CalDAV-Kalender anlegen",
	QuickAddProjectList:      "Projektvorschläge",
	QuickAddLabelSuggestions: "Labelvorschläge",
	QuickAddSaveAction:       "Speichern",
	QuickAddSavePending:      "Speichern ...",
	QuickAddDefaultProject:   "Default-Projekt",
	QuickAddCreateNew:        "Neu anlegen",
	QuickAddNewLabel:         "Neu",
	QuickAddFound:            "Gefunden",
	QuickAddDefault:          "Default",
	QuickAddClearValue:       "Entfernen",
	None:                     "Keine",
	PriorityHigh:             "Hoch",
	PriorityMedium:           "Mittel",
	PriorityLow:              "Niedrig",
	SettingsPageTitle:        "Einstellungen",
	SettingsCalDAVHelp:       "Verbindung speichern nur nach erfolgreichem Test.",
	SettingsCalDAVURL:        "CalDAV-URL",
	SettingsCalDAVUsername:   "Benutzername",
	SettingsCalDAVPassword:   "Passwort / App-Passwort",
	SettingsPasswordKeep:     "unverändert lassen",
	SettingsPasswordNew:      "Passwort oder App-Passwort",
	SettingsPasswordKeepHelp: "Leer lassen, um das gespeicherte Passwort beizubehalten.",
	SettingsPasswordNewHelp:  "Ein Passwort oder App-Passwort ist für den Verbindungstest erforderlich.",
	SettingsCalDAVTest:       "Verbindung testen",
	SettingsCalDAVSubmit:     "CalDAV speichern",
	SettingsCalDAVPending:    "Verbindung wird getestet ...",
	SettingsCalendarsTitle:   "Kalender & Projektmapping",
	SettingsCalendarsHelp:    "Ausgewählte CalDAV-Kalender werden als Projekte geführt. Bestehende Projekte mit Aufgaben bleiben erhalten.",
	SettingsNoCalendars:      "Keine CalDAV-Kalender geladen. Prüfe die CalDAV-Verbindung.",
	SettingsNotMapped:        "Noch nicht als Projekt hinzugefügt",
	SettingsProjectPrefix:    "Projekt",
	SettingsOpenTasks:        "offene Aufgaben",
	SettingsUseAsDefault:     "Als Default-Projekt verwenden",
	SettingsSaveCalendars:    "Kalenderauswahl speichern",
	SettingsCalendarsPending: "Speichern ...",
	SettingsLocalOnlyTitle:   "Lokale Projekte ohne aktuell geladenen CalDAV-Kalender",
	SettingsTasks:            "Aufgaben",
	SettingsSyncTitle:        "Sync",
	SettingsIntervalMinutes:  "Intervall (Minuten)",
	SettingsSaveSync:         "Sync-Einstellungen speichern",
	SettingsSyncPending:      "Speichern ...",
	SettingsManualSync:       "Jetzt synchronisieren",
	SettingsManualPending:    "Synchronisieren ...",
	SettingsUITitle:          "UI",
	SettingsShowCompleted:    "Erledigte Aufgaben anzeigen",
	SettingsUpcomingDays:     "Demnächst-Zeitraum (Tage)",
	SettingsLanguage:         "Sprache",
	SettingsDarkMode:         "Dark Mode",
	SettingsSaveUI:           "UI-Einstellungen speichern",
	SettingsSecurityTitle:    "Sicherheitsstatus",
	SettingsProxyHeader:      "Reverse-Proxy-Header",
	SettingsDetected:         "erkannt",
	SettingsNotDetected:      "nicht erkannt",
	SettingsHTTPSStatus:      "HTTPS-Status",
	SettingsActive:           "aktiv",
	SettingsInconsistent:     "inkonsistent",
}

var englishTexts = Texts{
	AppSubtitle:              "CalDAV tasks",
	CurrentView:              "Current view",
	OpenNavigation:           "Open navigation",
	MainNavigation:           "Main navigation",
	MobileMainNavigation:     "Mobile main navigation",
	Close:                    "Close",
	SystemFilters:            "System filters",
	Filters:                  "Filters",
	AllFilters:               "All filters",
	NoSavedFilters:           "No saved filters",
	Labels:                   "Labels",
	AllLabels:                "All labels",
	NoLabels:                 "No labels",
	Projects:                 "Projects",
	AllProjects:              "All projects",
	NoProjects:               "No projects",
	NewTask:                  "New task",
	Search:                   "Search",
	Settings:                 "Settings",
	Favorites:                "Favorites",
	Today:                    "Today",
	Upcoming:                 "Upcoming",
	Overdue:                  "Overdue",
	NoDate:                   "No date",
	Completed:                "Completed",
	Conflicts:                "Conflicts",
	ShortcutHelp:             "Keyboard shortcuts",
	ShortcutNewTask:          "New task",
	ShortcutSearch:           "Search",
	ShortcutToday:            "Today",
	ShortcutUpcoming:         "Upcoming",
	ShortcutFavorites:        "Favorites",
	ShortcutProjects:         "Projects",
	ShortcutLabels:           "Labels",
	ShortcutFilters:          "Filters",
	ShortcutConflicts:        "Conflicts",
	ShortcutSettings:         "Settings",
	ShortcutOpenHelp:         "Open help",
	Or:                       "or",
	Then:                     "then",
	OpenItemsAria:            "open items",
	ThemeSystem:              "System",
	ThemeLight:               "Light",
	ThemeDark:                "Dark",
	ThemeSettings:            "Appearance",
	QuickAddTitle:            "Quick Add",
	QuickAddTaskLabel:        "Task",
	QuickAddPreviewAction:    "Preview",
	QuickAddPreviewPending:   "Creating preview ...",
	QuickAddPreviewHeading:   "Preview",
	QuickAddFieldTitle:       "Title",
	QuickAddFieldProject:     "Project",
	QuickAddFieldLabels:      "Labels",
	QuickAddFieldDate:        "Date",
	QuickAddFieldRecurrence:  "Recurrence",
	QuickAddFieldPriority:    "Priority",
	QuickAddDateResolved:     "Recognized",
	QuickAddDateReviewHint:   "Check date",
	QuickAddRRuleInvalid:     "Check recurrence. Use an RRULE like FREQ=WEEKLY or remove the value.",
	QuickAddCreateProject:    "as CalDAV calendar",
	QuickAddProjectList:      "Project suggestions",
	QuickAddLabelSuggestions: "Label suggestions",
	QuickAddSaveAction:       "Save",
	QuickAddSavePending:      "Saving ...",
	QuickAddDefaultProject:   "Default project",
	QuickAddCreateNew:        "Create new",
	QuickAddNewLabel:         "New",
	QuickAddFound:            "Found",
	QuickAddDefault:          "Default",
	QuickAddClearValue:       "Remove",
	None:                     "None",
	PriorityHigh:             "High",
	PriorityMedium:           "Medium",
	PriorityLow:              "Low",
	SettingsPageTitle:        "Settings",
	SettingsCalDAVHelp:       "Connection settings are saved only after a successful test.",
	SettingsCalDAVURL:        "CalDAV URL",
	SettingsCalDAVUsername:   "Username",
	SettingsCalDAVPassword:   "Password / app password",
	SettingsPasswordKeep:     "leave unchanged",
	SettingsPasswordNew:      "Password or app password",
	SettingsPasswordKeepHelp: "Leave empty to keep the stored password.",
	SettingsPasswordNewHelp:  "A password or app password is required for the connection test.",
	SettingsCalDAVTest:       "Test connection",
	SettingsCalDAVSubmit:     "Save CalDAV",
	SettingsCalDAVPending:    "Testing connection ...",
	SettingsCalendarsTitle:   "Calendars & project mapping",
	SettingsCalendarsHelp:    "Selected CalDAV calendars are managed as projects. Existing projects with tasks are preserved.",
	SettingsNoCalendars:      "No CalDAV calendars loaded. Check the CalDAV connection.",
	SettingsNotMapped:        "Not added as a project yet",
	SettingsProjectPrefix:    "Project",
	SettingsOpenTasks:        "open tasks",
	SettingsUseAsDefault:     "Use as default project",
	SettingsSaveCalendars:    "Save calendar selection",
	SettingsCalendarsPending: "Saving ...",
	SettingsLocalOnlyTitle:   "Local projects without a currently loaded CalDAV calendar",
	SettingsTasks:            "tasks",
	SettingsSyncTitle:        "Sync",
	SettingsIntervalMinutes:  "Interval (minutes)",
	SettingsSaveSync:         "Save sync settings",
	SettingsSyncPending:      "Saving ...",
	SettingsManualSync:       "Sync now",
	SettingsManualPending:    "Syncing ...",
	SettingsUITitle:          "UI",
	SettingsShowCompleted:    "Show completed tasks",
	SettingsUpcomingDays:     "Upcoming range (days)",
	SettingsLanguage:         "Language",
	SettingsDarkMode:         "Dark mode",
	SettingsSaveUI:           "Save UI settings",
	SettingsSecurityTitle:    "Security status",
	SettingsProxyHeader:      "Reverse proxy header",
	SettingsDetected:         "detected",
	SettingsNotDetected:      "not detected",
	SettingsHTTPSStatus:      "HTTPS status",
	SettingsActive:           "active",
	SettingsInconsistent:     "inconsistent",
}

// Text returns the text catalog for the current UI language.
func Text(ctx context.Context) Texts {
	if UILanguage(ctx) == "en" {
		return englishTexts
	}
	return germanTexts
}

// DisplayTitle returns a localized title for known application pages.
func DisplayTitle(ctx context.Context, title string) string {
	if UILanguage(ctx) != "en" {
		return title
	}
	switch title {
	case "Heute":
		return englishTexts.Today
	case "Demnächst":
		return englishTexts.Upcoming
	case "Überfällig":
		return englishTexts.Overdue
	case "Favoriten":
		return englishTexts.Favorites
	case "Ohne Datum":
		return englishTexts.NoDate
	case "Erledigt", "Erledigte Aufgaben", "Abgeschlossen":
		return englishTexts.Completed
	case "Suche":
		return englishTexts.Search
	case "Projekte":
		return englishTexts.Projects
	case "Labels":
		return englishTexts.Labels
	case "Filter":
		return englishTexts.Filters
	case "Konflikte", "Konfliktdetail":
		return englishTexts.Conflicts
	case "Einstellungen":
		return englishTexts.Settings
	default:
		return title
	}
}

// ThemeModeLabel returns the localized label for a dark-mode preference.
func ThemeModeLabel(ctx context.Context, mode string) string {
	text := Text(ctx)
	switch NormalizeDarkMode(mode) {
	case "light":
		return text.ThemeLight
	case "dark":
		return text.ThemeDark
	default:
		return text.ThemeSystem
	}
}
