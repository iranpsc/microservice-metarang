package repository

import (
	"context"
	"database/sql"
	"fmt"
)

type ValidationRepository struct {
	db *sql.DB
}

func NewValidationRepository(db *sql.DB) *ValidationRepository {
	return &ValidationRepository{db: db}
}

// CheckPendingRequest checks if there's a pending join request between two users
func (r *ValidationRepository) CheckPendingRequest(ctx context.Context, fromUser, toUser uint64) (bool, error) {
	query := `SELECT EXISTS(
		SELECT 1 FROM join_requests 
		WHERE from_user = ? AND to_user = ? AND status = 0
	)`
	
	var exists bool
	err := r.db.QueryRowContext(ctx, query, fromUser, toUser).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check pending request: %w", err)
	}
	
	return exists, nil
}

// CheckRejectedRequest checks if there's a rejected join request between two users
func (r *ValidationRepository) CheckRejectedRequest(ctx context.Context, fromUser, toUser uint64) (bool, error) {
	query := `SELECT EXISTS(
		SELECT 1 FROM join_requests 
		WHERE from_user = ? AND to_user = ? AND status = -1
	)`
	
	var exists bool
	err := r.db.QueryRowContext(ctx, query, fromUser, toUser).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check rejected request: %w", err)
	}
	
	return exists, nil
}

// CheckUserInFamily checks if a user is already in a family (as non-owner)
func (r *ValidationRepository) CheckUserInFamily(ctx context.Context, userID uint64) (bool, error) {
	query := `SELECT EXISTS(
		SELECT 1 FROM family_members 
		WHERE user_id = ? AND relationship != 'owner'
	)`
	
	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check user in family: %w", err)
	}
	
	return exists, nil
}

// CountFamilyMembersByRelationship counts family members with specific relationship
func (r *ValidationRepository) CountFamilyMembersByRelationship(ctx context.Context, familyID uint64, relationship string) (int, error) {
	query := `SELECT COUNT(*) FROM family_members 
	          WHERE family_id = ? AND relationship = ?`
	
	var count int
	err := r.db.QueryRowContext(ctx, query, familyID, relationship).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count family members: %w", err)
	}
	
	return count, nil
}

// GetFamilyByDynastyID gets family for a dynasty
func (r *ValidationRepository) GetFamilyByDynastyID(ctx context.Context, dynastyID uint64) (uint64, error) {
	query := `SELECT id FROM families WHERE dynasty_id = ?`
	
	var familyID uint64
	err := r.db.QueryRowContext(ctx, query, dynastyID).Scan(&familyID)
	if err != nil {
		return 0, fmt.Errorf("failed to get family: %w", err)
	}
	
	return familyID, nil
}

// CheckUserHasDynasty checks if user has a dynasty
func (r *ValidationRepository) CheckUserHasDynasty(ctx context.Context, userID uint64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM dynasties WHERE user_id = ?)`
	
	var exists bool
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check dynasty: %w", err)
	}
	
	return exists, nil
}

// CheckUserDMPermission checks if under-18 user has verified DM permission
func (r *ValidationRepository) CheckUserDMPermission(ctx context.Context, userID uint64) (bool, error) {
	query := `SELECT verified AND DM FROM children_permissions WHERE user_id = ?`
	
	var hasPermission bool
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&hasPermission)
	if err == sql.ErrNoRows {
		return false, nil // No permission record means no permission
	}
	if err != nil {
		return false, fmt.Errorf("failed to check DM permission: %w", err)
	}
	
	return hasPermission, nil
}

