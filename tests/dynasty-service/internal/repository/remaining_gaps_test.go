package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/dynasty-service/internal/models"
	"metarang/dynasty-service/internal/repository"
)

func TestRemainingRepositoryGaps(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	ctx := context.Background()
	now := time.Now()

	familyRepo := repository.NewFamilyRepository(db)
	mock.ExpectQuery("SELECT id, family_id, user_id").
		WithArgs(uint64(1), uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "family_id", "user_id", "relationship", "created_at", "updated_at"}).
			AddRow(9, 2, 1, "brother", now, now))
	member, err := familyRepo.FindMemberByUserAndFamily(ctx, 1, 2)
	require.NoError(t, err)
	require.NotNil(t, member)
	assert.Equal(t, "brother", member.Relationship)

	joinRepo := repository.NewJoinRequestRepository(db)
	mock.ExpectQuery("SELECT id, from_user, to_user").
		WithArgs(uint64(1), uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "from_user", "to_user", "status", "relationship", "message", "created_at", "updated_at"}).
			AddRow(3, 1, 2, 0, "sister", nil, now, now))
	latest, err := joinRepo.GetLatestRequest(ctx, 1, 2)
	require.NoError(t, err)
	require.NotNil(t, latest)

	permRepo := repository.NewPermissionRepository(db)
	mock.ExpectExec("UPDATE children_permissions SET verified").
		WithArgs(uint64(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, permRepo.VerifyPermissions(ctx, 4))

	perm := &models.ChildPermission{
		UserID: 4, Verified: true, BFR: true, SF: true, W: true, JU: true, DM: true,
		PIUP: true, PITC: true, PIC: true, ESOO: true, COTB: true,
	}
	mock.ExpectExec("UPDATE children_permissions").
		WithArgs(true, true, true, true, true, true, true, true, true, true, true, uint64(4)).
		WillReturnResult(sqlmock.NewResult(0, 1))
	require.NoError(t, permRepo.UpdateAllPermissions(ctx, perm))

	prizeRepo := repository.NewPrizeRepository(db)
	mock.ExpectQuery("SELECT COUNT").
		WithArgs(uint64(1), uint64(2)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	claimed, err := prizeRepo.CheckPrizeClaimed(ctx, 1, 2)
	require.NoError(t, err)
	assert.False(t, claimed)

	mock.ExpectExec("INSERT INTO received_prizes").
		WithArgs(uint64(1), uint64(2)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	require.NoError(t, prizeRepo.ClaimPrize(ctx, 1, 2))

	require.NoError(t, mock.ExpectationsWereMet())
}
