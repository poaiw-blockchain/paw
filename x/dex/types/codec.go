package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// RegisterLegacyAminoCodec registers the necessary x/dex interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreatePool{}, "paw/dex/MsgCreatePool", nil)
	cdc.RegisterConcrete(&MsgAddLiquidity{}, "paw/dex/MsgAddLiquidity", nil)
	cdc.RegisterConcrete(&MsgRemoveLiquidity{}, "paw/dex/MsgRemoveLiquidity", nil)
	cdc.RegisterConcrete(&MsgSwap{}, "paw/dex/MsgSwap", nil)
	cdc.RegisterConcrete(&MsgCommitSwap{}, "paw/dex/MsgCommitSwap", nil)
	cdc.RegisterConcrete(&MsgRevealSwap{}, "paw/dex/MsgRevealSwap", nil)
	cdc.RegisterConcrete(&MsgBatchSwap{}, "paw/dex/MsgBatchSwap", nil)
}

// RegisterInterfaces registers the x/dex interfaces types with the interface registry
func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCreatePool{},
		&MsgAddLiquidity{},
		&MsgRemoveLiquidity{},
		&MsgSwap{},
		&MsgCommitSwap{},
		&MsgRevealSwap{},
		&MsgBatchSwap{},
	)

	registry.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&MsgCreatePoolResponse{},
		&MsgAddLiquidityResponse{},
		&MsgRemoveLiquidityResponse{},
		&MsgSwapResponse{},
		&MsgCommitSwapResponse{},
		&MsgRevealSwapResponse{},
		&MsgBatchSwapResponse{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino = codec.NewLegacyAmino()
)

func init() {
	RegisterLegacyAminoCodec(amino)
}
