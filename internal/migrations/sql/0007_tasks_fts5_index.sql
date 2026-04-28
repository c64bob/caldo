CREATE VIRTUAL TABLE tasks_fts USING fts5(
    title,
    description,
    label_names,
    project_name,
    content=tasks,
    content_rowid=rowid,
    tokenize='unicode61 remove_diacritics 1'
);

CREATE TRIGGER tasks_ai AFTER INSERT ON tasks BEGIN
    INSERT INTO tasks_fts(rowid, title, description, label_names, project_name)
    VALUES (new.rowid, new.title, new.description, new.label_names, new.project_name);
END;

CREATE TRIGGER tasks_ad AFTER DELETE ON tasks BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, title, description, label_names, project_name)
    VALUES ('delete', old.rowid, old.title, old.description, old.label_names, old.project_name);
END;

CREATE TRIGGER tasks_au AFTER UPDATE ON tasks BEGIN
    INSERT INTO tasks_fts(tasks_fts, rowid, title, description, label_names, project_name)
    VALUES ('delete', old.rowid, old.title, old.description, old.label_names, old.project_name);
    INSERT INTO tasks_fts(rowid, title, description, label_names, project_name)
    VALUES (new.rowid, new.title, new.description, new.label_names, new.project_name);
END;

INSERT INTO tasks_fts(tasks_fts) VALUES ('rebuild');
