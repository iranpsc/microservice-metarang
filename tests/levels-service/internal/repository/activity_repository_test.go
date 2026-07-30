package repository_test

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "metarang/shared/pb/levels"

	"metarang/levels-service/internal/repository"
)

func TestActivityRepository_CreateActivity(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO user_activities (user_id, start, ip, created_at, updated_at) VALUES (?, NOW(), ?, NOW(), NOW())",
	)).WithArgs(uint64(10), "127.0.0.1").WillReturnResult(sqlmock.NewResult(99, 1))

	id, err := repo.CreateActivity(ctx, &pb.LogActivityRequest{UserId: 10, Ip: "127.0.0.1"})
	require.NoError(t, err)
	assert.Equal(t, uint64(99), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestActivityRepository_FindByUserID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()
	start := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2024, 6, 1, 11, 0, 0, 0, time.UTC)

	t.Run("WithLimit", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "user_id", "start", "end", "total", "ip"}).
			AddRow(1, 5, start, end, 60, "1.2.3.4")
		mock.ExpectQuery("SELECT id, user_id, start, end, COALESCE\\(total, 0\\) as total, ip").
			WithArgs(uint64(5), int32(10)).
			WillReturnRows(rows)

		activities, err := repo.FindByUserID(ctx, 5, 10)
		require.NoError(t, err)
		require.Len(t, activities, 1)
		assert.Equal(t, uint64(1), activities[0].Id)
		assert.Equal(t, start.Format(time.RFC3339), activities[0].Start)
		assert.Equal(t, end.Format(time.RFC3339), activities[0].End)
	})

	t.Run("NoLimit", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "user_id", "start", "end", "total", "ip"}).
			AddRow(2, 5, start, sql.NullTime{}, 0, "1.2.3.4")
		mock.ExpectQuery("SELECT id, user_id, start, end, COALESCE\\(total, 0\\) as total, ip").
			WithArgs(uint64(5)).
			WillReturnRows(rows)

		activities, err := repo.FindByUserID(ctx, 5, 0)
		require.NoError(t, err)
		require.Len(t, activities, 1)
		assert.Empty(t, activities[0].End)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestActivityRepository_GetLatestActivity(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()
	start := time.Date(2024, 6, 1, 10, 0, 0, 0, time.UTC)

	t.Run("Found", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, user_id, start, end, COALESCE\\(total, 0\\) as total, ip").
			WithArgs(uint64(7)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "start", "end", "total", "ip"}).
				AddRow(3, 7, start, sql.NullTime{}, 30, "10.0.0.1"))

		activity, err := repo.GetLatestActivity(ctx, 7)
		require.NoError(t, err)
		require.NotNil(t, activity)
		assert.Equal(t, uint64(3), activity.Id)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, user_id, start, end, COALESCE\\(total, 0\\) as total, ip").
			WithArgs(uint64(999)).
			WillReturnError(sql.ErrNoRows)

		activity, err := repo.GetLatestActivity(ctx, 999)
		assert.ErrorIs(t, err, sql.ErrNoRows)
		assert.Nil(t, activity)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestActivityRepository_UpdateActivity(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()
	endTime := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

	mock.ExpectExec(regexp.QuoteMeta(
		"UPDATE user_activities SET end = ?, total = ?, updated_at = NOW() WHERE id = ?",
	)).WithArgs(endTime, int32(120), uint64(5)).WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateActivity(ctx, 5, endTime, 120)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestActivityRepository_GetTotalActivityMinutes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT COALESCE(SUM(total), 0) FROM user_activities WHERE user_id = ?")).
		WithArgs(uint64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(180))

	total, err := repo.GetTotalActivityMinutes(ctx, 3)
	require.NoError(t, err)
	assert.Equal(t, int32(180), total)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestActivityRepository_CreateUserEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()

	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO user_events (user_id, event, ip, device, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, NOW(), NOW())",
	)).WithArgs(uint64(1), "login", "127.0.0.1", "web", int8(1)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.CreateUserEvent(ctx, 1, "login", "127.0.0.1", "web", 1)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestActivityRepository_GetVariableRate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM system_variables WHERE name = ? LIMIT 1")).
			WithArgs("trade_rate").
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("1.5"))

		rate, err := repo.GetVariableRate(ctx, "trade_rate")
		require.NoError(t, err)
		assert.Equal(t, 1.5, rate)
	})

	t.Run("ParseError", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM system_variables WHERE name = ? LIMIT 1")).
			WithArgs("bad_rate").
			WillReturnRows(sqlmock.NewRows([]string{"value"}).AddRow("not-a-number"))

		_, err := repo.GetVariableRate(ctx, "bad_rate")
		assert.Error(t, err)
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta("SELECT value FROM system_variables WHERE name = ? LIMIT 1")).
			WithArgs("missing").
			WillReturnError(sql.ErrNoRows)

		_, err := repo.GetVariableRate(ctx, "missing")
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestActivityRepository_GetSignificantTradeCount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewActivityRepository(db)
	ctx := context.Background()

	mock.ExpectQuery("SELECT COUNT\\(\\*\\)").
		WithArgs(uint64(8), 1000.0, 50.0, uint64(8), 1000.0, 50.0).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.GetSignificantTradeCount(ctx, 8, 1000, 50)
	require.NoError(t, err)
	assert.Equal(t, int32(3), count)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestActivityRepository_CreateActivity_DBError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := repository.NewActivityRepository(db)
	mock.ExpectExec("INSERT INTO user_activities").
		WillReturnError(errors.New("db error"))

	_, err = repo.CreateActivity(context.Background(), &pb.LogActivityRequest{UserId: 1, Ip: "x"})
	assert.Error(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
