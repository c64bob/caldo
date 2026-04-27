CREATE TABLE undo_snapshots (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    tab_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    snapshot_vtodo TEXT NOT NULL,
    snapshot_fields TEXT NOT NULL,
    etag_at_snapshot TEXT,
    created_at DATETIME NOT NULL,
    expires_at DATETIME NOT NULL,
    UNIQUE(session_id, tab_id)
);

CREATE INDEX idx_undo_session_tab ON undo_snapshots(session_id, tab_id);
CREATE INDEX idx_undo_expires ON undo_snapshots(expires_at);

CREATE TABLE conflicts (
    id TEXT PRIMARY KEY,
    task_id TEXT,
    project_id TEXT REFERENCES projects(id),
    conflict_type TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    resolved_at DATETIME,
    resolution TEXT,
    base_vtodo TEXT,
    local_vtodo TEXT,
    remote_vtodo TEXT,
    resolved_vtodo TEXT
);
