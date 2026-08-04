package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"liike_app/internal/repository"
	service "liike_app/internal/services"

	_ "modernc.org/sqlite"
)

// setupTestDB luo handler-testejä varten SQLite in-memory -tietokannan.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("testitietokannan avaaminen epäonnistui: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("testitietokannan sulkeminen epäonnistui: %v", err)
		}
	})

	_, err = db.Exec(`
		CREATE TABLE users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX idx_users_email ON users(email);
	`)
	if err != nil {
		t.Fatalf("users-taulun luonti epäonnistui: %v", err)
	}

	return db
}

func newTestAuthService(t *testing.T) *service.AuthService {
	t.Helper()

	db := setupTestDB(t)
	repo := repository.NewRepository(db)

	return service.NewAuthService(repo, "test-secret-key")
}

// --- AuthHandler.Register ---

func TestAuthHandler_Register(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		wantStatusCode int
		wantError      string
	}{
		{
			name: "success",
			body: `{
				"email": "test@example.com",
				"name": "Testi Käyttäjä",
				"password": "password123"
			}`,
			wantStatusCode: http.StatusCreated,
		},
		{
			name:           "invalid json",
			body:           `{"email":`,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "virheellinen JSON-syöte",
		},
		{
			name: "missing email",
			body: `{
				"name": "Testi Käyttäjä",
				"password": "password123"
			}`,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "sähköpostiosoite on pakollinen",
		},
		{
			name: "missing name",
			body: `{
				"email": "test@example.com",
				"password": "password123"
			}`,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "nimi on pakollinen",
		},
		{
			name: "password too short",
			body: `{
				"email": "test@example.com",
				"name": "Testi Käyttäjä",
				"password": "1234567"
			}`,
			wantStatusCode: http.StatusBadRequest,
			wantError:      "salasanan on oltava vähintään 8 merkkiä pitkä",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authSvc := newTestAuthService(t)
			handler := NewAuthHandler(authSvc)

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/auth/register",
				bytes.NewBufferString(tt.body),
			)
			request.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()

			handler.Register(recorder, request)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf(
					"odotettu HTTP-status %d, saatiin %d",
					tt.wantStatusCode,
					recorder.Code,
				)
			}

			if tt.wantError != "" {
				var response map[string]string

				if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
					t.Fatalf(
						"virhevastauksen JSON:n lukeminen epäonnistui: %v",
						err,
					)
				}

				if response["error"] != tt.wantError {
					t.Errorf(
						"odotettu error '%s', saatiin '%s'",
						tt.wantError,
						response["error"],
					)
				}

				return
			}

			var response service.AuthResponse

			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf(
					"onnistuneen vastauksen JSON:n lukeminen epäonnistui: %v",
					err,
				)
			}

			if response.Token == "" {
				t.Error("onnistuneessa Register-vastauksessa pitäisi olla token")
			}

			if response.User == nil {
				t.Fatal("onnistuneessa Register-vastauksessa pitäisi olla user")
			}

			if response.User.Email != "test@example.com" {
				t.Errorf(
					"odotettu email 'test@example.com', saatiin '%s'",
					response.User.Email,
				)
			}

			if response.User.Name != "Testi Käyttäjä" {
				t.Errorf(
					"odotettu name 'Testi Käyttäjä', saatiin '%s'",
					response.User.Name,
				)
			}
		})
	}
}

func TestAuthHandler_Register_EmailAlreadyExists(t *testing.T) {
	authSvc := newTestAuthService(t)
	handler := NewAuthHandler(authSvc)

	body := `{
		"email": "duplicate@example.com",
		"name": "Testi Käyttäjä",
		"password": "password123"
	}`

	firstRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(body),
	)
	firstRequest.Header.Set("Content-Type", "application/json")

	firstRecorder := httptest.NewRecorder()

	handler.Register(firstRecorder, firstRequest)

	if firstRecorder.Code != http.StatusCreated {
		t.Fatalf(
			"ensimmäisen rekisteröinnin odotettiin palauttavan 201, saatiin %d",
			firstRecorder.Code,
		)
	}

	secondRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		bytes.NewBufferString(body),
	)
	secondRequest.Header.Set("Content-Type", "application/json")

	secondRecorder := httptest.NewRecorder()

	handler.Register(secondRecorder, secondRequest)

	if secondRecorder.Code != http.StatusConflict {
		t.Errorf(
			"odotettu HTTP-status %d, saatiin %d",
			http.StatusConflict,
			secondRecorder.Code,
		)
	}

	var response map[string]string

	if err := json.NewDecoder(secondRecorder.Body).Decode(&response); err != nil {
		t.Fatalf(
			"virhevastauksen JSON:n lukeminen epäonnistui: %v",
			err,
		)
	}

	if response["error"] != repository.ErrEmailAlreadyExists.Error() {
		t.Errorf(
			"odotettu error '%s', saatiin '%s'",
			repository.ErrEmailAlreadyExists.Error(),
			response["error"],
		)
	}
}

