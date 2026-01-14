package app

import (
	"encoding/json"
	"fmt"
	stdmath "math"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// GenesisState represents the genesis state of the PAW blockchain
type GenesisState map[string]json.RawMessage

// NewDefaultGenesisState generates the default genesis state from node-config.yaml parameters
func NewDefaultGenesisState(chainID string) GenesisState {
	// Start with all module defaults
	cdc := MakeEncodingConfig().Codec
	genesis := ModuleBasics.DefaultGenesis(cdc)

	// All module defaults are already set via ModuleBasics.DefaultGenesis(cdc)
	// We can customize specific parameters here if needed in the future

	return genesis
}

// NewGenesisStateFromConfig creates genesis state with custom parameters.
func NewGenesisStateFromConfig(config *GenesisConfig) GenesisState {
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
	if config.DowntimeWindowBlocks > uint64(stdmath.MaxInt64) {
		panic(fmt.Sprintf("downtime window blocks %d exceed int64 range", config.DowntimeWindowBlocks))
	}
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
		ChainID:                     "paw-mvp-1",
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
