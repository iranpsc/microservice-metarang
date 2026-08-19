package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"metarang/dynasty-service/internal/client"
	pbCommon "metarang/shared/pb/common"
	pb "metarang/shared/pb/notifications"
)

type fakeNotificationServiceClient struct {
	lastRequest *pb.SendNotificationRequest
	requests    []*pb.SendNotificationRequest
	resp        *pb.NotificationResponse
	err         error
}

func newFakeNotificationServiceClient() *fakeNotificationServiceClient {
	return &fakeNotificationServiceClient{
		resp: &pb.NotificationResponse{Id: 1, Sent: true},
	}
}

func (f *fakeNotificationServiceClient) SendNotification(_ context.Context, in *pb.SendNotificationRequest, _ ...grpc.CallOption) (*pb.NotificationResponse, error) {
	f.lastRequest = in
	f.requests = append(f.requests, in)
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func (f *fakeNotificationServiceClient) GetNotifications(context.Context, *pb.GetNotificationsRequest, ...grpc.CallOption) (*pb.NotificationsResponse, error) {
	return nil, nil
}
func (f *fakeNotificationServiceClient) GetNotification(context.Context, *pb.GetNotificationRequest, ...grpc.CallOption) (*pb.Notification, error) {
	return nil, nil
}
func (f *fakeNotificationServiceClient) MarkAsRead(context.Context, *pb.MarkAsReadRequest, ...grpc.CallOption) (*pbCommon.Empty, error) {
	return nil, nil
}
func (f *fakeNotificationServiceClient) MarkAllAsRead(context.Context, *pb.MarkAllAsReadRequest, ...grpc.CallOption) (*pbCommon.Empty, error) {
	return nil, nil
}

var _ pb.NotificationServiceClient = (*fakeNotificationServiceClient)(nil)

func TestNotificationClient_SendNotification_MapsRequestFields(t *testing.T) {
	fake := newFakeNotificationServiceClient()
	c := client.NewNotificationClientFromGRPC(fake, nil)

	data := map[string]string{"relationship": "brother"}
	err := c.SendNotification(context.Background(), 42, "dynasty_join_request", "Dynasty", "hello", data, false, false)
	require.NoError(t, err)
	require.NotNil(t, fake.lastRequest)
	assert.Equal(t, uint64(42), fake.lastRequest.UserId)
	assert.Equal(t, "dynasty_join_request", fake.lastRequest.Type)
	assert.Equal(t, "Dynasty", fake.lastRequest.Title)
	assert.Equal(t, "hello", fake.lastRequest.Message)
	assert.Equal(t, data, fake.lastRequest.Data)
	assert.False(t, fake.lastRequest.SendSms)
	assert.False(t, fake.lastRequest.SendEmail)
}

func TestNotificationClient_SendNotification_SendSmsTrue_And_SendEmailTrue(t *testing.T) {
	fake := newFakeNotificationServiceClient()
	c := client.NewNotificationClientFromGRPC(fake, nil)

	err := c.SendNotification(context.Background(), 7, "type", "title", "msg", nil, true, true)
	require.NoError(t, err)
	require.NotNil(t, fake.lastRequest)
	assert.True(t, fake.lastRequest.SendSms, "SendSms must forward true")
	assert.True(t, fake.lastRequest.SendEmail, "SendEmail must forward true")
}

func TestNotificationClient_SendNotification_ErrorWhenSentFalse(t *testing.T) {
	fake := newFakeNotificationServiceClient()
	fake.resp = &pb.NotificationResponse{Id: 9, Sent: false}
	c := client.NewNotificationClientFromGRPC(fake, nil)

	err := c.SendNotification(context.Background(), 1, "t", "ti", "m", nil, false, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "notification was not sent")
}

func TestNotificationClient_SendNotification_gRPCFailure(t *testing.T) {
	fake := newFakeNotificationServiceClient()
	fake.err = status.Error(codes.Unavailable, "notifications down")
	c := client.NewNotificationClientFromGRPC(fake, nil)

	err := c.SendNotification(context.Background(), 1, "t", "ti", "m", nil, true, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send notification")
	assert.Contains(t, err.Error(), "notifications down")
}

func TestNotificationClient_Close_NilConn(t *testing.T) {
	c := client.NewNotificationClientFromGRPC(newFakeNotificationServiceClient(), nil)
	require.NoError(t, c.Close())
}

func TestNotificationClient_SendNotification_GenericError(t *testing.T) {
	fake := newFakeNotificationServiceClient()
	fake.err = errors.New("transport boom")
	c := client.NewNotificationClientFromGRPC(fake, nil)
	err := c.SendNotification(context.Background(), 1, "t", "ti", "m", map[string]string{"k": "v"}, false, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transport boom")
}
