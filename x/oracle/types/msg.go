package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Message type URLs
const (
	TypeMsgSubmitPrice           = "submit_price"
	TypeMsgDelegateFeedConsent   = "delegate_feed_consent"
	TypeMsgUpdateParams          = "update_params"
	TypeMsgEmergencyPauseOracle  = "emergency_pause_oracle"
	TypeMsgResumeOracle          = "resume_oracle"
)

var (
	_ sdk.Msg = &MsgSubmitPrice{}
	_ sdk.Msg = &MsgDelegateFeedConsent{}
	_ sdk.Msg = &MsgUpdateParams{}
	_ sdk.Msg = &MsgEmergencyPauseOracle{}
	_ sdk.Msg = &MsgResumeOracle{}
)

// NewMsgSubmitPrice creates a new MsgSubmitPrice instance
func NewMsgSubmitPrice(validator string, feeder string, asset string, price math.LegacyDec) *MsgSubmitPrice {
	return &MsgSubmitPrice{
		Validator: validator,
		Feeder:    feeder,
		Asset:     asset,
		Price:     price,
	}
}

const maxAssetLen = 128

// Route implements sdk.Msg
func (msg *MsgSubmitPrice) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg *MsgSubmitPrice) Type() string {
	return TypeMsgSubmitPrice
}

// GetSigners implements sdk.Msg
// Assumes address is valid (validated in ValidateBasic)
func (msg *MsgSubmitPrice) GetSigners() []sdk.AccAddress {
	feeder, _ := sdk.AccAddressFromBech32(msg.Feeder)
	return []sdk.AccAddress{feeder}
}

// GetSignBytes implements sdk.Msg
func (msg *MsgSubmitPrice) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg
func (msg *MsgSubmitPrice) ValidateBasic() error {
	if _, err := sdk.ValAddressFromBech32(msg.Validator); err != nil {
		return ErrValidatorNotFound.Wrapf("invalid validator address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Feeder); err != nil {
		return ErrFeederNotAuthorized.Wrapf("invalid feeder address: %s", err)
	}

	if msg.Asset == "" {
		return ErrInvalidAsset.Wrap("asset cannot be empty")
	}
	if len(msg.Asset) > maxAssetLen {
		return ErrInvalidAsset.Wrapf("asset too long (max %d chars)", maxAssetLen)
	}

	if msg.Price.IsNil() || msg.Price.LTE(math.LegacyZeroDec()) {
		return ErrInvalidPrice.Wrap("price must be positive")
	}

	return nil
}

// NewMsgDelegateFeedConsent creates a new MsgDelegateFeedConsent instance
func NewMsgDelegateFeedConsent(validator string, delegate string) *MsgDelegateFeedConsent {
	return &MsgDelegateFeedConsent{
		Validator: validator,
		Delegate:  delegate,
	}
}

// Route implements sdk.Msg
func (msg *MsgDelegateFeedConsent) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg *MsgDelegateFeedConsent) Type() string {
	return TypeMsgDelegateFeedConsent
}

// GetSigners implements sdk.Msg
// Assumes address is valid (validated in ValidateBasic)
func (msg *MsgDelegateFeedConsent) GetSigners() []sdk.AccAddress {
	validator, _ := sdk.ValAddressFromBech32(msg.Validator)
	return []sdk.AccAddress{sdk.AccAddress(validator)}
}

// GetSignBytes implements sdk.Msg
func (msg *MsgDelegateFeedConsent) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg
func (msg *MsgDelegateFeedConsent) ValidateBasic() error {
	if _, err := sdk.ValAddressFromBech32(msg.Validator); err != nil {
		return ErrValidatorNotFound.Wrapf("invalid validator address: %s", err)
	}

	if _, err := sdk.AccAddressFromBech32(msg.Delegate); err != nil {
		return ErrFeederNotAuthorized.Wrapf("invalid delegate address: %s", err)
	}

	return nil
}

