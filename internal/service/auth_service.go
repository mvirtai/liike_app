package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"liike_app/internal/domain"
	"liike_app/internal/repository"
)

// ErrInvalidCredentials palautetaan, kun sähköposti tai salasana on väärä.
var ErrInvalidCredentials = errors.New("virheellinen sähköpostiosoite tai salasana")

// AuthService käsittelee rekisteröinnin ja kirjautumisen.
type AuthService struct {
	repo      *repository.Repository
	jwtSecret []byte
}

// NewAuthService luo uuden AuthService-instanssin.
func NewAuthService(repo *repository.Repository, jwtSecret string) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

// RegisterInput sisältää rekisteröinnin syöttötiedot.
type RegisterInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

// LoginInput sisältää kirjautumisen syöttötiedot.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthResponse on onnistuneen auth-operaation vastaus.
type AuthResponse struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

// Register rekisteröi uuden käyttäjän.
// Validoi syötteet, hajauttaa salasanan bcrypt:llä ja tallentaa kantaan.
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResponse, error) {
	// 1. Perussyötteiden validointi
	if err := validateRegisterInput(input); err != nil {
		return nil, err
	}

	// 2. Hajautetaan salasana bcrypt:llä (cost 12)
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("virhe salasanan hajautuksessa: %w", err)
	}

	// 3. Tallennetaan käyttäjä tietokantaan
	user, err := s.repo.CreateUser(ctx, input.Email, input.Name, string(hash))
	if err != nil {
		return nil, err // sisältää ErrEmailAlreadyExists
	}

	// 4. Generoidaan JWT
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("virhe tokenin luonnissa: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// Login kirjaa käyttäjän sisään ja palauttaa JWT-tokenin.
func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResponse, error) {
	// 1. Haetaan käyttäjä sähköpostilla
	user, err := s.repo.GetUserByEmail(ctx, input.Email)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("virhe käyttäjän haussa: %w", err)
	}

	// 2. Verrataan salasanaa bcrypt-hajautukseen
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 3. Generoidaan JWT
	token, err := s.generateToken(user)
	if err != nil {
		return nil, fmt.Errorf("virhe tokenin luonnissa: %w", err)
	}

	return &AuthResponse{Token: token, User: user}, nil
}

// GetUserByID hakee käyttäjän ID:n perusteella (middleware-käyttöä varten)
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

// Claims on JWT-tokenin payload-rakenne
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// generateToken luo ja allekirjoittaa JWT-tokenin (voimassa 24h)
func (s *AuthService) generateToken(user *domain.User) (string, error) {
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// ValidateToken parsii ja validoi JWT-tokenin.
func (s *AuthService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("odottamaton allekirjoitusmetodi: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("virheellinen token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("virheellinen token")
	}

	return claims, nil
}

// validateRegisterInput tarkistaa rekisteröinnin syötteet.
func validateRegisterInput(input RegisterInput) error {
	if input.Email == "" {
		return errors.New("sähköpostiosoite on pakollinen")
	}
	if input.Name == "" {
		return errors.New("nimi on pakollinen")
	}
	if len(input.Password) < 8 {
		return errors.New("salasanan on oltava vähintään 8 merkkiä pitkä")
	}
	return nil
}
