package handler

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/levels-service/internal/repository"
	"metargb/levels-service/internal/service"
	pb "metargb/shared/pb/levels"
)

func TestLevelHandler_GetAllLevels(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	levelService := service.NewLevelService(levelRepo, userLogRepo)
	handler := NewLevelHandler(levelService)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg").
			AddRow(2, "Level 2", "level-2", 200, "bg2.jpg", "")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WillReturnRows(rows)

		req := &pb.GetAllLevelsRequest{}
		resp, err := handler.GetAllLevels(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Levels, 2)
		assert.Equal(t, uint64(1), resp.Levels[0].Id)
		assert.Equal(t, "Level 1", resp.Levels[0].Name)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	levelService := service.NewLevelService(levelRepo, userLogRepo)
	handler := NewLevelHandler(levelService)

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

		req := &pb.GetLevelRequest{LevelId: 1}
		resp, err := handler.GetLevel(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Level)
		assert.Equal(t, uint64(1), resp.Level.Id)
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

		req := &pb.GetLevelRequest{LevelSlug: "level-1"}
		resp, err := handler.GetLevel(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Level)
		assert.Equal(t, "level-1", resp.Level.Slug)
	})

	t.Run("InvalidRequest", func(t *testing.T) {
		req := &pb.GetLevelRequest{}
		resp, err := handler.GetLevel(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("NotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("non-existent").
			WillReturnError(sql.ErrNoRows)

		req := &pb.GetLevelRequest{LevelSlug: "non-existent"}
		resp, err := handler.GetLevel(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.NotFound, st.Code())
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetLevelGeneralInfo(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	levelService := service.NewLevelService(levelRepo, userLogRepo)
	handler := NewLevelHandler(levelService)

	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
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

		req := &pb.GetLevelGeneralInfoRequest{LevelSlug: "level-1"}
		resp, err := handler.GetLevelGeneralInfo(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.GeneralInfo)
		assert.Equal(t, "Description", resp.GeneralInfo.Description)
	})

	t.Run("GeneralInfo_NotFound_ReturnsNull", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, score, rank").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)

		req := &pb.GetLevelGeneralInfoRequest{LevelSlug: "level-1"}
		resp, err := handler.GetLevelGeneralInfo(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Nil(t, resp.GeneralInfo) // Missing general info returns null, not error
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetLevelGem(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	levelService := service.NewLevelService(levelRepo, userLogRepo)
	handler := NewLevelHandler(levelService)

	ctx := context.Background()

	t.Run("Gem_NotFound_ReturnsNull", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, name, description").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)

		req := &pb.GetLevelGemRequest{LevelSlug: "level-1"}
		resp, err := handler.GetLevelGem(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Nil(t, resp.Gem) // Missing gem returns null, not error
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetLevelPrizes(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	levelService := service.NewLevelService(levelRepo, userLogRepo)
	handler := NewLevelHandler(levelService)

	ctx := context.Background()

	t.Run("Prize_NotFound_ReturnsNull", func(t *testing.T) {
		levelRows := sqlmock.NewRows([]string{"id", "name", "slug", "score", "background_image", "image_url"}).
			AddRow(1, "Level 1", "level-1", 100, "bg1.jpg", "img1.jpg")

		mock.ExpectQuery("SELECT l.id, l.name, l.slug, CAST\\(l.score AS UNSIGNED\\) as score").
			WithArgs("level-1").
			WillReturnRows(levelRows)

		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(uint64(1)).
			WillReturnError(sql.ErrNoRows)

		req := &pb.GetLevelPrizesRequest{LevelSlug: "level-1"}
		resp, err := handler.GetLevelPrizes(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Nil(t, resp.Prize) // Missing prize returns null, not error
	})

	t.Run("ByLevelID", func(t *testing.T) {
		prizeRows := sqlmock.NewRows([]string{"id", "level_id", "psc", "blue", "red", "yellow", "effect", "satisfaction", "created_at"}).
			AddRow(1, 1, 1000, 5, 3, 2, 10, 50.75, nil)

		mock.ExpectQuery("SELECT id, level_id, psc, blue, red, yellow, effect, satisfaction, created_at").
			WithArgs(uint64(1)).
			WillReturnRows(prizeRows)

		req := &pb.GetLevelPrizesRequest{LevelId: 1}
		resp, err := handler.GetLevelPrizes(ctx, req)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.Prize)
		assert.Equal(t, "1000", resp.Prize.Psc)
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_GetUserLevel(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	levelService := service.NewLevelService(levelRepo, userLogRepo)
	handler := NewLevelHandler(levelService)

	ctx := context.Background()

	t.Run("InvalidRequest", func(t *testing.T) {
		req := &pb.GetUserLevelRequest{UserId: 0}
		resp, err := handler.GetUserLevel(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestLevelHandler_ClaimPrize(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	levelRepo := repository.NewLevelRepository(db)
	userLogRepo := repository.NewUserLogRepository(db)
	levelService := service.NewLevelService(levelRepo, userLogRepo)
	handler := NewLevelHandler(levelService)

	ctx := context.Background()

	t.Run("InvalidRequest_MissingUserID", func(t *testing.T) {
		req := &pb.ClaimPrizeRequest{LevelId: 1}
		resp, err := handler.ClaimPrize(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("InvalidRequest_MissingLevelID", func(t *testing.T) {
		req := &pb.ClaimPrizeRequest{UserId: 1}
		resp, err := handler.ClaimPrize(ctx, req)
		assert.Error(t, err)
		assert.Nil(t, resp)
		st, ok := status.FromError(err)
		require.True(t, ok)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	require.NoError(t, mock.ExpectationsWereMet())
}
