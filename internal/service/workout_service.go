package service

import (
	"context"
	"errors"
	"time"

	"liike_app/internal/domain"
	"liike_app/internal/repository"
)

var (
	ErrInvalidStartTime  = errors.New("aloitusaika on pakollinen")
	ErrInvalidEndTime    = errors.New("lopetusaika ei voi olla ennen aloitusaikaa")
	ErrInvalidScoreValue = errors.New("jousiammunnan pistearvon pitää olla 0 ja 10 välillä")
	ErrInvalidXScore     = errors.New("vain 10 pisteen osuma voi olla X")
	ErrWorkoutAccess     = errors.New("ei oikeutta suoritukseen")
)

type CreateWorkoutInput struct {
	ExerciseTypeID  string                `json:"exercise_type_id"`
	StartTime       time.Time             `json:"start_time"`
	EndTime         *time.Time            `json:"end_time,omitempty"`
	DurationSeconds *int                  `json:"duration_seconds,omitempty"`
	DistanceKm      *float64              `json:"distance_km,omitempty"`
	AvgHeartRate    *int                  `json:"avg_heart_rate,omitempty"`
	CaloriesBurned  *int                  `json:"calories_burned,omitempty"`
	Notes           *string               `json:"notes,omitempty"`
	Sets            []domain.WorkoutSet   `json:"sets,omitempty"`
	ArcheryScores   []domain.ArcheryScore `json:"archery_scores,omitempty"`
}

// ArcherySummary tiivistää jousiammuntasuorituksen ammuntojen kokonaistuloksen.
type ArcherySummary struct {
	TotalScore   int     `json:"total_score"`
	TotalArrows  int     `json:"total_arrows"`
	TotalXCount  int     `json:"total_x_count"`
	Total10Count int     `json:"total_10_count"`
	AverageArrow float64 `json:"average_arrow"`
}

type WorkoutDetailResponse struct {
	Workout        *domain.Workout `json:"workout"`
	ArcherySummary *ArcherySummary `json:"archery_summary,omitempty"`
}

type WorkoutService struct {
	repo *repository.Repository
}

func NewWorkoutService(repo *repository.Repository) *WorkoutService {
	return &WorkoutService{repo: repo}
}

// CreateWorkout validoi syötteen ja luo uuden suorituksen.
func (s *WorkoutService) CreateWorkout(ctx context.Context, userID string, input CreateWorkoutInput) (*WorkoutDetailResponse, error) {
	if input.StartTime.IsZero() {
		return nil, ErrInvalidStartTime
	}
	if input.EndTime != nil && input.EndTime.Before(input.StartTime) {
		return nil, ErrInvalidEndTime
	}

	// Validoi jousiammunta-ammunnat
	for _, sc := range input.ArcheryScores {
		if sc.ScoreValue < 0 || sc.ScoreValue > 10 {
			return nil, ErrInvalidScoreValue
		}
		if sc.IsX && sc.ScoreValue != 10 {
			return nil, ErrInvalidXScore
		}
	}

	// Automaattinen kestolaskenta jos start & end annettu
	duration := input.DurationSeconds
	if duration == nil && input.EndTime != nil {
		secs := int(input.EndTime.Sub(input.StartTime).Seconds())
		duration = &secs
	}

	w := &domain.Workout{
		UserID:          userID,
		ExerciseTypeID:  input.ExerciseTypeID,
		StartTime:       input.StartTime,
		EndTime:         input.EndTime,
		DurationSeconds: duration,
		DistanceKm:      input.DistanceKm,
		AvgHeartRate:    input.AvgHeartRate,
		CaloriesBurned:  input.CaloriesBurned,
		Notes:           input.Notes,
	}

	created, err := s.repo.CreateWorkout(ctx, w, input.Sets, input.ArcheryScores)
	if err != nil {
		return nil, err
	}

	return s.buildWorkoutResponse(created), nil
}

// GetWorkoutByID hakee suorituksen ja laskee mahdolliset jousiammuntasummat.
func (s *WorkoutService) GetWorkoutByID(ctx context.Context, workoutID, userID string) (*WorkoutDetailResponse, error) {
	w, err := s.repo.GetWorkoutByID(ctx, workoutID, userID)
	if err != nil {
		return nil, err
	}
	return s.buildWorkoutResponse(w), nil
}

func (s *WorkoutService) ListWorkouts(ctx context.Context, userID string, filter repository.WorkoutFilter) ([]domain.Workout, error) {
	return s.repo.ListWorkoutsByUserID(ctx, userID, filter)
}

func (s *WorkoutService) DeleteWorkout(ctx context.Context, workoutID, userID string) error {
	return s.repo.DeleteWorkout(ctx, workoutID, userID)
}

// AddArcheryScores lisää ja validoi uusia nuolituloksia suoritukseen.
func (s *WorkoutService) AddArcheryScores(ctx context.Context, workoutID, userID string, scores []domain.ArcheryScore) (*WorkoutDetailResponse, error) {
	// Varmistetaan omistajuus
	_, err := s.repo.GetWorkoutByID(ctx, workoutID, userID)
	if err != nil {
		return nil, err
	}

	for _, sc := range scores {
		if sc.ScoreValue < 0 || sc.ScoreValue > 10 {
			return nil, ErrInvalidScoreValue
		}
		if sc.IsX && sc.ScoreValue != 10 {
			return nil, ErrInvalidXScore
		}
	}

	if err := s.repo.AddArcheryScores(ctx, workoutID, scores); err != nil {
		return nil, err
	}

	return s.GetWorkoutByID(ctx, workoutID, userID)
}

// buildWorkoutResponse laskee jousiammunnan yhteenvedon jos suoritus sisältää nuolituloksia.
func (s *WorkoutService) buildWorkoutResponse(w *domain.Workout) *WorkoutDetailResponse {
	resp := &WorkoutDetailResponse{Workout: w}

	if len(w.ArcheryScores) > 0 {
		sum := &ArcherySummary{}
		for _, sc := range w.ArcheryScores {
			sum.TotalScore += sc.ScoreValue
			sum.TotalArrows++
			if sc.ScoreValue == 10 {
				sum.Total10Count++
			}
			if sc.IsX {
				sum.TotalXCount++
			}
		}
		if sum.TotalArrows > 0 {
			sum.AverageArrow = float64(sum.TotalScore) / float64(sum.TotalArrows)
		}
		resp.ArcherySummary = sum
	}

	return resp
}