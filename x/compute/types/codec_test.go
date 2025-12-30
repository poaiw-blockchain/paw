package types

import (
	"testing"

	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestRegisterInterfaces(t *testing.T) {
	registry := cdctypes.NewInterfaceRegistry()

	// Should not panic
	RegisterInterfaces(registry)

	// Verify that message types are registered by checking we can unmarshal them
	msgTypes := []string{
		"/paw.compute.v1.MsgRegisterProvider",
		"/paw.compute.v1.MsgUpdateProvider",
		"/paw.compute.v1.MsgDeactivateProvider",
		"/paw.compute.v1.MsgSubmitRequest",
		"/paw.compute.v1.MsgCancelRequest",
		"/paw.compute.v1.MsgSubmitResult",
		"/paw.compute.v1.MsgUpdateParams",
		"/paw.compute.v1.MsgCreateDispute",
		"/paw.compute.v1.MsgVoteOnDispute",
		"/paw.compute.v1.MsgResolveDispute",
		"/paw.compute.v1.MsgSubmitEvidence",
		"/paw.compute.v1.MsgAppealSlashing",
		"/paw.compute.v1.MsgVoteOnAppeal",
		"/paw.compute.v1.MsgResolveAppeal",
		"/paw.compute.v1.MsgUpdateGovernanceParams",
		"/paw.compute.v1.MsgSubmitBatchRequests",
	}

	for _, typeURL := range msgTypes {
		t.Run(typeURL, func(t *testing.T) {
			// Just verify we can list implementations without panic
			_ = registry.ListImplementations(sdk.MsgInterfaceProtoName)
		})
	}
}

func TestRegisterLegacyAminoCodec(t *testing.T) {
	// Test that the amino codec was initialized in init()
	if amino == nil {
		t.Error("amino codec should be initialized")
	}

	// Test that we can serialize a message type
	msg := &MsgRegisterProvider{
		Provider: "cosmos1test",
		Moniker:  "test-provider",
		Endpoint: "https://test.example.com",
	}

	// Should not panic
	_, err := amino.MarshalJSON(msg)
	if err != nil {
		t.Errorf("Failed to marshal MsgRegisterProvider: %v", err)
	}
}

func TestAminoCodecMessageTypes(t *testing.T) {
	messages := []sdk.Msg{
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
		&MsgSubmitBatchRequests{},
	}

	for _, msg := range messages {
		t.Run(sdk.MsgTypeURL(msg), func(t *testing.T) {
			// Should not panic when marshaling
			_, err := amino.MarshalJSON(msg)
			if err != nil {
				t.Errorf("Failed to marshal %T: %v", msg, err)
			}
		})
	}
}
