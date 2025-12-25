package types

import (
	"math"
	"testing"
	"time"
)

func TestSaturateUint64ToInt64(t *testing.T) {
	tests := []struct {
		name  string
		input uint64
		want  int64
	}{
		{
			name:  "zero",
			input: 0,
			want:  0,
		},
		{
			name:  "small positive value",
			input: 12345,
			want:  12345,
		},
		{
			name:  "max int64",
			input: math.MaxInt64,
			want:  math.MaxInt64,
		},
		{
			name:  "max int64 + 1 (overflow)",
			input: math.MaxInt64 + 1,
			want:  math.MaxInt64,
		},
		{
			name:  "max uint64 (overflow)",
			input: math.MaxUint64,
			want:  math.MaxInt64,
		},
		{
			name:  "value just below max int64",
			input: math.MaxInt64 - 1,
			want:  math.MaxInt64 - 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SaturateUint64ToInt64(tt.input)
			if got != tt.want {
				t.Errorf("SaturateUint64ToInt64(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSaturateUint64ToUint32(t *testing.T) {
	tests := []struct {
		name  string
		input uint64
		want  uint32
	}{
		{
			name:  "zero",
			input: 0,
			want:  0,
		},
		{
			name:  "small value",
			input: 12345,
			want:  12345,
		},
		{
			name:  "max uint32",
			input: math.MaxUint32,
			want:  math.MaxUint32,
		},
		{
			name:  "max uint32 + 1 (overflow)",
			input: math.MaxUint32 + 1,
			want:  math.MaxUint32,
		},
		{
			name:  "max uint64 (overflow)",
			input: math.MaxUint64,
			want:  math.MaxUint32,
		},
		{
			name:  "value just below max uint32",
			input: math.MaxUint32 - 1,
			want:  math.MaxUint32 - 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SaturateUint64ToUint32(tt.input)
			if got != tt.want {
				t.Errorf("SaturateUint64ToUint32(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSaturateInt64ToUint64(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  uint64
	}{
		{
			name:  "zero",
			input: 0,
			want:  0,
		},
		{
			name:  "positive value",
			input: 12345,
			want:  12345,
		},
		{
			name:  "max int64",
			input: math.MaxInt64,
			want:  math.MaxInt64,
		},
		{
			name:  "negative value -1",
			input: -1,
			want:  0,
		},
		{
			name:  "negative value -12345",
			input: -12345,
			want:  0,
		},
		{
			name:  "min int64",
			input: math.MinInt64,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SaturateInt64ToUint64(tt.input)
			if got != tt.want {
				t.Errorf("SaturateInt64ToUint64(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSaturateInt64ToUint32(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  uint32
	}{
		{
			name:  "zero",
			input: 0,
			want:  0,
		},
		{
			name:  "positive value",
			input: 12345,
			want:  12345,
		},
		{
			name:  "max uint32 as int64",
			input: int64(math.MaxUint32),
			want:  math.MaxUint32,
		},
		{
			name:  "value exceeding max uint32",
			input: int64(math.MaxUint32) + 1,
			want:  math.MaxUint32,
		},
		{
			name:  "max int64 (overflow)",
			input: math.MaxInt64,
			want:  math.MaxUint32,
		},
		{
			name:  "negative value -1",
			input: -1,
			want:  0,
		},
		{
			name:  "negative value -12345",
			input: -12345,
			want:  0,
		},
		{
			name:  "min int64",
			input: math.MinInt64,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SaturateInt64ToUint32(tt.input)
			if got != tt.want {
				t.Errorf("SaturateInt64ToUint32(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSaturateInt32ToUint32(t *testing.T) {
	tests := []struct {
		name  string
		input int32
		want  uint32
	}{
		{
			name:  "zero",
			input: 0,
			want:  0,
		},
		{
			name:  "positive value",
			input: 12345,
			want:  12345,
		},
		{
			name:  "max int32",
			input: math.MaxInt32,
			want:  math.MaxInt32,
		},
		{
			name:  "negative value -1",
			input: -1,
			want:  0,
		},
		{
			name:  "negative value -12345",
			input: -12345,
			want:  0,
		},
		{
			name:  "min int32",
			input: math.MinInt32,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SaturateInt32ToUint32(tt.input)
			if got != tt.want {
				t.Errorf("SaturateInt32ToUint32(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSaturateIntToUint64(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  uint64
	}{
		{
			name:  "zero",
			input: 0,
			want:  0,
		},
		{
			name:  "positive value",
			input: 12345,
			want:  12345,
		},
		{
			name:  "large positive value",
			input: math.MaxInt32,
			want:  math.MaxInt32,
		},
		{
			name:  "negative value -1",
			input: -1,
			want:  0,
		},
		{
			name:  "negative value -12345",
			input: -12345,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SaturateIntToUint64(tt.input)
			if got != tt.want {
				t.Errorf("SaturateIntToUint64(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSaturateIntToUint32(t *testing.T) {
	tests := []struct {
		name  string
		input int
		want  uint32
	}{
		{
			name:  "zero",
			input: 0,
			want:  0,
		},
		{
			name:  "positive value",
			input: 12345,
			want:  12345,
		},
		{
			name:  "max uint32",
			input: int(math.MaxUint32),
			want:  math.MaxUint32,
		},
		{
			name:  "negative value -1",
			input: -1,
			want:  0,
		},
		{
			name:  "negative value -12345",
			input: -12345,
			want:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SaturateIntToUint32(tt.input)
			if got != tt.want {
				t.Errorf("SaturateIntToUint32(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSecondsToDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds uint64
		want    time.Duration
	}{
		{
			name:    "zero seconds",
			seconds: 0,
			want:    0,
		},
		{
			name:    "one second",
			seconds: 1,
			want:    time.Second,
		},
		{
			name:    "one minute",
			seconds: 60,
			want:    60 * time.Second,
		},
		{
			name:    "one hour",
			seconds: 3600,
			want:    3600 * time.Second,
		},
		{
			name:    "one day",
			seconds: 86400,
			want:    86400 * time.Second,
		},
		{
			name:    "max safe duration in seconds",
			seconds: uint64(maxSecondsDuration),
			want:    time.Duration(maxSecondsDuration) * time.Second,
		},
		{
			name:    "value exceeding max safe duration (overflow protection)",
			seconds: uint64(maxSecondsDuration) + 1000,
			want:    time.Duration(maxSecondsDuration) * time.Second,
		},
		{
			name:    "very large value (overflow protection)",
			seconds: math.MaxUint64,
			want:    time.Duration(maxSecondsDuration) * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SecondsToDuration(tt.seconds)
			if got != tt.want {
				t.Errorf("SecondsToDuration(%v) = %v, want %v", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestSecondsToDuration_NoOverflow(t *testing.T) {
	// Verify that SecondsToDuration never panics, even with extreme values
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("SecondsToDuration panicked with extreme value: %v", r)
		}
	}()

	testValues := []uint64{
		0,
		1,
		math.MaxInt64,
		math.MaxUint64,
		uint64(maxSecondsDuration),
		uint64(maxSecondsDuration) + 1,
	}

	for _, val := range testValues {
		_ = SecondsToDuration(val)
	}
}

func TestMaxSecondsDuration(t *testing.T) {
	// Verify that maxSecondsDuration is calculated correctly
	expected := int64(math.MaxInt64 / int64(time.Second))
	if maxSecondsDuration != expected {
		t.Errorf("maxSecondsDuration = %v, want %v", maxSecondsDuration, expected)
	}

	// Verify that converting maxSecondsDuration to Duration doesn't overflow
	duration := time.Duration(maxSecondsDuration) * time.Second
	if duration < 0 {
		t.Error("maxSecondsDuration * time.Second resulted in negative duration (overflow)")
	}
}

func BenchmarkSaturateUint64ToInt64(b *testing.B) {
	testValues := []uint64{0, 12345, math.MaxInt64, math.MaxUint64}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, val := range testValues {
			_ = SaturateUint64ToInt64(val)
		}
	}
}

func BenchmarkSaturateInt64ToUint64(b *testing.B) {
	testValues := []int64{0, 12345, -12345, math.MaxInt64, math.MinInt64}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, val := range testValues {
			_ = SaturateInt64ToUint64(val)
		}
	}
}

func BenchmarkSecondsToDuration(b *testing.B) {
	testValues := []uint64{0, 60, 3600, 86400, math.MaxUint64}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, val := range testValues {
			_ = SecondsToDuration(val)
		}
	}
}

func TestSaturationConsistency(t *testing.T) {
	// Test that converting back and forth maintains consistency
	tests := []int64{-1000, -1, 0, 1, 1000, math.MaxInt64}

	for _, val := range tests {
		// int64 -> uint64 -> int64
		asUint64 := SaturateInt64ToUint64(val)
		backToInt64 := SaturateUint64ToInt64(asUint64)

		// For non-negative values, should be equal
		// For negative values, should become 0
		expected := val
		if val < 0 {
			expected = 0
		}

		if backToInt64 != expected {
			t.Errorf("Round-trip int64->uint64->int64 failed for %v: got %v, want %v", val, backToInt64, expected)
		}
	}
}

func TestEdgeCaseBoundaries(t *testing.T) {
	// Test values right at the boundaries
	t.Run("uint64 to int64 at boundary", func(t *testing.T) {
		// math.MaxInt64 should pass through unchanged
		if SaturateUint64ToInt64(math.MaxInt64) != math.MaxInt64 {
			t.Error("MaxInt64 was not preserved")
		}

		// math.MaxInt64 + 1 should saturate to MaxInt64
		if SaturateUint64ToInt64(math.MaxInt64+1) != math.MaxInt64 {
			t.Error("MaxInt64 + 1 did not saturate to MaxInt64")
		}
	})

	t.Run("uint64 to uint32 at boundary", func(t *testing.T) {
		// math.MaxUint32 should pass through unchanged
		if SaturateUint64ToUint32(math.MaxUint32) != math.MaxUint32 {
			t.Error("MaxUint32 was not preserved")
		}

		// math.MaxUint32 + 1 should saturate to MaxUint32
		if SaturateUint64ToUint32(math.MaxUint32+1) != math.MaxUint32 {
			t.Error("MaxUint32 + 1 did not saturate to MaxUint32")
		}
	})

	t.Run("int64 to uint32 at boundary", func(t *testing.T) {
		// int64(math.MaxUint32) should pass through unchanged
		if SaturateInt64ToUint32(int64(math.MaxUint32)) != math.MaxUint32 {
			t.Error("int64(MaxUint32) was not preserved")
		}

		// int64(math.MaxUint32) + 1 should saturate to MaxUint32
		if SaturateInt64ToUint32(int64(math.MaxUint32)+1) != math.MaxUint32 {
			t.Error("int64(MaxUint32) + 1 did not saturate to MaxUint32")
		}
	})
}
