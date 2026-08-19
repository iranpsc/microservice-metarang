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
	"metarang/social-service/internal/testutil"
)

func TestChallengeHandler_GetTimings_InternalError(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
			getTimings: func(context.Context, uint64) (*models.TimingsData, error) {
				return nil, errors.New("db")
			},
		})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.GetTimings(context.Background(), &pb.GetTimingsRequest{UserId: 1})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestChallengeHandler_GetQuestion_InternalError(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
			getQuestion: func(context.Context, uint64) (*models.QuestionResource, error) {
				return nil, errors.New("db")
			},
		})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.GetQuestion(context.Background(), &pb.GetQuestionRequest{UserId: 1})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestChallengeHandler_GetAdvertisement_InternalError(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{
			getAdvertisement: func(context.Context) ([]models.Advertisement, error) {
				return nil, errors.New("ads failed")
			},
		})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.GetAdvertisement(context.Background(), &pb.GetAdvertisementRequest{})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("got %v", err)
	}
}

func TestChallengeHandler_SubmitAnswer_MissingUserID(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.SubmitAnswer(context.Background(), &pb.SubmitAnswerRequest{
		QuestionId: 2, AnswerId: 3,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
}

func TestChallengeHandler_SubmitAnswer_MissingAnswerID(t *testing.T) {
	conn, cleanup := testutil.DialBufConn(func(gs *grpc.Server) {
		handler.RegisterChallengeHandler(gs, &stubChallengeSvc{})
	})
	defer cleanup()
	cli := pb.NewChallengeServiceClient(conn)
	_, err := cli.SubmitAnswer(context.Background(), &pb.SubmitAnswerRequest{
		UserId: 1, QuestionId: 2,
	})
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("got %v", err)
	}
}
