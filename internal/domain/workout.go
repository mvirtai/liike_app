package domain

import "time"

type Workout struct {
	ID              string     `json:"id"`
	UserID          string     `json:"user_id"`
	ExerciseTypeID  string     `json:"exercise_type_id"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time"`
	DurationSeconds *int       `json:"duration_seconds"`
	DistanceKm      *float64   `json:"distance_km"`
	AvgHeartRate    *int       `json:"avg_heart_rate"`
	CaloriesBurned  *int       `json:"calories_burned"`
	Notes           *string    `json:"notes"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`

	ExerciseType  *ExerciseType  `json:"exercise_type,omitempty"`
	Sets          []WorkoutSet   `json:"set,omitempty"`
	ArcheryScores []ArcheryScore `json:"archery_scores,omitempty"`
}

type WorkoutSet struct {
	ID              string    `json:"id"`
	WorkoutID       string    `json:"workout_id"`
	SetNumber       int       `json:"set_number"`
	Reps            *int      `json:"reps"`
	WeightKg        *float64  `json:"weight_kg"`
	DurationSeconds *int      `json:"duration_seconds"`
	CreatedAt       time.Time `json:"created_at"`
}
