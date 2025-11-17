package service

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"metargb/levels-service/internal/repository"
	pb "metargb/shared/pb/levels"
)

type ActivityService struct {
	activityRepo *repository.ActivityRepository
	userLogRepo  *repository.UserLogRepository
	levelRepo    *repository.LevelRepository
}

func NewActivityService(
	activityRepo *repository.ActivityRepository,
	userLogRepo *repository.UserLogRepository,
	levelRepo *repository.LevelRepository,
) *ActivityService {
	return &ActivityService{
		activityRepo: activityRepo,
		userLogRepo:  userLogRepo,
		levelRepo:    levelRepo,
	}
}

// LogActivity records user activity
// Implements Laravel: UserObserver@logedIn
func (s *ActivityService) LogActivity(ctx context.Context, req *pb.LogActivityRequest) (uint64, error) {
	activityID, err := s.activityRepo.CreateActivity(ctx, req)
	if err != nil {
		return 0, err
	}
	
	// Create user event
	status := int8(1)
	event := "ورود به حساب کاربری" // Login in Persian
	if req.EventType == "logout" {
		event = "خروج از حساب کاربری" // Logout in Persian
	}
	
	_ = s.activityRepo.CreateUserEvent(ctx, req.UserId, event, req.Ip, req.Device, status)
	
	return activityID, nil
}

