package types

import (
	"encoding/json"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IBC Packet Types for DEX Module
//
// This file defines all IBC packet structures for cross-chain DEX operations.
// Packets are serialized as JSON for IBC transmission.

const (
	// DEX IBC version
	IBCVersion = "paw-dex-1"

	// Packet types
	QueryPoolsType     = "query_pools"
	ExecuteSwapType    = "execute_swap"
	CrossChainSwapType = "cross_chain_swap"
	PoolUpdateType     = "pool_update"
)

var (
	ErrInvalidPacket = errors.Register(ModuleName, 1500, "invalid packet")
)

// IBCPacketData is the base interface for all DEX IBC packets
type IBCPacketData interface {
	ValidateBasic() error
	GetType() string
}

// QueryPoolsPacketData requests pool information from remote chain
type QueryPoolsPacketData struct {
	Type   string `json:"type"`
	TokenA string `json:"token_a"`
	TokenB string `json:"token_b"`
}

func (p QueryPoolsPacketData) ValidateBasic() error {
	if p.Type != QueryPoolsType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.TokenA == "" {
		return errors.Wrap(ErrInvalidPacket, "token A cannot be empty")
	}
	if p.TokenB == "" {
		return errors.Wrap(ErrInvalidPacket, "token B cannot be empty")
	}
	return nil
}

func (p QueryPoolsPacketData) GetType() string {
	return p.Type
}

func (p QueryPoolsPacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// QueryPoolsAcknowledgement is the response to pool query
type QueryPoolsAcknowledgement struct {
	Success bool       `json:"success"`
	Pools   []PoolInfo `json:"pools,omitempty"`
	Error   string     `json:"error,omitempty"`
}

type PoolInfo struct {
	PoolID      string   `json:"pool_id"`
	TokenA      string   `json:"token_a"`
	TokenB      string   `json:"token_b"`
	ReserveA    math.Int `json:"reserve_a"`
	ReserveB    math.Int `json:"reserve_b"`
	SwapFee     math.LegacyDec  `json:"swap_fee"`
	TotalShares math.Int `json:"total_shares"`
}

func (a QueryPoolsAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// ExecuteSwapPacketData executes a swap on remote chain
type ExecuteSwapPacketData struct {
	Type         string   `json:"type"`
	PoolID       string   `json:"pool_id"`
	TokenIn      string   `json:"token_in"`
	TokenOut     string   `json:"token_out"`
	AmountIn     math.Int `json:"amount_in"`
	MinAmountOut math.Int `json:"min_amount_out"`
	Sender       string   `json:"sender"`
	Receiver     string   `json:"receiver"`
	Timeout      uint64   `json:"timeout"`
}

func (p ExecuteSwapPacketData) ValidateBasic() error {
	if p.Type != ExecuteSwapType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.PoolID == "" {
		return errors.Wrap(ErrInvalidPacket, "pool ID cannot be empty")
	}
	if p.TokenIn == "" {
		return errors.Wrap(ErrInvalidPacket, "token in cannot be empty")
	}
	if p.TokenOut == "" {
		return errors.Wrap(ErrInvalidPacket, "token out cannot be empty")
	}
	if p.AmountIn.IsNil() || !p.AmountIn.IsPositive() {
		return errors.Wrap(ErrInvalidPacket, "amount in must be positive")
	}
	if p.MinAmountOut.IsNil() || p.MinAmountOut.IsNegative() {
		return errors.Wrap(ErrInvalidPacket, "min amount out cannot be negative")
	}
	if _, err := sdk.AccAddressFromBech32(p.Sender); err != nil {
		return errors.Wrapf(ErrInvalidPacket, "invalid sender address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(p.Receiver); err != nil {
		return errors.Wrapf(ErrInvalidPacket, "invalid receiver address: %s", err)
	}
	return nil
}

func (p ExecuteSwapPacketData) GetType() string {
	return p.Type
}

func (p ExecuteSwapPacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// ExecuteSwapAcknowledgement is the response to swap execution
type ExecuteSwapAcknowledgement struct {
	Success   bool     `json:"success"`
	AmountOut math.Int `json:"amount_out,omitempty"`
	SwapFee   math.Int `json:"swap_fee,omitempty"`
	Error     string   `json:"error,omitempty"`
}

func (a ExecuteSwapAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// CrossChainSwapPacketData performs multi-hop swap across chains
type CrossChainSwapPacketData struct {
	Type      string     `json:"type"`
	Route     []SwapHop  `json:"route"`
	Sender    string     `json:"sender"`
	Receiver  string     `json:"receiver"`
	AmountIn  math.Int   `json:"amount_in"`
	MinOut    math.Int   `json:"min_out"`
	Timeout   uint64     `json:"timeout"`
}

type SwapHop struct {
	ChainID      string   `json:"chain_id"`
	PoolID       string   `json:"pool_id"`
	TokenIn      string   `json:"token_in"`
	TokenOut     string   `json:"token_out"`
	MinAmountOut math.Int `json:"min_amount_out"`
}

func (p CrossChainSwapPacketData) ValidateBasic() error {
	if p.Type != CrossChainSwapType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if len(p.Route) == 0 {
		return errors.Wrap(ErrInvalidPacket, "route cannot be empty")
	}
	if p.AmountIn.IsNil() || !p.AmountIn.IsPositive() {
		return errors.Wrap(ErrInvalidPacket, "amount in must be positive")
	}
	if p.MinOut.IsNil() || p.MinOut.IsNegative() {
		return errors.Wrap(ErrInvalidPacket, "min out cannot be negative")
	}
	if _, err := sdk.AccAddressFromBech32(p.Sender); err != nil {
		return errors.Wrapf(ErrInvalidPacket, "invalid sender address: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(p.Receiver); err != nil {
		return errors.Wrapf(ErrInvalidPacket, "invalid receiver address: %s", err)
	}

	// Validate each hop
	for i, hop := range p.Route {
		if hop.ChainID == "" {
			return errors.Wrapf(ErrInvalidPacket, "hop %d: chain ID cannot be empty", i)
		}
		if hop.PoolID == "" {
			return errors.Wrapf(ErrInvalidPacket, "hop %d: pool ID cannot be empty", i)
		}
		if hop.TokenIn == "" {
			return errors.Wrapf(ErrInvalidPacket, "hop %d: token in cannot be empty", i)
		}
		if hop.TokenOut == "" {
			return errors.Wrapf(ErrInvalidPacket, "hop %d: token out cannot be empty", i)
		}
	}

	return nil
}

func (p CrossChainSwapPacketData) GetType() string {
	return p.Type
}

func (p CrossChainSwapPacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// CrossChainSwapAcknowledgement is the response to cross-chain swap
type CrossChainSwapAcknowledgement struct {
	Success      bool     `json:"success"`
	FinalAmount  math.Int `json:"final_amount,omitempty"`
	HopsExecuted int      `json:"hops_executed,omitempty"`
	TotalFees    math.Int `json:"total_fees,omitempty"`
	Error        string   `json:"error,omitempty"`
}

func (a CrossChainSwapAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// PoolUpdatePacketData broadcasts pool state changes
type PoolUpdatePacketData struct {
	Type        string   `json:"type"`
	PoolID      string   `json:"pool_id"`
	ReserveA    math.Int `json:"reserve_a"`
	ReserveB    math.Int `json:"reserve_b"`
	TotalShares math.Int `json:"total_shares"`
	Timestamp   int64    `json:"timestamp"`
}

func (p PoolUpdatePacketData) ValidateBasic() error {
	if p.Type != PoolUpdateType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.PoolID == "" {
		return errors.Wrap(ErrInvalidPacket, "pool ID cannot be empty")
	}
	if p.ReserveA.IsNil() || p.ReserveA.IsNegative() {
		return errors.Wrap(ErrInvalidPacket, "reserve A cannot be negative")
	}
	if p.ReserveB.IsNil() || p.ReserveB.IsNegative() {
		return errors.Wrap(ErrInvalidPacket, "reserve B cannot be negative")
	}
	return nil
}

func (p PoolUpdatePacketData) GetType() string {
	return p.Type
}

func (p PoolUpdatePacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
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
	case QueryPoolsType:
		var packet QueryPoolsPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal query pools packet")
		}
		return packet, nil

	case ExecuteSwapType:
		var packet ExecuteSwapPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal execute swap packet")
		}
		return packet, nil

	case CrossChainSwapType:
		var packet CrossChainSwapPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal cross-chain swap packet")
		}
		return packet, nil

	case PoolUpdateType:
		var packet PoolUpdatePacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal pool update packet")
		}
		return packet, nil

	default:
		return nil, errors.Wrapf(ErrInvalidPacket, "unknown packet type: %s", basePacket.Type)
	}
}

// NewQueryPoolsPacket creates a new query pools packet
func NewQueryPoolsPacket(tokenA, tokenB string) QueryPoolsPacketData {
	return QueryPoolsPacketData{
		Type:   QueryPoolsType,
		TokenA: tokenA,
		TokenB: tokenB,
	}
}

// NewExecuteSwapPacket creates a new execute swap packet
func NewExecuteSwapPacket(
	poolID string,
	tokenIn string,
	tokenOut string,
	amountIn math.Int,
	minAmountOut math.Int,
	sender string,
	receiver string,
	timeout uint64,
) ExecuteSwapPacketData {
	return ExecuteSwapPacketData{
		Type:         ExecuteSwapType,
		PoolID:       poolID,
		TokenIn:      tokenIn,
		TokenOut:     tokenOut,
		AmountIn:     amountIn,
		MinAmountOut: minAmountOut,
		Sender:       sender,
		Receiver:     receiver,
		Timeout:      timeout,
	}
}

// NewCrossChainSwapPacket creates a new cross-chain swap packet
func NewCrossChainSwapPacket(
	route []SwapHop,
	sender string,
	receiver string,
	amountIn math.Int,
	minOut math.Int,
	timeout uint64,
) CrossChainSwapPacketData {
	return CrossChainSwapPacketData{
		Type:     CrossChainSwapType,
		Route:    route,
		Sender:   sender,
		Receiver: receiver,
		AmountIn: amountIn,
		MinOut:   minOut,
		Timeout:  timeout,
	}
}
