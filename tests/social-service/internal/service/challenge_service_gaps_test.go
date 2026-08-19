package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"metarang/social-service/internal/models"
	"metarang/social-service/internal/service"
	"metarang/social-service/internal/testutil"
)

func TestChallengeService_GetTimings_FallbacksAndErrors(t *testing.T) {
	t.Run("system variable errors fall back to 15", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetSystemVariableFunc: func(context.Context, string) (float64, error) {
				return 0, errors.New("missing")
			},
			GetTotalParticipantsCountFunc: func(context.Context) (int32, error) { return 1, nil },
			GetTotalViewsCountFunc:        func(context.Context) (int32, error) { return 2, nil },
			GetUserAnswerCountFunc:        func(context.Context, uint64, bool) (int32, error) { return 0, nil },
		}
		svc := service.NewChallengeService(repo, nil)
		out, err := svc.GetTimings(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, int32(15), out.DisplayAdInterval)
		require.Equal(t, int32(15), out.DisplayQuestionInterval)
		require.Equal(t, int32(15), out.DisplayAnswerInterval)
	})

	t.Run("participants error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetTotalParticipantsCountFunc: func(context.Context) (int32, error) {
				return 0, errors.New("participants")
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.GetTimings(context.Background(), 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get participants count")
	})

	t.Run("views error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetTotalViewsCountFunc: func(context.Context) (int32, error) {
				return 0, errors.New("views")
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.GetTimings(context.Background(), 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get views count")
	})

	t.Run("correct answers error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetUserAnswerCountFunc: func(_ context.Context, _ uint64, isCorrect bool) (int32, error) {
				if isCorrect {
					return 0, errors.New("correct")
				}
				return 0, nil
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.GetTimings(context.Background(), 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get correct answers count")
	})

	t.Run("wrong answers error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetUserAnswerCountFunc: func(_ context.Context, _ uint64, isCorrect bool) (int32, error) {
				if !isCorrect {
					return 0, errors.New("wrong")
				}
				return 1, nil
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.GetTimings(context.Background(), 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get wrong answers count")
	})
}

func TestChallengeService_GetQuestion_ErrorPaths(t *testing.T) {
	t.Run("repo error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetRandomUnansweredQuestionFunc: func(context.Context, uint64) (*models.Question, error) {
				return nil, errors.New("db")
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.GetQuestion(context.Background(), 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get question")
	})

	t.Run("increment views error still returns question", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetRandomUnansweredQuestionFunc: func(context.Context, uint64) (*models.Question, error) {
				return &models.Question{ID: 3, Title: "Q"}, nil
			},
			IncrementQuestionViewsFunc: func(context.Context, uint64) error {
				return errors.New("views")
			},
			GetAnswersByQuestionIDFunc: func(context.Context, uint64) ([]*models.Answer, error) {
				return []*models.Answer{{ID: 1, Title: "A"}}, nil
			},
		}
		svc := service.NewChallengeService(repo, nil)
		q, err := svc.GetQuestion(context.Background(), 1)
		require.NoError(t, err)
		require.Equal(t, uint64(3), q.ID)
	})

	t.Run("get answers error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetRandomUnansweredQuestionFunc: func(context.Context, uint64) (*models.Question, error) {
				return &models.Question{ID: 3, Title: "Q"}, nil
			},
			GetAnswersByQuestionIDFunc: func(context.Context, uint64) ([]*models.Answer, error) {
				return nil, errors.New("answers")
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.GetQuestion(context.Background(), 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get answers")
	})
}

