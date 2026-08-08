package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/dynasty-service/internal/repository"
	"metarang/dynasty-service/internal/service"
)

func TestJoinRequestServiceEnhanced_SendJoinRequest_UsesMessageRepositoryTemplates(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewJoinRequestServiceEnhanced(
		repository.NewJoinRequestRepository(db),
		repository.NewDynastyRepository(db),
		repository.NewFamilyRepository(db),
		repository.NewValidationRepository(db),
		repository.NewMessageRepository(db),
		repository.NewPrizeRepository(db),
	)

	fromUserID, toUserID := uint64(1), uint64(2)
	now := time.Now()

	mock.ExpectQuery("SELECT TIMESTAMPDIFF").
		WithArgs(fromUserID).
		WillReturnRows(sqlmock.NewRows([]string{"is_under_18"}).AddRow(false))
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(fromUserID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(fromUserID, toUserID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(fromUserID, toUserID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT EXISTS").
		WithArgs(toUserID).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))
	mock.ExpectQuery("SELECT id, user_id, feature_id").
		WithArgs(fromUserID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "feature_id", "created_at", "updated_at"}).
			AddRow(10, fromUserID, 100, now, now))
	mock.ExpectQuery("SELECT id FROM families").
		WithArgs(uint64(10)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(20))
	mock.ExpectQuery("SELECT message FROM dynasty_messages").
		WithArgs("requester_confirmation_message").
		WillReturnRows(sqlmock.NewRows([]string{"message"}).AddRow("sent [relationship] to [reciever-code]"))
	mock.ExpectQuery("SELECT message FROM dynasty_messages").
		WithArgs("reciever_message").
		WillReturnRows(sqlmock.NewRows([]string{"message"}).AddRow("from [sender-code] as [relationship]"))
	mock.ExpectExec("INSERT INTO join_requests").
		WithArgs(fromUserID, toUserID, 0, "brother", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req, senderMsg, receiverMsg, err := svc.SendJoinRequest(context.Background(), fromUserID, toUserID, "brother", nil)
	require.NoError(t, err)
	require.NotNil(t, req)
	assert.Contains(t, senderMsg, "برادر")
	assert.Contains(t, senderMsg, "USER-2")
	assert.Contains(t, receiverMsg, "USER-1")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestJoinRequestServiceEnhanced_RejectJoinRequest_PrepareRejectMessages(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := service.NewJoinRequestServiceEnhanced(
		repository.NewJoinRequestRepository(db),
		repository.NewDynastyRepository(db),
		repository.NewFamilyRepository(db),
		repository.NewValidationRepository(db),
		repository.NewMessageRepository(db),
		repository.NewPrizeRepository(db),
	)

	requestID, fromUserID, toUserID := uint64(1), uint64(10), uint64(20)
	now := time.Now()
	mock.ExpectQuery("SELECT id, from_user, to_user").
		WithArgs(requestID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_user", "to_user", "status", "relationship", "message", "created_at", "updated_at"}).
			AddRow(requestID, fromUserID, toUserID, 0, "sister", nil, now, now))
	mock.ExpectExec("UPDATE join_requests SET status").
		WithArgs(-1, requestID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	require.NoError(t, svc.RejectJoinRequest(context.Background(), requestID, toUserID))
	require.NoError(t, mock.ExpectationsWereMet())
}
