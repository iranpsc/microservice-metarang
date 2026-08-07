// Package middleware provides HTTP authentication for auth-service.
package middleware

import (
	"context"
	"net/http"
	"strings"

	authpkg "metarang/shared/pkg/auth"
)

// AuthMiddleware validates Bearer/cookie tokens via the local token validator.
func AuthMiddleware(validator authpkg.TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if validator == nil {
				writeError(w, http.StatusUnauthorized, "Unauthenticated")
				return
			}
			token := extractTokenFromHeader(r)
			if token == "" {
				writeError(w, http.StatusUnauthorized, "Unauthenticated")
				return
			}
			userCtx, err := validator.ValidateToken(r.Context(), token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "Unauthenticated")
				return
			}
			ctx := context.WithValue(r.Context(), authpkg.UserContextKey{}, userCtx)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware validates a token when present; invalid/missing tokens are ignored.
func OptionalAuthMiddleware(validator authpkg.TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if validator != nil {
				token := extractTokenFromHeader(r)
				if token != "" {
					if userCtx, err := validator.ValidateToken(r.Context(), token); err == nil {
						ctx := context.WithValue(r.Context(), authpkg.UserContextKey{}, userCtx)
						r = r.WithContext(ctx)
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GuestMiddleware rejects requests that carry a valid authentication token.
func GuestMiddleware(validator authpkg.TokenValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if validator != nil {
				token := extractTokenFromHeader(r)
				if token != "" {
					if _, err := validator.ValidateToken(r.Context(), token); err == nil {
						writeError(w, http.StatusForbidden, "Forbidden")
						return
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromRequest retrieves user context set by auth middleware.
func GetUserFromRequest(r *http.Request) (*authpkg.UserContext, error) {
	return authpkg.GetUserFromContext(r.Context())
}

func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		if cookie, err := r.Cookie("token"); err == nil && cookie != nil {
			return cookie.Value
		}
		return ""
	}
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return authHeader
	}
	return strings.TrimPrefix(authHeader, bearerPrefix)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
