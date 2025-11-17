package handler

import (
	"context"
	"fmt"

	"metargb/features-service/internal/models"
	"metargb/features-service/internal/service"
	pb "metargb/shared/pb/features"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ProfitHandler struct {
	pb.UnimplementedFeatureProfitServiceServer
	service *service.ProfitService
}

func NewProfitHandler(service *service.ProfitService) *ProfitHandler {
	return &ProfitHandler{
		service: service,
	}
}

// GetHourlyProfits retrieves all hourly profits for a user with totals by karbari
// Implements Laravel's FeatureHourlyProfitController@index
func (h *ProfitHandler) GetHourlyProfits(ctx context.Context, req *pb.GetHourlyProfitsRequest) (*pb.HourlyProfitsResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	profitsRaw, totalMaskoni, totalTejari, totalAmozeshi, err := h.service.GetHourlyProfits(ctx, req.UserId, req.Page, req.PageSize)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get hourly profits: %v", err)
	}

	// Convert internal models to protobuf
	profits := []*pb.HourlyProfit{}
	if profitsSlice, ok := profitsRaw.([]*models.FeatureHourlyProfit); ok {
		for _, p := range profitsSlice {
			profits = append(profits, &pb.HourlyProfit{
				Id:        p.ID,
				FeatureId: p.FeatureID,
				UserId:    p.UserID,
				Asset:     p.Asset,
				Amount:    fmt.Sprintf("%.6f", p.Amount),
				DeadLine:  p.Deadline.Format("2006-01-02 15:04:05"),
				IsActive:  p.IsActive,
			})
		}
	}

	return &pb.HourlyProfitsResponse{
		Profits:              profits,
		TotalMaskoniProfit:   totalMaskoni,
		TotalTejariProfit:    totalTejari,
		TotalAmozeshiProfit:  totalAmozeshi,
	}, nil
}

// GetSingleProfit retrieves and processes a single profit
// Implements Laravel's FeatureHourlyProfitController@getSingleProfit
func (h *ProfitHandler) GetSingleProfit(ctx context.Context, req *pb.GetSingleProfitRequest) (*pb.HourlyProfitResponse, error) {
	if req.ProfitId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "profit_id is required")
	}
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	amount, err := h.service.GetSingleProfit(ctx, req.ProfitId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get single profit: %v", err)
	}

	return &pb.HourlyProfitResponse{
		Profit: &pb.HourlyProfit{
			Id:     req.ProfitId,
			UserId: req.UserId,
			Amount: fmt.Sprintf("%.6f", amount),
		},
		Success: true,
	}, nil
}

// GetProfitsByApplication retrieves profits by karbari (m/t/a) and transfers to wallet
// Implements Laravel's FeatureHourlyProfitController@getProfitsByApplication
func (h *ProfitHandler) GetProfitsByApplication(ctx context.Context, req *pb.GetProfitsByApplicationRequest) (*pb.ProfitsByApplicationResponse, error) {
	if req.UserId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	// Validate karbari
	if req.Karbari != "m" && req.Karbari != "t" && req.Karbari != "a" {
		return nil, status.Errorf(codes.InvalidArgument, "karbari must be one of: m, t, a")
	}

	totalAmount, err := h.service.GetProfitsByApplication(ctx, req.UserId, req.Karbari)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get profits by application: %v", err)
	}

	return &pb.ProfitsByApplicationResponse{
		TotalAmount: fmt.Sprintf("%.6f", totalAmount),
		Success:     true,
	}, nil
}

