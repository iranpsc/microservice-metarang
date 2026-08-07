package service_test

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"

	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
)

type fakePLRepo struct {
	byID     map[uint64]*models.ProfileLimitation
	exists   bool
	createErr error
	updateErr error
	deleteErr error
}

func (f *fakePLRepo) Create(_ context.Context, limitation *models.ProfileLimitation) error {
	if f.createErr != nil {
		return f.createErr
	}
	if limitation.ID == 0 {
		limitation.ID = uint64(len(f.byID) + 1)
	}
	cp := *limitation
	f.byID[limitation.ID] = &cp
	return nil
}

func (f *fakePLRepo) FindByID(_ context.Context, id uint64) (*models.ProfileLimitation, error) {
	if lim, ok := f.byID[id]; ok {
		cp := *lim
		return &cp, nil
	}
	return nil, nil
}

func (f *fakePLRepo) FindByLimiterAndLimited(context.Context, uint64, uint64) (*models.ProfileLimitation, error) {
	return nil, nil
}

func (f *fakePLRepo) FindBetweenUsers(_ context.Context, userID1, userID2 uint64) (*models.ProfileLimitation, error) {
	for _, lim := range f.byID {
		if (lim.LimiterUserID == userID1 && lim.LimitedUserID == userID2) ||
			(lim.LimiterUserID == userID2 && lim.LimitedUserID == userID1) {
			cp := *lim
			return &cp, nil
		}
	}
	return nil, nil
}

func (f *fakePLRepo) Update(_ context.Context, limitation *models.ProfileLimitation) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	cp := *limitation
	f.byID[limitation.ID] = &cp
	return nil
}

func (f *fakePLRepo) Delete(_ context.Context, id uint64) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	delete(f.byID, id)
	return nil
}

func (f *fakePLRepo) ExistsForLimiterAndLimited(context.Context, uint64, uint64) (bool, error) {
	return f.exists, nil
}

func TestProfileLimitationService_Unit(t *testing.T) {
	ctx := context.Background()
	users := newFakeUserRepository(map[uint64]*models.User{
		1: {ID: 1, Name: "a"},
		2: {ID: 2, Name: "b"},
	})
	repo := &fakePLRepo{byID: map[uint64]*models.ProfileLimitation{}}
	svc := service.NewProfileLimitationService(repo, users)
	opts := models.DefaultOptions()
	note := "hello"
	nu := service.NoteUpdate{Present: true, Value: &note}

	t.Run("create success", func(t *testing.T) {
		lim, err := svc.Create(ctx, 1, 2, opts, nu)
		require.NoError(t, err)
		require.Equal(t, uint64(1), lim.ID)
		require.True(t, lim.Note.Valid)
	})

	t.Run("create already exists", func(t *testing.T) {
		repo.exists = true
		_, err := svc.Create(ctx, 1, 2, opts, service.NoteUpdate{})
		require.ErrorIs(t, err, service.ErrProfileLimitationAlreadyExists)
		repo.exists = false
	})

	t.Run("create user not found", func(t *testing.T) {
		_, err := svc.Create(ctx, 1, 99, opts, service.NoteUpdate{})
		require.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("create duplicate key", func(t *testing.T) {
		repo.createErr = &mysql.MySQLError{Number: 1062}
		_, err := svc.Create(ctx, 1, 2, opts, service.NoteUpdate{})
		require.ErrorIs(t, err, service.ErrProfileLimitationAlreadyExists)
		repo.createErr = nil
	})

	t.Run("note too long", func(t *testing.T) {
		longNote := strings.Repeat("a", 501)
		_, err := svc.Create(ctx, 1, 2, opts, service.NoteUpdate{Present: true, Value: &longNote})
		require.ErrorIs(t, err, service.ErrNoteTooLong)
	})

	t.Run("get update delete", func(t *testing.T) {
		repo.byID = map[uint64]*models.ProfileLimitation{
			5: {
				ID: 5, LimiterUserID: 1, LimitedUserID: 2,
				Options: opts, Note: sql.NullString{String: "n", Valid: true},
			},
		}
		got, err := svc.GetByID(ctx, 5)
		require.NoError(t, err)
		require.Equal(t, uint64(5), got.ID)

		_, err = svc.GetByID(ctx, 404)
		require.ErrorIs(t, err, service.ErrProfileLimitationNotFound)

		updated, err := svc.Update(ctx, 5, 1, opts, service.NoteUpdate{Present: true, Value: nil})
		require.NoError(t, err)
		require.False(t, updated.Note.Valid)

		_, err = svc.Update(ctx, 5, 2, opts, service.NoteUpdate{})
		require.ErrorIs(t, err, service.ErrUnauthorized)

		_, err = svc.Update(ctx, 404, 1, opts, service.NoteUpdate{})
		require.ErrorIs(t, err, service.ErrProfileLimitationNotFound)

		require.NoError(t, svc.Delete(ctx, 5, 1))
		require.ErrorIs(t, svc.Delete(ctx, 5, 1), service.ErrProfileLimitationNotFound)

		repo.byID[6] = &models.ProfileLimitation{ID: 6, LimiterUserID: 1, LimitedUserID: 2, Options: opts}
		require.ErrorIs(t, svc.Delete(ctx, 6, 2), service.ErrUnauthorized)
	})

	t.Run("get between users", func(t *testing.T) {
		repo.byID = map[uint64]*models.ProfileLimitation{
			7: {ID: 7, LimiterUserID: 1, LimitedUserID: 2, Options: opts},
		}
		got, err := svc.GetBetweenUsers(ctx, 1, 2)
		require.NoError(t, err)
		require.Equal(t, uint64(7), got.ID)

		_, err = svc.GetBetweenUsers(ctx, 1, 99)
		require.ErrorIs(t, err, service.ErrUserNotFound)
	})

	t.Run("create other error", func(t *testing.T) {
		repo.createErr = errors.New("db")
		_, err := svc.Create(ctx, 1, 2, opts, service.NoteUpdate{})
		require.Error(t, err)
		repo.createErr = nil
	})
}
