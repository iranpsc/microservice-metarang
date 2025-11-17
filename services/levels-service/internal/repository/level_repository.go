package repository

import (
	"context"
	"database/sql"
	"fmt"

	pb "metargb/shared/pb/levels"
)

// LevelRepository handles level database operations
// Implements Laravel's Level model and relationships
type LevelRepository struct {
	db *sql.DB
}

func NewLevelRepository(db *sql.DB) *LevelRepository {
	return &LevelRepository{db: db}
}

// GetUserLatestLevel retrieves user's latest achieved level
// Implements Laravel: $user->levels()->orderByDesc('id')->first()
func (r *LevelRepository) GetUserLatestLevel(ctx context.Context, userID uint64) (*pb.Level, error) {
	query := `
		SELECT l.id, l.name, l.slug, l.score, l.background_image,
		       COALESCE(i.url, '') as image_url
		FROM levels l
		INNER JOIN level_user lu ON l.id = lu.level_id
		LEFT JOIN images i ON i.imageable_id = l.id AND i.imageable_type = 'App\\Models\\Levels\\Level'
		WHERE lu.user_id = ?
		ORDER BY l.id DESC
		LIMIT 1
	`
	
	var level pb.Level
	var imageURL sql.NullString
	var backgroundImage sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&level.Id,
		&level.Name,
		&level.Slug,
		&level.Score,
		&backgroundImage,
		&imageURL,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user has no level")
		}
		return nil, err
	}
	
	if imageURL.Valid {
		level.ImageUrl = imageURL.String
	}
	if backgroundImage.Valid {
		level.BackgroundImage = backgroundImage.String
	}
	
	return &level, nil
}

// GetLevelsBelowScore retrieves all levels with score less than given score
// Implements Laravel: Level::where('score', '<', $score)->orderBy('score')->get()
func (r *LevelRepository) GetLevelsBelowScore(ctx context.Context, score int32) ([]*pb.Level, error) {
	query := `
		SELECT l.id, l.name, l.slug, l.score, l.background_image,
		       COALESCE(i.url, '') as image_url
		FROM levels l
		LEFT JOIN images i ON i.imageable_id = l.id AND i.imageable_type = 'App\\Models\\Levels\\Level'
		WHERE l.score < ?
		ORDER BY l.score ASC
	`
	
	rows, err := r.db.QueryContext(ctx, query, score)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var levels []*pb.Level
	for rows.Next() {
		var level pb.Level
		var imageURL sql.NullString
		var backgroundImage sql.NullString
		
		if err := rows.Scan(&level.Id, &level.Name, &level.Slug, &level.Score, &backgroundImage, &imageURL); err != nil {
			return nil, err
		}
		
		if imageURL.Valid {
			level.ImageUrl = imageURL.String
		}
		if backgroundImage.Valid {
			level.BackgroundImage = backgroundImage.String
		}
		
		levels = append(levels, &level)
	}
	
	return levels, nil
}

// GetNextLevel retrieves the next level above current score
// Used for calculating progress percentage
func (r *LevelRepository) GetNextLevel(ctx context.Context, currentScore int32) (*pb.Level, error) {
	query := `
		SELECT l.id, l.name, l.slug, l.score, l.background_image,
		       COALESCE(i.url, '') as image_url
		FROM levels l
		LEFT JOIN images i ON i.imageable_id = l.id AND i.imageable_type = 'App\\Models\\Levels\\Level'
		WHERE l.score > ?
		ORDER BY l.score ASC
		LIMIT 1
	`
	
	var level pb.Level
	var imageURL sql.NullString
	var backgroundImage sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, currentScore).Scan(
		&level.Id,
		&level.Name,
		&level.Slug,
		&level.Score,
		&backgroundImage,
		&imageURL,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No next level
		}
		return nil, err
	}
	
	if imageURL.Valid {
		level.ImageUrl = imageURL.String
	}
	if backgroundImage.Valid {
		level.BackgroundImage = backgroundImage.String
	}
	
	return &level, nil
}

