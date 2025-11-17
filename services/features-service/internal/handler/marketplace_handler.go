package handler

import (
	"context"

	"metargb/features-service/internal/service"
	pb "metargb/shared/pb/features"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MarketplaceHandler struct {
	pb.UnimplementedFeatureMarketplaceServiceServer
	service *service.MarketplaceService
}

func NewMarketplaceHandler(service *service.MarketplaceService) *MarketplaceHandler {
	return &MarketplaceHandler{
		service: service,
	}
}

// BuyFeature handles direct feature purchase
// Implements the logic from Laravel's BuyFeatureController@buy
func (h *MarketplaceHandler) BuyFeature(ctx context.Context, req *pb.BuyFeatureRequest) (*pb.BuyFeatureResponse, error) {
	if req.FeatureId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "feature_id is required")
	}
	if req.BuyerId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "buyer_id is required")
	}

	err := h.service.BuyFeature(ctx, req.FeatureId, req.BuyerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to buy feature: %v", err)
	}

	return &pb.BuyFeatureResponse{
		Success: true,
		Message: "Feature purchased successfully",
		Feature: nil, // Feature details can be fetched separately if needed
	}, nil
}

// SendBuyRequest creates a buy request for a feature
// Implements Laravel's BuyRequestsController@store
func (h *MarketplaceHandler) SendBuyRequest(ctx context.Context, req *pb.SendBuyRequestRequest) (*pb.BuyRequestResponse, error) {
	if req.FeatureId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "feature_id is required")
	}
	if req.BuyerId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "buyer_id is required")
	}

	_, err := h.service.SendBuyRequest(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to send buy request: %v", err)
	}

	// Return stub response - TODO: implement fully
	return &pb.BuyRequestResponse{
		Id:        0,
		BuyerId:   req.BuyerId,
		SellerId:  0,
		FeatureId: req.FeatureId,
		Status:    0,
		Note:      req.Note,
		PricePsc:  req.PricePsc,
		PriceIrr:  req.PriceIrr,
	}, nil
}

// AcceptBuyRequest accepts a pending buy request
// Implements Laravel's BuyRequestsController@acceptBuyRequest
func (h *MarketplaceHandler) AcceptBuyRequest(ctx context.Context, req *pb.AcceptBuyRequestRequest) (*pb.BuyRequestResponse, error) {
	if req.RequestId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "request_id is required")
	}
	if req.SellerId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "seller_id is required")
	}

	_, err := h.service.AcceptBuyRequest(ctx, req.RequestId, req.SellerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to accept buy request: %v", err)
	}

	// Return stub response - TODO: implement fully
	return &pb.BuyRequestResponse{
		Id:       req.RequestId,
		SellerId: req.SellerId,
		Status:   1, // Accepted
	}, nil
}

// CreateSellRequest creates a sell request for a feature
// Implements Laravel's SellRequestsController@store
func (h *MarketplaceHandler) CreateSellRequest(ctx context.Context, req *pb.CreateSellRequestRequest) (*pb.SellRequestResponse, error) {
	if req.FeatureId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "feature_id is required")
	}
	if req.SellerId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "seller_id is required")
	}

	_, err := h.service.CreateSellRequest(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create sell request: %v", err)
	}

	// Return stub response - TODO: implement fully
	return &pb.SellRequestResponse{
		Id:        0,
		SellerId:  req.SellerId,
		FeatureId: req.FeatureId,
		PricePsc:  req.PricePsc,
		PriceIrr:  req.PriceIrr,
		Note:      req.Note,
	}, nil
}

// RequestGracePeriod adds grace period to a buy request
// Implements Laravel's BuyRequestsController@addGracePeriod
func (h *MarketplaceHandler) RequestGracePeriod(ctx context.Context, req *pb.RequestGracePeriodRequest) (*pb.GracePeriodResponse, error) {
	if req.RequestId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "request_id is required")
	}

	err := h.service.RequestGracePeriod(ctx, req.RequestId, req.BuyerId, req.GracePeriod)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to request grace period: %v", err)
	}

	return &pb.GracePeriodResponse{
		Approved: true,
		Message:  "Grace period added successfully",
	}, nil
}
