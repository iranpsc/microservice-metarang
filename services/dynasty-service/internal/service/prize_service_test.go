package service

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metargb/dynasty-service/internal/repository"
)

func TestPrizeService_GetUserReceivedPrizes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	prizeRepo := repository.NewPrizeRepository(db)
	service := NewPrizeService(prizeRepo)

	ctx := context.Background()
	userID := uint64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT rp.id, rp.user_id").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"rp.id", "rp.user_id", "rp.prize_id", "rp.message", "rp.created_at", "rp.updated_at", "dp.id", "dp.member", "dp.satisfaction", "dp.introduction_profit_increase", "dp.accumulated_capital_reserve", "dp.data_storage", "dp.psc"}).
				AddRow(1, userID, 1, "Congratulations!", time.Now(), time.Now(), 1, "offspring", 0.1, 0.05, 0.02, 0.03, 1000))

		prizes, total, err := service.GetUserReceivedPrizes(ctx, userID, 1, 10)
		require.NoError(t, err)
		assert.Equal(t, int32(1), total)
		assert.Len(t, prizes, 1)
		if len(prizes) > 0 {
			assert.Equal(t, userID, prizes[0].UserID)
			assert.NotNil(t, prizes[0].Prize)
		}
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPrizeService_ClaimPrize(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	prizeRepo := repository.NewPrizeRepository(db)
	service := NewPrizeService(prizeRepo)

	ctx := context.Background()
	receivedPrizeID := uint64(1)
	userID := uint64(1)

	t.Run("Success", func(t *testing.T) {
		// Get received prize
		mock.ExpectQuery("SELECT rp.id, rp.user_id").
			WithArgs(receivedPrizeID).
			WillReturnRows(sqlmock.NewRows([]string{"rp.id", "rp.user_id", "rp.prize_id", "rp.message", "rp.created_at", "rp.updated_at", "dp.member", "dp.satisfaction", "dp.introduction_profit_increase", "dp.accumulated_capital_reserve", "dp.data_storage", "dp.psc"}).
				AddRow(receivedPrizeID, userID, 1, "Congratulations!", time.Now(), time.Now(), "offspring", 0.1, 0.05, 0.02, 0.03, 1000))

		// Delete received prize
		mock.ExpectExec("DELETE FROM received_prizes").
			WithArgs(receivedPrizeID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := service.ClaimPrize(ctx, receivedPrizeID, userID)
		require.NoError(t, err)
	})

	t.Run("Unauthorized", func(t *testing.T) {
		otherUserID := uint64(2)
		mock.ExpectQuery("SELECT rp.id, rp.user_id").
			WithArgs(receivedPrizeID).
			WillReturnRows(sqlmock.NewRows([]string{"rp.id", "rp.user_id", "rp.prize_id", "rp.message", "rp.created_at", "rp.updated_at", "dp.member", "dp.satisfaction", "dp.introduction_profit_increase", "dp.accumulated_capital_reserve", "dp.data_storage", "dp.psc"}).
				AddRow(receivedPrizeID, userID, 1, "Congratulations!", time.Now(), time.Now(), "offspring", 0.1, 0.05, 0.02, 0.03, 1000))

		err := service.ClaimPrize(ctx, receivedPrizeID, otherUserID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unauthorized")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}
