package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "metargb/shared/pb/social"
	"metargb/shared/pkg/auth"
	"metargb/social-service/internal/models"
	"metargb/social-service/internal/service"
)

type challengeHandler struct {
	pb.UnimplementedChallengeServiceServer
	challengeService service.ChallengeService
}

func RegisterChallengeHandler(grpcServer *grpc.Server, challengeService service.ChallengeService) {
	pb.RegisterChallengeServiceServer(grpcServer, &challengeHandler{
		challengeService: challengeService,
	})
}

func (h *challengeHandler) GetTimings(ctx context.Context, req *pb.GetTimingsRequest) (*pb.GetTimingsResponse, error) {
	// Get user ID from context (set by auth interceptor or gateway)
	userID := getUserIDFromContext(ctx)
	// If not in context, try to get from metadata (set by gateway)
	if userID == 0 {
		// Gateway should set user_id in metadata
		// For now, return error if not found
		return nil, status.Errorf(codes.Unauthenticated, "authentication required")
	}

	timings, err := h.challengeService.GetTimings(ctx, userID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get timings: %v", err)
	}

	return &pb.GetTimingsResponse{
		Data: &pb.TimingsData{
			DisplayAdInterval:       timings.DisplayAdInterval,
			DisplayQuestionInterval: timings.DisplayQuestionInterval,
			DisplayAnswerInterval:   timings.DisplayAnswerInterval,
			Participants:            timings.Participants,
			CorrectAnswers:          timings.CorrectAnswers,
			WrongAnswers:            timings.WrongAnswers,
		},
	}, nil
}

func (h *challengeHandler) GetQuestion(ctx context.Context, req *pb.GetQuestionRequest) (*pb.GetQuestionResponse, error) {
	// Use user ID from request (set by gateway from authenticated user)
	userID := req.UserId
	if userID == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	question, err := h.challengeService.GetQuestion(ctx, userID)
	if err != nil {
		if errors.Is(err, service.ErrNoUnansweredQuestions) {
			return nil, status.Errorf(codes.NotFound, "no unanswered questions available")
		}
		return nil, status.Errorf(codes.Internal, "failed to get question: %v", err)
	}

	return &pb.GetQuestionResponse{
		Data: convertQuestionResourceToProto(question),
	}, nil
}

func (h *challengeHandler) SubmitAnswer(ctx context.Context, req *pb.SubmitAnswerRequest) (*pb.SubmitAnswerResponse, error) {
	// Validate required fields
	if req.QuestionId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "question_id is required")
	}
	if req.AnswerId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "answer_id is required")
	}

	// Use user ID from request (set by gateway from authenticated user)
	userID := req.UserId
	if userID == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "user_id is required")
	}

	question, err := h.challengeService.SubmitAnswer(ctx, userID, req.QuestionId, req.AnswerId)
	if err != nil {
		return nil, mapChallengeError(err)
	}

	return &pb.SubmitAnswerResponse{
		Data: convertQuestionResourceToProto(question),
	}, nil
}

func convertQuestionResourceToProto(resource *models.QuestionResource) *pb.QuestionResource {
	answerResources := make([]*pb.AnswerResource, 0, len(resource.Answers))
	for _, answer := range resource.Answers {
		answerResources = append(answerResources, &pb.AnswerResource{
			Id:             answer.ID,
			Title:          answer.Title,
			Image:          answer.Image,
			IsCorrect:      answer.IsCorrect,
			VotePercentage: answer.VotePercentage,
		})
	}

	return &pb.QuestionResource{
		Id:           resource.ID,
		Title:        resource.Title,
		Image:        resource.Image,
		Prize:        resource.Prize,
		Participants: int32(resource.Participants),
		Views:        int32(resource.Views),
		CreatorCode:  resource.CreatorCode,
		Answers:      answerResources,
	}
}

func mapChallengeError(err error) error {
	switch {
	case errors.Is(err, service.ErrQuestionNotFound):
		return status.Errorf(codes.NotFound, "question not found")
	case errors.Is(err, service.ErrAnswerNotFound):
		return status.Errorf(codes.NotFound, "answer not found")
	case errors.Is(err, service.ErrAnswerMismatch):
		return status.Errorf(codes.InvalidArgument, "answer does not belong to the given question")
	case errors.Is(err, service.ErrAlreadyAnswered):
		return status.Errorf(codes.PermissionDenied, "user has already answered this question correctly")
	case errors.Is(err, service.ErrNoUnansweredQuestions):
		return status.Errorf(codes.NotFound, "no unanswered questions available")
	default:
		return status.Errorf(codes.Internal, "operation failed: %v", err)
	}
}

// getUserIDFromContext extracts user ID from context (set by auth interceptor)
func getUserIDFromContext(ctx context.Context) uint64 {
	userCtx := ctx.Value(auth.UserContextKey{})
	if userCtx == nil {
		return 0
	}
	uc, ok := userCtx.(*auth.UserContext)
	if !ok {
		return 0
	}
	return uc.UserID
}
