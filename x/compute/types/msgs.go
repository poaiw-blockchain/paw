package types

import (
	"fmt"

	"cosmossdk.io/math"
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
	TypeMsgRegisterSigningKey = "register_signing_key"
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
	_ sdk.Msg = &MsgRegisterSigningKey{}
)

// MsgRegisterSigningKey is the message for registering a provider's signing key.
// SEC-2 FIX: This message MUST be submitted by providers BEFORE they can submit results.
// This prevents trust-on-first-use attacks where an attacker submits a result with their
// own key before the legitimate provider registers their key.
type MsgRegisterSigningKey struct {
	// Provider is the bech32 address of the provider registering their key
	Provider string `json:"provider"`
	// PublicKey is the Ed25519 public key (32 bytes) to register
	PublicKey []byte `json:"public_key"`
	// OldKeySignature is required when rotating an existing key.
	// Must be a signature of "ROTATE_KEY:" + provider address + new public key
	// signed with the existing registered key.
	OldKeySignature []byte `json:"old_key_signature,omitempty"`
}

// MsgRegisterSigningKeyResponse is the response for MsgRegisterSigningKey.
type MsgRegisterSigningKeyResponse struct{}

// ProtoMessage implements proto.Message for MsgRegisterSigningKey
func (msg *MsgRegisterSigningKey) ProtoMessage() {}

// Reset implements proto.Message for MsgRegisterSigningKey
func (msg *MsgRegisterSigningKey) Reset() { *msg = MsgRegisterSigningKey{} }

// String implements proto.Message for MsgRegisterSigningKey
func (msg *MsgRegisterSigningKey) String() string {
	return fmt.Sprintf("MsgRegisterSigningKey{Provider: %s, PublicKey: %x}", msg.Provider, msg.PublicKey)
}

// ProtoMessage implements proto.Message for MsgRegisterSigningKeyResponse
func (msg *MsgRegisterSigningKeyResponse) ProtoMessage() {}

// Reset implements proto.Message for MsgRegisterSigningKeyResponse
func (msg *MsgRegisterSigningKeyResponse) Reset() { *msg = MsgRegisterSigningKeyResponse{} }

// String implements proto.Message for MsgRegisterSigningKeyResponse
func (msg *MsgRegisterSigningKeyResponse) String() string {
	return "MsgRegisterSigningKeyResponse{}"
}

