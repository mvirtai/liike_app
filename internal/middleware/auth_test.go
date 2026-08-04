package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"liike_app/internal/repository"
	service "liike_app/internal/services"

	_ "modernc.org/sqlite"
)

// setupTestDB luo testitietokannan middleware-testejä varten.
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

func createTestToken(t *testing.T, authSvc *service.AuthService) string {
	t.Helper()

	// Käytetään AuthService.Registeria tokenin luomiseen.
	// Näin testissä ei tarvitse päästä käsiksi generateTokeniin.
	resp, err := authSvc.Register(
		context.Background(),
		service.RegisterInput{
			Email:    "test@example.com",
			Name:     "Testi Käyttäjä",
			Password: "password123",
		},
	)
	if err != nil {
		t.Fatalf("testikäyttäjän rekisteröinti epäonnistui: %v", err)
	}

	return resp.Token
}

// --- AuthMiddleware ---

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		authorization  string
		wantStatusCode int
		wantNextCalled bool
		wantUserID     string
		wantUserEmail  string
		wantError      string
	}{
		{
			name:           "success",
			authorization:  "Bearer valid-token",
			wantStatusCode: http.StatusOK,
			wantNextCalled: true,
			wantUserID:     "test-user-id",
			wantUserEmail:  "test@example.com",
		},
		{
			name:           "missing authorization header",
			authorization:  "",
			wantStatusCode: http.StatusUnauthorized,
			wantNextCalled: false,
			wantError:      "puuttuva Authorization-otsikko",
		},
		{
			name:           "invalid authorization format",
			authorization:  "Basic abc123",
			wantStatusCode: http.StatusUnauthorized,
			wantNextCalled: false,
			wantError:      "virheellinen Authorization-otsikkomuoto",
		},
		{
			name:           "missing bearer token",
			authorization:  "Bearer",
			wantStatusCode: http.StatusUnauthorized,
			wantNextCalled: false,
			wantError:      "virheellinen Authorization-otsikkomuoto",
		},
		{
			name:           "invalid token",
			authorization:  "Bearer invalid-token",
			wantStatusCode: http.StatusUnauthorized,
			wantNextCalled: false,
			wantError:      "virheellinen tai vanhentunut token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authSvc := newTestAuthService(t)

			var token string

			if tt.name == "success" {
				// Register luo oikean JWT-tokenin, mutta tarvitsemme
				// testissä halutun user_id:n tarkistamista varten.
				resp, err := authSvc.Register(
					context.Background(),
					service.RegisterInput{
						Email:    "test@example.com",
						Name:     "Testi Käyttäjä",
						Password: "password123",
					},
				)
				if err != nil {
					t.Fatalf("testikäyttäjän rekisteröinti epäonnistui: %v", err)
				}

				token = resp.Token
			} else {
				token = tt.authorization
			}

			request := httptest.NewRequest(
				http.MethodGet,
				"/protected",
				nil,
			)

			if tt.name == "success" {
				request.Header.Set("Authorization", "Bearer "+token)
			} else if tt.authorization != "" {
				request.Header.Set("Authorization", tt.authorization)
			}

			nextCalled := false

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				nextCalled = true

				userID, ok := GetUserIDFromContext(r.Context())
				if !ok {
					t.Error("user_id puuttuu request-kontekstista")
				}

				if userID == "" {
					t.Error("user_id ei saisi olla tyhjä")
				}

				email, ok := r.Context().Value(UserEmailKey).(string)
				if !ok {
					t.Error("user_email puuttuu request-kontekstista")
				}

				if email == "" {
					t.Error("user_email ei saisi olla tyhjä")
				}

				w.WriteHeader(http.StatusOK)
			})

			handler := AuthMiddleware(authSvc)(next)

			recorder := httptest.NewRecorder()

			// Virheelliset authorization-headerit ovat tällä hetkellä
			// tarkoituksella mukana testeissä, jotta middlewarestä
			// löytyvät puuttuvat return-lauseet.
			handler.ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatusCode {
				t.Errorf(
					"odotettu HTTP-status %d, saatiin %d",
					tt.wantStatusCode,
					recorder.Code,
				)
			}

			if nextCalled != tt.wantNextCalled {
				t.Errorf(
					"nextCalled: odotettiin %v, saatiin %v",
					tt.wantNextCalled,
					nextCalled,
				)
			}

			if tt.wantError != "" {
				var response ErrorResponse

				if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
					t.Fatalf(
						"virhevastauksen JSON:n lukeminen epäonnistui: %v",
						err,
					)
				}

				if response.Error != tt.wantError {
					t.Errorf(
						"odotettu virhe '%s', saatiin '%s'",
						tt.wantError,
						response.Error,
					)
				}

				contentType := recorder.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf(
						"odotettu Content-Type application/json, saatiin %q",
						contentType,
					)
				}
			}
		})
	}
}

// TestAuthMiddleware_Context varmistaa erikseen, että middleware
// siirtää tokenin käyttäjätiedot request-kontekstiin.
func TestAuthMiddleware_Context(t *testing.T) {
	authSvc := newTestAuthService(t)

	response, err := authSvc.Register(
		context.Background(),
		service.RegisterInput{
			Email:    "context@example.com",
			Name:     "Context Test",
			Password: "password123",
		},
	)
	if err != nil {
		t.Fatalf("testikäyttäjän rekisteröinti epäonnistui: %v", err)
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/protected",
		nil,
	)
	request.Header.Set("Authorization", "Bearer "+response.Token)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserIDFromContext(r.Context())
		if !ok {
			t.Fatal("user_id puuttuu contextista")
		}

		if userID != response.User.ID {
			t.Errorf(
				"odotettu user_id '%s', saatiin '%s'",
				response.User.ID,
				userID,
			)
		}

		email, ok := r.Context().Value(UserEmailKey).(string)
		if !ok {
			t.Fatal("user_email puuttuu contextista")
		}

		if email != response.User.Email {
			t.Errorf(
				"odotettu email '%s', saatiin '%s'",
				response.User.Email,
				email,
			)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMiddleware(authSvc)(next)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Errorf(
			"odotettu HTTP-status 200, saatiin %d",
			recorder.Code,
		)
	}
}

// --- GetUserIDFromContext ---

func TestGetUserIDFromContext(t *testing.T) {
	tests := []struct {
		name    string
		context context.Context
		wantID  string
		wantOK  bool
	}{
		{
			name: "user id exists",
			context: context.WithValue(
				context.Background(),
				UserIDKey,
				"user-123",
			),
			wantID: "user-123",
			wantOK: true,
		},
		{
			name:    "user id does not exist",
			context: context.Background(),
			wantID:  "",
			wantOK:  false,
		},
		{
			name: "user id has wrong type",
			context: context.WithValue(
				context.Background(),
				UserIDKey,
				123,
			),
			wantID: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := GetUserIDFromContext(tt.context)

			if gotID != tt.wantID {
				t.Errorf(
					"odotettu ID '%s', saatiin '%s'",
					tt.wantID,
					gotID,
				)
			}

			if gotOK != tt.wantOK {
				t.Errorf(
					"odotettu ok=%v, saatiin ok=%v",
					tt.wantOK,
					gotOK,
				)
			}
		})
	}
}
