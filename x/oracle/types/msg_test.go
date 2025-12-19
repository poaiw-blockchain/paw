package types

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// Test addresses for validation tests - using valid bech32 cosmos addresses
var (
	validAddress    = "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q"
	validValAddress = "cosmosvaloper1zg69v7ys40x77y352eufp27daufrg4nckx5hjn"
	invalidAddress  = "invalid"
	moduleAuthority string
	moduleAccAddr   sdk.AccAddress
)

func init() {
	// Initialize SDK config to use cosmos prefix
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("cosmos", "cosmospub")
	config.SetBech32PrefixForValidator("cosmosvaloper", "cosmosvaloperpub")
	config.SetBech32PrefixForConsensusNode("cosmosvalcons", "cosmosvalconspub")
	moduleAccAddr = authtypes.NewModuleAddress(govtypes.ModuleName)
	moduleAuthority = moduleAccAddr.String()
}

// ============================================================================
// MsgSubmitPrice Tests
// ============================================================================

func TestMsgSubmitPrice_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgSubmitPrice
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    validAddress,
				Asset:     "BTC",
				Price:     math.LegacyNewDec(50000),
			},
			wantErr: false,
		},
		{
			name: "invalid validator address",
			msg: MsgSubmitPrice{
				Validator: invalidAddress,
				Feeder:    validAddress,
				Asset:     "BTC",
				Price:     math.LegacyNewDec(50000),
			},
			wantErr: true,
			errMsg:  "invalid validator address",
		},
		{
			name: "invalid feeder address",
			msg: MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    invalidAddress,
				Asset:     "BTC",
				Price:     math.LegacyNewDec(50000),
			},
			wantErr: true,
			errMsg:  "invalid feeder address",
		},
		{
			name: "empty asset",
			msg: MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    validAddress,
				Asset:     "",
				Price:     math.LegacyNewDec(50000),
			},
			wantErr: true,
			errMsg:  "asset cannot be empty",
		},
		{
			name: "zero price",
			msg: MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    validAddress,
				Asset:     "BTC",
				Price:     math.LegacyZeroDec(),
			},
			wantErr: true,
			errMsg:  "price must be positive",
		},
		{
			name: "negative price",
			msg: MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    validAddress,
				Asset:     "BTC",
				Price:     math.LegacyNewDec(-50000),
			},
			wantErr: true,
			errMsg:  "price must be positive",
		},
		{
			name: "fractional price",
			msg: MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    validAddress,
				Asset:     "BTC",
				Price:     math.LegacyNewDecWithPrec(500005, 1), // 50000.5
			},
			wantErr: false,
		},
		{
			name: "small positive price",
			msg: MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    validAddress,
				Asset:     "SHIB",
				Price:     math.LegacyNewDecWithPrec(1, 8), // 0.00000001
			},
			wantErr: false,
		},
		{
			name: "asset too long",
			msg: MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    validAddress,
				Asset:     strings.Repeat("A", maxAssetLen+1),
				Price:     math.LegacyNewDec(50000),
			},
			wantErr: true,
			errMsg:  "asset too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgSubmitPrice.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgSubmitPrice.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgSubmitPrice_GetSigners(t *testing.T) {
	msg := MsgSubmitPrice{
		Validator: validValAddress,
		Feeder:    validAddress,
		Asset:     "BTC",
		Price:     math.LegacyNewDec(50000),
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

func TestMsgSubmitPrice_Type(t *testing.T) {
	msg := MsgSubmitPrice{}
	if msg.Type() != TypeMsgSubmitPrice {
		t.Errorf("Expected type %s, got %s", TypeMsgSubmitPrice, msg.Type())
	}
}

func TestMsgSubmitPrice_Route(t *testing.T) {
	msg := MsgSubmitPrice{}
	if msg.Route() != RouterKey {
		t.Errorf("Expected route %s, got %s", RouterKey, msg.Route())
	}
}

func TestNewMsgSubmitPrice(t *testing.T) {
	msg := NewMsgSubmitPrice(validValAddress, validAddress, "BTC", math.LegacyNewDec(50000))

	if msg.Validator != validValAddress {
		t.Errorf("Expected validator %s, got %s", validValAddress, msg.Validator)
	}
	if msg.Feeder != validAddress {
		t.Errorf("Expected feeder %s, got %s", validAddress, msg.Feeder)
	}
	if msg.Asset != "BTC" {
		t.Errorf("Expected asset BTC, got %s", msg.Asset)
	}
	if !msg.Price.Equal(math.LegacyNewDec(50000)) {
		t.Errorf("Expected price 50000, got %s", msg.Price)
	}
}

// ============================================================================
// MsgDelegateFeedConsent Tests
// ============================================================================

func TestMsgDelegateFeedConsent_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgDelegateFeedConsent
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgDelegateFeedConsent{
				Validator: validValAddress,
				Delegate:  validAddress,
			},
			wantErr: false,
		},
		{
			name: "invalid validator address",
			msg: MsgDelegateFeedConsent{
				Validator: invalidAddress,
				Delegate:  validAddress,
			},
			wantErr: true,
			errMsg:  "invalid validator address",
		},
		{
			name: "invalid delegate address",
			msg: MsgDelegateFeedConsent{
				Validator: validValAddress,
				Delegate:  invalidAddress,
			},
			wantErr: true,
			errMsg:  "invalid delegate address",
		},
		{
			name: "empty validator",
			msg: MsgDelegateFeedConsent{
				Validator: "",
				Delegate:  validAddress,
			},
			wantErr: true,
			errMsg:  "invalid validator address",
		},
		{
			name: "empty delegate",
			msg: MsgDelegateFeedConsent{
				Validator: validValAddress,
				Delegate:  "",
			},
			wantErr: true,
			errMsg:  "invalid delegate address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgDelegateFeedConsent.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgDelegateFeedConsent.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgDelegateFeedConsent_GetSigners(t *testing.T) {
	msg := MsgDelegateFeedConsent{
		Validator: validValAddress,
		Delegate:  validAddress,
	}

	signers := msg.GetSigners()
	if len(signers) != 1 {
		t.Errorf("Expected 1 signer, got %d", len(signers))
	}

	valAddr, _ := sdk.ValAddressFromBech32(validValAddress)
	expected := sdk.AccAddress(valAddr)
	if !signers[0].Equals(expected) {
		t.Errorf("Expected signer %s, got %s", expected, signers[0])
	}
}

