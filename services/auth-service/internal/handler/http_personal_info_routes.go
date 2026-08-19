package handler

import (
	"net/http"
)

// PersonalInfoRoutes handles GET and PUT/PATCH for /api/personal-info (singleton resource).
func PersonalInfoRoutes(h *HTTPAuthHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch EffectiveHTTPMethod(r) {
		case http.MethodGet:
			h.GetPersonalInfo(w, r)
		case http.MethodPut, http.MethodPatch:
			h.UpdatePersonalInfo(w, r)
		default:
			http.NotFound(w, r)
		}
	}
}
