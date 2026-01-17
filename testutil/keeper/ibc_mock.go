package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
)

// MockChannelKeeper implements only SendPacket for unit tests.
type MockChannelKeeper struct {
	NextSeq uint64
	Sent    int
}

func (m *MockChannelKeeper) SendPacket(
	_ sdk.Context,
	_ *capabilitytypes.Capability,
	_ string,
	_ string,
	_ clienttypes.Height,
	_ uint64,
	_ []byte,
) (uint64, error) {
	m.Sent++
	m.NextSeq++
	return m.NextSeq, nil
}
