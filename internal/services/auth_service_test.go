package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"liike_app/internal/domain"
	"liike_app/internal/repository"

	_ "modernc.org/sqlite"
)

// --- Test helpers ---

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

func newTestAuthService(t *testing.T) *AuthService {
	t.Helper()

	db := setupTestDB(t)
	repo := repository.NewRepository(db)

	return NewAuthService(repo, "test-secret-key")
}

func createTestUser(t *testing.T, repo *repository.Repository) {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword(
		[]byte("password123"),
		bcrypt.DefaultCost,
	)
	if err != nil {
		t.Fatalf("testisalasanan hajautus epäonnistui: %v", err)
	}

	_, err = repo.CreateUser(
		context.Background(),
		"user@example.com",
		"Testi Käyttäjä",
		string(hash),
	)
	if err != nil {
		t.Fatalf("testikäyttäjän luonti epäonnistui: %v", err)
	}
}

// --- validateRegisterInput ---

func TestValidateRegisterInput(t *testing.T) {
	tests := []struct {
		name    string
		input   RegisterInput
		wantErr string
	}{
		{
			name: "success",
			input: RegisterInput{
				Email:    "test@example.com",
				Name:     "Testi Käyttäjä",
				Password: "password123",
			},
		},
		{
			name: "missing email",
			input: RegisterInput{
				Email:    "",
				Name:     "Testi Käyttäjä",
				Password: "password123",
			},
			wantErr: "sähköpostiosoite on pakollinen",
		},
		{
			name: "missing name",
			input: RegisterInput{
				Email:    "test@example.com",
				Name:     "",
				Password: "password123",
			},
			wantErr: "nimi on pakollinen",
		},
		{
			name: "password too short",
			input: RegisterInput{
				Email:    "test@example.com",
				Name:     "Testi Käyttäjä",
				Password: "1234567",
			},
			wantErr: "salasanan on oltava vähintään 8 merkkiä pitkä",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRegisterInput(tt.input)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("odotettiin nil-virhettä, saatiin: %v", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("odotettiin virhettä '%s', saatiin nil", tt.wantErr)
			}

			if err.Error() != tt.wantErr {
				t.Errorf(
					"odotettu virhe '%s', saatiin '%s'",
					tt.wantErr,
					err.Error(),
				)
			}
		})
	}
}

// --- AuthService.Register ---

func TestAuthService_Register(t *testing.T) {
	tests := []struct {
		name    string
		input   RegisterInput
		wantErr error
	}{
		{
			name: "success",
			input: RegisterInput{
				Email:    "new@example.com",
				Name:     "Uusi Käyttäjä",
				Password: "password123",
			},
		},
		{
			name: "missing email",
			input: RegisterInput{
				Name:     "Uusi Käyttäjä",
				Password: "password123",
			},
			wantErr: errors.New("sähköpostiosoite on pakollinen"),
		},
		{
			name: "missing name",
			input: RegisterInput{
				Email:    "new@example.com",
				Password: "password123",
			},
			wantErr: errors.New("nimi on pakollinen"),
		},
		{
			name: "password too short",
			input: RegisterInput{
				Email:    "new@example.com",
				Name:     "Uusi Käyttäjä",
				Password: "1234567",
			},
			wantErr: errors.New("salasanan on oltava vähintään 8 merkkiä pitkä"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestAuthService(t)

			response, err := svc.Register(
				context.Background(),
				tt.input,
			)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Register epäonnistui: %v", err)
				}

				if response == nil {
					t.Fatal("Register palautti nil-vastauksen")
				}

				if response.User == nil {
					t.Fatal("Register palautti nil-käyttäjän")
				}

				if response.Token == "" {
					t.Error("Registerin pitäisi palauttaa JWT-token")
				}

				if response.User.Email != tt.input.Email {
					t.Errorf(
						"odotettu email '%s', saatiin '%s'",
						tt.input.Email,
						response.User.Email,
					)
				}

				if response.User.Name != tt.input.Name {
					t.Errorf(
						"odotettu name '%s', saatiin '%s'",
						tt.input.Name,
						response.User.Name,
					)
				}

				if response.User.PasswordHash == "" {
					t.Error("PasswordHash ei saisi olla tyhjä")
				}

				if response.User.PasswordHash == tt.input.Password {
					t.Error("salasanaa ei saa tallentaa selväkielisenä")
				}

				if response.User.CreatedAt.IsZero() {
					t.Error("CreatedAt ei saisi olla zero value")
				}

				if response.User.UpdatedAt.IsZero() {
					t.Error("UpdatedAt ei saisi olla zero value")
				}

				// Varmistetaan, että bcrypt-hash vastaa annettua salasanaa.
				if err := bcrypt.CompareHashAndPassword(
					[]byte(response.User.PasswordHash),
					[]byte(tt.input.Password),
				); err != nil {
					t.Errorf("PasswordHash ei vastaa annettua salasanaa: %v", err)
				}

				return
			}

			if response != nil {
				t.Error("virhetilanteessa responsen pitäisi olla nil")
			}

			if err == nil {
				t.Fatalf("odotettiin virhettä '%v', saatiin nil", tt.wantErr)
			}

			if err.Error() != tt.wantErr.Error() {
				t.Errorf(
					"odotettu virhe '%s', saatiin '%s'",
					tt.wantErr,
					err,
				)
			}
		})
	}
}

