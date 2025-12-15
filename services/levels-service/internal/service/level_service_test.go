package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metargb/levels-service/internal/repository"
)

func TestLevelService_GetAllLevels(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	service := NewLevelService(levelRepo, userLogRepo)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg").
			AddRow(2, "Level 2", "level-2", 200, "bg2.jpg", "")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WillReturnRows(rows)

		levels, err := service.GetAllLevels(ctx)
		require.NoError(t, err)
		assert.Len(t, levels, 2)
		assert.Equal(t, uint64(1), levels[0].Id)
		assert.Equal(t, "Level 1", levels[0].Name)
		assert.Equal(t, "level-1", levels[0].Slug)
		assert.Equal(t, int32(100), levels[0].Score)
		assert.Equal(t, "img1.jpg", levels[0].ImageUrl)
		assert.Equal(t, "bg1.jpg", levels[0].BackgroundImage)
	})

	t.Run("EmptyResult", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"})
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WillReturnRows(rows)

		levels, err := service.GetAllLevels(ctx)
		require.NoError(t, err)
		assert.Len(t, levels, 0)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	service := NewLevelService(levelRepo, userLogRepo)

	ctx := context.Background()

	t.Run("ByID_Success", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		generalInfoRows := sqlmock.NewRows([]string{"id", "level_id", "score", "rank", "description", "subcategories",
			"persian_font", "english_font", "file_volume", "used_colors", "points", "lines",
			"has_animation", "designer", "model_designer", "creation_date", "png_file", "fbx_file", "gif_file"}).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs(uint64(1)).
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, score, rank").
			WithArgs(uint64(1)).
			WillReturnRows(generalInfoRows)

		level, err := service.GetLevel(ctx, 1, "")
		require.NoError(t, err)
		assert.NotNil(t, level)
		assert.Equal(t, uint64(1), level.Id)
		assert.Equal(t, "Level 1", level.Name)
	})

	t.Run("BySlug_Success", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		generalInfoRows := sqlmock.NewRows([]string{"id", "level_id", "score", "rank", "description", "subcategories",
			"persian_font", "english_font", "file_volume", "used_colors", "points", "lines",
			"has_animation", "designer", "model_designer", "creation_date", "png_file", "fbx_file", "gif_file"}).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, score, rank").
			WithArgs(uint64(1)).
			WillReturnRows(generalInfoRows)

		level, err := service.GetLevel(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, level)
		assert.Equal(t, "level-1", level.Slug)
	})

	t.Run("BySlug_NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("non-existent").
			WillReturnError(sql.ErrNoRows)

		level, err := service.GetLevel(ctx, 0, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, level)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelGeneralInfo(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	service := NewLevelService(levelRepo, userLogRepo)

	ctx := context.Background()

	t.Run("BySlug_Success", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		generalInfoRows := sqlmock.NewRows([]string{"id", "level_id", "score", "rank", "description", "subcategories",
			"persian_font", "english_font", "file_volume", "used_colors", "points", "lines",
			"has_animation", "designer", "model_designer", "creation_date", "png_file", "fbx_file", "gif_file"}).
			AddRow(1, 1, 100, 1, "Description", 2, "Font1", "Font2", 1.5, "Colors", 100, 200, 1, "Designer", "Model Designer", "2024-01-01", "png", "fbx", "gif")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, score, rank").
			WithArgs(uint64(1)).
			WillReturnRows(generalInfoRows)

		info, err := service.GetLevelGeneralInfo(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, info)
		assert.Equal(t, uint64(1), info.Id)
		assert.Equal(t, "Description", info.Description)
	})

	t.Run("GeneralInfo_NotFound", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, score, rank").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)

		info, err := service.GetLevelGeneralInfo(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, info) // Missing general info returns nil, not error
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelGem(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	service := NewLevelService(levelRepo, userLogRepo)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		gemRows := sqlmock.NewRows([]string{"id", "level_id", "name", "description", "thread", "points", "volume", "color",
			"has_animation", "lines", "png_file", "fbx_file", "encryption", "designer"}).
			AddRow(1, 1, "Gem 1", "Description", "thread1", 50, "vol1", "red", 1, 100, "png", "fbx", 0, "Designer")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).
			WillReturnRows(gemRows)

		gem, err := service.GetLevelGem(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, gem)
		assert.Equal(t, uint64(1), gem.Id)
		assert.Equal(t, "Gem 1", gem.Name)
	})

	t.Run("Gem_NotFound", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)

		gem, err := service.GetLevelGem(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, gem) // Missing gem returns nil, not error
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelGift(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	service := NewLevelService(levelRepo, userLogRepo)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		giftRows := sqlmock.NewRows([]string{"id", "level_id", "name", "description", "monthly_capacity_count", "store_capacity",
			"sell_capacity", "features", "sell", "vod_document_registration", "seller_link", "designer",
			"three_d_model_volume", "three_d_model_points", "three_d_model_lines", "has_animation",
			"png_file", "fbx_file", "gif_file", "rent", "vod_count", "start_vod_id", "end_vod_id"}).
			AddRow(1, 1, "Gift 1", "Description", 10, 1, 1, "features", 1, 0, "link", "Designer",
				"vol", 100, 200, 1, "png", "fbx", "gif", 0, 5, "start", "end")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).
			WillReturnRows(giftRows)

		gift, err := service.GetLevelGift(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, gift)
		assert.Equal(t, uint64(1), gift.Id)
		assert.Equal(t, "Gift 1", gift.Name)
	})

	t.Run("Gift_NotFound", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)

		gift, err := service.GetLevelGift(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, gift) // Missing gift returns nil, not error
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelLicenses(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	service := NewLevelService(levelRepo, userLogRepo)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		licenseRows := sqlmock.NewRows([]string{"id", "level_id", "create_union", "add_memeber_to_union", "observation_license",
			"gate_license", "lawyer_license", "city_counsile_entry", "establish_special_residential_property",
			"establish_property_on_surface", "judge_entry", "upload_image", "delete_image",
			"inter_level_general_points", "inter_level_special_points", "rent_out_satisfaction",
			"access_to_answer_questions_unit", "create_challenge_questions", "upload_music"}).
			AddRow(1, 1, 1, 0, 1, 0, 1, 0, 1, 0, 1, 1, 0, 1, 0, 1, 1, 0, 1)

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, create_union").
			WithArgs(uint64(1)).
			WillReturnRows(licenseRows)

		licenses, err := service.GetLevelLicenses(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, licenses)
		assert.Equal(t, uint64(1), licenses.Id)
		assert.True(t, licenses.CreateUnion)
		assert.False(t, licenses.AddMemeberToUnion)
	})

	t.Run("Licenses_NotFound", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, create_union").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)

		licenses, err := service.GetLevelLicenses(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, licenses) // Missing licenses returns nil, not error
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetLevelPrizes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	service := NewLevelService(levelRepo, userLogRepo)

	ctx := context.Background()

	t.Run("BySlug_Success", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		prizeRows := sqlmock.NewRows([]string{"id", "level_id", "psc", "blue", "red", "yellow", "effect", "satisfaction", "created_at"}).
			AddRow(1, 1, 1000, 5, 3, 2, 10, 50.75, nil)

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(uint64(1)).
			WillReturnRows(prizeRows)

		prize, err := service.GetLevelPrizes(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.NotNil(t, prize)
		assert.Equal(t, uint64(1), prize.Id)
		assert.Equal(t, "1000", prize.Psc)
		assert.Equal(t, "5", prize.Blue)
		assert.Equal(t, "50.75", prize.Satisfaction)
	})

	t.Run("Prize_NotFound", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)

		prize, err := service.GetLevelPrizes(ctx, 0, "level-1")
		require.NoError(t, err)
		assert.Nil(t, prize) // Missing prize returns nil, not error
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_GetUserLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	service := NewLevelService(levelRepo, userLogRepo)

	ctx := context.Background()
	userID := uint64(1)

	t.Run("Success", func(t *testing.T) {
		// Get latest level
		latestLevelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(2, "Level 2", "level-2", 200, "bg2.jpg", "img2.jpg")

		// Get previous levels
		previousLevelsRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		// Get next level
		nextLevelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(3, "Level 3", "level-3", 300, "bg3.jpg", "img3.jpg")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, l.score").
			WithArgs(userID).
			WillReturnRows(latestLevelRows)

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs(int32(200)).
			WillReturnRows(previousLevelsRows)

		mock.ExpectQuery("SELECT score FROM user_logs").
			WithArgs(userID).
			WillReturnRows(sqlmock.NewRows([]string{"score"}).AddRow(250))

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs(int32(200)).
			WillReturnRows(nextLevelRows)

		userLevel, err := service.GetUserLevel(ctx, userID)
		require.NoError(t, err)
		assert.NotNil(t, userLevel)
		assert.NotNil(t, userLevel.LatestLevel)
		assert.Equal(t, uint64(2), userLevel.LatestLevel.Id)
		assert.Len(t, userLevel.PreviousLevels, 1)
		assert.Equal(t, int32(250), userLevel.UserScore)
	})

	t.Run("UserHasNoLevel", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, l.score").
			WithArgs(userID).
			WillReturnError(sql.ErrNoRows)

		userLevel, err := service.GetUserLevel(ctx, userID)
		require.NoError(t, err)
		assert.NotNil(t, userLevel)
		assert.Nil(t, userLevel.LatestLevel)
		assert.Len(t, userLevel.PreviousLevels, 0)
		assert.Equal(t, int32(0), userLevel.UserScore)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelService_ClaimPrize(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	service := NewLevelService(levelRepo, userLogRepo)

	ctx := context.Background()
	userID := uint64(1)
	levelID := uint64(1)

	t.Run("Success", func(t *testing.T) {
		prizeRows := sqlmock.NewRows([]string{"id", "level_id", "psc", "blue", "red", "yellow", "effect", "satisfaction", "created_at"}).
			AddRow(1, levelID, 1000, 5, 3, 2, 10, 50.75, nil)

		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(levelID).
			WillReturnRows(prizeRows)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM recieved_level_prizes").
			WithArgs(userID, uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		mock.ExpectExec("INSERT INTO recieved_level_prizes").
			WithArgs(userID, uint64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := service.ClaimPrize(ctx, userID, levelID)
		require.NoError(t, err)
	})

	t.Run("PrizeNotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(levelID).
			WillReturnError(sql.ErrNoRows)

		err := service.ClaimPrize(ctx, userID, levelID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get level prize")
	})

	t.Run("PrizeAlreadyClaimed", func(t *testing.T) {
		prizeRows := sqlmock.NewRows([]string{"id", "level_id", "psc", "blue", "red", "yellow", "effect", "satisfaction", "created_at"}).
			AddRow(1, levelID, 1000, 5, 3, 2, 10, 50.75, nil)

		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(levelID).
			WillReturnRows(prizeRows)

		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM recieved_level_prizes").
			WithArgs(userID, uint64(1)).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		err := service.ClaimPrize(ctx, userID, levelID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "prize already claimed")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}
