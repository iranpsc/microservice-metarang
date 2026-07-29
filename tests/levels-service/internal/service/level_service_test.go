package service_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metarang/levels-service/internal/repository"
	"metarang/levels-service/internal/service"
	"metarang/levels-service/tests/internal/testutil"
	pb "metarang/shared/pb/levels"
)

var errTest = errors.New("test error")

// newLevelSvc builds a LevelService backed by a real sqlmock DB.
func newLevelSvc(db *sql.DB) *service.LevelService {
	return service.NewLevelService(
		repository.NewLevelRepository(db, ""),
		repository.NewUserLogRepository(db),
		&testutil.MockCommercialClient{},
	)
}

// newMockLevelSvc builds a LevelService backed by mock interfaces (no DB).
func newMockLevelSvc(levelRepo *testutil.MockLevelRepository, userLogRepo *testutil.MockUserLogRepository, cc *testutil.MockCommercialClient) *service.LevelService {
	return service.NewLevelService(levelRepo, userLogRepo, cc)
}

// ---------------------------------------------------------------------------
// DB-backed tests (sqlmock) – migrated from the original tests/levels-service
// ---------------------------------------------------------------------------

func TestLevelService_GetAllLevels(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := newLevelSvc(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg").
			AddRow(2, "Level 2", "level-2", 200, "bg2.jpg", "")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WillReturnRows(rows)

		levels, err := svc.GetAllLevels(ctx)
		require.NoError(t, err)
		assert.Len(t, levels, 2)
		assert.Equal(t, uint64(1), levels[0].Id)
		assert.Equal(t, "Level 1", levels[0].Name)
		assert.Equal(t, "level-1", levels[0].Slug)
		assert.Equal(t, int32(100), levels[0].Score)
		assert.Equal(t, "/uploads/img1.jpg", levels[0].ImageUrl)
		assert.Equal(t, "bg1.jpg", levels[0].BackgroundImage)
	})

	t.Run("EmptyResult", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"})
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WillReturnRows(rows)

		levels, err := svc.GetAllLevels(ctx)
		require.NoError(t, err)
		assert.Len(t, levels, 0)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := newLevelSvc(db)
	ctx := context.Background()

	t.Run("ByID_Success", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs(uint64(1)).WillReturnRows(levelRows)
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		level, err := svc.GetLevel(ctx, 1, "")
		require.NoError(t, err)
		assert.NotNil(t, level)
		assert.Equal(t, uint64(1), level.Id)
	})

	t.Run("BySlug_Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		// FindBySlug internal GetLevelGeneralInfo call
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		level, err := svc.GetLevel(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Equal(t, "level-1", level.Slug)
	})

	t.Run("BySlug_NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("non-existent").WillReturnError(sql.ErrNoRows)

		level, err := svc.GetLevel(ctx, 0, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, level)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelGeneralInfo(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := newLevelSvc(db)
	ctx := context.Background()

	generalInfoCols := []string{"id", "level_id", "score", "rank", "description", "subcategories",
		"persian_font", "english_font", "file_volume", "used_colors", "points", "lines",
		"has_animation", "designer", "model_designer", "creation_date", "png_file", "fbx_file", "gif_file"}

	t.Run("BySlug_Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		// FindBySlug internal GetLevelGeneralInfo call
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		// GetLevelGeneralInfo direct call
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows(generalInfoCols).
				AddRow(1, 1, 100, 1, "Description", 2, "Font1", "Font2", 1.5, "Colors", 100, 200, 1, "Designer", "Model Designer", "2024-01-01", "png", "fbx", "gif"))

		info, err := svc.GetLevelGeneralInfo(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, "Description", info.Description)
	})

	t.Run("GeneralInfo_NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		// FindBySlug internal GetLevelGeneralInfo call
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		// GetLevelGeneralInfo direct call
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		info, err := svc.GetLevelGeneralInfo(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, info)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelGem(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := newLevelSvc(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "level_id", "name", "description", "thread", "points", "volume", "color",
				"has_animation", "lines", "png_file", "fbx_file", "encryption", "designer"}).
				AddRow(1, 1, "Gem 1", "Description", "thread1", 50, "vol1", "red", 1, 100, "png", "fbx", 0, "Designer"))

		gem, err := svc.GetLevelGem(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, gem)
		assert.Equal(t, "Gem 1", gem.Name)
	})

	t.Run("Gem_NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		gem, err := svc.GetLevelGem(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, gem)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelGift(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := newLevelSvc(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "level_id", "name", "description", "monthly_capacity_count", "store_capacity",
				"sell_capacity", "features", "sell", "vod_document_registration", "seller_link", "designer",
				"three_d_model_volume", "three_d_model_points", "three_d_model_lines", "has_animation",
				"png_file", "fbx_file", "gif_file", "rent", "vod_count", "start_vod_id", "end_vod_id"}).
				AddRow(1, 1, "Gift 1", "Description", 10, 1, 1, "features", 1, 0, "link", "Designer",
					"vol", 100, 200, 1, "png", "fbx", "gif", 0, 5, "start", "end"))

		gift, err := svc.GetLevelGift(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, gift)
		assert.Equal(t, "Gift 1", gift.Name)
	})

	t.Run("Gift_NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		gift, err := svc.GetLevelGift(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, gift)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelLicenses(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := newLevelSvc(db)
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, level_id, create_union").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "level_id", "create_union", "add_memeber_to_union", "observation_license",
				"gate_license", "lawyer_license", "city_counsile_entry", "establish_special_residential_property",
				"establish_property_on_surface", "judge_entry", "upload_image", "delete_image",
				"inter_level_general_points", "inter_level_special_points", "rent_out_satisfaction",
				"access_to_answer_questions_unit", "create_challenge_questions", "upload_music"}).
				AddRow(1, 1, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1, 1, 0, 1))

		licenses, err := svc.GetLevelLicenses(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, licenses)
		assert.True(t, licenses.CreateUnion)
		assert.False(t, licenses.AddMemeberToUnion)
	})

	t.Run("Licenses_NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, level_id, create_union").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		licenses, err := svc.GetLevelLicenses(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, licenses)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelPrizes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := newLevelSvc(db)
	ctx := context.Background()

	t.Run("BySlug_Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "level_id", "psc", "blue", "red", "yellow", "effect", "satisfaction", "created_at"}).
				AddRow(1, 1, 1000, 5, 3, 2, 10, 50.75, nil))

		prize, err := svc.GetLevelPrizes(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, prize)
		assert.Equal(t, "1000", prize.Psc)
		assert.Equal(t, "5", prize.Blue)
		assert.Equal(t, "50.75", prize.Satisfaction)
	})

	t.Run("Prize_NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT id, level_id, score").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)
		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(uint64(1)).WillReturnError(sql.ErrNoRows)

		prize, err := svc.GetLevelPrizes(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, prize)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetUserLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := newLevelSvc(db)
	ctx := context.Background()
	userID := uint64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url", "gem_fbx_file"}).
				AddRow(2, "Level 2", "level-2", 200, "bg2.jpg", "img2.jpg", ""))
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs(int32(200)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg"))
		mock.ExpectQuery("SELECT score FROM users").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"score"}).AddRow(250))
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs(int32(200)).
			WillReturnRows(sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
				AddRow(3, "Level 3", "level-3", 300, "bg3.jpg", "img3.jpg"))

		userLevel, err := svc.GetUserLevel(ctx, userID)
		require.NoError(t, err)
		assert.NotNil(t, userLevel.LatestLevel)
		assert.Equal(t, uint64(2), userLevel.LatestLevel.Id)
		assert.Len(t, userLevel.PreviousLevels, 1)
		assert.Equal(t, int32(250), userLevel.UserScore)
	})

	t.Run("UserHasNoLevel", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST").
			WithArgs(userID).WillReturnError(sql.ErrNoRows)

		userLevel, err := svc.GetUserLevel(ctx, userID)
		require.NoError(t, err)
		assert.Nil(t, userLevel.LatestLevel)
		assert.Len(t, userLevel.PreviousLevels, 0)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_ClaimPrize(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	svc := newLevelSvc(db)
	ctx := context.Background()
	userID := uint64(1)
	levelID := uint64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(levelID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "level_id", "psc", "blue", "red", "yellow", "effect", "satisfaction", "created_at"}).
				AddRow(1, levelID, 1000, 5, 3, 2, 10, 50.75, nil))
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM recieved_level_prizes").
			WithArgs(userID, uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectExec("INSERT INTO recieved_level_prizes").
			WithArgs(userID, uint64(1)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := svc.ClaimPrize(ctx, userID, levelID)
		require.NoError(t, err)
	})

	t.Run("PrizeNotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(levelID).WillReturnError(sql.ErrNoRows)

		err := svc.ClaimPrize(ctx, userID, levelID)
		assert.Error(t, err)
	})

	t.Run("PrizeAlreadyClaimed", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(levelID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "level_id", "psc", "blue", "red", "yellow", "effect", "satisfaction", "created_at"}).
				AddRow(1, levelID, 1000, 5, 3, 2, 10, 50.75, nil))
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM recieved_level_prizes").
			WithArgs(userID, uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		err := svc.ClaimPrize(ctx, userID, levelID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prize already claimed")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

// ---------------------------------------------------------------------------
// Mock-interface tests – error paths and edge cases (new tests)
// ---------------------------------------------------------------------------

func TestLevelService_GetUserLevel_BelowScoreError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			GetUserLatestLevelFunc: func(ctx context.Context, userID uint64) (*pb.Level, error) {
				return &pb.Level{Id: 1, Score: 100}, nil
			},
			GetLevelsBelowScoreFunc: func(ctx context.Context, score int32) ([]*pb.Level, error) {
				return nil, errTest
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	_, err := svc.GetUserLevel(context.Background(), 1)
	assert.Error(t, err)
}

func TestLevelService_GetLevel_ByIDError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			FindByIDFunc: func(ctx context.Context, id uint64) (*pb.Level, error) { return nil, errTest },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	_, err := svc.GetLevel(context.Background(), 1, "")
	assert.Error(t, err)
}

func TestLevelService_GetLevel_BySlugError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			FindBySlugFunc: func(ctx context.Context, slug string) (*pb.Level, error) { return nil, errTest },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	_, err := svc.GetLevel(context.Background(), 0, "gold")
	assert.Error(t, err)
}

