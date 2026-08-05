package handler

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"liike_app/internal/middleware"
	"liike_app/internal/repository"
	"liike_app/internal/service"

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
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

		CREATE TABLE IF NOT EXISTS exercise_types (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			category TEXT NOT NULL,
			description TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);

		INSERT OR IGNORE INTO exercise_types (id, name, category, description) VALUES
			('10000000-0000-4000-8000-000000000001', 'Kävely', 'cardio', 'Kävelylenkki matkalla, kestolla ja sykkeellä'),
			('10000000-0000-4000-8000-000000000002', 'Juoksu', 'cardio', 'Juoksulenkki matkalla, kestolla ja sykkeellä'),
			('10000000-0000-4000-8000-000000000003', 'Jousiammunta', 'archery', 'Jousiammunnan sarja- ja nuolikohtainen tuloskirjaus'),
			('10000000-0000-4000-8000-000000000004', 'Kyykky', 'strength', 'Jalkatreeni toistoilla ja painoilla'),
			('10000000-0000-4000-8000-000000000005', 'Vatsalihaslankku', 'strength', 'Keskivartalon pito sekunneissa'),
			('10000000-0000-4000-8000-000000000006', 'Painoharjoittelu', 'strength', 'Yleinen kuntosalitreeni sarjoilla ja painoilla'),
			('10000000-0000-4000-8000-000000000007', 'Jooga', 'flexibility', 'Kehonhuolto ja joogaharjoitus muistiinpanoilla');

		CREATE TABLE IF NOT EXISTS workouts (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			exercise_type_id TEXT NOT NULL,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			duration_seconds INTEGER,
			distance_km REAL,
			avg_heart_rate INTEGER,
			calories_burned INTEGER,
			notes TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (exercise_type_id) REFERENCES exercise_types(id) ON DELETE RESTRICT
		);

		CREATE TABLE IF NOT EXISTS workout_sets (
			id TEXT PRIMARY KEY,
			workout_id TEXT NOT NULL,
			set_number INTEGER NOT NULL,
			reps INTEGER,
			weight_kg REAL,
			duration_seconds INTEGER,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE,
			UNIQUE(workout_id, set_number)
		);

		CREATE TABLE IF NOT EXISTS archery_scores (
			id TEXT PRIMARY KEY,
			workout_id TEXT NOT NULL,
			end_number INTEGER NOT NULL,
			arrow_number INTEGER NOT NULL,
			score_value INTEGER NOT NULL CHECK(score_value >= 0 AND score_value <= 10),
			is_x BOOLEAN NOT NULL DEFAULT 0 CHECK(is_x IN (0, 1)),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (workout_id) REFERENCES workouts(id) ON DELETE CASCADE,
			UNIQUE(workout_id, end_number, arrow_number)
		);
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
					t.Fatalf(
						"Login-inputin JSON-muunnos epäonnistui: %v",
						err,
					)
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
					"onnistuneen Login-vastauksen JSON:n lukeminen epäonnistui: %v",
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

// --- AuthHandler.Me ---

func TestAuthHandler_Me(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		authSvc := newTestAuthService(t)
		handler := NewAuthHandler(authSvc)

		registerResponse, err := authSvc.Register(
			context.Background(),
			service.RegisterInput{
				Email:    "me@example.com",
				Name:     "Me Test User",
				Password: "password123",
			},
		)
		if err != nil {
			t.Fatalf(
				"testikäyttäjän rekisteröinti epäonnistui: %v",
				err,
			)
		}

		// Simuloidaan AuthMiddlewaren toimintaa:
		// middleware lisää käyttäjän ID:n requestin contextiin.
		ctx := context.WithValue(
			context.Background(),
			middleware.UserIDKey,
			registerResponse.User.ID,
		)

		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/auth/me",
			nil,
		).WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.Me(recorder, request)

		if recorder.Code != http.StatusOK {
			t.Fatalf(
				"odotettu HTTP-status %d, saatiin %d",
				http.StatusOK,
				recorder.Code,
			)
		}

		if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
			t.Errorf(
				"odotettu Content-Type application/json, saatiin %q",
				contentType,
			)
		}

		var response map[string]any

		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf(
				"Me-vastauksen JSON:n lukeminen epäonnistui: %v",
				err,
			)
		}

		if response["id"] != registerResponse.User.ID {
			t.Errorf(
				"odotettu käyttäjän ID '%s', saatiin '%v'",
				registerResponse.User.ID,
				response["id"],
			)
		}

		if response["email"] != "me@example.com" {
			t.Errorf(
				"odotettu email 'me@example.com', saatiin '%v'",
				response["email"],
			)
		}

		if response["name"] != "Me Test User" {
			t.Errorf(
				"odotettu nimi 'Me Test User', saatiin '%v'",
				response["name"],
			)
		}

		// PasswordHash on json:"-":n ansiosta piilotettu.
		if strings.Contains(recorder.Body.String(), "password_hash") ||
			strings.Contains(recorder.Body.String(), "PasswordHash") {
			t.Error(
				"Me-vastauksessa ei saa olla password_hash-kenttää",
			)
		}
	})

	t.Run("missing user id in context", func(t *testing.T) {
		authSvc := newTestAuthService(t)
		handler := NewAuthHandler(authSvc)

		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/auth/me",
			nil,
		)

		recorder := httptest.NewRecorder()

		handler.Me(recorder, request)

		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf(
				"odotettu HTTP-status %d, saatiin %d",
				http.StatusUnauthorized,
				recorder.Code,
			)
		}

		var response map[string]string

		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf(
				"virhevastauksen JSON:n lukeminen epäonnistui: %v",
				err,
			)
		}

		if response["error"] != "autentikaatio puuttuu" {
			t.Errorf(
				"odotettu error 'autentikaatio puuttuu', saatiin '%s'",
				response["error"],
			)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		authSvc := newTestAuthService(t)
		handler := NewAuthHandler(authSvc)

		ctx := context.WithValue(
			context.Background(),
			middleware.UserIDKey,
			"non-existent-user-id",
		)

		request := httptest.NewRequest(
			http.MethodGet,
			"/api/v1/auth/me",
			nil,
		).WithContext(ctx)

		recorder := httptest.NewRecorder()

		handler.Me(recorder, request)

		if recorder.Code != http.StatusNotFound {
			t.Fatalf(
				"odotettu HTTP-status %d, saatiin %d",
				http.StatusNotFound,
				recorder.Code,
			)
		}

		var response map[string]string

		if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
			t.Fatalf(
				"virhevastauksen JSON:n lukeminen epäonnistui: %v",
				err,
			)
		}

		if response["error"] != "käyttäjää ei löydy" {
			t.Errorf(
				"odotettu error 'käyttäjää ei löydy', saatiin '%s'",
				response["error"],
			)
		}
	})
}

// --- writeJSON ---

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
		t.Fatalf(
			"JSON-vastauksen lukeminen epäonnistui: %v",
			err,
		)
	}

	if response["message"] != "created" {
		t.Errorf(
			"odotettu message 'created', saatiin '%s'",
			response["message"],
		)
	}
}
