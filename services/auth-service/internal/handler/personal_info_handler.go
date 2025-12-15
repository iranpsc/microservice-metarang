package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"metargb/auth-service/internal/models"
	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
)

type personalInfoHandler struct {
	pb.UnimplementedPersonalInfoServiceServer
	personalInfoService service.PersonalInfoService
}

func RegisterPersonalInfoHandler(grpcServer *grpc.Server, personalInfoService service.PersonalInfoService) {
	pb.RegisterPersonalInfoServiceServer(grpcServer, &personalInfoHandler{
		personalInfoService: personalInfoService,
	})
}

func (h *personalInfoHandler) GetPersonalInfo(ctx context.Context, req *pb.GetPersonalInfoRequest) (*pb.GetPersonalInfoResponse, error) {
	personalInfo, err := h.personalInfoService.GetPersonalInfo(ctx, req.UserId)
	if err != nil {
		return nil, mapPersonalInfoServiceError(err)
	}

	// If not found, return empty array (matches Laravel behavior)
	if personalInfo == nil {
		return &pb.GetPersonalInfoResponse{
			Data: &pb.PersonalInfoData{},
		}, nil
	}

	return &pb.GetPersonalInfoResponse{
		Data: convertPersonalInfoToProto(personalInfo),
	}, nil
}

func (h *personalInfoHandler) UpdatePersonalInfo(ctx context.Context, req *pb.UpdatePersonalInfoRequest) (*emptypb.Empty, error) {
	// Convert passions map from proto to Go map
	passions := make(map[string]bool)
	if req.Passions != nil {
		for key, value := range req.Passions {
			passions[key] = value
		}
	}

	err := h.personalInfoService.UpdatePersonalInfo(
		ctx,
		req.UserId,
		req.Occupation,
		req.Education,
		req.Memory,
		req.LovedCity,
		req.LovedCountry,
		req.LovedLanguage,
		req.ProblemSolving,
		req.Prediction,
		req.About,
		passions,
	)
	if err != nil {
		return nil, mapPersonalInfoServiceError(err)
	}

	return &emptypb.Empty{}, nil
}

func convertPersonalInfoToProto(personalInfo *models.PersonalInfo) *pb.PersonalInfoData {
	if personalInfo == nil {
		return &pb.PersonalInfoData{}
	}

	data := &pb.PersonalInfoData{}

	// Convert nullable string fields
	if personalInfo.Occupation.Valid {
		data.Occupation = personalInfo.Occupation.String
	}
	if personalInfo.Education.Valid {
		data.Education = personalInfo.Education.String
	}
	if personalInfo.Memory.Valid {
		data.Memory = personalInfo.Memory.String
	}
	if personalInfo.LovedCity.Valid {
		data.LovedCity = personalInfo.LovedCity.String
	}
	if personalInfo.LovedCountry.Valid {
		data.LovedCountry = personalInfo.LovedCountry.String
	}
	if personalInfo.LovedLanguage.Valid {
		data.LovedLanguage = personalInfo.LovedLanguage.String
	}
	if personalInfo.ProblemSolving.Valid {
		data.ProblemSolving = personalInfo.ProblemSolving.String
	}
	if personalInfo.Prediction.Valid {
		data.Prediction = personalInfo.Prediction.String
	}
	if personalInfo.About.Valid {
		data.About = personalInfo.About.String
	}

	// Convert passions map
	if personalInfo.Passions != nil {
		data.Passions = personalInfo.Passions
	} else {
		data.Passions = make(map[string]bool)
	}

	return data
}

func mapPersonalInfoServiceError(err error) error {
	switch {
	case errors.Is(err, service.ErrInvalidOccupation),
		errors.Is(err, service.ErrInvalidEducation),
		errors.Is(err, service.ErrInvalidMemory),
		errors.Is(err, service.ErrInvalidLovedCity),
		errors.Is(err, service.ErrInvalidLovedCountry),
		errors.Is(err, service.ErrInvalidLovedLanguage),
		errors.Is(err, service.ErrInvalidProblemSolving),
		errors.Is(err, service.ErrInvalidPrediction),
		errors.Is(err, service.ErrInvalidAbout),
		errors.Is(err, service.ErrInvalidPassionKey):
		return status.Errorf(codes.InvalidArgument, "%s", err.Error())
	default:
		return status.Errorf(codes.Internal, "operation failed: %v", err)
	}
}
