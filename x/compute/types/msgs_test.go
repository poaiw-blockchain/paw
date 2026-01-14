package types

import (
	"bytes"
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
	validEndpoint   = "https://compute.example.com:8080"
	validContainer  = "docker.io/library/ubuntu:latest"
	validOutputHash = "a665a45920422f9d417e4867efdc4fb8a04a1f3fff1fa07e998e86f7f7a27ae3"
	validOutputURL  = "https://storage.example.com/results/123"
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
// MsgRegisterProvider Tests
// ============================================================================

func TestMsgRegisterProvider_ValidateBasic(t *testing.T) {
	validSpecs := ComputeSpec{
		CpuCores:       4,
		MemoryMb:       8192,
		TimeoutSeconds: 3600,
	}
	validPricing := Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(100),
		MemoryPricePerMbHour:  math.LegacyNewDec(10),
		GpuPricePerHour:       math.LegacyNewDec(1000),
		StoragePricePerGbHour: math.LegacyNewDec(5),
	}

	tests := []struct {
		name    string
		msg     MsgRegisterProvider
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgRegisterProvider{
				Provider:       validAddress,
				Moniker:        "test-provider",
				Endpoint:       validEndpoint,
				Stake:          math.NewInt(1000000),
				AvailableSpecs: validSpecs,
				Pricing:        validPricing,
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: MsgRegisterProvider{
				Provider:       invalidAddress,
				Moniker:        "test-provider",
				Endpoint:       validEndpoint,
				Stake:          math.NewInt(1000000),
				AvailableSpecs: validSpecs,
				Pricing:        validPricing,
			},
			wantErr: true,
			errMsg:  "invalid provider address",
		},
		{
			name: "empty moniker",
			msg: MsgRegisterProvider{
				Provider:       validAddress,
				Moniker:        "",
				Endpoint:       validEndpoint,
				Stake:          math.NewInt(1000000),
				AvailableSpecs: validSpecs,
				Pricing:        validPricing,
			},
			wantErr: true,
			errMsg:  "invalid moniker",
		},
		{
			name: "empty endpoint",
			msg: MsgRegisterProvider{
				Provider:       validAddress,
				Moniker:        "test-provider",
				Endpoint:       "",
				Stake:          math.NewInt(1000000),
				AvailableSpecs: validSpecs,
				Pricing:        validPricing,
			},
			wantErr: true,
			errMsg:  "invalid endpoint",
		},
		{
			name: "zero stake",
			msg: MsgRegisterProvider{
				Provider:       validAddress,
				Moniker:        "test-provider",
				Endpoint:       validEndpoint,
				Stake:          math.NewInt(0),
				AvailableSpecs: validSpecs,
				Pricing:        validPricing,
			},
			wantErr: true,
			errMsg:  "stake must be positive",
		},
		{
			name: "negative stake",
			msg: MsgRegisterProvider{
				Provider:       validAddress,
				Moniker:        "test-provider",
				Endpoint:       validEndpoint,
				Stake:          math.NewInt(-1000),
				AvailableSpecs: validSpecs,
				Pricing:        validPricing,
			},
			wantErr: true,
			errMsg:  "stake must be positive",
		},
		{
			name: "zero cpu cores",
			msg: MsgRegisterProvider{
				Provider: validAddress,
				Moniker:  "test-provider",
				Endpoint: validEndpoint,
				Stake:    math.NewInt(1000000),
				AvailableSpecs: ComputeSpec{
					CpuCores:       0,
					MemoryMb:       8192,
					TimeoutSeconds: 3600,
				},
				Pricing: validPricing,
			},
			wantErr: true,
			errMsg:  "cpu_cores must be greater than 0",
		},
		{
			name: "zero memory",
			msg: MsgRegisterProvider{
				Provider: validAddress,
				Moniker:  "test-provider",
				Endpoint: validEndpoint,
				Stake:    math.NewInt(1000000),
				AvailableSpecs: ComputeSpec{
					CpuCores:       4,
					MemoryMb:       0,
					TimeoutSeconds: 3600,
				},
				Pricing: validPricing,
			},
			wantErr: true,
			errMsg:  "memory_mb must be greater than 0",
		},
		{
			name: "negative pricing",
			msg: MsgRegisterProvider{
				Provider:       validAddress,
				Moniker:        "test-provider",
				Endpoint:       validEndpoint,
				Stake:          math.NewInt(1000000),
				AvailableSpecs: validSpecs,
				Pricing: Pricing{
					CpuPricePerMcoreHour:  math.LegacyNewDec(-100),
					MemoryPricePerMbHour:  math.LegacyNewDec(10),
					GpuPricePerHour:       math.LegacyNewDec(1000),
					StoragePricePerGbHour: math.LegacyNewDec(5),
				},
			},
			wantErr: true,
			errMsg:  "cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgRegisterProvider.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgRegisterProvider.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgRegisterProvider_GetSigners(t *testing.T) {
	msg := MsgRegisterProvider{
		Provider: validAddress,
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
// MsgUpdateProvider Tests
// ============================================================================

func TestMsgUpdateProvider_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgUpdateProvider
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message with no updates",
			msg: MsgUpdateProvider{
				Provider: validAddress,
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: MsgUpdateProvider{
				Provider: invalidAddress,
			},
			wantErr: true,
			errMsg:  "invalid provider address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgUpdateProvider.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgUpdateProvider.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgDeactivateProvider Tests
// ============================================================================

func TestMsgDeactivateProvider_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgDeactivateProvider
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgDeactivateProvider{
				Provider: validAddress,
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: MsgDeactivateProvider{
				Provider: invalidAddress,
			},
			wantErr: true,
			errMsg:  "invalid provider address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgDeactivateProvider.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgDeactivateProvider.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgSubmitRequest Tests
// ============================================================================

func TestMsgSubmitRequest_ValidateBasic(t *testing.T) {
	validSpecs := ComputeSpec{
		CpuCores:       4,
		MemoryMb:       8192,
		TimeoutSeconds: 3600,
	}

	tests := []struct {
		name    string
		msg     MsgSubmitRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgSubmitRequest{
				Requester:      validAddress,
				Specs:          validSpecs,
				ContainerImage: validContainer,
				Command:        []string{"python", "script.py"},
				EnvVars:        map[string]string{"ENV": "test"},
				MaxPayment:     math.NewInt(1000000),
			},
			wantErr: false,
		},
		{
			name: "invalid requester address",
			msg: MsgSubmitRequest{
				Requester:      invalidAddress,
				Specs:          validSpecs,
				ContainerImage: validContainer,
				Command:        []string{"python", "script.py"},
				MaxPayment:     math.NewInt(1000000),
			},
			wantErr: true,
			errMsg:  "invalid requester address",
		},
		{
			name: "zero cpu cores",
			msg: MsgSubmitRequest{
				Requester: validAddress,
				Specs: ComputeSpec{
					CpuCores:       0,
					MemoryMb:       8192,
					TimeoutSeconds: 3600,
				},
				ContainerImage: validContainer,
				Command:        []string{"python", "script.py"},
				MaxPayment:     math.NewInt(1000000),
			},
			wantErr: true,
			errMsg:  "cpu_cores must be greater than 0",
		},
		{
			name: "zero max payment",
			msg: MsgSubmitRequest{
				Requester:      validAddress,
				Specs:          validSpecs,
				ContainerImage: validContainer,
				Command:        []string{"python", "script.py"},
				MaxPayment:     math.NewInt(0),
			},
			wantErr: true,
			errMsg:  "max payment must be positive",
		},
		{
			name: "valid with preferred provider",
			msg: MsgSubmitRequest{
				Requester:         validAddress,
				Specs:             validSpecs,
				ContainerImage:    validContainer,
				Command:           []string{"python", "script.py"},
				MaxPayment:        math.NewInt(1000000),
				PreferredProvider: validAddress,
			},
			wantErr: false,
		},
		{
			name: "invalid preferred provider",
			msg: MsgSubmitRequest{
				Requester:         validAddress,
				Specs:             validSpecs,
				ContainerImage:    validContainer,
				Command:           []string{"python", "script.py"},
				MaxPayment:        math.NewInt(1000000),
				PreferredProvider: invalidAddress,
			},
			wantErr: true,
			errMsg:  "invalid preferred provider address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgSubmitRequest.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgSubmitRequest.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgCancelRequest Tests
// ============================================================================

func TestMsgCancelRequest_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgCancelRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgCancelRequest{
				Requester: validAddress,
				RequestId: 1,
			},
			wantErr: false,
		},
		{
			name: "invalid requester address",
			msg: MsgCancelRequest{
				Requester: invalidAddress,
				RequestId: 1,
			},
			wantErr: true,
			errMsg:  "invalid requester address",
		},
		{
			name: "zero request ID",
			msg: MsgCancelRequest{
				Requester: validAddress,
				RequestId: 0,
			},
			wantErr: true,
			errMsg:  "request ID must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgCancelRequest.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgCancelRequest.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgSubmitResult Tests
// ============================================================================

func TestMsgSubmitResult_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgSubmitResult
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgSubmitResult{
				Provider:   validAddress,
				RequestId:  1,
				OutputHash: validOutputHash,
				OutputUrl:  validOutputURL,
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: MsgSubmitResult{
				Provider:   invalidAddress,
				RequestId:  1,
				OutputHash: validOutputHash,
				OutputUrl:  validOutputURL,
			},
			wantErr: true,
			errMsg:  "invalid provider address",
		},
		{
			name: "zero request ID",
			msg: MsgSubmitResult{
				Provider:   validAddress,
				RequestId:  0,
				OutputHash: validOutputHash,
				OutputUrl:  validOutputURL,
			},
			wantErr: true,
			errMsg:  "request ID must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgSubmitResult.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgSubmitResult.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgCreateDispute Tests
// ============================================================================

func TestMsgCreateDispute_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgCreateDispute
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgCreateDispute{
				Requester:     validAddress,
				RequestId:     1,
				Reason:        "Provider returned incorrect results",
				DepositAmount: math.NewInt(100000),
			},
			wantErr: false,
		},
		{
			name: "invalid requester address",
			msg: MsgCreateDispute{
				Requester:     invalidAddress,
				RequestId:     1,
				Reason:        "Provider returned incorrect results",
				DepositAmount: math.NewInt(100000),
			},
			wantErr: true,
			errMsg:  "invalid requester address",
		},
		{
			name: "zero request ID",
			msg: MsgCreateDispute{
				Requester:     validAddress,
				RequestId:     0,
				Reason:        "Provider returned incorrect results",
				DepositAmount: math.NewInt(100000),
			},
			wantErr: true,
			errMsg:  "request ID must be greater than 0",
		},
		{
			name: "empty reason",
			msg: MsgCreateDispute{
				Requester:     validAddress,
				RequestId:     1,
				Reason:        "",
				DepositAmount: math.NewInt(100000),
			},
			wantErr: true,
			errMsg:  "reason is required",
		},
		{
			name: "reason too long",
			msg: MsgCreateDispute{
				Requester:     validAddress,
				RequestId:     1,
				Reason:        strings.Repeat("a", maxDisputeReasonLength+1),
				DepositAmount: math.NewInt(100000),
			},
			wantErr: true,
			errMsg:  "reason exceeds max length",
		},
		{
			name: "zero deposit",
			msg: MsgCreateDispute{
				Requester:     validAddress,
				RequestId:     1,
				Reason:        "Provider returned incorrect results",
				DepositAmount: math.NewInt(0),
			},
			wantErr: true,
			errMsg:  "deposit must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgCreateDispute.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgCreateDispute.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgVoteOnDispute Tests
// ============================================================================

func TestMsgVoteOnDispute_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgVoteOnDispute
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgVoteOnDispute{
				Validator: validValAddress,
				DisputeId: 1,
			},
			wantErr: false,
		},
		{
			name: "invalid validator address",
			msg: MsgVoteOnDispute{
				Validator: invalidAddress,
				DisputeId: 1,
			},
			wantErr: true,
			errMsg:  "invalid validator address",
		},
		{
			name: "zero dispute ID",
			msg: MsgVoteOnDispute{
				Validator: validValAddress,
				DisputeId: 0,
			},
			wantErr: true,
			errMsg:  "dispute ID must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgVoteOnDispute.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgVoteOnDispute.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgResolveDispute Tests
// ============================================================================

func TestMsgResolveDispute_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgResolveDispute
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgResolveDispute{
				Authority: moduleAuthority,
				DisputeId: 1,
			},
			wantErr: false,
		},
		{
			name: "unauthorized authority",
			msg: MsgResolveDispute{
				Authority: validAddress,
				DisputeId: 1,
			},
			wantErr: true,
			errMsg:  "invalid authority",
		},
		{
			name: "invalid authority address",
			msg: MsgResolveDispute{
				Authority: invalidAddress,
				DisputeId: 1,
			},
			wantErr: true,
			errMsg:  "invalid authority address",
		},
		{
			name: "zero dispute ID",
			msg: MsgResolveDispute{
				Authority: moduleAuthority,
				DisputeId: 0,
			},
			wantErr: true,
			errMsg:  "dispute ID must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgResolveDispute.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgResolveDispute.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgSubmitEvidence Tests
// ============================================================================

func TestMsgSubmitEvidence_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgSubmitEvidence
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgSubmitEvidence{
				Submitter: validAddress,
				DisputeId: 1,
				Data:      []byte("evidence data"),
			},
			wantErr: false,
		},
		{
			name: "invalid submitter address",
			msg: MsgSubmitEvidence{
				Submitter: invalidAddress,
				DisputeId: 1,
				Data:      []byte("evidence data"),
			},
			wantErr: true,
			errMsg:  "invalid submitter address",
		},
		{
			name: "zero dispute ID",
			msg: MsgSubmitEvidence{
				Submitter: validAddress,
				DisputeId: 0,
				Data:      []byte("evidence data"),
			},
			wantErr: true,
			errMsg:  "dispute ID must be greater than 0",
		},
		{
			name: "empty evidence data",
			msg: MsgSubmitEvidence{
				Submitter: validAddress,
				DisputeId: 1,
				Data:      []byte{},
			},
			wantErr: true,
			errMsg:  "evidence data cannot be empty",
		},
		{
			name: "evidence too large",
			msg: MsgSubmitEvidence{
				Submitter:    validAddress,
				DisputeId:    1,
				Data:         bytes.Repeat([]byte{0x01}, int(maxGovernanceEvidenceSizeLimit)+1),
				EvidenceType: "log",
			},
			wantErr: true,
			errMsg:  "exceeds hard limit",
		},
		{
			name: "description too long",
			msg: MsgSubmitEvidence{
				Submitter:    validAddress,
				DisputeId:    1,
				Data:         []byte{0x01},
				EvidenceType: "log",
				Description:  strings.Repeat("a", maxEvidenceDescriptionLength+1),
			},
			wantErr: true,
			errMsg:  "description exceeds max length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgSubmitEvidence.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgSubmitEvidence.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgAppealSlashing Tests
// ============================================================================

func TestMsgAppealSlashing_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgAppealSlashing
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgAppealSlashing{
				Provider:      validAddress,
				SlashId:       1,
				Justification: "The slashing was unfair because...",
				DepositAmount: math.NewInt(100000),
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: MsgAppealSlashing{
				Provider:      invalidAddress,
				SlashId:       1,
				Justification: "The slashing was unfair",
				DepositAmount: math.NewInt(100000),
			},
			wantErr: true,
			errMsg:  "invalid provider address",
		},
		{
			name: "zero slash ID",
			msg: MsgAppealSlashing{
				Provider:      validAddress,
				SlashId:       0,
				Justification: "The slashing was unfair",
				DepositAmount: math.NewInt(100000),
			},
			wantErr: true,
			errMsg:  "slash ID must be greater than 0",
		},
		{
			name: "empty justification",
			msg: MsgAppealSlashing{
				Provider:      validAddress,
				SlashId:       1,
				Justification: "",
				DepositAmount: math.NewInt(100000),
			},
			wantErr: true,
			errMsg:  "justification is required",
		},
		{
			name: "zero deposit",
			msg: MsgAppealSlashing{
				Provider:      validAddress,
				SlashId:       1,
				Justification: "The slashing was unfair",
				DepositAmount: math.NewInt(0),
			},
			wantErr: true,
			errMsg:  "deposit must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgAppealSlashing.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgAppealSlashing.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgVoteOnAppeal Tests
// ============================================================================

func TestMsgVoteOnAppeal_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgVoteOnAppeal
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgVoteOnAppeal{
				Validator: validValAddress,
				AppealId:  1,
			},
			wantErr: false,
		},
		{
			name: "invalid validator address",
			msg: MsgVoteOnAppeal{
				Validator: invalidAddress,
				AppealId:  1,
			},
			wantErr: true,
			errMsg:  "invalid validator address",
		},
		{
			name: "zero appeal ID",
			msg: MsgVoteOnAppeal{
				Validator: validValAddress,
				AppealId:  0,
			},
			wantErr: true,
			errMsg:  "appeal ID must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgVoteOnAppeal.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgVoteOnAppeal.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgResolveAppeal Tests
// ============================================================================

func TestMsgResolveAppeal_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgResolveAppeal
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgResolveAppeal{
				Authority: moduleAuthority,
				AppealId:  1,
			},
			wantErr: false,
		},
		{
			name: "unauthorized authority",
			msg: MsgResolveAppeal{
				Authority: validAddress,
				AppealId:  1,
			},
			wantErr: true,
			errMsg:  "invalid authority",
		},
		{
			name: "invalid authority address",
			msg: MsgResolveAppeal{
				Authority: invalidAddress,
				AppealId:  1,
			},
			wantErr: true,
			errMsg:  "invalid authority address",
		},
		{
			name: "zero appeal ID",
			msg: MsgResolveAppeal{
				Authority: moduleAuthority,
				AppealId:  0,
			},
			wantErr: true,
			errMsg:  "appeal ID must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgResolveAppeal.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgResolveAppeal.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgUpdateGovernanceParams Tests
// ============================================================================

func TestMsgUpdateGovernanceParams_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgUpdateGovernanceParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message",
			msg: MsgUpdateGovernanceParams{
				Authority: moduleAuthority,
				Params:    DefaultGovernanceParams(),
			},
			wantErr: false,
		},
		{
			name: "unauthorized authority",
			msg: MsgUpdateGovernanceParams{
				Authority: validAddress,
				Params:    DefaultGovernanceParams(),
			},
			wantErr: true,
			errMsg:  "invalid authority",
		},
		{
			name: "invalid authority address",
			msg: MsgUpdateGovernanceParams{
				Authority: invalidAddress,
			},
			wantErr: true,
			errMsg:  "invalid authority address",
		},
		{
			name: "evidence size exceeds hard limit",
			msg: MsgUpdateGovernanceParams{
				Authority: moduleAuthority,
				Params: func() GovernanceParams {
					p := DefaultGovernanceParams()
					p.MaxEvidenceSize = maxGovernanceEvidenceSizeLimit + 1
					return p
				}(),
			},
			wantErr: true,
			errMsg:  "max_evidence_size exceeds hard limit",
		},
		{
			name: "zero evidence size",
			msg: MsgUpdateGovernanceParams{
				Authority: moduleAuthority,
				Params: func() GovernanceParams {
					p := DefaultGovernanceParams()
					p.MaxEvidenceSize = 0
					return p
				}(),
			},
			wantErr: true,
			errMsg:  "max_evidence_size must be greater than 0",
		},
		{
			name: "quorum above 1",
			msg: MsgUpdateGovernanceParams{
				Authority: moduleAuthority,
				Params: func() GovernanceParams {
					p := DefaultGovernanceParams()
					p.QuorumPercentage = math.LegacyMustNewDecFromStr("1.1")
					return p
				}(),
			},
			wantErr: true,
			errMsg:  "quorum_percentage must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgUpdateGovernanceParams.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgUpdateGovernanceParams.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// GetSigners Tests for All Messages
// ============================================================================

// Custom interface to access GetSigners on messages that implement it
type msgWithSigners interface {
	GetSigners() []sdk.AccAddress
}

func TestGetSigners(t *testing.T) {
	addr, _ := sdk.AccAddressFromBech32(validAddress)
	valAddr, _ := sdk.ValAddressFromBech32(validValAddress)

	tests := []struct {
		name     string
		msg      msgWithSigners
		expected []sdk.AccAddress
	}{
		{
			name:     "MsgUpdateProvider",
			msg:      &MsgUpdateProvider{Provider: validAddress},
			expected: []sdk.AccAddress{addr},
		},
		{
			name:     "MsgDeactivateProvider",
			msg:      &MsgDeactivateProvider{Provider: validAddress},
			expected: []sdk.AccAddress{addr},
		},
		{
			name:     "MsgSubmitRequest",
			msg:      &MsgSubmitRequest{Requester: validAddress},
			expected: []sdk.AccAddress{addr},
		},
		{
			name:     "MsgCancelRequest",
			msg:      &MsgCancelRequest{Requester: validAddress},
			expected: []sdk.AccAddress{addr},
		},
		{
			name:     "MsgSubmitResult",
			msg:      &MsgSubmitResult{Provider: validAddress},
			expected: []sdk.AccAddress{addr},
		},
		{
			name:     "MsgCreateDispute",
			msg:      &MsgCreateDispute{Requester: validAddress},
			expected: []sdk.AccAddress{addr},
		},
		{
			name:     "MsgVoteOnDispute",
			msg:      &MsgVoteOnDispute{Validator: validValAddress},
			expected: []sdk.AccAddress{sdk.AccAddress(valAddr)},
		},
		{
			name:     "MsgResolveDispute",
			msg:      &MsgResolveDispute{Authority: moduleAuthority},
			expected: []sdk.AccAddress{moduleAccAddr},
		},
		{
			name:     "MsgSubmitEvidence",
			msg:      &MsgSubmitEvidence{Submitter: validAddress},
			expected: []sdk.AccAddress{addr},
		},
		{
			name:     "MsgAppealSlashing",
			msg:      &MsgAppealSlashing{Provider: validAddress},
			expected: []sdk.AccAddress{addr},
		},
		{
			name:     "MsgVoteOnAppeal",
			msg:      &MsgVoteOnAppeal{Validator: validValAddress},
			expected: []sdk.AccAddress{sdk.AccAddress(valAddr)},
		},
		{
			name:     "MsgResolveAppeal",
			msg:      &MsgResolveAppeal{Authority: moduleAuthority},
			expected: []sdk.AccAddress{moduleAccAddr},
		},
		{
			name:     "MsgUpdateGovernanceParams",
			msg:      &MsgUpdateGovernanceParams{Authority: moduleAuthority},
			expected: []sdk.AccAddress{moduleAccAddr},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			signers := tt.msg.GetSigners()
			if len(signers) != len(tt.expected) {
				t.Errorf("%s.GetSigners() returned %d signers, expected %d", tt.name, len(signers), len(tt.expected))
				return
			}
			for i, signer := range signers {
				if !signer.Equals(tt.expected[i]) {
					t.Errorf("%s.GetSigners()[%d] = %s, expected %s", tt.name, i, signer, tt.expected[i])
				}
			}
		})
	}
}

// ============================================================================
// MsgUpdateParams Tests
// ============================================================================

func TestMsgUpdateParams_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     MsgUpdateParams
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message with default params",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params:    DefaultParams(),
			},
			wantErr: false,
		},
		{
			name: "invalid authority address",
			msg: MsgUpdateParams{
				Authority: invalidAddress,
				Params:    DefaultParams(),
			},
			wantErr: true,
			errMsg:  "invalid authority address",
		},
		{
			name: "unauthorized authority",
			msg: MsgUpdateParams{
				Authority: validAddress,
				Params:    DefaultParams(),
			},
			wantErr: true,
			errMsg:  "invalid authority",
		},
		{
			name: "nil min provider stake",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					MinProviderStake:           math.Int{},
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
				},
			},
			wantErr: true,
			errMsg:  "min_provider_stake must be non-negative",
		},
		{
			name: "negative min provider stake",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					MinProviderStake:           math.NewInt(-1000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
				},
			},
			wantErr: true,
			errMsg:  "min_provider_stake must be non-negative",
		},
		{
			name: "zero verification timeout",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 0,
					MaxRequestTimeoutSeconds:   3600,
				},
			},
			wantErr: true,
			errMsg:  "verification_timeout_seconds must be greater than 0",
		},
		{
			name: "zero max request timeout",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   0,
				},
			},
			wantErr: true,
			errMsg:  "max_request_timeout_seconds must be greater than 0",
		},
		{
			name: "reputation slash percentage exceeds 100",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  101,
				},
			},
			wantErr: true,
			errMsg:  "reputation_slash_percentage cannot exceed 100",
		},
		{
			name: "stake slash percentage exceeds 100",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       101,
				},
			},
			wantErr: true,
			errMsg:  "stake_slash_percentage cannot exceed 100",
		},
		{
			name: "min reputation score exceeds 100",
			msg: MsgUpdateParams{
				Authority: moduleAuthority,
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       5,
					MinReputationScore:         101,
				},
			},
			wantErr: true,
			errMsg:  "min_reputation_score cannot exceed 100",
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
	}

	signers := msg.GetSigners()
	if len(signers) != 1 {
		t.Errorf("Expected 1 signer, got %d", len(signers))
	}

	if !signers[0].Equals(moduleAccAddr) {
		t.Errorf("Expected signer %s, got %s", moduleAccAddr, signers[0])
	}
}

