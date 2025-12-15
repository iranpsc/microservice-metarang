//go:build nocommercial
// +build nocommercial

package client

import (
	"context"
	"fmt"
)

// CommercialClient wraps gRPC clients for Commercial Service
type CommercialClient interface {
	AddBalance(ctx context.Context, userID uint64, asset string, amount float64) error
	Close() error
}

type commercialClientStub struct{}

// NewCommercialClient creates a stub client when proto files are not available
func NewCommercialClient(address string) (CommercialClient, error) {
	return &commercialClientStub{}, nil
}

// Close closes the stub client
func (c *commercialClientStub) Close() error {
	return nil
}

// AddBalance is a stub that returns an error indicating proto files are not generated
func (c *commercialClientStub) AddBalance(ctx context.Context, userID uint64, asset string, amount float64) error {
	return fmt.Errorf("commercial service client not available: proto files not generated. Run 'make proto' to generate them")
}
