package metrics_test

import (
	"testing"

	"metarang/features-service/internal/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestMarketplaceMetrics_Records(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := metrics.NewMarketplaceMetricsWithRegisterer(reg)
	require.NotNil(t, m)

	m.RecordBuyRequest("accepted")
	m.RecordBuyRequest("rejected")
	m.RecordSellRequest()
	m.RecordTrade("user", 10, 1000)
	m.RecordTrade("rgb", 0, 0)
	m.UpdateLockedAssets(5, 50)

	families, err := reg.Gather()
	require.NoError(t, err)
	require.NotEmpty(t, families)
}

func TestNewMarketplaceMetrics_DefaultRegistry(t *testing.T) {
	// Cover the default-registry wrapper once per process.
	m := metrics.NewMarketplaceMetrics()
	require.NotNil(t, m)
	m.RecordSellRequest()
}
