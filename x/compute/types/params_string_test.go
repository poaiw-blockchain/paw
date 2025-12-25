package types

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
)

func TestParams_String(t *testing.T) {
	tests := []struct {
		name   string
		params *Params
		want   string
	}{
		{
			name:   "nil params",
			params: nil,
			want:   "",
		},
		{
			name: "params with values",
			params: &Params{
				MinProviderStake:           math.NewInt(1000000),
				VerificationTimeoutSeconds: 300,
			},
			want: "min_provider_stake",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.String()
			if tt.want == "" && got != "" {
				t.Errorf("Params.String() = %v, want empty string", got)
			}
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Errorf("Params.String() = %v, should contain %v", got, tt.want)
			}
		})
	}
}
