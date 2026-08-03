-- Testimigraatio 002: exercise_types-taulu
CREATE TABLE IF NOT EXISTS exercise_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    category TEXT NOT NULL,
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