func TestLevelService_GetLevelGeneralInfo_SlugResolveError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			FindBySlugFunc: func(ctx context.Context, slug string) (*pb.Level, error) { return nil, errTest },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	_, err := svc.GetLevelGeneralInfo(context.Background(), 0, "gold")
	assert.Error(t, err)
}

func TestLevelService_GetLevelGem_SlugResolveError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			FindBySlugFunc: func(ctx context.Context, slug string) (*pb.Level, error) { return nil, errTest },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	_, err := svc.GetLevelGem(context.Background(), 0, "gold")
	assert.Error(t, err)
}

func TestLevelService_GetLevelGift_SlugResolveError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			FindBySlugFunc: func(ctx context.Context, slug string) (*pb.Level, error) { return nil, errTest },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	_, err := svc.GetLevelGift(context.Background(), 0, "gold")
	assert.Error(t, err)
}

func TestLevelService_GetLevelLicenses_SlugResolveError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			FindBySlugFunc: func(ctx context.Context, slug string) (*pb.Level, error) { return nil, errTest },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	_, err := svc.GetLevelLicenses(context.Background(), 0, "gold")
	assert.Error(t, err)
}

func TestLevelService_GetLevelPrizes_SlugResolveError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			FindBySlugFunc: func(ctx context.Context, slug string) (*pb.Level, error) { return nil, errTest },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	_, err := svc.GetLevelPrizes(context.Background(), 0, "gold")
	assert.Error(t, err)
}

