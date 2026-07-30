package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/calendar-service/internal/handler"
	calendarpb "metarang/shared/pb/calendar"
	commonpb "metarang/shared/pb/common"
	authpkg "metarang/shared/pkg/auth"
)

type mockCalendarAPI struct {
	GetEventsFunc         func(context.Context, *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error)
	GetEventFunc          func(context.Context, *calendarpb.GetEventRequest) (*calendarpb.EventResponse, error)
	FilterByDateRangeFunc func(context.Context, *calendarpb.FilterByDateRangeRequest) (*calendarpb.SimplifiedEventsResponse, error)
	GetLatestVersionFunc  func(context.Context, *calendarpb.GetLatestVersionRequest) (*calendarpb.LatestVersionResponse, error)
	AddInteractionFunc    func(context.Context, *calendarpb.AddInteractionRequest) (*calendarpb.EventResponse, error)
}

func (m *mockCalendarAPI) GetEvents(ctx context.Context, req *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error) {
	if m.GetEventsFunc != nil {
		return m.GetEventsFunc(ctx, req)
	}
	return &calendarpb.EventsResponse{}, nil
}

func (m *mockCalendarAPI) GetEvent(ctx context.Context, req *calendarpb.GetEventRequest) (*calendarpb.EventResponse, error) {
	if m.GetEventFunc != nil {
		return m.GetEventFunc(ctx, req)
	}
	return &calendarpb.EventResponse{}, nil
}

func (m *mockCalendarAPI) FilterByDateRange(ctx context.Context, req *calendarpb.FilterByDateRangeRequest) (*calendarpb.SimplifiedEventsResponse, error) {
	if m.FilterByDateRangeFunc != nil {
		return m.FilterByDateRangeFunc(ctx, req)
	}
	return &calendarpb.SimplifiedEventsResponse{}, nil
}

func (m *mockCalendarAPI) GetLatestVersion(ctx context.Context, req *calendarpb.GetLatestVersionRequest) (*calendarpb.LatestVersionResponse, error) {
	if m.GetLatestVersionFunc != nil {
		return m.GetLatestVersionFunc(ctx, req)
	}
	return &calendarpb.LatestVersionResponse{}, nil
}

func (m *mockCalendarAPI) AddInteraction(ctx context.Context, req *calendarpb.AddInteractionRequest) (*calendarpb.EventResponse, error) {
	if m.AddInteractionFunc != nil {
		return m.AddInteractionFunc(ctx, req)
	}
	return &calendarpb.EventResponse{}, nil
}

func TestHTTPGetEvents_VersionLaravelShape(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetEventsFunc = func(_ context.Context, req *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error) {
		if req.Type != "version" {
			t.Fatalf("type=%s", req.Type)
		}
		return &calendarpb.EventsResponse{
			Events: []*calendarpb.EventResponse{
				{
					Id:           717,
					Title:        "Next.js migration",
					Description:  "<p>changelog</p>",
					StartsAt:     "1405/02/02 00:00",
					VersionTitle: "V1.1.32",
					IsVersion:    true,
					Views:        4,
					Likes:        1,
					Dislikes:     0,
					Color:        "#ff00ff",
				},
			},
		}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)

	req := httptest.NewRequest(http.MethodGet, "/api/calendar?type=version", nil)
	w := httptest.NewRecorder()
	h.GetEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].([]interface{})
	event := data[0].(map[string]interface{})

	if event["version_title"] != "V1.1.32" {
		t.Fatalf("version_title=%v", event["version_title"])
	}
	if _, ok := event["views"]; ok {
		t.Fatal("version entries must not expose views")
	}
	if _, ok := event["likes"]; ok {
		t.Fatal("version entries must not expose likes")
	}
}

