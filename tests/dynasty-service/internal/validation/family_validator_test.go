package validation

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metargb/dynasty-service/internal/repository"
	"metargb/dynasty-service/internal/validation"
)

func TestFamilyValidator_ValidateRelationship(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	validationRepo := repository.NewValidationRepository(db)
	validator := validation.NewFamilyValidator(validationRepo)

	ctx := context.Background()

	validRelationships := []string{"father", "mother", "offspring", "husband", "wife", "brother", "sister"}
	for _, rel := range validRelationships {
		t.Run("Valid_"+rel, func(t *testing.T) {
			err := validator.ValidateRelationship(rel)
			assert.NoError(t, err)
		})
	}

	t.Run("Invalid", func(t *testing.T) {
		err := validator.ValidateRelationship("invalid")
		assert.Error(t, err)
	})
}

func TestFamilyValidator_ValidateRelationshipLimits(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	validationRepo := repository.NewValidationRepository(db)
	validator := validation.NewFamilyValidator(validationRepo)

	ctx := context.Background()
	familyID := uint64(1)

	t.Run("SingleParent_Father", func(t *testing.T) {
		// Count existing fathers
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM family_members").
			WithArgs(familyID, "father").
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

		err := validator.ValidateRelationshipLimits(ctx, familyID, "father")
		assert.NoError(t, err)
	})

	t.Run("SingleParent_Mother", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM family_members").
			WithArgs(familyID, "mother").
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

		err := validator.ValidateRelationshipLimits(ctx, familyID, "mother")
		assert.NoError(t, err)
	})

	t.Run("SingleSpouse_Husband", func(t *testing.T) {
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM family_members").
			WithArgs(familyID, "husband").
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))

		err := validator.ValidateRelationshipLimits(ctx, familyID, "husband")
		assert.NoError(t, err)
	})

	t.Run("MaxSpouse_Wife", func(t *testing.T) {
		// Already have 4 wives
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM family_members").
			WithArgs(familyID, "wife").
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(4))

		err := validator.ValidateRelationshipLimits(ctx, familyID, "wife")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "همسر")
	})

	t.Run("BrotherSister_NoLimit", func(t *testing.T) {
		// Brother/sister don't have limits in current implementation
		err := validator.ValidateRelationshipLimits(ctx, familyID, "brother")
		assert.NoError(t, err)

		err = validator.ValidateRelationshipLimits(ctx, familyID, "sister")
		assert.NoError(t, err)
	})

	t.Run("MaxOffspring", func(t *testing.T) {
		// Already have 4 offspring
		mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM family_members").
			WithArgs(familyID, "offspring").
			WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(4))

		err := validator.ValidateRelationshipLimits(ctx, familyID, "offspring")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "offspring")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFamilyValidator_ValidateAddFamilyMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	validationRepo := repository.NewValidationRepository(db)
	validator := validation.NewFamilyValidator(validationRepo)

	ctx := context.Background()
	fromUserID := uint64(1)
	toUserID := uint64(2)

	t.Run("NoPendingRequest", func(t *testing.T) {
		// Check pending request
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(fromUserID, toUserID).
			WillReturnRows(sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM join_requests"}).AddRow(false))

		// Check rejected request
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(fromUserID, toUserID).
			WillReturnRows(sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM join_requests"}).AddRow(false))

		// Check user in family
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(toUserID).
			WillReturnRows(sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM family_members"}).AddRow(false))

		err := validator.ValidateAddFamilyMember(ctx, fromUserID, toUserID, "offspring", false)
		assert.NoError(t, err)
	})

	t.Run("HasPendingRequest", func(t *testing.T) {
		// Has pending request
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(fromUserID, toUserID).
			WillReturnRows(sqlmock.NewRows([]string{"EXISTS(SELECT 1 FROM join_requests"}).AddRow(true))

		err := validator.ValidateAddFamilyMember(ctx, fromUserID, toUserID, "offspring", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pending")
	})

	require.NoError(t, mock.ExpectationsWereMet())
}

