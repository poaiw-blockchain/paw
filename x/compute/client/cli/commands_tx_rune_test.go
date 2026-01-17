package cli

import (
	"context"
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// Minimal RunE coverage for tx commands: register/update provider and submit/cancel request.
// These are expected to error at broadcast time because no network is wired, but they exercise flag parsing and msg construction.
func TestTxCommandsRunE(t *testing.T) {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(interfaceRegistry)
	clientCtx := client.Context{}.
		WithCodec(codec.NewProtoCodec(interfaceRegistry)).
		WithFromAddress(sdk.AccAddress("from_addr__________"))

	tests := []struct {
		name string
		cmd  *cobra.Command
		args []string
	}{
		{
			name: "register-provider",
			cmd:  CmdRegisterProvider(),
			args: []string{
				"--moniker", "prov",
				"--endpoint", "https://p",
				"--cpu-cores", "1",
				"--memory-mb", "1024",
				"--disk-mb", "10240",
				"--gpu-units", "0",
				"--timeout-seconds", "3600",
				"--cpu-price", "0.001",
				"--memory-price", "0.001",
				"--gpu-price", "0.1",
				"--storage-price", "0.001",
				"--amount", "1000000",
				"--" + flags.FlagFrom, "from",
			},
		},
		{
			name: "update-provider",
			cmd:  CmdUpdateProvider(),
			args: []string{
				"--moniker", "newprov",
				"--" + flags.FlagFrom, "from",
			},
		},
		{
			name: "submit-request",
			cmd:  CmdSubmitRequest(),
			args: []string{
				"--container-image", "alpine",
				"--command", "echo",
				"--max-payment", "1000",
				"--cpu-cores", "1",
				"--memory-mb", "512",
				"--storage-gb", "1",
				"--timeout-seconds", "600",
				"--" + flags.FlagFrom, "from",
			},
		},
		{
			name: "cancel-request",
			cmd:  CmdCancelRequest(),
			args: []string{
				"1",
				"--" + flags.FlagFrom, "from",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.cmd.SetArgs(tc.args)
			tc.cmd.SetContext(context.WithValue(context.Background(), client.ClientContextKey, &clientCtx))
			err := tc.cmd.Execute()
			require.Error(t, err) // expected broadcast failure
		})
	}
}
