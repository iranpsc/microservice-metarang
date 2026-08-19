package handler

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"google.golang.org/grpc/metadata"

	"metarang/calendar-service/internal/middleware"
	calendarpb "metarang/shared/pb/calendar"
	commonpb "metarang/shared/pb/common"
	"metarang/shared/pkg/jalali"
	"metarang/shared/pkg/sentry"
)

// calendarAPI is the subset of calendar RPCs used by the HTTP layer.
type calendarAPI interface {
	GetEvents(context.Context, *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error)
	GetEvent(context.Context, *calendarpb.GetEventRequest) (*calendarpb.EventResponse, error)
	FilterByDateRange(context.Context, *calendarpb.FilterByDateRangeRequest) (*calendarpb.SimplifiedEventsResponse, error)
	GetLatestVersion(context.Context, *calendarpb.GetLatestVersionRequest) (*calendarpb.LatestVersionResponse, error)
	AddInteraction(context.Context, *calendarpb.AddInteractionRequest) (*calendarpb.EventResponse, error)
}

// HTTPCalendarHandler serves Kong-facing REST routes for calendar-service.
type HTTPCalendarHandler struct {
	api calendarAPI
}

// NewHTTPCalendarHandler wraps the gRPC calendar handler for local HTTP use.
func NewHTTPCalendarHandler(api calendarAPI) *HTTPCalendarHandler {
	return &HTTPCalendarHandler{api: api}
}

// RegisterHTTPRoutes registers calendar REST routes and /health.
func (h *HTTPCalendarHandler) RegisterHTTPRoutes(
	mux *http.ServeMux,
	authMiddleware func(http.Handler) http.Handler,
	optionalAuthMiddleware func(http.Handler) http.Handler,
) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.Handle("/api/calendar", optionalAuthMiddleware(http.HandlerFunc(h.GetEvents)))
	mux.Handle("/api/calendar/filter", optionalAuthMiddleware(http.HandlerFunc(h.FilterByDateRange)))
	mux.Handle("/api/calendar/latest-version", optionalAuthMiddleware(http.HandlerFunc(h.GetLatestVersion)))
	mux.Handle("/api/calendar/events/", authMiddleware(http.HandlerFunc(h.AddInteraction)))
	// Catch-all for GET /api/calendar/{id} — register after more specific paths.
	mux.Handle("/api/calendar/", optionalAuthMiddleware(http.HandlerFunc(h.GetEvent)))
}

// StartHTTPServer starts the public HTTP server (behind Kong).
func StartHTTPServer(
	httpHandler *HTTPCalendarHandler,
	port string,
	authMiddleware func(http.Handler) http.Handler,
	optionalAuthMiddleware func(http.Handler) http.Handler,
) error {
	mux := http.NewServeMux()
	httpHandler.RegisterHTTPRoutes(mux, authMiddleware, optionalAuthMiddleware)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: sentry.HTTPMiddleware(mux),
	}
	return server.ListenAndServe()
}

// GetEvents handles GET /api/calendar
func (h *HTTPCalendarHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	eventType := r.URL.Query().Get("type")
	if eventType == "" {
		eventType = "event"
	}
	search := r.URL.Query().Get("search")
	date := r.URL.Query().Get("date")

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

	var userID uint64
	if userCtx, err := middleware.GetUserFromRequest(r); err == nil {
		userID = userCtx.UserID
	}

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

	resp, err := h.api.GetEvents(r.Context(), grpcReq)
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	events := make([]map[string]interface{}, 0, len(resp.Events))
	for _, event := range resp.Events {
		events = append(events, buildCalendarEventMap(event, true, eventType))
	}

	response := map[string]interface{}{
		"data": events,
	}

	if date == "" && resp.Pagination != nil {
		itemCount := len(events)
		var from interface{}
		var to interface{}
		if itemCount > 0 {
			fromVal := int((page-1)*perPage) + 1
			from = fromVal
			to = fromVal + itemCount - 1
		}

		response["links"] = buildSimplePaginationLinks(r, page, resp.HasMore)
		response["meta"] = map[string]interface{}{
			"current_page": resp.Pagination.CurrentPage,
			"from":         from,
			"path":         requestPath(r),
			"per_page":     resp.Pagination.PerPage,
			"to":           to,
		}
	}

	writeJSON(w, http.StatusOK, response)
}

