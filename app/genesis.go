package app

import (
	"encoding/json"
	"time"

	"cosmossdk.io/math"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// GenesisState represents the genesis state of the PAW blockchain
type GenesisState map[string]json.RawMessage

// NewDefaultGenesisState generates the default genesis state from node-config.yaml parameters
func NewDefaultGenesisState(chainID string) GenesisState {
	genesis := make(GenesisState)

	// Auth module - account authentication
	authGenesis := authtypes.DefaultGenesisState()
	genesis[authtypes.ModuleName] = mustMarshalJSON(authGenesis)

	// Bank module - token balances and transfers
	bankGenesis := banktypes.DefaultGenesisState()
	bankGenesis.Params = banktypes.Params{
		SendEnabled:        []*banktypes.SendEnabled{},
		DefaultSendEnabled: true,
	}
	bankGenesis.Supply = sdk.NewCoins(
		sdk.NewInt64Coin("upaw", 50000000000000), // 50M PAW total supply
	)
	genesis[banktypes.ModuleName] = mustMarshalJSON(bankGenesis)

	// Staking module - validator and delegation management
	stakingGenesis := stakingtypes.DefaultGenesisState()
	stakingGenesis.Params = stakingtypes.Params{
		UnbondingTime:     time.Duration(1814400) * time.Second, // 21 days
		MaxValidators:     125,                                  // From node-config.yaml
		MaxEntries:        7,
		HistoricalEntries: 10000,
		BondDenom:         "upaw",
		MinCommissionRate: math.LegacyMustNewDecFromStr("0.05"), // 5% minimum commission
	}
	genesis[stakingtypes.ModuleName] = mustMarshalJSON(stakingGenesis)

	// Slashing module - validator punishment
	slashingGenesis := slashingtypes.DefaultGenesisState()
	slashingGenesis.Params = slashingtypes.Params{
		SignedBlocksWindow:      10000,                              // Blocks to track for downtime
		MinSignedPerWindow:      math.LegacyMustNewDecFromStr("0.50"),      // 50% minimum uptime
		DowntimeJailDuration:    time.Duration(86400) * time.Second, // 24 hours jail
		SlashFractionDoubleSign: math.LegacyMustNewDecFromStr("0.05"),      // 5% slash for double signing
		SlashFractionDowntime:   math.LegacyMustNewDecFromStr("0.001"),     // 0.1% slash for downtime
	}
	genesis[slashingtypes.ModuleName] = mustMarshalJSON(slashingGenesis)

	// Governance module - on-chain governance
	govGenesis := govtypes.DefaultGenesisState()
	govGenesis.Params = &govtypes.Params{
		MinDeposit:                 sdk.NewCoins(sdk.NewInt64Coin("upaw", 10000000000)), // 10,000 PAW
		MaxDepositPeriod:           durationPtr(time.Duration(604800) * time.Second),    // 7 days
		VotingPeriod:               durationPtr(time.Duration(1209600) * time.Second),   // 14 days
		Quorum:                     "0.400000000000000000",                              // 40% quorum
		Threshold:                  "0.667000000000000000",                              // 66.7% threshold
		VetoThreshold:              "0.333000000000000000",                              // 33.3% veto
		MinInitialDepositRatio:     "0.100000000000000000",                              // 10% initial deposit
		BurnVoteQuorum:             false,
		BurnProposalDepositPrevote: false,
		BurnVoteVeto:               false,
	}
	genesis["gov"] = mustMarshalJSON(govGenesis)

	// Distribution module - fee distribution
	distrGenesis := distrtypes.DefaultGenesisState()
	distrGenesis.Params = distrtypes.Params{
		CommunityTax:        math.LegacyMustNewDecFromStr("0.20"), // 20% to treasury
		BaseProposerReward:  math.LegacyZeroDec(),                 // Deprecated
		BonusProposerReward: math.LegacyZeroDec(),                 // Deprecated
		WithdrawAddrEnabled: true,
	}
	genesis[distrtypes.ModuleName] = mustMarshalJSON(distrGenesis)

	// Mint module - token emission (disabled, using fixed supply)
	mintGenesis := minttypes.DefaultGenesisState()
	mintGenesis.Params = minttypes.Params{
		MintDenom:           "upaw",
		InflationRateChange: math.LegacyMustNewDecFromStr("0.00"), // No inflation
		InflationMax:        math.LegacyMustNewDecFromStr("0.00"),
		InflationMin:        math.LegacyMustNewDecFromStr("0.00"),
		GoalBonded:          math.LegacyMustNewDecFromStr("0.67"),
		BlocksPerYear:       uint64(7884000), // ~4 second blocks
	}
	mintGenesis.Minter = minttypes.Minter{
		Inflation:        math.LegacyZeroDec(),
		AnnualProvisions: math.LegacyZeroDec(),
	}
	genesis[minttypes.ModuleName] = mustMarshalJSON(mintGenesis)

	// Crisis module - invariant checking
	crisisGenesis := crisistypes.DefaultGenesisState()
	crisisGenesis.ConstantFee = sdk.NewInt64Coin("upaw", 1000000000) // 1,000 PAW
	genesis[crisistypes.ModuleName] = mustMarshalJSON(crisisGenesis)

	// Wasm module - CosmWasm smart contracts
	wasmGenesis := wasmtypes.GenesisFixture()
	wasmGenesis.Params = wasmtypes.Params{
		CodeUploadAccess: wasmtypes.AccessConfig{
			Permission: wasmtypes.AccessTypeEverybody,
		},
		InstantiateDefaultPermission: wasmtypes.AccessTypeEverybody,
	}
	genesis[wasmtypes.ModuleName] = mustMarshalJSON(wasmGenesis)

	return genesis
}

// NewGenesisStateFromConfig creates genesis state with custom parameters
func NewGenesisStateFromConfig(config GenesisConfig) GenesisState {
	genesis := NewDefaultGenesisState(config.ChainID)

	// Override staking params
	var stakingGenesis stakingtypes.GenesisState
	mustUnmarshalJSON(genesis[stakingtypes.ModuleName], &stakingGenesis)
	stakingGenesis.Params.MaxValidators = config.MaxValidators
	stakingGenesis.Params.UnbondingTime = time.Duration(config.UnbondingPeriodSeconds) * time.Second
	genesis[stakingtypes.ModuleName] = mustMarshalJSON(stakingGenesis)

	// Override slashing params
	var slashingGenesis slashingtypes.GenesisState
	mustUnmarshalJSON(genesis[slashingtypes.ModuleName], &slashingGenesis)
	slashingGenesis.Params.SlashFractionDoubleSign = math.LegacyMustNewDecFromStr(config.DoubleSignPenalty)
	slashingGenesis.Params.SlashFractionDowntime = math.LegacyMustNewDecFromStr(config.DowntimePenalty)
	slashingGenesis.Params.SignedBlocksWindow = int64(config.DowntimeWindowBlocks)
	slashingGenesis.Params.DowntimeJailDuration = time.Duration(config.DowntimeJailDurationSeconds) * time.Second
	genesis[slashingtypes.ModuleName] = mustMarshalJSON(slashingGenesis)

	// Override governance params
	var govGenesis govtypes.GenesisState
	mustUnmarshalJSON(genesis["gov"], &govGenesis)
	govGenesis.Params.MinDeposit = sdk.NewCoins(sdk.NewInt64Coin("upaw", config.MinDepositAmount))
	govGenesis.Params.VotingPeriod = durationPtr(time.Duration(config.VotingPeriodSeconds) * time.Second)
	govGenesis.Params.Quorum = config.Quorum
	govGenesis.Params.Threshold = config.Threshold
	govGenesis.Params.VetoThreshold = config.VetoThreshold
	genesis["gov"] = mustMarshalJSON(govGenesis)

	// Override bank supply
	var bankGenesis banktypes.GenesisState
	mustUnmarshalJSON(genesis[banktypes.ModuleName], &bankGenesis)
	bankGenesis.Supply = sdk.NewCoins(sdk.NewInt64Coin("upaw", config.TotalSupply))
	genesis[banktypes.ModuleName] = mustMarshalJSON(bankGenesis)

	return genesis
}

// GenesisConfig holds configuration parameters for genesis state
type GenesisConfig struct {
	ChainID                     string
	TotalSupply                 int64
	MaxValidators               uint32
	UnbondingPeriodSeconds      int64
	DoubleSignPenalty           string
	DowntimePenalty             string
	DowntimeWindowBlocks        uint64
	DowntimeJailDurationSeconds int64
	MinDepositAmount            int64
	VotingPeriodSeconds         int64
	Quorum                      string
	Threshold                   string
	VetoThreshold               string
}

// DefaultGenesisConfig returns default configuration from node-config.yaml
func DefaultGenesisConfig() GenesisConfig {
	return GenesisConfig{
		ChainID:                     "paw-testnet",
		TotalSupply:                 50000000000000, // 50M PAW
		MaxValidators:               125,
		UnbondingPeriodSeconds:      1814400, // 21 days
		DoubleSignPenalty:           "0.05",  // 5%
		DowntimePenalty:             "0.001", // 0.1%
		DowntimeWindowBlocks:        10000,
		DowntimeJailDurationSeconds: 86400,                  // 24 hours
		MinDepositAmount:            10000000000,            // 10,000 PAW
		VotingPeriodSeconds:         1209600,                // 14 days
		Quorum:                      "0.400000000000000000", // 40%
		Threshold:                   "0.667000000000000000", // 66.7%
		VetoThreshold:               "0.333000000000000000", // 33.3%
	}
}

// Helper functions
func mustMarshalJSON(v interface{}) json.RawMessage {
	bz, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return bz
}

func mustUnmarshalJSON(bz []byte, v interface{}) {
	if err := json.Unmarshal(bz, v); err != nil {
		panic(err)
	}
}

func durationPtr(d time.Duration) *time.Duration {
	return &d
}
