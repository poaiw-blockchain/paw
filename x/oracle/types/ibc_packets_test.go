package types

import (
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/math"
)

func TestSubscribePricesPacketData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		packet  SubscribePricesPacketData
		wantErr string
	}{
		{
			name: "valid packet",
			packet: SubscribePricesPacketData{
				Type:           SubscribePricesType,
				Nonce:          1,
				Timestamp:      1234567890,
				Symbols:        []string{"BTC", "ETH"},
				UpdateInterval: 60,
				Subscriber:     validAddress,
			},
			wantErr: "",
		},
		{
			name: "invalid type",
			packet: SubscribePricesPacketData{
				Type:           "wrong_type",
				Nonce:          1,
				Timestamp:      1234567890,
				Symbols:        []string{"BTC"},
				UpdateInterval: 60,
				Subscriber:     validAddress,
			},
			wantErr: "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: SubscribePricesPacketData{
				Type:           SubscribePricesType,
				Nonce:          0,
				Timestamp:      1234567890,
				Symbols:        []string{"BTC"},
				UpdateInterval: 60,
				Subscriber:     validAddress,
			},
			wantErr: "nonce must be greater than zero",
		},
		{
			name: "zero timestamp",
			packet: SubscribePricesPacketData{
				Type:           SubscribePricesType,
				Nonce:          1,
				Timestamp:      0,
				Symbols:        []string{"BTC"},
				UpdateInterval: 60,
				Subscriber:     validAddress,
			},
			wantErr: "timestamp must be positive",
		},
		{
			name: "negative timestamp",
			packet: SubscribePricesPacketData{
				Type:           SubscribePricesType,
				Nonce:          1,
				Timestamp:      -1,
				Symbols:        []string{"BTC"},
				UpdateInterval: 60,
				Subscriber:     validAddress,
			},
			wantErr: "timestamp must be positive",
		},
		{
			name: "empty symbols",
			packet: SubscribePricesPacketData{
				Type:           SubscribePricesType,
				Nonce:          1,
				Timestamp:      1234567890,
				Symbols:        []string{},
				UpdateInterval: 60,
				Subscriber:     validAddress,
			},
			wantErr: "symbols cannot be empty",
		},
		{
			name: "zero update interval",
			packet: SubscribePricesPacketData{
				Type:           SubscribePricesType,
				Nonce:          1,
				Timestamp:      1234567890,
				Symbols:        []string{"BTC"},
				UpdateInterval: 0,
				Subscriber:     validAddress,
			},
			wantErr: "update interval must be positive",
		},
		{
			name: "invalid subscriber address",
			packet: SubscribePricesPacketData{
				Type:           SubscribePricesType,
				Nonce:          1,
				Timestamp:      1234567890,
				Symbols:        []string{"BTC"},
				UpdateInterval: 60,
				Subscriber:     invalidAddress,
			},
			wantErr: "invalid subscriber address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateBasic() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateBasic() error = nil, want error containing %q", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateBasic() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestSubscribePricesPacketData_GetType(t *testing.T) {
	packet := SubscribePricesPacketData{Type: SubscribePricesType}
	if packet.GetType() != SubscribePricesType {
		t.Errorf("GetType() = %s, want %s", packet.GetType(), SubscribePricesType)
	}
}

func TestQueryPricePacketData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		packet  QueryPricePacketData
		wantErr string
	}{
		{
			name: "valid packet",
			packet: QueryPricePacketData{
				Type:      QueryPriceType,
				Nonce:     1,
				Timestamp: 1234567890,
				Symbol:    "BTC",
				Sender:    validAddress,
			},
			wantErr: "",
		},
		{
			name: "invalid type",
			packet: QueryPricePacketData{
				Type:      "wrong_type",
				Nonce:     1,
				Timestamp: 1234567890,
				Symbol:    "BTC",
				Sender:    validAddress,
			},
			wantErr: "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: QueryPricePacketData{
				Type:      QueryPriceType,
				Nonce:     0,
				Timestamp: 1234567890,
				Symbol:    "BTC",
				Sender:    validAddress,
			},
			wantErr: "nonce must be greater than zero",
		},
		{
			name: "zero timestamp",
			packet: QueryPricePacketData{
				Type:      QueryPriceType,
				Nonce:     1,
				Timestamp: 0,
				Symbol:    "BTC",
				Sender:    validAddress,
			},
			wantErr: "timestamp must be positive",
		},
		{
			name: "empty symbol",
			packet: QueryPricePacketData{
				Type:      QueryPriceType,
				Nonce:     1,
				Timestamp: 1234567890,
				Symbol:    "",
				Sender:    validAddress,
			},
			wantErr: "symbol cannot be empty",
		},
		{
			name: "invalid sender",
			packet: QueryPricePacketData{
				Type:      QueryPriceType,
				Nonce:     1,
				Timestamp: 1234567890,
				Symbol:    "BTC",
				Sender:    invalidAddress,
			},
			wantErr: "invalid sender address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateBasic() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateBasic() error = nil, want error containing %q", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateBasic() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestPriceUpdatePacketData_ValidateBasic(t *testing.T) {
	validPriceData := PriceData{
		Symbol:      "BTC",
		Price:       math.LegacyNewDec(50000),
		Volume24h:   math.NewInt(1000000),
		Timestamp:   1234567890,
		Confidence:  math.LegacyMustNewDecFromStr("0.95"),
		OracleCount: 10,
	}

	tests := []struct {
		name    string
		packet  PriceUpdatePacketData
		wantErr string
	}{
		{
			name: "valid packet",
			packet: PriceUpdatePacketData{
				Type:      PriceUpdateType,
				Nonce:     1,
				Prices:    []PriceData{validPriceData},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
			wantErr: "",
		},
		{
			name: "invalid type",
			packet: PriceUpdatePacketData{
				Type:      "wrong_type",
				Nonce:     1,
				Prices:    []PriceData{validPriceData},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
			wantErr: "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: PriceUpdatePacketData{
				Type:      PriceUpdateType,
				Nonce:     0,
				Prices:    []PriceData{validPriceData},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
			wantErr: "nonce must be greater than zero",
		},
		{
			name: "empty prices",
			packet: PriceUpdatePacketData{
				Type:      PriceUpdateType,
				Nonce:     1,
				Prices:    []PriceData{},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
			wantErr: "prices cannot be empty",
		},
		{
			name: "price with empty symbol",
			packet: PriceUpdatePacketData{
				Type:  PriceUpdateType,
				Nonce: 1,
				Prices: []PriceData{
					{
						Symbol:      "",
						Price:       math.LegacyNewDec(50000),
						Confidence:  math.LegacyMustNewDecFromStr("0.95"),
						OracleCount: 10,
					},
				},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
			wantErr: "symbol cannot be empty",
		},
		{
			name: "price with zero value",
			packet: PriceUpdatePacketData{
				Type:  PriceUpdateType,
				Nonce: 1,
				Prices: []PriceData{
					{
						Symbol:      "BTC",
						Price:       math.LegacyZeroDec(),
						Confidence:  math.LegacyMustNewDecFromStr("0.95"),
						OracleCount: 10,
					},
				},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
			wantErr: "price must be positive",
		},
		{
			name: "price with negative confidence",
			packet: PriceUpdatePacketData{
				Type:  PriceUpdateType,
				Nonce: 1,
				Prices: []PriceData{
					{
						Symbol:      "BTC",
						Price:       math.LegacyNewDec(50000),
						Confidence:  math.LegacyNewDec(-1),
						OracleCount: 10,
					},
				},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
			wantErr: "confidence must be between 0 and 1",
		},
		{
			name: "price with confidence > 1",
			packet: PriceUpdatePacketData{
				Type:  PriceUpdateType,
				Nonce: 1,
				Prices: []PriceData{
					{
						Symbol:      "BTC",
						Price:       math.LegacyNewDec(50000),
						Confidence:  math.LegacyMustNewDecFromStr("1.5"),
						OracleCount: 10,
					},
				},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
			wantErr: "confidence must be between 0 and 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateBasic() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateBasic() error = nil, want error containing %q", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateBasic() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestOracleHeartbeatPacketData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		packet  OracleHeartbeatPacketData
		wantErr string
	}{
		{
			name: "valid packet",
			packet: OracleHeartbeatPacketData{
				Type:          OracleHeartbeatType,
				Nonce:         1,
				ChainID:       "chain-1",
				Timestamp:     1234567890,
				ActiveOracles: 10,
				BlockHeight:   1000,
			},
			wantErr: "",
		},
		{
			name: "invalid type",
			packet: OracleHeartbeatPacketData{
				Type:          "wrong_type",
				Nonce:         1,
				ChainID:       "chain-1",
				Timestamp:     1234567890,
				ActiveOracles: 10,
				BlockHeight:   1000,
			},
			wantErr: "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: OracleHeartbeatPacketData{
				Type:          OracleHeartbeatType,
				Nonce:         0,
				ChainID:       "chain-1",
				Timestamp:     1234567890,
				ActiveOracles: 10,
				BlockHeight:   1000,
			},
			wantErr: "nonce must be greater than zero",
		},
		{
			name: "empty chain ID",
			packet: OracleHeartbeatPacketData{
				Type:          OracleHeartbeatType,
				Nonce:         1,
				ChainID:       "",
				Timestamp:     1234567890,
				ActiveOracles: 10,
				BlockHeight:   1000,
			},
			wantErr: "chain ID cannot be empty",
		},
		{
			name: "zero timestamp",
			packet: OracleHeartbeatPacketData{
				Type:          OracleHeartbeatType,
				Nonce:         1,
				ChainID:       "chain-1",
				Timestamp:     0,
				ActiveOracles: 10,
				BlockHeight:   1000,
			},
			wantErr: "timestamp must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("ValidateBasic() error = %v, want nil", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateBasic() error = nil, want error containing %q", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ValidateBasic() error = %v, want error containing %q", err, tt.wantErr)
				}
			}
		})
	}
}

func TestParsePacketData(t *testing.T) {
	tests := []struct {
		name         string
		data         interface{}
		expectedType string
		wantErr      bool
	}{
		{
			name: "subscribe prices packet",
			data: SubscribePricesPacketData{
				Type:           SubscribePricesType,
				Nonce:          1,
				Timestamp:      1234567890,
				Symbols:        []string{"BTC"},
				UpdateInterval: 60,
				Subscriber:     validAddress,
			},
			expectedType: SubscribePricesType,
			wantErr:      false,
		},
		{
			name: "query price packet",
			data: QueryPricePacketData{
				Type:      QueryPriceType,
				Nonce:     1,
				Timestamp: 1234567890,
				Symbol:    "BTC",
				Sender:    validAddress,
			},
			expectedType: QueryPriceType,
			wantErr:      false,
		},
		{
			name: "price update packet",
			data: PriceUpdatePacketData{
				Type:  PriceUpdateType,
				Nonce: 1,
				Prices: []PriceData{
					{
						Symbol:     "BTC",
						Price:      math.LegacyNewDec(50000),
						Confidence: math.LegacyMustNewDecFromStr("0.95"),
					},
				},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
			expectedType: PriceUpdateType,
			wantErr:      false,
		},
		{
			name: "oracle heartbeat packet",
			data: OracleHeartbeatPacketData{
				Type:          OracleHeartbeatType,
				Nonce:         1,
				ChainID:       "chain-1",
				Timestamp:     1234567890,
				ActiveOracles: 10,
				BlockHeight:   1000,
			},
			expectedType: OracleHeartbeatType,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			jsonData, err := json.Marshal(tt.data)
			if err != nil {
				t.Fatalf("Failed to marshal test data: %v", err)
			}

			// Parse
			parsed, err := ParsePacketData(jsonData)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePacketData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if parsed == nil {
					t.Fatal("ParsePacketData() returned nil")
				}

				if parsed.GetType() != tt.expectedType {
					t.Errorf("ParsePacketData() type = %s, want %s", parsed.GetType(), tt.expectedType)
				}
			}
		})
	}
}

func TestParsePacketData_InvalidJSON(t *testing.T) {
	_, err := ParsePacketData([]byte("invalid json"))
	if err == nil {
		t.Error("ParsePacketData() should fail for invalid JSON")
	}
}

func TestParsePacketData_UnknownType(t *testing.T) {
	data := map[string]interface{}{
		"type": "unknown_type",
	}
	jsonData, _ := json.Marshal(data)

	_, err := ParsePacketData(jsonData)
	if err == nil {
		t.Error("ParsePacketData() should fail for unknown packet type")
	}
	if !strings.Contains(err.Error(), "unknown packet type") {
		t.Errorf("Expected error about unknown packet type, got: %v", err)
	}
}

func TestAcknowledgements_GetBytes(t *testing.T) {
	tests := []struct {
		name string
		ack  interface{ GetBytes() ([]byte, error) }
	}{
		{
			name: "SubscribePricesAcknowledgement",
			ack: SubscribePricesAcknowledgement{
				Nonce:             1,
				Success:           true,
				SubscribedSymbols: []string{"BTC", "ETH"},
				SubscriptionID:    "sub-123",
			},
		},
		{
			name: "QueryPriceAcknowledgement",
			ack: QueryPriceAcknowledgement{
				Nonce:   1,
				Success: true,
				PriceData: PriceData{
					Symbol:     "BTC",
					Price:      math.LegacyNewDec(50000),
					Confidence: math.LegacyMustNewDecFromStr("0.95"),
				},
			},
		},
		{
			name: "PriceUpdateAcknowledgement",
			ack: PriceUpdateAcknowledgement{
				Nonce:   1,
				Success: true,
			},
		},
		{
			name: "OracleHeartbeatAcknowledgement",
			ack: OracleHeartbeatAcknowledgement{
				Nonce:   1,
				Success: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := tt.ack.GetBytes()
			if err != nil {
				t.Errorf("GetBytes() error = %v, want nil", err)
			}
			if len(bytes) == 0 {
				t.Error("GetBytes() returned empty bytes")
			}

			// Verify it's valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal(bytes, &result); err != nil {
				t.Errorf("GetBytes() did not return valid JSON: %v", err)
			}
		})
	}
}

func TestPacketData_GetBytes(t *testing.T) {
	tests := []struct {
		name   string
		packet interface{ GetBytes() ([]byte, error) }
	}{
		{
			name: "SubscribePricesPacketData",
			packet: SubscribePricesPacketData{
				Type:           SubscribePricesType,
				Nonce:          1,
				Timestamp:      1234567890,
				Symbols:        []string{"BTC"},
				UpdateInterval: 60,
				Subscriber:     validAddress,
			},
		},
		{
			name: "QueryPricePacketData",
			packet: QueryPricePacketData{
				Type:      QueryPriceType,
				Nonce:     1,
				Timestamp: 1234567890,
				Symbol:    "BTC",
				Sender:    validAddress,
			},
		},
		{
			name: "PriceUpdatePacketData",
			packet: PriceUpdatePacketData{
				Type:  PriceUpdateType,
				Nonce: 1,
				Prices: []PriceData{
					{
						Symbol:     "BTC",
						Price:      math.LegacyNewDec(50000),
						Confidence: math.LegacyMustNewDecFromStr("0.95"),
					},
				},
				Timestamp: 1234567890,
				Source:    "chain-1",
			},
		},
		{
			name: "OracleHeartbeatPacketData",
			packet: OracleHeartbeatPacketData{
				Type:          OracleHeartbeatType,
				Nonce:         1,
				ChainID:       "chain-1",
				Timestamp:     1234567890,
				ActiveOracles: 10,
				BlockHeight:   1000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bytes, err := tt.packet.GetBytes()
			if err != nil {
				t.Errorf("GetBytes() error = %v, want nil", err)
			}
			if len(bytes) == 0 {
				t.Error("GetBytes() returned empty bytes")
			}

			// Verify it's valid JSON
			var result map[string]interface{}
			if err := json.Unmarshal(bytes, &result); err != nil {
				t.Errorf("GetBytes() did not return valid JSON: %v", err)
			}
		})
	}
}

func TestIBCPacketTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
	}{
		{"IBCVersion", IBCVersion},
		{"SubscribePricesType", SubscribePricesType},
		{"QueryPriceType", QueryPriceType},
		{"PriceUpdateType", PriceUpdateType},
		{"OracleHeartbeatType", OracleHeartbeatType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant == "" {
				t.Errorf("%s is empty", tt.name)
			}
		})
	}
}

func TestErrInvalidPacket(t *testing.T) {
	if ErrInvalidPacket == nil {
		t.Error("ErrInvalidPacket is nil")
	}

	errMsg := ErrInvalidPacket.Error()
	if !strings.Contains(errMsg, "invalid IBC packet") {
		t.Errorf("Expected error message to contain 'invalid IBC packet', got: %s", errMsg)
	}
}