func TestAuthService_Register_EmailAlreadyExists(t *testing.T) {
	svc := newTestAuthService(t)

	input := RegisterInput{
		Email:    "duplicate@example.com",
		Name:     "Ensimmäinen",
		Password: "password123",
	}

	_, err := svc.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("ensimmäinen Register epäonnistui: %v", err)
	}

	input.Name = "Toinen"

	response, err := svc.Register(context.Background(), input)

	if response != nil {
		t.Error("duplicate email -tapauksessa responsen pitäisi olla nil")
	}

	if !errors.Is(err, repository.ErrEmailAlreadyExists) {
		t.Errorf(
			"odotettiin ErrEmailAlreadyExists, saatiin %v",
			err,
		)
	}
}

// --- AuthService.Login ---

func TestAuthService_Login(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "success",
			email:    "user@example.com",
			password: "password123",
		},
		{
			name:     "wrong password",
			email:    "user@example.com",
			password: "wrong-password",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name:     "user not found",
			email:    "missing@example.com",
			password: "password123",
			wantErr:  ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestAuthService(t)

			createTestUser(t, svc.repo)

			response, err := svc.Login(
				context.Background(),
				LoginInput{
					Email:    tt.email,
					Password: tt.password,
				},
			)

			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("Login epäonnistui: %v", err)
				}

				if response == nil {
					t.Fatal("Login palautti nil-vastauksen")
				}

				if response.User == nil {
					t.Fatal("Login palautti nil-käyttäjän")
				}

				if response.Token == "" {
					t.Error("Loginin pitäisi palauttaa JWT-token")
				}

				if response.User.Email != "user@example.com" {
					t.Errorf(
						"odotettu email 'user@example.com', saatiin '%s'",
						response.User.Email,
					)
				}

				if response.User.Name != "Testi Käyttäjä" {
					t.Errorf(
						"odotettu name 'Testi Käyttäjä', saatiin '%s'",
						response.User.Name,
					)
				}

				return
			}

			if response != nil {
				t.Error("virhetilanteessa responsen pitäisi olla nil")
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf(
					"odotettiin virhettä %v, saatiin %v",
					tt.wantErr,
					err,
				)
			}
		})
	}
}

// --- AuthService.generateToken ---

func TestAuthService_generateToken(t *testing.T) {
	svc := NewAuthService(nil, "test-secret-key")

	user := &domain.User{
		ID:    "user-123",
		Email: "test@example.com",
	}

	tokenString, err := svc.generateToken(user)
	if err != nil {
		t.Fatalf("generateToken epäonnistui: %v", err)
	}

	if tokenString == "" {
		t.Fatal("generateToken palautti tyhjän tokenin")
	}

	token, err := jwt.ParseWithClaims(
		tokenString,
		&Claims{},
		func(token *jwt.Token) (interface{}, error) {
			return []byte("test-secret-key"), nil
		},
	)
	if err != nil {
		t.Fatalf("tokenin parsinta epäonnistui: %v", err)
	}

	if !token.Valid {
		t.Fatal("luodun tokenin pitäisi olla validi")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		t.Fatal("tokenin claims-tyyppi ei ole Claims")
	}

	if claims.UserID != user.ID {
		t.Errorf(
			"odotettu UserID '%s', saatiin '%s'",
			user.ID,
			claims.UserID,
		)
	}

	if claims.Email != user.Email {
		t.Errorf(
			"odotettu Email '%s', saatiin '%s'",
			user.Email,
			claims.Email,
		)
	}

	if claims.Subject != user.ID {
		t.Errorf(
			"odotettu Subject '%s', saatiin '%s'",
			user.ID,
			claims.Subject,
		)
	}

	if claims.IssuedAt == nil {
		t.Fatal("IssuedAt ei saisi olla nil")
	}

	if claims.ExpiresAt == nil {
		t.Fatal("ExpiresAt ei saisi olla nil")
	}

	// JWT NumericDate käyttää sekuntitarkkuutta.
	now := time.Now()

	if claims.IssuedAt.Time.After(now) {
		t.Error("IssuedAt ei saisi olla tulevaisuudessa")
	}

	expectedExpiration := claims.IssuedAt.Time.Add(24 * time.Hour)

	if !claims.ExpiresAt.Time.Equal(expectedExpiration) {
		t.Errorf(
			"ExpiresAt pitäisi olla tasan 24 tuntia IssuedAt:n jälkeen: issued=%v expires=%v",
			claims.IssuedAt.Time,
			claims.ExpiresAt.Time,
		)
	}
}

