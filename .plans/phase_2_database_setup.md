# Vaihe 2: Tietokanta-arkkitehtuuri, Migraatiot ja Käyttäjä/Laji-skeemat (UUID ID:t) – Yksityiskohtainen suunnitelma

**Sovellus:** Liike App  
**Moduuli:** Backend (Go + SQLite)  
**Dokumentti:** `.plans/phase_2_database_setup.md`  

---

## 📌 1. Tavoite ja Yhteenveto

Vaiheessa 2 rakennetaan **Liike App** -sovelluksen tietokanta-arkkitehtuuri ja automaattinen migraatiojärjestelmä. Tietokantana toimii **SQLite** (Pure Go ajuri `modernc.org/sqlite`).

**Ehdoton vaatimus:** Kaikki ID-kentät ja viiteavaimet (Foreign Keys) ovat ehdottomasti **UUID-muotoisia (TEXT / Go string UUID)**.

Asetamme tietokannalle asianmukaiset suorituskyky- ja eheysasetukset (`PRAGMA foreign_keys = ON;`, `PRAGMA journal_mode = WAL;`) ja rakennamme automaattisesti käynnistyksen yhteydessä suoritettavan schema-migraattorin Go 1.16+ `embed.FS` -mekanismilla.

Lisäksi määrittelemme tietokantaskeemat ja Go domain-entiteetit seuraaville entiteeteille UUID-tunnisteilla:

1. `users` – Käyttäjätilit (`id TEXT PRIMARY KEY`) ja autentikaatiotiedot.
2. `exercise_types` – Lajit & harjoitusmuodot (`id TEXT PRIMARY KEY`) (Kävely, Juoksu, Jousiammunta, Kyykky, Vatsalihaslankku, Jooga, jne.) sekä oletusdata kiinteillä UUID-arvoilla.
3. `workouts` – Suoritukset (`id TEXT PRIMARY KEY`, `user_id TEXT`, `exercise_type_id TEXT`).
4. `workout_sets` – Kuntosali-/toisto-/kestosarjat (`id TEXT PRIMARY KEY`, `workout_id TEXT`).
5. `archery_scores` – Jousiammunnan erikoistuloskirjaus (`id TEXT PRIMARY KEY`, `workout_id TEXT`).

---

## 🛠️ 2. Esivaatimukset

- Go 1.22+ ja asennetut riippuvuudet (`modernc.org/sqlite`, `github.com/google/uuid`).
- Vaiheen 1 perusrakenne toiminnassa (`cmd/server`, `internal/config`, `internal/handler`).

---

## 📂 3. Kohde-hakemistorakenne

Vaiheen 2 päätteeksi projektin tiedostorakenne laajenee seuraavasti:

```
liike_app/
├── .plans/
│   ├── action_plan.md
│   ├── phase_1_project_setup.md
│   └── phase_2_database_setup.md        # [NEW] Suunnitelmadokumentti UUID-määrittelyillä
├── cmd/
│   └── server/
│       └── main.go                         # [MODIFY] DB-alustus ja migraatioiden suoritus
├── internal/
│   ├── config/
│   │   └── config.go                       # [MODIFY] Tietokantapolkujen tarkistus
│   ├── database/
│   │   ├── database.go                     # [NEW] SQLite-yhteyspooli & PRAGMA-asetukset
│   │   ├── database_test.go                # [NEW] Integratio- & migraatiotestit (:memory:)
│   │   └── migrator.go                     # [NEW] Embed.FS SQL-migraattori
│   ├── domain/
│   │   ├── archery.go                      # [NEW] ArcheryScore entiteetti (UUID)
│   │   ├── exercise_type.go                # [NEW] ExerciseType entiteetti (UUID)
│   │   ├── user.go                         # [NEW] User entiteetti (UUID)
│   │   └── workout.go                      # [NEW] Workout & WorkoutSet entiteetit (UUID)
│   ├── handler/
│   │   └── health.go
│   ├── repository/
│   │   └── db.go                           # [NEW] DB-rajapinnan alustus
│   └── service/
├── migrations/
│   ├── 000001_init_schema.up.sql           # [NEW] Taulujen luonti (UUID) & seed-data
│   └── 000001_init_schema.down.sql         # [NEW] Taulujen poisto DDL
├── Taskfile.yml
├── go.mod
└── go.sum
```

---

## 📋 4. Yksityiskohtainen toteutusohje koodaajalle (Step-by-Step)

