package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"liike_app/internal/domain"

	"github.com/google/uuid"
)

// ErrUserNotFound palautetaan, kun käyttäjää ei löydy tietokannasta.
var ErrUserNotFound = errors.New("käyttäjää ei löydy")

// ErrEmailAlreadyExists palautetaan, kun sähköpostiosoite on jo käytössä.
var ErrEmailAlreadyExists = errors.New("sähköpostiosoite on jo käytössä")

// CreateUser lisää uuden käyttäjän tietokantaan.
// ID generoidaan automaattisesti (UUID v4)
func (r *Repository) CreateUser(ctx context.Context, email, name, passwordHash string) (*domain.User, error) {
	id := uuid.New().String()

	query := `
		INSERT INTO users (id, email, name, password_hash)
		VALUES (?, ?, ?, ?)
	`

	_, err := r.db.ExecContext(ctx, query, id, email, name, passwordHash)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return nil, ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("virhe käyttäjän luonnissa: %w", err)
	}

	return r.GetUserByID(ctx, id)
}

// GetUserByEmail hakee käyttäjän sähköpostiosoitteen perusteella.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM users
		WHERE email = ?
	`

	row := r.db.QueryRowContext(ctx, query, email)
	return scanUser(row)
}

// GetUserByID hakee käyttäjän UUID:n perusteella.
func (r *Repository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM users
		WHERE id = ?
	`
	row := r.db.QueryRowContext(ctx, query, id)
	return scanUser(row)
}

// scanUser lukee *sql.Row:sta domain.User-rakenteen.
func scanUser(row *sql.Row) (*domain.User, error) {
	var u domain.User
	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&u.Name,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("virhe käyttäjän lukemisessa: %w", err)
	}
	return &u, nil
}
