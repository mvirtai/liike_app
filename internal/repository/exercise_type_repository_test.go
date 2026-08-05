package repository_test

import (
	"context"
	"testing"

	"liike_app/internal/repository"
)

func TestExerciseTypeRepository_GetAllAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	ctx := context.Background()

	types, err := repo.GetAllExerciseTypes(ctx)
	if err != nil {
		t.Fatalf("GetAllExerciseTypes epäonnistui: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("odotettiin oletuslajeja kannasta, saatiin 0")
	}

	//  Testaa GetByID ensimmäisellä lajilla
	first := types[0]
	found, err := repo.GetExerciseTypeByID(ctx, first.ID)
	if err != nil {
		t.Fatalf("GetExerciseTypeByID epäonnistui: %v", err)
	}
	if found.Name != first.Name {
		t.Errorf("odotettu nimi %s, saatiin %s", first.Name, found.Name)
	}

	// Ei olemassa oleva ID
	_, err = repo.GetExerciseTypeByID(ctx, "non-existent-id")
	if err == nil {
		t.Error("odotettiin virhettä ei-olemassa olevalla ID:llä")
	}
}