### Vaihe 2.1: Migraatiotiedostot (`migrations/`)

Luo hakemisto `migrations/` ja seuraavat tiedostot:

#### A. `migrations/000001_init_schema.up.sql`

```sql
-- 1. Users table (id on UUID TEXT)
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- 2. Exercise Types table (id on UUID TEXT)
CREATE TABLE IF NOT EXISTS exercise_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    category TEXT NOT NULL, -- 'cardio', 'strength', 'archery', 'flexibility'
    description TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed default exercise types with explicit UUIDs
INSERT OR IGNORE INTO exercise_types (id, name, category, description) VALUES
    ('10000000-0000-4000-8000-000000000001', 'Kävely', 'cardio', 'Kävelylenkki matkalla, kestolla ja sykkeellä'),
    ('10000000-0000-4000-8000-000000000002', 'Juoksu', 'cardio', 'Juoksulenkki matkalla, kestolla ja sykkeellä'),
    ('10000000-0000-4000-8000-000000000003', 'Jousiammunta', 'archery', 'Jousiammunnan sarja- ja nuolikohtainen tuloskirjaus'),
    ('10000000-0000-4000-8000-000000000004', 'Kyykky', 'strength', 'Jalkatreeni toistoilla ja painoilla'),
    ('10000000-0000-4000-8000-000000000005', 'Vatsalihaslankku', 'strength', 'Keskivartalon pito sekunneissa'),
    ('10000000-0000-4000-8000-000000000006', 'Painoharjoittelu', 'strength', 'Yleinen kuntosalitreeni sarjoilla ja painoilla'),
    ('10000000-0000-4000-8000-000000000007', 'Jooga', 'flexibility', 'Kehonhuolto ja joogaharjoitus muistiinpanoilla');

-- 3. Workouts table (id, user_id, exercise_type_id ovat UUID TEXT)
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

-- 4. Workout Sets table (id, workout_id ovat UUID TEXT)
CREATE TABLE IF NOT EXISTS workout_sets (
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

-- 5. Archery Scores table (id, workout_id ovat UUID TEXT)
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

CREATE INDEX IF NOT EXISTS idx_archery_scores_workout_id ON archery_scores(workout_id);
```

#### B. `migrations/000001_init_schema.down.sql`

```sql
DROP TABLE IF EXISTS archery_scores;
DROP TABLE IF EXISTS workout_sets;
DROP TABLE IF EXISTS workouts;
DROP TABLE IF EXISTS exercise_types;
DROP TABLE IF EXISTS users;
```

---

### Vaihe 2.2: Tietokantamoduuli (`internal/database/database.go` ja `migrator.go`)

#### A. `internal/database/database.go`

```go
package database

import (
 "database/sql"
 "fmt"

 _ "modernc.org/sqlite"
)

// New avaa SQLite-tietokantayhteyden ja asettaa suorituskyky-PRAGMAt
func New(dbPath string) (*sql.DB, error) {
 db, err := sql.Open("sqlite", dbPath)
 if err != nil {
  return nil, fmt.Errorf("virhe tietokannan avaamisessa: %w", err)
 }

 pragmas := []string{
  "PRAGMA foreign_keys = ON;",
  "PRAGMA journal_mode = WAL;",
  "PRAGMA busy_timeout = 5000;",
  "PRAGMA synchronous = NORMAL;",
 }

 for _, p := range pragmas {
  if _, err := db.Exec(p); err != nil {
   db.Close()
   return nil, fmt.Errorf("virhe PRAGMA-asetuksessa (%s): %w", p, err)
  }
 }

 if err := db.Ping(); err != nil {
  db.Close()
  return nil, fmt.Errorf("virhe tietokantayhteydessä (ping): %w", err)
 }

 return db, nil
}
```

#### B. `internal/database/migrator.go`

