package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"liike_app/internal/domain"
	"liike_app/internal/middleware"
	"liike_app/internal/repository"
	"liike_app/internal/service"
)

func TestWorkoutHandler_FullFlow(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	svc := service.NewWorkoutService(repo)
	h := NewWorkoutHandler(svc)

	// Luodaan testikäyttäjä
	user, err := repo.CreateUser(context.Background(), "workout_h@example.com", "H User", "pass1234")
	if err != nil {
		t.Fatalf("CreateUser epäonnistui: %v", err)
	}

	// 1. POST /api/v1/workouts (Create)
	input := service.CreateWorkoutInput{
		ExerciseTypeID: "10000000-0000-4000-8000-000000000003", // Jousiammunta
		StartTime:      time.Now(),
		ArcheryScores: []domain.ArcheryScore{
			{EndNumber: 1, ArrowNumber: 1, ScoreValue: 10, IsX: true},
		},
	}
	body, _ := json.Marshal(input)

	reqCreate := httptest.NewRequest("POST", "/api/v1/workouts", bytes.NewReader(body))
	reqCreate = reqCreate.WithContext(context.WithValue(reqCreate.Context(), middleware.UserIDKey, user.ID))
	wCreate := httptest.NewRecorder()

	h.Create(wCreate, reqCreate)

	if wCreate.Code != http.StatusCreated {
		t.Fatalf("odotettiin tilaa 201 Created, saatiin %d", wCreate.Code)
	}

	var resp service.WorkoutDetailResponse
	if err := json.NewDecoder(wCreate.Body).Decode(&resp); err != nil {
		t.Fatalf("JSON-parsinta epäonnistui: %v", err)
	}
	workoutID := resp.Workout.ID

	// 2. GET /api/v1/workouts/{id} (GetByID)
	reqGet := httptest.NewRequest("GET", "/api/v1/workouts/"+workoutID, nil)
	reqGet.SetPathValue("id", workoutID)
	reqGet = reqGet.WithContext(context.WithValue(reqGet.Context(), middleware.UserIDKey, user.ID))
	wGet := httptest.NewRecorder()

	h.GetByID(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("odotettiin tilaa 200 OK, saatiin %d", wGet.Code)
	}

	// 3. GET /api/v1/workouts (List)
	reqList := httptest.NewRequest("GET", "/api/v1/workouts?limit=10", nil)
	reqList = reqList.WithContext(context.WithValue(reqList.Context(), middleware.UserIDKey, user.ID))
	wList := httptest.NewRecorder()

	h.List(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("odotettiin tilaa 200 OK, saatiin %d", wList.Code)
	}

	// 4. POST /api/v1/workouts/{id}/archery-scores (AddArcheryScores)
	newScores := []domain.ArcheryScore{
		{EndNumber: 1, ArrowNumber: 2, ScoreValue: 9, IsX: false},
	}
	scoresBody, _ := json.Marshal(newScores)
	reqScore := httptest.NewRequest("POST", "/api/v1/workouts/"+workoutID+"/archery-scores", bytes.NewReader(scoresBody))
	reqScore.SetPathValue("id", workoutID)
	reqScore = reqScore.WithContext(context.WithValue(reqScore.Context(), middleware.UserIDKey, user.ID))
	wScore := httptest.NewRecorder()

	h.AddArcheryScores(wScore, reqScore)

	if wScore.Code != http.StatusOK {
		t.Fatalf("odotettiin tilaa 200 OK, saatiin %d", wScore.Code)
	}

	// 5. DELETE /api/v1/workouts/{id} (Delete)
	reqDelete := httptest.NewRequest("DELETE", "/api/v1/workouts/"+workoutID, nil)
	reqDelete.SetPathValue("id", workoutID)
	reqDelete = reqDelete.WithContext(context.WithValue(reqDelete.Context(), middleware.UserIDKey, user.ID))
	wDelete := httptest.NewRecorder()

	h.Delete(wDelete, reqDelete)

	if wDelete.Code != http.StatusNoContent {
		t.Fatalf("odotettiin tilaa 24 StatusNoContent, saatiin %d", wDelete.Code)
	}
}

func TestWorkoutHandler_UnauthorizedAndErrors(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewRepository(db)
	svc := service.NewWorkoutService(repo)
	h := NewWorkoutHandler(svc)

	// 1. Ilman autentikaatiota (Context ei sisällä user_id)
	req := httptest.NewRequest("GET", "/api/v1/workouts", nil)
	w := httptest.NewRecorder()
	h.List(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("odotettiin tilaa 401 Unauthorized, saatiin %d", w.Code)
	}

	// 2. Virheellinen pyynnön muoto (Invalid JSON body)
	reqCreate := httptest.NewRequest("POST", "/api/v1/workouts", bytes.NewReader([]byte("invalid json")))
	reqCreate = reqCreate.WithContext(context.WithValue(reqCreate.Context(), middleware.UserIDKey, "user-123"))
	wCreate := httptest.NewRecorder()
	h.Create(wCreate, reqCreate)
	if wCreate.Code != http.StatusBadRequest {
		t.Errorf("odotettiin tilaa 400 Bad Request, saatiin %d", wCreate.Code)
	}

	// 3. Ei-olemassa oleva suoritus haussa
	reqGet := httptest.NewRequest("GET", "/api/v1/workouts/non-existent-id", nil)
	reqGet.SetPathValue("id", "non-existent-id")
	reqGet = reqGet.WithContext(context.WithValue(reqGet.Context(), middleware.UserIDKey, "user-123"))
	wGet := httptest.NewRecorder()
	h.GetByID(wGet, reqGet)
	if wGet.Code != http.StatusNotFound {
		t.Errorf("odotettiin tilaa 404 Not Found, saatiin %d", wGet.Code)
	}
}
