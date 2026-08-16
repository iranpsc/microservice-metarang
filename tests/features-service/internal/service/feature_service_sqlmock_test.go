package service_test

import (
	"context"
	"testing"

	"metarang/features-service/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeatureService_ListFeatures_RequiresFourPoints(t *testing.T) {
	svc := service.NewFeatureService(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, "")
	_, err := svc.ListFeatures(context.Background(), []string{"1,1"}, false, false, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least 4")
}
