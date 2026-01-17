package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// WithSendPacketStub sets a test-only sendPacketFn that returns the provided sequence
// and records the last payload seen. Intended for unit tests that need a happy-path
// IBC send without wiring the full IBC stack.
func (k *Keeper) WithSendPacketStub(seq uint64, recorder func(connectionID, channelID string, data []byte)) {
	k.sendPacketFn = func(ctx sdk.Context, connectionID, channelID string, data []byte, _ time.Duration) (uint64, error) {
		if recorder != nil {
			recorder(connectionID, channelID, data)
		}
		return seq, nil
	}
}
