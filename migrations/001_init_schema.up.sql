-- 1. Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- 2. Exercise Types table
CREATE TABLE IF NOT EXISTS exercise_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    category TEXT NOT NULL, -- 'cardio', 'strength', 'archery', 'flexibility'
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed default exercise types  with explicit UUIDs
INSERT OR IGNORE INTO exercise_types (id, name, category, description) VALUES
    ('10000000-0000-4000-8000-000000000001', 'Kävely', 'cardio', 'Kävelylenkki matkalla, kestolla ja sykkeellä'),
    ('10000000-0000-4000-8000-000000000002', 'Juoksu', 'cardio', 'Juoksulenkki matkalla, kestolla ja sykkeellä'),
    ('10000000-0000-4000-8000-000000000003', 'Jousiammunta', 'archery', 'Jousiammunnan sarja- ja nuolikohtainen tuloskirjaus'),
    ('10000000-0000-4000-8000-000000000004', 'Kyykky', 'strength', 'Jalkatreeni toistoilla ja painoilla'),
    ('10000000-0000-4000-8000-000000000005', 'Vatsalihaslankku', 'strength', 'Keskivartalon pito sekunneissa'),
    ('10000000-0000-4000-8000-000000000006', 'Painoharjoittelu', 'strength', 'Yleinen kuntosalitreeni sarjoilla ja painoilla'),
    ('10000000-0000-4000-8000-000000000007', 'Jooga', 'flexibility', 'Kehonhuolto ja joogaharjoitus muistiinpanoilla');

-- 3. Workouts table 
CREATE TABLE IF NOT EXISTS workouts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    exercise_type_id TEXT NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME,
    duration_seconds INTEGER,
    distance_km REAL,
    avg_heart_rate INTEGER,
    calories_burned INTEGER,
    notes TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (exercise_type_id) REFERENCES exercise_types(id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_workouts_user_id ON workouts(user_id);
CREATE INDEX IF NOT EXISTS idx_workouts_start_time ON workouts(start_time);

-- 4. Workout Sets table 
CREATE TABLE  IF NOT EXISTS workout_sets (
    id TEXT PRIMARY KEY,
    workout_id TEXT NOT NULL,
    set_number INTEGER NOT NULL,
    reps INTEGER,
    weight_kg REAL,
    duration_seconds INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE,
    UNIQUE(workout_id, set_number)
);

-- 5. Archery Scores table
CREATE TABLE IF NOT EXISTS archery_scores (
    id TEXT PRIMARY KEY,
    workout_id TEXT NOT NULL,
    end_number INTEGER NOT NULL,
    arrow_number INTEGER NOT NULL,
    score_value INTEGER NOT NULL CHECK(score_value >= 0 AND score_value <= 10),
    is_x BOOLEAN NOT NULL DEFAULT 0 CHECK(is_x IN (0, 1)),
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE,
    UNIQUE(workout_id, end_number, arrow_number)
);

CREATE INDEX IF NOT EXISTS i;dx_archery_scores_workout_id ON archery_scores (workout_id)