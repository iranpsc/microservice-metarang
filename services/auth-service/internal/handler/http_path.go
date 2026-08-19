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
