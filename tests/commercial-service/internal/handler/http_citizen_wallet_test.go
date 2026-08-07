package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/commercial-service/internal/handler"
	authpb "metarang/shared/pb/auth"
	commercialpb "metarang/shared/pb/commercial"
)

type mockCitizenUserInfoAPI struct {
	GetCitizenUserInfoFunc func(context.Context, *authpb.GetCitizenUserInfoRequest) (*authpb.GetCitizenUserInfoResponse, error)
}

func (m *mockCitizenUserInfoAPI) GetCitizenUserInfo(ctx context.Context, req *authpb.GetCitizenUserInfoRequest, _ ...grpc.CallOption) (*authpb.GetCitizenUserInfoResponse, error) {
	if m.GetCitizenUserInfoFunc != nil {
		return m.GetCitizenUserInfoFunc(ctx, req)
	}
	return &authpb.GetCitizenUserInfoResponse{
		UserId:  99,
		Privacy: map[string]int32{"psc": 1},
	}, nil
}

type mockWalletHistoryAPI struct {
	GetWalletHistorySummaryFunc func(context.Context, *commercialpb.GetWalletHistorySummaryRequest) (*commercialpb.GetWalletHistorySummaryResponse, error)
	GetWalletHistoryChartFunc   func(context.Context, *commercialpb.GetWalletHistoryChartRequest) (*commercialpb.GetWalletHistoryChartResponse, error)
}

func (m *mockWalletHistoryAPI) GetWalletHistorySummary(ctx context.Context, req *commercialpb.GetWalletHistorySummaryRequest) (*commercialpb.GetWalletHistorySummaryResponse, error) {
	if m.GetWalletHistorySummaryFunc != nil {
		return m.GetWalletHistorySummaryFunc(ctx, req)
	}
	return &commercialpb.GetWalletHistorySummaryResponse{}, nil
}

func (m *mockWalletHistoryAPI) GetWalletHistoryChart(ctx context.Context, req *commercialpb.GetWalletHistoryChartRequest) (*commercialpb.GetWalletHistoryChartResponse, error) {
	if m.GetWalletHistoryChartFunc != nil {
		return m.GetWalletHistoryChartFunc(ctx, req)
	}
	return &commercialpb.GetWalletHistoryChartResponse{}, nil
}

func newCitizenWalletHandler(citizen *mockCitizenUserInfoAPI, history *mockWalletHistoryAPI) *handler.HTTPCommercialHandler {
	return handler.NewHTTPCommercialHandler(&mockTransactionAPI{}, history, citizen)
}

func TestHTTPCitizenWalletHistorySummary_Success(t *testing.T) {
	citizen := &mockCitizenUserInfoAPI{}
	citizen.GetCitizenUserInfoFunc = func(_ context.Context, req *authpb.GetCitizenUserInfoRequest) (*authpb.GetCitizenUserInfoResponse, error) {
		if req.Code != "HM-2000000" {
			t.Fatalf("code=%q", req.Code)
		}
		return &authpb.GetCitizenUserInfoResponse{
			UserId:  42,
			Privacy: map[string]int32{"irr": 0, "psc": 1},
		}, nil
	}

	history := &mockWalletHistoryAPI{}
	history.GetWalletHistorySummaryFunc = func(_ context.Context, req *commercialpb.GetWalletHistorySummaryRequest) (*commercialpb.GetWalletHistorySummaryResponse, error) {
		if req.UserId != 42 || req.Period != "weekly" {
			t.Fatalf("req=%+v", req)
		}
		if len(req.Assets) != 2 || req.Assets[0] != "psc" || req.Assets[1] != "irr" {
			t.Fatalf("assets=%v", req.Assets)
		}
		if req.Privacy["psc"] != 1 || req.Privacy["irr"] != 0 {
			t.Fatalf("privacy=%v", req.Privacy)
		}
		return &commercialpb.GetWalletHistorySummaryResponse{
			Data: []*commercialpb.WalletAssetCard{
				{
					Asset:             "psc",
					CurrentBalance:    10,
					PeriodIncome:      2,
					PeriodSpending:    1,
					GrowthPercent:     5.5,
					Direction:         "up",
					PrivacyRestricted: false,
				},
				{
					Asset:             "irr",
					PrivacyRestricted: true,
				},
			},
		}, nil
	}

	h := newCitizenWalletHandler(citizen, history)
	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux, passThroughAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/citizen/HM-2000000/wallet/history/summary?period=weekly&assets=psc&assets=irr", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].([]interface{})
	if len(data) != 2 {
		t.Fatalf("data=%v", data)
	}
	visible := data[0].(map[string]interface{})
	if visible["asset"] != "psc" || visible["privacy_restricted"] != false {
		t.Fatalf("visible=%v", visible)
	}
	if visible["current_balance"].(float64) != 10 || visible["direction"] != "up" {
		t.Fatalf("visible fields=%v", visible)
	}
	restricted := data[1].(map[string]interface{})
	if restricted["asset"] != "irr" || restricted["privacy_restricted"] != true {
		t.Fatalf("restricted=%v", restricted)
	}
	if _, ok := restricted["current_balance"]; ok {
		t.Fatalf("restricted card must omit balances: %v", restricted)
	}
}

