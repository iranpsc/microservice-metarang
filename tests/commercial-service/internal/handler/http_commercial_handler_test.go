package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"metarang/commercial-service/internal/handler"
	commercialpb "metarang/shared/pb/commercial"
	authpkg "metarang/shared/pkg/auth"
)

type mockTransactionAPI struct {
	ListTransactionsFunc     func(context.Context, *commercialpb.ListTransactionsRequest) (*commercialpb.ListTransactionsResponse, error)
	GetLatestTransactionFunc func(context.Context, *commercialpb.GetLatestTransactionRequest) (*commercialpb.LatestTransactionResponse, error)
}

func (m *mockTransactionAPI) ListTransactions(ctx context.Context, req *commercialpb.ListTransactionsRequest) (*commercialpb.ListTransactionsResponse, error) {
	if m.ListTransactionsFunc != nil {
		return m.ListTransactionsFunc(ctx, req)
	}
	return &commercialpb.ListTransactionsResponse{}, nil
}

func (m *mockTransactionAPI) GetLatestTransaction(ctx context.Context, req *commercialpb.GetLatestTransactionRequest) (*commercialpb.LatestTransactionResponse, error) {
	if m.GetLatestTransactionFunc != nil {
		return m.GetLatestTransactionFunc(ctx, req)
	}
	return &commercialpb.LatestTransactionResponse{}, nil
}

func withUser(r *http.Request, userID uint64) *http.Request {
	userCtx := &authpkg.UserContext{UserID: userID, Email: "u@example.com", Token: "tok"}
	return r.WithContext(context.WithValue(r.Context(), authpkg.UserContextKey{}, userCtx))
}

