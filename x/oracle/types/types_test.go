package types

import (
	"strings"
	"testing"
)

func TestModuleConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"ModuleName", ModuleName, "oracle"},
		{"StoreKey", StoreKey, "oracle"},
		{"RouterKey", RouterKey, "oracle"},
		{"QuerierRoute", QuerierRoute, "oracle"},
		{"PortID", PortID, "oracle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestModuleConstantsNotEmpty(t *testing.T) {
	if ModuleName == "" {
		t.Error("ModuleName is empty")
	}
	if StoreKey == "" {
		t.Error("StoreKey is empty")
	}
	if RouterKey == "" {
		t.Error("RouterKey is empty")
	}
	if QuerierRoute == "" {
		t.Error("QuerierRoute is empty")
	}
	if PortID == "" {
		t.Error("PortID is empty")
	}
}

func TestModuleConstantsConsistency(t *testing.T) {
	// StoreKey should match ModuleName
	if StoreKey != ModuleName {
		t.Errorf("StoreKey %q should match ModuleName %q", StoreKey, ModuleName)
	}

	// RouterKey should match ModuleName
	if RouterKey != ModuleName {
		t.Errorf("RouterKey %q should match ModuleName %q", RouterKey, ModuleName)
	}

	// QuerierRoute should match ModuleName
	if QuerierRoute != ModuleName {
		t.Errorf("QuerierRoute %q should match ModuleName %q", QuerierRoute, ModuleName)
	}

	// PortID should match ModuleName
	if PortID != ModuleName {
		t.Errorf("PortID %q should match ModuleName %q", PortID, ModuleName)
	}
}

func TestIBCEventTypeConstantsInTypes(t *testing.T) {
	ibcEventTypes := []struct {
		name      string
		eventType string
		prefix    string
	}{
		{"EventTypeChannelOpen", EventTypeChannelOpen, "oracle_channel_"},
		{"EventTypeChannelOpenAck", EventTypeChannelOpenAck, "oracle_channel_"},
		{"EventTypeChannelOpenConfirm", EventTypeChannelOpenConfirm, "oracle_channel_"},
		{"EventTypeChannelClose", EventTypeChannelClose, "oracle_channel_"},
		{"EventTypePacketReceive", EventTypePacketReceive, "oracle_packet_"},
		{"EventTypePacketAck", EventTypePacketAck, "oracle_packet_"},
		{"EventTypePacketTimeout", EventTypePacketTimeout, "oracle_packet_"},
	}

	for _, tt := range ibcEventTypes {
		t.Run(tt.name, func(t *testing.T) {
			if tt.eventType == "" {
				t.Errorf("%s is empty", tt.name)
			}

			if !strings.HasPrefix(tt.eventType, tt.prefix) {
				t.Errorf("%s = %q should start with %q", tt.name, tt.eventType, tt.prefix)
			}

			// Check lowercase with underscores
			if strings.ToLower(tt.eventType) != tt.eventType {
				t.Errorf("%s = %q should be lowercase", tt.name, tt.eventType)
			}
		})
	}
}

func TestIBCAttributeKeyConstantsInTypes(t *testing.T) {
	attributeKeys := []struct {
		name string
		key  string
	}{
		{"AttributeKeyChannelID", AttributeKeyChannelID},
		{"AttributeKeyPortID", AttributeKeyPortID},
		{"AttributeKeyCounterpartyPortID", AttributeKeyCounterpartyPortID},
		{"AttributeKeyCounterpartyChannelID", AttributeKeyCounterpartyChannelID},
		{"AttributeKeyPacketType", AttributeKeyPacketType},
		{"AttributeKeySequence", AttributeKeySequence},
		{"AttributeKeyAckSuccess", AttributeKeyAckSuccess},
		{"AttributeKeyPendingOperations", AttributeKeyPendingOperations},
	}

	for _, ak := range attributeKeys {
		t.Run(ak.name, func(t *testing.T) {
			if ak.key == "" {
				t.Errorf("%s is empty", ak.name)
			}

			// Check lowercase with underscores
			if strings.ToLower(ak.key) != ak.key {
				t.Errorf("%s = %q should be lowercase", ak.name, ak.key)
			}

			// No hyphens in attribute keys
			if strings.Contains(ak.key, "-") {
				t.Errorf("%s = %q should use underscores, not hyphens", ak.name, ak.key)
			}
		})
	}
}

func TestIBCEventTypes_Unique(t *testing.T) {
	eventTypes := []string{
		EventTypeChannelOpen,
		EventTypeChannelOpenAck,
		EventTypeChannelOpenConfirm,
		EventTypeChannelClose,
		EventTypePacketReceive,
		EventTypePacketAck,
		EventTypePacketTimeout,
	}

	seen := make(map[string]bool)
	for _, et := range eventTypes {
		if seen[et] {
			t.Errorf("Duplicate IBC event type: %s", et)
		}
		seen[et] = true
	}
}

