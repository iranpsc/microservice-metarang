package client_test

import (
	"context"
	"testing"

	"metarang/features-service/internal/client"
	"metarang/features-service/tests/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommercialClient_CheckBalance(t *testing.T) {
	stub := testutil.NewCommercialStub()
	stub.Psc = "1000"
	stub.Irr = "1000"
	stub.Yellow = "50"
	c := client.NewCommercialClientFromGRPC(stub, stub)

	ok, err := c.CheckBalance(context.Background(), 2, "psc", 1000)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = c.CheckBalance(context.Background(), 2, "psc", 1001)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = c.CheckBalance(context.Background(), 2, "irr", 500)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = c.CheckBalance(context.Background(), 2, "irr", 1001)
	require.NoError(t, err)
	assert.False(t, ok)

	ok, err = c.CheckBalance(context.Background(), 2, "yellow", 50)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = c.CheckBalance(context.Background(), 2, "yellow", 51)
	require.NoError(t, err)
	assert.False(t, ok)

	_, err = c.CheckBalance(context.Background(), 2, "gold", 1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown asset")
}

func TestCommercialClient_DeductBalance_SuccessFalse(t *testing.T) {
	stub := testutil.NewCommercialStub()
	stub.FailNthDeduct = 1
	c := client.NewCommercialClientFromGRPC(stub, stub)

	err := c.DeductBalance(context.Background(), 2, "psc", 10)
	require.Error(t, err)
	var ce *client.CommercialError
	require.ErrorAs(t, err, &ce)
	assert.Equal(t, "validation_error", ce.Type)
}

func TestCommercialClient_AddBalance_Success(t *testing.T) {
	stub := testutil.NewCommercialStub()
	c := client.NewCommercialClientFromGRPC(stub, stub)

	require.NoError(t, c.AddBalance(context.Background(), 5, "yellow", 10))
	require.Len(t, stub.AddCalls, 1)
	assert.Equal(t, uint64(5), stub.AddCalls[0].UserID)
	assert.Equal(t, "yellow", stub.AddCalls[0].Asset)
	assert.Equal(t, 10.0, stub.AddCalls[0].Amount)
}
