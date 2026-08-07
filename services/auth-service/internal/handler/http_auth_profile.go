// Package handler provides Kong-facing HTTP handlers for auth-service.
package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"metarang/auth-service/internal/middleware"
	pb "metarang/shared/pb/auth"
)

// GetUser handles GET /api/user
func (h *HTTPAuthHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	grpcReq := &pb.GetUserRequest{
		UserId: userCtx.UserID,
	}

	resp, err := h.userClient.GetUser(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// UpdateProfile handles PUT/PATCH /api/user/profile
func (h *HTTPAuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	grpcReq := &pb.UpdateProfileRequest{
		UserId: userCtx.UserID,
		Name:   req.Name,
		Email:  req.Email,
		Phone:  req.Phone,
	}

	resp, err := h.userClient.UpdateProfile(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListUsers handles GET /api/users
func (h *HTTPAuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	orderBy := r.URL.Query().Get("order-by")
	pageStr := r.URL.Query().Get("page")
	page := int32(1)
	if pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			page = int32(p)
		}
	}

	grpcReq := &pb.ListUsersRequest{
		Search:  search,
		OrderBy: orderBy,
		Page:    page,
	}

	resp, err := h.userClient.ListUsers(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, buildListUsersHTTPResponse(r, resp))
}

// GetUserLevels handles GET /api/users/{user}/levels
func (h *HTTPAuthHandler) GetUserLevels(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path: /api/users/{user}/levels
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
	if len(cleanParts) < 2 || cleanParts[0] == "" || cleanParts[1] != "levels" {
		writeError(w, http.StatusBadRequest, "invalid path format: expected /api/users/{user}/levels")
		return
	}

	userIDStr := cleanParts[0]
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	grpcReq := &pb.GetUserLevelsRequest{
		UserId: userID,
	}

	resp, err := h.userClient.GetUserLevels(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	data := map[string]interface{}{}

	if resp.Data.LatestLevel != nil {
		latestLevel := map[string]interface{}{
			"id":    resp.Data.LatestLevel.Id,
			"name":  resp.Data.LatestLevel.Title,
			"score": resp.Data.LatestLevel.Score,
			"slug":  resp.Data.LatestLevel.Slug,
		}
		if resp.Data.LatestLevel.ImageUrl != "" {
			latestLevel["image"] = resp.Data.LatestLevel.ImageUrl
		}
		data["latest_level"] = latestLevel
	} else {
		data["latest_level"] = nil
	}

	previousLevels := make([]map[string]interface{}, 0, len(resp.Data.PreviousLevels))
	for _, level := range resp.Data.PreviousLevels {
		levelData := map[string]interface{}{
			"id":    level.Id,
			"name":  level.Title,
			"score": level.Score,
			"slug":  level.Slug,
		}
		if level.ImageUrl != "" {
			levelData["image"] = level.ImageUrl
		}
		previousLevels = append(previousLevels, levelData)
	}
	data["previous_levels"] = previousLevels
	data["score_percentage_to_next_level"] = resp.Data.ScorePercentageToNextLevel

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

// GetUserProfile handles GET /api/users/{user}/profile
func (h *HTTPAuthHandler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path: /api/users/{user}/profile
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
	if len(cleanParts) < 2 || cleanParts[0] == "" || cleanParts[1] != "profile" {
		writeError(w, http.StatusBadRequest, "invalid path format: expected /api/users/{user}/profile")
		return
	}

	userIDStr := cleanParts[0]
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	// Get viewer user ID from token if authenticated
	var viewerUserID uint64
	userCtx, err := middleware.GetUserFromRequest(r)
	if err == nil {
		viewerUserID = userCtx.UserID
	}

	grpcReq := &pb.GetUserProfileRequest{
		UserId:       userID,
		ViewerUserId: viewerUserID,
	}

	resp, err := h.userClient.GetUserProfile(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	data := map[string]interface{}{
		"id":             resp.Data.Id,
		"code":           resp.Data.Code,
		"profile_images": resp.Data.ProfileImages,
	}

	// Add optional fields (may be empty/null if privacy disallows)
	if resp.Data.Name != "" {
		data["name"] = resp.Data.Name
	}
	if resp.Data.RegisteredAt != "" {
		data["registered_at"] = resp.Data.RegisteredAt
	}
	if resp.Data.FollowersCount != 0 {
		data["followers_count"] = resp.Data.FollowersCount
	}
	if resp.Data.FollowingCount != 0 {
		data["following_count"] = resp.Data.FollowingCount
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

// GetUserFeaturesCount handles GET /api/users/{user}/features/count
func (h *HTTPAuthHandler) GetUserFeaturesCount(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path: /api/users/{user}/features/count
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/")
	if len(pathParts) < 3 || pathParts[0] == "" || pathParts[1] != "features" || pathParts[2] != "count" {
		writeError(w, http.StatusBadRequest, "invalid path format: expected /api/users/{user}/features/count")
		return
	}

	userIDStr := pathParts[0]
	userID, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user_id")
		return
	}

	grpcReq := &pb.GetUserFeaturesCountRequest{
		UserId: userID,
	}

	resp, err := h.userClient.GetUserFeaturesCount(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	data := map[string]interface{}{
		"maskoni_features_count":   resp.Data.MaskoniFeaturesCount,
		"tejari_features_count":    resp.Data.TejariFeaturesCount,
		"amoozeshi_features_count": resp.Data.AmoozeshiFeaturesCount,
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": data})
}

// HandleUsersRoutes handles all /api/users/{user}/... routes
func (h *HTTPAuthHandler) HandleUsersRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// Path format: /api/users/{user}/levels, /api/users/{user}/profile, etc.

	// Remove prefix to get the dynamic part
	userPath := strings.TrimPrefix(path, "/api/users/")
	// Clean up the path - remove leading/trailing slashes and query params
	userPath = strings.Trim(userPath, "/")
	if idx := strings.Index(userPath, "?"); idx != -1 {
		userPath = userPath[:idx]
	}

	if userPath == "" {
		// This should not happen as /api/users is handled above, but handle it anyway
		http.NotFound(w, r)
		return
	}

	pathParts := strings.Split(userPath, "/")
	// Filter out empty parts
	var cleanParts []string
	for _, part := range pathParts {
		if part != "" {
			cleanParts = append(cleanParts, part)
		}
	}

	if len(cleanParts) == 0 {
		http.NotFound(w, r)
		return
	}

	// First part is the user ID, rest is the endpoint
	endpoint := ""
	if len(cleanParts) > 1 {
		endpoint = cleanParts[1]
	}

	// Route to appropriate handler based on endpoint
	switch endpoint {
	case "levels":
		if r.Method == http.MethodGet {
			h.GetUserLevels(w, r)
		} else {
			http.NotFound(w, r)
		}
	case "profile":
		if r.Method == http.MethodGet {
			h.GetUserProfile(w, r)
		} else {
			http.NotFound(w, r)
		}
	case "wallet":
		if r.Method == http.MethodGet {
			h.GetUserWallet(w, r)
		} else {
			http.NotFound(w, r)
		}
	case "features":
		if len(cleanParts) >= 3 && cleanParts[2] == "count" {
			if r.Method == http.MethodGet {
				h.GetUserFeaturesCount(w, r)
			} else {
				http.NotFound(w, r)
			}
		} else {
			http.NotFound(w, r)
		}
	case "profile-limitations":
		// Authenticated route is registered as GET /api/users/{user}/profile-limitations
		http.NotFound(w, r)

	default:
		// If no endpoint specified, treat as invalid
		http.NotFound(w, r)
	}
}