func TestIBCAttributeKeys_Unique(t *testing.T) {
	attributeKeys := []string{
		AttributeKeyChannelID,
		AttributeKeyPortID,
		AttributeKeyCounterpartyPortID,
		AttributeKeyCounterpartyChannelID,
		AttributeKeyPacketType,
		AttributeKeySequence,
		AttributeKeyAckSuccess,
		AttributeKeyPendingOperations,
	}

	seen := make(map[string]bool)
	for _, ak := range attributeKeys {
		if seen[ak] {
			t.Errorf("Duplicate IBC attribute key: %s", ak)
		}
		seen[ak] = true
	}
}

func TestIBCEventTypeNamingConvention(t *testing.T) {
	// All IBC event types should follow oracle_<category>_<action> format
	tests := []struct {
		eventType      string
		expectedPrefix string
	}{
		{EventTypeChannelOpen, "oracle_channel_"},
		{EventTypeChannelOpenAck, "oracle_channel_"},
		{EventTypeChannelOpenConfirm, "oracle_channel_"},
		{EventTypeChannelClose, "oracle_channel_"},
		{EventTypePacketReceive, "oracle_packet_"},
		{EventTypePacketAck, "oracle_packet_"},
		{EventTypePacketTimeout, "oracle_packet_"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			if !strings.HasPrefix(tt.eventType, tt.expectedPrefix) {
				t.Errorf("Event type %q should start with %q", tt.eventType, tt.expectedPrefix)
			}
		})
	}
}

func TestIBCAttributeKeyNamingConvention(t *testing.T) {
	// Test that IBC attribute keys follow snake_case convention
	attributeKeys := []string{
		AttributeKeyChannelID,
		AttributeKeyPortID,
		AttributeKeyCounterpartyPortID,
		AttributeKeyCounterpartyChannelID,
		AttributeKeyPacketType,
		AttributeKeySequence,
		AttributeKeyAckSuccess,
		AttributeKeyPendingOperations,
	}

	for _, key := range attributeKeys {
		t.Run(key, func(t *testing.T) {
			// Check lowercase
			if key != strings.ToLower(key) {
				t.Errorf("Attribute key %q should be lowercase", key)
			}

			// Check no hyphens
			if strings.Contains(key, "-") {
				t.Errorf("Attribute key %q should use underscores, not hyphens", key)
			}

			// Check no camelCase (by checking for uppercase letters after first char)
			for i := 1; i < len(key); i++ {
				if key[i] >= 'A' && key[i] <= 'Z' {
					t.Errorf("Attribute key %q should be snake_case, not camelCase", key)
					break
				}
			}
		})
	}
}

func TestIBCEventTypeCategories(t *testing.T) {
	// Verify IBC events are properly categorized
	channelEvents := []string{
		EventTypeChannelOpen,
		EventTypeChannelOpenAck,
		EventTypeChannelOpenConfirm,
		EventTypeChannelClose,
	}

	packetEvents := []string{
		EventTypePacketReceive,
		EventTypePacketAck,
		EventTypePacketTimeout,
	}

	for _, event := range channelEvents {
		if !strings.Contains(event, "channel") {
			t.Errorf("Channel event %q should contain 'channel'", event)
		}
	}

	for _, event := range packetEvents {
		if !strings.Contains(event, "packet") {
			t.Errorf("Packet event %q should contain 'packet'", event)
		}
	}
}

func TestAttributeKeysDescriptive(t *testing.T) {
	// Verify attribute keys are descriptive
	tests := []struct {
		key     string
		keyword string
	}{
		{AttributeKeyChannelID, "channel"},
		{AttributeKeyPortID, "port"},
		{AttributeKeyCounterpartyPortID, "counterparty"},
		{AttributeKeyCounterpartyChannelID, "counterparty"},
		{AttributeKeyPacketType, "packet"},
		{AttributeKeySequence, "sequence"},
		{AttributeKeyAckSuccess, "ack"},
		{AttributeKeyPendingOperations, "pending"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if !strings.Contains(tt.key, tt.keyword) {
				t.Errorf("Attribute key %q should contain keyword %q for clarity", tt.key, tt.keyword)
			}
		})
	}
}

func TestModuleNameLowercase(t *testing.T) {
	if ModuleName != strings.ToLower(ModuleName) {
		t.Errorf("ModuleName %q should be lowercase", ModuleName)
	}
}

func TestConstantsNoWhitespace(t *testing.T) {
	constants := map[string]string{
		"ModuleName":   ModuleName,
		"StoreKey":     StoreKey,
		"RouterKey":    RouterKey,
		"QuerierRoute": QuerierRoute,
		"PortID":       PortID,
	}

	for name, value := range constants {
		if strings.TrimSpace(value) != value {
			t.Errorf("%s has leading or trailing whitespace: %q", name, value)
		}
		if strings.Contains(value, " ") {
			t.Errorf("%s contains whitespace: %q", name, value)
		}
	}
}
