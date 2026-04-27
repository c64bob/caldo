DROP TABLE IF EXISTS settings;

CREATE TABLE settings (
    id TEXT PRIMARY KEY DEFAULT 'default' CHECK (id = 'default'),
    setup_complete BOOLEAN NOT NULL DEFAULT FALSE,
    setup_step TEXT NOT NULL DEFAULT 'caldav' CHECK (setup_step IN ('caldav', 'calendars', 'import', 'complete')),
    caldav_url TEXT,
    caldav_username TEXT,
    caldav_password_enc TEXT,
    caldav_server_capabilities TEXT,
    sync_interval_minutes INTEGER NOT NULL DEFAULT 15,
    default_project_id TEXT REFERENCES projects(id) ON DELETE SET NULL,
    ui_language TEXT NOT NULL DEFAULT 'de',
    dark_mode TEXT NOT NULL DEFAULT 'system' CHECK (dark_mode IN ('light', 'dark', 'system')),
    upcoming_days INTEGER NOT NULL DEFAULT 7,
    show_completed BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at DATETIME NOT NULL
);

INSERT OR IGNORE INTO settings (id, updated_at)
VALUES ('default', CURRENT_TIMESTAMP);
