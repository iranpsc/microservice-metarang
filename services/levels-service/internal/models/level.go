package models

import "time"

// Level represents a user level in the system
// Maps to Laravel: App\Models\Levels\Level
type Level struct {
	ID              uint64    `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Slug            string    `json:"slug" db:"slug"`
	Score           int32     `json:"score" db:"score"`
	BackgroundImage *string   `json:"background_image" db:"background_image"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// LevelGeneralInfo represents detailed information about a level
// Maps to Laravel: App\Models\Levels\LevelGeneralInfo
type LevelGeneralInfo struct {
	ID            uint64    `json:"id" db:"id"`
	LevelID       uint64    `json:"level_id" db:"level_id"`
	Score         int32     `json:"score" db:"score"`
	Rank          string    `json:"rank" db:"rank"`
	Description   string    `json:"description" db:"description"`
	Subcategories *string   `json:"subcategories" db:"subcategories"`
	PersianFont   *string   `json:"persian_font" db:"persian_font"`
	EnglishFont   *string   `json:"english_font" db:"english_font"`
	FileVolume    *string   `json:"file_volume" db:"file_volume"`
	UsedColors    *string   `json:"used_colors" db:"used_colors"`
	Points        *string   `json:"points" db:"points"`
	Lines         *string   `json:"lines" db:"lines"`
	HasAnimation  *bool     `json:"has_animation" db:"has_animation"`
	Designer      *string   `json:"designer" db:"designer"`
	ModelDesigner *string   `json:"model_designer" db:"model_designer"`
	CreationDate  *string   `json:"creation_date" db:"creation_date"`
	PngFile       *string   `json:"png_file" db:"png_file"`
	FbxFile       *string   `json:"fbx_file" db:"fbx_file"`
	GifFile       *string   `json:"gif_file" db:"gif_file"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// LevelPrize represents prizes awarded when reaching a level
// Maps to Laravel: App\Models\Levels\LevelPrize (which references prizes table)
type LevelPrize struct {
	ID                       uint64    `json:"id" db:"id"`
	LevelID                  uint64    `json:"level_id" db:"level_id"`
	Psc                      *int64    `json:"psc" db:"psc"`
	Blue                     *int64    `json:"blue" db:"blue"`
	Red                      *int64    `json:"red" db:"red"`
	Yellow                   *int64    `json:"yellow" db:"yellow"`
	UnionLicense             *int8     `json:"union_license" db:"union_license"`
	UnionMembersCount        *int32    `json:"union_members_count" db:"union_members_count"`
	ObservingLicense         *int8     `json:"observing_license" db:"observing_license"`
	GateLicense              *int8     `json:"gate_license" db:"gate_license"`
	LawyerLicense            *int8     `json:"lawyer_license" db:"lawyer_license"`
	CityCouncilEntry         *int8     `json:"city_counsil_entry" db:"city_counsil_entry"`
	SpecialResidenceProperty *int64    `json:"special_residence_property" db:"special_residence_property"`
	PropertyOnArea           *int64    `json:"property_on_area" db:"property_on_area"`
	JudgeEntry               *int8     `json:"judge_entry" db:"judge_entry"`
	Satisfaction             float32   `json:"satisfaction" db:"satisfaction"`
	Effect                   int32     `json:"effect" db:"effect"`
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// LevelGem represents gem information for a level
// Maps to Laravel: App\Models\Levels\LevelGem
type LevelGem struct {
	ID          uint64    `json:"id" db:"id"`
	LevelID     uint64    `json:"level_id" db:"level_id"`
	Name        *string   `json:"name" db:"name"`
	Slug        *string   `json:"slug" db:"slug"`
	Description *string   `json:"description" db:"description"`
	ImageURL    *string   `json:"image_url" db:"image_url"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// LevelGift represents gift information for a level
// Maps to Laravel: App\Models\Levels\LevelGift
type LevelGift struct {
	ID      uint64 `json:"id" db:"id"`
	LevelID uint64 `json:"level_id" db:"level_id"`
	// Add gift-specific fields based on the actual database schema
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// LevelLicense represents license information for a level
// Maps to Laravel: App\Models\Levels\LevelLicense
type LevelLicense struct {
	ID      uint64 `json:"id" db:"id"`
	LevelID uint64 `json:"level_id" db:"level_id"`
	// Add license-specific fields based on the actual database schema
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// LevelUser represents the pivot table for user-level relationship
// Maps to Laravel: App\Models\Levels\LevelUser
type LevelUser struct {
	ID        uint64    `json:"id" db:"id"`
	UserID    uint64    `json:"user_id" db:"user_id"`
	LevelID   uint64    `json:"level_id" db:"level_id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ReceivedLevelPrize represents prizes that users have received
// Maps to Laravel: App\Models\Levels\RecievedLevelPrize
type ReceivedLevelPrize struct {
	ID           uint64    `json:"id" db:"id"`
	UserID       uint64    `json:"user_id" db:"user_id"`
	LevelPrizeID uint64    `json:"level_prize_id" db:"level_prize_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
