package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"liike_app/internal/domain"
	"liike_app/internal/repository"
)

func TestWorkoutService_CreateAndGetWorkout(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	svc := NewWorkoutService(repo)
	ctx := context.Background()

	// Luodaan käyttäjä
	user, err := repo.CreateUser(ctx, "workout_svc@example.com", "Svc User", "pass1234")
	if err != nil {
		t.Fatalf("CreateUser epäonnistui: %v", err)
	}

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now()

	// 1. Onnistunut luonti sarjoilla ja jousiammunnalla
	input := CreateWorkoutInput{
		ExerciseTypeID: "10000000-0000-4000-8000-000000000003", // Jousiammunta
		StartTime:      start,
		EndTime:        &end,
		Notes:          func(s string) *string { return &s }("Testiammunta"),
		ArcheryScores: []domain.ArcheryScore{
			{EndNumber: 1, ArrowNumber: 1, ScoreValue: 10, IsX: true},
			{EndNumber: 1, ArrowNumber: 2, ScoreValue: 10, IsX: false},
			{EndNumber: 1, ArrowNumber: 3, ScoreValue: 8, IsX: false},
		},
	}

	res, err := svc.CreateWorkout(ctx, user.ID, input)
	if err != nil {
		t.Fatalf("CreateWorkout epäonnistui: %v", err)
	}

	if res == nil || res.Workout == nil {
		t.Fatal("CreateWorkout palautti nil-vastauksen")
	}

	if res.ArcherySummary == nil {
		t.Fatal("ArcherySummary pitäisi olla mukana")
	}
	if res.ArcherySummary.TotalScore != 28 {
		t.Errorf("odotettu TotalScore 28, saatiin %d", res.ArcherySummary.TotalScore)
	}
	if res.ArcherySummary.TotalXCount != 1 {
		t.Errorf("odotettu TotalXCount 1, saatiin %d", res.ArcherySummary.TotalXCount)
	}
	if res.ArcherySummary.Total10Count != 2 {
		t.Errorf("odotettu Total10Count 2, saatiin %d", res.ArcherySummary.Total10Count)
	}
	if res.Workout.DurationSeconds == nil || *res.Workout.DurationSeconds < 3500 {
		t.Errorf("automaattisen keston pitäisi olla ~3600s, saatiin %v", res.Workout.DurationSeconds)
	}

	// 2. Hae ID:llä
	fetched, err := svc.GetWorkoutByID(ctx, res.Workout.ID, user.ID)
	if err != nil {
		t.Fatalf("GetWorkoutByID epäonnistui: %v", err)
	}
	if fetched.Workout.ID != res.Workout.ID {
		t.Errorf("odotettu ID %s, saatiin %s", res.Workout.ID, fetched.Workout.ID)
	}

	// 3. Listaus suodattimella
	list, err := svc.ListWorkouts(ctx, user.ID, repository.WorkoutFilter{})
	if err != nil {
		t.Fatalf("ListWorkouts epäonnistui: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("odotettiin 1 suoritus listauksessa, saatiin %d", len(list))
	}

	// 4. Lisää nuolituloksia
	newScores := []domain.ArcheryScore{
		{EndNumber: 2, ArrowNumber: 1, ScoreValue: 10, IsX: true},
	}
	updated, err := svc.AddArcheryScores(ctx, res.Workout.ID, user.ID, newScores)
	if err != nil {
		t.Fatalf("AddArcheryScores epäonnistui: %v", err)
	}
	if updated.ArcherySummary.TotalScore != 38 {
		t.Errorf("odotettu TotalScore 38 päivityksen jälkeen, saatiin %d", updated.ArcherySummary.TotalScore)
	}

	// 5. Poisto
	if err := svc.DeleteWorkout(ctx, res.Workout.ID, user.ID); err != nil {
		t.Fatalf("DeleteWorkout epäonnistui: %v", err)
	}

	// Varmistetaan virhe poistetun haussa
	_, err = svc.GetWorkoutByID(ctx, res.Workout.ID, user.ID)
	if err == nil {
		t.Error("odotettiin virhettä poistetun suorituksen haussa")
	}
}

func TestWorkoutService_Validations(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	svc := NewWorkoutService(repo)
	ctx := context.Background()

	// 1. Tyhjä aloitusaika
	_, err := svc.CreateWorkout(ctx, "u1", CreateWorkoutInput{})
	if !errors.Is(err, ErrInvalidStartTime) {
		t.Errorf("odotettiin ErrInvalidStartTime, saatiin %v", err)
	}

	// 2. Lopetusaika ennen aloitusaikaa
	now := time.Now()
	past := now.Add(-1 * time.Hour)
	_, err = svc.CreateWorkout(ctx, "u1", CreateWorkoutInput{
		StartTime: now,
		EndTime:   &past,
	})
	if !errors.Is(err, ErrInvalidEndTime) {
		t.Errorf("odotettiin ErrInvalidEndTime, saatiin %v", err)
	}

	// 3. Virheellinen nuolipistemäärä (>10)
	_, err = svc.CreateWorkout(ctx, "u1", CreateWorkoutInput{
		StartTime: now,
		ArcheryScores: []domain.ArcheryScore{
			{ScoreValue: 11},
		},
	})
	if !errors.Is(err, ErrInvalidScoreValue) {
		t.Errorf("odotettiin ErrInvalidScoreValue, saatiin %v", err)
	}

	// 4. Virheellinen X-tulos (Score 9 ei voi olla X)
	_, err = svc.CreateWorkout(ctx, "u1", CreateWorkoutInput{
		StartTime: now,
		ArcheryScores: []domain.ArcheryScore{
			{ScoreValue: 9, IsX: true},
		},
	})
	if !errors.Is(err, ErrInvalidXScore) {
		t.Errorf("odotettiin ErrInvalidXScore, saatiin %v", err)
	}
}
