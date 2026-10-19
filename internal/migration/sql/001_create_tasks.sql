CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT tasks_title_not_blank CHECK (char_length(btrim(title)) > 0),
    CONSTRAINT tasks_title_max_length CHECK (char_length(title) <= 120)
);
