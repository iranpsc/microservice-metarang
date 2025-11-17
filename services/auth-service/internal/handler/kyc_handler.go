package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metargb/auth-service/internal/service"
	pb "metargb/shared/pb/auth"
)

type kycHandler struct {
	pb.UnimplementedKYCServiceServer
	kycService service.KYCService
}

func RegisterKYCHandler(grpcServer *grpc.Server, kycService service.KYCService) {
	pb.RegisterKYCServiceServer(grpcServer, &kycHandler{
		kycService: kycService,
	})
}

func (h *kycHandler) SubmitKYC(ctx context.Context, req *pb.SubmitKYCRequest) (*pb.KYCResponse, error) {
	kyc, err := h.kycService.SubmitKYC(ctx, req.UserId, req.Fname, req.Lname, req.NationalCode, req.Birthdate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to submit kyc: %v", err)
	}

	birthdate := ""
	if kyc.Birthdate.Valid {
		birthdate = kyc.Birthdate.Time.Format("2006-01-02")
	}

	return &pb.KYCResponse{
		Id:        kyc.ID,
		UserId:    kyc.UserID,
		Fname:     kyc.Fname,
		Lname:     kyc.Lname,
		Status:    kyc.Status,
		Birthdate: birthdate,
	}, nil
}

func (h *kycHandler) GetKYCStatus(ctx context.Context, req *pb.GetKYCStatusRequest) (*pb.KYCResponse, error) {
	kyc, err := h.kycService.GetKYCStatus(ctx, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "kyc not found: %v", err)
	}

	birthdate := ""
	if kyc.Birthdate.Valid {
		birthdate = kyc.Birthdate.Time.Format("2006-01-02")
	}

	return &pb.KYCResponse{
		Id:        kyc.ID,
		UserId:    kyc.UserID,
		Fname:     kyc.Fname,
		Lname:     kyc.Lname,
		Status:    kyc.Status,
		Birthdate: birthdate,
	}, nil
}

func (h *kycHandler) VerifyBankAccount(ctx context.Context, req *pb.VerifyBankAccountRequest) (*pb.BankAccountResponse, error) {
	bankAccount, err := h.kycService.VerifyBankAccount(ctx, req.UserId, req.BankName, req.ShabaNum, req.CardNum)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to verify bank account: %v", err)
	}

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
	}, nil
}