// GetAllLevels retrieves all levels
// Implements Laravel: Level::select('id', 'name', 'slug')->with('image')->get()
func (r *LevelRepository) GetAllLevels(ctx context.Context) ([]*pb.Level, error) {
	query := `
		SELECT l.id, l.name, l.slug, l.score, l.background_image,
		       COALESCE(i.url, '') as image_url
		FROM levels l
		LEFT JOIN images i ON i.imageable_id = l.id AND i.imageable_type = 'App\\Models\\Levels\\Level'
		ORDER BY l.score ASC
	`
	
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var levels []*pb.Level
	for rows.Next() {
		var level pb.Level
		var imageURL sql.NullString
		var backgroundImage sql.NullString
		
		if err := rows.Scan(&level.Id, &level.Name, &level.Slug, &level.Score, &backgroundImage, &imageURL); err != nil {
			return nil, err
		}
		
		if imageURL.Valid {
			level.ImageUrl = imageURL.String
		}
		if backgroundImage.Valid {
			level.BackgroundImage = backgroundImage.String
		}
		
		levels = append(levels, &level)
	}
	
	return levels, nil
}

// FindByID retrieves a level by ID with all relationships
// Implements Laravel: Level::find($id)->load('image', 'generalInfo')
func (r *LevelRepository) FindByID(ctx context.Context, id uint64) (*pb.Level, error) {
	query := `
		SELECT l.id, l.name, l.slug, l.score, l.background_image,
		       COALESCE(i.url, '') as image_url
		FROM levels l
		LEFT JOIN images i ON i.imageable_id = l.id AND i.imageable_type = 'App\\Models\\Levels\\Level'
		WHERE l.id = ?
	`
	
	var level pb.Level
	var imageURL sql.NullString
	var backgroundImage sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&level.Id,
		&level.Name,
		&level.Slug,
		&level.Score,
		&backgroundImage,
		&imageURL,
	)
	
	if err != nil {
		return nil, err
	}
	
	if imageURL.Valid {
		level.ImageUrl = imageURL.String
	}
	if backgroundImage.Valid {
		level.BackgroundImage = backgroundImage.String
	}
	
	// Load general info
	generalInfo, err := r.GetLevelGeneralInfo(ctx, id)
	if err == nil {
		level.GeneralInfo = generalInfo
	}
	
	return &level, nil
}

// FindBySlug retrieves a level by slug
// Implements Laravel: Level::where('slug', $slug)->firstOrFail()
func (r *LevelRepository) FindBySlug(ctx context.Context, slug string) (*pb.Level, error) {
	query := `
		SELECT l.id, l.name, l.slug, l.score, l.background_image,
		       COALESCE(i.url, '') as image_url
		FROM levels l
		LEFT JOIN images i ON i.imageable_id = l.id AND i.imageable_type = 'App\\Models\\Levels\\Level'
		WHERE l.slug = ?
	`
	
	var level pb.Level
	var imageURL sql.NullString
	var backgroundImage sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, slug).Scan(
		&level.Id,
		&level.Name,
		&level.Slug,
		&level.Score,
		&backgroundImage,
		&imageURL,
	)
	
	if err != nil {
		return nil, err
	}
	
	if imageURL.Valid {
		level.ImageUrl = imageURL.String
	}
	if backgroundImage.Valid {
		level.BackgroundImage = backgroundImage.String
	}
	
	// Load general info
	generalInfo, err := r.GetLevelGeneralInfo(ctx, level.Id)
	if err == nil {
		level.GeneralInfo = generalInfo
	}
	
	return &level, nil
}

