package keeper

import (
	"time"

	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// saturateUint64ToInt64 bounds a uint64 before casting to int64 to avoid overflow.
func saturateUint64ToInt64(v uint64) int64 {
	return computetypes.SaturateUint64ToInt64(v)
}

// saturateUint64ToUint32 bounds a uint64 before casting to uint32 to avoid overflow.
func saturateUint64ToUint32(v uint64) uint32 {
	return computetypes.SaturateUint64ToUint32(v)
}

// saturateInt64ToUint64 bounds an int64 before casting to uint64, treating negatives as zero.
func saturateInt64ToUint64(v int64) uint64 {
	return computetypes.SaturateInt64ToUint64(v)
}

// saturateInt64ToUint32 bounds an int64 before casting to uint32, treating negatives as zero.
func saturateInt64ToUint32(v int64) uint32 {
	return computetypes.SaturateInt64ToUint32(v)
}

// secondsToDuration safely converts seconds into a time.Duration (seconds), clamping to the maximum representable duration.
func secondsToDuration(seconds uint64) time.Duration {
	return computetypes.SecondsToDuration(seconds)
}