// --- AuthHandler.Login ---

func TestAuthHandler_Login(t *testing.T) {
	tests := []struct {
		name           string
		email          string
		password       string
		wantStatusCode int
		wantError      string
	}{
		{
			name:           "success",
			email:          "test@example.com",
			password:       "password123",
			wantStatusCode: http.StatusOK,
		},
		{
			name:           "wrong password",
			email:          "test@example.com",
			password:       "wrong-password",
			wantStatusCode: http.StatusUnauthorized,
			wantError:      service.ErrInvalidCredentials.Error(),
		},
		{
			name:           "user not found",
			email:          "missing@example.com",
			password:       "password123",
			wantStatusCode: http.StatusUnauthorized,
			wantError:      service.ErrInvalidCredentials.Error(),
		},
		{
			name:           "invalid json",
			email:          "",
			password:       "",
			wantStatusCode: http.StatusBadRequest,
			wantError:      "virheellinen JSON-syöte",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authSvc := newTestAuthService(t)
			handler := NewAuthHandler(authSvc)

			// Luodaan käyttäjä Login-testejä varten.
			if tt.name != "user not found" && tt.name != "invalid json" {
				_, err := authSvc.Register(
					context.Background(),
					service.RegisterInput{
						Email:    "test@example.com",
						Name:     "Testi Käyttäjä",
						Password: "password123",
					},
				)
				if err != nil {
					t.Fatalf(
						"testikäyttäjän luonti epäonnistui: %v",
						err,
					)
				}
			}

			var body string

			if tt.name == "invalid json" {
				body = `{"email":`
			} else {
				bodyBytes, err := json.Marshal(service.LoginInput{
					Email:    tt.email,
					Password: tt.password,
				})
				if err != nil {
					t.Fatalf("Login-inputin JSON-muunnos epäonnistui: %v", err)
				}

				body = string(bodyBytes)
			}

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/auth/login",
				bytes.NewBufferString(body),
			)
			request.Header.Set("Content-Type", "application/json")

			recorder := httptest.NewRecorder()

			handler.Login(recorder, request)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf(
					"odotettu HTTP-status %d, saatiin %d",
					tt.wantStatusCode,
					recorder.Code,
				)
			}

			if tt.wantError != "" {
				var response map[string]string

				if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
					t.Fatalf(
						"virhevastauksen JSON:n lukeminen epäonnistui: %v",
						err,
					)
				}

				if response["error"] != tt.wantError {
					t.Errorf(
						"odotettu error '%s', saatiin '%s'",
						tt.wantError,
						response["error"],
					)
				}

				return
			}

			var response service.AuthResponse

			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf(
					"onnistuneen vastauksen JSON:n lukeminen epäonnistui: %v",
					err,
				)
			}

			if response.Token == "" {
				t.Error("onnistuneessa Login-vastauksessa pitäisi olla token")
			}

			if response.User == nil {
				t.Fatal("onnistuneessa Login-vastauksessa pitäisi olla user")
			}

			if response.User.Email != "test@example.com" {
				t.Errorf(
					"odotettu email 'test@example.com', saatiin '%s'",
					response.User.Email,
				)
			}
		})
	}
}

// --- HTTP response helpers ---

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeJSON(
		recorder,
		http.StatusCreated,
		map[string]string{
			"message": "created",
		},
	)

	if recorder.Code != http.StatusCreated {
		t.Errorf(
			"odotettu HTTP-status %d, saatiin %d",
			http.StatusCreated,
			recorder.Code,
		)
	}

	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf(
			"odotettu Content-Type application/json, saatiin %q",
			recorder.Header().Get("Content-Type"),
		)
	}

	var response map[string]string

	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("JSON-vastauksen lukeminen epäonnistui: %v", err)
	}

	if response["message"] != "created" {
		t.Errorf(
			"odotettu message 'created', saatiin '%s'",
			response["message"],
		)
	}
}

// Estetään mahdollinen käyttämätön errors-importti,
// jos testit myöhemmin muuttuvat.
var _ = errors.Is
