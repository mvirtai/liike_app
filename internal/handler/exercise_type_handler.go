package handler

import (
	"errors"
	"net/http"

	"liike_app/internal/repository"
	"liike_app/internal/service"
)

type ExerciseTypeHandler struct {
	service *service.ExerciseTypeService
}

func NewExerciseTypeHandler(service *service.ExerciseTypeService) *ExerciseTypeHandler {
	return &ExerciseTypeHandler{service: service}
}

// List (GET /api/v1/exercise-types)
func (h *ExerciseTypeHandler) List(w http.ResponseWriter, r *http.Request) {
	types, err := h.service.GetAllExerciseTypes(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Virhe harjoitusmuotojen haussa")
		return
	}
	writeJSON(w, http.StatusOK, types)
}

// GetByID (GET /api/v1/exercise-types/{id})
func (h *ExerciseTypeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "Laji-ID puuttuu")
		return
	}

	et, err := h.service.GetExerciseTypeByID(r.Context(), id)
	if errors.Is(err, repository.ErrExerciseTypeNotFound) {
		writeJSONError(w, http.StatusNotFound, "Harjoitusmuotoa ei löytynyt")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Virhe harjoitusmuodon haussa")
		return
	}

	writeJSON(w, http.StatusOK, et)
}
