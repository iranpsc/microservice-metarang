package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metarang/calendar-service/internal/handler"
	calendarpb "metarang/shared/pb/calendar"
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
