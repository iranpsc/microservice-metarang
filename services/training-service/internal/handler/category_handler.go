package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonpb "metargb/shared/pb/common"
	trainingpb "metargb/shared/pb/training"
	"metargb/training-service/internal/service"
)

type CategoryHandler struct {
	trainingpb.UnimplementedCategoryServiceServer
	service *service.CategoryService
}

func RegisterCategoryHandler(grpcServer *grpc.Server, svc *service.CategoryService) {
	handler := &CategoryHandler{service: svc}
	trainingpb.RegisterCategoryServiceServer(grpcServer, handler)
}

// GetCategories retrieves paginated categories
func (h *CategoryHandler) GetCategories(ctx context.Context, req *trainingpb.GetCategoriesRequest) (*trainingpb.CategoriesResponse, error) {
	page := int32(1)
	perPage := int32(30) // Default per API spec

	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PerPage > 0 {
			perPage = req.Pagination.PerPage
		}
	}

	categories, total, err := h.service.GetCategories(ctx, page, perPage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get categories: %v", err)
	}

	response := &trainingpb.CategoriesResponse{
		Categories: make([]*trainingpb.CategoryResponse, 0, len(categories)),
		Pagination: &commonpb.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       total,
			LastPage:    (total + perPage - 1) / perPage,
		},
	}

	for _, category := range categories {
		// Get stats for each category
		stats, _ := h.service.GetCategoryStats(ctx, category.ID)
		catResp := &trainingpb.CategoryResponse{
			Id:          category.ID,
			Name:        category.Name,
			Slug:        category.Slug,
			Description: category.Description,
		}
		if stats != nil {
			catResp.VideosCount = stats.VideosCount
		}
		response.Categories = append(response.Categories, catResp)
	}

	return response, nil
}

// GetCategory retrieves a category by slug
func (h *CategoryHandler) GetCategory(ctx context.Context, req *trainingpb.GetCategoryRequest) (*trainingpb.CategoryResponse, error) {
	details, err := h.service.GetCategoryBySlug(ctx, req.Slug)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "category not found: %v", err)
	}

	resp := &trainingpb.CategoryResponse{
		Id:          details.Category.ID,
		Name:        details.Category.Name,
		Slug:        details.Category.Slug,
		Description: details.Category.Description,
	}

	if details.Stats != nil {
		resp.VideosCount = details.Stats.VideosCount
	}

	// Add subcategories
	if len(details.SubCategories) > 0 {
		resp.SubCategories = make([]*trainingpb.SubCategoryInfo, 0, len(details.SubCategories))
		for _, subCat := range details.SubCategories {
			subCatInfo := &trainingpb.SubCategoryInfo{
				Id:   subCat.ID,
				Name: subCat.Name,
				Slug: subCat.Slug,
			}
			if _, ok := details.SubCategoryStats[subCat.ID]; ok {
				// Note: SubCategoryInfo doesn't have count fields in proto, but we can add them if needed
			}
			resp.SubCategories = append(resp.SubCategories, subCatInfo)
		}
	}

	return resp, nil
}

// GetSubCategory retrieves a subcategory by slugs
func (h *CategoryHandler) GetSubCategory(ctx context.Context, req *trainingpb.GetSubCategoryRequest) (*trainingpb.SubCategoryResponse, error) {
	details, err := h.service.GetSubCategoryBySlugs(ctx, req.CategorySlug, req.SubCategorySlug)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "subcategory not found: %v", err)
	}

	resp := &trainingpb.SubCategoryResponse{
		Id:          details.SubCategory.ID,
		Name:        details.SubCategory.Name,
		Slug:        details.SubCategory.Slug,
		Description: details.SubCategory.Description,
	}

	if details.Category != nil {
		resp.Category = &trainingpb.CategoryInfo{
			Id:   details.Category.ID,
			Name: details.Category.Name,
			Slug: details.Category.Slug,
		}
	}

	if details.Stats != nil {
		resp.VideosCount = details.Stats.VideosCount
	}

	return resp, nil
}

// GetCategoryVideos retrieves videos for a category
func (h *CategoryHandler) GetCategoryVideos(ctx context.Context, req *trainingpb.GetCategoryVideosRequest) (*trainingpb.VideosResponse, error) {
	page := int32(1)
	perPage := int32(18) // Default per API spec

	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PerPage > 0 {
			perPage = req.Pagination.PerPage
		}
	}

	videos, total, err := h.service.GetCategoryVideos(ctx, req.CategorySlug, page, perPage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get category videos: %v", err)
	}

	response := &trainingpb.VideosResponse{
		Videos: make([]*trainingpb.VideoResponse, 0, len(videos)),
		Pagination: &commonpb.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       total,
			LastPage:    (total + perPage - 1) / perPage,
		},
	}

	// Build video responses (would need video service for full details)
	// For now, return basic structure
	return response, nil
}
