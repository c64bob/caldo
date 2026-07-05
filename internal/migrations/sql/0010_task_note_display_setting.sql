ALTER TABLE settings
ADD COLUMN task_note_display TEXT NOT NULL DEFAULT 'first_two_lines'
CHECK (task_note_display IN ('none', 'full', 'first_line', 'first_two_lines'));