const (
	maxGovernanceEvidenceSizeLimit = 50 * 1024 * 1024 // 50 MB absolute ceiling

	maxDisputeReasonLength       = 1024
	maxEvidenceDescriptionLength = 2048
	maxAppealJustificationLength = 2048
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

// GetSigners returns the expected signers for MsgRegisterSigningKey
func (msg *MsgRegisterSigningKey) GetSigners() []sdk.AccAddress {
	provider, _ := sdk.AccAddressFromBech32(msg.Provider)
	return []sdk.AccAddress{provider}
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
	if msg.Authority != DefaultAuthority() {
		return fmt.Errorf("invalid authority: expected %s, got %s", DefaultAuthority(), msg.Authority)
	}

	return validateParams(msg.Params)
}

// Helper validation functions

// ComputeSpec validation constants
// SEC-3.2: Upper bounds prevent resource abuse and unrealistic requests
const (
	MaxCPUMillicores  = 256000        // 256 cores in millicores
	MaxMemoryMB       = 512 * 1024    // 512 GB in MB
	MaxStorageGB      = 10 * 1024     // 10 TB in GB
	MaxGPUCount       = 16            // 16 GPUs maximum
	MaxTimeoutSeconds = 7 * 24 * 3600 // 7 days maximum timeout
)

func validateComputeSpec(spec ComputeSpec) error {
	// Minimum validation
	if spec.CpuCores == 0 {
		return fmt.Errorf("cpu_cores must be greater than 0")
	}

	if spec.MemoryMb == 0 {
		return fmt.Errorf("memory_mb must be greater than 0")
	}

	if spec.TimeoutSeconds == 0 {
		return fmt.Errorf("timeout_seconds must be greater than 0")
	}

	// SEC-3.2: Upper bounds validation
	if spec.CpuCores > MaxCPUMillicores {
		return fmt.Errorf("cpu_cores exceeds maximum (%d millicores)", MaxCPUMillicores)
	}

	if spec.MemoryMb > MaxMemoryMB {
		return fmt.Errorf("memory_mb exceeds maximum (%d MB = 512 GB)", MaxMemoryMB)
	}

	if spec.StorageGb > MaxStorageGB {
		return fmt.Errorf("storage_gb exceeds maximum (%d GB = 10 TB)", MaxStorageGB)
	}

	if spec.GpuCount > MaxGPUCount {
		return fmt.Errorf("gpu_count exceeds maximum (%d)", MaxGPUCount)
	}

	if spec.TimeoutSeconds > MaxTimeoutSeconds {
		return fmt.Errorf("timeout_seconds exceeds maximum (%d seconds = 7 days)", MaxTimeoutSeconds)
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
	if len(msg.Reason) > maxDisputeReasonLength {
		return fmt.Errorf("reason exceeds max length (%d characters)", maxDisputeReasonLength)
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
	if msg.Authority != DefaultAuthority() {
		return fmt.Errorf("invalid authority: expected %s, got %s", DefaultAuthority(), msg.Authority)
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
	if uint64(len(msg.Data)) > maxGovernanceEvidenceSizeLimit {
		return fmt.Errorf("evidence data exceeds hard limit (%d bytes)", maxGovernanceEvidenceSizeLimit)
	}
	if len(msg.Description) > maxEvidenceDescriptionLength {
		return fmt.Errorf("evidence description exceeds max length (%d characters)", maxEvidenceDescriptionLength)
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
	if len(msg.Justification) > maxAppealJustificationLength {
		return fmt.Errorf("justification exceeds max length (%d characters)", maxAppealJustificationLength)
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
	if msg.Authority != DefaultAuthority() {
		return fmt.Errorf("invalid authority: expected %s, got %s", DefaultAuthority(), msg.Authority)
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
	if msg.Authority != DefaultAuthority() {
		return fmt.Errorf("invalid authority: expected %s, got %s", DefaultAuthority(), msg.Authority)
	}

	if err := validateGovernanceParams(msg.Params); err != nil {
		return err
	}

	return nil
}

// ValidateBasic performs basic validation of MsgRegisterSigningKey
// SEC-2 FIX: Validates the signing key registration message.
func (msg *MsgRegisterSigningKey) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}

	// Validate public key length (Ed25519 = 32 bytes)
	if len(msg.PublicKey) != 32 {
		return fmt.Errorf("invalid public key size: expected 32 bytes, got %d", len(msg.PublicKey))
	}

	// Check for all-zeros key (invalid)
	allZeros := true
	for _, b := range msg.PublicKey {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		return fmt.Errorf("invalid public key: all zeros")
	}

	// OldKeySignature is optional (only required for key rotation)
	// If provided, validate its length (Ed25519 signature = 64 bytes)
	if len(msg.OldKeySignature) > 0 && len(msg.OldKeySignature) != 64 {
		return fmt.Errorf("invalid old key signature size: expected 64 bytes, got %d", len(msg.OldKeySignature))
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

func validateGovernanceParams(params GovernanceParams) error {
	if params.DisputeDeposit.IsNil() || params.DisputeDeposit.IsNegative() {
		return fmt.Errorf("dispute_deposit must be non-negative")
	}

	if params.EvidencePeriodSeconds == 0 {
		return fmt.Errorf("evidence_period_seconds must be greater than 0")
	}

	if params.VotingPeriodSeconds == 0 {
		return fmt.Errorf("voting_period_seconds must be greater than 0")
	}

	if params.QuorumPercentage.LT(math.LegacyZeroDec()) || params.QuorumPercentage.GT(math.LegacyOneDec()) {
		return fmt.Errorf("quorum_percentage must be between 0 and 1")
	}

	if params.ConsensusThreshold.LTE(math.LegacyZeroDec()) || params.ConsensusThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("consensus_threshold must be between 0 and 1")
	}

	if params.SlashPercentage.LT(math.LegacyZeroDec()) || params.SlashPercentage.GT(math.LegacyOneDec()) {
		return fmt.Errorf("slash_percentage must be between 0 and 1")
	}

	if params.AppealDepositPercentage.LT(math.LegacyZeroDec()) || params.AppealDepositPercentage.GT(math.LegacyOneDec()) {
		return fmt.Errorf("appeal_deposit_percentage must be between 0 and 1")
	}

	if params.MaxEvidenceSize == 0 {
		return fmt.Errorf("max_evidence_size must be greater than 0")
	}

	if params.MaxEvidenceSize > maxGovernanceEvidenceSizeLimit {
		return fmt.Errorf("max_evidence_size exceeds hard limit (%d bytes)", maxGovernanceEvidenceSizeLimit)
	}

	return nil
}
