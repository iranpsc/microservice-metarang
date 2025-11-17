package handler

import (
	"context"
	"metargb/support-service/internal/models"
	"metargb/support-service/internal/service"
	"metargb/support-service/internal/utils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	pbCommon "metargb/shared/pb/common"
	pb "metargb/shared/pb/support"
)

type ReportHandler struct {
	pb.UnimplementedReportServiceServer
	reportService service.ReportService
}

func NewReportHandler(reportService service.ReportService) *ReportHandler {
	return &ReportHandler{
		reportService: reportService,
	}
}

func RegisterReportHandler(grpcServer *grpc.Server, reportService service.ReportService) {
	handler := NewReportHandler(reportService)
	pb.RegisterReportServiceServer(grpcServer, handler)
}

func (h *ReportHandler) CreateReport(ctx context.Context, req *pb.CreateReportRequest) (*pb.ReportResponse, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Reason == "" {
		return nil, status.Error(codes.InvalidArgument, "reason is required")
	}

	// Note: Laravel's Report model has different fields than what's in the proto
	// The proto expects reportable_type/reportable_id, but Laravel Report has subject/title/content/url
	// We'll map them appropriately
	report, err := h.reportService.CreateReport(ctx, req.UserId, req.ReportableType, req.Reason, req.Description, "", nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create report: %v", err)
	}

	return convertReportToProto(&report.Report), nil
}

func (h *ReportHandler) GetReports(ctx context.Context, req *pb.GetReportsRequest) (*pb.ReportsResponse, error) {
	if req.UserId == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	page := int32(1)
	perPage := int32(10)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PerPage > 0 {
			perPage = req.Pagination.PerPage
		}
	}

	reports, total, err := h.reportService.GetReports(ctx, req.UserId, page, perPage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get reports: %v", err)
	}

	response := &pb.ReportsResponse{
		Reports: make([]*pb.ReportResponse, len(reports)),
		Pagination: &pbCommon.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       int32(total),
			LastPage:    int32((total + int(perPage) - 1) / int(perPage)),
		},
	}

	for i, report := range reports {
		response.Reports[i] = convertReportToProto(report)
	}

	return response, nil
}

func (h *ReportHandler) GetReport(ctx context.Context, req *pb.GetReportRequest) (*pb.ReportResponse, error) {
	if req.ReportId == 0 {
		return nil, status.Error(codes.InvalidArgument, "report_id is required")
	}

	report, err := h.reportService.GetReport(ctx, req.ReportId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get report: %v", err)
	}

	if report == nil {
		return nil, status.Error(codes.NotFound, "report not found")
	}

	return convertReportToProto(&report.Report), nil
}

// Helper function to convert report model to proto response
func convertReportToProto(report *models.Report) *pb.ReportResponse {
	return &pb.ReportResponse{
		Id:             report.ID,
		UserId:         report.UserID,
		ReportableType: report.Subject, // Mapping subject to reportable_type
		ReportableId:   0,              // Not stored in Laravel Report model
		Reason:         report.Title,   // Mapping title to reason
		Description:    report.Content, // Mapping content to description
		CreatedAt:      utils.FormatJalaliDateTime(report.CreatedAt),
	}
}
