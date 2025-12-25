package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
)

func TestDefaultGenesis(t *testing.T) {
	genesis := DefaultGenesis()

	if genesis == nil {
		t.Fatal("DefaultGenesis() returned nil")
	}

	// Validate default parameters
	if genesis.Params.SwapFee.IsNil() || genesis.Params.SwapFee.IsNegative() {
		t.Errorf("Invalid default swap fee: %v", genesis.Params.SwapFee)
	}

	if genesis.Params.LpFee.IsNil() || genesis.Params.LpFee.IsNegative() {
		t.Errorf("Invalid default LP fee: %v", genesis.Params.LpFee)
	}

	if genesis.Params.ProtocolFee.IsNil() || genesis.Params.ProtocolFee.IsNegative() {
		t.Errorf("Invalid default protocol fee: %v", genesis.Params.ProtocolFee)
	}

	// Verify fee relationship
	if genesis.Params.LpFee.Add(genesis.Params.ProtocolFee).GT(genesis.Params.SwapFee) {
		t.Error("LP fee + protocol fee exceeds swap fee")
	}

	if genesis.Params.MinLiquidity.IsNegative() {
		t.Error("Minimum liquidity is negative")
	}

	if genesis.NextPoolId != 1 {
		t.Errorf("Expected NextPoolId to be 1, got %d", genesis.NextPoolId)
	}

	if genesis.Pools == nil {
		t.Error("Pools slice is nil")
	}

	if genesis.PoolTwapRecords == nil {
		t.Error("PoolTwapRecords slice is nil")
	}

	if genesis.SwapCommits == nil {
		t.Error("SwapCommits slice is nil")
	}

	// Validate genesis state
	if err := genesis.Validate(); err != nil {
		t.Errorf("Default genesis failed validation: %v", err)
	}
}

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		genesis func() *GenesisState
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid default genesis",
			genesis: func() *GenesisState {
				return DefaultGenesis()
			},
			wantErr: false,
		},
		{
			name: "negative swap fee",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.SwapFee = sdkmath.LegacyMustNewDecFromStr("-0.003")
				return gs
			},
			wantErr: true,
			errMsg:  "swap fee must be in [0,1)",
		},
		{
			name: "swap fee >= 1",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.SwapFee = sdkmath.LegacyMustNewDecFromStr("1.0")
				return gs
			},
			wantErr: true,
			errMsg:  "swap fee must be in [0,1)",
		},
		{
			name: "negative LP fee",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.LpFee = sdkmath.LegacyMustNewDecFromStr("-0.001")
				return gs
			},
			wantErr: true,
			errMsg:  "lp/protocol fees must be non-negative",
		},
		{
			name: "negative protocol fee",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.ProtocolFee = sdkmath.LegacyMustNewDecFromStr("-0.001")
				return gs
			},
			wantErr: true,
			errMsg:  "lp/protocol fees must be non-negative",
		},
		{
			name: "LP + protocol fees exceed swap fee",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.SwapFee = sdkmath.LegacyMustNewDecFromStr("0.003")
				gs.Params.LpFee = sdkmath.LegacyMustNewDecFromStr("0.002")
				gs.Params.ProtocolFee = sdkmath.LegacyMustNewDecFromStr("0.002")
				return gs
			},
			wantErr: true,
			errMsg:  "lp fee plus protocol fee must not exceed swap fee",
		},
		{
			name: "negative min liquidity",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.MinLiquidity = sdkmath.NewInt(-1000)
				return gs
			},
			wantErr: true,
			errMsg:  "min liquidity must be non-negative",
		},
		{
			name: "negative max slippage",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.MaxSlippagePercent = sdkmath.LegacyMustNewDecFromStr("-0.05")
				return gs
			},
			wantErr: true,
			errMsg:  "max slippage percent must be between 0 and 1",
		},
		{
			name: "max slippage > 1",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.MaxSlippagePercent = sdkmath.LegacyMustNewDecFromStr("1.5")
				return gs
			},
			wantErr: true,
			errMsg:  "max slippage percent must be between 0 and 1",
		},
		{
			name: "negative max pool drain",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.MaxPoolDrainPercent = sdkmath.LegacyMustNewDecFromStr("-0.1")
				return gs
			},
			wantErr: true,
			errMsg:  "max pool drain percent must be between 0 and 1",
		},
		{
			name: "max pool drain > 1",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.MaxPoolDrainPercent = sdkmath.LegacyMustNewDecFromStr("1.1")
				return gs
			},
			wantErr: true,
			errMsg:  "max pool drain percent must be between 0 and 1",
		},
		{
			name: "zero flash loan protection blocks",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.FlashLoanProtectionBlocks = 0
				return gs
			},
			wantErr: true,
			errMsg:  "flash loan protection blocks must be positive",
		},
		{
			name: "zero pool creation gas",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.PoolCreationGas = 0
				return gs
			},
			wantErr: true,
			errMsg:  "pool creation gas must be positive",
		},
		{
			name: "zero swap validation gas",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.SwapValidationGas = 0
				return gs
			},
			wantErr: true,
			errMsg:  "swap validation gas must be positive",
		},
		{
			name: "zero liquidity gas",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.LiquidityGas = 0
				return gs
			},
			wantErr: true,
			errMsg:  "liquidity gas must be positive",
		},
		{
			name: "empty authorized channel port_id",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.AuthorizedChannels = []AuthorizedChannel{
					{PortId: "", ChannelId: "channel-0"},
				}
				return gs
			},
			wantErr: true,
			errMsg:  "authorized channel port_id cannot be empty",
		},
		{
			name: "whitespace authorized channel port_id",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.AuthorizedChannels = []AuthorizedChannel{
					{PortId: "  ", ChannelId: "channel-0"},
				}
				return gs
			},
			wantErr: true,
			errMsg:  "authorized channel port_id cannot be empty",
		},
		{
			name: "empty authorized channel channel_id",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.AuthorizedChannels = []AuthorizedChannel{
					{PortId: "transfer", ChannelId: ""},
				}
				return gs
			},
			wantErr: true,
			errMsg:  "authorized channel channel_id cannot be empty",
		},
		{
			name: "negative recommended max slippage",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.RecommendedMaxSlippage = sdkmath.LegacyMustNewDecFromStr("-0.03")
				return gs
			},
			wantErr: true,
			errMsg:  "recommended max slippage must be between 0 and 1",
		},
		{
			name: "recommended max slippage > 1",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.RecommendedMaxSlippage = sdkmath.LegacyMustNewDecFromStr("1.5")
				return gs
			},
			wantErr: true,
			errMsg:  "recommended max slippage must be between 0 and 1",
		},
		{
			name: "commit-reveal enabled with zero delay",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.EnableCommitReveal = true
				gs.Params.CommitRevealDelay = 0
				return gs
			},
			wantErr: true,
			errMsg:  "commit reveal delay must be positive when commit-reveal is enabled",
		},
		{
			name: "commit-reveal enabled with zero timeout",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.EnableCommitReveal = true
				gs.Params.CommitTimeoutBlocks = 0
				return gs
			},
			wantErr: true,
			errMsg:  "commit timeout blocks must be positive when commit-reveal is enabled",
		},
		{
			name: "commit-reveal timeout <= delay",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.EnableCommitReveal = true
				gs.Params.CommitRevealDelay = 10
				gs.Params.CommitTimeoutBlocks = 10
				return gs
			},
			wantErr: true,
			errMsg:  "commit timeout blocks must be greater than commit reveal delay",
		},
		{
			name: "zero next pool id",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.NextPoolId = 0
				return gs
			},
			wantErr: true,
			errMsg:  "next pool id must be positive",
		},
		{
			name: "pool TWAP record missing pool id",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.PoolTwapRecords = []PoolTWAP{
					{PoolId: 0},
				}
				return gs
			},
			wantErr: true,
			errMsg:  "pool TWAP record missing pool id",
		},
		{
			name: "swap commit missing trader",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.SwapCommits = []SwapCommit{
					{
						Trader:       "",
						SwapHash:     "hash123",
						CommitHeight: 100,
						ExpiryHeight: 200,
					},
				}
				return gs
			},
			wantErr: true,
			errMsg:  "swap commit missing trader address",
		},
		{
			name: "swap commit missing swap hash",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.SwapCommits = []SwapCommit{
					{
						Trader:       "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
						SwapHash:     "",
						CommitHeight: 100,
						ExpiryHeight: 200,
					},
				}
				return gs
			},
			wantErr: true,
			errMsg:  "swap commit missing swap hash",
		},
		{
			name: "swap commit with zero commit height",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.SwapCommits = []SwapCommit{
					{
						Trader:       "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
						SwapHash:     "hash123",
						CommitHeight: 0,
						ExpiryHeight: 200,
					},
				}
				return gs
			},
			wantErr: true,
			errMsg:  "swap commit has invalid commit height",
		},
		{
			name: "swap commit with negative commit height",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.SwapCommits = []SwapCommit{
					{
						Trader:       "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
						SwapHash:     "hash123",
						CommitHeight: -1,
						ExpiryHeight: 200,
					},
				}
				return gs
			},
			wantErr: true,
			errMsg:  "swap commit has invalid commit height",
		},
		{
			name: "swap commit expiry <= commit height",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.SwapCommits = []SwapCommit{
					{
						Trader:       "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
						SwapHash:     "hash123",
						CommitHeight: 100,
						ExpiryHeight: 100,
					},
				}
				return gs
			},
			wantErr: true,
			errMsg:  "swap commit expiry height must be greater than commit height",
		},
		{
			name: "valid genesis with authorized channels",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.AuthorizedChannels = []AuthorizedChannel{
					{PortId: "transfer", ChannelId: "channel-0"},
					{PortId: "dex", ChannelId: "channel-1"},
				}
				return gs
			},
			wantErr: false,
		},
		{
			name: "valid genesis with commit-reveal enabled",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.Params.EnableCommitReveal = true
				gs.Params.CommitRevealDelay = 10
				gs.Params.CommitTimeoutBlocks = 100
				return gs
			},
			wantErr: false,
		},
		{
			name: "valid genesis with TWAP records",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.PoolTwapRecords = []PoolTWAP{
					{PoolId: 1},
					{PoolId: 2},
				}
				return gs
			},
			wantErr: false,
		},
		{
			name: "valid genesis with swap commits",
			genesis: func() *GenesisState {
				gs := DefaultGenesis()
				gs.SwapCommits = []SwapCommit{
					{
						Trader:       "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
						SwapHash:     "abc123",
						CommitHeight: 100,
						ExpiryHeight: 200,
					},
				}
				return gs
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := tt.genesis()
			err := gs.Validate()

			if (err != nil) != tt.wantErr {
				t.Errorf("GenesisState.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("GenesisState.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}
