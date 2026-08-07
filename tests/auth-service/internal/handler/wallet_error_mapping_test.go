package handler_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/auth-service/internal/handler"
	pb "metarang/shared/pb/auth"
	"metarang/shared/pkg/helpers"
)

func TestWriteWalletGRPCErrorMapping(t *testing.T) {
	s := grpc.NewServer()
	defer s.Stop()
	clients := handler.NewLocalClients(
		&pb.UnimplementedAuthServiceServer{},
		&pb.UnimplementedUserServiceServer{},
		&pb.UnimplementedKYCServiceServer{},
		&pb.UnimplementedCitizenServiceServer{},
		&pb.UnimplementedPersonalInfoServiceServer{},
		&pb.UnimplementedProfileLimitationServiceServer{},
		&pb.UnimplementedProfilePhotoServiceServer{},
		&pb.UnimplementedSettingsServiceServer{},
		&pb.UnimplementedUserEventsServiceServer{},
		&pb.UnimplementedSearchServiceServer{},
		handler.RegisterWalletConnectionHandler(s, &mockWalletConnectionService{}, "en"),
	)
	h := handler.NewHTTPWalletHandler(clients.WalletConnection, "en")

	cases := []struct {
		err  error
		code int
	}{
		{errors.New("plain"), http.StatusInternalServerError},
		{status.Error(codes.Unauthenticated, "u"), http.StatusUnauthorized},
		{status.Error(codes.PermissionDenied, "p"), http.StatusForbidden},
		{status.Error(codes.FailedPrecondition, "f"), http.StatusUnprocessableEntity},
		{status.Error(codes.NotFound, "n"), http.StatusNotFound},
		{status.Error(codes.Internal, "i"), http.StatusInternalServerError},
		{status.Error(codes.InvalidArgument, "bad"), http.StatusUnprocessableEntity},
		{status.Error(codes.InvalidArgument, helpers.EncodeValidationError(map[string]string{"address": "required"})), http.StatusUnprocessableEntity},
	}
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		handler.WriteWalletGRPCErrorForTest(h, rr, tc.err)
		if rr.Code != tc.code {
			t.Fatalf("err=%v code=%d want=%d body=%s", tc.err, rr.Code, tc.code, rr.Body.String())
		}
	}
}
