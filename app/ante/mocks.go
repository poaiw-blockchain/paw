package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	proto "google.golang.org/protobuf/proto"
)

// mockMemoTx is a minimal tx implementing sdk.TxWithMemo for testing memo limits.
type mockMemoTx struct {
	memo string
}

func (m mockMemoTx) GetMsgs() []sdk.Msg                  { return nil }
func (m mockMemoTx) GetMsgsV2() ([]proto.Message, error) { return nil, nil }
func (m mockMemoTx) ValidateBasic() error                { return nil }
func (m mockMemoTx) GetMemo() string                     { return m.memo }
