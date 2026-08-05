package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"liike_app/internal/domain"
	"liike_app/internal/repository"
	"liike_app/internal/service"
)

func TestExerciseTypeHandler_ListAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	svc := service.NewExerciseTypeService(repo)
	h := NewExerciseTypeHandler(svc)

	// 1. GET /api/v1/exercise-types (List)
	req := httptest.NewRequest("GET", "/api/v1/exercise-types", nil)
	w := httptest.NewRecorder()

	h.List(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("odotettiin tilaa 200, saatiin %d", w.Code)
	}

	var types []domain.ExerciseType
	if err := json.NewDecoder(w.Body).Decode(&types); err != nil {
		t.Fatalf("JSON-parsinta epäonnistui: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("odotettiin oletuslajeja, saatiin 0")
	}

	// 2. GET /api/v1/exercise-types/{id} (GetByID - Success)
	first := types[0]
	reqByID := httptest.NewRequest("GET", "/api/v1/exercise-types/"+first.ID, nil)
	reqByID.SetPathValue("id", first.ID)
	wByID := httptest.NewRecorder()

	h.GetByID(wByID, reqByID)

	if wByID.Code != http.StatusOK {
		t.Fatalf("odotettiin tilaa 200, saatiin %d", wByID.Code)
	}

	var found domain.ExerciseType
	if err := json.NewDecoder(wByID.Body).Decode(&found); err != nil {
		t.Fatalf("JSON-parsinta epäonnistui: %v", err)
	}
	if found.Name != first.Name {
		t.Errorf("odotettu nimi '%s', saatiin '%s'", first.Name, found.Name)
	}

	// 3. GET /api/v1/exercise-types/{id} (GetByID - Not Found)
	reqNotFound := httptest.NewRequest("GET", "/api/v1/exercise-types/non-existent-id", nil)
	reqNotFound.SetPathValue("id", "non-existent-id")
	wNotFound := httptest.NewRecorder()

	h.GetByID(wNotFound, reqNotFound)

	if wNotFound.Code != http.StatusNotFound {
		t.Errorf("odotettiin tilaa 404, saatiin %d", wNotFound.Code)
	}
}
