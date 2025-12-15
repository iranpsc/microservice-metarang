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
	"metargb/shared/pkg/jalali"
)

func (h *kycHandler) GetKYC(ctx context.Context, req *pb.GetKYCRequest) (*pb.KYCResponse, error) {
	kyc, err := h.kycService.GetKYC(ctx, req.UserId)
	if err != nil {
		return nil, mapKYCServiceError(err)
	}

	// If KYC not found, return empty response (matches Laravel behavior)
	if kyc == nil {
		return &pb.KYCResponse{}, nil
	}

	return convertKYCToProto(kyc), nil
}

func (h *kycHandler) UpdateKYC(ctx context.Context, req *pb.UpdateKYCRequest) (*pb.KYCResponse, error) {
	videoPath := ""
	videoName := ""
	if req.Video != nil {
		videoPath = req.Video.Path
		videoName = req.Video.Name
	}

	kyc, err := h.kycService.UpdateKYC(
		ctx,
		req.UserId,
		req.Fname,
		req.Lname,
		req.MelliCode,
		req.Birthdate,
		req.Province,
		req.MelliCard,
		videoPath,
		videoName,
		req.VerifyTextId,
		req.Gender,
	)
	if err != nil {
		return nil, mapKYCServiceError(err)
	}

	return convertKYCToProto(kyc), nil
}

// convertKYCToProto converts a KYC model to proto response
func convertKYCToProto(kyc *models.KYC) *pb.KYCResponse {
	birthdate := ""
	if kyc.Birthdate.Valid {
		birthdate = jalali.CarbonToJalali(kyc.Birthdate.Time)
	}

	video := ""
	if kyc.Video.Valid {
		video = kyc.Video.String
	}

	errors := ""
	if kyc.Errors.Valid {
		errors = kyc.Errors.String
	}

	gender := ""
	if kyc.Gender.Valid {
		gender = kyc.Gender.String
	}

	return &pb.KYCResponse{
		Id:        kyc.ID,
		MelliCard: kyc.MelliCard,
		Fname:     kyc.Fname,
		Lname:     kyc.Lname,
		MelliCode: kyc.MelliCode,
		Birthdate: birthdate,
		Province:  kyc.Province,
		Status:    kyc.Status,
		Video:     video,
		Errors:    errors,
		Gender:    gender,
	}
}

// mapKYCServiceError maps KYC service errors to gRPC status codes
func mapKYCServiceError(err error) error {
	switch {
	case errors.Is(err, service.ErrKYCNotFound):
		return status.Errorf(codes.NotFound, "%s", err.Error())
	case errors.Is(err, service.ErrKYCNotOwned):
		return status.Errorf(codes.PermissionDenied, "%s", err.Error())
	case errors.Is(err, service.ErrKYCNotRejected):
		return status.Errorf(codes.FailedPrecondition, "%s", err.Error())
	case errors.Is(err, service.ErrInvalidFname),
		errors.Is(err, service.ErrInvalidLname),
		errors.Is(err, service.ErrInvalidMelliCode),
		errors.Is(err, service.ErrInvalidBirthdate),
		errors.Is(err, service.ErrInvalidProvince),
		errors.Is(err, service.ErrInvalidGender),
		errors.Is(err, service.ErrMelliCodeNotUnique):
		return status.Errorf(codes.InvalidArgument, "%s", err.Error())
	default:
		return status.Errorf(codes.Internal, "operation failed: %v", err)
	}
}

type kycHandler struct {
	pb.UnimplementedKYCServiceServer
	kycService service.KYCService
}

func RegisterKYCHandler(grpcServer *grpc.Server, kycService service.KYCService) {
	pb.RegisterKYCServiceServer(grpcServer, &kycHandler{
		kycService: kycService,
	})
}

// mapServiceError maps bank account service errors to gRPC status codes
func mapServiceError(err error) error {
	switch {
	case errors.Is(err, service.ErrBankAccountNotFound):
		return status.Errorf(codes.NotFound, "%s", err.Error())
	case errors.Is(err, service.ErrBankAccountNotOwned):
		return status.Errorf(codes.PermissionDenied, "%s", err.Error())
	case errors.Is(err, service.ErrBankAccountNotRejected):
		return status.Errorf(codes.FailedPrecondition, "%s", err.Error())
	case errors.Is(err, service.ErrUserNotVerified):
		return status.Errorf(codes.PermissionDenied, "%s", err.Error())
	case errors.Is(err, service.ErrInvalidBankName),
		errors.Is(err, service.ErrInvalidShabaNum),
		errors.Is(err, service.ErrInvalidCardNum),
		errors.Is(err, service.ErrShabaNumNotUnique),
		errors.Is(err, service.ErrCardNumNotUnique):
		return status.Errorf(codes.InvalidArgument, "%s", err.Error())
	default:
		return status.Errorf(codes.Internal, "operation failed: %v", err)
	}
}

// convertBankAccountToProto converts a BankAccount model to proto response
func convertBankAccountToProto(bankAccount *models.BankAccount) *pb.BankAccountResponse {
	errors := ""
	if bankAccount.Errors.Valid {
		errors = bankAccount.Errors.String
	}

	return &pb.BankAccountResponse{
		Id:       bankAccount.ID,
		BankName: bankAccount.BankName,
		ShabaNum: bankAccount.ShabaNum,
		CardNum:  bankAccount.CardNum,
		Status:   bankAccount.Status,
		Errors:   errors,
	}
}

func (h *kycHandler) ListBankAccounts(ctx context.Context, req *pb.ListBankAccountsRequest) (*pb.ListBankAccountsResponse, error) {
	accounts, err := h.kycService.ListBankAccounts(ctx, req.UserId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	var protoAccounts []*pb.BankAccountResponse
	for _, account := range accounts {
		protoAccounts = append(protoAccounts, convertBankAccountToProto(account))
	}

	return &pb.ListBankAccountsResponse{
		Data: protoAccounts,
	}, nil
}

func (h *kycHandler) CreateBankAccount(ctx context.Context, req *pb.CreateBankAccountRequest) (*pb.BankAccountResponse, error) {
	bankAccount, err := h.kycService.CreateBankAccount(ctx, req.UserId, req.BankName, req.ShabaNum, req.CardNum)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return convertBankAccountToProto(bankAccount), nil
}

func (h *kycHandler) GetBankAccount(ctx context.Context, req *pb.GetBankAccountRequest) (*pb.BankAccountResponse, error) {
	bankAccount, err := h.kycService.GetBankAccount(ctx, req.UserId, req.BankAccountId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return convertBankAccountToProto(bankAccount), nil
}

func (h *kycHandler) UpdateBankAccount(ctx context.Context, req *pb.UpdateBankAccountRequest) (*pb.BankAccountResponse, error) {
	bankAccount, err := h.kycService.UpdateBankAccount(ctx, req.UserId, req.BankAccountId, req.BankName, req.ShabaNum, req.CardNum)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return convertBankAccountToProto(bankAccount), nil
}

func (h *kycHandler) DeleteBankAccount(ctx context.Context, req *pb.DeleteBankAccountRequest) (*emptypb.Empty, error) {
	err := h.kycService.DeleteBankAccount(ctx, req.UserId, req.BankAccountId)
	if err != nil {
		return nil, mapServiceError(err)
	}

	return &emptypb.Empty{}, nil
}
