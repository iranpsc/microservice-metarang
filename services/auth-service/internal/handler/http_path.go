// Package handler provides Kong-facing HTTP handlers for auth-service.
package handler

import (
	"net/http"
	"strings"
)

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	// Note: X-Forwarded-For can contain multiple IPs, take the first one
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For format: "client, proxy1, proxy2"
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr (format: "IP:port")
	remoteAddr := r.RemoteAddr
	if idx := strings.LastIndex(remoteAddr, ":"); idx != -1 {
		return remoteAddr[:idx]
	}

	return remoteAddr
}

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