func TestHTTPGetEvent_VersionFromTitleWhenFlagMissing(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetEventFunc = func(_ context.Context, _ *calendarpb.GetEventRequest) (*calendarpb.EventResponse, error) {
		return &calendarpb.EventResponse{
			Id:           717,
			Title:        "Next.js migration",
			Description:  "<p>changelog</p>",
			StartsAt:     "1405/02/02 00:00",
			VersionTitle: "V1.1.32",
			Views:        2,
			Likes:        1,
		}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)

	req := httptest.NewRequest(http.MethodGet, "/api/calendar/717", nil)
	w := httptest.NewRecorder()
	h.GetEvent(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	event := body["data"].(map[string]interface{})

	if event["version_title"] != "V1.1.32" {
		t.Fatalf("version_title=%v", event["version_title"])
	}
	if _, ok := event["views"]; ok {
		t.Fatal("version entries must not expose views")
	}
}

func TestHTTPGetEvent_EventShape(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetEventFunc = func(_ context.Context, _ *calendarpb.GetEventRequest) (*calendarpb.EventResponse, error) {
		return &calendarpb.EventResponse{
			Id:          710,
			Title:       "Event",
			Description: "desc",
			StartsAt:    "1405/05/19 09:00",
			EndsAt:      "1405/07/01 09:00",
			Views:       4,
			Likes:       1,
			Dislikes:    0,
			Color:       "#ff00ff",
		}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)

	req := httptest.NewRequest(http.MethodGet, "/api/calendar/710", nil)
	w := httptest.NewRecorder()
	h.GetEvent(w, req)

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	event := body["data"].(map[string]interface{})

	if event["views"] != float64(4) {
		t.Fatalf("views=%v", event["views"])
	}
	if event["ends_at"] != "1405/07/01 09:00" {
		t.Fatalf("ends_at=%v", event["ends_at"])
	}
	if _, ok := event["version_title"]; ok {
		t.Fatal("event must not include version_title")
	}
}

func TestHTTPFilterByDateRange_Validation(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/filter", nil)
	w := httptest.NewRecorder()
	h.FilterByDateRange(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPRegisterHTTPRoutes_Health(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	mux := http.NewServeMux()
	passThrough := func(next http.Handler) http.Handler { return next }
	h.RegisterHTTPRoutes(mux, passThrough, passThrough)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"ok"`) {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPGetEvents_PaginationAndLinks(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetEventsFunc = func(_ context.Context, req *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error) {
		if req.Pagination == nil || req.Pagination.Page != 2 || req.Pagination.PerPage != 5 {
			t.Fatalf("pagination=%+v", req.Pagination)
		}
		return &calendarpb.EventsResponse{
			Events:     []*calendarpb.EventResponse{{Id: 1, Title: "A", StartsAt: "1405/01/01 00:00"}},
			HasMore:    true,
			Pagination: &commonpb.PaginationMeta{CurrentPage: 2, PerPage: 5},
		}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)

	t.Setenv("APP_URL", "https://api.example.com")
	req := httptest.NewRequest(http.MethodGet, "/api/calendar?page=2&per_page=5", nil)
	w := httptest.NewRecorder()
	h.GetEvents(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	meta := body["meta"].(map[string]interface{})
	if meta["current_page"] != float64(2) || meta["from"] != float64(6) || meta["to"] != float64(6) {
		t.Fatalf("meta=%v", meta)
	}
	links := body["links"].(map[string]interface{})
	if links["next"] == nil || links["prev"] == nil {
		t.Fatalf("links=%v", links)
	}
}

func TestHTTPGetEvents_WithAuthenticatedUser(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetEventsFunc = func(_ context.Context, req *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error) {
		if req.UserId != 77 {
			t.Fatalf("userId=%d", req.UserId)
		}
		return &calendarpb.EventsResponse{
			Events:     []*calendarpb.EventResponse{{Id: 1, Title: "A", StartsAt: "1405/01/01 00:00", Likes: 1, Dislikes: 0, Color: "#000"}},
			Pagination: &commonpb.PaginationMeta{CurrentPage: 1, PerPage: 10},
		}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)

	req := httptest.NewRequest(http.MethodGet, "/api/calendar", nil)
	req = req.WithContext(context.WithValue(req.Context(), authpkg.UserContextKey{}, &authpkg.UserContext{UserID: 77}))
	w := httptest.NewRecorder()
	h.GetEvents(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPGetEvents_MethodNotAllowed(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/calendar", nil)
	w := httptest.NewRecorder()
	h.GetEvents(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPGetEvents_GrpcError(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetEventsFunc = func(context.Context, *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error) {
		return nil, status.Errorf(codes.NotFound, "missing")
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodGet, "/api/calendar", nil)
	w := httptest.NewRecorder()
	h.GetEvents(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPGetEvents_WithDateSkipsPagination(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetEventsFunc = func(_ context.Context, req *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error) {
		if req.Date != "1405/01/01" || req.Pagination != nil {
			t.Fatalf("req=%+v", req)
		}
		return &calendarpb.EventsResponse{
			Events: []*calendarpb.EventResponse{{Id: 1, Title: "D", StartsAt: "1405/01/01 00:00", Color: "#fff"}},
		}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodGet, "/api/calendar?date=1405/01/01&page=9", nil)
	w := httptest.NewRecorder()
	h.GetEvents(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["meta"]; ok {
		t.Fatal("meta should be omitted for date filter")
	}
}

func TestHTTPGetEvent_InvalidID(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/not-a-number", nil)
	w := httptest.NewRecorder()
	h.GetEvent(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPGetEvent_MethodNotAllowed(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/calendar/1", nil)
	w := httptest.NewRecorder()
	h.GetEvent(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPGetEvent_ClientIPForwarded(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetEventFunc = func(ctx context.Context, req *calendarpb.GetEventRequest) (*calendarpb.EventResponse, error) {
		if req.EventId != 5 {
			t.Fatalf("eventId=%d", req.EventId)
		}
		return &calendarpb.EventResponse{Id: 5, Title: "E", StartsAt: "1405/01/01 00:00", Color: "#000"}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/5", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	w := httptest.NewRecorder()
	h.GetEvent(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPFilterByDateRange_Success(t *testing.T) {
	api := &mockCalendarAPI{}
	api.FilterByDateRangeFunc = func(_ context.Context, req *calendarpb.FilterByDateRangeRequest) (*calendarpb.SimplifiedEventsResponse, error) {
		return &calendarpb.SimplifiedEventsResponse{
			Events: []*calendarpb.SimplifiedEventResponse{
				{Id: 1, Title: "E", StartsAt: "1403/01/01", EndsAt: "1403/01/05", Color: "#abc"},
			},
		}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/filter?start_date=1403/01/01&end_date=1403/01/10", nil)
	w := httptest.NewRecorder()
	h.FilterByDateRange(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPFilterByDateRange_InvalidStartDate(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/filter?start_date=bad&end_date=1403/01/10", nil)
	w := httptest.NewRecorder()
	h.FilterByDateRange(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPFilterByDateRange_InvalidEndDate(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/filter?start_date=1403/01/01&end_date=bad", nil)
	w := httptest.NewRecorder()
	h.FilterByDateRange(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPFilterByDateRange_EndBeforeStart(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/filter?start_date=1403/01/10&end_date=1403/01/01", nil)
	w := httptest.NewRecorder()
	h.FilterByDateRange(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPFilterByDateRange_MethodNotAllowed(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/calendar/filter?start_date=1403/01/01&end_date=1403/01/10", nil)
	w := httptest.NewRecorder()
	h.FilterByDateRange(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPFilterByDateRange_GrpcError(t *testing.T) {
	api := &mockCalendarAPI{}
	api.FilterByDateRangeFunc = func(context.Context, *calendarpb.FilterByDateRangeRequest) (*calendarpb.SimplifiedEventsResponse, error) {
		return nil, status.Errorf(codes.Internal, "db down")
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/filter?start_date=1403/01/01&end_date=1403/01/10", nil)
	w := httptest.NewRecorder()
	h.FilterByDateRange(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPGetLatestVersion_Success(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetLatestVersionFunc = func(context.Context, *calendarpb.GetLatestVersionRequest) (*calendarpb.LatestVersionResponse, error) {
		return &calendarpb.LatestVersionResponse{VersionTitle: "V2.0"}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/latest-version", nil)
	w := httptest.NewRecorder()
	h.GetLatestVersion(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].(map[string]interface{})
	if data["version_title"] != "V2.0" {
		t.Fatalf("data=%v", data)
	}
}

func TestHTTPGetLatestVersion_EmptyTitle(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetLatestVersionFunc = func(context.Context, *calendarpb.GetLatestVersionRequest) (*calendarpb.LatestVersionResponse, error) {
		return &calendarpb.LatestVersionResponse{}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/latest-version", nil)
	w := httptest.NewRecorder()
	h.GetLatestVersion(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPGetLatestVersion_MethodNotAllowed(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/calendar/latest-version", nil)
	w := httptest.NewRecorder()
	h.GetLatestVersion(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPAddInteraction_Success(t *testing.T) {
	api := &mockCalendarAPI{}
	api.AddInteractionFunc = func(_ context.Context, req *calendarpb.AddInteractionRequest) (*calendarpb.EventResponse, error) {
		if req.EventId != 9 || req.UserId != 3 || req.Liked != 1 {
			t.Fatalf("req=%+v", req)
		}
		return &calendarpb.EventResponse{
			Id: 9, Title: "E", StartsAt: "1405/01/01 00:00", Likes: 2, Dislikes: 0, Color: "#000",
		}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodPost, "/api/calendar/events/9/interact", strings.NewReader(`{"liked":1}`))
	req = req.WithContext(context.WithValue(req.Context(), authpkg.UserContextKey{}, &authpkg.UserContext{UserID: 3}))
	w := httptest.NewRecorder()
	h.AddInteraction(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPAddInteraction_Unauthorized(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/calendar/events/1/interact", strings.NewReader(`{"liked":1}`))
	w := httptest.NewRecorder()
	h.AddInteraction(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPAddInteraction_MissingLikedField(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/calendar/events/1/interact", strings.NewReader(`{}`))
	req = req.WithContext(context.WithValue(req.Context(), authpkg.UserContextKey{}, &authpkg.UserContext{UserID: 1}))
	w := httptest.NewRecorder()
	h.AddInteraction(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPAddInteraction_InvalidLikedValue(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/calendar/events/1/interact", strings.NewReader(`{"liked":5}`))
	req = req.WithContext(context.WithValue(req.Context(), authpkg.UserContextKey{}, &authpkg.UserContext{UserID: 1}))
	w := httptest.NewRecorder()
	h.AddInteraction(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPAddInteraction_InvalidEventID(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/calendar/events/abc/interact", strings.NewReader(`{"liked":1}`))
	req = req.WithContext(context.WithValue(req.Context(), authpkg.UserContextKey{}, &authpkg.UserContext{UserID: 1}))
	w := httptest.NewRecorder()
	h.AddInteraction(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPAddInteraction_InvalidJSON(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/calendar/events/1/interact", strings.NewReader(`not-json`))
	req = req.WithContext(context.WithValue(req.Context(), authpkg.UserContextKey{}, &authpkg.UserContext{UserID: 1}))
	w := httptest.NewRecorder()
	h.AddInteraction(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPAddInteraction_MethodNotAllowed(t *testing.T) {
	h := handler.NewHTTPCalendarHandler(&mockCalendarAPI{})
	req := httptest.NewRequest(http.MethodGet, "/api/calendar/events/1/interact", nil)
	req = req.WithContext(context.WithValue(req.Context(), authpkg.UserContextKey{}, &authpkg.UserContext{UserID: 1}))
	w := httptest.NewRecorder()
	h.AddInteraction(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPHandlerErrorCodes(t *testing.T) {
	cases := []struct {
		name string
		code codes.Code
		want int
	}{
		{"unauthenticated", codes.Unauthenticated, http.StatusUnauthorized},
		{"invalid argument", codes.InvalidArgument, http.StatusBadRequest},
		{"permission denied", codes.PermissionDenied, http.StatusForbidden},
		{"already exists", codes.AlreadyExists, http.StatusConflict},
		{"failed precondition", codes.FailedPrecondition, http.StatusPreconditionFailed},
		{"unavailable", codes.Unavailable, http.StatusServiceUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			api := &mockCalendarAPI{}
			api.GetLatestVersionFunc = func(context.Context, *calendarpb.GetLatestVersionRequest) (*calendarpb.LatestVersionResponse, error) {
				return nil, status.Errorf(tc.code, "msg")
			}
			h := handler.NewHTTPCalendarHandler(api)
			req := httptest.NewRequest(http.MethodGet, "/api/calendar/latest-version", nil)
			w := httptest.NewRecorder()
			h.GetLatestVersion(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestHTTPGetEvents_NonGrpcError(t *testing.T) {
	api := &mockCalendarAPI{}
	api.GetEventsFunc = func(context.Context, *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error) {
		return nil, errors.New("plain error")
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodGet, "/api/calendar", nil)
	w := httptest.NewRecorder()
	h.GetEvents(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPGetEvents_PublicBaseURLFromRequest(t *testing.T) {
	_ = os.Unsetenv("APP_URL")
	api := &mockCalendarAPI{}
	api.GetEventsFunc = func(context.Context, *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error) {
		return &calendarpb.EventsResponse{
			Events:     []*calendarpb.EventResponse{{Id: 1, Title: "A", StartsAt: "1405/01/01 00:00", Color: "#000"}},
			HasMore:    false,
			Pagination: &commonpb.PaginationMeta{CurrentPage: 1, PerPage: 10},
		}, nil
	}
	h := handler.NewHTTPCalendarHandler(api)
	req := httptest.NewRequest(http.MethodGet, "/api/calendar?page=1", nil)
	req.Host = "localhost:8060"
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()
	h.GetEvents(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	meta := body["meta"].(map[string]interface{})
	if !strings.HasPrefix(meta["path"].(string), "https://") {
		t.Fatalf("path=%v", meta["path"])
	}
}