// ============================================================================
// MsgRegisterSigningKey Tests
// ============================================================================

func TestMsgRegisterSigningKey_ValidateBasic(t *testing.T) {
	validPublicKey := make([]byte, 32)
	for i := range validPublicKey {
		validPublicKey[i] = byte(i + 1) // Non-zero key
	}

	tests := []struct {
		name    string
		msg     MsgRegisterSigningKey
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid message without old key signature",
			msg: MsgRegisterSigningKey{
				Provider:  validAddress,
				PublicKey: validPublicKey,
			},
			wantErr: false,
		},
		{
			name: "valid message with old key signature",
			msg: MsgRegisterSigningKey{
				Provider:        validAddress,
				PublicKey:       validPublicKey,
				OldKeySignature: make([]byte, 64),
			},
			wantErr: false,
		},
		{
			name: "invalid provider address",
			msg: MsgRegisterSigningKey{
				Provider:  invalidAddress,
				PublicKey: validPublicKey,
			},
			wantErr: true,
			errMsg:  "invalid provider address",
		},
		{
			name: "public key too short",
			msg: MsgRegisterSigningKey{
				Provider:  validAddress,
				PublicKey: make([]byte, 31),
			},
			wantErr: true,
			errMsg:  "invalid public key size",
		},
		{
			name: "public key too long",
			msg: MsgRegisterSigningKey{
				Provider:  validAddress,
				PublicKey: make([]byte, 33),
			},
			wantErr: true,
			errMsg:  "invalid public key size",
		},
		{
			name: "all zeros public key",
			msg: MsgRegisterSigningKey{
				Provider:  validAddress,
				PublicKey: make([]byte, 32), // All zeros
			},
			wantErr: true,
			errMsg:  "invalid public key: all zeros",
		},
		{
			name: "invalid old key signature size",
			msg: MsgRegisterSigningKey{
				Provider:        validAddress,
				PublicKey:       validPublicKey,
				OldKeySignature: make([]byte, 63), // Should be 64
			},
			wantErr: true,
			errMsg:  "invalid old key signature size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgRegisterSigningKey.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgRegisterSigningKey.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgRegisterSigningKey_GetSigners(t *testing.T) {
	msg := MsgRegisterSigningKey{
		Provider: validAddress,
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

func TestMsgRegisterSigningKey_ProtoMethods(t *testing.T) {
	msg := MsgRegisterSigningKey{
		Provider:  validAddress,
		PublicKey: make([]byte, 32),
	}

	// Test Reset
	msg.Reset()
	if msg.Provider != "" || len(msg.PublicKey) != 0 {
		t.Error("Reset() should clear all fields")
	}

	// Test String
	msg = MsgRegisterSigningKey{
		Provider:  validAddress,
		PublicKey: []byte{0x01, 0x02, 0x03},
	}
	str := msg.String()
	if str == "" {
		t.Error("String() should return non-empty string")
	}

	// Test ProtoMessage (just ensure it doesn't panic)
	msg.ProtoMessage()
}

func TestMsgRegisterSigningKeyResponse_ProtoMethods(t *testing.T) {
	resp := MsgRegisterSigningKeyResponse{}

	// Test Reset (should not panic)
	resp.Reset()

	// Test String
	str := resp.String()
	if str != "MsgRegisterSigningKeyResponse{}" {
		t.Errorf("String() = %v, want 'MsgRegisterSigningKeyResponse{}'", str)
	}

	// Test ProtoMessage (just ensure it doesn't panic)
	resp.ProtoMessage()
}

// ============================================================================
// MsgUpdateProvider Additional Tests
// ============================================================================

func TestMsgUpdateProvider_WithSpecs(t *testing.T) {
	validSpecs := ComputeSpec{
		CpuCores:       4,
		MemoryMb:       8192,
		TimeoutSeconds: 3600,
	}
	invalidSpecs := ComputeSpec{
		CpuCores:       0, // Invalid
		MemoryMb:       8192,
		TimeoutSeconds: 3600,
	}

	tests := []struct {
		name    string
		msg     MsgUpdateProvider
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with specs",
			msg: MsgUpdateProvider{
				Provider:       validAddress,
				AvailableSpecs: &validSpecs,
			},
			wantErr: false,
		},
		{
			name: "invalid specs",
			msg: MsgUpdateProvider{
				Provider:       validAddress,
				AvailableSpecs: &invalidSpecs,
			},
			wantErr: true,
			errMsg:  "cpu_cores must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgUpdateProvider.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgUpdateProvider.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestMsgUpdateProvider_WithPricing(t *testing.T) {
	validPricing := Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(100),
		MemoryPricePerMbHour:  math.LegacyNewDec(10),
		GpuPricePerHour:       math.LegacyNewDec(1000),
		StoragePricePerGbHour: math.LegacyNewDec(5),
	}
	invalidPricing := Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(-100), // Invalid
		MemoryPricePerMbHour:  math.LegacyNewDec(10),
		GpuPricePerHour:       math.LegacyNewDec(1000),
		StoragePricePerGbHour: math.LegacyNewDec(5),
	}

	tests := []struct {
		name    string
		msg     MsgUpdateProvider
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid with pricing",
			msg: MsgUpdateProvider{
				Provider: validAddress,
				Pricing:  &validPricing,
			},
			wantErr: false,
		},
		{
			name: "invalid pricing",
			msg: MsgUpdateProvider{
				Provider: validAddress,
				Pricing:  &invalidPricing,
			},
			wantErr: true,
			errMsg:  "cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("MsgUpdateProvider.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("MsgUpdateProvider.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// validatePricing Additional Tests
// ============================================================================

func TestValidatePricing(t *testing.T) {
	tests := []struct {
		name    string
		pricing Pricing
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pricing",
			pricing: Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDec(100),
				MemoryPricePerMbHour:  math.LegacyNewDec(10),
				GpuPricePerHour:       math.LegacyNewDec(1000),
				StoragePricePerGbHour: math.LegacyNewDec(5),
			},
			wantErr: false,
		},
		{
			name: "zero pricing allowed",
			pricing: Pricing{
				CpuPricePerMcoreHour:  math.LegacyZeroDec(),
				MemoryPricePerMbHour:  math.LegacyZeroDec(),
				GpuPricePerHour:       math.LegacyZeroDec(),
				StoragePricePerGbHour: math.LegacyZeroDec(),
			},
			wantErr: false,
		},
		{
			name: "negative memory price",
			pricing: Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDec(100),
				MemoryPricePerMbHour:  math.LegacyNewDec(-10),
				GpuPricePerHour:       math.LegacyNewDec(1000),
				StoragePricePerGbHour: math.LegacyNewDec(5),
			},
			wantErr: true,
			errMsg:  "memory_price_per_mb_hour cannot be negative",
		},
		{
			name: "negative GPU price",
			pricing: Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDec(100),
				MemoryPricePerMbHour:  math.LegacyNewDec(10),
				GpuPricePerHour:       math.LegacyNewDec(-1000),
				StoragePricePerGbHour: math.LegacyNewDec(5),
			},
			wantErr: true,
			errMsg:  "gpu_price_per_hour cannot be negative",
		},
		{
			name: "negative storage price",
			pricing: Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDec(100),
				MemoryPricePerMbHour:  math.LegacyNewDec(10),
				GpuPricePerHour:       math.LegacyNewDec(1000),
				StoragePricePerGbHour: math.LegacyNewDec(-5),
			},
			wantErr: true,
			errMsg:  "storage_price_per_gb_hour cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MsgRegisterProvider{
				Provider: validAddress,
				Moniker:  "test",
				Endpoint: validEndpoint,
				Stake:    math.NewInt(1000000),
				AvailableSpecs: ComputeSpec{
					CpuCores:       4,
					MemoryMb:       8192,
					TimeoutSeconds: 3600,
				},
				Pricing: tt.pricing,
			}
			err := msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePricing() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validatePricing() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// validateComputeSpec Additional Tests
// ============================================================================

func TestValidateComputeSpec(t *testing.T) {
	tests := []struct {
		name    string
		specs   ComputeSpec
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid specs",
			specs: ComputeSpec{
				CpuCores:       4,
				MemoryMb:       8192,
				TimeoutSeconds: 3600,
			},
			wantErr: false,
		},
		{
			name: "zero timeout",
			specs: ComputeSpec{
				CpuCores:       4,
				MemoryMb:       8192,
				TimeoutSeconds: 0,
			},
			wantErr: true,
			errMsg:  "timeout_seconds must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := MsgSubmitRequest{
				Requester:      validAddress,
				Specs:          tt.specs,
				ContainerImage: validContainer,
				Command:        []string{"python", "script.py"},
				MaxPayment:     math.NewInt(1000000),
			}
			err := msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateComputeSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateComputeSpec() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// MsgAppealSlashing Additional Tests
// ============================================================================

func TestMsgAppealSlashing_JustificationTooLong(t *testing.T) {
	msg := MsgAppealSlashing{
		Provider:      validAddress,
		SlashId:       1,
		Justification: strings.Repeat("a", maxAppealJustificationLength+1),
		DepositAmount: math.NewInt(100000),
	}

	err := msg.ValidateBasic()
	if err == nil {
		t.Error("Expected error for justification too long")
	}
	if !strings.Contains(err.Error(), "justification exceeds max length") {
		t.Errorf("Expected error about justification length, got: %v", err)
	}
}
