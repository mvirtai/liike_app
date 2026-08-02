package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

// HealthResponse määrittelee terveystarkastuksen rakenteen
type HealthResponse struct {
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Service   string    `json:"service"`
	Version   string    `json:"version"`
}

// HealthCheckHandler käsittelee GET /api/v1/health pyynnöt
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "OK",
		Timestamp: time.Now().UTC(),
		Service:   "Liike App API",
		Version:   "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Virhe vastauksen muodostamisessa",
			http.StatusInternalServerError)
	}
}
