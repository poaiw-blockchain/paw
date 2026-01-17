package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Smoke-test that all command builders construct without panicking.
// We don't execute RunE handlers (they rely on network), but constructing
// the root query/tx commands exercises all subcommand builders for coverage.
func TestCommandConstruction(t *testing.T) {
	t.Run("query commands build", func(t *testing.T) {
		cmd := GetQueryCmd()
		require.NotNil(t, cmd)
		ensureUsagesNonEmpty(t, cmd)
	})

	t.Run("tx commands build", func(t *testing.T) {
		cmd := GetTxCmd()
		require.NotNil(t, cmd)
		ensureUsagesNonEmpty(t, cmd)
	})
}

func ensureUsagesNonEmpty(t *testing.T, cmd *cobra.Command) {
	t.Helper()
	require.NotEmpty(t, cmd.Use)
	for _, c := range cmd.Commands() {
		require.NotEmpty(t, c.Use)
	}
}
