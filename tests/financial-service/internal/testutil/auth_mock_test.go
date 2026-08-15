package testutil

import "testing"

func TestMockAuthGRPCClientSatisfiesInterface(t *testing.T) {
	var _ interface{} = &MockAuthGRPCClient{}
}
