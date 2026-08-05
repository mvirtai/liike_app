package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"liike_app/internal/domain"
)

var ErrExerciseTypeNotFound = errors.New("harjoitusmuotoa ei löydy")

// GetAllExerciseTypes hakee kaikki järjestelmän lajit nimen mukaan järjestettynä.
func (r *Repository) GetAllExerciseTypes(ctx context.Context) ([]domain.ExerciseType, error) {
	query := `
		SELECT id, name, category, description, created_at
		FROM exercise_types
		ORDER BY name ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("virhe harjoitusmuotojen haussa: %w", err)
	}
	defer rows.Close()

	var types []domain.ExerciseType
	for rows.Next() {
		var et domain.ExerciseType
		var desc sql.NullString
		if err := rows.Scan(&et.ID, &et.Name, &et.Category, &desc, &et.CreatedAt); err != nil {
			return nil, fmt.Errorf("virhe rivien lukemisessa: %w", err)
		}
		if desc.Valid {
			et.Description = desc.String
		}
		types = append(types, et)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("virhe rivien läpikäynnissä: %w", err)
	}

	return types, nil
}

// GetExerciseTypeByID hakee tietyn lajin ID:n perusteella.
func (r *Repository) GetExerciseTypeByID(ctx context.Context, id string) (*domain.ExerciseType, error) {
	query := `
		SELECT id, name, category, description, created_at
		FROM exercise_types
		WHERE id = ?
	`

	var et domain.ExerciseType
	var desc sql.NullString
	err := r.db.QueryRowContext(ctx, query, id).Scan(&et.ID, &et.Name, &et.Category, &desc, &et.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrExerciseTypeNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("virhe harjoitusmuodon haussa ID:llä: %w", err)
	}

	if desc.Valid {
		et.Description = desc.String
	}

	return &et, nil
}
