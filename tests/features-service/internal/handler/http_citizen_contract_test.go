package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"metarang/features-service/internal/handler"
	authpb "metarang/shared/pb/auth"
	featurespb "metarang/shared/pb/features"
	"metarang/shared/pkg/period"
)

var (
	citizenContractRef          = time.Date(2026, 5, 15, 12, 0, 0, 0, time.Local)
	citizenContractRegisteredAt = time.Date(2024, 3, 1, 0, 0, 0, 0, time.Local)
)

type mockCitizenAuthClient struct {
	authpb.CitizenServiceClient
	userInfo func(context.Context, *authpb.GetCitizenUserInfoRequest, ...grpc.CallOption) (*authpb.GetCitizenUserInfoResponse, error)
}

func (m *mockCitizenAuthClient) GetCitizenProfile(context.Context, *authpb.GetCitizenProfileRequest, ...grpc.CallOption) (*authpb.CitizenProfileResponse, error) {
	return nil, nil
}
func (m *mockCitizenAuthClient) GetCitizenReferrals(context.Context, *authpb.GetCitizenReferralsRequest, ...grpc.CallOption) (*authpb.CitizenReferralsResponse, error) {
	return nil, nil
}
func (m *mockCitizenAuthClient) GetCitizenReferralChart(context.Context, *authpb.GetCitizenReferralChartRequest, ...grpc.CallOption) (*authpb.CitizenReferralChartResponse, error) {
	return nil, nil
}
func (m *mockCitizenAuthClient) GetCitizenUserInfo(ctx context.Context, req *authpb.GetCitizenUserInfoRequest, _ ...grpc.CallOption) (*authpb.GetCitizenUserInfoResponse, error) {
	if m.userInfo != nil {
		return m.userInfo(ctx, req)
	}
	return &authpb.GetCitizenUserInfoResponse{UserId: 42}, nil
}

type mockCitizenFeaturesHTTPAPI struct {
	summary func(context.Context, *featurespb.GetCitizenFeatureSummaryRequest) (*featurespb.GetCitizenFeatureSummaryResponse, error)
	chart   func(context.Context, *featurespb.GetCitizenFeatureChartRequest) (*featurespb.GetCitizenFeatureChartResponse, error)
	list    func(context.Context, *featurespb.ListCitizenFeaturesRequest) (*featurespb.ListCitizenFeaturesResponse, error)
}

func (m *mockCitizenFeaturesHTTPAPI) GetCitizenFeatureSummary(ctx context.Context, req *featurespb.GetCitizenFeatureSummaryRequest) (*featurespb.GetCitizenFeatureSummaryResponse, error) {
	if m.summary != nil {
		return m.summary(ctx, req)
	}
	return &featurespb.GetCitizenFeatureSummaryResponse{}, nil
}
func (m *mockCitizenFeaturesHTTPAPI) GetCitizenFeatureChart(ctx context.Context, req *featurespb.GetCitizenFeatureChartRequest) (*featurespb.GetCitizenFeatureChartResponse, error) {
	if m.chart != nil {
		return m.chart(ctx, req)
	}
	return &featurespb.GetCitizenFeatureChartResponse{}, nil
}
func (m *mockCitizenFeaturesHTTPAPI) ListCitizenFeatures(ctx context.Context, req *featurespb.ListCitizenFeaturesRequest) (*featurespb.ListCitizenFeaturesResponse, error) {
	if m.list != nil {
		return m.list(ctx, req)
	}
	return &featurespb.ListCitizenFeaturesResponse{}, nil
}

type mockCitizenBuildingsHTTPAPI struct {
	summary func(context.Context, *featurespb.GetCitizenBuildingSummaryRequest) (*featurespb.GetCitizenBuildingSummaryResponse, error)
	chart   func(context.Context, *featurespb.GetCitizenBuildingChartRequest) (*featurespb.GetCitizenBuildingChartResponse, error)
	list    func(context.Context, *featurespb.ListCitizenBuildingsRequest) (*featurespb.ListCitizenBuildingsResponse, error)
}

func (m *mockCitizenBuildingsHTTPAPI) GetCitizenBuildingSummary(ctx context.Context, req *featurespb.GetCitizenBuildingSummaryRequest) (*featurespb.GetCitizenBuildingSummaryResponse, error) {
	if m.summary != nil {
		return m.summary(ctx, req)
	}
	return &featurespb.GetCitizenBuildingSummaryResponse{}, nil
}
func (m *mockCitizenBuildingsHTTPAPI) GetCitizenBuildingChart(ctx context.Context, req *featurespb.GetCitizenBuildingChartRequest) (*featurespb.GetCitizenBuildingChartResponse, error) {
	if m.chart != nil {
		return m.chart(ctx, req)
	}
	return &featurespb.GetCitizenBuildingChartResponse{}, nil
}
func (m *mockCitizenBuildingsHTTPAPI) ListCitizenBuildings(ctx context.Context, req *featurespb.ListCitizenBuildingsRequest) (*featurespb.ListCitizenBuildingsResponse, error) {
	if m.list != nil {
		return m.list(ctx, req)
	}
	return &featurespb.ListCitizenBuildingsResponse{}, nil
}

