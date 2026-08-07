// Package handler provides HTTP handlers for the gRPC gateway service.
package handler

import (
	"net/http"
	"strings"
)

// extractTokenFromHeader extracts Bearer token from Authorization header
func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		// Try cookie as fallback
		cookie, err := r.Cookie("token")
		if err == nil && cookie != nil {
			return cookie.Value
		}
		return ""
	}

	// Check for Bearer token format
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		// If no Bearer prefix, assume the whole header is the token
		return authHeader
	}

	return strings.TrimPrefix(authHeader, bearerPrefix)
}

func extractIDFromPath(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	id := strings.TrimPrefix(path, prefix)
	// Remove any trailing slashes or query params
	id = strings.TrimSuffix(id, "/")
	if idx := strings.Index(id, "?"); idx != -1 {
		id = id[:idx]
	}
	return id
}
