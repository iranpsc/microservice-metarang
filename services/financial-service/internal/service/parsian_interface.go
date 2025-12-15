package service

import "metargb/financial-service/internal/parsian"

// ParsianClient interface for payment gateway operations
// Allows for easier testing with mocks
type ParsianClient interface {
	RequestPayment(params parsian.RequestParams) (*parsian.RequestResponse, error)
	VerifyPayment(params parsian.VerificationParams) (*parsian.VerificationResponse, error)
}
