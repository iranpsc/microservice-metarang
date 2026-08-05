package handler_test

import (
	"context"
	"errors"
	"testing"

	"metarang/features-service/internal/handler"
	"metarang/features-service/internal/models"
	pb "metarang/shared/pb/features"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockCitizenBuildingsPort struct {
	summary   func(ctx context.Context, userID uint64, allowedKarbaris []string) (*models.CitizenBuildingSummaryResult, error)
	chart     func(ctx context.Context, userID uint64, period string, allowedKarbaris []string) (*models.CitizenBuildingChartData, string, error)
	buildings func(ctx context.Context, userID uint64, allowedKarbaris []string, page int) (*models.CitizenBuildingsPage, error)
}

func (m *mockCitizenBuildingsPort) GetSummary(ctx context.Context, userID uint64, allowedKarbaris []string) (*models.CitizenBuildingSummaryResult, error) {
	return m.summary(ctx, userID, allowedKarbaris)
}

func (m *mockCitizenBuildingsPort) GetChart(ctx context.Context, userID uint64, period string, allowedKarbaris []string) (*models.CitizenBuildingChartData, string, error) {
	return m.chart(ctx, userID, period, allowedKarbaris)
}

func (m *mockCitizenBuildingsPort) GetBuildings(ctx context.Context, userID uint64, allowedKarbaris []string, page int) (*models.CitizenBuildingsPage, error) {
	return m.buildings(ctx, userID, allowedKarbaris, page)
}

func TestCitizenBuildingsHandler_Summary_MissingUserID(t *testing.T) {
	h := handler.NewCitizenBuildingsHandler(&mockCitizenBuildingsPort{})
	_, err := h.GetCitizenBuildingSummary(context.Background(), &pb.GetCitizenBuildingSummaryRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestCitizenBuildingsHandler_Summary_Success(t *testing.T) {
	m := &mockCitizenBuildingsPort{}
	m.summary = func(ctx context.Context, userID uint64, allowedKarbaris []string) (*models.CitizenBuildingSummaryResult, error) {
		assert.Equal(t, uint64(7), userID)
		assert.Equal(t, []string{"m"}, allowedKarbaris)
		return &models.CitizenBuildingSummaryResult{
			Items: []models.CitizenBuildingSummaryItem{
				{Karbari: "m", Label: "مسکونی", Count: 5},
			},
		}, nil
	}

	h := handler.NewCitizenBuildingsHandler(m)
	resp, err := h.GetCitizenBuildingSummary(context.Background(), &pb.GetCitizenBuildingSummaryRequest{
		UserId:          7,
		AllowedKarbaris: []string{"m"},
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, int32(5), resp.Data[0].Count)
}

func TestCitizenBuildingsHandler_Chart_Success(t *testing.T) {
	m := &mockCitizenBuildingsPort{}
	m.chart = func(ctx context.Context, userID uint64, period string, allowedKarbaris []string) (*models.CitizenBuildingChartData, string, error) {
		return &models.CitizenBuildingChartData{
			Labels:    []string{"a", "b"},
			Completed: []int32{1, 2},
		}, "weekly", nil
	}

	h := handler.NewCitizenBuildingsHandler(m)
	resp, err := h.GetCitizenBuildingChart(context.Background(), &pb.GetCitizenBuildingChartRequest{
		UserId:          1,
		Period:          "weekly",
		AllowedKarbaris: []string{"m"},
	})
	require.NoError(t, err)
	assert.Equal(t, "weekly", resp.Period)
	require.NotNil(t, resp.Data)
	assert.Equal(t, []int32{1, 2}, resp.Data.Completed)
	require.Equal(t, len(resp.Data.Labels), len(resp.Data.Completed))
}

func TestCitizenBuildingsHandler_Chart_DailyBucketCount(t *testing.T) {
	m := &mockCitizenBuildingsPort{}
	m.chart = func(ctx context.Context, userID uint64, period string, allowedKarbaris []string) (*models.CitizenBuildingChartData, string, error) {
		labels := make([]string, 24)
		completed := make([]int32, 24)
		return &models.CitizenBuildingChartData{Labels: labels, Completed: completed}, "daily", nil
	}
	h := handler.NewCitizenBuildingsHandler(m)
	resp, err := h.GetCitizenBuildingChart(context.Background(), &pb.GetCitizenBuildingChartRequest{
		UserId: 1,
		Period: "daily",
	})
	require.NoError(t, err)
	assert.Equal(t, "daily", resp.Period)
	require.Len(t, resp.Data.Labels, 24)
	require.Len(t, resp.Data.Completed, 24)
}

func TestCitizenBuildingsHandler_Chart_MonthlyBucketCount(t *testing.T) {
	m := &mockCitizenBuildingsPort{}
	m.chart = func(ctx context.Context, userID uint64, period string, allowedKarbaris []string) (*models.CitizenBuildingChartData, string, error) {
		labels := make([]string, 12)
		completed := make([]int32, 12)
		return &models.CitizenBuildingChartData{Labels: labels, Completed: completed}, "monthly", nil
	}
	h := handler.NewCitizenBuildingsHandler(m)
	resp, err := h.GetCitizenBuildingChart(context.Background(), &pb.GetCitizenBuildingChartRequest{
		UserId: 1,
		Period: "monthly",
	})
	require.NoError(t, err)
	assert.Equal(t, "monthly", resp.Period)
	require.Len(t, resp.Data.Labels, 12)
	require.Len(t, resp.Data.Completed, 12)
}

func TestCitizenBuildingsHandler_List_Success(t *testing.T) {
	area := 85.0
	endDate := "1403/02/22"
	m := &mockCitizenBuildingsPort{}
	m.buildings = func(ctx context.Context, userID uint64, allowedKarbaris []string, page int) (*models.CitizenBuildingsPage, error) {
		assert.Equal(t, 2, page)
		from := 11
		to := 11
		return &models.CitizenBuildingsPage{
			Items: []models.CitizenBuildingListItem{
				{
					FeaturePropertiesID: "H0-00991",
					Karbari:             "m",
					Area:                &area,
					ConstructionEndDate: &endDate,
				},
			},
			CurrentPage: 2,
			PerPage:     10,
			Total:       25,
			LastPage:    3,
			From:        &from,
			To:          &to,
		}, nil
	}

	h := handler.NewCitizenBuildingsHandler(m)
	resp, err := h.ListCitizenBuildings(context.Background(), &pb.ListCitizenBuildingsRequest{
		UserId:          7,
		AllowedKarbaris: []string{"m"},
		Page:            2,
	})
	require.NoError(t, err)
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "H0-00991", resp.Data[0].FeaturePropertiesId)
	require.NotNil(t, resp.Meta)
	assert.Equal(t, int32(25), resp.Meta.Total)
}

func TestCitizenBuildingsHandler_List_ServiceError(t *testing.T) {
	m := &mockCitizenBuildingsPort{}
	m.buildings = func(ctx context.Context, userID uint64, allowedKarbaris []string, page int) (*models.CitizenBuildingsPage, error) {
		return nil, errors.New("db down")
	}

	h := handler.NewCitizenBuildingsHandler(m)
	_, err := h.ListCitizenBuildings(context.Background(), &pb.ListCitizenBuildingsRequest{UserId: 1, Page: 1})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}