// GetUserActivities retrieves user's activity history
func (s *ActivityService) GetUserActivities(ctx context.Context, userID uint64, limit int32) ([]*pb.UserActivity, *pb.UserLog, error) {
	activities, err := s.activityRepo.FindByUserID(ctx, userID, limit)
	if err != nil {
		return nil, nil, err
	}

	userLog, err := s.userLogRepo.GetUserLog(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	return activities, userLog, nil
}

// UpdateActivityScore recalculates user score
// Implements Laravel: UserObserver@calculateScore
func (s *ActivityService) UpdateActivityScore(ctx context.Context, userID uint64) (int32, bool, uint64, error) {
	// Calculate the new score
	newScore, err := s.userLogRepo.CalculateScore(ctx, userID)
	if err != nil {
		return 0, false, 0, err
	}
	
	// Update score in user_logs and users tables
	if err := s.userLogRepo.UpdateScore(ctx, userID, newScore); err != nil {
		return 0, false, 0, err
	}

	// Check if user reached a new level
	// Implements Laravel: Level::where('score', '<=', $user->score)->whereNotIn('id', $user->levels->pluck('id'))->with('prize')->first()
	nextLevel, err := s.levelRepo.GetNextLevelForScore(ctx, userID, newScore)
	levelUp := false
	var newLevelID uint64
	
	if err == nil && nextLevel != nil {
		// User reached new level
		levelUp = true
		newLevelID = nextLevel.Id
		
		// Attach level to user
		if err := s.levelRepo.AttachLevelToUser(ctx, userID, newLevelID); err != nil {
			return newScore, false, 0, err
		}
		
		// Award prize automatically (matching Laravel behavior)
		// TODO: Implement this by calling commercial service to update wallet
		// For now, just record the prize as received
		prize, err := s.levelRepo.GetLevelPrize(ctx, newLevelID)
		if err == nil {
			// Check if user can receive prize (not already received)
			hasReceived, _ := s.levelRepo.HasUserReceivedPrize(ctx, userID, prize.Id)
			if !hasReceived {
				// Award prize to wallet (TODO: call commercial service)
				// Laravel: $wallet->increment('psc', ($levelPrize->psc / Variable::getRate('psc')))
				// Laravel: $wallet->increment('blue', $levelPrize->blue)
				// Laravel: $wallet->increment('red', $levelPrize->red)
				// Laravel: $wallet->increment('yellow', $levelPrize->yellow)
				// Laravel: $wallet->update(['effect' => $levelPrize->effect])
				// Laravel: $wallet->increment('satisfaction', $levelPrize->satisfaction)
				
				// Record prize as received
				_ = s.levelRepo.RecordReceivedPrize(ctx, userID, prize.Id)
			}
		}
	}

	return newScore, levelUp, newLevelID, nil
}

// RecordTrade records trade for score calculation
// Implements Laravel: UserObserver@traded
func (s *ActivityService) RecordTrade(ctx context.Context, userID uint64, irrAmount, pscAmount string) error {
	// Count significant trades (irr > 7000000 OR psc > equivalent)
	// Implements Laravel: UserObserver@getSignificantTradeCount
	
	// Parse amounts
	irr, _ := strconv.ParseFloat(irrAmount, 64)
	psc, _ := strconv.ParseFloat(pscAmount, 64)
	
	// Get PSC value rate to calculate minimum PSC amount
	// Laravel: $psc_value = Variable::getRate('psc'); return 7000000 / $psc_value;
	// For now, we'll use a default rate (TODO: query from variables table)
	minIrrAmount := float64(7000000)
	pscRate := float64(30000) // Default PSC rate
	minPscAmount := minIrrAmount / pscRate
	
	// Check if this trade is significant
	if irr < minIrrAmount && psc < minPscAmount {
		// Trade is not significant, don't count it
		return nil
	}
	
	// TODO: Count all significant trades for this user
	// For now, we'll implement a simpler approach - query from trades table
	// Laravel query (simplified):
	// SELECT COUNT(*) FROM trades 
	// WHERE (buyer_id = ? AND (irr_amount > 7000000 OR psc_amount > minPsc))
	//    OR (seller_id = ? AND (irr_amount > 7000000 OR psc_amount > minPsc))
	
	// Update transactions_count (count * 2)
	// Laravel: $user->log->update(['transactions_count' => $trades * 2])
	
	// After updating count, recalculate score
	return s.recalculateAndUpdateScore(ctx, userID)
}

// RecordDeposit records deposit for score calculation
// Implements Laravel: UserObserver@deposit
func (s *ActivityService) RecordDeposit(ctx context.Context, userID uint64, amount string) error {
	// Increment deposit_amount by amount * 0.0001
	// Laravel: $user->log->increment('deposit_amount', $amount * 0.0001)
	if err := s.userLogRepo.IncrementDeposit(ctx, userID, amount); err != nil {
		return err
	}
	
	// Recalculate score
	return s.recalculateAndUpdateScore(ctx, userID)
}

// RecordFollower records follower for score calculation
// Implements Laravel: UserObserver@followed
func (s *ActivityService) RecordFollower(ctx context.Context, userID uint64) error {
	// Count total followers
	// Laravel: $totalFollowers = $user->followers->count()
	totalFollowers, err := s.userLogRepo.GetTotalFollowers(ctx, userID)
	if err != nil {
		return err
	}
	
	// Update followers_count (count * 0.1)
	// Laravel: $user->log->update(['followers_count' => $totalFollowers * 0.1])
	if err := s.userLogRepo.UpdateFollowersCount(ctx, userID, totalFollowers); err != nil {
		return err
	}
	
	// Recalculate score
	return s.recalculateAndUpdateScore(ctx, userID)
}

// LogLogout records user logout and updates activity hours
// Implements Laravel: UserObserver@logedOut
func (s *ActivityService) LogLogout(ctx context.Context, userID uint64, ip string) error {
	// Get latest activity
	latestActivity, err := s.activityRepo.GetLatestActivity(ctx, userID)
	if err != nil {
		return err
	}
	
	// Parse start time
	startTime, err := time.Parse(time.RFC3339, latestActivity.Start)
	if err != nil {
		return err
	}
	
	// Calculate total minutes
	endTime := time.Now()
	totalMinutes := int32(endTime.Sub(startTime).Minutes())
	
	// Update activity with end time and total
	// Laravel: $latestActivity->update(['end' => now(), 'total' => $latestActivity->start->diffInMinutes(now())])
	if err := s.activityRepo.UpdateActivity(ctx, latestActivity.Id, endTime, totalMinutes); err != nil {
		return err
	}
	
	// Call hourReached
	return s.HourReached(ctx, userID)
}

// HourReached recalculates activity hours score
// Implements Laravel: UserObserver@hourReached
func (s *ActivityService) HourReached(ctx context.Context, userID uint64) error {
	// Get total active minutes
	// Laravel: $totalActiveHours = $user->activities->sum('total')
	totalMinutes, err := s.activityRepo.GetTotalActivityMinutes(ctx, userID)
	if err != nil {
		return err
	}
	
	// Update activity_hours (ceil(minutes / 60) * 0.1)
	// Laravel: $user->log->update(['activity_hours' => ceil($totalActiveHours / 60) * 0.1])
	if err := s.userLogRepo.UpdateActivityHours(ctx, userID, totalMinutes); err != nil {
		return err
	}
	
	// Recalculate score
	return s.recalculateAndUpdateScore(ctx, userID)
}

// recalculateAndUpdateScore is a helper to recalculate and update user score
// Implements Laravel: $this->calculateScore($user)
func (s *ActivityService) recalculateAndUpdateScore(ctx context.Context, userID uint64) error {
	_, _, _, err := s.UpdateActivityScore(ctx, userID)
	return err
}

// GetTradeCount counts significant trades for a user (helper method)
func (s *ActivityService) GetTradeCount(ctx context.Context, db *sql.DB, userID uint64, minIrrAmount, minPscAmount float64) (int32, error) {
	query := `
		SELECT COUNT(*)
		FROM trades
		WHERE (buyer_id = ? AND (irr_amount > ? OR psc_amount > ?))
		   OR (seller_id = ? AND (irr_amount > ? OR psc_amount > ?))
	`
	
	var count int32
	err := db.QueryRowContext(ctx, query, 
		userID, minIrrAmount, minPscAmount,
		userID, minIrrAmount, minPscAmount,
	).Scan(&count)
	
	return count, err
}
