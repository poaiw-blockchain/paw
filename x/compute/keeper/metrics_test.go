package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetComputeMetricsSingleton(t *testing.T) {
	m1 := GetComputeMetrics()
	require.NotNil(t, m1)

	m2 := GetComputeMetrics()
	require.Equal(t, m1, m2)

	// Pre-set global instance and ensure it is returned unchanged
	computeMetrics = NewComputeMetrics()
	require.Equal(t, computeMetrics, GetComputeMetrics())
}

func TestGetComputeMetricsIdempotent(t *testing.T) {
	m1 := GetComputeMetrics()
	m2 := GetComputeMetrics()
	require.Equal(t, m1, m2)
}