func TestLevelService_GetLevelPrizes_GetPrizeError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) { return nil, errTest },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	_, err := svc.GetLevelPrizes(context.Background(), 5, "")
	assert.Error(t, err)
}

func TestLevelService_ClaimPrize_NilPrize(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) { return nil, nil },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	err := svc.ClaimPrize(context.Background(), 1, 2)
	assert.Error(t, err)
}

func TestLevelService_ClaimPrize_CheckStatusError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 5}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) {
				return false, errTest
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	err := svc.ClaimPrize(context.Background(), 1, 2)
	assert.Error(t, err)
}

func TestLevelService_ClaimPrize_FallbackPSCRate(t *testing.T) {
	addCalls := 0
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 5, Psc: "1000", Blue: "1", Red: "1", Yellow: "1", Effect: 1, Satisfaction: "1.0"}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) {
				return false, nil
			},
			GetVariableRateFunc:     func(ctx context.Context, name string) (float64, error) { return 0, errTest },
			RecordReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) error { return nil },
		},
		&testutil.MockUserLogRepository{},
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

func TestLevelService_ClaimPrize_RecordError(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 5, Psc: "1000", Blue: "1", Red: "1", Yellow: "1", Effect: 1, Satisfaction: "1.0"}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) {
				return false, nil
			},
			GetVariableRateFunc:     func(ctx context.Context, name string) (float64, error) { return 100, nil },
			RecordReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) error { return errTest },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{
			AddBalanceFunc: func(ctx context.Context, userID uint64, asset string, amount float64) error { return nil },
		},
	)
	assert.Error(t, svc.ClaimPrize(context.Background(), 1, 2))
}

