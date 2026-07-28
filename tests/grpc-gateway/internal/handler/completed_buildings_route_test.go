package handler_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/grpc-gateway/internal/handler"
	"metarang/grpc-gateway/internal/testutil"
	featurespb "metarang/shared/pb/features"
)

func TestHandleFeaturesRoutes_CompletedBuildingsNotTreatedAsFeatureID(t *testing.T) {
	building := &testutil.MockBuildingService{
		ListCompletedBuildingsFunc: func(ctx context.Context, req *featurespb.ListCompletedBuildingsRequest) (*featurespb.ListCompletedBuildingsResponse, error) {
			return &featurespb.ListCompletedBuildingsResponse{
				Data: []*featurespb.CompletedBuilding{
					{Id: 1, FeatureId: 10, FeaturePropertiesId: "QA-1", Karbari: "residential"},
				},
				Links: &featurespb.PaginationLinks{},
				Meta:  &featurespb.FeatureTradeHistoryPaginationMeta{CurrentPage: 1, LastPage: 1, PerPage: 10, Total: 1},
			}, nil
		},
	}
	conn, cleanup := testutil.DialFeaturesConnWithBuilding(nil, nil, building)
	t.Cleanup(cleanup)
	h := handler.NewFeaturesHandler(conn, conn, "en")

	req := httptest.NewRequest(http.MethodGet, "/api/features/buildings/completed", nil)
	w := httptest.NewRecorder()
	h.HandleFeaturesRoutes(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.NotContains(t, w.Body.String(), "invalid feature ID")

	var body map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	data, ok := body["data"].([]interface{})
	require.True(t, ok)
	require.Len(t, data, 1)
}

func TestServeMux_CompletedBuildingsRegisteredBeforeCatchAll(t *testing.T) {
	building := &testutil.MockBuildingService{
		ListCompletedBuildingsFunc: func(ctx context.Context, req *featurespb.ListCompletedBuildingsRequest) (*featurespb.ListCompletedBuildingsResponse, error) {
			return &featurespb.ListCompletedBuildingsResponse{
				Data:  []*featurespb.CompletedBuilding{},
				Links: &featurespb.PaginationLinks{},
				Meta:  &featurespb.FeatureTradeHistoryPaginationMeta{CurrentPage: 1, LastPage: 1, PerPage: 10, Total: 0},
			}, nil
		},
	}
	conn, cleanup := testutil.DialFeaturesConnWithBuilding(nil, nil, building)
	t.Cleanup(cleanup)
	h := handler.NewFeaturesHandler(conn, conn, "en")

	mux := http.NewServeMux()
	mux.Handle("GET /api/features/buildings/completed", http.HandlerFunc(h.ListCompletedBuildings))
	mux.Handle("/api/features/", http.HandlerFunc(h.HandleFeaturesRoutes))

	req := httptest.NewRequest(http.MethodGet, "/api/features/buildings/completed", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	assert.NotContains(t, w.Body.String(), "invalid feature ID")
}

func TestHandleFeaturesRoutes_GetBuildingsStillWorks(t *testing.T) {
	building := &testutil.MockBuildingService{
		GetBuildingsFunc: func(ctx context.Context, req *featurespb.GetBuildingsRequest) (*featurespb.BuildingsResponse, error) {
			assert.Equal(t, uint64(42), req.FeatureId)
			return &featurespb.BuildingsResponse{}, nil
		},
	}
	conn, cleanup := testutil.DialFeaturesConnWithBuilding(nil, nil, building)
	t.Cleanup(cleanup)
	h := handler.NewFeaturesHandler(conn, conn, "en")

	req := httptest.NewRequest(http.MethodGet, "/api/features/42/build/buildings", nil)
	w := httptest.NewRecorder()
	h.HandleFeaturesRoutes(w, req)

	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
}
