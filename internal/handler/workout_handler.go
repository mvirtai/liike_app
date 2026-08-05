package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"liike_app/internal/domain"
	"liike_app/internal/middleware"
	"liike_app/internal/repository"
	"liike_app/internal/service"
)

type WorkoutHandler struct {
	service *service.WorkoutService
}

func NewWorkoutHandler(svc *service.WorkoutService) *WorkoutHandler {
	return &WorkoutHandler{service: svc}
}

// Create (POST /api/v1/workouts)
func (h *WorkoutHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Autentikaatio vaaditaan")
		return
	}

	var input service.CreateWorkoutInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Virheellinen pyynnön muoto")
		return
	}

	resp, err := h.service.CreateWorkout(r.Context(), userID, input)
	if errors.Is(err, service.ErrInvalidStartTime) || errors.Is(err, service.ErrInvalidEndTime) ||
		errors.Is(err, service.ErrInvalidScoreValue) || errors.Is(err, service.ErrInvalidXScore) {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Virhe suorituksen tallennuksessa: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

// GetByID (GET /api/v1/workouts/{id})
func (h *WorkoutHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Autentikaatio vaaditaan")
		return
	}

	workoutID := r.PathValue("id")
	resp, err := h.service.GetWorkoutByID(r.Context(), workoutID, userID)
	if errors.Is(err, repository.ErrWorkoutNotFound) {
		writeJSONError(w, http.StatusNotFound, "Suoritusta ei löytynyt")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Virhe suorituksen haussa")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// List (GET /api/v1/workouts)
func (h *WorkoutHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Autentikaatio vaaditaan")
		return
	}

	query := r.URL.Query()
	filter := repository.WorkoutFilter{
		ExerciseTypeID: query.Get("exercise_type_id"),
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = l
		}
	}
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = o
		}
	}

	workouts, err := h.service.ListWorkouts(r.Context(), userID, filter)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Virhe suoritusten listauksessa")
		return
	}

	writeJSON(w, http.StatusOK, workouts)
}

// Delete (DELETE /api/v1/workouts/{id})
func (h *WorkoutHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Autentikaatio vaaditaan")
		return
	}

	workoutID := r.PathValue("id")
	err := h.service.DeleteWorkout(r.Context(), workoutID, userID)
	if errors.Is(err, repository.ErrWorkoutNotFound) {
		writeJSONError(w, http.StatusNotFound, "Suoritusta ei löytynyt")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Virhe suorituksen poistossa")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddArcheryScores (POST /api/v1/workouts/{id}/archery-scores)
func (h *WorkoutHandler) AddArcheryScores(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "Autentikaatio vaaditaan")
		return
	}

	workoutID := r.PathValue("id")
	var scores []domain.ArcheryScore
	if err := json.NewDecoder(r.Body).Decode(&scores); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Virheellinen pyynnön muoto")
		return
	}

	resp, err := h.service.AddArcheryScores(r.Context(), workoutID, userID, scores)
	if errors.Is(err, repository.ErrWorkoutNotFound) {
		writeJSONError(w, http.StatusNotFound, "Suoritusta ei löytynyt")
		return
	}
	if errors.Is(err, service.ErrInvalidScoreValue) || errors.Is(err, service.ErrInvalidXScore) {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Virhe ammuntojen lisäyksessä")
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
