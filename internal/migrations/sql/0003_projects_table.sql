CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    calendar_href TEXT NOT NULL,
    display_name TEXT NOT NULL,
    ctag TEXT,
    sync_token TEXT,
    sync_strategy TEXT NOT NULL DEFAULT 'fullscan' CHECK (sync_strategy IN ('webdav_sync', 'ctag', 'fullscan')),
    server_version INTEGER NOT NULL DEFAULT 1,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
