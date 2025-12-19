package ante_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	protov2 "google.golang.org/protobuf/proto"

	"github.com/paw-chain/paw/app/ante"
)

type mockMsg struct {
	fail bool
}

func (m mockMsg) Reset()         {}
func (m mockMsg) String() string { return "mockMsg" }
func (m mockMsg) ProtoMessage()  {}
func (m mockMsg) GetSigners() []sdk.AccAddress {
	return nil
}

func (m mockMsg) ValidateBasic() error {
	if m.fail {
		return fmt.Errorf("validation failed")
	}
	return nil
}

type mockTx struct {
	msgs []sdk.Msg
}

func (m mockTx) GetMsgs() []sdk.Msg { return m.msgs }

func (m mockTx) GetMsgsV2() ([]protov2.Message, error) {
	return nil, nil
}

func TestGasLimitDecorator_AllowsValidTx(t *testing.T) {
	t.Parallel()

	ctx := sdk.Context{}.WithGasMeter(storetypes.NewGasMeter(ante.MaxGasPerTx))
	tx := mockTx{msgs: []sdk.Msg{mockMsg{}}}

	dec := ante.NewGasLimitDecorator()
	_, err := dec.AnteHandle(ctx, tx, false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	})
	require.NoError(t, err)
}

func TestGasLimitDecorator_MessageCountExceeded(t *testing.T) {
	t.Parallel()

	ctx := sdk.Context{}.WithGasMeter(storetypes.NewGasMeter(ante.MaxGasPerTx))
	var msgs []sdk.Msg
	for i := 0; i < ante.MaxMessagesPerTx+1; i++ {
		msgs = append(msgs, mockMsg{})
	}

	tx := mockTx{msgs: msgs}
	dec := ante.NewGasLimitDecorator()
	_, err := dec.AnteHandle(ctx, tx, false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "too many messages")
}

func TestGasLimitDecorator_ValidateBasicFailure(t *testing.T) {
	t.Parallel()

	ctx := sdk.Context{}.WithGasMeter(storetypes.NewGasMeter(ante.MaxGasPerTx))
	tx := mockTx{msgs: []sdk.Msg{mockMsg{fail: true}}}

	dec := ante.NewGasLimitDecorator()
	_, err := dec.AnteHandle(ctx, tx, false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "message validation failed")
}

func TestGasLimitDecorator_MaxGasExceeded(t *testing.T) {
	t.Parallel()

	ctx := sdk.Context{}.WithGasMeter(storetypes.NewGasMeter(ante.MaxGasPerTx + 1))
	tx := mockTx{msgs: []sdk.Msg{mockMsg{}}}

	dec := ante.NewGasLimitDecorator()
	_, err := dec.AnteHandle(ctx, tx, false, func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
		return ctx, nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "transaction gas limit too high")
}

func TestConsumeGasForOperation(t *testing.T) {
	t.Parallel()

	ctx := sdk.Context{}.WithGasMeter(storetypes.NewGasMeter(ante.MaxGasPerMessage))

	err := ante.ConsumeGasForOperation(ctx, ante.MaxGasPerMessage-1, "test_op", ante.MaxGasPerMessage)
	require.NoError(t, err)

	err = ante.ConsumeGasForOperation(ctx, ante.MaxGasPerMessage+1, "test_op", ante.MaxGasPerMessage)
	require.Error(t, err)
	require.Contains(t, err.Error(), "too much gas")
}

func TestIterateWithGasLimit(t *testing.T) {
	t.Parallel()

	ctx := sdk.Context{}.WithGasMeter(storetypes.NewGasMeter(ante.MaxGasPerTx))
	count := 0
	err := ante.IterateWithGasLimit(ctx, 5, 10, func(i int) (bool, error) {
		count++
		if i == 2 {
			return false, nil
		}
		return true, nil
	})
	require.NoError(t, err)
	require.Equal(t, 3, count)
}
