package policy

import (
	"context"

	"metargb/dynasty-service/internal/models"
)

// JoinRequestPolicy enforces join request authorization rules
type JoinRequestPolicy struct {
}

func NewJoinRequestPolicy() *JoinRequestPolicy {
	return &JoinRequestPolicy{}
}

// CanView checks if user can view a join request
// Implements JoinRequestPolicy::view from Laravel
func (p *JoinRequestPolicy) CanView(
	ctx context.Context,
	userID uint64,
	request *models.JoinRequest,
) bool {
	// User must be either sender or receiver
	return request.FromUser == userID || request.ToUser == userID
}

// CanDelete checks if user can delete a join request
// Implements JoinRequestPolicy::delete from Laravel
func (p *JoinRequestPolicy) CanDelete(
	ctx context.Context,
	userID uint64,
	request *models.JoinRequest,
) bool {
	// Only sender can delete, and only if status is pending (0)
	return request.FromUser == userID && request.Status == 0
}

// CanAccept checks if user can accept a join request
// Implements JoinRequestPolicy::accept from Laravel
func (p *JoinRequestPolicy) CanAccept(
	ctx context.Context,
	userID uint64,
	request *models.JoinRequest,
) bool {
	// Only receiver can accept, and only if status is pending (0)
	return request.ToUser == userID && request.Status == 0
}

// CanReject checks if user can reject a join request
// Implements JoinRequestPolicy::reject from Laravel
func (p *JoinRequestPolicy) CanReject(
	ctx context.Context,
	userID uint64,
	request *models.JoinRequest,
) bool {
	// Only receiver can reject, and only if status is pending (0)
	return request.ToUser == userID && request.Status == 0
}

