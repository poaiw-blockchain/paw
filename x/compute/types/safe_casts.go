package types

import (
	"math"
	"time"
)

const maxSecondsDuration = int64(math.MaxInt64 / int64(time.Second))

// SaturateUint64ToInt64 bounds a uint64 before casting to int64 to avoid overflow.
func SaturateUint64ToInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		return math.MaxInt64
	}
	return int64(v)
}

// SaturateUint64ToUint32 bounds a uint64 before casting to uint32 to avoid overflow.
func SaturateUint64ToUint32(v uint64) uint32 {
	if v > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(v)
}

// SaturateInt64ToUint64 bounds an int64 before casting to uint64, treating negatives as zero.
func SaturateInt64ToUint64(v int64) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}

// SaturateInt64ToUint32 bounds an int64 before casting to uint32, treating negatives as zero.
func SaturateInt64ToUint32(v int64) uint32 {
	if v < 0 {
		return 0
	}
	if v > int64(math.MaxUint32) {
		return math.MaxUint32
	}
	return uint32(v)
}

// SaturateInt32ToUint32 bounds an int32 before casting to uint32, treating negatives as zero.
func SaturateInt32ToUint32(v int32) uint32 {
	if v < 0 {
		return 0
	}
	return uint32(v)
}

// SaturateIntToUint64 bounds an int before casting to uint64, treating negatives as zero.
func SaturateIntToUint64(v int) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}

// SaturateIntToUint32 bounds an int before casting to uint32, treating negatives as zero.
func SaturateIntToUint32(v int) uint32 {
	if v < 0 {
		return 0
	}
	if v > int(math.MaxUint32) {
		return math.MaxUint32
	}
	return uint32(v)
}

// SecondsToDuration safely converts seconds into a time.Duration, clamping to the maximum representable duration.
func SecondsToDuration(seconds uint64) time.Duration {
	sec := SaturateUint64ToInt64(seconds)
	if sec > maxSecondsDuration {
		sec = maxSecondsDuration
	}
	return time.Duration(sec) * time.Second
}
