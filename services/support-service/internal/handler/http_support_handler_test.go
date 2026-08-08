package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbCommon "metarang/shared/pb/common"
	pbSupport "metarang/shared/pb/support"
	authpkg "metarang/shared/pkg/auth"
	"metarang/support-service/internal/handler"
)

type mockTicketAPI struct {
	GetTicketsFunc   func(context.Context, *pbSupport.GetTicketsRequest) (*pbSupport.TicketsResponse, error)
	CreateTicketFunc func(context.Context, *pbSupport.CreateTicketRequest) (*pbSupport.TicketResponse, error)
	GetTicketFunc    func(context.Context, *pbSupport.GetTicketRequest) (*pbSupport.TicketResponse, error)
	UpdateTicketFunc func(context.Context, *pbSupport.UpdateTicketRequest) (*pbSupport.TicketResponse, error)
	AddResponseFunc  func(context.Context, *pbSupport.AddResponseRequest) (*pbSupport.TicketResponse, error)
	CloseTicketFunc  func(context.Context, *pbSupport.CloseTicketRequest) (*pbSupport.TicketResponse, error)
}

func (m *mockTicketAPI) GetTickets(ctx context.Context, req *pbSupport.GetTicketsRequest) (*pbSupport.TicketsResponse, error) {
	if m.GetTicketsFunc != nil {
		return m.GetTicketsFunc(ctx, req)
	}
	return &pbSupport.TicketsResponse{}, nil
}
func (m *mockTicketAPI) CreateTicket(ctx context.Context, req *pbSupport.CreateTicketRequest) (*pbSupport.TicketResponse, error) {
	if m.CreateTicketFunc != nil {
		return m.CreateTicketFunc(ctx, req)
	}
	return &pbSupport.TicketResponse{}, nil
}
func (m *mockTicketAPI) GetTicket(ctx context.Context, req *pbSupport.GetTicketRequest) (*pbSupport.TicketResponse, error) {
	if m.GetTicketFunc != nil {
		return m.GetTicketFunc(ctx, req)
	}
	return &pbSupport.TicketResponse{}, nil
}
func (m *mockTicketAPI) UpdateTicket(ctx context.Context, req *pbSupport.UpdateTicketRequest) (*pbSupport.TicketResponse, error) {
	if m.UpdateTicketFunc != nil {
		return m.UpdateTicketFunc(ctx, req)
	}
	return &pbSupport.TicketResponse{}, nil
}
func (m *mockTicketAPI) AddResponse(ctx context.Context, req *pbSupport.AddResponseRequest) (*pbSupport.TicketResponse, error) {
	if m.AddResponseFunc != nil {
		return m.AddResponseFunc(ctx, req)
	}
	return &pbSupport.TicketResponse{}, nil
}
func (m *mockTicketAPI) CloseTicket(ctx context.Context, req *pbSupport.CloseTicketRequest) (*pbSupport.TicketResponse, error) {
	if m.CloseTicketFunc != nil {
		return m.CloseTicketFunc(ctx, req)
	}
	return &pbSupport.TicketResponse{}, nil
}

type mockReportAPI struct{}

func (m *mockReportAPI) GetReports(context.Context, *pbSupport.GetReportsRequest) (*pbSupport.ReportsResponse, error) {
	return &pbSupport.ReportsResponse{}, nil
}
func (m *mockReportAPI) CreateReport(context.Context, *pbSupport.CreateReportRequest) (*pbSupport.ReportResponse, error) {
	return &pbSupport.ReportResponse{}, nil
}
func (m *mockReportAPI) GetReport(context.Context, *pbSupport.GetReportRequest) (*pbSupport.ReportResponse, error) {
	return &pbSupport.ReportResponse{}, nil
}

type mockNoteAPI struct{}

func (m *mockNoteAPI) GetNotes(context.Context, *pbSupport.GetNotesRequest) (*pbSupport.NotesResponse, error) {
	return &pbSupport.NotesResponse{}, nil
}
func (m *mockNoteAPI) CreateNote(context.Context, *pbSupport.CreateNoteRequest) (*pbSupport.NoteResponse, error) {
	return &pbSupport.NoteResponse{}, nil
}
func (m *mockNoteAPI) GetNote(context.Context, *pbSupport.GetNoteRequest) (*pbSupport.NoteResponse, error) {
	return &pbSupport.NoteResponse{}, nil
}
func (m *mockNoteAPI) UpdateNote(context.Context, *pbSupport.UpdateNoteRequest) (*pbSupport.NoteResponse, error) {
	return &pbSupport.NoteResponse{}, nil
}
func (m *mockNoteAPI) DeleteNote(context.Context, *pbSupport.DeleteNoteRequest) (*pbCommon.Empty, error) {
	return &pbCommon.Empty{}, nil
}

