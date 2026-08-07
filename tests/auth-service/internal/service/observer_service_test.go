package service_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/pubsub"
	"metarang/auth-service/internal/service"
)

type fakePublisher struct {
	calls []bool
}

func (f *fakePublisher) PublishUserStatusChanged(_ context.Context, _ uint64, online bool) error {
	f.calls = append(f.calls, online)
	return nil
}
func (f *fakePublisher) Close() error { return nil }

var _ pubsub.RedisPublisher = (*fakePublisher)(nil)

func TestObserverService_LogoutCreatedScore(t *testing.T) {
	ctx := context.Background()
	users := newFakeUserRepository(map[uint64]*models.User{
		1: {ID: 1, Email: "a@x.com", IP: "1.1.1.1", Score: 1},
	})
	users.updateFunc = func(_ context.Context, u *models.User) error {
		users.users[u.ID] = u
		return nil
	}
	act := newFakeActivityRepository()
	act.latestActivity[1] = &models.UserActivity{
		ID: 1, UserID: 1, Start: time.Now().Add(-2 * time.Hour),
	}
	act.userLogs[1] = &models.UserLog{UserID: 1, Score: 0}
	pub := &fakePublisher{}

	svc := service.NewObserverService(users, act, pub)

	t.Run("logout", func(t *testing.T) {
		user := users.users[1]
		if err := svc.OnUserLogout(ctx, user, "1.1.1.1", "ua"); err != nil {
			t.Fatal(err)
		}
		if len(pub.calls) == 0 || pub.calls[len(pub.calls)-1] {
			t.Fatalf("expected offline publish, calls=%v", pub.calls)
		}
	})

	t.Run("created", func(t *testing.T) {
		user := &models.User{ID: 2, Email: "b@x.com", IP: "2.2.2.2"}
		users.users[2] = user
		if err := svc.OnUserCreated(ctx, user); err != nil {
			t.Fatal(err)
		}
		if !user.EmailVerifiedAt.Valid && users.users[2].EmailVerifiedAt.Valid {
			// marked via repo
		}
		if act.userLogs[2] == nil {
			t.Fatal("expected user log")
		}
	})

	t.Run("hour reached and score", func(t *testing.T) {
		user := users.users[1]
		if err := svc.OnHourReached(ctx, user); err != nil {
			t.Fatal(err)
		}
		if err := svc.CalculateScore(ctx, user); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("login with publisher", func(t *testing.T) {
		user := users.users[1]
		user.Phone = sql.NullString{String: "0912", Valid: true}
		user.PhoneVerifiedAt = sql.NullTime{Time: time.Now(), Valid: true}
		if err := svc.OnUserLogin(ctx, user, "1.1.1.1", "ua"); err != nil {
			t.Fatal(err)
		}
	})
}
