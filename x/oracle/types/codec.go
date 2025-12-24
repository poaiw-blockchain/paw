package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// RegisterLegacyAminoCodec registers the necessary x/oracle interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgSubmitPrice{}, "paw/oracle/MsgSubmitPrice", nil)
	cdc.RegisterConcrete(&MsgDelegateFeedConsent{}, "paw/oracle/MsgDelegateFeedConsent", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "paw/oracle/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgEmergencyPauseOracle{}, "paw/oracle/MsgEmergencyPauseOracle", nil)
	cdc.RegisterConcrete(&MsgResumeOracle{}, "paw/oracle/MsgResumeOracle", nil)
}

// RegisterInterfaces registers the x/oracle interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitPrice{},
		&MsgDelegateFeedConsent{},
		&MsgUpdateParams{},
		&MsgEmergencyPauseOracle{},
		&MsgResumeOracle{},
	)

	registry.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&MsgSubmitPriceResponse{},
		&MsgDelegateFeedConsentResponse{},
		&MsgUpdateParamsResponse{},
		&MsgEmergencyPauseOracleResponse{},
		&MsgResumeOracleResponse{},
	)
}

var (
	amino = codec.NewLegacyAmino()
	// ModuleCdc references the global x/oracle module codec
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
)

func init() {
	RegisterLegacyAminoCodec(amino)
	amino.Seal()
}
