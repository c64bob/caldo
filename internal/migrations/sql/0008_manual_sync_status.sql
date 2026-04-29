ALTER TABLE settings ADD COLUMN sync_state TEXT NOT NULL DEFAULT 'idle' CHECK (sync_state IN ('idle','running','error'));
ALTER TABLE settings ADD COLUMN sync_last_started_at DATETIME;
ALTER TABLE settings ADD COLUMN sync_last_finished_at DATETIME;
ALTER TABLE settings ADD COLUMN sync_last_success_at DATETIME;
ALTER TABLE settings ADD COLUMN sync_last_error_code TEXT;
