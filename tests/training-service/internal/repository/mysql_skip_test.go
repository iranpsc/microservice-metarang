package repository_test

import (
	"context"
	"testing"

	"metarang/training-service/internal/repository"
	"metarang/training-service/tests/internal/testutil"
)

func TestCategoryRepository_GetCategories_Integration(t *testing.T) {
	db := testutil.OpenMySQLOrSkip(t)
	defer db.Close()

	r := repository.NewCategoryRepository(db)
	_, _, err := r.GetCategories(context.Background(), 1, 5)
	if err != nil {
		t.Fatalf("integration GetCategories: %v", err)
	}
}
