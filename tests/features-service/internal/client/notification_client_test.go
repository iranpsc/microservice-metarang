package client_test

import (
	"context"
	"errors"
	"testing"

	"metarang/features-service/internal/client"
	"metarang/features-service/tests/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationClient_SendBuyRequestNotification(t *testing.T) {
	stub := testutil.NewNotificationStub()
	c := client.NewNotificationClientFromGRPC(stub)

	require.NoError(t, c.SendBuyRequestNotification(context.Background(), 2, "buyer", 9, 1, 10, 20))
	require.NoError(t, c.SendBuyRequestNotification(context.Background(), 3, "seller", 9, 1, 10, 20))

	require.Len(t, stub.Calls, 2)
	assert.Equal(t, "BuyRequestNotification", stub.Calls[0].Type)
	assert.Equal(t, uint64(2), stub.Calls[0].UserID)
	assert.Equal(t, "buyer", stub.Calls[0].Data["type"])
	assert.Equal(t, "transactions", stub.Calls[0].Data["related-to"])
	assert.Contains(t, stub.Calls[0].Message, "برداشت")

	assert.Equal(t, uint64(3), stub.Calls[1].UserID)
	assert.Equal(t, "seller", stub.Calls[1].Data["type"])
	assert.Contains(t, stub.Calls[1].Title, "دریافت")
}

func TestNotificationClient_SendBuyFeatureNotification(t *testing.T) {
	stub := testutil.NewNotificationStub()
	c := client.NewNotificationClientFromGRPC(stub)

	require.NoError(t, c.SendBuyFeatureNotification(context.Background(), 2, 1, true, "زرد", 10, 0, 0))
	require.NoError(t, c.SendBuyFeatureNotification(context.Background(), 2, 1, false, "", 0, 105, 105))

	require.Len(t, stub.Calls, 2)
	assert.Equal(t, "BuyFeatureNotification", stub.Calls[0].Type)
	assert.Equal(t, "rgb", stub.Calls[0].Data["purchase_type"])
	assert.Equal(t, "زرد", stub.Calls[0].Data["color"])
	assert.Contains(t, stub.Calls[0].Message, "لیتر")

	assert.Equal(t, "user", stub.Calls[1].Data["purchase_type"])
	assert.Equal(t, "105", stub.Calls[1].Data["psc_amount"])
	assert.Contains(t, stub.Calls[1].Message, "ریال")
}

func TestNotificationClient_SendSellRequestNotification(t *testing.T) {
	stub := testutil.NewNotificationStub()
	c := client.NewNotificationClientFromGRPC(stub)

	require.NoError(t, c.SendSellRequestNotification(context.Background(), 3, 1, "p1"))
	require.Len(t, stub.Calls, 1)
	assert.Equal(t, "SellRequestNotification", stub.Calls[0].Type)
	assert.Equal(t, "p1", stub.Calls[0].Data["properties_id"])
	assert.Equal(t, "sell-requests", stub.Calls[0].Data["related-to"])
}

func TestNotificationClient_SendFeatureHourlyProfitDeposit(t *testing.T) {
	stub := testutil.NewNotificationStub()
	c := client.NewNotificationClientFromGRPC(stub)

	require.NoError(t, c.SendFeatureHourlyProfitDeposit(context.Background(), 2, "yellow", 1.5, "m", "p1"))
	require.Len(t, stub.Calls, 1)
	assert.Equal(t, "FeatureHourlyProfitDeposit", stub.Calls[0].Type)
	assert.Equal(t, "yellow", stub.Calls[0].Data["asset"])
	assert.Equal(t, "مسکونی", stub.Calls[0].Data["karbari"])
	assert.Equal(t, "p1", stub.Calls[0].Data["id"])
	assert.Contains(t, stub.Calls[0].Message, "زرد")
}

func TestNotificationClient_SendNotification_RPCError(t *testing.T) {
	stub := testutil.NewNotificationStub()
	stub.Err = errors.New("unavailable")
	c := client.NewNotificationClientFromGRPC(stub)

	err := c.SendNotification(context.Background(), 1, "t", "title", "msg", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send notification")
}
