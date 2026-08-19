// Package handler provides Kong-facing HTTP handlers for auth-service.
package handler

import (
	"net/http"
	"strconv"
	"strings"

	"metarang/auth-service/internal/middleware"
	pb "metarang/shared/pb/auth"
)

func (h *HTTPAuthHandler) GetAuthenticatedUserWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	h.writeUserWalletResponse(w, r, userCtx.UserID)
}

// GetUserWallet handles GET /api/users/{user}/wallet
func (h *HTTPAuthHandler) GetUserWallet(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path: /api/users/{user}/wallet
	pathStr := strings.TrimPrefix(r.URL.Path, "/api/users/")
	// Remove query params if any
	if idx := strings.Index(pathStr, "?"); idx != -1 {
		pathStr = pathStr[:idx]
	}
	pathParts := strings.Split(strings.Trim(pathStr, "/"), "/")
	// Filter out empty parts
	var cleanParts []string
	for _, part := range pathParts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}
	if len(cleanParts) < 2 || cleanParts[0] == "" || cleanParts[1] != "wallet" {
		writeError(w, http.StatusBadRequest, "invalid path format: expected /api/users/{user}/wallet")
		return
	}

	userIDStr := cleanParts[0]
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	h.writeUserWalletResponse(w, r, userID)
}

func (h *HTTPAuthHandler) writeUserWalletResponse(w http.ResponseWriter, r *http.Request, userID uint64) {
	grpcReq := &pb.GetUserWalletRequest{
		UserId: userID,
	}

	resp, err := h.userClient.GetUserWallet(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	parseFloat := func(s string) float64 {
		if s == "" {
			return 0
		}
		val, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		return val
	}

	data := map[string]interface{}{
		"psc":          parseFloat(resp.Psc),
		"irr":          parseFloat(resp.Irr),
		"red":          parseFloat(resp.Red),
		"blue":         parseFloat(resp.Blue),
		"yellow":       parseFloat(resp.Yellow),
		"satisfaction": parseFloat(resp.Satisfaction),
		"effect":       resp.Effect,
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}
