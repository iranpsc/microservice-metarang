package handler

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"metarang/auth-service/internal/lang"
	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
	"metarang/shared/pkg/jalali"
)

func (h *kycHandler) GetKYC(ctx context.Context, req *pb.GetKYCRequest) (*pb.KYCResponse, error) {
	if err := authorizeSelfOrService(ctx, req.UserId); err != nil {
		return nil, err
	}

	kyc, err := h.kycService.GetKYC(ctx, req.UserId)
	if err != nil {
		return nil, mapKYCServiceError(err, getProjectLocale())
	}

	if kyc == nil {
		return &pb.KYCResponse{}, nil
	}

	return h.convertKYCToProto(kyc), nil
}

func (h *kycHandler) UpdateKYC(ctx context.Context, req *pb.UpdateKYCRequest) (*pb.KYCResponse, error) {
	locale := getProjectLocale()

	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	videoPath, videoName := "", ""
	if req.Video != nil {
		videoPath = req.Video.Path
		videoName = req.Video.Name
	}

	kyc, err := h.kycService.SubmitKYC(ctx, userID, service.KYCSubmission{
		Fname:                req.Fname,
		Lname:                req.Lname,
		MelliCode:            req.MelliCode,
		Birthdate:            req.Birthdate,
		Province:             req.Province,
		Gender:               req.Gender,
		VerifyTextID:         req.VerifyTextId,
		MelliCardData:        req.MelliCardData,
		MelliCardFilename:    req.MelliCardFilename,
		MelliCardContentType: req.MelliCardContentType,
		VideoPath:            videoPath,
		VideoName:            videoName,
	})
	if err != nil {
		return nil, mapKYCServiceError(err, locale)
	}

	return h.convertKYCToProto(kyc), nil
}

func (h *kycHandler) convertKYCToProto(kyc *models.KYC) *pb.KYCResponse {
	birthdate := ""
	if kyc.Birthdate.Valid {
		birthdate = jalali.CarbonToJalali(kyc.Birthdate.Time)
	}

	videoURL := ""
	if kyc.Video.Valid {
		videoURL = kyc.Video.String
	}

	errorStr := ""
	if kyc.Errors.Valid {
		errorStr = kyc.Errors.String
	}

	gender := ""
	if kyc.Gender.Valid {
		gender = kyc.Gender.String
	}

	return &pb.KYCResponse{
		Id:        kyc.ID,
		MelliCard: service.ResolvePublicURL(h.apiGatewayURL, kyc.MelliCard),
		Fname:     kyc.Fname,
		Lname:     kyc.Lname,
		MelliCode: kyc.MelliCode,
		Birthdate: birthdate,
		Province:  kyc.Province,
		Status:    kyc.Status,
		Video:     service.ResolvePublicURL(h.apiGatewayURL, videoURL),
		Errors:    errorStr,
		Gender:    gender,
	}
}

func mapKYCServiceError(err error, locale string) error {
	switch {
	case errors.Is(err, service.ErrKYCNotFound):
		return status.Errorf(codes.NotFound, "%s", err.Error())
	case errors.Is(err, service.ErrKYCNotOwned):
		return status.Errorf(codes.PermissionDenied, "%s", err.Error())
	case errors.Is(err, service.ErrKYCNotRejected):
		return status.Errorf(codes.PermissionDenied, "%s", err.Error())
	case errors.Is(err, service.ErrStorageUnavailable):
		return status.Errorf(codes.Internal, "%s", lang.T(locale, "storage service not available"))
	case errors.Is(err, service.ErrMelliCardFilenameRequired):
		return status.Errorf(codes.InvalidArgument, "%s", lang.T(locale, "melli_card_filename is required"))
	case errors.Is(err, service.ErrMelliCardContentTypeRequired):
		return status.Errorf(codes.InvalidArgument, "%s", lang.T(locale, "melli_card_content_type is required"))
	case errors.Is(err, service.ErrMelliCardTooLarge):
		return status.Errorf(codes.InvalidArgument, "%s", lang.T(locale, "melli_card file size exceeds maximum of 5MB"))
	case errors.Is(err, service.ErrInvalidMelliCardType):
		return status.Errorf(codes.InvalidArgument, "%s", lang.T(locale, "melli_card must be a PNG or JPEG image"))
	case errors.Is(err, service.ErrInvalidMelliCardExtension):
		return status.Errorf(codes.InvalidArgument, "%s", lang.T(locale, "melli_card filename must have .png, .jpg, or .jpeg extension"))
	case errors.Is(err, service.ErrInvalidFname),
		errors.Is(err, service.ErrInvalidLname),
		errors.Is(err, service.ErrInvalidMelliCode),
		errors.Is(err, service.ErrInvalidBirthdate),
		errors.Is(err, service.ErrInvalidProvince),
		errors.Is(err, service.ErrProvinceRequired),
		errors.Is(err, service.ErrInvalidGender),
		errors.Is(err, service.ErrGenderRequired),
		errors.Is(err, service.ErrVerifyTextIDRequired),
		errors.Is(err, service.ErrVerifyTextIDNotFound),
		errors.Is(err, service.ErrVideoRequired),
		errors.Is(err, service.ErrMelliCardRequired),
		errors.Is(err, service.ErrMelliCodeNotUnique):
		if fields, ok := mapServiceErrorToValidationFields(err, locale); ok {
			return returnValidationError(fields)
		}
		return status.Errorf(codes.InvalidArgument, "%s", err.Error())
	default:
		return status.Errorf(codes.Internal, "%s", lang.Tf(locale, "operation failed: %v", err))
	}
}

