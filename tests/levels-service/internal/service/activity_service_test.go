package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/levels-service/internal/service"
	"metarang/levels-service/tests/internal/testutil"
	pb "metarang/shared/pb/levels"
)

func newActivitySvc(
	activityRepo *testutil.MockActivityRepository,
	userLogRepo *testutil.MockUserLogRepository,
	levelRepo *testutil.MockLevelRepository,
) *service.ActivityService {
	return service.NewActivityService(activityRepo, userLogRepo, levelRepo, &testutil.MockCommercialClient{})
}

func TestActivityService_LogActivity(t *testing.T) {
	svc := newActivitySvc(
		&testutil.MockActivityRepository{
			CreateActivityFunc: func(ctx context.Context, req *pb.LogActivityRequest) (uint64, error) {
				return 12, nil
			},
			CreateUserEventFunc: func(ctx context.Context, userID uint64, event, ip, device string, status int8) error {
				return nil
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockLevelRepository{},
	)
	id, err := svc.LogActivity(context.Background(), &pb.LogActivityRequest{UserId: 1, EventType: "login"})
	require.NoError(t, err)
	assert.Equal(t, uint64(12), id)
}

func TestActivityService_LogActivity_Error(t *testing.T) {
	svc := newActivitySvc(
		&testutil.MockActivityRepository{
			CreateActivityFunc: func(ctx context.Context, req *pb.LogActivityRequest) (uint64, error) {
				return 0, errors.New("db error")
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockLevelRepository{},
	)
	_, err := svc.LogActivity(context.Background(), &pb.LogActivityRequest{UserId: 1, EventType: "login"})
	assert.Error(t, err)
}

func TestActivityService_UpdateActivityScore_LevelUp(t *testing.T) {
	addCalls := 0
	recorded := false
	attached := false

	svc := service.NewActivityService(
		&testutil.MockActivityRepository{
			GetVariableRateFunc: func(ctx context.Context, name string) (float64, error) { return 100, nil },
		},
		&testutil.MockUserLogRepository{
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) { return 250, nil },
			UpdateScoreFunc:    func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return &pb.Level{Id: 3}, nil
			},
			AttachLevelToUserFunc: func(ctx context.Context, userID, levelID uint64) error {
				attached = true
				return nil
			},
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 9, Psc: "1000", Blue: "1", Red: "1", Yellow: "1", Effect: 1, Satisfaction: "1.0"}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) { return false, nil },
			RecordReceivedPrizeFunc:  func(ctx context.Context, userID, prizeID uint64) error { recorded = true; return nil },
		},
		&testutil.MockCommercialClient{
			AddBalanceFunc: func(ctx context.Context, userID uint64, asset string, amount float64) error {
				addCalls++
				return nil
			},
		},
	)

	_, levelUp, newLevelID, err := svc.UpdateActivityScore(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, levelUp)
	assert.Equal(t, uint64(3), newLevelID)
	assert.True(t, attached)
	assert.True(t, recorded)
	assert.Equal(t, 6, addCalls)
}

func TestActivityService_UpdateActivityScore_CalculateError(t *testing.T) {
	svc := newActivitySvc(
		&testutil.MockActivityRepository{},
		&testutil.MockUserLogRepository{
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) {
				return 0, errors.New("db error")
			},
		},
		&testutil.MockLevelRepository{},
	)
	_, _, _, err := svc.UpdateActivityScore(context.Background(), 1)
	assert.Error(t, err)
}

func TestActivityService_UpdateActivityScore_UpdateScoreError(t *testing.T) {
	svc := newActivitySvc(
		&testutil.MockActivityRepository{},
		&testutil.MockUserLogRepository{
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) { return 100, nil },
			UpdateScoreFunc: func(ctx context.Context, userID uint64, score int32) error {
				return errors.New("update failed")
			},
		},
		&testutil.MockLevelRepository{},
	)
	_, _, _, err := svc.UpdateActivityScore(context.Background(), 1)
	assert.Error(t, err)
}

func TestActivityService_UpdateActivityScore_NoLevelUp(t *testing.T) {
	svc := newActivitySvc(
		&testutil.MockActivityRepository{},
		&testutil.MockUserLogRepository{
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) { return 50, nil },
			UpdateScoreFunc:    func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return nil, nil
			},
		},
	)
	score, levelUp, newLevelID, err := svc.UpdateActivityScore(context.Background(), 1)
	require.NoError(t, err)
	assert.False(t, levelUp)
	assert.Equal(t, uint64(0), newLevelID)
	assert.Equal(t, int32(50), score)
}

func TestActivityService_UpdateActivityScore_AttachError(t *testing.T) {
	svc := newActivitySvc(
		&testutil.MockActivityRepository{},
		&testutil.MockUserLogRepository{
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) { return 200, nil },
			UpdateScoreFunc:    func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return &pb.Level{Id: 3}, nil
			},
			AttachLevelToUserFunc: func(ctx context.Context, userID, levelID uint64) error {
				return errors.New("attach failed")
			},
		},
	)
	_, _, _, err := svc.UpdateActivityScore(context.Background(), 1)
	assert.Error(t, err)
}