func TestHTTPListTransactions_Unauthorized(t *testing.T) {
	h := handler.NewHTTPCommercialHandler(&mockTransactionAPI{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/user/transactions", nil)
	w := httptest.NewRecorder()
	h.ListTransactions(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPListTransactions_MethodNotAllowed(t *testing.T) {
	h := handler.NewHTTPCommercialHandler(&mockTransactionAPI{}, nil, nil)
	req := withUser(httptest.NewRequest(http.MethodPost, "/api/user/transactions", nil), 1)
	w := httptest.NewRecorder()
	h.ListTransactions(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPListTransactions_Success(t *testing.T) {
	_ = os.Setenv("APP_URL", "http://localhost:8000")
	t.Cleanup(func() { _ = os.Unsetenv("APP_URL") })

	api := &mockTransactionAPI{}
	api.ListTransactionsFunc = func(_ context.Context, req *commercialpb.ListTransactionsRequest) (*commercialpb.ListTransactionsResponse, error) {
		if req.UserId != 42 || req.Page != 2 || req.PerPage != 10 {
			t.Fatalf("req=%+v", req)
		}
		if req.Search != "psc" || req.Action != "deposit" || req.Asset != "psc" || req.Type != "order" {
			t.Fatalf("filters=%+v", req)
		}
		if len(req.Status) != 2 || req.Status[0] != 1 || req.Status[1] != 2 {
			t.Fatalf("status=%v", req.Status)
		}
		return &commercialpb.ListTransactionsResponse{
			Transactions: []*commercialpb.TransactionResource{
				{Id: "TR-1", Type: "order", Asset: "psc", Amount: 10.5, Action: "deposit", Status: 1, Date: "1404/01/01", Time: "12:00:00"},
			},
			CurrentPage:  2,
			HasMorePages: true,
		}, nil
	}
	h := handler.NewHTTPCommercialHandler(api, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/user/transactions?page=2&per_page=10&search=psc&action=deposit&asset=psc&type=order&status=1,2", nil)
	req = withUser(req, 42)
	w := httptest.NewRecorder()
	h.ListTransactions(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("data=%v", data)
	}
	tx := data[0].(map[string]interface{})
	if tx["id"] != "TR-1" || tx["asset"] != "psc" {
		t.Fatalf("tx=%v", tx)
	}
	meta := body["meta"].(map[string]interface{})
	if meta["current_page"].(float64) != 2 || meta["per_page"].(float64) != 10 {
		t.Fatalf("meta=%v", meta)
	}
	if meta["path"] != "http://localhost:8000/api/user/transactions" {
		t.Fatalf("path=%v", meta["path"])
	}
	links := body["links"].(map[string]interface{})
	if links["prev"] == nil || links["next"] == nil {
		t.Fatalf("links=%v", links)
	}
}

func TestHTTPListTransactions_StatusIndexedQuery(t *testing.T) {
	api := &mockTransactionAPI{}
	api.ListTransactionsFunc = func(_ context.Context, req *commercialpb.ListTransactionsRequest) (*commercialpb.ListTransactionsResponse, error) {
		if len(req.Status) != 2 || req.Status[0] != 3 || req.Status[1] != 4 {
			t.Fatalf("status=%v", req.Status)
		}
		return &commercialpb.ListTransactionsResponse{}, nil
	}
	h := handler.NewHTTPCommercialHandler(api, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/user/transactions?status[0]=3&status[1]=4", nil)
	req = withUser(req, 1)
	w := httptest.NewRecorder()
	h.ListTransactions(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPListTransactions_GRPCError(t *testing.T) {
	api := &mockTransactionAPI{}
	api.ListTransactionsFunc = func(context.Context, *commercialpb.ListTransactionsRequest) (*commercialpb.ListTransactionsResponse, error) {
		return nil, status.Error(codes.Internal, "boom")
	}
	h := handler.NewHTTPCommercialHandler(api, nil, nil)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/user/transactions", nil), 1)
	w := httptest.NewRecorder()
	h.ListTransactions(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPGetLatestTransaction_Unauthorized(t *testing.T) {
	h := handler.NewHTTPCommercialHandler(&mockTransactionAPI{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/user/transactions/latest", nil)
	w := httptest.NewRecorder()
	h.GetLatestTransaction(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPGetLatestTransaction_Success(t *testing.T) {
	createdAt := time.Date(2025, 3, 21, 10, 30, 0, 0, time.UTC)
	api := &mockTransactionAPI{}
	api.GetLatestTransactionFunc = func(_ context.Context, req *commercialpb.GetLatestTransactionRequest) (*commercialpb.LatestTransactionResponse, error) {
		if req.UserId != 7 {
			t.Fatalf("user=%d", req.UserId)
		}
		return &commercialpb.LatestTransactionResponse{
			LatestTransaction: &commercialpb.Transaction{
				Id:        "TR-9",
				Amount:    25,
				Status:    1,
				Asset:     "irr",
				CreatedAt: timestamppb.New(createdAt),
			},
			LatestPayment: &commercialpb.Payment{
				RefId:     12345,
				CreatedAt: timestamppb.New(createdAt),
			},
			LatestOrder: &commercialpb.Order{
				Asset:  "psc",
				Amount: 3,
			},
		}, nil
	}
	h := handler.NewHTTPCommercialHandler(api, nil, nil)

	req := withUser(httptest.NewRequest(http.MethodGet, "/api/user/transactions/latest", nil), 7)
	w := httptest.NewRecorder()
	h.GetLatestTransaction(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].(map[string]interface{})
	if data["id"] != "TR-9" || data["asset"] != "irr" || data["amount"].(float64) != 25 {
		t.Fatalf("data=%v", data)
	}
	if data["date"] == "" || data["time"] == "" {
		t.Fatalf("expected jalali date/time, got %v", data)
	}
	payment := data["payment_info"].(map[string]interface{})
	if payment["ref_id"].(float64) != 12345 {
		t.Fatalf("payment=%v", payment)
	}
	if data["product"] != "psc" || data["count"].(float64) != 3 {
		t.Fatalf("order fields=%v", data)
	}
}

func TestHTTPGetLatestTransaction_Empty(t *testing.T) {
	h := handler.NewHTTPCommercialHandler(&mockTransactionAPI{}, nil, nil)
	req := withUser(httptest.NewRequest(http.MethodGet, "/api/user/transactions/latest", nil), 1)
	w := httptest.NewRecorder()
	h.GetLatestTransaction(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].(map[string]interface{})
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %v", data)
	}
}

func TestRegisterHTTPRoutes_HealthAndAuth(t *testing.T) {
	api := &mockTransactionAPI{}
	h := handler.NewHTTPCommercialHandler(api, nil, nil)
	mux := http.NewServeMux()
	authMW := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeUnauthorized := true
			if r.Header.Get("Authorization") != "" {
				writeUnauthorized = false
			}
			if writeUnauthorized {
				http.Error(w, `{"error":"Unauthenticated"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, withUser(r, 1))
		})
	}
	h.RegisterHTTPRoutes(mux, authMW)

	healthReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	healthW := httptest.NewRecorder()
	mux.ServeHTTP(healthW, healthReq)
	if healthW.Code != http.StatusOK {
		t.Fatalf("health status=%d", healthW.Code)
	}

	unauthReq := httptest.NewRequest(http.MethodGet, "/api/user/transactions", nil)
	unauthW := httptest.NewRecorder()
	mux.ServeHTTP(unauthW, unauthReq)
	if unauthW.Code != http.StatusUnauthorized {
		t.Fatalf("unauth status=%d", unauthW.Code)
	}
}