// --- AuthService.ValidateToken ---

func TestAuthService_ValidateToken(t *testing.T) {
	tests := []struct {
		name    string
		token   func(t *testing.T, svc *AuthService) string
		wantErr bool
	}{
		{
			name: "success",
			token: func(t *testing.T, svc *AuthService) string {
				t.Helper()

				return mustGenerateTestToken(t, svc)
			},
		},
		{
			name: "invalid token",
			token: func(t *testing.T, svc *AuthService) string {
				t.Helper()

				return "this-is-not-a-jwt"
			},
			wantErr: true,
		},
		{
			name: "wrong secret",
			token: func(t *testing.T, svc *AuthService) string {
				t.Helper()

				signer := NewAuthService(nil, "different-secret")

				return mustGenerateTestToken(t, signer)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewAuthService(nil, "test-secret-key")

			token := tt.token(t, svc)

			claims, err := svc.ValidateToken(token)

			if tt.wantErr {
				if err == nil {
					t.Fatal("odotettiin tokenin validoinnin epäonnistuvan")
				}

				return
			}

			if err != nil {
				t.Fatalf("ValidateToken epäonnistui: %v", err)
			}

			if claims == nil {
				t.Fatal("ValidateToken palautti nil-claims")
			}

			if claims.UserID != "user-123" {
				t.Errorf(
					"odotettu UserID 'user-123', saatiin '%s'",
					claims.UserID,
				)
			}

			if claims.Email != "test@example.com" {
				t.Errorf(
					"odotettu Email 'test@example.com', saatiin '%s'",
					claims.Email,
				)
			}

			if claims.Subject != "user-123" {
				t.Errorf(
					"odotettu Subject 'user-123', saatiin '%s'",
					claims.Subject,
				)
			}
		})
	}
}

func TestAuthService_ValidateToken_Expired(t *testing.T) {
	svc := NewAuthService(nil, "test-secret-key")

	claims := Claims{
		UserID: "user-123",
		Email:  "test@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Subject:   "user-123",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte("test-secret-key"))
	if err != nil {
		t.Fatalf("tokenin allekirjoitus epäonnistui: %v", err)
	}

	_, err = svc.ValidateToken(tokenString)

	if err == nil {
		t.Fatal("vanhentuneen tokenin validoinnin pitäisi epäonnistua")
	}

	if !strings.Contains(err.Error(), "virheellinen token") {
		t.Errorf(
			"odotettiin 'virheellinen token' -virhettä, saatiin %v",
			err,
		)
	}
}

// --- AuthService.GetUserByID ---

func TestAuthService_GetUserByID(t *testing.T) {
	svc := newTestAuthService(t)

	createTestUser(t, svc.repo)

	user, err := svc.GetUserByID(
		context.Background(),
		"user-id-that-does-not-exist",
	)

	if err == nil {
		t.Fatal("odotettiin käyttäjän puuttumisesta virhettä")
	}

	if user != nil {
		t.Error("käyttäjän pitäisi olla nil, kun sitä ei löydy")
	}

	if !errors.Is(err, repository.ErrUserNotFound) {
		t.Errorf(
			"odotettiin ErrUserNotFound, saatiin %v",
			err,
		)
	}
}

// --- Helpers ---

func mustGenerateTestToken(t *testing.T, svc *AuthService) string {
	t.Helper()

	user := &domain.User{
		ID:    "user-123",
		Email: "test@example.com",
	}

	token, err := svc.generateToken(user)
	if err != nil {
		t.Fatalf("testitokenin luonti epäonnistui: %v", err)
	}

	return token
}
