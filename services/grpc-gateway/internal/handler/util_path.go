// Package handler provides HTTP handlers for the gRPC gateway service.
package handler

import (
	"strings"
)

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

func splitJalaliDateTime(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}

	parts := strings.Fields(value)
	if len(parts) == 1 {
		return parts[0], ""
	}

	return parts[0], parts[1]
}
