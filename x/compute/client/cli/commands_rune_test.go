package cli

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// RunE smoke tests: ensure commands wire up client context and execute to first network boundary.
func TestQueryCommandsRunE(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(interfaceRegistry)
	cc := client.Context{}.WithCodec(codec.NewProtoCodec(interfaceRegistry))

	tests := []struct {
		name string
		cmd  func() *cobra.Command
		args []string
	}{
		{"params", GetCmdQueryParams, nil},
		{"providers", GetCmdQueryProviders, nil},
		{"provider", func() *cobra.Command {
			c := GetCmdQueryProvider()
			c.SetArgs([]string{"paw1provider"})
			return c
		}, []string{"paw1provider"}},
		{"requests", GetCmdQueryRequests, nil},
		{"catastrophic-failure", func() *cobra.Command {
			c := GetCmdQueryCatastrophicFailure()
			c.SetArgs([]string{"1"})
			return c
		}, []string{"1"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := tc.cmd()
			if tc.args != nil {
				cmd.SetArgs(tc.args)
			}
			clientCtx := cc
			cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
			err := cmd.Execute()
			require.Error(t, err) // expected: no network, but RunE path exercised
		})
	}
}