func featureChartResponseForPeriod(periodValue string) *featurespb.GetCitizenFeatureChartResponse {
	window, err := period.ResolvePeriod(periodValue, citizenContractRef, citizenContractRegisteredAt)
	if err != nil {
		panic(err)
	}
	n := len(window.Buckets)
	labels := make([]string, n)
	bought := make([]int32, n)
	sold := make([]int32, n)
	for i, bucket := range window.Buckets {
		labels[i] = bucket.Label
	}
	return &featurespb.GetCitizenFeatureChartResponse{
		Data:   &featurespb.CitizenFeatureChartData{Labels: labels, Bought: bought, Sold: sold},
		Period: periodValue,
	}
}

func buildingChartResponseForPeriod(periodValue string) *featurespb.GetCitizenBuildingChartResponse {
	window, err := period.ResolvePeriod(periodValue, citizenContractRef, citizenContractRegisteredAt)
	if err != nil {
		panic(err)
	}
	n := len(window.Buckets)
	labels := make([]string, n)
	completed := make([]int32, n)
	for i, bucket := range window.Buckets {
		labels[i] = bucket.Label
	}
	return &featurespb.GetCitizenBuildingChartResponse{
		Data:   &featurespb.CitizenBuildingChartData{Labels: labels, Completed: completed},
		Period: periodValue,
	}
}

func newCitizenHTTPHandlers(t *testing.T, features *mockCitizenFeaturesHTTPAPI, buildings *mockCitizenBuildingsHTTPAPI) (*handler.HTTPCitizenFeaturesHandler, *handler.HTTPCitizenBuildingsHandler) {
	t.Helper()
	citizen := &mockCitizenAuthClient{
		userInfo: func(_ context.Context, _ *authpb.GetCitizenUserInfoRequest, _ ...grpc.CallOption) (*authpb.GetCitizenUserInfoResponse, error) {
			return &authpb.GetCitizenUserInfoResponse{UserId: 42}, nil
		},
	}
	featuresHandler := handler.NewHTTPCitizenFeaturesHandler(features, citizen)
	buildingsHandler := handler.NewHTTPCitizenBuildingsHandler(buildings, featuresHandler)
	return featuresHandler, buildingsHandler
}

func expectedYearlyBucketCount(t *testing.T) int {
	t.Helper()
	window, err := period.ResolvePeriod("yearly", citizenContractRef, citizenContractRegisteredAt)
	require.NoError(t, err)
	return len(window.Buckets)
}

func TestHTTPCitizenFeaturesSummary_PeriodInResponse(t *testing.T) {
	for _, test := range []struct {
		name, query, wantPeriod string
	}{
		{"weekly", "period=weekly", "weekly"},
		{"monthly", "period=monthly", "monthly"},
		{"yearly", "period=yearly", "yearly"},
		{"missing defaults daily", "", "daily"},
		{"invalid defaults daily", "period=bogus", "daily"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var gotPeriod string
			features := &mockCitizenFeaturesHTTPAPI{
				summary: func(_ context.Context, req *featurespb.GetCitizenFeatureSummaryRequest) (*featurespb.GetCitizenFeatureSummaryResponse, error) {
					gotPeriod = req.Period
					return &featurespb.GetCitizenFeatureSummaryResponse{
						Data:   []*featurespb.CitizenFeatureSummaryItem{{Karbari: "t", Label: "تجاری"}},
						Period: req.Period,
					}, nil
				},
			}
			featuresHandler, _ := newCitizenHTTPHandlers(t, features, &mockCitizenBuildingsHTTPAPI{})

			target := "/api/citizen/hm-1/features/summary"
			if test.query != "" {
				target += "?" + test.query
			}
			w := httptest.NewRecorder()
			featuresHandler.Handle(w, httptest.NewRequest(http.MethodGet, target, nil), "hm-1", []string{"summary"})

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, test.wantPeriod, gotPeriod)

			var body map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, test.wantPeriod, body["period"])
			_, hasData := body["data"]
			assert.True(t, hasData)
		})
	}
}

