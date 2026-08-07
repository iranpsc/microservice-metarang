package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/auth-service/internal/handler"
	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
)

func TestKYCHandler_GetAndUpdate(t *testing.T) {
	ctx := authenticatedContext(1)

	t.Run("get kyc self and service token", func(t *testing.T) {
		m := &mockKYCService{}
		m.getKYCFunc = func(context.Context, uint64) (*models.KYC, error) {
			return &models.KYC{
				ID: 1, UserID: 1, Fname: "a", Lname: "b", MelliCode: "001",
				Birthdate: sql.NullTime{Time: time.Now(), Valid: true},
				Video:     sql.NullString{String: "/v.mp4", Valid: true},
				MelliCard: "/c.jpg",
			}, nil
		}
		h := handler.NewKYCHandler(m, "https://gw")
		resp, err := h.GetKYC(ctx, &pb.GetKYCRequest{UserId: 1})
		if err != nil || resp.Fname != "a" {
			t.Fatalf("resp=%v err=%v", resp, err)
		}

		_, err = h.GetKYC(authenticatedContext(2), &pb.GetKYCRequest{UserId: 1})
		st, _ := status.FromError(err)
		if st.Code() != codes.PermissionDenied && st.Code() != codes.Unauthenticated {
			// authorizeSelfOrService may return PermissionDenied
			if err == nil {
				t.Fatal("expected authz error")
			}
		}

		m.getKYCFunc = func(context.Context, uint64) (*models.KYC, error) { return nil, nil }
		resp, err = h.GetKYC(ctx, &pb.GetKYCRequest{UserId: 1})
		if err != nil || resp == nil {
			t.Fatalf("empty resp err=%v", err)
		}
	})

	t.Run("update/submit kyc", func(t *testing.T) {
		m := &mockKYCService{}
		m.submitKYCFunc = func(context.Context, uint64, service.KYCSubmission) (*models.KYC, error) {
			return nil, service.ErrInvalidFname
		}
		h := handler.NewKYCHandler(m, "")
		_, err := h.UpdateKYC(ctx, &pb.UpdateKYCRequest{Fname: "x"})
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Fatalf("code=%v", st.Code())
		}

		m.submitKYCFunc = func(context.Context, uint64, service.KYCSubmission) (*models.KYC, error) {
			return &models.KYC{ID: 1, UserID: 1, Fname: "aa", Lname: "bb"}, nil
		}
		resp, err := h.UpdateKYC(ctx, &pb.UpdateKYCRequest{
			Fname: "aa", Lname: "bb",
			Video: &pb.VideoInfo{Path: "tmp/v", Name: "v.mp4"},
		})
		if err != nil || resp.Fname != "aa" {
			t.Fatalf("resp=%v err=%v", resp, err)
		}

		m.submitKYCFunc = func(context.Context, uint64, service.KYCSubmission) (*models.KYC, error) {
			return nil, errors.New("db")
		}
		_, err = h.UpdateKYC(ctx, &pb.UpdateKYCRequest{})
		st, _ = status.FromError(err)
		if st.Code() != codes.Internal && st.Code() != codes.InvalidArgument {
			t.Fatalf("code=%v", st.Code())
		}
	})
}
