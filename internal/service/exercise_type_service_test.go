package service

import (
	"context"
	"testing"

	"liike_app/internal/repository"
)

func TestExerciseTypeService_GetAllAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	svc := NewExerciseTypeService(repo)
	ctx := context.Background()

	// 1. GetAllExerciseTypes
	types, err := svc.GetAllExerciseTypes(ctx)
	if err != nil {
		t.Fatalf("GetAllExerciseTypes epäonnistui: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("odotettiin oletuslajeja, saatiin 0")
	}

	// 2. GetExerciseTypeByID
	first := types[0]
	found, err := svc.GetExerciseTypeByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetExerciseTypeByID epäonnistui: %v", err)
	}
	if found.Name != first.Name {
		t.Errorf("odotettu nimi '%s', saatiin '%s'", first.Name, found.Name)
	}

	// 3. Virhetilanne: Ei-olemassa oleva ID
	_, err = svc.GetExerciseTypeByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("odotettiin virhettä ei-olemassa olevalla ID:llä")
	}
}
