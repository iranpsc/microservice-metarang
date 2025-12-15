package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/financial-service/internal/service"
	pb "metargb/shared/pb/financial"
)

type StoreHandler struct {
	pb.UnimplementedStoreServiceServer
	storeService service.StoreService
}

func NewStoreHandler(storeService service.StoreService) *StoreHandler {
	return &StoreHandler{
		storeService: storeService,
	}
}

func RegisterStoreHandler(grpcServer *grpc.Server, storeService service.StoreService) {
	handler := NewStoreHandler(storeService)
	pb.RegisterStoreServiceServer(grpcServer, handler)
}

func (h *StoreHandler) GetStorePackages(ctx context.Context, req *pb.GetStorePackagesRequest) (*pb.GetStorePackagesResponse, error) {
	// Validation
	if len(req.Codes) < 2 {
		return nil, status.Errorf(codes.InvalidArgument, "codes must contain at least 2 items")
	}

	for _, code := range req.Codes {
		if len(code) < 2 {
			return nil, status.Errorf(codes.InvalidArgument, "each code must be at least 2 characters")
		}
	}

	// Call service
	packages, err := h.storeService.GetStorePackages(ctx, req.Codes)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCodes) || errors.Is(err, service.ErrInvalidCodeLength) {
			return nil, status.Errorf(codes.InvalidArgument, "%v", err)
		}
		return nil, status.Errorf(codes.Internal, "failed to get store packages: %v", err)
	}

	// Convert to proto messages
	pbPackages := make([]*pb.Package, 0, len(packages))
	for _, pkg := range packages {
		pbPackage := &pb.Package{
			Id:        pkg.ID,
			Code:      pkg.Code,
			Asset:     pkg.Asset,
			Amount:    pkg.Amount,
			UnitPrice: pkg.UnitPrice,
		}
		if pkg.Image != nil && *pkg.Image != "" {
			pbPackage.Image = pkg.Image
		}
		pbPackages = append(pbPackages, pbPackage)
	}

	return &pb.GetStorePackagesResponse{
		Packages: pbPackages,
	}, nil
}
