package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"liike_app/internal/repository"

	_ "modernc.org/sqlite"

	"github.com/google/uuid"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("testitietokannan avaaminen epäonnistui: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("testitietokannan sulkeminen epäonnistui: %v", err)
		}
	})

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

		CREATE TABLE IF NOT EXISTS exercise_types (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			category TEXT NOT NULL,
			description TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		INSERT OR IGNORE INTO exercise_types (id, name, category, description) VALUES
			('10000000-0000-4000-8000-000000000001', 'Kävely', 'cardio', 'Kävelylenkki matkalla, kestolla ja sykkeellä'),
			('10000000-0000-4000-8000-000000000002', 'Juoksu', 'cardio', 'Juoksulenkki matkalla, kestolla ja sykkeellä'),
			('10000000-0000-4000-8000-000000000003', 'Jousiammunta', 'archery', 'Jousiammunnan sarja- ja nuolikohtainen tuloskirjaus'),
			('10000000-0000-4000-8000-000000000004', 'Kyykky', 'strength', 'Jalkatreeni toistoilla ja painoilla'),
			('10000000-0000-4000-8000-000000000005', 'Vatsalihaslankku', 'strength', 'Keskivartalon pito sekunneissa'),
			('10000000-0000-4000-8000-000000000006', 'Painoharjoittelu', 'strength', 'Yleinen kuntosalitreeni sarjoilla ja painoilla'),
			('10000000-0000-4000-8000-000000000007', 'Jooga', 'flexibility', 'Kehonhuolto ja joogaharjoitus muistiinpanoilla');

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
	`)
	if err != nil {
		t.Fatalf("kannan alustus epäonnistui: %v", err)
	}

	return db
}

func newTestRepository(t *testing.T) *repository.Repository {
	t.Helper()

	db := setupTestDB(t)

	return repository.NewRepository(db)
}

// --- CreateUser ---

func TestCreateUser_Success(t *testing.T) {
	repo := newTestRepository(t)

	user, err := repo.CreateUser(
		context.Background(),
		"test@example.com",
		"Testi Käyttäjä",
		"hashed-password",
	)
	if err != nil {
		t.Fatalf("CreateUser epäonnistui: %v", err)
	}

	if user == nil {
		t.Fatal("CreateUser palautti nil-käyttäjän")
	}

	if user.ID == "" {
		t.Error("käyttäjän ID ei saisi olla tyhjä")
	}

	if _, err := uuid.Parse(user.ID); err != nil {
		t.Errorf("käyttäjän ID ei ole validi UUID: %v", err)
	}

	if user.Email != "test@example.com" {
		t.Errorf(
			"odotettu email 'test@example.com', saatiin '%s'",
			user.Email,
		)
	}

	if user.Name != "Testi Käyttäjä" {
		t.Errorf(
			"odotettu name 'Testi Käyttäjä', saatiin '%s'",
			user.Name,
		)
	}

	if user.PasswordHash != "hashed-password" {
		t.Errorf("PasswordHash ei vastaa annettua arvoa")
	}

	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt ei saisi olla zero value")
	}

	if user.UpdatedAt.IsZero() {
		t.Error("UpdatedAt ei saisi olla zero value")
	}
}

func TestCreateUser_EmailAlreadyExists(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	_, err := repo.CreateUser(
		ctx,
		"duplicate@example.com",
		"Ensimmäinen",
		"hash1",
	)
	if err != nil {
		t.Fatalf("ensimmäisen käyttäjän luonti epäonnistui: %v", err)
	}

	_, err = repo.CreateUser(
		ctx,
		"duplicate@example.com",
		"Toinen",
		"hash2",
	)

	if !errors.Is(err, repository.ErrEmailAlreadyExists) {
		t.Errorf(
			"odotettiin ErrEmailAlreadyExists, saatiin %v",
			err,
		)
	}
}

// --- GetUserByEmail ---

func TestGetUserByEmail_Success(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	created, err := repo.CreateUser(
		ctx,
		"email@example.com",
		"Testi Käyttäjä",
		"hashed-password",
	)
	if err != nil {
		t.Fatalf("CreateUser epäonnistui: %v", err)
	}

	user, err := repo.GetUserByEmail(ctx, "email@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail epäonnistui: %v", err)
	}

	if user == nil {
		t.Fatal("GetUserByEmail palautti nil")
	}

	if user.ID != created.ID {
		t.Errorf(
			"odotettu ID '%s', saatiin '%s'",
			created.ID,
			user.ID,
		)
	}

	if user.Email != "email@example.com" {
		t.Errorf(
			"odotettu email 'email@example.com', saatiin '%s'",
			user.Email,
		)
	}

	if user.Name != "Testi Käyttäjä" {
		t.Errorf(
			"odotettu name 'Testi Käyttäjä', saatiin '%s'",
			user.Name,
		)
	}

	if user.PasswordHash != "hashed-password" {
		t.Error("PasswordHash ei vastaa odotettua arvoa")
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	repo := newTestRepository(t)

	user, err := repo.GetUserByEmail(
		context.Background(),
		"missing@example.com",
	)

	if user != nil {
		t.Error("käyttäjän pitäisi olla nil, kun sitä ei löydy")
	}

	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf(
			"odotettiin ErrUserNotFound, saatiin %v",
			err,
		)
	}
}

// --- GetUserByID ---

func TestGetUserByID_Success(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	created, err := repo.CreateUser(
		ctx,
		"id@example.com",
		"ID Test",
		"hash",
	)
	if err != nil {
		t.Fatalf("CreateUser epäonnistui: %v", err)
	}

	user, err := repo.GetUserByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetUserByID epäonnistui: %v", err)
	}

	if user == nil {
		t.Fatal("GetUserByID palautti nil")
	}

	if user.ID != created.ID {
		t.Errorf(
			"odotettu ID '%s', saatiin '%s'",
			created.ID,
			user.ID,
		)
	}

	if user.Email != "id@example.com" {
		t.Errorf(
			"odotettu email 'id@example.com', saatiin '%s'",
			user.Email,
		)
	}

	if user.Name != "ID Test" {
		t.Errorf(
			"odotettu name 'ID Test', saatiin '%s'",
			user.Name,
		)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	repo := newTestRepository(t)

	user, err := repo.GetUserByID(
		context.Background(),
		"550e8400-e29b-41d4-a716-446655440000",
	)

	if user != nil {
		t.Error("käyttäjän pitäisi olla nil, kun sitä ei löydy")
	}

	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf(
			"odotettiin ErrUserNotFound, saatiin %v",
			err,
		)
	}
}
