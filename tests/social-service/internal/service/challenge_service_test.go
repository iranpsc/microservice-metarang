package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"metargb/social-service/internal/models"
)

// Mock repositories
type mockChallengeRepository struct {
	getRandomUnansweredQuestionFunc   func(ctx context.Context, userID uint64) (*models.Question, error)
	getQuestionByIDFunc               func(ctx context.Context, questionID uint64) (*models.Question, error)
	getAnswersByQuestionIDFunc        func(ctx context.Context, questionID uint64) ([]*models.Answer, error)
	getCorrectAnswerIDFunc            func(ctx context.Context, questionID uint64) (uint64, error)
	incrementQuestionViewsFunc        func(ctx context.Context, questionID uint64) error
	incrementQuestionParticipantsFunc func(ctx context.Context, questionID uint64) error
	createUserAnswerFunc              func(ctx context.Context, userID, questionID, answerID uint64) error
	hasUserAnsweredCorrectlyFunc      func(ctx context.Context, userID, questionID uint64) (bool, error)
	getUserAnswerCountFunc            func(ctx context.Context, userID uint64, isCorrect bool) (int32, error)
	getTotalParticipantsCountFunc     func(ctx context.Context) (int32, error)
	getSystemVariableFunc             func(ctx context.Context, slug string) (float64, error)
	getAnswerVoteCountFunc            func(ctx context.Context, answerID uint64) (int32, error)
	getQuestionTotalAnswersFunc       func(ctx context.Context, questionID uint64) (int32, error)
}