// GetEvent handles GET /api/calendar/{event}
func (h *HTTPCalendarHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/calendar/")
	eventID, err := strconv.ParseUint(path, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	var userID uint64
	if userCtx, err := middleware.GetUserFromRequest(r); err == nil {
		userID = userCtx.UserID
	}

	resp, err := h.api.GetEvent(contextWithClientIP(r), &calendarpb.GetEventRequest{
		EventId: eventID,
		UserId:  userID,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": buildCalendarEventMap(resp, true, ""),
	})
}

// FilterByDateRange handles GET /api/calendar/filter
func (h *HTTPCalendarHandler) FilterByDateRange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	startDate := r.URL.Query().Get("start_date")
	endDate := r.URL.Query().Get("end_date")

	if startDate == "" || endDate == "" {
		writeFieldValidationError(w, "start_date and end_date are required", map[string][]string{
			"start_date": {"The start date field is required."},
			"end_date":   {"The end date field is required."},
		})
		return
	}

	start, err := jalali.JalaliToCarbon(startDate)
	if err != nil {
		writeFieldValidationError(w, "The start date field is invalid.", map[string][]string{
			"start_date": {"The start date field is invalid."},
		})
		return
	}
	end, err := jalali.JalaliToCarbon(endDate)
	if err != nil {
		writeFieldValidationError(w, "The end date field is invalid.", map[string][]string{
			"end_date": {"The end date field is invalid."},
		})
		return
	}
	if end.Before(start) {
		writeFieldValidationError(w, "The end date field must be a date after or equal to start date.", map[string][]string{
			"end_date": {"The end date field must be a date after or equal to start date."},
		})
		return
	}

	resp, err := h.api.FilterByDateRange(r.Context(), &calendarpb.FilterByDateRangeRequest{
		StartDate: startDate,
		EndDate:   endDate,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	events := make([]map[string]interface{}, 0, len(resp.Events))
	for _, event := range resp.Events {
		events = append(events, map[string]interface{}{
			"id":        event.Id,
			"title":     event.Title,
			"starts_at": event.StartsAt,
			"ends_at":   event.EndsAt,
			"color":     event.Color,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"data": events})
}

// GetLatestVersion handles GET /api/calendar/latest-version
func (h *HTTPCalendarHandler) GetLatestVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	resp, err := h.api.GetLatestVersion(r.Context(), &calendarpb.GetLatestVersionRequest{})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	var versionTitle interface{}
	if resp.VersionTitle != "" {
		versionTitle = resp.VersionTitle
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"version_title": versionTitle,
		},
	})
}

// AddInteraction handles POST /api/calendar/events/{event}/interact
func (h *HTTPCalendarHandler) AddInteraction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userCtx, err := middleware.GetUserFromRequest(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/calendar/events/")
	path = strings.TrimSuffix(path, "/interact")
	eventID, err := strconv.ParseUint(path, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid event ID")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body) == 0 {
		writeFieldValidationError(w, "The liked field is required.", map[string][]string{
			"liked": {"The liked field is required."},
		})
		return
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if _, ok := raw["liked"]; !ok {
		writeFieldValidationError(w, "The liked field is required.", map[string][]string{
			"liked": {"The liked field is required."},
		})
		return
	}

	var req struct {
		Liked int32 `json:"liked"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Liked < -1 || req.Liked > 1 {
		writeFieldValidationError(w, "The liked field is invalid.", map[string][]string{
			"liked": {"The liked field must be -1, 0, or 1."},
		})
		return
	}

	resp, err := h.api.AddInteraction(contextWithClientIP(r), &calendarpb.AddInteractionRequest{
		EventId: eventID,
		UserId:  userCtx.UserID,
		Liked:   req.Liked,
	})
	if err != nil {
		writeHandlerError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": buildCalendarEventMap(resp, false, ""),
	})
}

func calendarEventIsVersion(event *calendarpb.EventResponse, requestType string) bool {
	if event == nil {
		return false
	}
	if event.IsVersion {
		return true
	}
	if event.VersionTitle != "" {
		return true
	}
	return requestType == "version"
}

func buildCalendarEventMap(event *calendarpb.EventResponse, includeViews bool, requestType string) map[string]interface{} {
	isVersion := calendarEventIsVersion(event, requestType)

	eventMap := map[string]interface{}{
		"id":          event.Id,
		"title":       event.Title,
		"description": event.Description,
		"starts_at":   event.StartsAt,
	}

	if isVersion {
		if event.VersionTitle != "" {
			eventMap["version_title"] = event.VersionTitle
		}
		return eventMap
	}

	if event.EndsAt != "" {
		eventMap["ends_at"] = event.EndsAt
	}
	if includeViews {
		eventMap["views"] = event.Views
	}
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

	return eventMap
}

func contextWithClientIP(r *http.Request) context.Context {
	ip := clientIPFromRequest(r)
	if ip == "" {
		return r.Context()
	}
	md := metadata.Pairs("x-forwarded-for", ip)
	return metadata.NewIncomingContext(r.Context(), md)
}

func clientIPFromRequest(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
