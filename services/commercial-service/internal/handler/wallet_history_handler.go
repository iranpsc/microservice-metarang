package handler

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/commercial-service/internal/models"
	"metarang/commercial-service/internal/service"
	pb "metarang/shared/pb/commercial"
	periodpkg "metarang/shared/pkg/period"
)

type walletHistoryService interface {
	GetSummary(ctx context.Context, userID uint64, periodStr string, assets []string, privacy map[string]int32) (*service.WalletHistorySummaryResult, error)
	GetChart(ctx context.Context, userID uint64, periodStr string, assets []string, privacy map[string]int32) (*service.WalletHistoryChartResult, error)
}

// WalletHistoryHandler implements commercial.WalletHistoryService.
type WalletHistoryHandler struct {
	pb.UnimplementedWalletHistoryServiceServer
	svc walletHistoryService
}

func NewWalletHistoryHandler(svc walletHistoryService) *WalletHistoryHandler {
	return &WalletHistoryHandler{svc: svc}
}

func RegisterWalletHistoryHandler(grpcServer *grpc.Server, svc *service.WalletHistoryService) *WalletHistoryHandler {
	handler := NewWalletHistoryHandler(svc)
	pb.RegisterWalletHistoryServiceServer(grpcServer, handler)
	return handler
}

func (h *WalletHistoryHandler) GetWalletHistorySummary(
	ctx context.Context,
	req *pb.GetWalletHistorySummaryRequest,
) (*pb.GetWalletHistorySummaryResponse, error) {
	if req == nil || req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if !isValidPeriod(req.Period) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid period [%s] provided", req.Period)
	}

	result, err := h.svc.GetSummary(ctx, req.UserId, req.Period, req.Assets, req.Privacy)
	if err != nil {
		if strings.Contains(err.Error(), "invalid period") {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to get wallet history summary: %v", err)
	}

	data := make([]*pb.WalletAssetCard, 0, len(result.Cards))
	for _, card := range result.Cards {
		data = append(data, &pb.WalletAssetCard{
			Asset:             card.Asset,
			CurrentBalance:    card.CurrentBalance,
			PeriodIncome:      card.PeriodIncome,
			PeriodSpending:    card.PeriodSpending,
			GrowthPercent:     card.GrowthPercent,
			Direction:         card.Direction,
			PrivacyRestricted: card.PrivacyRestricted,
		})
	}
	return &pb.GetWalletHistorySummaryResponse{Data: data, Period: result.Period}, nil
}

func (h *WalletHistoryHandler) GetWalletHistoryChart(
	ctx context.Context,
	req *pb.GetWalletHistoryChartRequest,
) (*pb.GetWalletHistoryChartResponse, error) {
	if req == nil || req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if !isValidPeriod(req.Period) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid period [%s] provided", req.Period)
	}

	result, err := h.svc.GetChart(ctx, req.UserId, req.Period, req.Assets, req.Privacy)
	if err != nil {
		if strings.Contains(err.Error(), "invalid period") {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to get wallet history chart: %v", err)
	}

	data := make(map[string]*pb.WalletAssetChartSeries, len(result.Charts))
	for asset, chart := range result.Charts {
		data[asset] = &pb.WalletAssetChartSeries{
			Income:   toProtoPoints(chart.Income),
			Spending: toProtoPoints(chart.Spending),
		}
	}
	return &pb.GetWalletHistoryChartResponse{Data: data, Period: result.Period}, nil
}

func toProtoPoints(points []models.WalletChartPoint) []*pb.WalletChartPoint {
	out := make([]*pb.WalletChartPoint, len(points))
	for i, p := range points {
		out[i] = &pb.WalletChartPoint{Label: p.Label, Amount: p.Amount}
	}
	return out
}

func isValidPeriod(period string) bool {
	for _, p := range periodpkg.ValidPeriods {
		if p == period {
			return true
		}
	}
	return false
}
