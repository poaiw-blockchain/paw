package keeper

import (
	"encoding/json"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// TASK 72: IBC timeout handling for DEX packets

// OnTimeoutSwapPacket handles timeout for cross-chain swap packets
func (k Keeper) OnTimeoutSwapPacket(ctx sdk.Context, packet channeltypes.Packet) error {
	var packetData map[string]interface{}
	if err := json.Unmarshal(packet.Data, &packetData); err != nil {
		return errorsmod.Wrap(err, "failed to unmarshal packet data")
	}

	swapID, ok := packetData["swap_id"].(string)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrInvalidType, "missing swap_id in packet")
	}

	ctx.Logger().Error("cross-chain swap timed out",
		"swap_id", swapID,
		"packet_sequence", packet.Sequence,
		"channel", packet.SourceChannel,
	)

	// Refund swap tokens to user
	if userAddr, hasUser := packetData["user"].(string); hasUser {
		if amountStr, hasAmount := packetData["amount"].(string); hasAmount {
			user, err := sdk.AccAddressFromBech32(userAddr)
			if err == nil {
				amount, _ := sdk.ParseCoinsNormalized(amountStr)

				// Refund from module account
				if err := k.bankKeeper.SendCoinsFromModuleToAccount(
					ctx, "dex", user, amount,
				); err != nil {
					ctx.Logger().Error("failed to refund swap", "error", err)
					return err
				}

				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						types.EventTypeDexCrossChainSwapTimeout,
						sdk.NewAttribute(types.AttributeKeySwapID, swapID),
						sdk.NewAttribute(types.AttributeKeyUserAddress, userAddr),
						sdk.NewAttribute(types.AttributeKeyAmount, amount.String()),
					),
				)
			}
		}
	}

	return nil
}

// OnAcknowledgementSwapPacket handles acknowledgements for swap packets
func (k Keeper) OnAcknowledgementSwapPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	ack channeltypes.Acknowledgement,
) error {
	var ackData map[string]interface{}
	if err := json.Unmarshal(ack.GetResult(), &ackData); err != nil {
		return errorsmod.Wrap(err, "failed to unmarshal acknowledgement")
	}

	// Check for error acknowledgement
	if errMsg, hasErr := ackData["error"].(string); hasErr {
		ctx.Logger().Error("cross-chain swap failed", "error", errMsg)

		// Refund tokens on failure
		var packetData map[string]interface{}
		if err := json.Unmarshal(packet.Data, &packetData); err == nil {
			if swapID, ok := packetData["swap_id"].(string); ok {
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						types.EventTypeDexCrossChainSwapFailed,
						sdk.NewAttribute(types.AttributeKeySwapID, swapID),
						sdk.NewAttribute(types.AttributeKeyError, errMsg),
					),
				)
			}
		}
	}

	return nil
}
