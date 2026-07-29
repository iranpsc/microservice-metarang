package handler_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/notifications-service/internal/errs"
	"metarang/notifications-service/internal/handler"
	"metarang/notifications-service/internal/models"
	"metarang/notifications-service/tests/internal/testutil"
	pb "metarang/shared/pb/notifications"
)

func newSMSClient(t *testing.T, svc *testutil.MockSMSService) pb.SMSServiceClient {
	s := testutil.NewBufGRPCTestServer()
	handler.RegisterSMSHandler(s.Server, svc)
	s.Start(t)
	return pb.NewSMSServiceClient(s.GRPCClientConn(t))
}

func TestSMSHandler_SendSMS(t *testing.T) {
	ctx := context.Background()
	client := newSMSClient(t, &testutil.MockSMSService{
		SendSMSFunc: func(_ context.Context, payload models.SMSPayload) (string, error) {
			assert.Equal(t, "09120000000", payload.Phone)
			return "sms-123", nil
		},
	})

	resp, err := client.SendSMS(ctx, &pb.SendSMSRequest{Phone: "09120000000", Message: "hello"})
	require.NoError(t, err)
	assert.True(t, resp.Sent)
	assert.Equal(t, "sms-123", resp.MessageId)
	assert.Equal(t, "queued", resp.Status)

	t.Run("missing phone", func(t *testing.T) {
		_, err := client.SendSMS(ctx, &pb.SendSMSRequest{Message: "hello"})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("missing message and template", func(t *testing.T) {
		_, err := client.SendSMS(ctx, &pb.SendSMSRequest{Phone: "09120000000"})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})

	t.Run("not configured", func(t *testing.T) {
		client := newSMSClient(t, &testutil.MockSMSService{
			SendSMSFunc: func(context.Context, models.SMSPayload) (string, error) {
				return "", errs.ErrNotImplemented
			},
		})
		_, err := client.SendSMS(ctx, &pb.SendSMSRequest{Phone: "09120000000", Message: "hello"})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.FailedPrecondition, st.Code())
	})

	t.Run("service error", func(t *testing.T) {
		client := newSMSClient(t, &testutil.MockSMSService{
			SendSMSFunc: func(context.Context, models.SMSPayload) (string, error) {
				return "", errors.New("provider down")
			},
		})
		_, err := client.SendSMS(ctx, &pb.SendSMSRequest{Phone: "09120000000", Message: "hello"})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
	})
}

func TestSMSHandler_SendOTP(t *testing.T) {
	ctx := context.Background()
	client := newSMSClient(t, &testutil.MockSMSService{
		SendOTPFunc: func(_ context.Context, payload models.OTPPayload) (string, error) {
			assert.Equal(t, "123456", payload.Code)
			return "otp-123", nil
		},
	})

	resp, err := client.SendOTP(ctx, &pb.SendOTPRequest{Phone: "09120000000", Code: "123456"})
	require.NoError(t, err)
	assert.True(t, resp.Sent)

	t.Run("missing code", func(t *testing.T) {
		_, err := client.SendOTP(ctx, &pb.SendOTPRequest{Phone: "09120000000"})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.InvalidArgument, st.Code())
	})
}
