package handler_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/auth-service/internal/handler"
	"metarang/auth-service/internal/models"
	"metarang/auth-service/internal/service"
	pb "metarang/shared/pb/auth"
)

type mockPersonalInfoService struct {
	getFunc    func(context.Context, uint64) (*models.PersonalInfo, error)
	updateFunc func(context.Context, uint64, string, string, string, string, string, string, string, string, string, map[string]bool) error
}

func (m *mockPersonalInfoService) GetPersonalInfo(ctx context.Context, userID uint64) (*models.PersonalInfo, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, userID)
	}
	return nil, nil
}

func (m *mockPersonalInfoService) UpdatePersonalInfo(ctx context.Context, userID uint64, occupation, education, memory, lovedCity, lovedCountry, lovedLanguage, problemSolving, prediction, about string, passions map[string]bool) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, userID, occupation, education, memory, lovedCity, lovedCountry, lovedLanguage, problemSolving, prediction, about, passions)
	}
	return nil
}

var _ service.PersonalInfoService = (*mockPersonalInfoService)(nil)

func TestPersonalInfoHandler(t *testing.T) {
	ctx := authenticatedContext(1)
	h := handler.RegisterPersonalInfoHandler(grpc.NewServer(), &mockPersonalInfoService{})

	t.Run("unauth", func(t *testing.T) {
		_, err := h.GetPersonalInfo(context.Background(), &pb.GetPersonalInfoRequest{})
		st, _ := status.FromError(err)
		if st.Code() != codes.Unauthenticated {
			t.Fatalf("code=%v", st.Code())
		}
	})

	t.Run("get empty", func(t *testing.T) {
		resp, err := h.GetPersonalInfo(ctx, &pb.GetPersonalInfoRequest{})
		if err != nil || resp.Data == nil {
			t.Fatalf("resp=%v err=%v", resp, err)
		}
	})

	t.Run("get with data", func(t *testing.T) {
		m := &mockPersonalInfoService{}
		m.getFunc = func(context.Context, uint64) (*models.PersonalInfo, error) {
			return &models.PersonalInfo{
				Occupation: sql.NullString{String: "eng", Valid: true},
				Passions:   map[string]bool{"music": true},
			}, nil
		}
		h := handler.RegisterPersonalInfoHandler(grpc.NewServer(), m)
		resp, err := h.GetPersonalInfo(ctx, &pb.GetPersonalInfoRequest{})
		if err != nil || resp.Data.Occupation != "eng" || !resp.Data.Passions["music"] {
			t.Fatalf("resp=%v err=%v", resp, err)
		}
	})

	t.Run("update errors", func(t *testing.T) {
		m := &mockPersonalInfoService{}
		m.updateFunc = func(context.Context, uint64, string, string, string, string, string, string, string, string, string, map[string]bool) error {
			return service.ErrInvalidOccupation
		}
		h := handler.RegisterPersonalInfoHandler(grpc.NewServer(), m)
		_, err := h.UpdatePersonalInfo(ctx, &pb.UpdatePersonalInfoRequest{Occupation: "x"})
		st, _ := status.FromError(err)
		if st.Code() != codes.InvalidArgument {
			t.Fatalf("code=%v", st.Code())
		}

		m.updateFunc = func(context.Context, uint64, string, string, string, string, string, string, string, string, string, map[string]bool) error {
			return errors.New("db")
		}
		_, err = h.UpdatePersonalInfo(ctx, &pb.UpdatePersonalInfoRequest{})
		st, _ = status.FromError(err)
		if st.Code() != codes.Internal {
			t.Fatalf("code=%v", st.Code())
		}
	})

	t.Run("update success", func(t *testing.T) {
		_, err := h.UpdatePersonalInfo(ctx, &pb.UpdatePersonalInfoRequest{
			Occupation: "dev",
			Passions:   map[string]bool{"art": true},
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}