func TestChallengeService_SubmitAnswer_ErrorPaths(t *testing.T) {
	question := &models.Question{ID: 9, Title: "Q", Prize: 10, Participants: 1}
	answers := []*models.Answer{
		{ID: 1, QuestionID: 9, Title: "A", IsCorrect: true},
		{ID: 2, QuestionID: 9, Title: "B", IsCorrect: false},
	}

	t.Run("get question error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetQuestionByIDFunc: func(context.Context, uint64) (*models.Question, error) {
				return nil, errors.New("db")
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.SubmitAnswer(context.Background(), 1, 9, 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get question")
	})

	t.Run("get answers error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetQuestionByIDFunc: func(context.Context, uint64) (*models.Question, error) {
				return question, nil
			},
			GetAnswersByQuestionIDFunc: func(context.Context, uint64) ([]*models.Answer, error) {
				return nil, errors.New("answers")
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.SubmitAnswer(context.Background(), 1, 9, 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get answers")
	})

	t.Run("has answered error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetQuestionByIDFunc: func(context.Context, uint64) (*models.Question, error) {
				return question, nil
			},
			GetAnswersByQuestionIDFunc: func(context.Context, uint64) ([]*models.Answer, error) {
				return answers, nil
			},
			HasUserAnsweredFunc: func(context.Context, uint64, uint64) (bool, error) {
				return false, errors.New("check")
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.SubmitAnswer(context.Background(), 1, 9, 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to check previous answer")
	})

	t.Run("create user answer error", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetQuestionByIDFunc: func(context.Context, uint64) (*models.Question, error) {
				return question, nil
			},
			GetAnswersByQuestionIDFunc: func(context.Context, uint64) ([]*models.Answer, error) {
				return answers, nil
			},
			CreateUserAnswerFunc: func(context.Context, uint64, uint64, uint64) error {
				return errors.New("insert")
			},
		}
		svc := service.NewChallengeService(repo, nil)
		_, err := svc.SubmitAnswer(context.Background(), 1, 9, 1)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create user answer")
	})

	t.Run("increment participants and add balance errors still succeed", func(t *testing.T) {
		repo := &testutil.MockChallengeRepository{
			GetQuestionByIDFunc: func(context.Context, uint64) (*models.Question, error) {
				return question, nil
			},
			GetAnswersByQuestionIDFunc: func(context.Context, uint64) ([]*models.Answer, error) {
				return answers, nil
			},
			IncrementQuestionParticipantsFunc: func(context.Context, uint64) error {
				return errors.New("participants")
			},
			GetQuestionTotalAnswersFunc: func(context.Context, uint64) (int32, error) {
				return 2, nil
			},
			GetAnswerVoteCountFunc: func(context.Context, uint64) (int32, error) {
				return 1, nil
			},
		}
		com := &testutil.MockCommercialClient{
			AddBalanceFunc: func(context.Context, uint64, string, float64) error {
				return errors.New("wallet")
			},
		}
		svc := service.NewChallengeService(repo, com)
		out, err := svc.SubmitAnswer(context.Background(), 1, 9, 1)
		require.NoError(t, err)
		require.Equal(t, uint64(9), out.ID)
	})

	t.Run("updated question fetch failure uses original", func(t *testing.T) {
		var gets int
		repo := &testutil.MockChallengeRepository{
			GetQuestionByIDFunc: func(context.Context, uint64) (*models.Question, error) {
				gets++
				if gets == 1 {
					return &models.Question{ID: 9, Title: "original", Prize: 10}, nil
				}
				return nil, errors.New("reload")
			},
			GetAnswersByQuestionIDFunc: func(context.Context, uint64) ([]*models.Answer, error) {
				return answers, nil
			},
			GetQuestionTotalAnswersFunc: func(context.Context, uint64) (int32, error) {
				return 0, errors.New("totals")
			},
			GetAnswerVoteCountFunc: func(context.Context, uint64) (int32, error) {
				return 0, errors.New("votes")
			},
		}
		svc := service.NewChallengeService(repo, nil)
		out, err := svc.SubmitAnswer(context.Background(), 1, 9, 1)
		require.NoError(t, err)
		require.Equal(t, "original", out.Title)
		require.Equal(t, int32(0), out.Answers[0].VotePercentage)
	})
}

func TestChallengeService_GetAdvertisement_EmptyProjectURL(t *testing.T) {
	svc := service.NewChallengeService(
		&testutil.MockChallengeRepository{},
		nil,
		service.ChallengeConfig{Locale: "en", ProjectURL: ""},
	)
	ads, err := svc.GetAdvertisement(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, ads)
	require.Equal(t, "/uploads/challenge/advertisement/bn-1000/bn-1000.mp4", ads[0].VideoURL)
	require.Equal(t, "/uploads/challenge/advertisement/bn-1000/bn-1000.jpg", ads[0].ImageURL)
}