func TestHTTPCitizenFeaturesChart_PeriodAndBucketCounts(t *testing.T) {
	yearlyBuckets := expectedYearlyBucketCount(t)

	for _, test := range []struct {
		name, query, wantPeriod string
		wantBuckets             int
	}{
		{"daily", "period=daily", "daily", 24},
		{"weekly", "period=weekly", "weekly", 4},
		{"monthly", "period=monthly", "monthly", 12},
		{"yearly", "period=yearly", "yearly", yearlyBuckets},
		{"missing defaults daily", "", "daily", 24},
	} {
		t.Run(test.name, func(t *testing.T) {
			var gotPeriod string
			features := &mockCitizenFeaturesHTTPAPI{
				chart: func(_ context.Context, req *featurespb.GetCitizenFeatureChartRequest) (*featurespb.GetCitizenFeatureChartResponse, error) {
					gotPeriod = req.Period
					return featureChartResponseForPeriod(req.Period), nil
				},
			}
			featuresHandler, _ := newCitizenHTTPHandlers(t, features, &mockCitizenBuildingsHTTPAPI{})

			target := "/api/citizen/hm-1/features/chart"
			if test.query != "" {
				target += "?" + test.query
			}
			w := httptest.NewRecorder()
			featuresHandler.Handle(w, httptest.NewRequest(http.MethodGet, target, nil), "hm-1", []string{"chart"})

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, test.wantPeriod, gotPeriod)

			var body map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, test.wantPeriod, body["period"])

			data := body["data"].(map[string]interface{})
			labels := data["labels"].([]interface{})
			bought := data["bought"].([]interface{})
			sold := data["sold"].([]interface{})
			assert.Len(t, labels, test.wantBuckets)
			assert.Len(t, bought, test.wantBuckets)
			assert.Len(t, sold, test.wantBuckets)
		})
	}
}

func TestHTTPCitizenBuildingsChart_PeriodAndBucketCounts(t *testing.T) {
	yearlyBuckets := expectedYearlyBucketCount(t)

	for _, test := range []struct {
		name, query, wantPeriod string
		wantBuckets             int
	}{
		{"daily", "period=daily", "daily", 24},
		{"weekly", "period=weekly", "weekly", 4},
		{"monthly", "period=monthly", "monthly", 12},
		{"yearly", "period=yearly", "yearly", yearlyBuckets},
	} {
		t.Run(test.name, func(t *testing.T) {
			var gotPeriod string
			buildings := &mockCitizenBuildingsHTTPAPI{
				chart: func(_ context.Context, req *featurespb.GetCitizenBuildingChartRequest) (*featurespb.GetCitizenBuildingChartResponse, error) {
					gotPeriod = req.Period
					return buildingChartResponseForPeriod(req.Period), nil
				},
			}
			_, buildingsHandler := newCitizenHTTPHandlers(t, &mockCitizenFeaturesHTTPAPI{}, buildings)

			target := "/api/citizen/hm-1/buildings/chart?" + test.query
			w := httptest.NewRecorder()
			buildingsHandler.Handle(w, httptest.NewRequest(http.MethodGet, target, nil), "hm-1", []string{"chart"})

			require.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, test.wantPeriod, gotPeriod)

			var body map[string]interface{}
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
			assert.Equal(t, test.wantPeriod, body["period"])

			data := body["data"].(map[string]interface{})
			labels := data["labels"].([]interface{})
			completed := data["completed"].([]interface{})
			assert.Len(t, labels, test.wantBuckets)
			assert.Len(t, completed, test.wantBuckets)
		})
	}
}

func TestHTTPCitizenBuildingsSummary_NoPeriodField(t *testing.T) {
	buildings := &mockCitizenBuildingsHTTPAPI{
		summary: func(_ context.Context, _ *featurespb.GetCitizenBuildingSummaryRequest) (*featurespb.GetCitizenBuildingSummaryResponse, error) {
			return &featurespb.GetCitizenBuildingSummaryResponse{
				Data: []*featurespb.CitizenBuildingSummaryItem{{Karbari: "m", Label: "مسکونی", Count: 2}},
			}, nil
		},
	}
	_, buildingsHandler := newCitizenHTTPHandlers(t, &mockCitizenFeaturesHTTPAPI{}, buildings)

	w := httptest.NewRecorder()
	buildingsHandler.Handle(w, httptest.NewRequest(http.MethodGet, "/api/citizen/hm-1/buildings/summary", nil), "hm-1", []string{"summary"})
	require.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	_, hasPeriod := body["period"]
	assert.False(t, hasPeriod)
	_, hasData := body["data"]
	assert.True(t, hasData)
}
