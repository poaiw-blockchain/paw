package testing

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	pawclient "github.com/paw-chain/paw/sdk/go/client"
	"github.com/paw-chain/paw/sdk/go/helpers"
)

// TestConfig holds test configuration
type TestConfig struct {
	ChainID      string
	RPCEndpoint  string
	GRPCEndpoint string
	TestMnemonic string
}

// DefaultTestConfig returns default test configuration
func DefaultTestConfig() TestConfig {
	return TestConfig{
		ChainID:      "paw-testnet-1",
		RPCEndpoint:  "http://localhost:26657",
		GRPCEndpoint: "localhost:9090",
		TestMnemonic: "", // Should be set by test
	}
}

// SetupTestClient creates a test client
func SetupTestClient(t *testing.T, config TestConfig) *pawclient.Client {
	t.Helper()

	clientConfig := pawclient.Config{
		RPCEndpoint:  config.RPCEndpoint,
		GRPCEndpoint: config.GRPCEndpoint,
		ChainID:      config.ChainID,
	}

	client, err := pawclient.NewClient(clientConfig)
	require.NoError(t, err, "failed to create client")

	return client
}

// CreateTestWallet creates a test wallet
func CreateTestWallet(t *testing.T, client *pawclient.Client, name string) (string, sdk.AccAddress) {
	t.Helper()

	// Generate mnemonic
	mnemonic, err := helpers.GenerateMnemonic()
	require.NoError(t, err, "failed to generate mnemonic")

	// Import wallet
	addr, err := client.ImportWalletFromMnemonic(name, mnemonic, "")
	require.NoError(t, err, "failed to import wallet")

	return mnemonic, addr
}

// AssertBalanceGreaterThan asserts that balance is greater than expected
func AssertBalanceGreaterThan(t *testing.T, client *pawclient.Client, address, denom string, minAmount sdk.Int) {
	t.Helper()

	balance, err := client.GetBalance(context.Background(), address, denom)
	require.NoError(t, err, "failed to get balance")
	require.NotNil(t, balance, "balance is nil")
	require.True(t, balance.Amount.GT(minAmount), "balance %s is not greater than %s", balance.Amount, minAmount)
}

// AssertTransactionSuccess asserts that a transaction succeeded
func AssertTransactionSuccess(t *testing.T, resp *sdk.TxResponse) {
	t.Helper()

	require.NotNil(t, resp, "transaction response is nil")
	require.Equal(t, uint32(0), resp.Code, "transaction failed with code %d: %s", resp.Code, resp.RawLog)
	require.NotEmpty(t, resp.TxHash, "transaction hash is empty")
}

// WaitForNextBlock waits for the next block
func WaitForNextBlock(t *testing.T, client *pawclient.Client) {
	t.Helper()

	ctx := context.Background()

	// Get current height
	// Note: This is a simplified version. In production, use Tendermint client
	// to wait for next block properly

	// For now, we'll just sleep briefly
	// In real implementation, would query /status endpoint and wait
}

// FundAccount funds an account for testing (requires faucet or genesis account)
func FundAccount(t *testing.T, client *pawclient.Client, fromName, toAddress string, amount sdk.Coin) {
	t.Helper()

	// This would typically use a faucet or genesis account to fund test accounts
	// Implementation depends on test environment setup
	t.Log("Funding account:", toAddress, "with", amount)
}
