package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/levels-service/internal/handler"
	"metarang/levels-service/internal/repository"
	"metarang/levels-service/internal/service"
	"metarang/levels-service/tests/internal/testutil"
	pb "metarang/shared/pb/levels"
)

var errHandler = errors.New("handler test error")

// newDBHandler creates a LevelHandler backed by a real sqlmock DB.
func newDBHandler(db *sql.DB) *handler.LevelHandler {
	svc := service.NewLevelService(
		repository.NewLevelRepository(db, ""),
		repository.NewUserLogRepository(db),
		&testutil.MockCommercialClient{},
	)
	return handler.NewLevelHandler(svc)
}

// newMockHandler creates a LevelHandler backed by mock interfaces.
func newMockHandler(levelRepo *testutil.MockLevelRepository) *handler.LevelHandler {
	svc := service.NewLevelService(levelRepo, &testutil.MockUserLogRepository{}, &testutil.MockCommercialClient{})
	return handler.NewLevelHandler(svc)
}

// ---------------------------------------------------------------------------
// DB-backed tests (sqlmock)
// ---------------------------------------------------------------------------

func TestLevelHandler_GetAllLevels(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := newDBHandler(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg").
			AddRow(2, "Level 2", "level-2", 200, "bg2.jpg", "")
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WillReturnRows(rows)

		resp, err := h.GetAllLevels(ctx, &pb.GetAllLevelsRequest{})
		require.NoError(t, err)
		assert.Len(t, resp.Levels, 2)
		assert.Equal(t, uint64(1), resp.Levels[0].Id)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := newDBHandler(db)
	ctx := context.Background()

	t.Run("ByID_Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		resp, err := h.GetLevel(ctx, &pb.GetLevelRequest{LevelId: 1})
		require.NoError(t, err)
		assert.Equal(t, uint64(1), resp.Level.Id)
	})

	t.Run("BySlug_Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		// FindBySlug internal call to GetLevelGeneralInfo
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		resp, err := h.GetLevel(ctx, &pb.GetLevelRequest{LevelSlug: "level-1"})
		require.NoError(t, err)
		assert.Equal(t, "level-1", resp.Level.Slug)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := h.GetLevel(ctx, &pb.GetLevelRequest{})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("non-existent").WillReturnError(sql.ErrNoRows)

		_, err := h.GetLevel(ctx, &pb.GetLevelRequest{LevelSlug: "non-existent"})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetLevelGeneralInfo(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := newDBHandler(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		// FindBySlug internal GetLevelGeneralInfo
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		// Direct GetLevelGeneralInfo call
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "level_id", "score", "rank", "description", "subcategories",
				"persian_font", "english_font", "file_volume", "used_colors", "points", "lines",
				"has_animation", "designer", "model_designer", "creation_date", "png_file", "fbx_file", "gif_file"}).
				AddRow(1, 1, 100, 1, "Description", 2, "Font1", "Font2", 1.5, "Colors", 100, 200, 1, "Designer", "Model Designer", "2024-01-01", "png", "fbx", "gif"))

		resp, err := h.GetLevelGeneralInfo(ctx, &pb.GetLevelGeneralInfoRequest{LevelSlug: "level-1"})
		require.NoError(t, err)
		assert.Equal(t, "Description", resp.GeneralInfo.Description)
	})

	t.Run("GeneralInfo_NotFound_ReturnsNull", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		// FindBySlug internal call
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		// Direct GetLevelGeneralInfo call
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		resp, err := h.GetLevelGeneralInfo(ctx, &pb.GetLevelGeneralInfoRequest{LevelSlug: "level-1"})
		require.NoError(t, err)
		assert.Nil(t, resp.GeneralInfo)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetLevelGem(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := newDBHandler(db)
	ctx := context.Background()

	t.Run("Gem_NotFound_ReturnsNull", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		resp, err := h.GetLevelGem(ctx, &pb.GetLevelGemRequest{LevelSlug: "level-1"})
		require.NoError(t, err)
		assert.Nil(t, resp.Gem)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetLevelPrizes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := newDBHandler(db)
	ctx := context.Background()

	t.Run("Prize_NotFound_ReturnsNull", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		resp, err := h.GetLevelPrizes(ctx, &pb.GetLevelPrizesRequest{LevelSlug: "level-1"})
		require.NoError(t, err)
		assert.Nil(t, resp.Prize)
	})

	t.Run("ByLevelID", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "level_id", "psc", "blue", "red", "yellow", "effect", "satisfaction", "created_at"}).
				AddRow(1, 1, 1000, 5, 3, 2, 10, 50.75, nil))

		resp, err := h.GetLevelPrizes(ctx, &pb.GetLevelPrizesRequest{LevelId: 1})
		require.NoError(t, err)
		assert.Equal(t, "1000", resp.Prize.Psc)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetUserLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := newDBHandler(db)
	ctx := context.Background()

	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := h.GetUserLevel(ctx, &pb.GetUserLevelRequest{UserId: 0})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_ClaimPrize(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	h := newDBHandler(db)
	ctx := context.Background()

	t.Run("InvalidRequest_MissingUserID", func(t *testing.T) {
		_, err := h.ClaimPrize(ctx, &pb.ClaimPrizeRequest{LevelId: 1})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("InvalidRequest_MissingLevelID", func(t *testing.T) {
		_, err := h.ClaimPrize(ctx, &pb.ClaimPrizeRequest{UserId: 1})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Mock-interface tests – validation and error mapping
// ---------------------------------------------------------------------------

func TestLevelHandler_GetAllLevels_Error(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetAllLevelsFunc: func(ctx context.Context) ([]*pb.Level, error) { return nil, errHandler },
	})
	_, err := h.GetAllLevels(context.Background(), &pb.GetAllLevelsRequest{})
	require.Error(t, err)
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestLevelHandler_GetLevel_InvalidArg(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{})
	_, err := h.GetLevel(context.Background(), &pb.GetLevelRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestLevelHandler_GetLevelGeneralInfo_InvalidArg(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{})
	_, err := h.GetLevelGeneralInfo(context.Background(), &pb.GetLevelGeneralInfoRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestLevelHandler_GetLevelGeneralInfo_Success(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetLevelGeneralInfoFunc: func(ctx context.Context, levelID uint64) (*pb.LevelGeneralInfo, error) {
			return &pb.LevelGeneralInfo{LevelId: levelID}, nil
		},
	})
	resp, err := h.GetLevelGeneralInfo(context.Background(), &pb.GetLevelGeneralInfoRequest{LevelId: 5})
	require.NoError(t, err)
	assert.Equal(t, uint64(5), resp.GeneralInfo.LevelId)
}

func TestLevelHandler_GetLevelGeneralInfo_Error(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetLevelGeneralInfoFunc: func(ctx context.Context, levelID uint64) (*pb.LevelGeneralInfo, error) {
			return nil, errHandler
		},
	})
	_, err := h.GetLevelGeneralInfo(context.Background(), &pb.GetLevelGeneralInfoRequest{LevelId: 5})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.NotFound, st.Code())
}

func TestLevelHandler_GetLevelGem_InvalidArg(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{})
	_, err := h.GetLevelGem(context.Background(), &pb.GetLevelGemRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestLevelHandler_GetLevelGem_Success(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetLevelGemFunc: func(ctx context.Context, levelID uint64) (*pb.LevelGem, error) {
			return &pb.LevelGem{LevelId: levelID}, nil
		},
	})
	resp, err := h.GetLevelGem(context.Background(), &pb.GetLevelGemRequest{LevelId: 3})
	require.NoError(t, err)
	assert.Equal(t, uint64(3), resp.Gem.LevelId)
}

