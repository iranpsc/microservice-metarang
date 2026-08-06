package handler_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"metarang/features-service/internal/handler"
	"metarang/features-service/internal/models"
	pb "metarang/shared/pb/features"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockCitizenFeaturesPort struct {
	summary func(ctx context.Context, userID uint64, period string, allowedKarbaris []string, reference time.Time) (*models.CitizenFeatureSummaryResult, error)
	chart   func(ctx context.Context, userID uint64, period string, allowedKarbaris []string, reference time.Time) (*models.CitizenFeatureChartResult, error)
	list    func(ctx context.Context, userID uint64, allowedKarbaris []string, search string, page, perPage int) (*models.CitizenFeaturesPage, error)
}

func (m *mockCitizenFeaturesPort) GetSummary(ctx context.Context, userID uint64, period string, allowedKarbaris []string, reference time.Time) (*models.CitizenFeatureSummaryResult, error) {
	return m.summary(ctx, userID, period, allowedKarbaris, reference)
}

func (m *mockCitizenFeaturesPort) GetChart(ctx context.Context, userID uint64, period string, allowedKarbaris []string, reference time.Time) (*models.CitizenFeatureChartResult, error) {
	return m.chart(ctx, userID, period, allowedKarbaris, reference)
}

func (m *mockCitizenFeaturesPort) GetFeatures(ctx context.Context, userID uint64, allowedKarbaris []string, search string, page, perPage int) (*models.CitizenFeaturesPage, error) {
	return m.list(ctx, userID, allowedKarbaris, search, page, perPage)
}

