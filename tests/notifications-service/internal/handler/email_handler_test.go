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

func newEmailClient(t *testing.T, svc *testutil.MockEmailService) pb.EmailServiceClient {
	s := testutil.NewBufGRPCTestServer()
	handler.RegisterEmailHandler(s.Server, svc)
	s.Start(t)
	return pb.NewEmailServiceClient(s.GRPCClientConn(t))
}

func TestEmailHandler_SendEmail(t *testing.T) {
	ctx := context.Background()
	client := newEmailClient(t, &testutil.MockEmailService{
		SendEmailFunc: func(_ context.Context, payload models.EmailPayload) (string, error) {
			assert.Equal(t, "user@example.com", payload.To)
			return "email-123", nil
		},
	})

	resp, err := client.SendEmail(ctx, &pb.SendEmailRequest{To: "user@example.com", Subject: "Hello", Body: "World"})
	require.NoError(t, err)
	assert.True(t, resp.Sent)
	assert.Equal(t, "email-123", resp.MessageId)

	for _, tc := range []struct {
		name string
		req  *pb.SendEmailRequest
	}{
		{"missing to", &pb.SendEmailRequest{Subject: "s", Body: "b"}},
		{"missing subject", &pb.SendEmailRequest{To: "a@b.com", Body: "b"}},
		{"missing body", &pb.SendEmailRequest{To: "a@b.com", Subject: "s"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.SendEmail(ctx, tc.req)
			require.Error(t, err)
			st, _ := status.FromError(err)
			assert.Equal(t, codes.InvalidArgument, st.Code())
		})
	}

	t.Run("not implemented", func(t *testing.T) {
		client := newEmailClient(t, &testutil.MockEmailService{
			SendEmailFunc: func(context.Context, models.EmailPayload) (string, error) {
				return "", errs.ErrNotImplemented
			},
		})
		_, err := client.SendEmail(ctx, &pb.SendEmailRequest{To: "a@b.com", Subject: "s", Body: "b"})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Unimplemented, st.Code())
	})

	t.Run("service error", func(t *testing.T) {
		client := newEmailClient(t, &testutil.MockEmailService{
			SendEmailFunc: func(context.Context, models.EmailPayload) (string, error) {
				return "", errors.New("smtp down")
			},
		})
		_, err := client.SendEmail(ctx, &pb.SendEmailRequest{To: "a@b.com", Subject: "s", Body: "b"})
		require.Error(t, err)
		st, _ := status.FromError(err)
		assert.Equal(t, codes.Internal, st.Code())
	})
}
