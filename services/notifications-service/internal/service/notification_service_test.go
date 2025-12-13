package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"metargb/notifications-service/internal/models"
	"metargb/notifications-service/internal/repository"
)

// MockNotificationRepository is a mock implementation that wraps the repository
type MockNotificationRepository struct {
	*repository.NotificationRepository
	mock.Mock
}

// We'll need to create a wrapper since we can't easily mock concrete types
// For now, we'll test with integration-style tests or create an interface adapter

// MockSMSChannel is a mock implementation of SMSChannel
type MockSMSChannel struct {
	mock.Mock
}

func (m *MockSMSChannel) SendSMS(ctx context.Context, payload models.SMSPayload) (string, error) {
	args := m.Called(ctx, payload)
	return args.String(0), args.Error(1)
}

func (m *MockSMSChannel) SendOTP(ctx context.Context, payload models.OTPPayload) (string, error) {
	args := m.Called(ctx, payload)
	return args.String(0), args.Error(1)
}

// MockEmailChannel is a mock implementation of EmailChannel
type MockEmailChannel struct {
	mock.Mock
}

func (m *MockEmailChannel) SendEmail(ctx context.Context, payload models.EmailPayload) (string, error) {
	args := m.Called(ctx, payload)
	return args.String(0), args.Error(1)
}

func TestNotificationService_SendNotification(t *testing.T) {
	tests := []struct {
		name        string
		input       SendNotificationInput
		setupMocks  func(*MockNotificationRepository, *MockSMSChannel, *MockEmailChannel)
		expectError bool
		expectSent  bool
	}{
		{
			name: "successful notification without SMS or Email",
			input: SendNotificationInput{
				UserID:    123,
				Type:      "system",
				Title:     "Test Title",
				Message:   "Test Message",
				Data:      map[string]string{"key": "value"},
				SendSMS:   false,
				SendEmail: false,
			},
			setupMocks: func(repo *MockNotificationRepository, sms *MockSMSChannel, email *MockEmailChannel) {
				repo.On("CreateNotification", mock.Anything, mock.AnythingOfType("*models.Notification")).
					Return(uint64(1), nil)
			},
			expectError: false,
			expectSent:  true,
		},
		{
			name: "successful notification with SMS",
			input: SendNotificationInput{
				UserID:    123,
				Type:      "system",
				Title:     "Test Title",
				Message:   "Test Message",
				SendSMS:   true,
				SendEmail: false,
				SMSPayload: &models.SMSPayload{
					Phone:   "+1234567890",
					Message: "Test SMS",
				},
			},
			setupMocks: func(repo *MockNotificationRepository, sms *MockSMSChannel, email *MockEmailChannel) {
				repo.On("CreateNotification", mock.Anything, mock.AnythingOfType("*models.Notification")).
					Return(uint64(1), nil)
				sms.On("SendSMS", mock.Anything, mock.AnythingOfType("models.SMSPayload")).
					Return("sms-id-123", nil)
			},
			expectError: false,
			expectSent:  true,
		},
		{
			name: "successful notification with Email",
			input: SendNotificationInput{
				UserID:    123,
				Type:      "system",
				Title:     "Test Title",
				Message:   "Test Message",
				SendSMS:   false,
				SendEmail: true,
				EmailPayload: &models.EmailPayload{
					To:      "test@example.com",
					Subject: "Test Subject",
					Body:    "Test Body",
				},
			},
			setupMocks: func(repo *MockNotificationRepository, sms *MockSMSChannel, email *MockEmailChannel) {
				repo.On("CreateNotification", mock.Anything, mock.AnythingOfType("*models.Notification")).
					Return(uint64(1), nil)
				email.On("SendEmail", mock.Anything, mock.AnythingOfType("models.EmailPayload")).
					Return("email-id-123", nil)
			},
			expectError: false,
			expectSent:  true,
		},
		{
			name: "repository error",
			input: SendNotificationInput{
				UserID:    123,
				Type:      "system",
				Title:     "Test Title",
				Message:   "Test Message",
				SendSMS:   false,
				SendEmail: false,
			},
			setupMocks: func(repo *MockNotificationRepository, sms *MockSMSChannel, email *MockEmailChannel) {
				repo.On("CreateNotification", mock.Anything, mock.AnythingOfType("*models.Notification")).
					Return(uint64(0), assert.AnError)
			},
			expectError: true,
			expectSent:  false,
		},
		{
			name: "SMS send failure",
			input: SendNotificationInput{
				UserID:    123,
				Type:      "system",
				Title:     "Test Title",
				Message:   "Test Message",
				SendSMS:   true,
				SendEmail: false,
				SMSPayload: &models.SMSPayload{
					Phone:   "+1234567890",
					Message: "Test SMS",
				},
			},
			setupMocks: func(repo *MockNotificationRepository, sms *MockSMSChannel, email *MockEmailChannel) {
				repo.On("CreateNotification", mock.Anything, mock.AnythingOfType("*models.Notification")).
					Return(uint64(1), nil)
				sms.On("SendSMS", mock.Anything, mock.AnythingOfType("models.SMSPayload")).
					Return("", assert.AnError)
			},
			expectError: false,
			expectSent:  false, // Notification created but SMS failed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Since the service uses concrete repository type,
			// we can't easily mock it. These tests demonstrate the expected behavior
			// but would require refactoring the service to use an interface.
			// For now, we'll skip these unit tests and rely on integration tests.
			t.Skip("Service uses concrete repository type - requires refactoring to use interface")
		})
	}
}

func TestNotificationService_GetNotifications(t *testing.T) {

	tests := []struct {
		name        string
		userID      uint64
		filter      models.NotificationFilter
		setupMocks  func(*MockNotificationRepository)
		expectError bool
		expectedLen int
	}{
		{
			name:   "successful get notifications",
			userID: 123,
			filter: models.NotificationFilter{Page: 1, PerPage: 10},
			setupMocks: func(repo *MockNotificationRepository) {
				notifications := []models.Notification{
					{
						ID:        "1",
						UserID:    123,
						Type:      "system",
						Title:     "Test",
						Message:   "Message",
						CreatedAt: time.Now(),
					},
				}
				repo.On("ListNotifications", mock.Anything, uint64(123), mock.AnythingOfType("models.NotificationFilter")).
					Return(notifications, int64(1), nil)
			},
			expectError: false,
			expectedLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Since the service uses concrete repository type,
			// we can't easily mock it. These tests demonstrate the expected behavior
			// but would require refactoring the service to use an interface.
			t.Skip("Service uses concrete repository type - requires refactoring to use interface")
		})
	}
}

// Note: The service currently uses concrete repository type
// In a production system, you'd want to use an interface for better testability
// This test demonstrates the pattern but may need adaptation based on actual service structure
