package ante

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMemoLimitDecorator(t *testing.T) {
	dec := NewMemoLimitDecorator(10)

	txExact := mockMemoTx{memo: "0123456789"}
	txOver := mockMemoTx{memo: "0123456789a"}

	ctx := sdk.Context{}.WithTxBytes([]byte{})
	ante := sdk.ChainAnteDecorators(dec)

	// exact size passes
	_, err := ante(ctx, txExact, false)
	require.NoError(t, err)

	// oversize fails
	_, err = ante(ctx, txOver, false)
	require.Error(t, err)
}