type kycHandler struct {
	pb.UnimplementedKYCServiceServer
	kycService    service.KYCService
	apiGatewayURL string
}

func RegisterKYCHandler(grpcServer *grpc.Server, kycService service.KYCService, apiGatewayURL string) pb.KYCServiceServer {
	h := NewKYCHandler(kycService, apiGatewayURL)
	pb.RegisterKYCServiceServer(grpcServer, h)
	return h
}

func mapServiceError(err error, locale string) error {
	switch {
	case errors.Is(err, service.ErrBankAccountNotFound):
		return status.Errorf(codes.NotFound, "%s", err.Error())
	case errors.Is(err, service.ErrBankAccountNotOwned),
		errors.Is(err, service.ErrBankAccountNotRejected):
		return status.Errorf(codes.PermissionDenied, "%s", err.Error())
	case errors.Is(err, service.ErrUserNotVerified):
		return status.Errorf(codes.PermissionDenied, "%s", err.Error())
	case errors.Is(err, service.ErrInvalidBankName),
		errors.Is(err, service.ErrInvalidShabaNum),
		errors.Is(err, service.ErrInvalidCardNum),
		errors.Is(err, service.ErrShabaNumNotUnique),
		errors.Is(err, service.ErrCardNumNotUnique):
		if fields, ok := mapServiceErrorToValidationFields(err, locale); ok {
			return returnValidationError(fields)
		}
		return status.Errorf(codes.InvalidArgument, "%s", err.Error())
	default:
		return status.Errorf(codes.Internal, "%s", lang.Tf(locale, "operation failed: %v", err))
	}
}

func convertBankAccountToProto(bankAccount *models.BankAccount) *pb.BankAccountResponse {
	errorStr := ""
	if bankAccount.Errors.Valid {
		errorStr = bankAccount.Errors.String
	}

	return &pb.BankAccountResponse{
		Id:       bankAccount.ID,
		BankName: bankAccount.BankName,
		ShabaNum: bankAccount.ShabaNum,
		CardNum:  bankAccount.CardNum,
		Status:   bankAccount.Status,
		Errors:   errorStr,
	}
}

func (h *kycHandler) ListBankAccounts(ctx context.Context, req *pb.ListBankAccountsRequest) (*pb.ListBankAccountsResponse, error) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	accounts, err := h.kycService.ListBankAccounts(ctx, userID)
	if err != nil {
		return nil, mapServiceError(err, getProjectLocale())
	}

	protoAccounts := make([]*pb.BankAccountResponse, 0, len(accounts))
	for _, account := range accounts {
		protoAccounts = append(protoAccounts, convertBankAccountToProto(account))
	}

	return &pb.ListBankAccountsResponse{Data: protoAccounts}, nil
}

func (h *kycHandler) CreateBankAccount(ctx context.Context, req *pb.CreateBankAccountRequest) (*pb.BankAccountResponse, error) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	bankAccount, err := h.kycService.CreateBankAccount(ctx, userID, req.BankName, req.ShabaNum, req.CardNum)
	if err != nil {
		return nil, mapServiceError(err, getProjectLocale())
	}

	return convertBankAccountToProto(bankAccount), nil
}

func (h *kycHandler) GetBankAccount(ctx context.Context, req *pb.GetBankAccountRequest) (*pb.BankAccountResponse, error) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	bankAccount, err := h.kycService.GetBankAccount(ctx, userID, req.BankAccountId)
	if err != nil {
		return nil, mapServiceError(err, getProjectLocale())
	}

	return convertBankAccountToProto(bankAccount), nil
}

func (h *kycHandler) UpdateBankAccount(ctx context.Context, req *pb.UpdateBankAccountRequest) (*pb.BankAccountResponse, error) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	bankAccount, err := h.kycService.UpdateBankAccount(ctx, userID, req.BankAccountId, req.BankName, req.ShabaNum, req.CardNum)
	if err != nil {
		return nil, mapServiceError(err, getProjectLocale())
	}

	return convertBankAccountToProto(bankAccount), nil
}

func (h *kycHandler) DeleteBankAccount(ctx context.Context, req *pb.DeleteBankAccountRequest) (*emptypb.Empty, error) {
	userID, err := authenticatedUserID(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.kycService.DeleteBankAccount(ctx, userID, req.BankAccountId); err != nil {
		return nil, mapServiceError(err, getProjectLocale())
	}

	return &emptypb.Empty{}, nil
}
