package simulation

import (
	"encoding/json"
	"math/rand"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"

	"github.com/paw-chain/paw/app"
)

// SimulationOperations reconstructs the weighted operation set using the app's simulation manager.
func SimulationOperations(simApp *app.PAWApp, cdc codec.JSONCodec, config simtypes.Config) []simtypes.WeightedOperation {
	simState := module.SimulationState{
		AppParams: make(simtypes.AppParams),
		Cdc:       cdc,
		TxConfig:  simApp.TxConfig(),
		Rand:      rand.New(rand.NewSource(config.Seed)),
		BondDenom: sdk.DefaultBondDenom,
	}

	if config.ParamsFile != "" {
		bz, err := os.ReadFile(config.ParamsFile)
		if err != nil {
			panic(err)
		}

		if err := json.Unmarshal(bz, &simState.AppParams); err != nil {
			panic(err)
		}
	}

	// Populate proposal generators for modules that expose governance hooks.
	//nolint:staticcheck // maintained for legacy param change support
	simState.LegacyProposalContents = simApp.SimulationManager().GetProposalContents(simState)
	simState.ProposalMsgs = simApp.SimulationManager().GetProposalMsgs(simState)

	return simApp.SimulationManager().WeightedOperations(simState)
}