func TestMsgDelegateFeedConsent_Type(t *testing.T) {
	msg := MsgDelegateFeedConsent{}
	if msg.Type() != TypeMsgDelegateFeedConsent {
		t.Errorf("Expected type %s, got %s", TypeMsgDelegateFeedConsent, msg.Type())
	}
}

func TestNewMsgDelegateFeedConsent(t *testing.T) {
	msg := NewMsgDelegateFeedConsent(validValAddress, validAddress)

	if msg.Validator != validValAddress {
		t.Errorf("Expected validator %s, got %s", validValAddress, msg.Validator)
	}
	if msg.Delegate != validAddress {
		t.Errorf("Expected delegate %s, got %s", validAddress, msg.Delegate)
	}
}

// ============================================================================
// MsgUpdateParams Tests
// ============================================================================

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	validParams := Params{
		VotePeriod:    10,
		VoteThreshold: math.LegacyNewDecWithPrec(5, 1), // 0.5
		SlashFraction: math.LegacyNewDecWithPrec(1, 2), // 0.01
		SlashWindow:   1000,
	}

	tests := []struct {
		name    string
		msg     MsgUpdateParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params:    validParams,
			},
			wantErr: false,
		},
		{
			name: "unauthorized authority",
			msg: MsgUpdateParams{
				Authority: validAddress,
				Params:    validParams,
			},
			wantErr: true,
			errMsg:  "invalid authority",
		},
		{
			name: "invalid authority address",
			msg: MsgUpdateParams{
				Authority: invalidAddress,
				Params:    validParams,
			},
			wantErr: true,
			errMsg:  "invalid authority address",
		},
		{
			name: "zero vote period",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					VotePeriod:    0,
					VoteThreshold: math.LegacyNewDecWithPrec(5, 1),
					SlashFraction: math.LegacyNewDecWithPrec(1, 2),
				},
			},
			wantErr: true,
			errMsg:  "vote period must be positive",
		},
		{
			name: "vote threshold zero",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					VotePeriod:    10,
					VoteThreshold: math.LegacyZeroDec(),
					SlashFraction: math.LegacyNewDecWithPrec(1, 2),
				},
			},
			wantErr: true,
			errMsg:  "vote threshold must be between 0 and 1",
		},
		{
			name: "vote threshold greater than 1",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					VotePeriod:    10,
					VoteThreshold: math.LegacyNewDecWithPrec(15, 1), // 1.5
					SlashFraction: math.LegacyNewDecWithPrec(1, 2),
				},
			},
			wantErr: true,
			errMsg:  "vote threshold must be between 0 and 1",
		},
		{
			name: "slash fraction negative",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					VotePeriod:    10,
					VoteThreshold: math.LegacyNewDecWithPrec(5, 1),
					SlashFraction: math.LegacyNewDec(-1),
				},
			},
			wantErr: true,
			errMsg:  "slash fraction must be between 0 and 1",
		},
		{
			name: "slash fraction greater than 1",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					VotePeriod:    10,
					VoteThreshold: math.LegacyNewDecWithPrec(5, 1),
					SlashFraction: math.LegacyNewDecWithPrec(15, 1), // 1.5
				},
			},
			wantErr: true,
			errMsg:  "slash fraction must be between 0 and 1",
		},
		{
			name: "vote threshold at boundary (exactly 1)",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					VotePeriod:    10,
					VoteThreshold: math.LegacyOneDec(),
					SlashFraction: math.LegacyNewDecWithPrec(1, 2),
				},
			},
			wantErr: false,
		},
		{
			name: "slash fraction at zero boundary",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					VotePeriod:    10,
					VoteThreshold: math.LegacyNewDecWithPrec(5, 1),
					SlashFraction: math.LegacyZeroDec(),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgUpdateParams.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgUpdateParams.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgUpdateParams_GetSigners(t *testing.T) {
	msg := MsgUpdateParams{
		Authority: moduleAuthority,
		Params:    Params{},
	}

	signers := msg.GetSigners()
	if len(signers) != 1 {
		t.Errorf("Expected 1 signer, got %d", len(signers))
	}

	expected := moduleAccAddr
	if !signers[0].Equals(expected) {
		t.Errorf("Expected signer %s, got %s", expected, signers[0])
	}
}

func TestMsgUpdateParams_Type(t *testing.T) {
	msg := MsgUpdateParams{}
	if msg.Type() != TypeMsgUpdateParams {
		t.Errorf("Expected type %s, got %s", TypeMsgUpdateParams, msg.Type())
	}
}

func TestNewMsgUpdateParams(t *testing.T) {
	params := Params{
		VotePeriod:    10,
		VoteThreshold: math.LegacyNewDecWithPrec(5, 1),
		SlashFraction: math.LegacyNewDecWithPrec(1, 2),
	}
	msg := NewMsgUpdateParams(moduleAuthority, params)

	if msg.Authority != moduleAuthority {
		t.Errorf("Expected authority %s, got %s", moduleAuthority, msg.Authority)
	}
	if msg.Params.VotePeriod != params.VotePeriod {
		t.Errorf("Expected vote period %d, got %d", params.VotePeriod, msg.Params.VotePeriod)
	}
}

// ============================================================================
// Edge Cases and Security Tests
// ============================================================================

func TestMsgSubmitPrice_PriceEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		price   math.LegacyDec
		wantErr bool
	}{
		{
			name:    "very large price",
			price:   math.LegacyNewDec(1000000000000), // 1 trillion
			wantErr: false,
		},
		{
			name:    "very small positive price",
			price:   math.LegacyNewDecWithPrec(1, 18), // smallest positive
			wantErr: false,
		},
		{
			name:    "nil price",
			price:   math.LegacyDec{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    validAddress,
				Asset:     "BTC",
				Price:     tt.price,
			}
			err := msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgSubmitPrice.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMsgSubmitPrice_AssetEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		asset   string
		wantErr bool
	}{
		{
			name:    "normal asset",
			asset:   "BTC",
			wantErr: false,
		},
		{
			name:    "lowercase asset",
			asset:   "btc",
			wantErr: false,
		},
		{
			name:    "asset with numbers",
			asset:   "BTC2",
			wantErr: false,
		},
		{
			name:    "long asset name",
			asset:   "VERYLONGASSETNAME123",
			wantErr: false,
		},
		{
			name:    "asset with special chars",
			asset:   "BTC/USD",
			wantErr: false,
		},
		{
			name:    "empty asset",
			asset:   "",
			wantErr: true,
		},
		{
			name:    "whitespace only asset",
			asset:   "   ",
			wantErr: false, // ValidateBasic doesn't trim - this is valid but may fail at keeper level
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MsgSubmitPrice{
				Validator: validValAddress,
				Feeder:    validAddress,
				Asset:     tt.asset,
				Price:     math.LegacyNewDec(50000),
			}
			err := msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgSubmitPrice.ValidateBasic() asset=%q error = %v, wantErr %v", tt.asset, err, tt.wantErr)
			}
		})
	}
}