func TestLevelHandler_GetLevelGift_InvalidArg(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{})
	_, err := h.GetLevelGift(context.Background(), &pb.GetLevelGiftRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestLevelHandler_GetLevelGift_Success(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetLevelGiftFunc: func(ctx context.Context, levelID uint64) (*pb.LevelGift, error) {
			return &pb.LevelGift{LevelId: levelID}, nil
		},
	})
	resp, err := h.GetLevelGift(context.Background(), &pb.GetLevelGiftRequest{LevelId: 4})
	require.NoError(t, err)
	assert.Equal(t, uint64(4), resp.Gift.LevelId)
}

func TestLevelHandler_GetLevelLicenses_InvalidArg(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{})
	_, err := h.GetLevelLicenses(context.Background(), &pb.GetLevelLicensesRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestLevelHandler_GetLevelLicenses_Success(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetLevelLicensesFunc: func(ctx context.Context, levelID uint64) (*pb.LevelLicense, error) {
			return &pb.LevelLicense{LevelId: levelID}, nil
		},
	})
	resp, err := h.GetLevelLicenses(context.Background(), &pb.GetLevelLicensesRequest{LevelId: 6})
	require.NoError(t, err)
	assert.Equal(t, uint64(6), resp.Licenses.LevelId)
}

func TestLevelHandler_GetLevelPrizes_InvalidArg(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{})
	_, err := h.GetLevelPrizes(context.Background(), &pb.GetLevelPrizesRequest{})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

func TestLevelHandler_GetLevelPrizes_Success(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
			return &pb.LevelPrize{LevelId: levelID}, nil
		},
	})
	resp, err := h.GetLevelPrizes(context.Background(), &pb.GetLevelPrizesRequest{LevelId: 7})
	require.NoError(t, err)
	assert.Equal(t, uint64(7), resp.Prize.LevelId)
}

func TestLevelHandler_ClaimPrize_Success(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
			return &pb.LevelPrize{Id: 1, Psc: "100", Blue: "1", Red: "1", Yellow: "1", Effect: 1, Satisfaction: "1.0"}, nil
		},
		HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) { return false, nil },
		GetVariableRateFunc:      func(ctx context.Context, name string) (float64, error) { return 100, nil },
		RecordReceivedPrizeFunc:  func(ctx context.Context, userID, prizeID uint64) error { return nil },
	})
	resp, err := h.ClaimPrize(context.Background(), &pb.ClaimPrizeRequest{UserId: 1, LevelId: 2})
	require.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestLevelHandler_ClaimPrize_ServiceError(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
			return nil, errHandler
		},
	})
	_, err := h.ClaimPrize(context.Background(), &pb.ClaimPrizeRequest{UserId: 1, LevelId: 2})
	st, _ := status.FromError(err)
	assert.Equal(t, codes.Internal, st.Code())
}

func TestLevelHandler_GetUserLevel_Success(t *testing.T) {
	h := newMockHandler(&testutil.MockLevelRepository{
		GetUserLatestLevelFunc: func(ctx context.Context, userID uint64) (*pb.Level, error) {
			return nil, errors.New("no level")
		},
	})
	resp, err := h.GetUserLevel(context.Background(), &pb.GetUserLevelRequest{UserId: 1})
	require.NoError(t, err)
	assert.Nil(t, resp.LatestLevel)
}
