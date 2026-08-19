package constants_test

import (
	"testing"

	"metarang/features-service/internal/constants"

	"github.com/stretchr/testify/assert"
)

func TestFeeCalculations(t *testing.T) {
	assert.Equal(t, 105.0, constants.CalculateBuyerCharge(100))
	assert.Equal(t, 95.0, constants.CalculateSellerPayment(100))
	assert.Equal(t, 10.0, constants.CalculatePlatformFee(100))
	assert.Equal(t, 5.0, constants.CalculateFee(100))
}

func TestFeeZeroSum(t *testing.T) {
	prices := []float64{0, 1, 100, 999.5}
	for _, p := range prices {
		assert.InDelta(t, constants.CalculatePlatformFee(p),
			constants.CalculateBuyerCharge(p)-constants.CalculateSellerPayment(p),
			1e-9)
	}
}
