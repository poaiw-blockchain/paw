package types

import proto "github.com/cosmos/gogoproto/proto"

// String satisfies proto.Message for Params because gogo's generated code omits it.
func (m *Params) String() string {
	if m == nil {
		return ""
	}
	return proto.CompactTextString(m)
}
