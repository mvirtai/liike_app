package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"liike_app/internal/handler"
)

func TestHealthCheckHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/health", nil)
	if err != nil {
		t.Fatalf("failed to create HTTP request: %v", err)
	}

	rr := httptest.NewRecorder()
	http.HandlerFunc(handler.HealthCheckHandler).ServeHTTP(rr, req)

	// 1. Verify Status Code
	if rr.Code != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", rr.Code, http.StatusOK)
	}

	// 2. Verify Content-Type Header
	expectedContentType := "application/json"
	contentType := rr.Header().Get("Content-Type")
	if contentType != expectedContentType {
		t.Errorf("handler returned wrong content-type header: got '%s' want '%s'", contentType, expectedContentType)
	}

	// 3. Verify JSON Response Body
	var resp handler.HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response JSON body: %v", err)
	}

	if resp.Status != "OK" {
		t.Errorf("expected Status 'OK', got '%s'", resp.Status)
	}

	if resp.Service != "Liike App API" {
		t.Errorf("expected Service 'Liike App API', got '%s'", resp.Service)
	}

	if resp.Version != "1.0.0" {
		t.Errorf("expected Version '1.0.0', got '%s'", resp.Version)
	}

	if resp.Timestamp.IsZero() {
		t.Errorf("expected non-zero Timestamp, got zero time")
	}

	// Ensure timestamp is reasonably recent (within 5 seconds)
	if time.Since(resp.Timestamp) > 5*time.Second {
		t.Errorf("timestamp is too old: %v", resp.Timestamp)
	}
}
