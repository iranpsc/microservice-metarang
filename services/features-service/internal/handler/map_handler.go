package handler

import (
	"context"

	"metargb/features-service/internal/service"
	pb "metargb/shared/pb/features"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type MapHandler struct {
	pb.UnimplementedMapsServiceServer
	service *service.MapService
}

func NewMapHandler(service *service.MapService) *MapHandler {
	return &MapHandler{
		service: service,
	}
}

// ListMaps returns all maps with basic information
func (h *MapHandler) ListMaps(ctx context.Context, req *pb.ListMapsRequest) (*pb.ListMapsResponse, error) {
	maps, err := h.service.ListMaps(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list maps: %v", err)
	}

	return &pb.ListMapsResponse{
		Maps: maps,
	}, nil
}

// GetMap returns a single map with detailed information
func (h *MapHandler) GetMap(ctx context.Context, req *pb.GetMapRequest) (*pb.GetMapResponse, error) {
	if req.MapId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "map_id is required")
	}

	m, err := h.service.GetMap(ctx, req.MapId)
	if err != nil {
		if err.Error() == "map not found" {
			return nil, status.Errorf(codes.NotFound, "map not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get map: %v", err)
	}

	return &pb.GetMapResponse{
		Map: m,
	}, nil
}

// GetMapBorder returns just the border coordinates for a map
func (h *MapHandler) GetMapBorder(ctx context.Context, req *pb.GetMapRequest) (*pb.GetMapBorderResponse, error) {
	if req.MapId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "map_id is required")
	}

	borderCoordinates, err := h.service.GetMapBorder(ctx, req.MapId)
	if err != nil {
		if err.Error() == "map not found" {
			return nil, status.Errorf(codes.NotFound, "map not found")
		}
		return nil, status.Errorf(codes.Internal, "failed to get map border: %v", err)
	}

	return &pb.GetMapBorderResponse{
		Data: &pb.MapBorderData{
			BorderCoordinates: borderCoordinates,
		},
	}, nil
}
