package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Message type URLs
const (
	TypeMsgRegisterProvider   = "register_provider"
	TypeMsgUpdateProvider     = "update_provider"
	TypeMsgDeactivateProvider = "deactivate_provider"
	TypeMsgSubmitRequest      = "submit_request"
	TypeMsgCancelRequest      = "cancel_request"
	TypeMsgSubmitResult       = "submit_result"
	TypeMsgUpdateParams       = "update_params"
	TypeMsgCreateDispute      = "create_dispute"
	TypeMsgVoteOnDispute      = "vote_on_dispute"
	TypeMsgResolveDispute     = "resolve_dispute"
	TypeMsgSubmitEvidence     = "submit_evidence"
	TypeMsgAppealSlashing     = "appeal_slashing"
	TypeMsgVoteOnAppeal       = "vote_on_appeal"
	TypeMsgResolveAppeal      = "resolve_appeal"
	TypeMsgUpdateGovParams    = "update_governance_params"
)

var (
	_ sdk.Msg = &MsgRegisterProvider{}
	_ sdk.Msg = &MsgUpdateProvider{}
	_ sdk.Msg = &MsgDeactivateProvider{}
	_ sdk.Msg = &MsgSubmitRequest{}
	_ sdk.Msg = &MsgCancelRequest{}
	_ sdk.Msg = &MsgSubmitResult{}
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgCreateDispute{}
	_ sdk.Msg = &MsgVoteOnDispute{}
	_ sdk.Msg = &MsgResolveDispute{}
	_ sdk.Msg = &MsgSubmitEvidence{}
	_ sdk.Msg = &MsgAppealSlashing{}
	_ sdk.Msg = &MsgVoteOnAppeal{}
	_ sdk.Msg = &MsgResolveAppeal{}
	_ sdk.Msg = &MsgUpdateGovernanceParams{}
)

// GetSigners implementations - these assume addresses are valid (validated in ValidateBasic)

// GetSigners returns the expected signers for MsgRegisterProvider
func (msg *MsgRegisterProvider) GetSigners() []sdk.AccAddress {
	provider, _ := sdk.AccAddressFromBech32(msg.Provider)
	return []sdk.AccAddress{provider}
}

// GetSigners returns the expected signers for MsgUpdateProvider
func (msg *MsgUpdateProvider) GetSigners() []sdk.AccAddress {
	provider, _ := sdk.AccAddressFromBech32(msg.Provider)
	return []sdk.AccAddress{provider}
}

// GetSigners returns the expected signers for MsgDeactivateProvider
func (msg *MsgDeactivateProvider) GetSigners() []sdk.AccAddress {
	provider, _ := sdk.AccAddressFromBech32(msg.Provider)
	return []sdk.AccAddress{provider}
}

// GetSigners returns the expected signers for MsgSubmitRequest
func (msg *MsgSubmitRequest) GetSigners() []sdk.AccAddress {
	requester, _ := sdk.AccAddressFromBech32(msg.Requester)
	return []sdk.AccAddress{requester}
}

// GetSigners returns the expected signers for MsgCancelRequest
func (msg *MsgCancelRequest) GetSigners() []sdk.AccAddress {
	requester, _ := sdk.AccAddressFromBech32(msg.Requester)
	return []sdk.AccAddress{requester}
}

// GetSigners returns the expected signers for MsgSubmitResult
func (msg *MsgSubmitResult) GetSigners() []sdk.AccAddress {
	provider, _ := sdk.AccAddressFromBech32(msg.Provider)
	return []sdk.AccAddress{provider}
}

// GetSigners returns the expected signers for MsgUpdateParams
func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

// GetSigners returns the expected signers for MsgCreateDispute
func (msg *MsgCreateDispute) GetSigners() []sdk.AccAddress {
	req, _ := sdk.AccAddressFromBech32(msg.Requester)
	return []sdk.AccAddress{req}
}

// GetSigners returns the expected signers for MsgVoteOnDispute
func (msg *MsgVoteOnDispute) GetSigners() []sdk.AccAddress {
	val, _ := sdk.ValAddressFromBech32(msg.Validator)
	return []sdk.AccAddress{sdk.AccAddress(val)}
}

// GetSigners returns the expected signers for MsgResolveDispute
func (msg *MsgResolveDispute) GetSigners() []sdk.AccAddress {
	auth, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{auth}
}

// GetSigners returns the expected signers for MsgSubmitEvidence
func (msg *MsgSubmitEvidence) GetSigners() []sdk.AccAddress {
	submitter, _ := sdk.AccAddressFromBech32(msg.Submitter)
	return []sdk.AccAddress{submitter}
}

