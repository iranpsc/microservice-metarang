package handler

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc"

	pb "metargb/shared/pb/auth"
	calendarpb "metargb/shared/pb/calendar"
	commonpb "metargb/shared/pb/common"
)

type CalendarHandler struct {
	calendarClient calendarpb.CalendarServiceClient
	authClient     pb.AuthServiceClient // For token validation
}

func NewCalendarHandler(calendarConn *grpc.ClientConn, authConn *grpc.ClientConn) *CalendarHandler {
	return &CalendarHandler{
		calendarClient: calendarpb.NewCalendarServiceClient(calendarConn),
		authClient:     pb.NewAuthServiceClient(authConn),
	}
}

// GetEvents handles GET /api/calendar
// Query params: type (event|version), search, date (Jalali), page, per_page
func (h *CalendarHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse query parameters
	eventType := r.URL.Query().Get("type")
	if eventType == "" {
		eventType = "event" // Default per API spec
	}
	search := r.URL.Query().Get("search")
	date := r.URL.Query().Get("date")

	// Parse pagination (only used when date is not provided)
	var page, perPage int32 = 1, 10
	if date == "" {
		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.ParseInt(p, 10, 32); err == nil && parsed > 0 {
				page = int32(parsed)
			}
		}
		if pp := r.URL.Query().Get("per_page"); pp != "" {
			if parsed, err := strconv.ParseInt(pp, 10, 32); err == nil && parsed > 0 {
				perPage = int32(parsed)
			}
		}
	}

	// Extract user ID from token if authenticated (optional - calendar is public)
	var userID uint64
	token := h.extractTokenFromHeader(r)
	if token != "" {
		// Try to validate token to get user ID (optional - calendar is public)
		validateReq := &pb.ValidateTokenRequest{Token: token}
		if validateResp, err := h.authClient.ValidateToken(r.Context(), validateReq); err == nil && validateResp.Valid {
			userID = validateResp.UserId
		}
	}

	// Build gRPC request
	grpcReq := &calendarpb.GetEventsRequest{
		Type:   eventType,
		Search: search,
		Date:   date,
		UserId: userID,
	}

	if date == "" {
		grpcReq.Pagination = &commonpb.PaginationRequest{
			Page:    page,
			PerPage: perPage,
		}
	}

	// Call gRPC service
	resp, err := h.calendarClient.GetEvents(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Build response matching Laravel EventResource format
	events := make([]map[string]interface{}, 0, len(resp.Events))
	for _, event := range resp.Events {
		eventMap := map[string]interface{}{
			"id":          event.Id,
			"title":       event.Title,
			"description": event.Description,
			"starts_at":   event.StartsAt,
		}

		// Conditional fields based on is_version (inferred from presence of version_title)
		if event.VersionTitle != "" {
			// Version entry
			eventMap["version_title"] = event.VersionTitle
		} else {
			// Regular event
			if event.EndsAt != "" {
				eventMap["ends_at"] = event.EndsAt
			}
			eventMap["views"] = event.Views
			eventMap["likes"] = event.Likes
			eventMap["dislikes"] = event.Dislikes
			if event.BtnName != "" {
				eventMap["btn_name"] = event.BtnName
			}
			if event.BtnLink != "" {
				eventMap["btn_link"] = event.BtnLink
			}
			eventMap["color"] = event.Color
			if event.Image != "" {
				eventMap["image"] = event.Image
			}
			if event.UserInteraction != nil {
				eventMap["user_interaction"] = map[string]bool{
					"has_liked":    event.UserInteraction.HasLiked,
					"has_disliked": event.UserInteraction.HasDisliked,
				}
			}
		}

		events = append(events, eventMap)
	}

	// Build response with pagination (only when date is not provided)
	response := map[string]interface{}{
		"data": events,
	}

	if date == "" && resp.Pagination != nil {
		response["links"] = buildPaginationLinks(r, resp.Pagination)
		response["meta"] = map[string]interface{}{
			"current_page": resp.Pagination.CurrentPage,
			"per_page":     resp.Pagination.PerPage,
			"total":        resp.Pagination.Total,
			"last_page":    resp.Pagination.LastPage,
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// GetEvent handles GET /api/calendar/{event}
func (h *CalendarHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract event ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/calendar/")
	eventID, err := strconv.ParseUint(path, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	// Extract user ID from token if authenticated
	var userID uint64
	token := extractTokenFromHeader(r)
	if token != "" {
		// Try to validate token to get user ID (optional - calendar is public)
	}

	// Build gRPC request
	grpcReq := &calendarpb.GetEventRequest{
		EventId: eventID,
		UserId:  userID,
	}

	// Call gRPC service
	resp, err := h.calendarClient.GetEvent(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Build response matching Laravel EventResource format
	eventMap := map[string]interface{}{
		"id":          resp.Id,
		"title":       resp.Title,
		"description": resp.Description,
		"starts_at":   resp.StartsAt,
	}

	// Conditional fields based on is_version (inferred from presence of version_title)
	if resp.VersionTitle != "" {
		// Version entry
		eventMap["version_title"] = resp.VersionTitle
	} else {
		// Regular event
		if resp.EndsAt != "" {
			eventMap["ends_at"] = resp.EndsAt
		}
		eventMap["views"] = resp.Views
		eventMap["likes"] = resp.Likes
		eventMap["dislikes"] = resp.Dislikes
		if resp.BtnName != "" {
			eventMap["btn_name"] = resp.BtnName
		}
		if resp.BtnLink != "" {
			eventMap["btn_link"] = resp.BtnLink
		}
		eventMap["color"] = resp.Color
		if resp.Image != "" {
			eventMap["image"] = resp.Image
		}
		if resp.UserInteraction != nil {
			eventMap["user_interaction"] = map[string]bool{
				"has_liked":    resp.UserInteraction.HasLiked,
				"has_disliked": resp.UserInteraction.HasDisliked,
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": eventMap})
}

// FilterByDateRange handles GET /api/calendar/filter
// Query params: start_date (Jalali), end_date (Jalali)
func (h *CalendarHandler) FilterByDateRange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Parse query parameters
	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		writeValidationError(w, "start_date and end_date are required")
		return
	}

	// Build gRPC request
	grpcReq := &calendarpb.FilterByDateRangeRequest{
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Call gRPC service
	resp, err := h.calendarClient.FilterByDateRange(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Build simplified response matching API spec
	events := make([]map[string]interface{}, 0, len(resp.Events))
	for _, event := range resp.Events {
		eventMap := map[string]interface{}{
			"id":        event.Id,
			"title":     event.Title,
			"starts_at": event.StartsAt,
			"ends_at":   event.EndsAt,
			"color":     event.Color,
		}
		events = append(events, eventMap)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": events})
}

// GetLatestVersion handles GET /api/calendar/latest-version
func (h *CalendarHandler) GetLatestVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Build gRPC request
	grpcReq := &calendarpb.GetLatestVersionRequest{}

	// Call gRPC service
	resp, err := h.calendarClient.GetLatestVersion(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Build response matching API spec
	response := map[string]interface{}{
		"data": map[string]interface{}{
			"version_title": resp.VersionTitle,
		},
	}

	writeJSON(w, http.StatusOK, response)
}

// AddInteraction handles POST /api/calendar/events/{event}/interact
// Requires authentication
func (h *CalendarHandler) AddInteraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Extract token from Authorization header (required for this endpoint)
	token := h.extractTokenFromHeader(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	// Extract event ID from path
	path := strings.TrimPrefix(r.URL.Path, "/api/calendar/events/")
	path = strings.TrimSuffix(path, "/interact")
	eventID, err := strconv.ParseUint(path, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	// Parse request body
	var req struct {
		Liked int32 `json:"liked"` // 1=like, 0=dislike, -1=remove
	}

	if err := decodeJSONBody(r, &req); err != nil {
		if err == io.EOF {
			writeError(w, http.StatusBadRequest, "request body is required")
		} else {
			writeError(w, http.StatusBadRequest, "invalid request body")
		}
		return
	}

	// Validate liked value
	if req.Liked < -1 || req.Liked > 1 {
		writeValidationError(w, "liked must be -1, 0, or 1")
		return
	}

	// Validate token to get user ID
	validateReq := &pb.ValidateTokenRequest{Token: token}
	validateResp, err := h.authClient.ValidateToken(r.Context(), validateReq)
	if err != nil || !validateResp.Valid {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	userID := validateResp.UserId

	// Build gRPC request
	grpcReq := &calendarpb.AddInteractionRequest{
		EventId: eventID,
		UserId:  userID,
		Liked:   req.Liked,
	}

	// Call gRPC service
	resp, err := h.calendarClient.AddInteraction(r.Context(), grpcReq)
	if err != nil {
		writeGRPCError(w, err)
		return
	}

	// Build response matching Laravel EventResource format
	eventMap := map[string]interface{}{
		"id":          resp.Id,
		"title":       resp.Title,
		"description": resp.Description,
		"starts_at":   resp.StartsAt,
	}

	// Conditional fields
	if resp.VersionTitle != "" {
		eventMap["version_title"] = resp.VersionTitle
	} else {
		if resp.EndsAt != "" {
			eventMap["ends_at"] = resp.EndsAt
		}
		eventMap["views"] = resp.Views
		eventMap["likes"] = resp.Likes
		eventMap["dislikes"] = resp.Dislikes
		if resp.BtnName != "" {
			eventMap["btn_name"] = resp.BtnName
		}
		if resp.BtnLink != "" {
			eventMap["btn_link"] = resp.BtnLink
		}
		eventMap["color"] = resp.Color
		if resp.Image != "" {
			eventMap["image"] = resp.Image
		}
		if resp.UserInteraction != nil {
			eventMap["user_interaction"] = map[string]bool{
				"has_liked":    resp.UserInteraction.HasLiked,
				"has_disliked": resp.UserInteraction.HasDisliked,
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": eventMap})
}

// Helper function to build pagination links
func buildPaginationLinks(r *http.Request, pagination *commonpb.PaginationMeta) map[string]interface{} {
	baseURL := r.URL.Scheme + "://" + r.Host + r.URL.Path
	query := r.URL.Query()

	links := map[string]interface{}{}

	// First page
	query.Set("page", "1")
	links["first"] = baseURL + "?" + query.Encode()

	// Last page
	query.Set("page", strconv.FormatInt(int64(pagination.LastPage), 10))
	links["last"] = baseURL + "?" + query.Encode()

	// Prev page
	if pagination.CurrentPage > 1 {
		query.Set("page", strconv.FormatInt(int64(pagination.CurrentPage-1), 10))
		links["prev"] = baseURL + "?" + query.Encode()
	} else {
		links["prev"] = nil
	}

	// Next page
	if pagination.CurrentPage < pagination.LastPage {
		query.Set("page", strconv.FormatInt(int64(pagination.CurrentPage+1), 10))
		links["next"] = baseURL + "?" + query.Encode()
	} else {
		links["next"] = nil
	}

	return links
}

// extractTokenFromHeader extracts Bearer token from Authorization header
func (h *CalendarHandler) extractTokenFromHeader(r *http.Request) string {
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
	if strings.HasPrefix(authHeader, "Bearer ") {
		return strings.TrimPrefix(authHeader, "Bearer ")
	}

	// Also support token as direct value
	return authHeader
}
