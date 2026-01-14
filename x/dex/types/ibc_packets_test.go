package types

import (
	"encoding/json"
	"testing"

	"cosmossdk.io/math"
)

func TestIBCVersion(t *testing.T) {
	if IBCVersion != "paw-dex-1" {
		t.Errorf("Expected IBC version 'paw-dex-1', got %s", IBCVersion)
	}
}

func TestPacketTypes(t *testing.T) {
	types := map[string]string{
		"QueryPoolsType":     QueryPoolsType,
		"ExecuteSwapType":    ExecuteSwapType,
		"CrossChainSwapType": CrossChainSwapType,
		"PoolUpdateType":     PoolUpdateType,
	}

	expected := map[string]string{
		"QueryPoolsType":     "query_pools",
		"ExecuteSwapType":    "execute_swap",
		"CrossChainSwapType": "cross_chain_swap",
		"PoolUpdateType":     "pool_update",
	}

	for name, typ := range types {
		if typ != expected[name] {
			t.Errorf("Expected %s to be %q, got %q", name, expected[name], typ)
		}
	}
}

// ============================================================================
// QueryPoolsPacketData Tests
// ============================================================================

func TestQueryPoolsPacketData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		packet  QueryPoolsPacketData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid packet",
			packet: QueryPoolsPacketData{
				Type:      QueryPoolsType,
				Nonce:     1,
				Timestamp: 1700000000,
				TokenA:    "upaw",
				TokenB:    "uatom",
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			packet: QueryPoolsPacketData{
				Type:      "wrong_type",
				Nonce:     1,
				Timestamp: 1700000000,
				TokenA:    "upaw",
				TokenB:    "uatom",
			},
			wantErr: true,
			errMsg:  "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: QueryPoolsPacketData{
				Type:      QueryPoolsType,
				Nonce:     0,
				Timestamp: 1700000000,
				TokenA:    "upaw",
				TokenB:    "uatom",
			},
			wantErr: true,
			errMsg:  "nonce must be greater than zero",
		},
		{
			name: "zero timestamp",
			packet: QueryPoolsPacketData{
				Type:      QueryPoolsType,
				Nonce:     1,
				Timestamp: 0,
				TokenA:    "upaw",
				TokenB:    "uatom",
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
		{
			name: "negative timestamp",
			packet: QueryPoolsPacketData{
				Type:      QueryPoolsType,
				Nonce:     1,
				Timestamp: -100,
				TokenA:    "upaw",
				TokenB:    "uatom",
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
		{
			name: "empty token A",
			packet: QueryPoolsPacketData{
				Type:      QueryPoolsType,
				Nonce:     1,
				Timestamp: 1700000000,
				TokenA:    "",
				TokenB:    "uatom",
			},
			wantErr: true,
			errMsg:  "token A cannot be empty",
		},
		{
			name: "empty token B",
			packet: QueryPoolsPacketData{
				Type:      QueryPoolsType,
				Nonce:     1,
				Timestamp: 1700000000,
				TokenA:    "upaw",
				TokenB:    "",
			},
			wantErr: true,
			errMsg:  "token B cannot be empty",
		},
		{
			name: "invalid token A denom",
			packet: QueryPoolsPacketData{
				Type:      QueryPoolsType,
				Nonce:     1,
				Timestamp: 1700000000,
				TokenA:    "Invalid Denom",
				TokenB:    "uatom",
			},
			wantErr: true,
			errMsg:  "invalid token A denom",
		},
		{
			name: "invalid token B denom",
			packet: QueryPoolsPacketData{
				Type:      QueryPoolsType,
				Nonce:     1,
				Timestamp: 1700000000,
				TokenA:    "upaw",
				TokenB:    "123bad",
			},
			wantErr: true,
			errMsg:  "invalid token B denom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestQueryPoolsPacketData_GetType(t *testing.T) {
	packet := QueryPoolsPacketData{Type: QueryPoolsType}
	if packet.GetType() != QueryPoolsType {
		t.Errorf("Expected type %s, got %s", QueryPoolsType, packet.GetType())
	}
}

func TestQueryPoolsPacketData_GetBytes(t *testing.T) {
	packet := QueryPoolsPacketData{
		Type:      QueryPoolsType,
		Nonce:     1,
		Timestamp: 1700000000,
		TokenA:    "upaw",
		TokenB:    "uatom",
	}

	bytes, err := packet.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() error = %v", err)
	}

	// Verify can unmarshal back
	var unmarshaled QueryPoolsPacketData
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Type != packet.Type {
		t.Error("Type mismatch after unmarshal")
	}
}

// ============================================================================
// ExecuteSwapPacketData Tests
// ============================================================================

func TestExecuteSwapPacketData_ValidateBasic(t *testing.T) {
	validAddr := "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q"

	tests := []struct {
		name    string
		packet  ExecuteSwapPacketData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid packet",
			packet: ExecuteSwapPacketData{
				Type:         ExecuteSwapType,
				Nonce:        1,
				Timestamp:    1700000000,
				PoolID:       "1",
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Sender:       validAddr,
				Receiver:     validAddr,
				Timeout:      1700001000,
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			packet: ExecuteSwapPacketData{
				Type:         "wrong",
				Nonce:        1,
				Timestamp:    1700000000,
				PoolID:       "1",
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Sender:       validAddr,
				Receiver:     validAddr,
			},
			wantErr: true,
			errMsg:  "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: ExecuteSwapPacketData{
				Type:         ExecuteSwapType,
				Nonce:        0,
				Timestamp:    1700000000,
				PoolID:       "1",
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Sender:       validAddr,
				Receiver:     validAddr,
			},
			wantErr: true,
			errMsg:  "nonce must be greater than zero",
		},
		{
			name: "empty pool ID",
			packet: ExecuteSwapPacketData{
				Type:         ExecuteSwapType,
				Nonce:        1,
				Timestamp:    1700000000,
				PoolID:       "",
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Sender:       validAddr,
				Receiver:     validAddr,
			},
			wantErr: true,
			errMsg:  "pool ID cannot be empty",
		},
		{
			name: "zero amount in",
			packet: ExecuteSwapPacketData{
				Type:         ExecuteSwapType,
				Nonce:        1,
				Timestamp:    1700000000,
				PoolID:       "1",
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(0),
				MinAmountOut: math.NewInt(900000),
				Sender:       validAddr,
				Receiver:     validAddr,
			},
			wantErr: true,
			errMsg:  "amount in must be positive",
		},
		{
			name: "negative min amount out",
			packet: ExecuteSwapPacketData{
				Type:         ExecuteSwapType,
				Nonce:        1,
				Timestamp:    1700000000,
				PoolID:       "1",
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(-1),
				Sender:       validAddr,
				Receiver:     validAddr,
			},
			wantErr: true,
			errMsg:  "min amount out cannot be negative",
		},
		{
			name: "invalid sender",
			packet: ExecuteSwapPacketData{
				Type:         ExecuteSwapType,
				Nonce:        1,
				Timestamp:    1700000000,
				PoolID:       "1",
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Sender:       "invalid",
				Receiver:     validAddr,
			},
			wantErr: true,
			errMsg:  "invalid sender address",
		},
		{
			name: "invalid receiver",
			packet: ExecuteSwapPacketData{
				Type:         ExecuteSwapType,
				Nonce:        1,
				Timestamp:    1700000000,
				PoolID:       "1",
				TokenIn:      "upaw",
				TokenOut:     "uatom",
				AmountIn:     math.NewInt(1000000),
				MinAmountOut: math.NewInt(900000),
				Sender:       validAddr,
				Receiver:     "invalid",
			},
			wantErr: true,
			errMsg:  "invalid receiver address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// CrossChainSwapPacketData Tests
// ============================================================================

func TestCrossChainSwapPacketData_ValidateBasic(t *testing.T) {
	validAddr := "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q"

	tests := []struct {
		name    string
		packet  CrossChainSwapPacketData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid packet",
			packet: CrossChainSwapPacketData{
				Type:      CrossChainSwapType,
				Nonce:     1,
				Timestamp: 1700000000,
				Route: []SwapHop{
					{
						ChainID:      "chain-1",
						PoolID:       "pool-1",
						TokenIn:      "upaw",
						TokenOut:     "uatom",
						MinAmountOut: math.NewInt(100000),
					},
				},
				Sender:   validAddr,
				Receiver: validAddr,
				AmountIn: math.NewInt(1000000),
				MinOut:   math.NewInt(900000),
				Timeout:  1700001000,
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			packet: CrossChainSwapPacketData{
				Type:      "wrong",
				Nonce:     1,
				Timestamp: 1700000000,
				Route: []SwapHop{
					{
						ChainID:  "chain-1",
						PoolID:   "pool-1",
						TokenIn:  "upaw",
						TokenOut: "uatom",
					},
				},
				Sender:   validAddr,
				Receiver: validAddr,
				AmountIn: math.NewInt(1000000),
				MinOut:   math.NewInt(900000),
			},
			wantErr: true,
			errMsg:  "invalid packet type",
		},
		{
			name: "empty route",
			packet: CrossChainSwapPacketData{
				Type:      CrossChainSwapType,
				Nonce:     1,
				Timestamp: 1700000000,
				Route:     []SwapHop{},
				Sender:    validAddr,
				Receiver:  validAddr,
				AmountIn:  math.NewInt(1000000),
				MinOut:    math.NewInt(900000),
			},
			wantErr: true,
			errMsg:  "route cannot be empty",
		},
		{
			name: "zero amount in",
			packet: CrossChainSwapPacketData{
				Type:      CrossChainSwapType,
				Nonce:     1,
				Timestamp: 1700000000,
				Route: []SwapHop{
					{ChainID: "chain-1", PoolID: "pool-1", TokenIn: "upaw", TokenOut: "uatom"},
				},
				Sender:   validAddr,
				Receiver: validAddr,
				AmountIn: math.NewInt(0),
				MinOut:   math.NewInt(900000),
			},
			wantErr: true,
			errMsg:  "amount in must be positive",
		},
		{
			name: "negative min out",
			packet: CrossChainSwapPacketData{
				Type:      CrossChainSwapType,
				Nonce:     1,
				Timestamp: 1700000000,
				Route: []SwapHop{
					{ChainID: "chain-1", PoolID: "pool-1", TokenIn: "upaw", TokenOut: "uatom"},
				},
				Sender:   validAddr,
				Receiver: validAddr,
				AmountIn: math.NewInt(1000000),
				MinOut:   math.NewInt(-1),
			},
			wantErr: true,
			errMsg:  "min out cannot be negative",
		},
		{
			name: "hop missing chain ID",
			packet: CrossChainSwapPacketData{
				Type:      CrossChainSwapType,
				Nonce:     1,
				Timestamp: 1700000000,
				Route: []SwapHop{
					{ChainID: "", PoolID: "pool-1", TokenIn: "upaw", TokenOut: "uatom"},
				},
				Sender:   validAddr,
				Receiver: validAddr,
				AmountIn: math.NewInt(1000000),
				MinOut:   math.NewInt(900000),
			},
			wantErr: true,
			errMsg:  "chain ID cannot be empty",
		},
		{
			name: "hop missing pool ID",
			packet: CrossChainSwapPacketData{
				Type:      CrossChainSwapType,
				Nonce:     1,
				Timestamp: 1700000000,
				Route: []SwapHop{
					{ChainID: "chain-1", PoolID: "", TokenIn: "upaw", TokenOut: "uatom"},
				},
				Sender:   validAddr,
				Receiver: validAddr,
				AmountIn: math.NewInt(1000000),
				MinOut:   math.NewInt(900000),
			},
			wantErr: true,
			errMsg:  "pool ID cannot be empty",
		},
		{
			name: "hop with invalid token in denom",
			packet: CrossChainSwapPacketData{
				Type:      CrossChainSwapType,
				Nonce:     1,
				Timestamp: 1700000000,
				Route: []SwapHop{
					{ChainID: "chain-1", PoolID: "pool-1", TokenIn: "Bad Token", TokenOut: "uatom"},
				},
				Sender:   validAddr,
				Receiver: validAddr,
				AmountIn: math.NewInt(1000000),
				MinOut:   math.NewInt(900000),
			},
			wantErr: true,
			errMsg:  "invalid token in denom",
		},
		{
			name: "multi-hop route",
			packet: CrossChainSwapPacketData{
				Type:      CrossChainSwapType,
				Nonce:     1,
				Timestamp: 1700000000,
				Route: []SwapHop{
					{ChainID: "chain-1", PoolID: "pool-1", TokenIn: "upaw", TokenOut: "uatom", MinAmountOut: math.NewInt(100)},
					{ChainID: "chain-2", PoolID: "pool-2", TokenIn: "uatom", TokenOut: "uosmo", MinAmountOut: math.NewInt(90)},
					{ChainID: "chain-3", PoolID: "pool-3", TokenIn: "uosmo", TokenOut: "ujuno", MinAmountOut: math.NewInt(80)},
				},
				Sender:   validAddr,
				Receiver: validAddr,
				AmountIn: math.NewInt(1000000),
				MinOut:   math.NewInt(900000),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// PoolUpdatePacketData Tests
// ============================================================================

func TestPoolUpdatePacketData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		packet  PoolUpdatePacketData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid packet",
			packet: PoolUpdatePacketData{
				Type:        PoolUpdateType,
				Nonce:       1,
				PoolID:      "pool-1",
				ReserveA:    math.NewInt(1000000),
				ReserveB:    math.NewInt(2000000),
				TotalShares: math.NewInt(1414213),
				Timestamp:   1700000000,
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			packet: PoolUpdatePacketData{
				Type:        "wrong",
				Nonce:       1,
				PoolID:      "pool-1",
				ReserveA:    math.NewInt(1000000),
				ReserveB:    math.NewInt(2000000),
				TotalShares: math.NewInt(1414213),
			},
			wantErr: true,
			errMsg:  "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: PoolUpdatePacketData{
				Type:        PoolUpdateType,
				Nonce:       0,
				PoolID:      "pool-1",
				ReserveA:    math.NewInt(1000000),
				ReserveB:    math.NewInt(2000000),
				TotalShares: math.NewInt(1414213),
			},
			wantErr: true,
			errMsg:  "nonce must be greater than zero",
		},
		{
			name: "empty pool ID",
			packet: PoolUpdatePacketData{
				Type:        PoolUpdateType,
				Nonce:       1,
				PoolID:      "",
				ReserveA:    math.NewInt(1000000),
				ReserveB:    math.NewInt(2000000),
				TotalShares: math.NewInt(1414213),
			},
			wantErr: true,
			errMsg:  "pool ID cannot be empty",
		},
		{
			name: "negative reserve A",
			packet: PoolUpdatePacketData{
				Type:        PoolUpdateType,
				Nonce:       1,
				PoolID:      "pool-1",
				ReserveA:    math.NewInt(-1000),
				ReserveB:    math.NewInt(2000000),
				TotalShares: math.NewInt(1414213),
			},
			wantErr: true,
			errMsg:  "reserve A cannot be negative",
		},
		{
			name: "negative reserve B",
			packet: PoolUpdatePacketData{
				Type:        PoolUpdateType,
				Nonce:       1,
				PoolID:      "pool-1",
				ReserveA:    math.NewInt(1000000),
				ReserveB:    math.NewInt(-2000),
				TotalShares: math.NewInt(1414213),
			},
			wantErr: true,
			errMsg:  "reserve B cannot be negative",
		},
		{
			name: "zero reserves allowed",
			packet: PoolUpdatePacketData{
				Type:        PoolUpdateType,
				Nonce:       1,
				PoolID:      "pool-1",
				ReserveA:    math.NewInt(0),
				ReserveB:    math.NewInt(0),
				TotalShares: math.NewInt(0),
				Timestamp:   1700000000,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// ParsePacketData Tests
// ============================================================================

func TestParsePacketData(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		wantType string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "query pools packet",
			data:     []byte(`{"type":"query_pools","nonce":1,"timestamp":1700000000,"token_a":"upaw","token_b":"uatom"}`),
			wantType: QueryPoolsType,
			wantErr:  false,
		},
		{
			name:     "execute swap packet",
			data:     []byte(`{"type":"execute_swap","nonce":1,"timestamp":1700000000,"pool_id":"1","token_in":"upaw","token_out":"uatom","amount_in":"1000000","min_amount_out":"900000","sender":"cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q","receiver":"cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q","timeout":1700001000}`),
			wantType: ExecuteSwapType,
			wantErr:  false,
		},
		{
			name:     "pool update packet",
			data:     []byte(`{"type":"pool_update","nonce":1,"pool_id":"pool-1","reserve_a":"1000000","reserve_b":"2000000","total_shares":"1414213","timestamp":1700000000}`),
			wantType: PoolUpdateType,
			wantErr:  false,
		},
		{
			name:    "unknown type",
			data:    []byte(`{"type":"unknown_type"}`),
			wantErr: true,
			errMsg:  "unknown packet type",
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{invalid json`),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte(``),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet, err := ParsePacketData(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePacketData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if packet.GetType() != tt.wantType {
					t.Errorf("Expected type %s, got %s", tt.wantType, packet.GetType())
				}
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

// ============================================================================
// Constructor Tests
// ============================================================================

func TestNewQueryPoolsPacket(t *testing.T) {
	packet := NewQueryPoolsPacket("upaw", "uatom", 42)

	if packet.Type != QueryPoolsType {
		t.Errorf("Expected type %s, got %s", QueryPoolsType, packet.Type)
	}
	if packet.Nonce != 42 {
		t.Errorf("Expected nonce 42, got %d", packet.Nonce)
	}
	if packet.TokenA != "upaw" {
		t.Errorf("Expected TokenA 'upaw', got %s", packet.TokenA)
	}
	if packet.TokenB != "uatom" {
		t.Errorf("Expected TokenB 'uatom', got %s", packet.TokenB)
	}
}

func TestNewExecuteSwapPacket(t *testing.T) {
	validAddr := "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q"
	packet := NewExecuteSwapPacket(
		42,
		"pool-1",
		"upaw",
		"uatom",
		math.NewInt(1000000),
		math.NewInt(900000),
		validAddr,
		validAddr,
		1700001000,
	)

	if packet.Type != ExecuteSwapType {
		t.Errorf("Expected type %s, got %s", ExecuteSwapType, packet.Type)
	}
	if packet.Nonce != 42 {
		t.Errorf("Expected nonce 42, got %d", packet.Nonce)
	}
	if packet.PoolID != "pool-1" {
		t.Errorf("Expected PoolID 'pool-1', got %s", packet.PoolID)
	}
}

func TestNewCrossChainSwapPacket(t *testing.T) {
	validAddr := "cosmos1zg69v7ys40x77y352eufp27daufrg4ncnjqz7q"
	route := []SwapHop{
		{ChainID: "chain-1", PoolID: "pool-1", TokenIn: "upaw", TokenOut: "uatom", MinAmountOut: math.NewInt(100)},
	}

	packet := NewCrossChainSwapPacket(
		42,
		route,
		validAddr,
		validAddr,
		math.NewInt(1000000),
		math.NewInt(900000),
		1700001000,
	)

	if packet.Type != CrossChainSwapType {
		t.Errorf("Expected type %s, got %s", CrossChainSwapType, packet.Type)
	}
	if packet.Nonce != 42 {
		t.Errorf("Expected nonce 42, got %d", packet.Nonce)
	}
	if len(packet.Route) != 1 {
		t.Errorf("Expected 1 hop, got %d", len(packet.Route))
	}
}

// ============================================================================
// Acknowledgement Tests
// ============================================================================

func TestQueryPoolsAcknowledgement_GetBytes(t *testing.T) {
	ack := QueryPoolsAcknowledgement{
		Nonce:   1,
		Success: true,
		Pools: []PoolInfo{
			{
				ChainID:     "chain-1",
				PoolID:      "pool-1",
				TokenA:      "upaw",
				TokenB:      "uatom",
				ReserveA:    math.NewInt(1000000),
				ReserveB:    math.NewInt(2000000),
				SwapFee:     math.LegacyMustNewDecFromStr("0.003"),
				TotalShares: math.NewInt(1414213),
			},
		},
	}

	bytes, err := ack.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() error = %v", err)
	}

	var unmarshaled QueryPoolsAcknowledgement
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Success != ack.Success {
		t.Error("Success mismatch after unmarshal")
	}
}

func TestExecuteSwapAcknowledgement_GetBytes(t *testing.T) {
	ack := ExecuteSwapAcknowledgement{
		Nonce:     1,
		Success:   true,
		AmountOut: math.NewInt(990000),
		SwapFee:   math.NewInt(3000),
	}

	bytes, err := ack.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() error = %v", err)
	}

	var unmarshaled ExecuteSwapAcknowledgement
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if !unmarshaled.AmountOut.Equal(ack.AmountOut) {
		t.Error("AmountOut mismatch after unmarshal")
	}
}

func TestCrossChainSwapAcknowledgement_GetBytes(t *testing.T) {
	ack := CrossChainSwapAcknowledgement{
		Nonce:        1,
		Success:      true,
		FinalAmount:  math.NewInt(980000),
		HopsExecuted: 3,
		TotalFees:    math.NewInt(20000),
	}

	bytes, err := ack.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() error = %v", err)
	}

	var unmarshaled CrossChainSwapAcknowledgement
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.HopsExecuted != ack.HopsExecuted {
		t.Error("HopsExecuted mismatch after unmarshal")
	}
}

func TestPoolUpdateAcknowledgement_GetBytes(t *testing.T) {
	ack := PoolUpdateAcknowledgement{
		Nonce:   1,
		Success: true,
	}

	bytes, err := ack.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() error = %v", err)
	}

	var unmarshaled PoolUpdateAcknowledgement
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if unmarshaled.Success != ack.Success {
		t.Error("Success mismatch after unmarshal")
	}
}