// GetSigners returns the expected signers for MsgAppealSlashing
func (msg *MsgAppealSlashing) GetSigners() []sdk.AccAddress {
	provider, _ := sdk.AccAddressFromBech32(msg.Provider)
	return []sdk.AccAddress{provider}
}

// GetSigners returns the expected signers for MsgVoteOnAppeal
func (msg *MsgVoteOnAppeal) GetSigners() []sdk.AccAddress {
	val, _ := sdk.ValAddressFromBech32(msg.Validator)
	return []sdk.AccAddress{sdk.AccAddress(val)}
}

// GetSigners returns the expected signers for MsgResolveAppeal
func (msg *MsgResolveAppeal) GetSigners() []sdk.AccAddress {
	auth, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{auth}
}

// GetSigners returns the expected signers for MsgUpdateGovernanceParams
func (msg *MsgUpdateGovernanceParams) GetSigners() []sdk.AccAddress {
	auth, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{auth}
}

// ValidateBasic performs basic validation of MsgRegisterProvider
func (msg *MsgRegisterProvider) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}

	if err := ValidateMoniker(msg.Moniker); err != nil {
		return fmt.Errorf("invalid moniker: %w", err)
	}

	if err := ValidateEndpoint(msg.Endpoint); err != nil {
		return fmt.Errorf("invalid endpoint: %w", err)
	}

	if msg.Stake.IsNil() || msg.Stake.IsZero() || msg.Stake.IsNegative() {
		return fmt.Errorf("stake must be positive")
	}

	if err := validateComputeSpec(msg.AvailableSpecs); err != nil {
		return fmt.Errorf("invalid specs: %w", err)
	}

	if err := validatePricing(msg.Pricing); err != nil {
		return fmt.Errorf("invalid pricing: %w", err)
	}

	return nil
}

// ValidateBasic performs basic validation of MsgUpdateProvider
func (msg *MsgUpdateProvider) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}

	if msg.AvailableSpecs != nil {
		if err := validateComputeSpec(*msg.AvailableSpecs); err != nil {
			return fmt.Errorf("invalid specs: %w", err)
		}
	}

	if msg.Pricing != nil {
		if err := validatePricing(*msg.Pricing); err != nil {
			return fmt.Errorf("invalid pricing: %w", err)
		}
	}

	return nil
}

// ValidateBasic performs basic validation of MsgDeactivateProvider
func (msg *MsgDeactivateProvider) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}

	return nil
}

// ValidateBasic performs basic validation of MsgSubmitRequest
func (msg *MsgSubmitRequest) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Requester); err != nil {
		return fmt.Errorf("invalid requester address: %w", err)
	}

	if err := validateComputeSpec(msg.Specs); err != nil {
		return fmt.Errorf("invalid specs: %w", err)
	}

	if err := ValidateContainerImage(msg.ContainerImage); err != nil {
		return fmt.Errorf("invalid container image: %w", err)
	}

	if err := ValidateCommand(msg.Command); err != nil {
		return fmt.Errorf("invalid command: %w", err)
	}

	if err := ValidateEnvVars(msg.EnvVars); err != nil {
		return fmt.Errorf("invalid environment variables: %w", err)
	}

	if msg.MaxPayment.IsNil() || msg.MaxPayment.IsZero() || msg.MaxPayment.IsNegative() {
		return fmt.Errorf("max payment must be positive")
	}

	if msg.PreferredProvider != "" {
		if _, err := sdk.AccAddressFromBech32(msg.PreferredProvider); err != nil {
			return fmt.Errorf("invalid preferred provider address: %w", err)
		}
	}

	return nil
}

// ValidateBasic performs basic validation of MsgCancelRequest
func (msg *MsgCancelRequest) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Requester); err != nil {
		return fmt.Errorf("invalid requester address: %w", err)
	}

	if msg.RequestId == 0 {
		return fmt.Errorf("request ID must be greater than 0")
	}

	return nil
}

// ValidateBasic performs basic validation of MsgSubmitResult
func (msg *MsgSubmitResult) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}

	if msg.RequestId == 0 {
		return fmt.Errorf("request ID must be greater than 0")
	}

	if err := ValidateOutputHash(msg.OutputHash); err != nil {
		return fmt.Errorf("invalid output hash: %w", err)
	}

	if err := ValidateOutputURL(msg.OutputUrl); err != nil {
		return fmt.Errorf("invalid output URL: %w", err)
	}

	return nil
}

// ValidateBasic performs basic validation of MsgUpdateParams
func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	return validateParams(msg.Params)
}

// Helper validation functions

