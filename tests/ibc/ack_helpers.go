package ibc_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// ackFromBytes unmarshals acknowledgement JSON into a channeltypes.Acknowledgement.
func ackFromBytes(t testing.TB, ackBz []byte) channeltypes.Acknowledgement {
	t.Helper()

	var ack channeltypes.Acknowledgement
	require.NoError(t, channeltypes.SubModuleCdc.UnmarshalJSON(ackBz, &ack))
	return ack
}

// ackResult extracts the result bytes from a successful acknowledgement and fails the test on errors.
func ackResult(t testing.TB, ackBz []byte) []byte {
	t.Helper()

	ack := ackFromBytes(t, ackBz)
	switch resp := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Result:
		return resp.Result
	case *channeltypes.Acknowledgement_Error:
		require.FailNowf(t, "unexpected error acknowledgement", resp.Error)
	default:
		require.FailNowf(t, "unknown acknowledgement type", "%T", resp)
	}

	return nil
}

// ackError extracts the error string from a failed acknowledgement and fails the test on success.
func ackError(t testing.TB, ackBz []byte) string {
	t.Helper()

	ack := ackFromBytes(t, ackBz)
	switch resp := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:
		return resp.Error
	case *channeltypes.Acknowledgement_Result:
		require.FailNow(t, "expected error acknowledgement, got result")
	default:
		require.FailNowf(t, "unknown acknowledgement type", "%T", resp)
	}

	return ""
}