// NewMsgUpdateParams creates a new MsgUpdateParams instance
func NewMsgUpdateParams(authority string, params Params) *MsgUpdateParams {
	return &MsgUpdateParams{
		Authority: authority,
		Params:    params,
	}
}

// Route implements sdk.Msg
func (msg *MsgUpdateParams) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg *MsgUpdateParams) Type() string {
	return TypeMsgUpdateParams
}

// GetSigners implements sdk.Msg
// Assumes address is valid (validated in ValidateBasic)
func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

// GetSignBytes implements sdk.Msg
func (msg *MsgUpdateParams) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg
func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrInvalidAsset.Wrapf("invalid authority address: %s", err)
	}
	if msg.Authority != DefaultAuthority() {
		return ErrInvalidAsset.Wrapf("invalid authority: expected %s, got %s", DefaultAuthority(), msg.Authority)
	}

	if msg.Params.VotePeriod == 0 {
		return ErrInvalidVotePeriod.Wrap("vote period must be positive")
	}

	if msg.Params.VoteThreshold.IsNil() || msg.Params.VoteThreshold.LTE(math.LegacyZeroDec()) || msg.Params.VoteThreshold.GT(math.LegacyOneDec()) {
		return ErrInvalidThreshold.Wrap("vote threshold must be between 0 and 1")
	}

	if msg.Params.SlashFraction.IsNil() || msg.Params.SlashFraction.LT(math.LegacyZeroDec()) || msg.Params.SlashFraction.GT(math.LegacyOneDec()) {
		return ErrInvalidSlashFraction.Wrap("slash fraction must be between 0 and 1")
	}

	return nil
}

// NewMsgEmergencyPauseOracle creates a new MsgEmergencyPauseOracle instance
func NewMsgEmergencyPauseOracle(signer string, reason string) *MsgEmergencyPauseOracle {
	return &MsgEmergencyPauseOracle{
		Signer: signer,
		Reason: reason,
	}
}

// Route implements sdk.Msg
func (msg *MsgEmergencyPauseOracle) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg *MsgEmergencyPauseOracle) Type() string {
	return TypeMsgEmergencyPauseOracle
}

// GetSigners implements sdk.Msg
func (msg *MsgEmergencyPauseOracle) GetSigners() []sdk.AccAddress {
	signer, _ := sdk.AccAddressFromBech32(msg.Signer)
	return []sdk.AccAddress{signer}
}

// GetSignBytes implements sdk.Msg
func (msg *MsgEmergencyPauseOracle) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg
func (msg *MsgEmergencyPauseOracle) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Signer); err != nil {
		return ErrUnauthorizedPause.Wrapf("invalid signer address: %s", err)
	}

	if msg.Reason == "" {
		return ErrUnauthorizedPause.Wrap("pause reason cannot be empty")
	}

	if len(msg.Reason) > 512 {
		return ErrUnauthorizedPause.Wrap("pause reason too long (max 512 chars)")
	}

	return nil
}

// NewMsgResumeOracle creates a new MsgResumeOracle instance
func NewMsgResumeOracle(authority string, reason string) *MsgResumeOracle {
	return &MsgResumeOracle{
		Authority: authority,
		Reason:    reason,
	}
}

// Route implements sdk.Msg
func (msg *MsgResumeOracle) Route() string {
	return RouterKey
}

// Type implements sdk.Msg
func (msg *MsgResumeOracle) Type() string {
	return TypeMsgResumeOracle
}

// GetSigners implements sdk.Msg
func (msg *MsgResumeOracle) GetSigners() []sdk.AccAddress {
	authority, _ := sdk.AccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{authority}
}

// GetSignBytes implements sdk.Msg
func (msg *MsgResumeOracle) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements sdk.Msg
func (msg *MsgResumeOracle) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return ErrUnauthorizedResume.Wrapf("invalid authority address: %s", err)
	}

	if msg.Reason == "" {
		return ErrUnauthorizedResume.Wrap("resume reason cannot be empty")
	}

	if len(msg.Reason) > 512 {
		return ErrUnauthorizedResume.Wrap("resume reason too long (max 512 chars)")
	}

	return nil
}