func validateComputeSpec(spec ComputeSpec) error {
	if spec.CpuCores == 0 {
		return fmt.Errorf("cpu_cores must be greater than 0")
	}

	if spec.MemoryMb == 0 {
		return fmt.Errorf("memory_mb must be greater than 0")
	}

	if spec.TimeoutSeconds == 0 {
		return fmt.Errorf("timeout_seconds must be greater than 0")
	}

	return nil
}

// ValidateBasic performs basic validation of MsgCreateDispute
func (msg *MsgCreateDispute) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Requester); err != nil {
		return fmt.Errorf("invalid requester address: %w", err)
	}
	if msg.RequestId == 0 {
		return fmt.Errorf("request ID must be greater than 0")
	}
	if msg.DepositAmount.IsNil() || !msg.DepositAmount.IsPositive() {
		return fmt.Errorf("deposit must be positive")
	}
	if msg.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	return nil
}

// ValidateBasic performs basic validation of MsgVoteOnDispute
func (msg *MsgVoteOnDispute) ValidateBasic() error {
	if _, err := sdk.ValAddressFromBech32(msg.Validator); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if msg.DisputeId == 0 {
		return fmt.Errorf("dispute ID must be greater than 0")
	}
	return nil
}

// ValidateBasic performs basic validation of MsgResolveDispute
func (msg *MsgResolveDispute) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.DisputeId == 0 {
		return fmt.Errorf("dispute ID must be greater than 0")
	}
	return nil
}

// ValidateBasic performs basic validation of MsgSubmitEvidence
func (msg *MsgSubmitEvidence) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Submitter); err != nil {
		return fmt.Errorf("invalid submitter address: %w", err)
	}
	if msg.DisputeId == 0 {
		return fmt.Errorf("dispute ID must be greater than 0")
	}
	if len(msg.Data) == 0 {
		return fmt.Errorf("evidence data cannot be empty")
	}
	return nil
}

// ValidateBasic performs basic validation of MsgAppealSlashing
func (msg *MsgAppealSlashing) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}
	if msg.SlashId == 0 {
		return fmt.Errorf("slash ID must be greater than 0")
	}
	if msg.DepositAmount.IsNil() || !msg.DepositAmount.IsPositive() {
		return fmt.Errorf("deposit must be positive")
	}
	if msg.Justification == "" {
		return fmt.Errorf("justification is required")
	}
	return nil
}

// ValidateBasic performs basic validation of MsgVoteOnAppeal
func (msg *MsgVoteOnAppeal) ValidateBasic() error {
	if _, err := sdk.ValAddressFromBech32(msg.Validator); err != nil {
		return fmt.Errorf("invalid validator address: %w", err)
	}
	if msg.AppealId == 0 {
		return fmt.Errorf("appeal ID must be greater than 0")
	}
	return nil
}

// ValidateBasic performs basic validation of MsgResolveAppeal
func (msg *MsgResolveAppeal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	if msg.AppealId == 0 {
		return fmt.Errorf("appeal ID must be greater than 0")
	}
	return nil
}

// ValidateBasic performs basic validation of MsgUpdateGovernanceParams
func (msg *MsgUpdateGovernanceParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}
	return nil
}

func validatePricing(pricing Pricing) error {
	if pricing.CpuPricePerMcoreHour.IsNegative() {
		return fmt.Errorf("cpu_price_per_mcore_hour cannot be negative")
	}

	if pricing.MemoryPricePerMbHour.IsNegative() {
		return fmt.Errorf("memory_price_per_mb_hour cannot be negative")
	}

	if pricing.GpuPricePerHour.IsNegative() {
		return fmt.Errorf("gpu_price_per_hour cannot be negative")
	}

	if pricing.StoragePricePerGbHour.IsNegative() {
		return fmt.Errorf("storage_price_per_gb_hour cannot be negative")
	}

	return nil
}

func validateParams(params Params) error {
	if params.MinProviderStake.IsNil() || params.MinProviderStake.IsNegative() {
		return fmt.Errorf("min_provider_stake must be non-negative")
	}

	if params.VerificationTimeoutSeconds == 0 {
		return fmt.Errorf("verification_timeout_seconds must be greater than 0")
	}

	if params.MaxRequestTimeoutSeconds == 0 {
		return fmt.Errorf("max_request_timeout_seconds must be greater than 0")
	}

	if params.ReputationSlashPercentage > 100 {
		return fmt.Errorf("reputation_slash_percentage cannot exceed 100")
	}

	if params.StakeSlashPercentage > 100 {
		return fmt.Errorf("stake_slash_percentage cannot exceed 100")
	}

	if params.MinReputationScore > 100 {
		return fmt.Errorf("min_reputation_score cannot exceed 100")
	}

	return nil
}
