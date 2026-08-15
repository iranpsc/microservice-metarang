package service_test

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/notifications-service/internal/errs"
	"metarang/notifications-service/internal/models"
	"metarang/notifications-service/internal/repository"
	"metarang/notifications-service/internal/service"
	"metarang/notifications-service/tests/internal/testutil"
)

func TestNewEmailChannel_SendEmailNotImplemented(t *testing.T) {
	ch := service.NewEmailChannel()
	_, err := ch.SendEmail(context.Background(), models.EmailPayload{To: "a@b.com", Subject: "s", Body: "b"})
	assert.ErrorIs(t, err, errs.ErrNotImplemented)
}

func TestNewSMSService_NilChannelSendOTP(t *testing.T) {
	svc := service.NewSMSService(nil)
	_, err := svc.SendOTP(context.Background(), models.OTPPayload{Phone: "09120000000", Code: "1234"})
	assert.ErrorIs(t, err, errs.ErrNotImplemented)
}

func TestNotificationService_SendNotification_NilChannelsSkipDelivery(t *testing.T) {
	ctx := context.Background()

	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewNotificationService(repository.NewNotificationRepository(db), nil, nil)
	expectCreateNotification(mock)

	result, err := svc.SendNotification(ctx, service.SendNotificationInput{
		UserID: 123, Type: "system", Title: "Test", Message: "Message",
		SendSMS: true, SMSPayload: &models.SMSPayload{Phone: "09120000000", Message: "hi"},
		SendEmail: true, EmailPayload: &models.EmailPayload{To: "a@b.com", Subject: "s", Body: "b"},
	})
	require.NoError(t, err)
	assert.True(t, result.Sent)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNotificationService_SendNotification_SkipsNilPayloads(t *testing.T) {
	ctx := context.Background()

	t.Run("SendSMS true with nil payload still sent", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		smsCalled := false
		svc := newNotificationService(db, &testutil.MockSMSChannel{
			SendSMSFunc: func(context.Context, models.SMSPayload) (string, error) {
				smsCalled = true
				return "sms", nil
			},
		}, &testutil.MockEmailChannel{})
		expectCreateNotification(mock)

		result, err := svc.SendNotification(ctx, service.SendNotificationInput{
			UserID: 123, Type: "system", Title: "Test", Message: "Message",
			SendSMS: true, SMSPayload: nil,
		})
		require.NoError(t, err)
		assert.True(t, result.Sent)
		assert.False(t, smsCalled)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SendEmail true with nil payload still sent", func(t *testing.T) {
		db, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer db.Close()

		emailCalled := false
		svc := newNotificationService(db, &testutil.MockSMSChannel{}, &testutil.MockEmailChannel{
			SendEmailFunc: func(context.Context, models.EmailPayload) (string, error) {
				emailCalled = true
				return "email", nil
			},
		})
		expectCreateNotification(mock)

		result, err := svc.SendNotification(ctx, service.SendNotificationInput{
			UserID: 123, Type: "system", Title: "Test", Message: "Message",
			SendEmail: true, EmailPayload: nil,
		})
		require.NoError(t, err)
		assert.True(t, result.Sent)
		assert.False(t, emailCalled)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
