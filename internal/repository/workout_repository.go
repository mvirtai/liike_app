package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"liike_app/internal/domain"

	"github.com/google/uuid"
)

var ErrWorkoutNotFound = errors.New("suoritusta ei löydy")

type WorkoutFilter struct {
	ExerciseTypeID string
	FromDate       *time.Time
	ToDate         *time.Time
	Limit          int
	Offset         int
}

// CreateWorkout luo suorituksen ja mahdolliset sarjat / ammunnat transaktiossa.
func (r *Repository) CreateWorkout(ctx context.Context, w *domain.Workout, sets []domain.WorkoutSet, scores []domain.ArcheryScore) (*domain.Workout, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("virhe transaktion aloituksessa: %w", err)
	}
	defer tx.Rollback()

	if w.ID == "" {
		w.ID = uuid.New().String()
	}
	now := time.Now()
	w.CreatedAt = now
	w.UpdatedAt = now

	insertWorkout := `
		INSERT INTO workouts (id, user_id, exercise_type_id, start_time, end_time, duration_seconds, distance_km, avg_heart_rate, calories_burned, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = tx.ExecContext(ctx, insertWorkout,
		w.ID, w.UserID, w.ExerciseTypeID, w.StartTime, w.EndTime,
		w.DurationSeconds, w.DistanceKm, w.AvgHeartRate, w.CaloriesBurned,
		w.Notes, w.CreatedAt, w.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("virhe suorituksen tallennuksessa: %w", err)
	}

	// Lisätään sarjat jos annettu
	for i := range sets {
		s := &sets[i]
		if s.ID == "" {
			s.ID = uuid.New().String()
		}
		s.WorkoutID = w.ID
		s.CreatedAt = now

		insertSet := `
			INSERT INTO workout_sets (id, workout_id, set_number, reps, weight_kg, duration_seconds, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`

		_, err := tx.ExecContext(ctx, insertSet, s.ID, s.WorkoutID, s.SetNumber, s.Reps, s.WeightKg, s.DurationSeconds, s.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("virhe sarjan tallennuksessa: %w", err)
		}
	}

	// Lisätään jousiammunta-ammunnat jos annettu
	for i := range scores {
		sc := &scores[i]
		if sc.ID == "" {
			sc.ID = uuid.New().String()
		}
		sc.WorkoutID = w.ID
		sc.CreatedAt = now

		insertScore := `
			INSERT INTO archery_scores (id, workout_id, end_number, arrow_number, score_value, is_x, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`

		_, err := tx.ExecContext(ctx, insertScore, sc.ID, sc.WorkoutID, sc.EndNumber, sc.ArrowNumber, sc.ScoreValue, sc.IsX, sc.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("virhe jousiammunta-ammunnan tallennuksessa: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("virhe transaktion vahvistuksessa: %w", err)
	}

	return r.GetWorkoutByID(ctx, w.ID, w.UserID)
}

// GetWorkoutByID hakee suorituksen kaikkine relaatioineen (varmistaa user_id omistajuuden).
func (r *Repository) GetWorkoutByID(ctx context.Context, workoutID, userID string) (*domain.Workout, error) {
	queryWorkout := `
  SELECT w.id, w.user_id, w.exercise_type_id, w.start_time, w.end_time,
         w.duration_seconds, w.distance_km, w.avg_heart_rate, w.calories_burned,
         w.notes, w.created_at, w.updated_at,
         et.name, et.category, et.description
  FROM workouts w
  JOIN exercise_types et ON w.exercise_type_id = et.id
  WHERE w.id = ? AND w.user_id = ?
 `

	var w domain.Workout
	var et domain.ExerciseType
	var etDesc sql.NullString

	err := r.db.QueryRowContext(ctx, queryWorkout, workoutID, userID).Scan(
		&w.ID, &w.UserID, &w.ExerciseTypeID, &w.StartTime, &w.EndTime,
		&w.DurationSeconds, &w.DistanceKm, &w.AvgHeartRate, &w.CaloriesBurned,
		&w.Notes, &w.CreatedAt, &w.UpdatedAt,
		&et.Name, &et.Category, &etDesc,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWorkoutNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("virhe suorituksen haussa: %w", err)
	}

	et.ID = w.ExerciseTypeID
	if etDesc.Valid {
		et.Description = etDesc.String
	}
	w.ExerciseType = &et

	// Haetaan sarjat
	querySets := `
		SELECT id, workout_id, set_number, reps, weight_kg, duration_seconds, created_at
		FROM workout_sets
		WHERE workout_id = ?
		ORDER BY set_number ASC
	`

	rowsSets, err := r.db.QueryContext(ctx, querySets, workoutID)
	if err != nil {
		return nil, fmt.Errorf("virhe sarjojen haussa: %w", err)
	}
	defer rowsSets.Close()

	for rowsSets.Next() {
		var s domain.WorkoutSet
		if err := rowsSets.Scan(&s.ID, &s.WorkoutID, &s.SetNumber, &s.Reps, &s.WeightKg, &s.DurationSeconds, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("virhe sarjarivin lukemisessa: %w", err)
		}
		w.Sets = append(w.Sets, s)
	}
	if err := rowsSets.Err(); err != nil {
		return nil, fmt.Errorf("virhe sarjojen läpikäynnissä: %w", err)
	}

	// Haetaan jousiammuntatulokset
	queryScores := `
		SELECT id, workout_id, end_number, arrow_number, score_value, is_x, created_at
		FROM archery_scores
		WHERE workout_id = ?
		ORDER BY end_number ASC, arrow_number ASC
	`

	rowsScores, err := r.db.QueryContext(ctx, queryScores, workoutID)
	if err != nil {
		return nil, fmt.Errorf("virhe ammuntojen haussa: %w", err)
	}
	defer rowsScores.Close()

	for rowsScores.Next() {
		var sc domain.ArcheryScore
		if err := rowsScores.Scan(&sc.ID, &sc.WorkoutID, &sc.EndNumber, &sc.ArrowNumber, &sc.ScoreValue, &sc.IsX, &sc.CreatedAt); err != nil {
			return nil, fmt.Errorf("virhe ammuntojen lukemisessa: %w", err)
		}
		w.ArcheryScores = append(w.ArcheryScores, sc)
	}
	if err := rowsScores.Err(); err != nil {
		return nil, fmt.Errorf("virhe ammuntojen läpikäynnissä: %w", err)
	}

	return &w, nil
}

// ListWorkoutsByUserID hakee käyttäjän suoritukset suodattimien mukaan.
func (r *Repository) ListWorkoutsByUserID(ctx context.Context, userID string, filter WorkoutFilter) ([]domain.Workout, error) {
	query := `
		SELECT w.id, w.user_id, w.exercise_type_id, w.start_time, w.end_time,
			   w.duration_seconds, w.distance_km, w.avg_heart_rate, w.calories_burned,
			   w.notes, w.created_at, w.updated_at,
			   et.name, et.category, et.description
		FROM workouts w
		JOIN exercise_types et ON w.exercise_type_id = et.id
		WHERE w.user_id = ?
	`
	args := []any{userID}

	if filter.ExerciseTypeID != "" {
		query += ` AND w.exercise_type_id = ?`
		args = append(args, filter.ExerciseTypeID)
	}

	if filter.FromDate != nil {
		query += ` AND w.start_time >= ?`
		args = append(args, *filter.FromDate)
	}

	if filter.ToDate != nil {
		query += ` AND w.start_time <= ?`
		args = append(args, *filter.ToDate)
	}

	query += ` ORDER BY w.start_time DESC`

	limit := 20
	if filter.Limit > 0 {
		limit = filter.Limit
	}
	query += ` LIMIT ? OFFSET ?`
	args = append(args, limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("virhe suoritusten listauksessa: %w", err)
	}
	defer rows.Close()

	var result []domain.Workout
	for rows.Next() {
		var w domain.Workout
		var et domain.ExerciseType
		var etDesc sql.NullString

		err := rows.Scan(
			&w.ID, &w.UserID, &w.ExerciseTypeID, &w.StartTime, &w.EndTime,
			&w.DurationSeconds, &w.DistanceKm, &w.AvgHeartRate, &w.CaloriesBurned,
			&w.Notes, &w.CreatedAt, &w.UpdatedAt,
			&et.Name, &et.Category, &etDesc,
		)
		if err != nil {
			return nil, fmt.Errorf("virhe rivin scanauksessa: %w", err)
		}
		et.ID = w.ExerciseTypeID
		if etDesc.Valid {
			et.Description = etDesc.String
		}
		w.ExerciseType = &et
		result = append(result, w)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("virhe rivien läpikäynnissä: %w", err)
	}

	return result, nil
}

// UpdateWorkout päivittää suorituksen perustiedot.
func (r *Repository) UpdateWorkout(ctx context.Context, w *domain.Workout) error {
	query := `
		UPDATE workouts
		SET start_time = ?, end_time = ?, duration_seconds = ?, distance_km = ?,
			avg_heart_rate = ?, calories_burned = ?, notes = ?, updated_at = ?
		WHERE id = ? AND user_id = ?
	`
	res, err := r.db.ExecContext(ctx, query,
		w.StartTime, w.EndTime, w.DurationSeconds, w.DistanceKm,
		w.AvgHeartRate, w.CaloriesBurned, w.Notes, w.UpdatedAt,
		w.ID, w.UserID,
	)
	if err != nil {
		return fmt.Errorf("virhe suorituksen päivityksessä: %w", err)
	}
	affected, _ := res.RowsAffected()
	fmt.Print(affected)
	if affected == 0 {
		return ErrWorkoutNotFound
	}
	return nil
}

// DeleteWorkout poistaa suorituksen (SQLite CASCADE poistaa sarjat ja ammunnat).
func (r *Repository) DeleteWorkout(ctx context.Context, workoutID, userID string) error {
	query := `DELETE FROM workouts WHERE id = ? AND user_id = ?`
	res, err := r.db.ExecContext(ctx, query, workoutID, userID)
	if err != nil {
		return fmt.Errorf("virhe suorituksen poistossa: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrWorkoutNotFound
	}
	return nil
}

// AddWorkoutSets lisää sarjoja olemassa olevaan suoritukseen.
func (r *Repository) AddWorkoutSets(ctx context.Context, workoutID string, sets []domain.WorkoutSet) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	for i := range sets {
		s := &sets[i]
		if s.ID == "" {
			s.ID = uuid.New().String()
		}
		s.WorkoutID = workoutID
		s.CreatedAt = now

		insertSet := `
			INSERT INTO workout_sets (id, workout_id, set_number, reps, weight_kg, duration_seconds, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`

		_, err := tx.ExecContext(ctx, insertSet, s.ID, s.WorkoutID, s.SetNumber, s.Reps, s.WeightKg, s.DurationSeconds, s.CreatedAt)
		if err != nil {
			return err
		}

	}

	return tx.Commit()
}

// AddArcheryScores lisää nuolituloksia olemassa olevaan jousiammuntasuoritukseen.
func (r *Repository) AddArcheryScores(ctx context.Context, workoutID string, scores []domain.ArcheryScore) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now()
	for i := range scores {
		sc := &scores[i]
		if sc.ID == "" {
			sc.ID = uuid.New().String()
		}
		sc.WorkoutID = workoutID
		sc.CreatedAt = now

		insertScore := `
		  INSERT INTO archery_scores (id, workout_id, end_number, arrow_number, score_value, is_x, created_at)
		  VALUES (?, ?, ?, ?, ?, ?, ?)
		`
		_, err := tx.ExecContext(ctx, insertScore, sc.ID, sc.WorkoutID, sc.EndNumber, sc.ArrowNumber, sc.ScoreValue, sc.IsX, sc.CreatedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