func TestActivityService_UpdateActivityScore_PrizeAlreadyReceived(t *testing.T) {
	attached := false
	svc := newActivitySvc(
		&testutil.MockActivityRepository{
			GetVariableRateFunc: func(ctx context.Context, name string) (float64, error) { return 100, nil },
		},
		&testutil.MockUserLogRepository{
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) { return 200, nil },
			UpdateScoreFunc:    func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return &pb.Level{Id: 3}, nil
			},
			AttachLevelToUserFunc: func(ctx context.Context, userID, levelID uint64) error {
				attached = true
				return nil
			},
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 9}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) {
				return true, nil
			},
		},
	)
	_, levelUp, _, err := svc.UpdateActivityScore(context.Background(), 1)
	require.NoError(t, err)
	assert.True(t, attached)
	assert.True(t, levelUp)
}

func TestActivityService_RecordTrade(t *testing.T) {
	updatedCount := ""
	svc := newActivitySvc(
		&testutil.MockActivityRepository{
			GetVariableRateFunc: func(ctx context.Context, name string) (float64, error) { return 30000, nil },
			GetSignificantTradeCountFunc: func(ctx context.Context, userID uint64, minIrrAmount, minPscAmount float64) (int32, error) {
				return 5, nil
			},
		},
		&testutil.MockUserLogRepository{
			UpdateTransactionsCountFunc: func(ctx context.Context, userID uint64, count string) error {
				updatedCount = count
				return nil
			},
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) { return 10, nil },
			UpdateScoreFunc:    func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return nil, nil
			},
		},
	)
	require.NoError(t, svc.RecordTrade(context.Background(), 1, "8000000", "0"))
	assert.Equal(t, "10", updatedCount)
}

func TestActivityService_RecordTrade_NonSignificant(t *testing.T) {
	called := false
	svc := newActivitySvc(
		&testutil.MockActivityRepository{
			GetVariableRateFunc: func(ctx context.Context, name string) (float64, error) { return 30000, nil },
		},
		&testutil.MockUserLogRepository{
			UpdateTransactionsCountFunc: func(ctx context.Context, userID uint64, count string) error {
				called = true
				return nil
			},
		},
		&testutil.MockLevelRepository{},
	)
	require.NoError(t, svc.RecordTrade(context.Background(), 1, "100", "0"))
	assert.False(t, called)
}

func TestActivityService_OtherFlows(t *testing.T) {
	svc := newActivitySvc(
		&testutil.MockActivityRepository{
			FindByUserIDFunc: func(ctx context.Context, userID uint64, limit int32) ([]*pb.UserActivity, error) {
				return []*pb.UserActivity{{Id: 1}}, nil
			},
			GetLatestActivityFunc: func(ctx context.Context, userID uint64) (*pb.UserActivity, error) {
				return &pb.UserActivity{Id: 2, Start: time.Now().Add(-30 * time.Minute).Format(time.RFC3339)}, nil
			},
			UpdateActivityFunc:          func(ctx context.Context, activityID uint64, endTime time.Time, totalMinutes int32) error { return nil },
			GetTotalActivityMinutesFunc: func(ctx context.Context, userID uint64) (int32, error) { return 60, nil },
		},
		&testutil.MockUserLogRepository{
			GetUserLogFunc:           func(ctx context.Context, userID uint64) (*pb.UserLog, error) { return &pb.UserLog{UserId: userID}, nil },
			IncrementDepositFunc:     func(ctx context.Context, userID uint64, amount string) error { return nil },
			GetTotalFollowersFunc:    func(ctx context.Context, userID uint64) (int32, error) { return 9, nil },
			UpdateFollowersCountFunc: func(ctx context.Context, userID uint64, totalFollowers int32) error { return nil },
			UpdateActivityHoursFunc:  func(ctx context.Context, userID uint64, totalMinutes int32) error { return nil },
			CalculateScoreFunc:       func(ctx context.Context, userID uint64) (int32, error) { return 1, nil },
			UpdateScoreFunc:          func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return nil, nil
			},
		},
	)

	_, _, err := svc.GetUserActivities(context.Background(), 1, 5)
	require.NoError(t, err)
	require.NoError(t, svc.RecordDeposit(context.Background(), 1, "100000"))
	require.NoError(t, svc.RecordFollower(context.Background(), 1))
	require.NoError(t, svc.LogLogout(context.Background(), 1, "127.0.0.1"))
}

func TestActivityService_LogLogout_ParseTimeError(t *testing.T) {
	svc := newActivitySvc(
		&testutil.MockActivityRepository{
			GetLatestActivityFunc: func(ctx context.Context, userID uint64) (*pb.UserActivity, error) {
				return &pb.UserActivity{Id: 1, Start: "not-a-valid-time"}, nil
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockLevelRepository{},
	)
	assert.Error(t, svc.LogLogout(context.Background(), 1, "127.0.0.1"))
}

func TestActivityService_HourReached_Direct(t *testing.T) {
	updated := false
	svc := newActivitySvc(
		&testutil.MockActivityRepository{
			GetTotalActivityMinutesFunc: func(ctx context.Context, userID uint64) (int32, error) { return 120, nil },
			GetVariableRateFunc:         func(ctx context.Context, name string) (float64, error) { return 100, nil },
		},
		&testutil.MockUserLogRepository{
			UpdateActivityHoursFunc: func(ctx context.Context, userID uint64, totalMinutes int32) error {
				updated = true
				return nil
			},
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) { return 10, nil },
			UpdateScoreFunc:    func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return nil, nil
			},
		},
	)
	require.NoError(t, svc.HourReached(context.Background(), 1))
	assert.True(t, updated)
}