```go
package database

import (
 "database/sql"
 "embed"
 "fmt"
 "sort"
 "strings"
)

// RunMigrations ajaa kaikki annetut .up.sql migraatiot järjestyksessä
func RunMigrations(db *sql.DB, fs embed.FS, dir string) error {
 createTableQuery := `
 CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
 );`
 if _, err := db.Exec(createTableQuery); err != nil {
  return fmt.Errorf("virhe schema_migrations-taulun luonnissa: %w", err)
 }

 entries, err := fs.ReadDir(dir)
 if err != nil {
  return fmt.Errorf("virhe migraatiohakemiston luvussa: %w", err)
 }

 var upFiles []string
 for _, entry := range entries {
  if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".up.sql") {
   upFiles = append(upFiles, entry.Name())
  }
 }
 sort.Strings(upFiles)

 for _, filename := range upFiles {
  var version int
  _, err := fmt.Sscanf(filename, "%d_", &version)
  if err != nil {
   return fmt.Errorf("virheellinen migraatiotiedoston nimi %s: %w", filename, err)
  }

  var count int
  err = db.QueryRow("SELECT COUNT(1) FROM schema_migrations WHERE version = ?", version).Scan(&count)
  if err != nil {
   return fmt.Errorf("virhe migraatioversion tarkistuksessa: %w", err)
  }

  if count > 0 {
   continue // Migraatio jo suoritettu
  }

  content, err := fs.ReadFile(dir + "/" + filename)
  if err != nil {
   return fmt.Errorf("virhe migraatiotiedoston %s luvussa: %w", filename, err)
  }

  tx, err := db.Begin()
  if err != nil {
   return fmt.Errorf("virhe transaktion aloituksessa (%s): %w", filename, err)
  }

  if _, err := tx.Exec(string(content)); err != nil {
   _ = tx.Rollback()
   return fmt.Errorf("virhe migraation %s suorituksessa: %w", filename, err)
  }

  if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES (?)", version); err != nil {
   _ = tx.Rollback()
   return fmt.Errorf("virhe migraatioversion tallennuksessa: %w", err)
  }

  if err := tx.Commit(); err != nil {
   return fmt.Errorf("virhe transaktion vahvistuksessa: %w", err)
  }
 }

 return nil
}
```

---

### Vaihe 2.3: Domain Entities (`internal/domain/`) UUID-muodossa

#### A. `internal/domain/user.go`

```go
package domain

import "time"

type User struct {
 ID           string    `json:"id"` // UUID v4
 Email        string    `json:"email"`
 PasswordHash string    `json:"-"`
 Name         string    `json:"name"`
 CreatedAt    time.Time `json:"created_at"`
 UpdatedAt    time.Time `json:"updated_at"`
}
```

#### B. `internal/domain/exercise_type.go`

```go
package domain

import "time"

type ExerciseCategory string

const (
 CategoryCardio      ExerciseCategory = "cardio"
 CategoryStrength    ExerciseCategory = "strength"
 CategoryArchery     ExerciseCategory = "archery"
 CategoryFlexibility ExerciseCategory = "flexibility"
)

type ExerciseType struct {
 ID          string           `json:"id"` // UUID v4
 Name        string           `json:"name"`
 Category    ExerciseCategory `json:"category"`
 Description string           `json:"description,omitempty"`
 CreatedAt   time.Time        `json:"created_at"`
}
```

#### C. `internal/domain/workout.go`

```go
package domain

import "time"

type Workout struct {
 ID              string        `json:"id"`               // UUID v4
 UserID          string        `json:"user_id"`          // UUID v4
 ExerciseTypeID  string        `json:"exercise_type_id"` // UUID v4
 StartTime       time.Time     `json:"start_time"`
 EndTime         *time.Time    `json:"end_time,omitempty"`
 DurationSeconds *int          `json:"duration_seconds,omitempty"`
 DistanceKM      *float64      `json:"distance_km,omitempty"`
 AvgHeartRate    *int          `json:"avg_heart_rate,omitempty"`
 CaloriesBurned  *int          `json:"calories_burned,omitempty"`
 Notes           *string       `json:"notes,omitempty"`
 CreatedAt       time.Time     `json:"created_at"`
 UpdatedAt       time.Time     `json:"updated_at"`

 ExerciseType    *ExerciseType  `json:"exercise_type,omitempty"`
 Sets            []WorkoutSet   `json:"sets,omitempty"`
 ArcheryScores   []ArcheryScore `json:"archery_scores,omitempty"`
}

type WorkoutSet struct {
 ID              string    `json:"id"`         // UUID v4
 WorkoutID       string    `json:"workout_id"` // UUID v4
 SetNumber       int       `json:"set_number"`
 Reps            *int      `json:"reps,omitempty"`
 WeightKG        *float64  `json:"weight_kg,omitempty"`
 DurationSeconds *int      `json:"duration_seconds,omitempty"`
 CreatedAt       time.Time `json:"created_at"`
}
```

#### D. `internal/domain/archery.go`

