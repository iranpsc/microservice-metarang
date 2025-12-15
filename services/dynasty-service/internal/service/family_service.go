package service

import (
	"context"
	"fmt"

	"metargb/dynasty-service/internal/models"
	"metargb/dynasty-service/internal/repository"
)

type FamilyService struct {
	familyRepo  *repository.FamilyRepository
	dynastyRepo *repository.DynastyRepository
}

func NewFamilyService(
	familyRepo *repository.FamilyRepository,
	dynastyRepo *repository.DynastyRepository,
) *FamilyService {
	return &FamilyService{
		familyRepo:  familyRepo,
		dynastyRepo: dynastyRepo,
	}
}

// GetFamily retrieves a family by ID or dynasty ID
func (s *FamilyService) GetFamily(ctx context.Context, familyID, dynastyID uint64) (*models.Family, error) {
	if familyID > 0 {
		return s.familyRepo.GetFamilyByID(ctx, familyID)
	} else if dynastyID > 0 {
		return s.familyRepo.GetFamilyByDynastyID(ctx, dynastyID)
	}
	return nil, fmt.Errorf("either familyID or dynastyID must be provided")
}

// GetFamilyMembers retrieves all members of a family
func (s *FamilyService) GetFamilyMembers(ctx context.Context, familyID uint64, page, perPage int32) ([]*models.FamilyMember, int32, error) {
	return s.familyRepo.GetFamilyMembers(ctx, familyID, page, perPage)
}

// GetUserBasicInfo retrieves basic user information
func (s *FamilyService) GetUserBasicInfo(ctx context.Context, userID uint64) (*models.UserBasic, error) {
	return s.familyRepo.GetUserBasicInfo(ctx, userID)
}