func (m *mockChallengeRepository) GetRandomUnansweredQuestion(ctx context.Context, userID uint64) (*models.Question, error) {
	if m.getRandomUnansweredQuestionFunc != nil {
		return m.getRandomUnansweredQuestionFunc(ctx, userID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockChallengeRepository) GetQuestionByID(ctx context.Context, questionID uint64) (*models.Question, error) {
	if m.getQuestionByIDFunc != nil {
		return m.getQuestionByIDFunc(ctx, questionID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockChallengeRepository) GetAnswersByQuestionID(ctx context.Context, questionID uint64) ([]*models.Answer, error) {
	if m.getAnswersByQuestionIDFunc != nil {
		return m.getAnswersByQuestionIDFunc(ctx, questionID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockChallengeRepository) GetCorrectAnswerID(ctx context.Context, questionID uint64) (uint64, error) {
	if m.getCorrectAnswerIDFunc != nil {
		return m.getCorrectAnswerIDFunc(ctx, questionID)
	}
	return 0, errors.New("not implemented")
}

func (m *mockChallengeRepository) IncrementQuestionViews(ctx context.Context, questionID uint64) error {
	if m.incrementQuestionViewsFunc != nil {
		return m.incrementQuestionViewsFunc(ctx, questionID)
	}
	return errors.New("not implemented")
}

func (m *mockChallengeRepository) IncrementQuestionParticipants(ctx context.Context, questionID uint64) error {
	if m.incrementQuestionParticipantsFunc != nil {
		return m.incrementQuestionParticipantsFunc(ctx, questionID)
	}
	return errors.New("not implemented")
}

func (m *mockChallengeRepository) CreateUserAnswer(ctx context.Context, userID, questionID, answerID uint64) error {
	if m.createUserAnswerFunc != nil {
		return m.createUserAnswerFunc(ctx, userID, questionID, answerID)
	}
	return errors.New("not implemented")
}

func (m *mockChallengeRepository) HasUserAnsweredCorrectly(ctx context.Context, userID, questionID uint64) (bool, error) {
	if m.hasUserAnsweredCorrectlyFunc != nil {
		return m.hasUserAnsweredCorrectlyFunc(ctx, userID, questionID)
	}
	return false, errors.New("not implemented")
}

func (m *mockChallengeRepository) GetUserAnswerCount(ctx context.Context, userID uint64, isCorrect bool) (int32, error) {
	if m.getUserAnswerCountFunc != nil {
		return m.getUserAnswerCountFunc(ctx, userID, isCorrect)
	}
	return 0, errors.New("not implemented")
}

func (m *mockChallengeRepository) GetTotalParticipantsCount(ctx context.Context) (int32, error) {
	if m.getTotalParticipantsCountFunc != nil {
		return m.getTotalParticipantsCountFunc(ctx)
	}
	return 0, errors.New("not implemented")
}

func (m *mockChallengeRepository) GetSystemVariable(ctx context.Context, slug string) (float64, error) {
	if m.getSystemVariableFunc != nil {
		return m.getSystemVariableFunc(ctx, slug)
	}
	return 15.0, nil // Default
}

func (m *mockChallengeRepository) GetAnswerVoteCount(ctx context.Context, answerID uint64) (int32, error) {
	if m.getAnswerVoteCountFunc != nil {
		return m.getAnswerVoteCountFunc(ctx, answerID)
	}
	return 0, errors.New("not implemented")
}

func (m *mockChallengeRepository) GetQuestionTotalAnswers(ctx context.Context, questionID uint64) (int32, error) {
	if m.getQuestionTotalAnswersFunc != nil {
		return m.getQuestionTotalAnswersFunc(ctx, questionID)
	}
	return 0, errors.New("not implemented")
}

type mockCommercialClient struct {
	addBalanceFunc func(ctx context.Context, userID uint64, asset string, amount float64) error
}

func (m *mockCommercialClient) AddBalance(ctx context.Context, userID uint64, asset string, amount float64) error {
	if m.addBalanceFunc != nil {
		return m.addBalanceFunc(ctx, userID, asset, amount)
	}
	return nil
}

func (m *mockCommercialClient) Close() error {
	return nil
}

func TestChallengeService_GetTimings(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get timings", func(t *testing.T) {
		repo := &mockChallengeRepository{}
		repo.getSystemVariableFunc = func(ctx context.Context, slug string) (float64, error) {
			return 15.0, nil
		}
		repo.getTotalParticipantsCountFunc = func(ctx context.Context) (int32, error) {
			return 100, nil
		}
		repo.getUserAnswerCountFunc = func(ctx context.Context, userID uint64, isCorrect bool) (int32, error) {
			if isCorrect {
				return 5, nil
			}
			return 3, nil
		}

		client := &mockCommercialClient{}
		service := NewChallengeService(repo, client)

		timings, err := service.GetTimings(ctx, 1)

		if err != nil {
			t.Fatalf("GetTimings failed: %v", err)
		}
		if timings.DisplayAdInterval != 15 {
			t.Fatalf("Expected DisplayAdInterval 15, got %d", timings.DisplayAdInterval)
		}
		if timings.Participants != 100 {
			t.Fatalf("Expected Participants 100, got %d", timings.Participants)
		}
		if timings.CorrectAnswers != 5 {
			t.Fatalf("Expected CorrectAnswers 5, got %d", timings.CorrectAnswers)
		}
		if timings.WrongAnswers != 3 {
			t.Fatalf("Expected WrongAnswers 3, got %d", timings.WrongAnswers)
		}
	})
}

func TestChallengeService_GetQuestion(t *testing.T) {
	ctx := context.Background()

	t.Run("successful get question", func(t *testing.T) {
		repo := &mockChallengeRepository{}
		repo.getRandomUnansweredQuestionFunc = func(ctx context.Context, userID uint64) (*models.Question, error) {
			return &models.Question{
				ID:           1,
				Code:         "Q1",
				Title:        "Test Question",
				Image:        "question.jpg",
				CreatorCode:  "USER1",
				Prize:        25,
				Views:        100,
				Participants: 50,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}, nil
		}
		repo.incrementQuestionViewsFunc = func(ctx context.Context, questionID uint64) error {
			return nil
		}
		repo.getAnswersByQuestionIDFunc = func(ctx context.Context, questionID uint64) ([]*models.Answer, error) {
			return []*models.Answer{
				{
					ID:         1,
					QuestionID: 1,
					Title:      "Answer 1",
					Image:      "answer1.jpg",
					IsCorrect:  true,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
				{
					ID:         2,
					QuestionID: 1,
					Title:      "Answer 2",
					Image:      "answer2.jpg",
					IsCorrect:  false,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
			}, nil
		}

		client := &mockCommercialClient{}
		service := NewChallengeService(repo, client)

		question, err := service.GetQuestion(ctx, 1)

		if err != nil {
			t.Fatalf("GetQuestion failed: %v", err)
		}
		if question.ID != 1 {
			t.Fatalf("Expected question ID 1, got %d", question.ID)
		}
		if len(question.Answers) != 2 {
			t.Fatalf("Expected 2 answers, got %d", len(question.Answers))
		}
		// Answers should not have is_correct or vote_percentage in GetQuestion response
		if question.Answers[0].IsCorrect {
			t.Fatal("Answers should not have is_correct in GetQuestion response")
		}
	})

	t.Run("no unanswered questions", func(t *testing.T) {
		repo := &mockChallengeRepository{}
		repo.getRandomUnansweredQuestionFunc = func(ctx context.Context, userID uint64) (*models.Question, error) {
			return nil, nil // No question found
		}

		client := &mockCommercialClient{}
		service := NewChallengeService(repo, client)

		_, err := service.GetQuestion(ctx, 1)

		if err == nil {
			t.Fatal("Expected error when no unanswered questions")
		}
		if err != ErrNoUnansweredQuestions {
			t.Fatalf("Expected ErrNoUnansweredQuestions, got: %v", err)
		}
	})
}

func TestChallengeService_SubmitAnswer(t *testing.T) {
	ctx := context.Background()

	t.Run("successful correct answer", func(t *testing.T) {
		repo := &mockChallengeRepository{}
		repo.getQuestionByIDFunc = func(ctx context.Context, questionID uint64) (*models.Question, error) {
			return &models.Question{
				ID:           1,
				Code:         "Q1",
				Title:        "Test Question",
				Image:        "question.jpg",
				CreatorCode:  "USER1",
				Prize:        25,
				Views:        100,
				Participants: 50,
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}, nil
		}
		repo.getAnswersByQuestionIDFunc = func(ctx context.Context, questionID uint64) ([]*models.Answer, error) {
			return []*models.Answer{
				{
					ID:         1,
					QuestionID: 1,
					Title:      "Correct Answer",
					Image:      "answer1.jpg",
					IsCorrect:  true,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
				{
					ID:         2,
					QuestionID: 1,
					Title:      "Wrong Answer",
					Image:      "answer2.jpg",
					IsCorrect:  false,
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
			}, nil
		}
		repo.hasUserAnsweredCorrectlyFunc = func(ctx context.Context, userID, questionID uint64) (bool, error) {
			return false, nil // Not answered correctly before
		}
		repo.createUserAnswerFunc = func(ctx context.Context, userID, questionID, answerID uint64) error {
			return nil
		}
		repo.incrementQuestionParticipantsFunc = func(ctx context.Context, questionID uint64) error {
			return nil
		}
		repo.getQuestionTotalAnswersFunc = func(ctx context.Context, questionID uint64) (int32, error) {
			return 10, nil
		}
		repo.getAnswerVoteCountFunc = func(ctx context.Context, answerID uint64) (int32, error) {
			if answerID == 1 {
				return 7, nil // 7 votes for correct answer
			}
			return 3, nil // 3 votes for wrong answer
		}

		creditCalled := false
		client := &mockCommercialClient{}
		client.addBalanceFunc = func(ctx context.Context, userID uint64, asset string, amount float64) error {
			creditCalled = true
			if userID != 1 {
				t.Fatalf("Expected userID 1, got %d", userID)
			}
			if asset != "psc" {
				t.Fatalf("Expected asset 'psc', got %s", asset)
			}
			if amount != 25.0 {
				t.Fatalf("Expected amount 25.0, got %f", amount)
			}
			return nil
		}

		service := NewChallengeService(repo, client)

		question, err := service.SubmitAnswer(ctx, 1, 1, 1) // User 1, Question 1, Answer 1 (correct)

		if err != nil {
			t.Fatalf("SubmitAnswer failed: %v", err)
		}
		if question.ID != 1 {
			t.Fatalf("Expected question ID 1, got %d", question.ID)
		}
		if !creditCalled {
			t.Fatal("Expected AddBalance to be called for correct answer")
		}
		// Check that answers now have is_correct and vote_percentage
		if !question.Answers[0].IsCorrect {
			t.Fatal("Expected first answer to be marked as correct")
		}
		if question.Answers[0].VotePercentage != 70 {
			t.Fatalf("Expected vote percentage 70, got %d", question.Answers[0].VotePercentage)
		}
	})

	t.Run("already answered correctly", func(t *testing.T) {
		repo := &mockChallengeRepository{}
		repo.getQuestionByIDFunc = func(ctx context.Context, questionID uint64) (*models.Question, error) {
			return &models.Question{
				ID: 1,
			}, nil
		}
		repo.getAnswersByQuestionIDFunc = func(ctx context.Context, questionID uint64) ([]*models.Answer, error) {
			return []*models.Answer{
				{ID: 1, QuestionID: 1, IsCorrect: true},
			}, nil
		}
		repo.hasUserAnsweredCorrectlyFunc = func(ctx context.Context, userID, questionID uint64) (bool, error) {
			return true, nil // Already answered correctly
		}

		client := &mockCommercialClient{}
		service := NewChallengeService(repo, client)

		_, err := service.SubmitAnswer(ctx, 1, 1, 1)

		if err == nil {
			t.Fatal("Expected error when already answered correctly")
		}
		if err != ErrAlreadyAnswered {
			t.Fatalf("Expected ErrAlreadyAnswered, got: %v", err)
		}
	})

	t.Run("answer mismatch", func(t *testing.T) {
		repo := &mockChallengeRepository{}
		repo.getQuestionByIDFunc = func(ctx context.Context, questionID uint64) (*models.Question, error) {
			return &models.Question{ID: 1}, nil
		}
		repo.getAnswersByQuestionIDFunc = func(ctx context.Context, questionID uint64) ([]*models.Answer, error) {
			return []*models.Answer{
				{ID: 1, QuestionID: 1},
			}, nil
		}
		repo.hasUserAnsweredCorrectlyFunc = func(ctx context.Context, userID, questionID uint64) (bool, error) {
			return false, nil
		}

		client := &mockCommercialClient{}
		service := NewChallengeService(repo, client)

		_, err := service.SubmitAnswer(ctx, 1, 1, 999) // Answer 999 doesn't exist

		if err == nil {
			t.Fatal("Expected error when answer doesn't belong to question")
		}
		if err != ErrAnswerMismatch {
			t.Fatalf("Expected ErrAnswerMismatch, got: %v", err)
		}
	})
}