```go
package domain

import "time"

type ArcheryScore struct {
 ID          string    `json:"id"`         // UUID v4
 WorkoutID   string    `json:"workout_id"` // UUID v4
 EndNumber   int       `json:"end_number"`
 ArrowNumber int       `json:"arrow_number"`
 ScoreValue  int       `json:"score_value"`
 IsX         bool      `json:"is_x"`
 CreatedAt   time.Time `json:"created_at"`
}
```

---

### Vaihe 2.4: Repository Base (`internal/repository/db.go`)

```go
package repository

import (
 "database/sql"
)

type Repository struct {
 db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
 return &Repository{db: db}
}
```

---

### Vaihe 2.5: Pääprosessin päivitys (`cmd/server/main.go`)

Päivitetään main.go alustamaan tietokantayhteys ja ajamaan migraatiot `embed.FS`:n avulla:

```go
package main

import (
 "context"
 "embed"
 "errors"
 "fmt"
 "log"
 "net/http"
 "os"
 "os/signal"
 "syscall"
 "time"

 "liike_app/internal/config"
 "liike_app/internal/database"
 "liike_app/internal/handler"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

func main() {
 cfg := config.Load()

 // 1. Tietokannan alustus & PRAGMAt
 db, err := database.New(cfg.DatabaseURL)
 if err != nil {
  log.Fatalf("[DB] Kriittinen virhe tietokantayhteydessä: %v", err)
 }
 defer db.Close()
 log.Printf("[DB] Tietokantayhteys avattu: %s", cfg.DatabaseURL)

 // 2. Ajetaan migraatiot
 if err := database.RunMigrations(db, migrationFS, "migrations"); err != nil {
  log.Fatalf("[DB] Virhe migraatioiden suorituksessa: %v", err)
 }
 log.Println("[DB] Tietokantamigraatiot ajettu onnistuneesti.")

 // 3. HTTP Server
 mux := http.NewServeMux()
 mux.HandleFunc("GET /api/v1/health", handler.HealthCheckHandler)

 server := &http.Server{
  Addr:         fmt.Sprintf(":%s", cfg.Port),
  Handler:      mux,
  ReadTimeout:  10 * time.Second,
  WriteTimeout: 10 * time.Second,
  IdleTimeout:  60 * time.Second,
 }

 go func() {
  log.Printf("[SERVER] Liike App backend käynnissä osoitteessa http://localhost%s", server.Addr)
  if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
   log.Fatalf("[SERVER] Kriittinen virhe palvelinta käynnistettäessä: %v", err)
  }
 }()

 stop := make(chan os.Signal, 1)
 signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
 <-stop

 log.Println("[SERVER] Sammutussignaali vastaanotettu. Suljetaan palvelin...")
 ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
 defer cancel()

 if err := server.Shutdown(ctx); err != nil {
  log.Fatalf("[SERVER] Virhe palvelimen hallitussa sammutuksessa: %v", err)
 }

 log.Println("[SERVER] Palvelin ja tietokantayhteydet suljettu onnistuneesti.")
}
```

---

## 🧪 5. Verifiointi ja Testaus

### 1. Automaattiset yksikkö- ja migraatiotestit (`internal/database/database_test.go`)

Aja testit komennolla:

```bash
task test
```

### 2. Palvelimen ajonaikainen testaus

Aja kehityspalvelin:

```bash
task dev
```

*Varmista, että lokiin tulostuu:*
`[DB] Tietokantayhteys avattu: liike.db`  
`[DB] Tietokantamigraatiot ajettu onnistuneesti.`

---

## ✅ 6. Hyväksyntäkriteerit (Acceptance Criteria)

- [ ] Tiedostot `migrations/000001_init_schema.up.sql` ja `down.sql` on luotu UUID TEXT primary key -määrittelyillä.
- [ ] SQLite PRAGMA-asetukset (foreign keys, WAL mode) ja yhteyspooli toimivat virheettömästi.
- [ ] Embed.FS migraattori ajaa migraatiot ja tallentaa tilan `schema_migrations`-tauluun.
- [ ] Oletuslajit (`exercise_types`) tallentuvat kantaan kiinteillä UUID-tunnisteilla.
- [ ] Domain-entiteetit (`User`, `ExerciseType`, `Workout`, `WorkoutSet`, `ArcheryScore`) on määritelty UUID `string`-kentillä.
- [ ] `task check` ja `task test` menevät läpi ilman virheitä.
