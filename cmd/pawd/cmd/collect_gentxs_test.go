package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"

	"github.com/paw-chain/paw/app"
)

func TestMsgCreateValidatorToGenesisValidator(t *testing.T) {
	encodingConfig := app.MakeEncodingConfig()
	pk := ed25519.GenPrivKey().PubKey()

	valAddr := sdk.ValAddress(pk.Address())
	msg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		pk,
		sdk.NewCoin("upaw", sdkmath.NewInt(5_000_000)),
		stakingtypes.NewDescription("node1", "", "", "", ""),
		stakingtypes.NewCommissionRates(math.LegacyMustNewDecFromStr("0.10"), math.LegacyMustNewDecFromStr("0.20"), math.LegacyMustNewDecFromStr("0.01")),
		math.NewInt(1),
	)
	require.NoError(t, err)

	validator, err := msgCreateValidatorToGenesisValidator(encodingConfig.InterfaceRegistry, msg)
	require.NoError(t, err)
	require.Equal(t, valAddr, sdk.ValAddress(validator.Address))
	require.Equal(t, "node1", validator.Name)
	require.Equal(t, sdk.TokensToConsensusPower(msg.Value.Amount, sdk.DefaultPowerReduction), validator.Power)
}

func TestMsgCreateValidatorToGenesisValidatorZeroPower(t *testing.T) {
	encodingConfig := app.MakeEncodingConfig()
	pk := ed25519.GenPrivKey().PubKey()

	valAddr := sdk.ValAddress(pk.Address())
	msg, err := stakingtypes.NewMsgCreateValidator(
		valAddr.String(),
		pk,
		sdk.NewCoin("upaw", sdkmath.ZeroInt()),
		stakingtypes.NewDescription("node1", "", "", "", ""),
		stakingtypes.NewCommissionRates(math.LegacyMustNewDecFromStr("0.10"), math.LegacyMustNewDecFromStr("0.20"), math.LegacyMustNewDecFromStr("0.01")),
		math.NewInt(1),
	)
	require.NoError(t, err)

	_, err = msgCreateValidatorToGenesisValidator(encodingConfig.InterfaceRegistry, msg)
	require.Error(t, err)
}
