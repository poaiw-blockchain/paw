package fuzz

import (
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"

	computetypes "github.com/paw-chain/paw/x/compute/types"
	oracletype "github.com/paw-chain/paw/x/oracle/types"
)

// Ensure oracle asset length cap is enforced in ValidateBasic.
func TestOracleAssetLengthLimit(t *testing.T) {
	msg := oracletype.MsgSubmitPrice{
		Validator: "cosmosvaloper1zg69v7ys40x77y352eufp27daufrg4nckx5hjn",
		Feeder:    "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		Asset:     strings.Repeat("A", 129),
		Price:     sdkmath.LegacyOneDec(),
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Fatalf("expected oversized asset to fail validation")
	}
}

// Ensure compute evidence description length cap is enforced in ValidateBasic.
func TestComputeEvidenceDescriptionLengthLimit(t *testing.T) {
	msg := computetypes.MsgSubmitEvidence{
		Submitter:    "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		DisputeId:    1,
		Data:         []byte{0x01},
		EvidenceType: "log",
		Description:  strings.Repeat("a", 2048+1),
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Fatalf("expected oversized evidence description to fail validation")
	}
}
