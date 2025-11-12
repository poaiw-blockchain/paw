package cometmock

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// MockConfig provides configuration options for CometMock
type MockConfig struct {
	// Consensus parameters
	BlockTime    time.Duration
	MaxBlockSize int64
	MaxGas       int64

	// Network parameters
	ChainID       string
	NumValidators int
	InitialHeight int64

	// Testing parameters
	AutoCommit    bool
	EnableLogging bool
	FastMode      bool // Skip validation and go straight to block production

	// Account configuration
	AccountFunding sdk.Coins
	NumAccounts    int
}

// DefaultMockConfig returns sensible defaults for testing
func DefaultMockConfig() MockConfig {
	return MockConfig{
		BlockTime:      1 * time.Second, // Fast blocks for testing
		MaxBlockSize:   200000,
		MaxGas:         2000000,
		ChainID:        "paw-test-1",
		NumValidators:  4,
		InitialHeight:  1,
		AutoCommit:     true,
		EnableLogging:  false,
		FastMode:       true,
		AccountFunding: sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000000)),
		NumAccounts:    10,
	}
}

// FastMockConfig returns configuration optimized for speed
func FastMockConfig() MockConfig {
	cfg := DefaultMockConfig()
	cfg.BlockTime = 100 * time.Millisecond
	cfg.NumValidators = 1
	cfg.FastMode = true
	return cfg
}

// RealisticMockConfig returns configuration similar to production
func RealisticMockConfig() MockConfig {
	cfg := DefaultMockConfig()
	cfg.BlockTime = 5 * time.Second
	cfg.NumValidators = 100
	cfg.FastMode = false
	cfg.EnableLogging = true
	return cfg
}

// WithChainID sets the chain ID
func (c MockConfig) WithChainID(chainID string) MockConfig {
	c.ChainID = chainID
	return c
}

// WithValidators sets the number of validators
func (c MockConfig) WithValidators(num int) MockConfig {
	c.NumValidators = num
	return c
}

// WithBlockTime sets the block time
func (c MockConfig) WithBlockTime(duration time.Duration) MockConfig {
	c.BlockTime = duration
	return c
}

// WithFastMode enables or disables fast mode
func (c MockConfig) WithFastMode(enabled bool) MockConfig {
	c.FastMode = enabled
	return c
}

// WithLogging enables or disables logging
func (c MockConfig) WithLogging(enabled bool) MockConfig {
	c.EnableLogging = enabled
	return c
}

// WithAccountFunding sets the initial funding for test accounts
func (c MockConfig) WithAccountFunding(coins sdk.Coins) MockConfig {
	c.AccountFunding = coins
	return c
}

// WithNumAccounts sets the number of pre-funded test accounts
func (c MockConfig) WithNumAccounts(num int) MockConfig {
	c.NumAccounts = num
	return c
}

// Validate checks if the configuration is valid
func (c MockConfig) Validate() error {
	if c.NumValidators < 1 {
		return ErrInvalidValidatorCount
	}
	if c.BlockTime < 1*time.Millisecond {
		return ErrInvalidBlockTime
	}
	if c.MaxBlockSize < 1 {
		return ErrInvalidBlockSize
	}
	if c.MaxGas < 1 {
		return ErrInvalidMaxGas
	}
	if c.ChainID == "" {
		return ErrMissingChainID
	}
	if c.InitialHeight < 1 {
		return ErrInvalidInitialHeight
	}
	return nil
}
