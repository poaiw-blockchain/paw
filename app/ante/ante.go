package ante

import (
	"fmt"

	feegrantkeeper "cosmossdk.io/x/feegrant/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
)

// HandlerOptions are the options required for constructing a default SDK AnteHandler.
type HandlerOptions struct {
	AccountKeeper   authkeeper.AccountKeeper
	BankKeeper      bankkeeper.Keeper
	FeegrantKeeper  feegrantkeeper.Keeper
	SignModeHandler *signing.HandlerMap
	SigGasConsumer  func(meter sdk.GasMeter, sig signing.SignatureV2, params authtypes.Params) error
	IBCKeeper       *ibckeeper.Keeper
	ComputeKeeper   *computekeeper.Keeper
	DEXKeeper       *dexkeeper.Keeper
	OracleKeeper    *oraclekeeper.Keeper
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer. It also includes custom decorators for PAW modules.
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {
	if options.AccountKeeper == nil {
		return nil, fmt.Errorf("account keeper is required for ante builder")
	}

	if options.BankKeeper == nil {
		return nil, fmt.Errorf("bank keeper is required for ante builder")
	}

	if options.SignModeHandler == nil {
		return nil, fmt.Errorf("sign mode handler is required for ante builder")
	}

	anteDecorators := []sdk.AnteDecorator{
		ante.NewSetUpContextDecorator(), // outermost AnteDecorator. SetUpContext must be called first
		ante.NewExtensionOptionsDecorator(nil),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, nil),
		ante.NewSetPubKeyDecorator(options.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	}

	// Add custom module decorators
	if options.ComputeKeeper != nil {
		anteDecorators = append(anteDecorators, NewComputeDecorator(*options.ComputeKeeper))
	}

	if options.DEXKeeper != nil {
		anteDecorators = append(anteDecorators, NewDEXDecorator(*options.DEXKeeper))
	}

	if options.OracleKeeper != nil {
		anteDecorators = append(anteDecorators, NewOracleDecorator(*options.OracleKeeper))
	}

	return sdk.ChainAnteDecorators(anteDecorators...), nil
}