func TestHTTPCitizenWalletHistoryChart_Success(t *testing.T) {
	citizen := &mockCitizenUserInfoAPI{}
	history := &mockWalletHistoryAPI{}
	history.GetWalletHistoryChartFunc = func(_ context.Context, req *commercialpb.GetWalletHistoryChartRequest) (*commercialpb.GetWalletHistoryChartResponse, error) {
		if req.UserId != 99 || req.Period != "daily" {
			t.Fatalf("req=%+v", req)
		}
		if len(req.Assets) != 1 || req.Assets[0] != "psc" {
			t.Fatalf("assets=%v", req.Assets)
		}
		return &commercialpb.GetWalletHistoryChartResponse{
			Data: map[string]*commercialpb.WalletAssetChartSeries{
				"psc": {
					Income:   []*commercialpb.WalletChartPoint{{Label: "Sat", Amount: 1.5}},
					Spending: []*commercialpb.WalletChartPoint{{Label: "Sat", Amount: 0.5}},
				},
			},
		}, nil
	}

	h := newCitizenWalletHandler(citizen, history)
	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux, passThroughAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/citizen/HM-1/wallet/history/chart?period=daily&assets[]=psc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	data := body["data"].(map[string]interface{})
	series := data["psc"].(map[string]interface{})
	income := series["income"].([]interface{})
	if len(income) != 1 {
		t.Fatalf("income=%v", income)
	}
	point := income[0].(map[string]interface{})
	if point["label"] != "Sat" || point["amount"].(float64) != 1.5 {
		t.Fatalf("point=%v", point)
	}
}

func TestHTTPCitizenWalletHistory_PeriodRequired(t *testing.T) {
	h := newCitizenWalletHandler(&mockCitizenUserInfoAPI{}, &mockWalletHistoryAPI{})
	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux, passThroughAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/citizen/HM-1/wallet/history/summary", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPCitizenWalletHistory_InvalidPeriod(t *testing.T) {
	h := newCitizenWalletHandler(&mockCitizenUserInfoAPI{}, &mockWalletHistoryAPI{})
	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux, passThroughAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/citizen/HM-1/wallet/history/chart?period=hourly", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPCitizenWalletHistory_InvalidAsset(t *testing.T) {
	h := newCitizenWalletHandler(&mockCitizenUserInfoAPI{}, &mockWalletHistoryAPI{})
	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux, passThroughAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/citizen/HM-1/wallet/history/summary?period=weekly&assets=btc", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPCitizenWalletHistory_CitizenNotFound(t *testing.T) {
	citizen := &mockCitizenUserInfoAPI{}
	citizen.GetCitizenUserInfoFunc = func(context.Context, *authpb.GetCitizenUserInfoRequest) (*authpb.GetCitizenUserInfoResponse, error) {
		return nil, status.Error(codes.NotFound, "citizen not found")
	}
	h := newCitizenWalletHandler(citizen, &mockWalletHistoryAPI{})
	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux, passThroughAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/citizen/MISSING/wallet/history/summary?period=weekly", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPCitizenWalletHistory_IndexedAssets(t *testing.T) {
	history := &mockWalletHistoryAPI{}
	history.GetWalletHistorySummaryFunc = func(_ context.Context, req *commercialpb.GetWalletHistorySummaryRequest) (*commercialpb.GetWalletHistorySummaryResponse, error) {
		if len(req.Assets) != 2 || req.Assets[0] != "red" || req.Assets[1] != "blue" {
			t.Fatalf("assets=%v", req.Assets)
		}
		return &commercialpb.GetWalletHistorySummaryResponse{Data: []*commercialpb.WalletAssetCard{}}, nil
	}
	h := newCitizenWalletHandler(&mockCitizenUserInfoAPI{}, history)
	mux := http.NewServeMux()
	h.RegisterHTTPRoutes(mux, passThroughAuth)

	req := httptest.NewRequest(http.MethodGet, "/api/citizen/HM-1/wallet/history/summary?period=monthly&assets[0]=red&assets[1]=blue", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

func TestHTTPCitizenWalletHistory_MethodNotAllowed(t *testing.T) {
	h := newCitizenWalletHandler(&mockCitizenUserInfoAPI{}, &mockWalletHistoryAPI{})
	req := httptest.NewRequest(http.MethodPost, "/api/citizen/HM-1/wallet/history/summary?period=weekly", nil)
	req.SetPathValue("code", "HM-1")
	w := httptest.NewRecorder()
	h.GetCitizenWalletHistorySummary(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d", w.Code)
	}
}

func TestHTTPCitizenWalletHistory_ServiceUnavailableWithoutDeps(t *testing.T) {
	h := handler.NewHTTPCommercialHandler(&mockTransactionAPI{}, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/citizen/HM-1/wallet/history/summary?period=weekly", nil)
	req.SetPathValue("code", "HM-1")
	w := httptest.NewRecorder()
	h.GetCitizenWalletHistorySummary(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", w.Code)
	}
}

func passThroughAuth(next http.Handler) http.Handler {
	return next
}
