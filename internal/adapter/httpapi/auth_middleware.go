package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	authapplication "service-platform/internal/auth/application"
)

// AccessTokenVerifier validates encoded access tokens.
type AccessTokenVerifier interface {
	ParseAccessToken(encodedToken string) (authapplication.AccessTokenClaims, error)
}

// RequireAccessToken rejects requests that do not carry a valid bearer access token.
func RequireAccessToken(verifier AccessTokenVerifier, logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r)
			if !ok {
				writeError(logger, w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}

			if _, err := verifier.ParseAccessToken(token); err != nil {
				writeError(logger, w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func bearerToken(r *http.Request) (string, bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	token = strings.TrimSpace(token)
	if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
		return "", false
	}

	return token, true
}
