package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"metarang/auth-service/internal/lang"
	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
)

type profileLimitationHandler struct {
	pb.UnimplementedProfileLimitationServiceServer
	limitationService service.ProfileLimitationService
}

func RegisterProfileLimitationHandler(grpcServer *grpc.Server, limitationService service.ProfileLimitationService) pb.ProfileLimitationServiceServer {
	h := NewProfileLimitationHandler(limitationService)
	pb.RegisterProfileLimitationServiceServer(grpcServer, h)
	return h
}

func (h *profileLimitationHandler) CreateProfileLimitation(ctx context.Context, req *pb.CreateProfileLimitationRequest) (*pb.ProfileLimitationResponse, error) {
	locale := getProjectLocale()
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", lang.Tf(locale, "request is required"))
	}
	if req.Options == nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", lang.Tf(locale, "options is required"))
	}

	limiterUserID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	options, err := convertProtoOptions(req.Options)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", lang.Tf(locale, "%v", err))
	}

	limitation, err := h.limitationService.Create(
		ctx,
		limiterUserID,
		req.LimitedUserId,
		options,
		noteUpdateFromProto(req.Note),
	)
	if err != nil {
		return nil, MapProfileLimitationError(err, locale)
	}

	return &pb.ProfileLimitationResponse{
		Data: convertProfileLimitationToProto(limitation, limiterUserID),
	}, nil
}

func (h *profileLimitationHandler) UpdateProfileLimitation(ctx context.Context, req *pb.UpdateProfileLimitationRequest) (*pb.ProfileLimitationResponse, error) {
	locale := getProjectLocale()
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", lang.Tf(locale, "request is required"))
	}
	if req.Options == nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", lang.Tf(locale, "options is required"))
	}

	limiterUserID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	options, err := convertProtoOptions(req.Options)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", lang.Tf(locale, "%v", err))
	}

	limitation, err := h.limitationService.Update(
		ctx,
		req.LimitationId,
		limiterUserID,
		options,
		noteUpdateFromProto(req.Note),
	)
	if err != nil {
		return nil, MapProfileLimitationError(err, locale)
	}

	return &pb.ProfileLimitationResponse{
		Data: convertProfileLimitationToProto(limitation, limiterUserID),
	}, nil
}

func (h *profileLimitationHandler) DeleteProfileLimitation(ctx context.Context, req *pb.DeleteProfileLimitationRequest) (*emptypb.Empty, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", lang.Tf(getProjectLocale(), "request is required"))
	}
	limiterUserID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}
	if err := h.limitationService.Delete(ctx, req.LimitationId, limiterUserID); err != nil {
		return nil, MapProfileLimitationError(err, getProjectLocale())
	}

	return &emptypb.Empty{}, nil
}

// MapProfileLimitationError maps service errors to gRPC status codes
func MapProfileLimitationError(err error, locale string) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, service.ErrProfileLimitationNotFound):
		return status.Errorf(codes.NotFound, "%s", err.Error())
	case errors.Is(err, service.ErrProfileLimitationAlreadyExists):
		return status.Errorf(codes.PermissionDenied, "%s", err.Error())
	case errors.Is(err, service.ErrInvalidOptions):
		return status.Errorf(codes.InvalidArgument, "%s", err.Error())
	case errors.Is(err, service.ErrNoteTooLong):
		return status.Errorf(codes.InvalidArgument, "%s", err.Error())
	case errors.Is(err, service.ErrUserNotFound):
		return status.Errorf(codes.NotFound, "%s", err.Error())
	case errors.Is(err, service.ErrUnauthorized):
		return status.Errorf(codes.PermissionDenied, "%s", err.Error())
	default:
		return status.Errorf(codes.Internal, "%s", lang.Tf(locale, "operation failed: %v", err))
	}
}

func convertProtoOptions(opts *pb.ProfileLimitationOptions) (models.ProfileLimitationOptions, error) {
	if opts == nil {
		return models.ProfileLimitationOptions{}, service.ErrInvalidOptions
	}
	if opts.Follow == nil ||
		opts.SendMessage == nil ||
		opts.Share == nil ||
		opts.SendTicket == nil ||
		opts.ViewProfileImages == nil ||
		opts.ViewFeaturesLocations == nil {
		return models.ProfileLimitationOptions{}, service.ErrInvalidOptions
	}

	return models.ProfileLimitationOptions{
		Follow:                *opts.Follow,
		SendMessage:           *opts.SendMessage,
		Share:                 *opts.Share,
		SendTicket:            *opts.SendTicket,
		ViewProfileImages:     *opts.ViewProfileImages,
		ViewFeaturesLocations: *opts.ViewFeaturesLocations,
	}, nil
}

func noteUpdateFromProto(note *string) service.NoteUpdate {
	if note == nil {
		return service.NoteUpdate{Present: false}
	}
	// Empty string is the wire representation of an explicit clear/null.
	if *note == "" {
		return service.NoteUpdate{Present: true, Value: nil}
	}
	value := *note
	return service.NoteUpdate{Present: true, Value: &value}
}

// convertProfileLimitationToProto converts a ProfileLimitation model to proto.
// callerUserID determines whether note is visible (only to the limiter).
func convertProfileLimitationToProto(limitation *models.ProfileLimitation, callerUserID uint64) *pb.ProfileLimitation {
	follow := limitation.Options.Follow
	sendMessage := limitation.Options.SendMessage
	share := limitation.Options.Share
	sendTicket := limitation.Options.SendTicket
	viewProfileImages := limitation.Options.ViewProfileImages
	viewFeaturesLocations := limitation.Options.ViewFeaturesLocations

	proto := &pb.ProfileLimitation{
		Id:            limitation.ID,
		LimiterUserId: limitation.LimiterUserID,
		LimitedUserId: limitation.LimitedUserID,
		Options: &pb.ProfileLimitationOptions{
			Follow:                &follow,
			SendMessage:           &sendMessage,
			Share:                 &share,
			SendTicket:            &sendTicket,
			ViewProfileImages:     &viewProfileImages,
			ViewFeaturesLocations: &viewFeaturesLocations,
		},
		CreatedAt: timestamppb.New(limitation.CreatedAt),
		UpdatedAt: timestamppb.New(limitation.UpdatedAt),
	}

	// Include note (even when empty/null) only when the caller is the limiter.
	if callerUserID == limitation.LimiterUserID {
		note := ""
		if limitation.Note.Valid {
			note = limitation.Note.String
		}
		proto.Note = &note
	}

	return proto
}
