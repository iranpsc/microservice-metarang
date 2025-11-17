package service

import (
	"context"
	"fmt"

	"metargb/levels-service/internal/repository"
	pb "metargb/shared/pb/levels"
)

type ChallengeService struct {
	challengeRepo *repository.ChallengeRepository
}

func NewChallengeService(challengeRepo *repository.ChallengeRepository) *ChallengeService {
	return &ChallengeService{
		challengeRepo: challengeRepo,
	}
}

// GetQuestion retrieves a random unanswered question for the user
// Implements Laravel: ChallengeController@getQuestion
func (s *ChallengeService) GetQuestion(ctx context.Context, userID uint64) (*pb.Question, bool, error) {
	// Get random unanswered question
	// Laravel: while loop in selectQuestion method
	question, err := s.challengeRepo.GetRandomUnansweredQuestion(ctx, userID)
	if err != nil {
		return nil, false, err
	}

	if question == nil {
		return nil, false, nil
	}

	// Increment views
	// Laravel: $question->increment('views')
	if err := s.challengeRepo.IncrementViews(ctx, question.Id); err != nil {
		return question, true, err
	}

	return question, true, nil
}

// SubmitAnswer submits an answer and returns result
// Implements Laravel: ChallengeController@answerResult
func (s *ChallengeService) SubmitAnswer(ctx context.Context, userID, questionID, answerID uint64) (bool, string, *pb.Question, error) {
	// Validate answer belongs to question
	// Laravel: if ($answer->question->isNot($question))
	isValid, err := s.challengeRepo.ValidateAnswer(ctx, questionID, answerID)
	if err != nil || !isValid {
		return false, "", nil, fmt.Errorf("answer is not valid")
	}

	// Check if user has already answered this question (authorization)
	// Laravel: $this->authorize('answer', $question)
	hasAnswered, err := s.challengeRepo.HasUserAnsweredQuestion(ctx, userID, questionID)
	if err != nil {
		return false, "", nil, err
	}
	if hasAnswered {
		return false, "", nil, fmt.Errorf("question already answered")
	}

	// Record user's answer
	// Laravel: UserQuestionAnswer::create([...])
	if err := s.challengeRepo.RecordUserAnswer(ctx, userID, questionID, answerID); err != nil {
		return false, "", nil, err
	}

	// Increment participants count
	// Laravel: $question->increment('participants')
	if err := s.challengeRepo.IncrementParticipants(ctx, questionID); err != nil {
		return false, "", nil, err
	}

	// Check if answer is correct
	// Laravel: if ($answer->isCorrect())
	isCorrect, prize, err := s.challengeRepo.CheckAnswer(ctx, answerID, questionID)
	if err != nil {
		return false, "", nil, err
	}

	prizeAwarded := "0"
	if isCorrect {
		// Award prize to user wallet
		// Laravel: $request->user()->wallet->increment('psc', $question->prize)
		// TODO: Call commercial service to increment wallet PSC
		prizeAwarded = prize
	}

	// Get question with answers to return
	question, err := s.challengeRepo.GetQuestionByID(ctx, questionID)
	if err != nil {
		return isCorrect, prizeAwarded, nil, err
	}

	return isCorrect, prizeAwarded, question, nil
}

// GetTimings retrieves challenge configuration and user stats
// Implements Laravel: ChallengeController@getTimings
func (s *ChallengeService) GetTimings(ctx context.Context, userID uint64) (*pb.TimingsResponse, error) {
	// Get system variables for intervals
	// Laravel: SystemVariable::getByKey('challenge_display_ad_interval') ?? 15
	adInterval, questionInterval, answerInterval, err := s.challengeRepo.GetChallengeIntervals(ctx)
	if err != nil {
		// Use defaults on error
		adInterval, questionInterval, answerInterval = 15, 15, 15
	}

	// Get user's correct and wrong answers
	// Laravel: $this->getCorrectAnswers() and $this->getWrongAnswers()
	correct, wrong, err := s.challengeRepo.GetUserAnswerCounts(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get total participants
	// Laravel: UserQuestionAnswer::distinct()->count('user_id')
	participants, err := s.challengeRepo.GetTotalParticipants(ctx)
	if err != nil {
		participants = 0
	}

	return &pb.TimingsResponse{
		DisplayAdInterval:       adInterval,
		DisplayQuestionInterval: questionInterval,
		DisplayAnswerInterval:   answerInterval,
		Participants:            participants,
		CorrectAnswers:          correct,
		WrongAnswers:            wrong,
	}, nil
}
