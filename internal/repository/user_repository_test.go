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
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX idx_users_email ON users(email);
	`)
	if err != nil {
		t.Fatalf("users-taulun luonti epäonnistui: %v", err)
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
