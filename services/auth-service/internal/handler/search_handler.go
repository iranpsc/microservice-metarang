package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
)

type searchHandler struct {
	pb.UnimplementedSearchServiceServer
	searchService service.SearchService
}

func RegisterSearchHandler(grpcServer *grpc.Server, searchService service.SearchService) {
	pb.RegisterSearchServiceServer(grpcServer, &searchHandler{
		searchService: searchService,
	})
}

// SearchUsers handles user search requests
func (h *searchHandler) SearchUsers(ctx context.Context, req *pb.SearchUsersRequest) (*pb.SearchUsersResponse, error) {
	// Validate request
	if req.SearchTerm == "" {
		return &pb.SearchUsersResponse{
			Data: []*pb.SearchUserResult{},
		}, nil
	}

	// Call service
	results, err := h.searchService.SearchUsers(ctx, req.SearchTerm)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	// Convert service results to protobuf
	pbResults := make([]*pb.SearchUserResult, 0, len(results))
	for _, result := range results {
		pbResult := &pb.SearchUserResult{
			Id:        result.ID,
			Code:      result.Code,
			Name:      result.Name,
			Followers: result.Followers,
		}

		if result.Level != nil {
			pbResult.Level = *result.Level
		}
		if result.Photo != nil {
			pbResult.Photo = *result.Photo
		}

		pbResults = append(pbResults, pbResult)
	}

	return &pb.SearchUsersResponse{
		Data: pbResults,
	}, nil
}

// SearchFeatures handles feature search requests
func (h *searchHandler) SearchFeatures(ctx context.Context, req *pb.SearchFeaturesRequest) (*pb.SearchFeaturesResponse, error) {
	// Validate request
	if req.SearchTerm == "" {
		return &pb.SearchFeaturesResponse{
			Data: []*pb.SearchFeatureResult{},
		}, nil
	}

	// Call service
	results, err := h.searchService.SearchFeatures(ctx, req.SearchTerm)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	// Convert service results to protobuf
	pbResults := make([]*pb.SearchFeatureResult, 0, len(results))
	for _, result := range results {
		pbResult := &pb.SearchFeatureResult{
			Id:                  result.ID,
			FeaturePropertiesId: result.FeaturePropertiesID,
			Address:             result.Address,
			Karbari:             result.Karbari,
			PricePsc:            result.PricePsc,
			PriceIrr:            result.PriceIrr,
			OwnerCode:           result.OwnerCode,
		}

		// Convert coordinates
		pbResult.Coordinates = make([]*pb.Coordinate, 0, len(result.Coordinates))
		for _, coord := range result.Coordinates {
			pbResult.Coordinates = append(pbResult.Coordinates, &pb.Coordinate{
				Id: coord.ID,
				X:  coord.X,
				Y:  coord.Y,
			})
		}

		pbResults = append(pbResults, pbResult)
	}

	return &pb.SearchFeaturesResponse{
		Data: pbResults,
	}, nil
}

// SearchIsicCodes handles ISIC code search requests
func (h *searchHandler) SearchIsicCodes(ctx context.Context, req *pb.SearchIsicCodesRequest) (*pb.SearchIsicCodesResponse, error) {
	// Validate request
	if req.SearchTerm == "" {
		return &pb.SearchIsicCodesResponse{
			Data: []*pb.IsicCodeResult{},
		}, nil
	}

	// Call service
	results, err := h.searchService.SearchIsicCodes(ctx, req.SearchTerm)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "search failed: %v", err)
	}

	// Convert service results to protobuf
	pbResults := make([]*pb.IsicCodeResult, 0, len(results))
	for _, result := range results {
		pbResults = append(pbResults, &pb.IsicCodeResult{
			Id:   result.ID,
			Name: result.Name,
			Code: result.Code,
		})
	}

	return &pb.SearchIsicCodesResponse{
		Data: pbResults,
	}, nil
}
