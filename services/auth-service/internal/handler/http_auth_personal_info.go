// Package handler provides Kong-facing HTTP handlers for auth-service.
package handler

import (
	"io"
	"net/http"

	"metarang/auth-service/internal/middleware"
	pb "metarang/shared/pb/auth"
)

// ============================================================================
// Personal Info Service Handlers
// ============================================================================

// GetPersonalInfo handles GET /api/personal-info
func (h *HTTPAuthHandler) GetPersonalInfo(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	grpcReq := &pb.GetPersonalInfoRequest{
		UserId: userCtx.UserID,
	}

	resp, err := h.personalInfoClient.GetPersonalInfo(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	// Check if personal info exists (has any data)
	if resp.Data == nil || !hasPersonalInfoData(resp.Data) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data": []interface{}{},
		})
		return
	}

	data := map[string]interface{}{}
	if resp.Data.Occupation != "" {
		data["occupation"] = resp.Data.Occupation
	}
	if resp.Data.Education != "" {
		data["education"] = resp.Data.Education
	}
	if resp.Data.Memory != "" {
		data["memory"] = resp.Data.Memory
	}
	if resp.Data.LovedCity != "" {
		data["loved_city"] = resp.Data.LovedCity
	}
	if resp.Data.LovedCountry != "" {
		data["loved_country"] = resp.Data.LovedCountry
	}
	if resp.Data.LovedLanguage != "" {
		data["loved_language"] = resp.Data.LovedLanguage
	}
	if resp.Data.ProblemSolving != "" {
		data["problem_solving"] = resp.Data.ProblemSolving
	}
	if resp.Data.Prediction != "" {
		data["prediction"] = resp.Data.Prediction
	}
	if resp.Data.About != "" {
		data["about"] = resp.Data.About
	}
	if len(resp.Data.Passions) > 0 {
		data["passions"] = resp.Data.Passions
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": data,
	})
}

// hasPersonalInfoData checks if PersonalInfoData has any non-empty values
func hasPersonalInfoData(data *pb.PersonalInfoData) bool {
	if data == nil {
		return false
	}
	if data.Occupation != "" || data.Education != "" || data.Memory != "" ||
		data.LovedCity != "" || data.LovedCountry != "" || data.LovedLanguage != "" ||
		data.ProblemSolving != "" || data.Prediction != "" || data.About != "" {
		return true
	}
	// Check if any passion is true
	if data.Passions != nil {
		for _, value := range data.Passions {
			if value {
				return true
			}
		}
	}
	return false
}

// UpdatePersonalInfo handles PUT/PATCH /api/personal-info
func (h *HTTPAuthHandler) UpdatePersonalInfo(w http.ResponseWriter, r *http.Request) {
	if m := EffectiveHTTPMethod(r); m != http.MethodPut && m != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req struct {
		Occupation     string          `json:"occupation"`
		Education      string          `json:"education"`
		Memory         string          `json:"memory"`
		LovedCity      string          `json:"loved_city"`
		LovedCountry   string          `json:"loved_country"`
		LovedLanguage  string          `json:"loved_language"`
		ProblemSolving string          `json:"problem_solving"`
		Prediction     string          `json:"prediction"`
		About          string          `json:"about"`
		Passions       map[string]bool `json:"passions"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	grpcReq := &pb.UpdatePersonalInfoRequest{
		UserId:         userCtx.UserID,
		Occupation:     req.Occupation,
		Education:      req.Education,
		Memory:         req.Memory,
		LovedCity:      req.LovedCity,
		LovedCountry:   req.LovedCountry,
		LovedLanguage:  req.LovedLanguage,
		ProblemSolving: req.ProblemSolving,
		Prediction:     req.Prediction,
		About:          req.About,
		Passions:       req.Passions,
	}

	_, err = h.personalInfoClient.UpdatePersonalInfo(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
