package handler_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metarang/shared/pb/social"
	"metarang/social-service/internal/handler"
	"metarang/social-service/internal/models"
	"metarang/social-service/internal/service"
	"metarang/social-service/internal/testutil"
)

type stubChallengeSvc struct {
	getTimings       func(context.Context, uint64) (*models.TimingsData, error)
	getQuestion      func(context.Context, uint64) (*models.QuestionResource, error)
	submitAnswer     func(context.Context, uint64, uint64, uint64) (*models.QuestionResource, error)
	getAdvertisement func(context.Context) ([]models.Advertisement, error)
}

func (s *stubChallengeSvc) GetTimings(ctx context.Context, userID uint64) (*models.TimingsData, error) {
	if s.getTimings != nil {
		return s.getTimings(ctx, userID)
	}
	return &models.TimingsData{}, nil
}

func (s *stubChallengeSvc) GetQuestion(ctx context.Context, userID uint64) (*models.QuestionResource, error) {
	if s.getQuestion != nil {
		return s.getQuestion(ctx, userID)
	}
	return nil, nil
}

func (s *stubChallengeSvc) SubmitAnswer(ctx context.Context, userID, questionID, answerID uint64) (*models.QuestionResource, error) {
	if s.submitAnswer != nil {
		return s.submitAnswer(ctx, userID, questionID, answerID)
	}
	return nil, nil
}

func (s *stubChallengeSvc) GetAdvertisement(ctx context.Context) ([]models.Advertisement, error) {
	if s.getAdvertisement != nil {
		return s.getAdvertisement(ctx)
	}
	return nil, nil
}

func TestChallengeHandler_GetTimings_OK(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
			getTimings: func(ctx context.Context, uid uint64) (*models.TimingsData, error) {
				return &models.TimingsData{
					DisplayAdInterval: 1, DisplayQuestionInterval: 2, DisplayAnswerInterval: 3,
					Participants: 4, CorrectAnswers: 5, WrongAnswers: 6,
				}, nil
			},
		})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	resp, err := cli.GetTimings(context.Background(), &pb.GetTimingsRequest{UserId: 42})
	if err != nil || resp.Data.Participants != 4 {
		t.Fatalf("err=%v resp=%+v", err, resp)
	}
}

func TestChallengeHandler_GetTimings_MissingUserID(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.GetTimings(context.Background(), &pb.GetTimingsRequest{})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
}

func TestChallengeHandler_GetQuestion_NotFound(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
			getQuestion: func(ctx context.Context, uid uint64) (*models.QuestionResource, error) {
				return nil, service.ErrNoUnansweredQuestions
			},
		})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.GetQuestion(context.Background(), &pb.GetQuestionRequest{UserId: 1})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Fatalf("got %v", err)
	}
}

func TestChallengeHandler_GetQuestion_OK(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
			getQuestion: func(ctx context.Context, uid uint64) (*models.QuestionResource, error) {
				return &models.QuestionResource{
					ID: 1, Title: "Q", Image: "i", Prize: 5, PrizeType: "coin",
					Participants: 2, Views: 3, CreatorCode: "c",
					Answers: []models.AnswerResource{{ID: 10, Title: "A", IsCorrect: true, VotePercentage: 50}},
				}, nil
			},
		})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	resp, err := cli.GetQuestion(context.Background(), &pb.GetQuestionRequest{UserId: 1})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data.Title != "Q" || len(resp.Data.Answers) != 1 || !resp.Data.Answers[0].IsCorrect {
		t.Fatalf("unexpected: %+v", resp.Data)
	}
}

func TestChallengeHandler_GetQuestion_MissingUserID(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.GetQuestion(context.Background(), &pb.GetQuestionRequest{})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
}

func TestChallengeHandler_SubmitAnswer_OK(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
			submitAnswer: func(ctx context.Context, userID, questionID, answerID uint64) (*models.QuestionResource, error) {
				return &models.QuestionResource{
					ID: questionID, Title: "Q",
					Answers: []models.AnswerResource{{ID: answerID, Title: "A", VotePercentage: 100}},
				}, nil
			},
		})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	resp, err := cli.SubmitAnswer(context.Background(), &pb.SubmitAnswerRequest{
		UserId: 1, QuestionId: 2, AnswerId: 3,
	})
	if err != nil || resp.Data.Id != 2 || len(resp.Data.Answers) != 1 {
		t.Fatalf("err=%v resp=%+v", err, resp)
	}
}

func TestChallengeHandler_SubmitAnswer_ErrorMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code codes.Code
	}{
		{"question not found", service.ErrQuestionNotFound, codes.NotFound},
		{"answer not found", service.ErrAnswerNotFound, codes.NotFound},
		{"mismatch", service.ErrAnswerMismatch, codes.InvalidArgument},
		{"no questions", service.ErrNoUnansweredQuestions, codes.NotFound},
		{"internal", errors.New("db"), codes.Internal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
				handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
					submitAnswer: func(ctx context.Context, userID, questionID, answerID uint64) (*models.QuestionResource, error) {
						return nil, tc.err
					},
				})
			})
			defer cleanup()
			cli := pb.NewChallengeServiceClient(conn)
			_, err := cli.SubmitAnswer(context.Background(), &pb.SubmitAnswerRequest{
				UserId: 1, QuestionId: 2, AnswerId: 3,
			})
			st, ok := status.FromError(err)
			if !ok || st.Code() != tc.code {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestChallengeHandler_SubmitAnswer_Validation(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.SubmitAnswer(context.Background(), &pb.SubmitAnswerRequest{UserId: 1})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
}

func TestChallengeHandler_SubmitAnswer_AlreadyAnswered(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
			submitAnswer: func(ctx context.Context, userID, questionID, answerID uint64) (*models.QuestionResource, error) {
				return nil, service.ErrAlreadyAnswered
			},
		})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.SubmitAnswer(context.Background(), &pb.SubmitAnswerRequest{
		UserId: 1, QuestionId: 2, AnswerId: 3,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("got %v", err)
	}
}

func TestChallengeHandler_GetAdvertisement_OK(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
			getAdvertisement: func(ctx context.Context) ([]models.Advertisement, error) {
				return []models.Advertisement{{
					Code:            "bn-1000",
					Title:           "Matrix exit box",
					URL:             "https://metarang.com/fa/citizen/bn-1000",
					InvestmentAsset: "red",
				}}, nil
			},
		})
	})
	defer cleanup()

	cli := pb.NewChallengeServiceClient(conn)
	resp, err := cli.GetAdvertisement(context.Background(), &pb.GetAdvertisementRequest{})
	if err != nil {
		t.Fatalf("GetAdvertisement failed: %v", err)
	}
	if len(resp.Advertisements) != 1 {
		t.Fatalf("expected 1 advertisement, got %d", len(resp.Advertisements))
	}
	if resp.Advertisements[0].Url != "https://metarang.com/fa/citizen/bn-1000" {
		t.Fatalf("unexpected url: %s", resp.Advertisements[0].Url)
	}
	if resp.Advertisements[0].InvestmentAsset != "red" {
		t.Fatalf("unexpected investment_asset: %s", resp.Advertisements[0].InvestmentAsset)
	}
}
