package domain

import "time"

type ArcheryScore struct {
	ID          string    `json:"id"`
	WorkoutID   string    `json:"workout_id"`
	EndNumber   int       `json:"end_number"`
	ArrowNumber int       `json:"arrow_number"`
	ScoreValue  int       `json:"score_value"`
	IsX         bool      `json:"is_x"`
	CreatedAt   time.Time `json:"created_at"`
}
