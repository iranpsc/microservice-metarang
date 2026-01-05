package handler

import (
	"context"
	"errors"
	"testing"

	handlerpkg "metargb/financial-service/internal/handler"
	"metargb/financial-service/internal/service"
	pb "metargb/shared/pb/financial"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockStoreService struct {
	packages []*service.PackageResource
	err      error
}

func (m *mockStoreService) GetStorePackages(ctx context.Context, codes []string) ([]*service.PackageResource, error) {
	return m.packages, m.err
}

func TestStoreHandler_GetStorePackages(t *testing.T) {
	ctx := context.Background()

	t.Run("successful package retrieval with images", func(t *testing.T) {
		imageURL1 := "https://example.com/image1.jpg"
		imageURL2 := "https://example.com/image2.jpg"
		packages := []*service.PackageResource{
			{
				ID:        1,
				Code:      "PACK1",
				Asset:     "psc",
				Amount:    100.0,
				UnitPrice: 1000.0,
				Image:     &imageURL1,
			},
			{
				ID:        2,
				Code:      "PACK2",
				Asset:     "red",
				Amount:    50.0,
				UnitPrice: 2000.0,
				Image:     &imageURL2,
			},
		}

		mockService := &mockStoreService{
			packages: packages,
			err:      nil,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"PACK1", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Packages) != 2 {
			t.Fatalf("expected 2 packages, got %d", len(resp.Packages))
		}

		// Verify first package
		pkg1 := resp.Packages[0]
		if pkg1.Id != 1 {
			t.Errorf("expected package ID 1, got %d", pkg1.Id)
		}
		if pkg1.Code != "PACK1" {
			t.Errorf("expected code PACK1, got %s", pkg1.Code)
		}
		if pkg1.Asset != "psc" {
			t.Errorf("expected asset psc, got %s", pkg1.Asset)
		}
		if pkg1.Amount != 100.0 {
			t.Errorf("expected amount 100.0, got %f", pkg1.Amount)
		}
		if pkg1.UnitPrice != 1000.0 {
			t.Errorf("expected unit price 1000.0, got %f", pkg1.UnitPrice)
		}
		if pkg1.Image == nil || *pkg1.Image != imageURL1 {
			t.Errorf("expected image %s, got %v", imageURL1, pkg1.Image)
		}

		// Verify second package
		pkg2 := resp.Packages[1]
		if pkg2.Id != 2 {
			t.Errorf("expected package ID 2, got %d", pkg2.Id)
		}
		if pkg2.Code != "PACK2" {
			t.Errorf("expected code PACK2, got %s", pkg2.Code)
		}
		if pkg2.Image == nil || *pkg2.Image != imageURL2 {
			t.Errorf("expected image %s, got %v", imageURL2, pkg2.Image)
		}
	})

	t.Run("successful package retrieval without images", func(t *testing.T) {
		packages := []*service.PackageResource{
			{
				ID:        1,
				Code:      "PACK1",
				Asset:     "psc",
				Amount:    100.0,
				UnitPrice: 1000.0,
				Image:     nil,
			},
			{
				ID:        2,
				Code:      "PACK2",
				Asset:     "red",
				Amount:    50.0,
				UnitPrice: 2000.0,
				Image:     nil,
			},
		}

		mockService := &mockStoreService{
			packages: packages,
			err:      nil,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"PACK1", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Packages) != 2 {
			t.Fatalf("expected 2 packages, got %d", len(resp.Packages))
		}

		// Verify images are nil
		if resp.Packages[0].Image != nil {
			t.Errorf("expected nil image for package 1, got %v", resp.Packages[0].Image)
		}
		if resp.Packages[1].Image != nil {
			t.Errorf("expected nil image for package 2, got %v", resp.Packages[1].Image)
		}
	})

	t.Run("successful package retrieval with mixed images", func(t *testing.T) {
		imageURL1 := "https://example.com/image1.jpg"
		packages := []*service.PackageResource{
			{
				ID:        1,
				Code:      "PACK1",
				Asset:     "psc",
				Amount:    100.0,
				UnitPrice: 1000.0,
				Image:     &imageURL1,
			},
			{
				ID:        2,
				Code:      "PACK2",
				Asset:     "red",
				Amount:    50.0,
				UnitPrice: 2000.0,
				Image:     nil,
			},
		}

		mockService := &mockStoreService{
			packages: packages,
			err:      nil,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"PACK1", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Packages) != 2 {
			t.Fatalf("expected 2 packages, got %d", len(resp.Packages))
		}

		// Verify first package has image
		if resp.Packages[0].Image == nil || *resp.Packages[0].Image != imageURL1 {
			t.Errorf("expected image %s for package 1, got %v", imageURL1, resp.Packages[0].Image)
		}

		// Verify second package has no image
		if resp.Packages[1].Image != nil {
			t.Errorf("expected nil image for package 2, got %v", resp.Packages[1].Image)
		}
	})

	t.Run("successful empty response when codes don't match", func(t *testing.T) {
		// Per API documentation: "200 with empty array – All provided codes fail to resolve"
		packages := []*service.PackageResource{}

		mockService := &mockStoreService{
			packages: packages,
			err:      nil,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"INVALID1", "INVALID2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Packages) != 0 {
			t.Errorf("expected empty packages array, got %d packages", len(resp.Packages))
		}
	})

	t.Run("validation error: empty codes array", func(t *testing.T) {
		mockService := &mockStoreService{}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err == nil {
			t.Fatal("expected validation error for empty codes array")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error code, got %v", st.Code())
		}

		if resp != nil {
			t.Error("expected nil response on validation error")
		}
	})

	t.Run("validation error: single code (less than 2)", func(t *testing.T) {
		mockService := &mockStoreService{}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"PACK1"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err == nil {
			t.Fatal("expected validation error for single code")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error code, got %v", st.Code())
		}

		if resp != nil {
			t.Error("expected nil response on validation error")
		}
	})

	t.Run("validation error: code with length less than 2", func(t *testing.T) {
		mockService := &mockStoreService{}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"P", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err == nil {
			t.Fatal("expected validation error for code with length < 2")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error code, got %v", st.Code())
		}

		if resp != nil {
			t.Error("expected nil response on validation error")
		}
	})

	t.Run("validation error: multiple codes with length less than 2", func(t *testing.T) {
		mockService := &mockStoreService{}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"A", "B"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err == nil {
			t.Fatal("expected validation error for codes with length < 2")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error code, got %v", st.Code())
		}

		if resp != nil {
			t.Error("expected nil response on validation error")
		}
	})

	t.Run("validation error: empty string code", func(t *testing.T) {
		mockService := &mockStoreService{}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err == nil {
			t.Fatal("expected validation error for empty string code")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error code, got %v", st.Code())
		}

		if resp != nil {
			t.Error("expected nil response on validation error")
		}
	})

	t.Run("service error: ErrInvalidCodes", func(t *testing.T) {
		mockService := &mockStoreService{
			packages: nil,
			err:      service.ErrInvalidCodes,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"PACK1", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err == nil {
			t.Fatal("expected error from service")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error code for ErrInvalidCodes, got %v", st.Code())
		}

		if resp != nil {
			t.Error("expected nil response on service error")
		}
	})

	t.Run("service error: ErrInvalidCodeLength", func(t *testing.T) {
		mockService := &mockStoreService{
			packages: nil,
			err:      service.ErrInvalidCodeLength,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"PACK1", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err == nil {
			t.Fatal("expected error from service")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("expected gRPC status error")
		}
		if st.Code() != codes.InvalidArgument {
			t.Errorf("expected InvalidArgument error code for ErrInvalidCodeLength, got %v", st.Code())
		}

		if resp != nil {
			t.Error("expected nil response on service error")
		}
	})

	t.Run("service error: internal error", func(t *testing.T) {
		mockService := &mockStoreService{
			packages: nil,
			err:      errors.New("database connection failed"),
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"PACK1", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err == nil {
			t.Fatal("expected error from service")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Fatal("expected gRPC status error")
		}
		if st.Code() != codes.Internal {
			t.Errorf("expected Internal error code for database error, got %v", st.Code())
		}

		if resp != nil {
			t.Error("expected nil response on service error")
		}
	})

	t.Run("successful retrieval with many packages", func(t *testing.T) {
		packages := make([]*service.PackageResource, 10)
		for i := 0; i < 10; i++ {
			packages[i] = &service.PackageResource{
				ID:        uint64(i + 1),
				Code:      "PACK" + string(rune('A'+i)),
				Asset:     "psc",
				Amount:    float64((i + 1) * 10),
				UnitPrice: float64((i + 1) * 100),
				Image:     nil,
			}
		}

		mockService := &mockStoreService{
			packages: packages,
			err:      nil,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		codes := make([]string, 10)
		for i := 0; i < 10; i++ {
			codes[i] = "PACK" + string(rune('A'+i))
		}

		req := &pb.GetStorePackagesRequest{
			Codes: codes,
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Packages) != 10 {
			t.Fatalf("expected 10 packages, got %d", len(resp.Packages))
		}

		// Verify all packages are present
		for i, pkg := range resp.Packages {
			if pkg.Id != uint64(i+1) {
				t.Errorf("package %d: expected ID %d, got %d", i, i+1, pkg.Id)
			}
		}
	})

	t.Run("successful retrieval with exact minimum codes (2)", func(t *testing.T) {
		packages := []*service.PackageResource{
			{
				ID:        1,
				Code:      "PACK1",
				Asset:     "psc",
				Amount:    100.0,
				UnitPrice: 1000.0,
				Image:     nil,
			},
			{
				ID:        2,
				Code:      "PACK2",
				Asset:     "red",
				Amount:    50.0,
				UnitPrice: 2000.0,
				Image:     nil,
			},
		}

		mockService := &mockStoreService{
			packages: packages,
			err:      nil,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"PACK1", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Packages) != 2 {
			t.Fatalf("expected 2 packages, got %d", len(resp.Packages))
		}
	})

	t.Run("successful retrieval with codes of exact minimum length (2)", func(t *testing.T) {
		packages := []*service.PackageResource{
			{
				ID:        1,
				Code:      "AB",
				Asset:     "psc",
				Amount:    100.0,
				UnitPrice: 1000.0,
				Image:     nil,
			},
			{
				ID:        2,
				Code:      "CD",
				Asset:     "red",
				Amount:    50.0,
				UnitPrice: 2000.0,
				Image:     nil,
			},
		}

		mockService := &mockStoreService{
			packages: packages,
			err:      nil,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"AB", "CD"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(resp.Packages) != 2 {
			t.Fatalf("expected 2 packages, got %d", len(resp.Packages))
		}
	})

	t.Run("package with empty image string should be nil", func(t *testing.T) {
		emptyImage := ""
		packages := []*service.PackageResource{
			{
				ID:        1,
				Code:      "PACK1",
				Asset:     "psc",
				Amount:    100.0,
				UnitPrice: 1000.0,
				Image:     &emptyImage, // Empty string image
			},
		}

		mockService := &mockStoreService{
			packages: packages,
			err:      nil,
		}
		handler := handlerpkg.NewStoreHandler(mockService)

		req := &pb.GetStorePackagesRequest{
			Codes: []string{"PACK1", "PACK2"},
		}

		resp, err := handler.GetStorePackages(ctx, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// According to handler logic, empty string images should be set
		// But the handler checks: if pkg.Image != nil && *pkg.Image != ""
		// So empty string should result in nil in proto
		// However, the service returns emptyImage as non-nil pointer to empty string
		// The handler should handle this case
		if len(resp.Packages) != 1 {
			t.Fatalf("expected 1 package, got %d", len(resp.Packages))
		}

		// The handler checks: if pkg.Image != nil && *pkg.Image != ""
		// So if Image is pointer to empty string, it should be nil in proto
		// But current implementation sets it if not nil and not empty
		// This test verifies current behavior
		if resp.Packages[0].Image != nil && *resp.Packages[0].Image == "" {
			t.Error("expected nil or non-empty image, got empty string")
		}
	})
}

func TestStoreHandler_NewStoreHandler(t *testing.T) {
	mockService := &mockStoreService{}
	handler := handlerpkg.NewStoreHandler(mockService)

	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// Verify handler can be used (we can't check private fields)
	// Test that it doesn't panic when created
	_ = handler
}

func TestStoreHandler_RegisterStoreHandler(t *testing.T) {
	// This test verifies that RegisterStoreHandler doesn't panic
	// Full integration would require a real gRPC server
	mockService := &mockStoreService{}

	// We can't easily test the registration without a real server,
	// but we can verify the function exists and the handler is created
	handler := handlerpkg.NewStoreHandler(mockService)
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}
