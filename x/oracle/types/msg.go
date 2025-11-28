package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Message type URLs
const (
	TypeMsgSubmitPrice         = "submit_price"
	TypeMsgDelegateFeedConsent = "delegate_feed_consent"
	TypeMsgUpdateParams        = "update_params"
)

var (
	_ sdk.Msg = &MsgSubmitPrice{}
	_ sdk.Msg = &MsgDelegateFeedConsent{}
	_ sdk.Msg = &MsgUpdateParams{}
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
