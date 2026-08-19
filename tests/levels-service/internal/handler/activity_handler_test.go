package handler_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/levels-service/internal/handler"
	"metarang/levels-service/internal/service"
	"metarang/levels-service/tests/internal/testutil"
	pb "metarang/shared/pb/levels"
)

func newActivityHandler(
	activityRepo *testutil.MockActivityRepository,
	userLogRepo *testutil.MockUserLogRepository,
	levelRepo *testutil.MockLevelRepository,
) *handler.ActivityHandler {
	svc := service.NewActivityService(activityRepo, userLogRepo, levelRepo, &testutil.MockCommercialClient{})
	return handler.NewActivityHandler(svc)
}

func TestActivityHandler_LogActivity_MissingUserID(t *testing.T) {
	h := newActivityHandler(&testutil.MockActivityRepository{}, &testutil.MockUserLogRepository{}, &testutil.MockLevelRepository{})
	_, err := h.LogActivity(context.Background(), &pb.LogActivityRequest{EventType: "login"})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestActivityHandler_LogActivity_MissingEventType(t *testing.T) {
	h := newActivityHandler(&testutil.MockActivityRepository{}, &testutil.MockUserLogRepository{}, &testutil.MockLevelRepository{})
	_, err := h.LogActivity(context.Background(), &pb.LogActivityRequest{UserId: 1})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestActivityHandler_LogActivity_Success(t *testing.T) {
	h := newActivityHandler(
		&testutil.MockActivityRepository{
			CreateActivityFunc: func(ctx context.Context, req *pb.LogActivityRequest) (uint64, error) { return 5, nil },
			CreateUserEventFunc: func(ctx context.Context, userID uint64, event, ip, device string, s int8) error {
				return nil
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockLevelRepository{},
	)
	resp, err := h.LogActivity(context.Background(), &pb.LogActivityRequest{UserId: 1, EventType: "login"})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, uint64(5), resp.ActivityId)
}

func TestActivityHandler_LogActivity_ServiceError(t *testing.T) {
	h := newActivityHandler(
		&testutil.MockActivityRepository{
			CreateActivityFunc: func(ctx context.Context, req *pb.LogActivityRequest) (uint64, error) {
				return 0, errHandler
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockLevelRepository{},
	)
	_, err := h.LogActivity(context.Background(), &pb.LogActivityRequest{UserId: 1, EventType: "login"})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestActivityHandler_GetUserActivities_MissingUserID(t *testing.T) {
	h := newActivityHandler(&testutil.MockActivityRepository{}, &testutil.MockUserLogRepository{}, &testutil.MockLevelRepository{})
	_, err := h.GetUserActivities(context.Background(), &pb.GetUserActivitiesRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestActivityHandler_GetUserActivities_Success(t *testing.T) {
	h := newActivityHandler(
		&testutil.MockActivityRepository{
			FindByUserIDFunc: func(ctx context.Context, userID uint64, limit int32) ([]*pb.UserActivity, error) {
				return []*pb.UserActivity{{Id: 1}}, nil
			},
		},
		&testutil.MockUserLogRepository{
			GetUserLogFunc: func(ctx context.Context, userID uint64) (*pb.UserLog, error) {
				return &pb.UserLog{UserId: userID}, nil
			},
		},
		&testutil.MockLevelRepository{},
	)
	resp, err := h.GetUserActivities(context.Background(), &pb.GetUserActivitiesRequest{UserId: 1, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, resp.Activities, 1)
}

func TestActivityHandler_GetUserActivities_Error(t *testing.T) {
	h := newActivityHandler(
		&testutil.MockActivityRepository{
			FindByUserIDFunc: func(ctx context.Context, userID uint64, limit int32) ([]*pb.UserActivity, error) {
				return nil, errHandler
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockLevelRepository{},
	)
	_, err := h.GetUserActivities(context.Background(), &pb.GetUserActivitiesRequest{UserId: 1})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestActivityHandler_UpdateActivityScore_MissingUserID(t *testing.T) {
	h := newActivityHandler(&testutil.MockActivityRepository{}, &testutil.MockUserLogRepository{}, &testutil.MockLevelRepository{})
	_, err := h.UpdateActivityScore(context.Background(), &pb.UpdateActivityScoreRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestActivityHandler_UpdateActivityScore_Success(t *testing.T) {
	h := newActivityHandler(
		&testutil.MockActivityRepository{},
		&testutil.MockUserLogRepository{
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) { return 100, nil },
			UpdateScoreFunc:    func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return nil, nil
			},
		},
	)
	resp, err := h.UpdateActivityScore(context.Background(), &pb.UpdateActivityScoreRequest{UserId: 1})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, int32(100), resp.NewScore)
}

func TestActivityHandler_UpdateActivityScore_Error(t *testing.T) {
	h := newActivityHandler(
		&testutil.MockActivityRepository{},
		&testutil.MockUserLogRepository{
			CalculateScoreFunc: func(ctx context.Context, userID uint64) (int32, error) { return 0, errHandler },
		},
		&testutil.MockLevelRepository{},
	)
	_, err := h.UpdateActivityScore(context.Background(), &pb.UpdateActivityScoreRequest{UserId: 1})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestActivityHandler_RecordTrade_MissingUserID(t *testing.T) {
	h := newActivityHandler(&testutil.MockActivityRepository{}, &testutil.MockUserLogRepository{}, &testutil.MockLevelRepository{})
	_, err := h.RecordTrade(context.Background(), &pb.RecordTradeRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestActivityHandler_RecordTrade_Success(t *testing.T) {
	h := newActivityHandler(
		&testutil.MockActivityRepository{
			GetVariableRateFunc: func(ctx context.Context, name string) (float64, error) { return 30000, nil },
			GetSignificantTradeCountFunc: func(ctx context.Context, userID uint64, minIrrAmount, minPscAmount float64) (int32, error) {
				return 3, nil
			},
		},
		&testutil.MockUserLogRepository{
			UpdateTransactionsCountFunc: func(ctx context.Context, userID uint64, count string) error { return nil },
			CalculateScoreFunc:          func(ctx context.Context, userID uint64) (int32, error) { return 1, nil },
			UpdateScoreFunc:             func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return nil, nil
			},
		},
	)
	resp, err := h.RecordTrade(context.Background(), &pb.RecordTradeRequest{UserId: 1, IrrAmount: "8000000", PscAmount: "0"})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestActivityHandler_RecordDeposit_MissingUserID(t *testing.T) {
	h := newActivityHandler(&testutil.MockActivityRepository{}, &testutil.MockUserLogRepository{}, &testutil.MockLevelRepository{})
	_, err := h.RecordDeposit(context.Background(), &pb.RecordDepositRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestActivityHandler_RecordDeposit_Success(t *testing.T) {
	h := newActivityHandler(
		&testutil.MockActivityRepository{},
		&testutil.MockUserLogRepository{
			IncrementDepositFunc: func(ctx context.Context, userID uint64, amount string) error { return nil },
			CalculateScoreFunc:   func(ctx context.Context, userID uint64) (int32, error) { return 1, nil },
			UpdateScoreFunc:      func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return nil, nil
			},
		},
	)
	resp, err := h.RecordDeposit(context.Background(), &pb.RecordDepositRequest{UserId: 1, Amount: "50000"})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestActivityHandler_RecordFollower_MissingUserID(t *testing.T) {
	h := newActivityHandler(&testutil.MockActivityRepository{}, &testutil.MockUserLogRepository{}, &testutil.MockLevelRepository{})
	_, err := h.RecordFollower(context.Background(), &pb.RecordFollowerRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestActivityHandler_RecordFollower_Success(t *testing.T) {
	h := newActivityHandler(
		&testutil.MockActivityRepository{},
		&testutil.MockUserLogRepository{
			GetTotalFollowersFunc:    func(ctx context.Context, userID uint64) (int32, error) { return 5, nil },
			UpdateFollowersCountFunc: func(ctx context.Context, userID uint64, totalFollowers int32) error { return nil },
			CalculateScoreFunc:       func(ctx context.Context, userID uint64) (int32, error) { return 1, nil },
			UpdateScoreFunc:          func(ctx context.Context, userID uint64, score int32) error { return nil },
		},
		&testutil.MockLevelRepository{
			GetNextLevelForScoreFunc: func(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
				return nil, nil
			},
		},
	)
	resp, err := h.RecordFollower(context.Background(), &pb.RecordFollowerRequest{UserId: 1})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}
