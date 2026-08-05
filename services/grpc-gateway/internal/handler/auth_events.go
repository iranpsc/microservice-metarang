// Package handler provides HTTP handlers for the gRPC gateway service.
package handler

import (
	"io"
	"net/http"
	"strconv"

	"metarang/grpc-gateway/internal/middleware"
	pb "metarang/shared/pb/auth"
)

// ============================================================================
// User Events Service Handlers
// ============================================================================

// ListUserEvents handles GET /api/events
func (h *AuthHandler) ListUserEvents(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Get page from query parameter
	pageStr := r.URL.Query().Get("page")
	page := int32(1)
	if pageStr != "" {
		if p, err := strconv.ParseInt(pageStr, 10, 32); err == nil && p > 0 {
			page = int32(p)
		}
	}

	grpcReq := &pb.ListUserEventsRequest{
		UserId: userCtx.UserID,
		Page:   page,
	}

	resp, err := h.userEventsClient.ListUserEvents(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": resp.Data,
		"links": map[string]interface{}{
			"next": resp.Pagination.NextPageUrl,
			"prev": resp.Pagination.PrevPageUrl,
		},
		"meta": map[string]interface{}{
			"current_page": resp.Pagination.CurrentPage,
		},
	})
}

// GetUserEvent handles GET /api/events/{userEvent}
func (h *AuthHandler) GetUserEvent(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Extract event ID from path: /api/events/{userEvent}
	eventIDStr := extractIDFromPath(r.URL.Path, "/api/events/")
	if eventIDStr == "" {
		writeError(w, http.StatusBadRequest, "event_id is required")
		return
	}

	eventID, err := strconv.ParseUint(eventIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event_id")
		return
	}

	grpcReq := &pb.GetUserEventRequest{
		UserId:  userCtx.UserID,
		EventId: eventID,
	}

	resp, err := h.userEventsClient.GetUserEvent(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp.Data)
}

// ReportUserEvent handles POST /api/events/report/{userEvent}
func (h *AuthHandler) ReportUserEvent(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Extract event ID from path: /api/events/report/{userEvent}
	eventIDStr := extractIDFromPath(r.URL.Path, "/api/events/report/")
	if eventIDStr == "" {
		writeError(w, http.StatusBadRequest, "event_id is required")
		return
	}

	eventID, err := strconv.ParseUint(eventIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event_id")
		return
	}

	var req struct {
		SuspeciousCitizen string `json:"suspecious_citizen,omitempty"`
		EventDescription  string `json:"event_description"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	grpcReq := &pb.ReportUserEventRequest{
		UserId:            userCtx.UserID,
		EventId:           eventID,
		SuspeciousCitizen: req.SuspeciousCitizen,
		EventDescription:  req.EventDescription,
	}

	resp, err := h.userEventsClient.ReportUserEvent(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp.Data)
}

// SendReportResponse handles POST /api/events/report/response/{userEvent}
func (h *AuthHandler) SendReportResponse(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Extract event ID from path: /api/events/report/response/{userEvent}
	eventIDStr := extractIDFromPath(r.URL.Path, "/api/events/report/response/")
	if eventIDStr == "" {
		writeError(w, http.StatusBadRequest, "event_id is required")
		return
	}

	eventID, err := strconv.ParseUint(eventIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event_id")
		return
	}

	var req struct {
		Response string `json:"response"`
	}

	if err := decodeRequestBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	grpcReq := &pb.SendReportResponseRequest{
		UserId:   userCtx.UserID,
		EventId:  eventID,
		Response: req.Response,
	}

	resp, err := h.userEventsClient.SendReportResponse(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp.Data)
}

// CloseEventReport handles POST /api/events/report/close/{userEvent}
func (h *AuthHandler) CloseEventReport(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Extract event ID from path: /api/events/report/close/{userEvent}
	eventIDStr := extractIDFromPath(r.URL.Path, "/api/events/report/close/")
	if eventIDStr == "" {
		writeError(w, http.StatusBadRequest, "event_id is required")
		return
	}

	eventID, err := strconv.ParseUint(eventIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event_id")
		return
	}

	grpcReq := &pb.CloseEventReportRequest{
		UserId:  userCtx.UserID,
		EventId: eventID,
	}

	_, err = h.userEventsClient.CloseEventReport(r.Context(), grpcReq)
	if err != nil {
		h.writeGRPCErrorLocale(w, err)
		return
	}

	// Return 204 No Content per API specification
	w.WriteHeader(http.StatusNoContent)
}
