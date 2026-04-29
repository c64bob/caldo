CREATE TABLE saved_filters (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    query TEXT NOT NULL,
    is_favorite INTEGER NOT NULL DEFAULT 0,
    server_version INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (is_favorite IN (0, 1))
);

CREATE UNIQUE INDEX idx_saved_filters_name ON saved_filters(name);
CREATE INDEX idx_saved_filters_favorite_name ON saved_filters(is_favorite, name);
