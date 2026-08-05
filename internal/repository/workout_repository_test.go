package repository_test

import (
	"context"
	"testing"
	"time"

	"liike_app/internal/domain"
	"liike_app/internal/repository"
)

func TestWorkoutRepository_CRUD(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	ctx := context.Background()

	// 1. Luodaan testikäyttäjä
	user, err := repo.CreateUser(ctx, "workout_test@example.com", "Workout Tester", "hash1234")
	if err != nil {
		t.Fatalf("CreateUser epäonnistui: %v", err)
	}

	// 2. Luodaan suoritus ja 2 sarjaa
	reps := 10
	weight := 70.0
	now := time.Now()
	w := &domain.Workout{
		UserID:         user.ID,
		ExerciseTypeID: "10000000-0000-4000-8000-000000000004", // Kyykky
		StartTime:      now,
	}
	sets := []domain.WorkoutSet{
		{SetNumber: 1, Reps: &reps, WeightKg: &weight},
		{SetNumber: 2, Reps: &reps, WeightKg: &weight},
	}

	created, err := repo.CreateWorkout(ctx, w, sets, nil)
	if err != nil {
		t.Fatalf("CreateWorkout epäonnistui: %v", err)
	}

	if created.ID == "" {
		t.Error("luodun suorituksen ID on tyhjä")
	}
	if len(created.Sets) != 2 {
		t.Errorf("odotettiin 2 sarjaa, saatiin %d", len(created.Sets))
	}

	// 3. Haetaan suoritus ID:llä
	found, err := repo.GetWorkoutByID(ctx, created.ID, user.ID)
	if err != nil {
		t.Fatalf("GetWorkoutByID epäonnistui: %v", err)
	}
	if found.ExerciseType == nil || found.ExerciseType.Name != "Kyykky" {
		t.Errorf("odotettiin lajia 'Kyykky', saatiin %v", found.ExerciseType)
	}

	// 4. Päivitetään suoritus (UpdateWorkout)
	newNotes := "Päivitetty kyykytreeni"
	found.Notes = &newNotes
	if err := repo.UpdateWorkout(ctx, found); err != nil {
		t.Fatalf("UpdateWorkout epäonnistui: %v", err)
	}

	updated, err := repo.GetWorkoutByID(ctx, created.ID, user.ID)
	if err != nil {
		t.Fatalf("GetWorkoutByID päivityksen jälkeen epäonnistui: %v", err)
	}
	if updated.Notes == nil || *updated.Notes != newNotes {
		t.Errorf("odotettiin huomautusta '%s', saatiin '%v'", newNotes, updated.Notes)
	}

	// 5. Lisätään sarjoja (AddWorkoutSets)
	newSets := []domain.WorkoutSet{
		{SetNumber: 3, Reps: &reps, WeightKg: &weight},
	}
	if err := repo.AddWorkoutSets(ctx, created.ID, newSets); err != nil {
		t.Fatalf("AddWorkoutSets epäonnistui: %v", err)
	}

	withSets, _ := repo.GetWorkoutByID(ctx, created.ID, user.ID)
	if len(withSets.Sets) != 3 {
		t.Errorf("odotettiin 3 sarjaa, saatiin %d", len(withSets.Sets))
	}

	// 6. Lisätään nuolituloksia (AddArcheryScores)
	scores := []domain.ArcheryScore{
		{EndNumber: 1, ArrowNumber: 1, ScoreValue: 10, IsX: true},
	}
	if err := repo.AddArcheryScores(ctx, created.ID, scores); err != nil {
		t.Fatalf("AddArcheryScores epäonnistui: %v", err)
	}

	withScores, _ := repo.GetWorkoutByID(ctx, created.ID, user.ID)
	if len(withScores.ArcheryScores) != 1 {
		t.Errorf("odotettiin 1 nuolitulos, saatiin %d", len(withScores.ArcheryScores))
	}

	// 7. Listataan käyttäjän suoritukset suodattimilla
	from := now.Add(-1 * time.Hour)
	to := now.Add(1 * time.Hour)
	filter := repository.WorkoutFilter{
		ExerciseTypeID: "10000000-0000-4000-8000-000000000004",
		FromDate:       &from,
		ToDate:         &to,
		Limit:          10,
		Offset:         0,
	}
	list, err := repo.ListWorkoutsByUserID(ctx, user.ID, filter)
	if err != nil {
		t.Fatalf("ListWorkoutsByUserID epäonnistui: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("odotettiin 1 suoritus, saatiin %d", len(list))
	}

	// 8. Poistetaan suoritus
	if err := repo.DeleteWorkout(ctx, created.ID, user.ID); err != nil {
		t.Fatalf("DeleteWorkout epäonnistui: %v", err)
	}

	// Varmistetaan että poistui
	_, err = repo.GetWorkoutByID(ctx, created.ID, user.ID)
	if err == nil {
		t.Error("odotettiin virhettä poistetun suorituksen haussa")
	}
}
