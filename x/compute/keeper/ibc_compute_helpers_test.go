package keeper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMax32AndHashBytes(t *testing.T) {
	require.Equal(t, uint32(5), max32(5, 3))
	require.Equal(t, uint32(7), max32(4, 7))

	require.Equal(t, "", hashBytes(nil))
	require.NotEmpty(t, hashBytes([]byte("data")))
}
