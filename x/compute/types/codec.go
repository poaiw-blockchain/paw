package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
)

// RegisterLegacyAminoCodec registers the necessary x/compute interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgRegisterProvider{}, "paw/compute/MsgRegisterProvider", nil)
	cdc.RegisterConcrete(&MsgUpdateProvider{}, "paw/compute/MsgUpdateProvider", nil)
	cdc.RegisterConcrete(&MsgDeactivateProvider{}, "paw/compute/MsgDeactivateProvider", nil)
	cdc.RegisterConcrete(&MsgSubmitRequest{}, "paw/compute/MsgSubmitRequest", nil)
	cdc.RegisterConcrete(&MsgCancelRequest{}, "paw/compute/MsgCancelRequest", nil)
	cdc.RegisterConcrete(&MsgSubmitResult{}, "paw/compute/MsgSubmitResult", nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, "paw/compute/MsgUpdateParams", nil)
	cdc.RegisterConcrete(&MsgCreateDispute{}, "paw/compute/MsgCreateDispute", nil)
	cdc.RegisterConcrete(&MsgVoteOnDispute{}, "paw/compute/MsgVoteOnDispute", nil)
	cdc.RegisterConcrete(&MsgResolveDispute{}, "paw/compute/MsgResolveDispute", nil)
	cdc.RegisterConcrete(&MsgSubmitEvidence{}, "paw/compute/MsgSubmitEvidence", nil)
	cdc.RegisterConcrete(&MsgAppealSlashing{}, "paw/compute/MsgAppealSlashing", nil)
	cdc.RegisterConcrete(&MsgVoteOnAppeal{}, "paw/compute/MsgVoteOnAppeal", nil)
	cdc.RegisterConcrete(&MsgResolveAppeal{}, "paw/compute/MsgResolveAppeal", nil)
	cdc.RegisterConcrete(&MsgUpdateGovernanceParams{}, "paw/compute/MsgUpdateGovernanceParams", nil)
}

// RegisterInterfaces registers the x/compute interfaces types with the interface registry
func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterProvider{},
		&MsgUpdateProvider{},
		&MsgDeactivateProvider{},
		&MsgSubmitRequest{},
		&MsgCancelRequest{},
		&MsgSubmitResult{},
		&MsgUpdateParams{},
		&MsgCreateDispute{},
		&MsgVoteOnDispute{},
		&MsgResolveDispute{},
		&MsgSubmitEvidence{},
		&MsgAppealSlashing{},
		&MsgVoteOnAppeal{},
		&MsgResolveAppeal{},
		&MsgUpdateGovernanceParams{},
	)

	registry.RegisterImplementations((*txtypes.MsgResponse)(nil),
		&MsgRegisterProviderResponse{},
		&MsgUpdateProviderResponse{},
		&MsgDeactivateProviderResponse{},
		&MsgSubmitRequestResponse{},
		&MsgCancelRequestResponse{},
		&MsgSubmitResultResponse{},
		&MsgUpdateParamsResponse{},
		&MsgCreateDisputeResponse{},
		&MsgVoteOnDisputeResponse{},
		&MsgResolveDisputeResponse{},
		&MsgSubmitEvidenceResponse{},
		&MsgAppealSlashingResponse{},
		&MsgVoteOnAppealResponse{},
		&MsgResolveAppealResponse{},
		&MsgUpdateGovernanceParamsResponse{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino = codec.NewLegacyAmino()
)

func init() {
	RegisterLegacyAminoCodec(amino)
}
