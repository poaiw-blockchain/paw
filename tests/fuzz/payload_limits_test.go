package fuzz

import (
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"

	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// Ensure oversize dispute reasons are rejected to bound proposal payloads.
func TestDisputeReasonLengthLimit(t *testing.T) {
	msg := computetypes.MsgCreateDispute{
		Requester:     "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q",
		RequestId:     1,
		Reason:        strings.Repeat("a", 1024+10),
		DepositAmount: sdkmath.NewInt(1),
	}

	if err := msg.ValidateBasic(); err == nil {
		t.Fatalf("expected oversized reason to fail validation")
	}
}
