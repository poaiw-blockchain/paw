package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRootCmdTxConfigPersistsAfterClientConfigLoad(t *testing.T) {
	initSDKConfig()

	homeDir := t.TempDir()
	configDir := filepath.Join(homeDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	clientToml := `keyring-backend = "test"
node = "tcp://localhost:26657"
output = "json"
broadcast-mode = "sync"
`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "client.toml"), []byte(clientToml), 0o644))

	cmd := NewRootCmd(true)
	cmd.SetContext(context.Background())

	setPersistentFlag(t, cmd, flags.FlagHome, homeDir)
	setPersistentFlag(t, cmd, flags.FlagChainID, "paw-devnet")
	setPersistentFlag(t, cmd, flags.FlagNode, "tcp://localhost:26657")
	setPersistentFlag(t, cmd, flags.FlagKeyringBackend, "test")
	setPersistentFlag(t, cmd, flags.FlagOutput, "json")

	err := cmd.PersistentPreRunE(cmd, []string{})
	require.NoError(t, err)

	clientCtx, err := client.GetClientTxContext(cmd)
	require.NoError(t, err)

	require.NotNil(t, clientCtx.TxConfig)
	txBuilder := clientCtx.TxConfig.NewTxBuilder()
	require.NotNil(t, txBuilder)
}

func setPersistentFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	if cmd.PersistentFlags().Lookup(name) == nil {
		cmd.PersistentFlags().String(name, value, "")
	}
	require.NoError(t, cmd.PersistentFlags().Set(name, value))
}