func withUser(r *http.Request, userID uint64) *http.Request {
	ctx := context.WithValue(r.Context(), authpkg.UserContextKey{}, &authpkg.UserContext{
		UserID: userID,
		Email:  "u@example.com",
		Token:  "tok",
	})
	return r.WithContext(ctx)
}

func TestHTTPListTickets(t *testing.T) {
	tickets := &mockTicketAPI{}
	tickets.GetTicketsFunc = func(_ context.Context, req *pbSupport.GetTicketsRequest) (*pbSupport.TicketsResponse, error) {
		if req.UserId != 7 {
			t.Fatalf("user_id=%d", req.UserId)
		}
		return &pbSupport.TicketsResponse{
			Tickets: []*pbSupport.TicketResponse{
				{Id: 1, Title: "Help", Content: "Please", Code: 100001, Status: 0, UpdatedAt: "1403/01/01 12:00:00"},
			},
		}, nil
	}

	h := handler.NewHTTPSupportHandler(tickets, &mockReportAPI{}, &mockNoteAPI{}, "", "http://localhost:8000")
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tickets", nil), 7)
	rr := httptest.NewRecorder()
	h.ListTickets(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data, ok := body["data"].([]interface{})
	if !ok || len(data) != 1 {
		t.Fatalf("data=%v", body["data"])
	}
}

func TestHTTPCreateTicket_JSON(t *testing.T) {
	tickets := &mockTicketAPI{}
	tickets.CreateTicketFunc = func(_ context.Context, req *pbSupport.CreateTicketRequest) (*pbSupport.TicketResponse, error) {
		if req.UserId != 7 || req.Title != "Title" || req.Content != "Body" || req.Department != "support" {
			t.Fatalf("req=%+v", req)
		}
		if req.Attachment != "" {
			t.Fatalf("unexpected attachment=%q", req.Attachment)
		}
		return &pbSupport.TicketResponse{
			Id: 9, Title: req.Title, Content: req.Content, Code: 100009, Status: 0, Department: req.Department, UpdatedAt: "1403/01/02 10:00:00",
		}, nil
	}

	h := handler.NewHTTPSupportHandler(tickets, &mockReportAPI{}, &mockNoteAPI{}, "", "http://localhost:8000")
	payload := `{"title":"Title","content":"Body","department":"support"}`
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/tickets", bytes.NewBufferString(payload)), 7)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.CreateTicket(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), `"Title"`) {
		t.Fatalf("body=%s", rr.Body.String())
	}
}

func TestHTTPGetTicket_PrefixedPath(t *testing.T) {
	tickets := &mockTicketAPI{}
	tickets.GetTicketFunc = func(_ context.Context, req *pbSupport.GetTicketRequest) (*pbSupport.TicketResponse, error) {
		if req.TicketId != 5 || req.UserId != 7 {
			t.Fatalf("req=%+v", req)
		}
		return &pbSupport.TicketResponse{Id: 5, Title: "T", Content: "C", UpdatedAt: "1403/01/01 00:00:00"}, nil
	}

	h := handler.NewHTTPSupportHandler(tickets, &mockReportAPI{}, &mockNoteAPI{}, "", "http://localhost:8000")
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/support/tickets/5", nil), 7)
	rr := httptest.NewRecorder()
	h.GetTicket(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPListTickets_Unauthenticated(t *testing.T) {
	h := handler.NewHTTPSupportHandler(&mockTicketAPI{}, &mockReportAPI{}, &mockNoteAPI{}, "", "http://localhost:8000")
	req := httptest.NewRequest(http.MethodGet, "/api/tickets", nil)
	rr := httptest.NewRecorder()
	h.ListTickets(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHTTPHandlerError_NotFound(t *testing.T) {
	tickets := &mockTicketAPI{}
	tickets.GetTicketFunc = func(context.Context, *pbSupport.GetTicketRequest) (*pbSupport.TicketResponse, error) {
		return nil, status.Error(codes.NotFound, "ticket not found")
	}
	h := handler.NewHTTPSupportHandler(tickets, &mockReportAPI{}, &mockNoteAPI{}, "", "http://localhost:8000")
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/tickets/99", nil), 1)
	rr := httptest.NewRecorder()
	h.GetTicket(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("code=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestRegisterHTTPRoutes_Health(t *testing.T) {
	h := handler.NewHTTPSupportHandler(&mockTicketAPI{}, &mockReportAPI{}, &mockNoteAPI{}, "", "http://localhost:8000")
	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux, func(next http.Handler) http.Handler { return next })

	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("code=%d", rr.Code)
	}
}
