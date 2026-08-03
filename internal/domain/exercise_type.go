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
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Category    ExerciseCategory `json:"category"`
	Description string           `json:"description,omtempty"`
	CreatedAt   time.Time        `json:"created_at"`
}
