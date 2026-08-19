package models_test

import (
	"database/sql"
	"testing"
	"time"

	"metarang/features-service/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureToPB_WithPropertiesAndGeometry(t *testing.T) {
	f := &models.Feature{ID: 9, OwnerID: 2}
	props := &models.FeatureProperties{
		ID: "p1", Address: "addr", Density: 3, Area: 12.5, Stability: 8.25,
		Label: "l", Karbari: "m", Owner: "o", RGB: "a", PricePSC: "10", PriceIRR: "20",
		MinimumPricePercentage: 80,
	}
	g := &models.Geometry{ID: 4, Type: "Polygon"}

	pb := models.FeatureToPB(f, props, g)
	require.NotNil(t, pb.Properties)
	assert.Equal(t, uint64(9), pb.Id)
	assert.Equal(t, "12.50", pb.Properties.Area)
	assert.Equal(t, "8.25", pb.Properties.Stability)
	assert.Equal(t, "Polygon", pb.Geometry.Type)
}

func TestFeatureToPB_Nils(t *testing.T) {
	pb := models.FeatureToPB(&models.Feature{ID: 1, OwnerID: 2}, nil, nil)
	assert.Nil(t, pb.Properties)
	assert.Nil(t, pb.Geometry)
}

func TestFeaturesToPB(t *testing.T) {
	out := models.FeaturesToPB([]*models.Feature{{ID: 1}, {ID: 2}})
	require.Len(t, out, 2)
	assert.Equal(t, uint64(2), out[1].Id)
}

func TestTradeTimestamp(t *testing.T) {
	fallback := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	created := time.Date(2024, 6, 1, 15, 4, 5, 0, time.UTC)
	dated := time.Date(2023, 3, 2, 11, 0, 0, 0, time.UTC)

	assert.Equal(t, created, models.TradeHistoryTrade{CreatedAt: sql.NullTime{Time: created, Valid: true}}.TradeTimestamp(fallback))
	got := models.TradeHistoryTrade{Date: sql.NullTime{Time: dated, Valid: true}}.TradeTimestamp(fallback)
	assert.Equal(t, time.Date(2023, 3, 2, 0, 0, 0, 0, time.UTC), got)
	assert.Equal(t, fallback, models.TradeHistoryTrade{}.TradeTimestamp(fallback))
}

func TestNewNullString(t *testing.T) {
	assert.False(t, models.NewNullString("").Valid)
	ns := models.NewNullString("x")
	assert.True(t, ns.Valid)
	assert.Equal(t, "x", ns.String)
}
