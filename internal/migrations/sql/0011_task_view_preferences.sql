CREATE TABLE task_view_preferences (
    view_kind TEXT NOT NULL,
    view_id TEXT NOT NULL DEFAULT '',
    sort_by TEXT NOT NULL DEFAULT 'default'
        CHECK (sort_by IN ('default', 'due', 'priority', 'name', 'added')),
    sort_order TEXT NOT NULL DEFAULT 'asc'
        CHECK (sort_order IN ('asc', 'desc')),
    group_by TEXT NOT NULL DEFAULT 'none'
        CHECK (group_by IN ('none', 'project', 'due', 'added', 'priority')),
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (view_kind, view_id),
    CHECK (view_kind IN ('today', 'upcoming', 'overdue', 'favorites', 'no_date', 'completed', 'project', 'label', 'filter', 'search'))
);
