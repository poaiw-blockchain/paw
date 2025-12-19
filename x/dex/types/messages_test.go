package types

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Test addresses for validation tests - using valid bech32 cosmos addresses
var (
	validAddress   = "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q"
	invalidAddress = "invalid"
)

func init() {
	// Initialize SDK config to use cosmos prefix
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("cosmos", "cosmospub")
	config.SetBech32PrefixForValidator("cosmosvaloper", "cosmosvaloperpub")
	config.SetBech32PrefixForConsensusNode("cosmosvalcons", "cosmosvalconspub")
}

// ============================================================================
// MsgCreatePool Tests
// ============================================================================

func TestMsgCreatePool_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgCreatePool
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "upaw",
				TokenB:  "uatom",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			},
			wantErr: false,
		},
		{
			name: "invalid creator address",
			msg: MsgCreatePool{
				Creator: invalidAddress,
				TokenA:  "upaw",
				TokenB:  "uatom",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "invalid creator address",
		},
		{
			name: "empty token_a",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "",
				TokenB:  "uatom",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "token_a cannot be empty",
		},
		{
			name: "empty token_b",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "upaw",
				TokenB:  "",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "token_b cannot be empty",
		},
		{
			name: "same tokens",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "upaw",
				TokenB:  "upaw",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "tokens must be different",
		},
		{
			name: "invalid token_a denom",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "Bad Denom",
				TokenB:  "uatom",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "invalid denom",
		},
		{
			name: "invalid token_b denom",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "upaw",
				TokenB:  "123bad",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "invalid denom",
		},
		{
			name: "zero amount_a",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "upaw",
				TokenB:  "uatom",
				AmountA: math.NewInt(0),
				AmountB: math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "amount_a must be positive",
		},
		{
			name: "negative amount_a",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "upaw",
				TokenB:  "uatom",
				AmountA: math.NewInt(-1000),
				AmountB: math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "amount_a must be positive",
		},
		{
			name: "zero amount_b",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "upaw",
				TokenB:  "uatom",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(0),
			},
			wantErr: true,
			errMsg:  "amount_b must be positive",
		},
		{
			name: "negative amount_b",
			msg: MsgCreatePool{
				Creator: validAddress,
				TokenA:  "upaw",
				TokenB:  "uatom",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(-2000),
			},
			wantErr: true,
			errMsg:  "amount_b must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgCreatePool.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg && !contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgCreatePool.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgCreatePool_GetSigners(t *testing.T) {
	msg := MsgCreatePool{
		Creator: validAddress,
		TokenA:  "upaw",
		TokenB:  "uatom",
		AmountA: math.NewInt(1000000),
		AmountB: math.NewInt(2000000),
	}

	signers := msg.GetSigners()
	if len(signers) != 1 {
		t.Errorf("Expected 1 signer, got %d", len(signers))
	}

	expected, _ := sdk.AccAddressFromBech32(validAddress)
	if !signers[0].Equals(expected) {
		t.Errorf("Expected signer %s, got %s", expected, signers[0])
	}
}

// ============================================================================
// MsgAddLiquidity Tests
// ============================================================================