func TestCitizenFeaturesHandler_Summary_MissingUserID(t *testing.T) {
	h := handler.NewCitizenFeaturesHandler(&mockCitizenFeaturesPort{})
	_, err := h.GetCitizenFeatureSummary(context.Background(), &pb.GetCitizenFeatureSummaryRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestCitizenFeaturesHandler_Summary_Success(t *testing.T) {
	m := &mockCitizenFeaturesPort{}
	m.summary = func(ctx context.Context, userID uint64, period string, allowedKarbaris []string, reference time.Time) (*models.CitizenFeatureSummaryResult, error) {
		assert.Equal(t, uint64(7), userID)
		assert.Equal(t, "weekly", period)
		assert.Equal(t, []string{"t"}, allowedKarbaris)
		return &models.CitizenFeatureSummaryResult{
			Period: "weekly",
			Items: []models.CitizenFeatureSummaryItem{
				{Karbari: "t", Label: "تجاری", CurrentCount: 3, BoughtCount: 2, SoldCount: 1},
			},
		}, nil
	}

	h := handler.NewCitizenFeaturesHandler(m)
	resp, err := h.GetCitizenFeatureSummary(context.Background(), &pb.GetCitizenFeatureSummaryRequest{
		UserId:          7,
		Period:          "weekly",
		AllowedKarbaris: []string{"t"},
	})
	require.NoError(t, err)
	assert.Equal(t, "weekly", resp.Period)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, int32(3), resp.Data[0].CurrentCount)
}

func TestCitizenFeaturesHandler_Summary_InvalidPeriodNormalizedByService(t *testing.T) {
	m := &mockCitizenFeaturesPort{}
	m.summary = func(ctx context.Context, userID uint64, period string, allowedKarbaris []string, reference time.Time) (*models.CitizenFeatureSummaryResult, error) {
		assert.Equal(t, "bogus", period) // handler passes through; service normalizes
		return &models.CitizenFeatureSummaryResult{Period: "daily", Items: nil}, nil
	}
	h := handler.NewCitizenFeaturesHandler(m)
	resp, err := h.GetCitizenFeatureSummary(context.Background(), &pb.GetCitizenFeatureSummaryRequest{
		UserId: 1,
		Period: "bogus",
	})
	require.NoError(t, err)
	assert.Equal(t, "daily", resp.Period)
}

func TestCitizenFeaturesHandler_Chart_MissingUserID(t *testing.T) {
	h := handler.NewCitizenFeaturesHandler(&mockCitizenFeaturesPort{})
	_, err := h.GetCitizenFeatureChart(context.Background(), &pb.GetCitizenFeatureChartRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestCitizenFeaturesHandler_Chart_Success(t *testing.T) {
	m := &mockCitizenFeaturesPort{}
	m.chart = func(ctx context.Context, userID uint64, period string, allowedKarbaris []string, reference time.Time) (*models.CitizenFeatureChartResult, error) {
		return &models.CitizenFeatureChartResult{
			Bought: []models.CitizenChartPoint{
				{Karbari: "t", Label: "a", Amount: 1},
				{Karbari: "t", Label: "b", Amount: 0},
			},
			Sold: []models.CitizenChartPoint{
				{Karbari: "t", Label: "a", Amount: 0},
				{Karbari: "t", Label: "b", Amount: 2},
			},
			Period: "daily",
		}, nil
	}
	h := handler.NewCitizenFeaturesHandler(m)
	resp, err := h.GetCitizenFeatureChart(context.Background(), &pb.GetCitizenFeatureChartRequest{
		UserId: 1,
		Period: "daily",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.Data)
	assert.Equal(t, "daily", resp.Period)
	require.Len(t, resp.Data.Bought, 2)
	require.Len(t, resp.Data.Sold, 2)
	assert.Equal(t, "t", resp.Data.Bought[0].Karbari)
	assert.Equal(t, 1.0, resp.Data.Bought[0].Amount)
	assert.Equal(t, 2.0, resp.Data.Sold[1].Amount)
}

func TestCitizenFeaturesHandler_Chart_WeeklyBucketCount(t *testing.T) {
	m := &mockCitizenFeaturesPort{}
	m.chart = func(ctx context.Context, userID uint64, period string, allowedKarbaris []string, reference time.Time) (*models.CitizenFeatureChartResult, error) {
		bought := make([]models.CitizenChartPoint, 4)
		sold := make([]models.CitizenChartPoint, 4)
		for i := range bought {
			bought[i] = models.CitizenChartPoint{Label: "bucket"}
			sold[i] = models.CitizenChartPoint{Label: "bucket"}
		}
		return &models.CitizenFeatureChartResult{Bought: bought, Sold: sold, Period: "weekly"}, nil
	}
	h := handler.NewCitizenFeaturesHandler(m)
	resp, err := h.GetCitizenFeatureChart(context.Background(), &pb.GetCitizenFeatureChartRequest{
		UserId: 1,
		Period: "weekly",
	})
	require.NoError(t, err)
	assert.Equal(t, "weekly", resp.Period)
	require.Len(t, resp.Data.Bought, 4)
	require.Len(t, resp.Data.Sold, 4)
}

func TestCitizenFeaturesHandler_Chart_DailyBucketCount(t *testing.T) {
	m := &mockCitizenFeaturesPort{}
	m.chart = func(ctx context.Context, userID uint64, period string, allowedKarbaris []string, reference time.Time) (*models.CitizenFeatureChartResult, error) {
		bought := make([]models.CitizenChartPoint, 24)
		sold := make([]models.CitizenChartPoint, 24)
		for i := range bought {
			bought[i] = models.CitizenChartPoint{Label: "bucket"}
			sold[i] = models.CitizenChartPoint{Label: "bucket"}
		}
		return &models.CitizenFeatureChartResult{Bought: bought, Sold: sold, Period: "daily"}, nil
	}
	h := handler.NewCitizenFeaturesHandler(m)
	resp, err := h.GetCitizenFeatureChart(context.Background(), &pb.GetCitizenFeatureChartRequest{
		UserId: 1,
		Period: "daily",
	})
	require.NoError(t, err)
	require.Len(t, resp.Data.Bought, 24)
	require.Len(t, resp.Data.Sold, 24)
}

func TestCitizenFeaturesHandler_List_MissingUserID(t *testing.T) {
	h := handler.NewCitizenFeaturesHandler(&mockCitizenFeaturesPort{})
	_, err := h.ListCitizenFeatures(context.Background(), &pb.ListCitizenFeaturesRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestCitizenFeaturesHandler_List_DefaultPerPage(t *testing.T) {
	var gotPage, gotPerPage int
	m := &mockCitizenFeaturesPort{}
	m.list = func(ctx context.Context, userID uint64, allowedKarbaris []string, search string, page, perPage int) (*models.CitizenFeaturesPage, error) {
		gotPage = page
		gotPerPage = perPage
		from, to := 1, 1
		center := &models.CitizenFeatureCenter{X: 1.5, Y: 2.5}
		return &models.CitizenFeaturesPage{
			Items: []models.CitizenFeatureListItem{
				{
					ID: 1, VodID: "TO111-1", Address: "VOD", Area: 2956, Density: 3,
					Karbari: "t", OwnerCode: "HM-2000000", PricePSC: "0", PriceIRR: "0",
					Center: center, Label: "label",
					Images: []models.CitizenFeatureImage{{ID: 9, URL: "http://img"}},
				},
			},
			MapMarkers: []models.CitizenFeatureMapMarker{
				{ID: 1, Center: center, Karbari: "t"},
			},
			CurrentPage: 1,
			PerPage:     15,
			Total:       1,
			LastPage:    1,
			From:        &from,
			To:          &to,
			Path:        "/api/citizen/HM-1/features",
		}, nil
	}

	h := handler.NewCitizenFeaturesHandler(m)
	resp, err := h.ListCitizenFeatures(context.Background(), &pb.ListCitizenFeaturesRequest{
		UserId: 1,
		Page:   0,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, gotPage)
	assert.Equal(t, 15, gotPerPage)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "TO111-1", resp.Data[0].VodId)
	require.NotNil(t, resp.Data[0].Center)
	assert.Equal(t, 1.5, resp.Data[0].Center.X)
	require.Len(t, resp.MapMarkers, 1)
	assert.Equal(t, "/api/citizen/HM-1/features?page=1", resp.Links.First)
	assert.Equal(t, int32(1), resp.Meta.Total)
}

func TestCitizenFeaturesHandler_List_ServiceError(t *testing.T) {
	m := &mockCitizenFeaturesPort{}
	m.list = func(ctx context.Context, userID uint64, allowedKarbaris []string, search string, page, perPage int) (*models.CitizenFeaturesPage, error) {
		return nil, errors.New("db down")
	}
	h := handler.NewCitizenFeaturesHandler(m)
	_, err := h.ListCitizenFeatures(context.Background(), &pb.ListCitizenFeaturesRequest{UserId: 1})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}
