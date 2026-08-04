package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	service "liike_app/internal/services"
)

// contextKey on yksityinen tyyppi context-avaimille törmäysten välttämiseksi.
type contextKey string

const UserIDKey contextKey = "user_id"
const UserEmailKey contextKey = "user_email"

// ErrorResponse on standardisoitu JSON-virheviesti.
type ErrorResponse struct {
	Error string `json:"error"`
}

// AuthMiddleware palauttaa HTTP-middlewaren, joka validoi JWT-tokenin.
// Onnistuessaan lisää user_id:n ja user_email:n request-kontekstiin.
func AuthMiddleware(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Luetaan Authorization-otsikko
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, "puuttuva Authorization-otsikko", http.StatusUnauthorized)
				return
			}

			// 2. Tarkistetaan "Bearer"-etulitte
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				writeJSONError(w, "virheellinen Authorization-otsikkomuoto", http.StatusUnauthorized)
				return
			}

			// 3. Validoidaan token
			claims, err := authSvc.ValidateToken(parts[1])
			if err != nil {
				writeJSONError(w, "virheellinen tai vanhentunut token", http.StatusUnauthorized)
				return
			}

			// 4. Lisätään käyttäjätiedot kontekstiin
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserEmailKey, claims.Email)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext hakee user_id:n request-kontekstista.
func GetUserIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(UserIDKey).(string)
	return id, ok
}

// writeJSONError kirjoittaa standardisoidun JSON-virhevasteen.
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(ErrorResponse{Error: message})
}
