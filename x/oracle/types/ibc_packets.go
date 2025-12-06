package types

import (
	"encoding/json"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IBC Packet Types for Oracle Module
//
// This file defines all IBC packet structures for cross-chain oracle operations.

const (
	// Oracle IBC version
	IBCVersion = "paw-oracle-1"

	// Packet types
	SubscribePricesType = "subscribe_prices"
	QueryPriceType      = "query_price"
	PriceUpdateType     = "price_update"
	OracleHeartbeatType = "oracle_heartbeat"
)

// IBCPacketData is the base interface for all Oracle IBC packets
type IBCPacketData interface {
	ValidateBasic() error
	GetType() string
}

// SubscribePricesPacketData subscribes to price feeds
type SubscribePricesPacketData struct {
	Type           string   `json:"type"`
	Nonce          uint64   `json:"nonce"`
	Timestamp      int64    `json:"timestamp"`
	Symbols        []string `json:"symbols"`
	UpdateInterval uint64   `json:"update_interval"` // seconds
	Subscriber     string   `json:"subscriber"`
}

func (p SubscribePricesPacketData) ValidateBasic() error {
	if p.Type != SubscribePricesType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.Nonce == 0 {
		return errors.Wrap(ErrInvalidPacket, "nonce must be greater than zero")
	}
	if p.Timestamp <= 0 {
		return errors.Wrap(ErrInvalidPacket, "timestamp must be positive")
	}
	if len(p.Symbols) == 0 {
		return errors.Wrap(ErrInvalidPacket, "symbols cannot be empty")
	}
	if p.UpdateInterval == 0 {
		return errors.Wrap(ErrInvalidPacket, "update interval must be positive")
	}
	if _, err := sdk.AccAddressFromBech32(p.Subscriber); err != nil {
		return errors.Wrapf(ErrInvalidPacket, "invalid subscriber address: %s", err)
	}
	return nil
}

func (p SubscribePricesPacketData) GetType() string {
	return p.Type
}

func (p SubscribePricesPacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// SubscribePricesAcknowledgement acknowledges subscription
type SubscribePricesAcknowledgement struct {
	Nonce             uint64   `json:"nonce"`
	Success           bool     `json:"success"`
	SubscribedSymbols []string `json:"subscribed_symbols,omitempty"`
	SubscriptionID    string   `json:"subscription_id,omitempty"`
	Error             string   `json:"error,omitempty"`
}

func (a SubscribePricesAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// QueryPricePacketData queries current price
type QueryPricePacketData struct {
	Type      string `json:"type"`
	Nonce     uint64 `json:"nonce"`
	Timestamp int64  `json:"timestamp"`
	Symbol    string `json:"symbol"`
	Sender    string `json:"sender"`
}

func (p QueryPricePacketData) ValidateBasic() error {
	if p.Type != QueryPriceType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.Nonce == 0 {
		return errors.Wrap(ErrInvalidPacket, "nonce must be greater than zero")
	}
	if p.Timestamp <= 0 {
		return errors.Wrap(ErrInvalidPacket, "timestamp must be positive")
	}
	if p.Symbol == "" {
		return errors.Wrap(ErrInvalidPacket, "symbol cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(p.Sender); err != nil {
		return errors.Wrapf(ErrInvalidPacket, "invalid sender address: %s", err)
	}
	return nil
}

func (p QueryPricePacketData) GetType() string {
	return p.Type
}

func (p QueryPricePacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// PriceData represents oracle price information
type PriceData struct {
	Symbol      string         `json:"symbol"`
	Price       math.LegacyDec `json:"price"`
	Volume24h   math.Int       `json:"volume_24h"`
	Timestamp   int64          `json:"timestamp"`
	Confidence  math.LegacyDec `json:"confidence"`
	OracleCount uint32         `json:"oracle_count"`
}

// QueryPriceAcknowledgement returns price data
type QueryPriceAcknowledgement struct {
	Nonce     uint64    `json:"nonce"`
	Success   bool      `json:"success"`
	PriceData PriceData `json:"price_data,omitempty"`
	Error     string    `json:"error,omitempty"`
}

func (a QueryPriceAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// PriceUpdatePacketData broadcasts price updates
type PriceUpdatePacketData struct {
	Type      string      `json:"type"`
	Nonce     uint64      `json:"nonce"`
	Prices    []PriceData `json:"prices"`
	Timestamp int64       `json:"timestamp"`
	Source    string      `json:"source"`
}

func (p PriceUpdatePacketData) ValidateBasic() error {
	if p.Type != PriceUpdateType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.Nonce == 0 {
		return errors.Wrap(ErrInvalidPacket, "nonce must be greater than zero")
	}
	if len(p.Prices) == 0 {
		return errors.Wrap(ErrInvalidPacket, "prices cannot be empty")
	}
	for i, price := range p.Prices {
		if price.Symbol == "" {
			return errors.Wrapf(ErrInvalidPacket, "price %d: symbol cannot be empty", i)
		}
		if price.Price.IsNil() || !price.Price.IsPositive() {
			return errors.Wrapf(ErrInvalidPacket, "price %d: price must be positive", i)
		}
		if price.Confidence.IsNil() || price.Confidence.IsNegative() || price.Confidence.GT(math.LegacyOneDec()) {
			return errors.Wrapf(ErrInvalidPacket, "price %d: confidence must be between 0 and 1", i)
		}
	}
	return nil
}

func (p PriceUpdatePacketData) GetType() string {
	return p.Type
}

func (p PriceUpdatePacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// PriceUpdateAcknowledgement is returned after handling a price update packet
type PriceUpdateAcknowledgement struct {
	Nonce   uint64 `json:"nonce"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (a PriceUpdateAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// OracleHeartbeatPacketData for liveness monitoring
type OracleHeartbeatPacketData struct {
	Type          string `json:"type"`
	Nonce         uint64 `json:"nonce"`
	ChainID       string `json:"chain_id"`
	Timestamp     int64  `json:"timestamp"`
	ActiveOracles uint32 `json:"active_oracles"`
	BlockHeight   int64  `json:"block_height"`
}

func (p OracleHeartbeatPacketData) ValidateBasic() error {
	if p.Type != OracleHeartbeatType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.Nonce == 0 {
		return errors.Wrap(ErrInvalidPacket, "nonce must be greater than zero")
	}
	if p.ChainID == "" {
		return errors.Wrap(ErrInvalidPacket, "chain ID cannot be empty")
	}
	if p.Timestamp <= 0 {
		return errors.Wrap(ErrInvalidPacket, "timestamp must be positive")
	}
	return nil
}

func (p OracleHeartbeatPacketData) GetType() string {
	return p.Type
}

func (p OracleHeartbeatPacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// OracleHeartbeatAcknowledgement is used for heartbeat back-replies
type OracleHeartbeatAcknowledgement struct {
	Nonce   uint64 `json:"nonce"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

func (a OracleHeartbeatAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// ParsePacketData parses IBC packet data based on type
func ParsePacketData(data []byte) (IBCPacketData, error) {
	var basePacket struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(data, &basePacket); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal packet data")
	}

	switch basePacket.Type {
	case SubscribePricesType:
		var packet SubscribePricesPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal subscribe prices packet")
		}
		return packet, nil

	case QueryPriceType:
		var packet QueryPricePacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal query price packet")
		}
		return packet, nil

	case PriceUpdateType:
		var packet PriceUpdatePacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal price update packet")
		}
		return packet, nil

	case OracleHeartbeatType:
		var packet OracleHeartbeatPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal oracle heartbeat packet")
		}
		return packet, nil

	default:
		return nil, errors.Wrapf(ErrInvalidPacket, "unknown packet type: %s", basePacket.Type)
	}
}

var (
	ErrInvalidPacket = errors.Register(ModuleName, 2001, "invalid IBC packet")
)
