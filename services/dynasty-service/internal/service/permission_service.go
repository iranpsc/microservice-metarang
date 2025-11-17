package service

import (
	"context"
	"fmt"

	"metargb/dynasty-service/internal/models"
	"metargb/dynasty-service/internal/repository"
)

// PermissionService manages children permissions
type PermissionService struct {
	joinRequestRepo *repository.JoinRequestRepository
	familyRepo      *repository.FamilyRepository
	dynastyRepo     *repository.DynastyRepository
}

func NewPermissionService(
	joinRequestRepo *repository.JoinRequestRepository,
	familyRepo *repository.FamilyRepository,
	dynastyRepo *repository.DynastyRepository,
) *PermissionService {
	return &PermissionService{
		joinRequestRepo: joinRequestRepo,
		familyRepo:      familyRepo,
		dynastyRepo:     dynastyRepo,
	}
}

// UpdateChildPermission updates a single permission for a child
func (s *PermissionService) UpdateChildPermission(
	ctx context.Context,
	parentUserID, childUserID uint64,
	permission string,
	status bool,
) error {
	// Check if parent can control child (policy)
	canControl, err := s.CanControlPermissions(ctx, parentUserID, childUserID)
	if err != nil {
		return fmt.Errorf("failed to check permissions: %w", err)
	}
	if !canControl {
		return fmt.Errorf("شما مجاز به تغییر دسترسی این کاربر نیستید")
	}
	
	// Get existing permissions
	existingPerm, err := s.joinRequestRepo.GetChildPermission(ctx, childUserID)
	if err != nil {
		return fmt.Errorf("failed to get permissions: %w", err)
	}
	if existingPerm == nil {
		return fmt.Errorf("child has no permission record")
	}
	
	// Update specific permission
	switch permission {
	case "BFR":
		existingPerm.BFR = status
	case "SF":
		existingPerm.SF = status
	case "W":
		existingPerm.W = status
	case "JU":
		existingPerm.JU = status
	case "DM":
		existingPerm.DM = status
	case "PIUP":
		existingPerm.PIUP = status
	case "PITC":
		existingPerm.PITC = status
	case "PIC":
		existingPerm.PIC = status
	case "ESOO":
		existingPerm.ESOO = status
	case "COTB":
		existingPerm.COTB = status
	default:
		return fmt.Errorf("invalid permission: %s", permission)
	}
	
	// Save updated permissions
	if err := s.joinRequestRepo.UpdateChildPermission(ctx, childUserID, existingPerm); err != nil {
		return fmt.Errorf("failed to update permission: %w", err)
	}
	
	return nil
}

// CanControlPermissions checks if parent can control child's permissions
// Implements UserPolicy::controlPermissions from Laravel
func (s *PermissionService) CanControlPermissions(
	ctx context.Context,
	parentUserID, childUserID uint64,
) (bool, error) {
	// Cannot control self
	if parentUserID == childUserID {
		return false, nil
	}
	
	// Child must be under 18
	isUnder18, err := s.joinRequestRepo.CheckUserAge(ctx, childUserID)
	if err != nil {
		return false, fmt.Errorf("failed to check child age: %w", err)
	}
	if !isUnder18 {
		return false, nil
	}
	
	// Get parent's dynasty
	dynasty, err := s.dynastyRepo.GetDynastyByUserID(ctx, parentUserID)
	if err != nil {
		return false, fmt.Errorf("failed to get dynasty: %w", err)
	}
	if dynasty == nil {
		return false, nil
	}
	
	// Get family
	family, err := s.familyRepo.GetFamilyByDynastyID(ctx, dynasty.ID)
	if err != nil {
		return false, fmt.Errorf("failed to get family: %w", err)
	}
	
	// Check if child is in family
	// Use large page size to get all members at once
	members, _, err := s.familyRepo.GetFamilyMembers(ctx, family.ID, 1, 1000)
	if err != nil {
		return false, fmt.Errorf("failed to get family members: %w", err)
	}
	
	for _, member := range members {
		if member.UserID == childUserID {
			return true, nil
		}
	}
	
	return false, nil
}

// GetDefaultPermissions retrieves default dynasty permissions
func (s *PermissionService) GetDefaultPermissions(ctx context.Context) (*models.DynastyPermission, error) {
	perm, err := s.joinRequestRepo.GetDynastyPermission(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default permissions: %w", err)
	}
	
	return perm, nil
}

// CheckPermission checks if user has a specific permission
func (s *PermissionService) CheckPermission(
	ctx context.Context,
	userID uint64,
	permission string,
) (bool, error) {
	// Check if user is under 18
	isUnder18, err := s.joinRequestRepo.CheckUserAge(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to check age: %w", err)
	}
	
	// If not under 18, all permissions are granted
	if !isUnder18 {
		return true, nil
	}
	
	// Get permissions
	perm, err := s.joinRequestRepo.GetChildPermission(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("failed to get permissions: %w", err)
	}
	if perm == nil {
		// No permissions record means no permissions
		return false, nil
	}
	
	// Check if verified
	if !perm.Verified {
		return false, nil
	}
	
	// Check specific permission
	switch permission {
	case "BFR":
		return perm.BFR, nil
	case "SF":
		return perm.SF, nil
	case "W":
		return perm.W, nil
	case "JU":
		return perm.JU, nil
	case "DM":
		return perm.DM, nil
	case "PIUP":
		return perm.PIUP, nil
	case "PITC":
		return perm.PITC, nil
	case "PIC":
		return perm.PIC, nil
	case "ESOO":
		return perm.ESOO, nil
	case "COTB":
		return perm.COTB, nil
	default:
		return false, fmt.Errorf("invalid permission: %s", permission)
	}
}

