CREATE TABLE IF NOT EXISTS quick_ideas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    file_path TEXT NOT NULL CHECK (length(trim(file_path)) > 0),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_quick_ideas_created_at ON quick_ideas(created_at DESC);
