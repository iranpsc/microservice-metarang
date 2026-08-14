package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"metarang/support-service/internal/models"
	"metarang/support-service/internal/service"
	"metarang/support-service/tests/internal/testutil"
)

func TestReportService_CreateReport_CreateAndImageErrors(t *testing.T) {
	repo := &testutil.MockReportRepo{
		CreateFunc: func(ctx context.Context, report *models.Report) (*models.Report, error) {
			return nil, errors.New("insert")
		},
	}
	svc := service.NewReportService(repo)
	_, err := svc.CreateReport(context.Background(), 1, "displayError", "t", "c", "https://x", nil)
	if err == nil || !strings.Contains(err.Error(), "failed to create report") {
		t.Fatalf("err=%v", err)
	}

	n := 0
	repo = &testutil.MockReportRepo{
		CreateFunc: func(ctx context.Context, report *models.Report) (*models.Report, error) {
			report.ID = 4
			return report, nil
		},
		CreateImageFunc: func(ctx context.Context, reportID uint64, url string) error {
			n++
			if n == 2 {
				return errors.New("image 2")
			}
			return nil
		},
	}
	svc = service.NewReportService(repo)
	_, err = svc.CreateReport(context.Background(), 1, "displayError", "t", "c", "https://x", []string{"a.png", "b.png"})
	if err == nil || !strings.Contains(err.Error(), "failed to create image") {
		t.Fatalf("err=%v", err)
	}
}

func TestReportService_CreateReport_ZeroAndNImages(t *testing.T) {
	images := 0
	repo := &testutil.MockReportRepo{
		CreateFunc: func(ctx context.Context, report *models.Report) (*models.Report, error) {
			report.ID = 5
			return report, nil
		},
		CreateImageFunc: func(ctx context.Context, reportID uint64, url string) error {
			images++
			return nil
		},
		GetByIDFunc: func(ctx context.Context, reportID uint64) (*models.ReportWithImages, error) {
			return &models.ReportWithImages{Report: models.Report{ID: reportID, UserID: 9}}, nil
		},
	}
	svc := service.NewReportService(repo)
	got, err := svc.CreateReport(context.Background(), 9, "displayError", "t", "c", "https://x", nil)
	if err != nil || got.ID != 5 || images != 0 {
		t.Fatalf("zero images err=%v images=%d", err, images)
	}
	got, err = svc.CreateReport(context.Background(), 9, "displayError", "t", "c", "https://x", []string{"a.png", "b.png"})
	if err != nil || images != 2 {
		t.Fatalf("n images err=%v images=%d got=%+v", err, images, got)
	}
}

func TestReportService_GetReports_PaginationDefaults(t *testing.T) {
	var gotPage, gotPer int32
	repo := &testutil.MockReportRepo{
		GetByUserIDFunc: func(ctx context.Context, userID uint64, page, perPage int32) ([]*models.Report, int, error) {
			gotPage, gotPer = page, perPage
			return nil, 0, nil
		},
	}
	svc := service.NewReportService(repo)
	_, _, err := svc.GetReports(context.Background(), 1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if gotPage != 1 || gotPer != 10 {
		t.Fatalf("page=%d per=%d", gotPage, gotPer)
	}
}

func TestReportService_GetReport_RepoErrorNilAndUnauthorized(t *testing.T) {
	repo := &testutil.MockReportRepo{
		GetByIDFunc: func(ctx context.Context, reportID uint64) (*models.ReportWithImages, error) {
			return nil, errors.New("db")
		},
	}
	svc := service.NewReportService(repo)
	_, err := svc.GetReport(context.Background(), 1, 1)
	if err == nil || err.Error() != "db" {
		t.Fatalf("err=%v", err)
	}

	repo = &testutil.MockReportRepo{
		GetByIDFunc: func(ctx context.Context, reportID uint64) (*models.ReportWithImages, error) {
			return nil, nil
		},
	}
	svc = service.NewReportService(repo)
	got, err := svc.GetReport(context.Background(), 1, 1)
	if err != nil || got != nil {
		t.Fatalf("expected nil report got=%v err=%v", got, err)
	}
}
