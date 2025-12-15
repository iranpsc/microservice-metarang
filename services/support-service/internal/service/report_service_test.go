package service

import (
	"context"
	"testing"
	"time"

	"metargb/support-service/internal/models"
)

// mockReportRepository implements ReportRepository for testing
type mockReportRepository struct {
	reports          map[uint64]*models.ReportWithImages
	userReports      map[uint64][]*models.Report
	images           map[uint64][]models.Image
	createCount      int
	createImageCount int
}

func newMockReportRepository() *mockReportRepository {
	return &mockReportRepository{
		reports:     make(map[uint64]*models.ReportWithImages),
		userReports: make(map[uint64][]*models.Report),
		images:      make(map[uint64][]models.Image),
	}
}

func (m *mockReportRepository) Create(ctx context.Context, report *models.Report) (*models.Report, error) {
	m.createCount++
	id := uint64(len(m.reports) + 1)
	report.ID = id
	report.CreatedAt = time.Now()
	report.UpdatedAt = time.Now()

	reportWithImages := &models.ReportWithImages{
		Report: *report,
		Images: []models.Image{},
	}
	m.reports[id] = reportWithImages
	m.userReports[report.UserID] = append(m.userReports[report.UserID], report)
	return report, nil
}

func (m *mockReportRepository) GetByID(ctx context.Context, reportID uint64) (*models.ReportWithImages, error) {
	report, exists := m.reports[reportID]
	if !exists {
		return nil, nil
	}
	if images, ok := m.images[reportID]; ok {
		report.Images = images
	}
	return report, nil
}

func (m *mockReportRepository) GetByUserID(ctx context.Context, userID uint64, page, perPage int32) ([]*models.Report, int, error) {
	reports := m.userReports[userID]
	return reports, len(reports), nil
}

func (m *mockReportRepository) CreateImage(ctx context.Context, reportID uint64, url string) error {
	m.createImageCount++
	image := models.Image{
		ID:            uint64(len(m.images[reportID]) + 1),
		ImageableType: "App\\Models\\Report",
		ImageableID:   reportID,
		URL:           url,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	m.images[reportID] = append(m.images[reportID], image)
	return nil
}

func TestReportService_CreateReport(t *testing.T) {
	ctx := context.Background()
	repo := newMockReportRepository()
	service := NewReportService(repo)

	t.Run("successful creation", func(t *testing.T) {
		userID := uint64(1)
		subject := "displayError"
		title := "Test Report"
		content := "Test Content"
		url := "https://example.com"

		report, err := service.CreateReport(ctx, userID, subject, title, content, url, nil)
		if err != nil {
			t.Fatalf("CreateReport failed: %v", err)
		}

		if report.Subject != subject {
			t.Errorf("Expected subject %s, got %s", subject, report.Subject)
		}
		if report.Title != title {
			t.Errorf("Expected title %s, got %s", title, report.Title)
		}
	})

	t.Run("creation with images", func(t *testing.T) {
		userID := uint64(1)
		imageURLs := []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"}

		report, err := service.CreateReport(ctx, userID, "spellingError", "Title", "Content", "", imageURLs)
		if err != nil {
			t.Fatalf("CreateReport failed: %v", err)
		}

		if len(report.Images) != 2 {
			t.Errorf("Expected 2 images, got %d", len(report.Images))
		}
	})
}

func TestReportService_GetReports(t *testing.T) {
	ctx := context.Background()
	repo := newMockReportRepository()
	service := NewReportService(repo)

	userID := uint64(1)
	_, _ = service.CreateReport(ctx, userID, "displayError", "Report 1", "Content 1", "", nil)
	_, _ = service.CreateReport(ctx, userID, "codingError", "Report 2", "Content 2", "", nil)

	t.Run("get all reports for user", func(t *testing.T) {
		reports, total, err := service.GetReports(ctx, userID, 1, 10)
		if err != nil {
			t.Fatalf("GetReports failed: %v", err)
		}

		if len(reports) != 2 {
			t.Errorf("Expected 2 reports, got %d", len(reports))
		}
		if total != 2 {
			t.Errorf("Expected total 2, got %d", total)
		}
	})
}

func TestReportService_GetReport(t *testing.T) {
	ctx := context.Background()
	repo := newMockReportRepository()
	service := NewReportService(repo)

	userID := uint64(1)
	report, _ := service.CreateReport(ctx, userID, "displayError", "Test Report", "Content", "", nil)

	t.Run("successful get", func(t *testing.T) {
		retrieved, err := service.GetReport(ctx, report.ID)
		if err != nil {
			t.Fatalf("GetReport failed: %v", err)
		}

		if retrieved.ID != report.ID {
			t.Errorf("Expected ID %d, got %d", report.ID, retrieved.ID)
		}
	})

	t.Run("get non-existent report", func(t *testing.T) {
		retrieved, err := service.GetReport(ctx, 99999)
		if err != nil {
			t.Fatalf("GetReport failed: %v", err)
		}
		if retrieved != nil {
			t.Error("Expected nil for non-existent report")
		}
	})
}
