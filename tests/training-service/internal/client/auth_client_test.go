package client_test

import (
	"testing"

	"metarang/training-service/internal/client"
)

func TestAuthClient_CloseNilConn(t *testing.T) {
	c := &client.AuthClient{}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
