package keeper

import (
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
)

// TASK 73: IBC timeout handling for oracle packets

// OnTimeoutPricePacket handles timeout for cross-chain price feed packets
func (k Keeper) OnTimeoutPricePacket(ctx sdk.Context, packet channeltypes.Packet) error {
	var packetData map[string]interface{}
	if err := json.Unmarshal(packet.Data, &packetData); err != nil {
		return sdkerrors.Wrap(err, "failed to unmarshal packet data")
	}

	asset, ok := packetData["asset"].(string)
	if !ok {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidType, "missing asset in packet")
	}

	ctx.Logger().Warn("cross-chain price feed timed out",
		"asset", asset,
		"packet_sequence", packet.Sequence,
		"channel", packet.SourceChannel,
	)

	// Emit timeout event for monitoring
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"cross_chain_price_timeout",
			sdk.NewAttribute("asset", asset),
			sdk.NewAttribute("packet_sequence", fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute("channel", packet.SourceChannel),
		),
	)

	// Note: No refund needed for oracle price feeds as they don't involve escrow
	return nil
}

// OnAcknowledgementPricePacket handles acknowledgements for price feed packets
func (k Keeper) OnAcknowledgementPricePacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	ack channeltypes.Acknowledgement,
) error {
	var ackData map[string]interface{}
	if err := json.Unmarshal(ack.GetResult(), &ackData); err != nil {
		return sdkerrors.Wrap(err, "failed to unmarshal acknowledgement")
	}

	// Check for error acknowledgement
	if errMsg, hasErr := ackData["error"].(string); hasErr {
		ctx.Logger().Error("cross-chain price feed failed", "error", errMsg)

		var packetData map[string]interface{}
		if err := json.Unmarshal(packet.Data, &packetData); err == nil {
			if asset, ok := packetData["asset"].(string); ok {
				ctx.EventManager().EmitEvent(
					sdk.NewEvent(
						"cross_chain_price_failed",
						sdk.NewAttribute("asset", asset),
						sdk.NewAttribute("error", errMsg),
					),
				)
			}
		}
		return nil
	}

	// Success case - log remote price acknowledgement
	if priceStr, hasPrice := ackData["price"].(string); hasPrice {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"cross_chain_price_acknowledged",
				sdk.NewAttribute("price", priceStr),
			),
		)
	}

	return nil
}
