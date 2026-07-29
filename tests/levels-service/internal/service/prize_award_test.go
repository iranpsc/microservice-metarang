package service_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/levels-service/internal/service"
	"metarang/levels-service/tests/internal/testutil"
	pb "metarang/shared/pb/levels"
)

func newPrizeTestSvc(repo *testutil.MockLevelRepository, cc *testutil.MockCommercialClient) *service.LevelService {
	return service.NewLevelService(repo, &testutil.MockUserLogRepository{}, cc)
}

func TestClaimPrize_InvalidPscString(t *testing.T) {
	svc := newPrizeTestSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 1, Psc: "not-a-number", Blue: "1", Red: "1", Yellow: "1", Effect: 1, Satisfaction: "1.0"}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) { return false, nil },
			GetVariableRateFunc:      func(ctx context.Context, name string) (float64, error) { return 100, nil },
		},
		&testutil.MockCommercialClient{},
	)
	assert.Error(t, svc.ClaimPrize(context.Background(), 1, 2))
}

func TestClaimPrize_InvalidBlueString(t *testing.T) {
	svc := newPrizeTestSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 1, Psc: "100", Blue: "bad", Red: "1", Yellow: "1", Effect: 1, Satisfaction: "1.0"}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) { return false, nil },
			GetVariableRateFunc:      func(ctx context.Context, name string) (float64, error) { return 100, nil },
		},
		&testutil.MockCommercialClient{},
	)
	assert.Error(t, svc.ClaimPrize(context.Background(), 1, 2))
}

func TestClaimPrize_InvalidSatisfactionString(t *testing.T) {
	svc := newPrizeTestSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 1, Psc: "100", Blue: "1", Red: "1", Yellow: "1", Effect: 1, Satisfaction: "bad"}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) { return false, nil },
			GetVariableRateFunc:      func(ctx context.Context, name string) (float64, error) { return 100, nil },
		},
		&testutil.MockCommercialClient{},
	)
	assert.Error(t, svc.ClaimPrize(context.Background(), 1, 2))
}

func TestClaimPrize_AddBalanceError(t *testing.T) {
	svc := newPrizeTestSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 1, Psc: "100", Blue: "1", Red: "1", Yellow: "1", Effect: 1, Satisfaction: "1.0"}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) { return false, nil },
			GetVariableRateFunc:      func(ctx context.Context, name string) (float64, error) { return 100, nil },
		},
		&testutil.MockCommercialClient{
			AddBalanceFunc: func(ctx context.Context, userID uint64, asset string, amount float64) error {
				return errTest
			},
		},
	)
	assert.Error(t, svc.ClaimPrize(context.Background(), 1, 2))
}

func TestClaimPrize_Success_AllAssets(t *testing.T) {
	addCalls := 0
	svc := newPrizeTestSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 1, Psc: "1000", Blue: "2", Red: "3", Yellow: "4", Effect: 5, Satisfaction: "1.50"}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) { return false, nil },
			GetVariableRateFunc:      func(ctx context.Context, name string) (float64, error) { return 100, nil },
			RecordReceivedPrizeFunc:  func(ctx context.Context, userID, prizeID uint64) error { return nil },
		},
		&testutil.MockCommercialClient{
			AddBalanceFunc: func(ctx context.Context, userID uint64, asset string, amount float64) error {
				addCalls++
				return nil
			},
		},
	)
	require.NoError(t, svc.ClaimPrize(context.Background(), 1, 2))
	assert.Equal(t, 6, addCalls)
}