// GetLevelGeneralInfo retrieves general information for a level
// Implements Laravel: $level->generalInfo
func (r *LevelRepository) GetLevelGeneralInfo(ctx context.Context, levelID uint64) (*pb.LevelGeneralInfo, error) {
	query := `
		SELECT id, level_id, score, rank, description,
		       COALESCE(subcategories, '') as subcategories,
		       COALESCE(persian_font, '') as persian_font,
		       COALESCE(english_font, '') as english_font,
		       COALESCE(file_volume, '') as file_volume,
		       COALESCE(used_colors, '') as used_colors,
		       COALESCE(points, '') as points,
		       COALESCE(lines, '') as lines,
		       COALESCE(has_animation, 0) as has_animation,
		       COALESCE(designer, '') as designer,
		       COALESCE(model_designer, '') as model_designer,
		       COALESCE(creation_date, '') as creation_date,
		       COALESCE(png_file, '') as png_file,
		       COALESCE(fbx_file, '') as fbx_file,
		       COALESCE(gif_file, '') as gif_file
		FROM level_general_infos
		WHERE level_id = ?
	`
	
	var info pb.LevelGeneralInfo
	
	err := r.db.QueryRowContext(ctx, query, levelID).Scan(
		&info.Id,
		&info.LevelId,
		&info.Score,
		&info.Rank,
		&info.Description,
		&info.Subcategories,
		&info.PersianFont,
		&info.EnglishFont,
		&info.FileVolume,
		&info.UsedColors,
		&info.Points,
		&info.Lines,
		&info.HasAnimation,
		&info.Designer,
		&info.ModelDesigner,
		&info.CreationDate,
		&info.PngFile,
		&info.FbxFile,
		&info.GifFile,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &info, nil
}

// GetLevelPrize retrieves prize information for a level
// Implements Laravel: $level->prize
func (r *LevelRepository) GetLevelPrize(ctx context.Context, levelID uint64) (*pb.LevelPrize, error) {
	query := `
		SELECT id, level_id,
		       COALESCE(psc, 0) as psc,
		       COALESCE(blue, 0) as blue,
		       COALESCE(red, 0) as red,
		       COALESCE(yellow, 0) as yellow,
		       COALESCE(union_license, 0) as union_license,
		       COALESCE(union_members_count, 0) as union_members_count,
		       COALESCE(observing_license, 0) as observing_license,
		       COALESCE(gate_license, 0) as gate_license,
		       COALESCE(lawyer_license, 0) as lawyer_license,
		       COALESCE(city_counsil_entry, 0) as city_counsil_entry,
		       COALESCE(special_residence_property, 0) as special_residence_property,
		       COALESCE(property_on_area, 0) as property_on_area,
		       COALESCE(judge_entry, 0) as judge_entry,
		       satisfaction,
		       effect
		FROM prizes
		WHERE level_id = ?
	`
	
	var prize pb.LevelPrize
	var psc, blue, red, yellow, specialProp, propArea int64
	
	err := r.db.QueryRowContext(ctx, query, levelID).Scan(
		&prize.Id,
		&prize.LevelId,
		&psc,
		&blue,
		&red,
		&yellow,
		&prize.UnionLicense,
		&prize.UnionMembersCount,
		&prize.ObservingLicense,
		&prize.GateLicense,
		&prize.LawyerLicense,
		&prize.CityCounsilEntry,
		&specialProp,
		&propArea,
		&prize.JudgeEntry,
		&prize.Satisfaction,
		&prize.Effect,
	)
	
	if err != nil {
		return nil, err
	}
	
	// Convert integers to strings for consistency with Laravel
	prize.Psc = fmt.Sprintf("%d", psc)
	prize.Blue = fmt.Sprintf("%d", blue)
	prize.Red = fmt.Sprintf("%d", red)
	prize.Yellow = fmt.Sprintf("%d", yellow)
	prize.SpecialResidenceProperty = fmt.Sprintf("%d", specialProp)
	prize.PropertyOnArea = fmt.Sprintf("%d", propArea)
	
	return &prize, nil
}

// GetLevelGem retrieves gem information for a level
// Implements Laravel: $level->gem
func (r *LevelRepository) GetLevelGem(ctx context.Context, levelID uint64) (*pb.LevelGem, error) {
	query := `
		SELECT id, level_id,
		       COALESCE(name, '') as name,
		       COALESCE(slug, '') as slug,
		       COALESCE(description, '') as description,
		       COALESCE(image_url, '') as image_url
		FROM level_gems
		WHERE level_id = ?
	`
	
	var gem pb.LevelGem
	
	err := r.db.QueryRowContext(ctx, query, levelID).Scan(
		&gem.Id,
		&gem.LevelId,
		&gem.Name,
		&gem.Slug,
		&gem.Description,
		&gem.ImageUrl,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &gem, nil
}

// GetLevelGift retrieves gift information for a level
// Implements Laravel: $level->gift
func (r *LevelRepository) GetLevelGift(ctx context.Context, levelID uint64) (*pb.LevelGift, error) {
	query := `
		SELECT id, level_id
		FROM level_gifts
		WHERE level_id = ?
	`
	
	var gift pb.LevelGift
	
	err := r.db.QueryRowContext(ctx, query, levelID).Scan(
		&gift.Id,
		&gift.LevelId,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &gift, nil
}

// GetLevelLicenses retrieves license information for a level
// Implements Laravel: $level->licenses
func (r *LevelRepository) GetLevelLicenses(ctx context.Context, levelID uint64) (*pb.LevelLicense, error) {
	query := `
		SELECT id, level_id
		FROM level_licenses
		WHERE level_id = ?
	`
	
	var license pb.LevelLicense
	
	err := r.db.QueryRowContext(ctx, query, levelID).Scan(
		&license.Id,
		&license.LevelId,
	)
	
	if err != nil {
		return nil, err
	}
	
	return &license, nil
}

// GetNextLevelForScore finds the next level a user should achieve based on their score
// Implements Laravel: Level::where('score', '<=', $user->score)->whereNotIn('id', $user->levels->pluck('id'))->first()
func (r *LevelRepository) GetNextLevelForScore(ctx context.Context, userID uint64, score int32) (*pb.Level, error) {
	query := `
		SELECT l.id, l.name, l.slug, l.score, l.background_image,
		       COALESCE(i.url, '') as image_url
		FROM levels l
		LEFT JOIN images i ON i.imageable_id = l.id AND i.imageable_type = 'App\\Models\\Levels\\Level'
		WHERE l.score <= ?
		  AND l.id NOT IN (
		      SELECT level_id FROM level_user WHERE user_id = ?
		  )
		ORDER BY l.score DESC
		LIMIT 1
	`
	
	var level pb.Level
	var imageURL sql.NullString
	var backgroundImage sql.NullString
	
	err := r.db.QueryRowContext(ctx, query, score, userID).Scan(
		&level.Id,
		&level.Name,
		&level.Slug,
		&level.Score,
		&backgroundImage,
		&imageURL,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No new level to achieve
		}
		return nil, err
	}
	
	if imageURL.Valid {
		level.ImageUrl = imageURL.String
	}
	if backgroundImage.Valid {
		level.BackgroundImage = backgroundImage.String
	}
	
	return &level, nil
}

// AttachLevelToUser attaches a level to a user
// Implements Laravel: $user->levels()->attach($level_id)
func (r *LevelRepository) AttachLevelToUser(ctx context.Context, userID, levelID uint64) error {
	query := "INSERT INTO level_user (user_id, level_id, created_at, updated_at) VALUES (?, ?, NOW(), NOW())"
	_, err := r.db.ExecContext(ctx, query, userID, levelID)
	return err
}

// HasUserReceivedPrize checks if user has received prize for a level
// Implements Laravel: $user->recievedLevelPrizes()->where('level_prize_id', $prize_id)->exists()
func (r *LevelRepository) HasUserReceivedPrize(ctx context.Context, userID, prizeID uint64) (bool, error) {
	query := "SELECT COUNT(*) FROM recieved_level_prizes WHERE user_id = ? AND level_prize_id = ?"
	var count int
	err := r.db.QueryRowContext(ctx, query, userID, prizeID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// RecordReceivedPrize records that user has received a prize
// Implements Laravel: $user->recievedLevelPrizes()->attach($prize_id)
func (r *LevelRepository) RecordReceivedPrize(ctx context.Context, userID, prizeID uint64) error {
	query := "INSERT INTO recieved_level_prizes (user_id, level_prize_id, created_at, updated_at) VALUES (?, ?, NOW(), NOW())"
	_, err := r.db.ExecContext(ctx, query, userID, prizeID)
	return err
}