func TestMsgAddLiquidity_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgAddLiquidity
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgAddLiquidity{
				Provider: validAddress,
				PoolId:   1,
				AmountA:  math.NewInt(1000000),
				AmountB:  math.NewInt(2000000),
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: MsgAddLiquidity{
				Provider: invalidAddress,
				PoolId:   1,
				AmountA:  math.NewInt(1000000),
				AmountB:  math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "invalid provider address",
		},
		{
			name: "zero pool_id",
			msg: MsgAddLiquidity{
				Provider: validAddress,
				PoolId:   0,
				AmountA:  math.NewInt(1000000),
				AmountB:  math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "pool_id must be positive",
		},
		{
			name: "zero amount_a",
			msg: MsgAddLiquidity{
				Provider: validAddress,
				PoolId:   1,
				AmountA:  math.NewInt(0),
				AmountB:  math.NewInt(2000000),
			},
			wantErr: true,
			errMsg:  "amount_a must be positive",
		},
		{
			name: "negative amount_b",
			msg: MsgAddLiquidity{
				Provider: validAddress,
				PoolId:   1,
				AmountA:  math.NewInt(1000000),
				AmountB:  math.NewInt(-1),
			},
			wantErr: true,
			errMsg:  "amount_b must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgAddLiquidity.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgAddLiquidity.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgAddLiquidity_GetSigners(t *testing.T) {
	msg := MsgAddLiquidity{
		Provider: validAddress,
		PoolId:   1,
		AmountA:  math.NewInt(1000000),
		AmountB:  math.NewInt(2000000),
	}

	signers := msg.GetSigners()
	if len(signers) != 1 {
		t.Errorf("Expected 1 signer, got %d", len(signers))
	}

	expected, _ := sdk.AccAddressFromBech32(validAddress)
	if !signers[0].Equals(expected) {
		t.Errorf("Expected signer %s, got %s", expected, signers[0])
	}
}

// ============================================================================
// MsgRemoveLiquidity Tests
// ============================================================================

func TestMsgRemoveLiquidity_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgRemoveLiquidity
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgRemoveLiquidity{
				Provider: validAddress,
				PoolId:   1,
				Shares:   math.NewInt(1000000),
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: MsgRemoveLiquidity{
				Provider: invalidAddress,
				PoolId:   1,
				Shares:   math.NewInt(1000000),
			},
			wantErr: true,
			errMsg:  "invalid provider address",
		},
		{
			name: "zero pool_id",
			msg: MsgRemoveLiquidity{
				Provider: validAddress,
				PoolId:   0,
				Shares:   math.NewInt(1000000),
			},
			wantErr: true,
			errMsg:  "pool_id must be positive",
		},
		{
			name: "zero shares",
			msg: MsgRemoveLiquidity{
				Provider: validAddress,
				PoolId:   1,
				Shares:   math.NewInt(0),
			},
			wantErr: true,
			errMsg:  "shares must be positive",
		},
		{
			name: "negative shares",
			msg: MsgRemoveLiquidity{
				Provider: validAddress,
				PoolId:   1,
				Shares:   math.NewInt(-1000),
			},
			wantErr: true,
			errMsg:  "shares must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgRemoveLiquidity.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgRemoveLiquidity.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgRemoveLiquidity_GetSigners(t *testing.T) {
	msg := MsgRemoveLiquidity{
		Provider: validAddress,
		PoolId:   1,
		Shares:   math.NewInt(1000000),
	}

	signers := msg.GetSigners()
	if len(signers) != 1 {
		t.Errorf("Expected 1 signer, got %d", len(signers))
	}

	expected, _ := sdk.AccAddressFromBech32(validAddress)
	if !signers[0].Equals(expected) {
		t.Errorf("Expected signer %s, got %s", expected, signers[0])
	}
}

// ============================================================================
// MsgSwap Tests
// ============================================================================

func TestMsgSwap_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgSwap
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			},
			wantErr: false,
		},
		{
			name: "empty trader address",
			msg: MsgSwap{
				Trader:       "",
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "trader address cannot be empty",
		},
		{
			name: "invalid trader address",
			msg: MsgSwap{
				Trader:       invalidAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "invalid trader address",
		},
		{
			name: "zero pool_id",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       0,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "pool_id must be positive",
		},
		{
			name: "empty token_in",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "token_in cannot be empty",
		},
		{
			name: "empty token_out",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "token_out cannot be empty",
		},
		{
			name: "same tokens",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "upaw",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "cannot swap the same token",
		},
		{
			name: "zero amount_in",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(0),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "amount_in must be positive",
		},
		{
			name: "negative amount_in",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(-1000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "amount_in must be positive",
		},
		{
			name: "negative min_amount_out",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(-1),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "min_amount_out cannot be negative",
		},
		{
			name: "zero min_amount_out - slippage protection",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(0),
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "min_amount_out must be positive for slippage protection",
		},
		{
			name: "zero deadline",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Deadline:     0,
			},
			wantErr: true,
			errMsg:  "deadline must be set",
		},
		{
			name: "unreasonably high min_amount_out",
			msg: MsgSwap{
				Trader:       validAddress,
				PoolId:       1,
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000),
				MinAmountOut: math.NewInt(10000000), // 10000x input
				Deadline:     1700000000,
			},
			wantErr: true,
			errMsg:  "min_amount_out unreasonably high",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgSwap.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgSwap.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgSwap_GetSigners(t *testing.T) {
	msg := MsgSwap{
		Trader:       validAddress,
		PoolId:       1,
		TokenIn:      "upaw",
		TokenOut:     "uatom",
		AmountIn:     math.NewInt(1000000),
		MinAmountOut: math.NewInt(900000),
		Deadline:     1700000000,
	}

	signers := msg.GetSigners()
	if len(signers) != 1 {
		t.Errorf("Expected 1 signer, got %d", len(signers))
	}

	expected, _ := sdk.AccAddressFromBech32(validAddress)
	if !signers[0].Equals(expected) {
		t.Errorf("Expected signer %s, got %s", expected, signers[0])
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
