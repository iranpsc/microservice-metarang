package repository

import (
	"context"
	"database/sql"
	"testing"

	"metargb/auth-service/internal/models"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupProfileLimitationTestDB(t *testing.T) *sql.DB {
	// Use test database connection
	dsn := "root@tcp(localhost:3306)/metargb_db_test?parseTime=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Skipf("Database not available: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("Database ping failed: %v", err)
	}

	// Clean up test data
	_, _ = db.Exec("DELETE FROM profile_limitations")

	return db
}

func TestProfileLimitationRepository_Create(t *testing.T) {
	db := setupProfileLimitationTestDB(t)
	defer db.Close()

	repo := NewProfileLimitationRepository(db)
	ctx := context.Background()

	limitation := &models.ProfileLimitation{
		LimiterUserID: 1,
		LimitedUserID: 2,
		Options:       models.DefaultOptions(),
	}
	limitation.Options.Follow = false
	limitation.Options.SendMessage = false

	err := repo.Create(ctx, limitation)
	require.NoError(t, err)
	assert.NotZero(t, limitation.ID)
	assert.NotZero(t, limitation.CreatedAt)
	assert.NotZero(t, limitation.UpdatedAt)
}

func TestProfileLimitationRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProfileLimitationRepository(db)
	ctx := context.Background()

	// Create a limitation
	limitation := &models.ProfileLimitation{
		LimiterUserID: 1,
		LimitedUserID: 2,
		Options:       models.DefaultOptions(),
	}
	err := repo.Create(ctx, limitation)
	require.NoError(t, err)

	// Find it
	found, err := repo.FindByID(ctx, limitation.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, limitation.ID, found.ID)
	assert.Equal(t, limitation.LimiterUserID, found.LimiterUserID)
	assert.Equal(t, limitation.LimitedUserID, found.LimitedUserID)
	assert.Equal(t, limitation.Options.Follow, found.Options.Follow)
}

func TestProfileLimitationRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProfileLimitationRepository(db)
	ctx := context.Background()

	found, err := repo.FindByID(ctx, 99999)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestProfileLimitationRepository_FindByLimiterAndLimited(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProfileLimitationRepository(db)
	ctx := context.Background()

	// Create a limitation
	limitation := &models.ProfileLimitation{
		LimiterUserID: 1,
		LimitedUserID: 2,
		Options:       models.DefaultOptions(),
	}
	err := repo.Create(ctx, limitation)
	require.NoError(t, err)

	// Find it
	found, err := repo.FindByLimiterAndLimited(ctx, 1, 2)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, limitation.ID, found.ID)
}

func TestProfileLimitationRepository_FindBetweenUsers(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProfileLimitationRepository(db)
	ctx := context.Background()

	// Create a limitation (user 1 limiting user 2)
	limitation := &models.ProfileLimitation{
		LimiterUserID: 1,
		LimitedUserID: 2,
		Options:       models.DefaultOptions(),
	}
	err := repo.Create(ctx, limitation)
	require.NoError(t, err)

	// Find it from both directions
	found1, err := repo.FindBetweenUsers(ctx, 1, 2)
	require.NoError(t, err)
	require.NotNil(t, found1)
	assert.Equal(t, limitation.ID, found1.ID)

	found2, err := repo.FindBetweenUsers(ctx, 2, 1)
	require.NoError(t, err)
	require.NotNil(t, found2)
	assert.Equal(t, limitation.ID, found2.ID)
}

func TestProfileLimitationRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProfileLimitationRepository(db)
	ctx := context.Background()

	// Create a limitation
	limitation := &models.ProfileLimitation{
		LimiterUserID: 1,
		LimitedUserID: 2,
		Options:       models.DefaultOptions(),
	}
	err := repo.Create(ctx, limitation)
	require.NoError(t, err)

	// Update it
	limitation.Options.Follow = false
	limitation.Options.SendMessage = false
	limitation.Note = sql.NullString{String: "Test note", Valid: true}

	err = repo.Update(ctx, limitation)
	require.NoError(t, err)

	// Verify update
	found, err := repo.FindByID(ctx, limitation.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.False(t, found.Options.Follow)
	assert.False(t, found.Options.SendMessage)
	assert.True(t, found.Note.Valid)
	assert.Equal(t, "Test note", found.Note.String)
}

func TestProfileLimitationRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProfileLimitationRepository(db)
	ctx := context.Background()

	// Create a limitation
	limitation := &models.ProfileLimitation{
		LimiterUserID: 1,
		LimitedUserID: 2,
		Options:       models.DefaultOptions(),
	}
	err := repo.Create(ctx, limitation)
	require.NoError(t, err)

	// Delete it
	err = repo.Delete(ctx, limitation.ID)
	require.NoError(t, err)

	// Verify deletion
	found, err := repo.FindByID(ctx, limitation.ID)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestProfileLimitationRepository_ExistsForLimiterAndLimited(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProfileLimitationRepository(db)
	ctx := context.Background()

	// Initially doesn't exist
	exists, err := repo.ExistsForLimiterAndLimited(ctx, 1, 2)
	require.NoError(t, err)
	assert.False(t, exists)

	// Create a limitation
	limitation := &models.ProfileLimitation{
		LimiterUserID: 1,
		LimitedUserID: 2,
		Options:       models.DefaultOptions(),
	}
	err = repo.Create(ctx, limitation)
	require.NoError(t, err)

	// Now it exists
	exists, err = repo.ExistsForLimiterAndLimited(ctx, 1, 2)
	require.NoError(t, err)
	assert.True(t, exists)

	// But not for reverse direction
	exists, err = repo.ExistsForLimiterAndLimited(ctx, 2, 1)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestProfileLimitationRepository_OptionsJSON(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewProfileLimitationRepository(db)
	ctx := context.Background()

	// Create with custom options
	limitation := &models.ProfileLimitation{
		LimiterUserID: 1,
		LimitedUserID: 2,
		Options: models.ProfileLimitationOptions{
			Follow:                false,
			SendMessage:           false,
			Share:                 true,
			SendTicket:            true,
			ViewProfileImages:     false,
			ViewFeaturesLocations: true,
		},
	}
	err := repo.Create(ctx, limitation)
	require.NoError(t, err)

	// Retrieve and verify all options
	found, err := repo.FindByID(ctx, limitation.ID)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.False(t, found.Options.Follow)
	assert.False(t, found.Options.SendMessage)
	assert.True(t, found.Options.Share)
	assert.True(t, found.Options.SendTicket)
	assert.False(t, found.Options.ViewProfileImages)
	assert.True(t, found.Options.ViewFeaturesLocations)
}