func TestLevelService_ClaimPrize_Success_WithMocks(t *testing.T) {
	addCalls := 0
	recorded := false
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{Id: 5, Psc: "1000", Blue: "2", Red: "3", Yellow: "4", Effect: 1, Satisfaction: "1.50"}, nil
			},
			HasUserReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) (bool, error) {
				return false, nil
			},
			GetVariableRateFunc:     func(ctx context.Context, name string) (float64, error) { return 100, nil },
			RecordReceivedPrizeFunc: func(ctx context.Context, userID, prizeID uint64) error { recorded = true; return nil },
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{
			AddBalanceFunc: func(ctx context.Context, userID uint64, asset string, amount float64) error {
				addCalls++
				return nil
			},
		},
	)
	require.NoError(t, svc.ClaimPrize(context.Background(), 10, 2))
	assert.Equal(t, 6, addCalls)
	assert.True(t, recorded)
}

func TestLevelService_GetUserLevel_NoLevel_WithMocks(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			GetUserLatestLevelFunc: func(ctx context.Context, userID uint64) (*pb.Level, error) {
				return nil, errTest
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)
	resp, err := svc.GetUserLevel(context.Background(), 1)
	require.NoError(t, err)
	assert.Nil(t, resp.LatestLevel)
	assert.Empty(t, resp.PreviousLevels)
}

func TestLevelService_PassThroughMethods(t *testing.T) {
	svc := newMockLevelSvc(
		&testutil.MockLevelRepository{
			GetAllLevelsFunc: func(ctx context.Context) ([]*pb.Level, error) { return []*pb.Level{{Id: 1}}, nil },
			FindByIDFunc:     func(ctx context.Context, id uint64) (*pb.Level, error) { return &pb.Level{Id: id}, nil },
			FindBySlugFunc:   func(ctx context.Context, slug string) (*pb.Level, error) { return &pb.Level{Slug: slug}, nil },
			GetLevelGeneralInfoFunc: func(ctx context.Context, levelID uint64) (*pb.LevelGeneralInfo, error) {
				return &pb.LevelGeneralInfo{LevelId: levelID}, nil
			},
			GetLevelGemFunc: func(ctx context.Context, levelID uint64) (*pb.LevelGem, error) {
				return &pb.LevelGem{LevelId: levelID}, nil
			},
			GetLevelGiftFunc: func(ctx context.Context, levelID uint64) (*pb.LevelGift, error) {
				return &pb.LevelGift{LevelId: levelID}, nil
			},
			GetLevelLicensesFunc: func(ctx context.Context, levelID uint64) (*pb.LevelLicense, error) {
				return &pb.LevelLicense{LevelId: levelID}, nil
			},
			GetLevelPrizeFunc: func(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
				return &pb.LevelPrize{LevelId: levelID}, nil
			},
		},
		&testutil.MockUserLogRepository{},
		&testutil.MockCommercialClient{},
	)

	_, err := svc.GetAllLevels(context.Background())
	require.NoError(t, err)
	_, err = svc.GetLevel(context.Background(), 1, "")
	require.NoError(t, err)
	_, err = svc.GetLevel(context.Background(), 0, "gold")
	require.NoError(t, err)
	_, err = svc.GetLevelGeneralInfo(context.Background(), 0, "gold")
	require.NoError(t, err)
	_, err = svc.GetLevelGem(context.Background(), 0, "gold")
	require.NoError(t, err)
	_, err = svc.GetLevelGift(context.Background(), 0, "gold")
	require.NoError(t, err)
	_, err = svc.GetLevelLicenses(context.Background(), 0, "gold")
	require.NoError(t, err)
	_, err = svc.GetLevelPrizes(context.Background(), 0, "gold")
	require.NoError(t, err)
}
