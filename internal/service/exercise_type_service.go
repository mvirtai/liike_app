package service

import (
	"context"

	"liike_app/internal/domain"
	"liike_app/internal/repository"
)

type ExerciseTypeService struct {
	repo *repository.Repository
}

func NewExerciseTypeService(repo *repository.Repository) *ExerciseTypeService {
	return &ExerciseTypeService{repo: repo}
}

func (s *ExerciseTypeService) GetAllExerciseTypes(ctx context.Context) ([]domain.ExerciseType, error) {
	return s.repo.GetAllExerciseTypes(ctx)
}

func (s *ExerciseTypeService) GetExerciseTypeByID(ctx context.Context, id string) (*domain.ExerciseType, error) {
	return s.repo.GetExerciseTypeByID(ctx, id)
}
