package types

import (
	"strings"
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		prefix    string
	}{
		{"EventTypeOraclePriceUpdate", EventTypeOraclePriceUpdate, "oracle_"},
		{"EventTypeOraclePriceSubmitted", EventTypeOraclePriceSubmitted, "oracle_"},
		{"EventTypeOraclePriceAggregated", EventTypeOraclePriceAggregated, "oracle_"},
		{"EventTypeOracleVote", EventTypeOracleVote, "oracle_"},
		{"EventTypeOracleVoteAggregated", EventTypeOracleVoteAggregated, "oracle_"},
		{"EventTypeOracleFeederDelegated", EventTypeOracleFeederDelegated, "oracle_"},
		{"EventTypeOracleSlash", EventTypeOracleSlash, "oracle_"},
		{"EventTypeOracleSlashOutlier", EventTypeOracleSlashOutlier, "oracle_"},
		{"EventTypeOracleJail", EventTypeOracleJail, "oracle_"},
		{"EventTypeOracleOutlier", EventTypeOracleOutlier, "oracle_"},
		{"EventTypeOracleCrossChainPrice", EventTypeOracleCrossChainPrice, "oracle_"},
		{"EventTypeOraclePriceRelay", EventTypeOraclePriceRelay, "oracle_"},
		{"EventTypeOracleParamsUpdated", EventTypeOracleParamsUpdated, "oracle_"},
		{"EventTypeCircuitBreakerOpen", EventTypeCircuitBreakerOpen, "oracle_"},
		{"EventTypeCircuitBreakerClose", EventTypeCircuitBreakerClose, "oracle_"},
		{"EventTypePriceOverride", EventTypePriceOverride, "oracle_"},
		{"EventTypePriceOverrideClear", EventTypePriceOverrideClear, "oracle_"},
		{"EventTypeSlashingDisabled", EventTypeSlashingDisabled, "oracle_"},
		{"EventTypeSlashingEnabled", EventTypeSlashingEnabled, "oracle_"},
		{"EventTypeEmergencyPause", EventTypeEmergencyPause, "oracle_"},
		{"EventTypeEmergencyResume", EventTypeEmergencyResume, "oracle_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Check event type is not empty
			if tt.eventType == "" {
				t.Error("Event type is empty")
			}

			// Check event type has correct prefix
			if !strings.HasPrefix(tt.eventType, tt.prefix) {
				t.Errorf("Event type %q should start with %q", tt.eventType, tt.prefix)
			}

			// Check event type uses lowercase with underscores
			if strings.ToLower(tt.eventType) != tt.eventType {
				t.Errorf("Event type %q should be lowercase", tt.eventType)
			}

			if strings.Contains(tt.eventType, "-") {
				t.Errorf("Event type %q should use underscores, not hyphens", tt.eventType)
			}
		})
	}
}

func TestIBCEventTypeConstants(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		prefix    string
	}{
		{"EventTypeChannelOpen", EventTypeChannelOpen, "oracle_"},
		{"EventTypeChannelOpenAck", EventTypeChannelOpenAck, "oracle_"},
		{"EventTypeChannelOpenConfirm", EventTypeChannelOpenConfirm, "oracle_"},
		{"EventTypeChannelClose", EventTypeChannelClose, "oracle_"},
		{"EventTypePacketReceive", EventTypePacketReceive, "oracle_"},
		{"EventTypePacketAck", EventTypePacketAck, "oracle_"},
		{"EventTypePacketTimeout", EventTypePacketTimeout, "oracle_"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.eventType == "" {
				t.Error("IBC event type is empty")
			}

			if !strings.HasPrefix(tt.eventType, tt.prefix) {
				t.Errorf("IBC event type %q should start with %q", tt.eventType, tt.prefix)
			}
		})
	}
}

func TestAttributeKeyConstants(t *testing.T) {
	attributeKeys := []struct {
		name string
		key  string
	}{
		{"AttributeKeyAsset", AttributeKeyAsset},
		{"AttributeKeyAssets", AttributeKeyAssets},
		{"AttributeKeyDenom", AttributeKeyDenom},
		{"AttributeKeyPrice", AttributeKeyPrice},
		{"AttributeKeyPrices", AttributeKeyPrices},
		{"AttributeKeyMedian", AttributeKeyMedian},
		{"AttributeKeyMean", AttributeKeyMean},
		{"AttributeKeyStdDev", AttributeKeyStdDev},
		{"AttributeKeyMAD", AttributeKeyMAD},
		{"AttributeKeyValidator", AttributeKeyValidator},
		{"AttributeKeyValidators", AttributeKeyValidators},
		{"AttributeKeyFeeder", AttributeKeyFeeder},
		{"AttributeKeyDelegate", AttributeKeyDelegate},
		{"AttributeKeyVotingPower", AttributeKeyVotingPower},
		{"AttributeKeyNumValidators", AttributeKeyNumValidators},
		{"AttributeKeyTotalVotingPower", AttributeKeyTotalVotingPower},
		{"AttributeKeyDeviation", AttributeKeyDeviation},
		{"AttributeKeyDeviationPercent", AttributeKeyDeviationPercent},
		{"AttributeKeyThreshold", AttributeKeyThreshold},
		{"AttributeKeyReason", AttributeKeyReason},
		{"AttributeKeySlashFraction", AttributeKeySlashFraction},
		{"AttributeKeySlashAmount", AttributeKeySlashAmount},
		{"AttributeKeySeverity", AttributeKeySeverity},
		{"AttributeKeyJailed", AttributeKeyJailed},
		{"AttributeKeyBlockHeight", AttributeKeyBlockHeight},
		{"AttributeKeyTimestamp", AttributeKeyTimestamp},
		{"AttributeKeyNumSubmissions", AttributeKeyNumSubmissions},
		{"AttributeKeyNumOutliers", AttributeKeyNumOutliers},
		{"AttributeKeyConfidence", AttributeKeyConfidence},
		{"AttributeKeySourceChain", AttributeKeySourceChain},
		{"AttributeKeyTargetChain", AttributeKeyTargetChain},
		{"AttributeKeyStatus", AttributeKeyStatus},
		{"AttributeKeyError", AttributeKeyError},
		{"AttributeKeyActor", AttributeKeyActor},
		{"AttributeKeyPair", AttributeKeyPair},
		{"AttributeKeyFeedType", AttributeKeyFeedType},
		{"AttributeKeyPausedBy", AttributeKeyPausedBy},
		{"AttributeKeyPauseReason", AttributeKeyPauseReason},
		{"AttributeKeyResumeReason", AttributeKeyResumeReason},
	}

	for _, ak := range attributeKeys {
		t.Run(ak.name, func(t *testing.T) {
			// Check attribute key is not empty
			if ak.key == "" {
				t.Errorf("%s is empty", ak.name)
			}

			// Check attribute key uses lowercase with underscores
			if strings.ToLower(ak.key) != ak.key {
				t.Errorf("%s = %q should be lowercase", ak.name, ak.key)
			}

			if strings.Contains(ak.key, "-") {
				t.Errorf("%s = %q should use underscores, not hyphens", ak.name, ak.key)
			}
		})
	}
}

func TestIBCAttributeKeyConstants(t *testing.T) {
	ibcAttributes := []struct {
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

	for _, ak := range ibcAttributes {
		t.Run(ak.name, func(t *testing.T) {
			if ak.key == "" {
				t.Errorf("%s is empty", ak.name)
			}

			if strings.ToLower(ak.key) != ak.key {
				t.Errorf("%s = %q should be lowercase", ak.name, ak.key)
			}
		})
	}
}

func TestEventTypes_Unique(t *testing.T) {
	// Collect all event types
	eventTypes := []string{
		EventTypeOraclePriceUpdate,
		EventTypeOraclePriceSubmitted,
		EventTypeOraclePriceAggregated,
		EventTypeOracleVote,
		EventTypeOracleVoteAggregated,
		EventTypeOracleFeederDelegated,
		EventTypeOracleSlash,
		EventTypeOracleSlashOutlier,
		EventTypeOracleJail,
		EventTypeOracleOutlier,
		EventTypeOracleCrossChainPrice,
		EventTypeOraclePriceRelay,
		EventTypeOracleParamsUpdated,
		EventTypeCircuitBreakerOpen,
		EventTypeCircuitBreakerClose,
		EventTypePriceOverride,
		EventTypePriceOverrideClear,
		EventTypeSlashingDisabled,
		EventTypeSlashingEnabled,
		EventTypeEmergencyPause,
		EventTypeEmergencyResume,
		EventTypeChannelOpen,
		EventTypeChannelOpenAck,
		EventTypeChannelOpenConfirm,
		EventTypeChannelClose,
		EventTypePacketReceive,
		EventTypePacketAck,
		EventTypePacketTimeout,
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, et := range eventTypes {
		if seen[et] {
			t.Errorf("Duplicate event type: %s", et)
		}
		seen[et] = true
	}
}

func TestAttributeKeys_Unique(t *testing.T) {
	// Collect all attribute keys
	attributeKeys := []string{
		AttributeKeyAsset,
		AttributeKeyAssets,
		AttributeKeyDenom,
		AttributeKeyPrice,
		AttributeKeyPrices,
		AttributeKeyMedian,
		AttributeKeyMean,
		AttributeKeyStdDev,
		AttributeKeyMAD,
		AttributeKeyValidator,
		AttributeKeyValidators,
		AttributeKeyFeeder,
		AttributeKeyDelegate,
		AttributeKeyVotingPower,
		AttributeKeyNumValidators,
		AttributeKeyTotalVotingPower,
		AttributeKeyDeviation,
		AttributeKeyDeviationPercent,
		AttributeKeyThreshold,
		AttributeKeyReason,
		AttributeKeySlashFraction,
		AttributeKeySlashAmount,
		AttributeKeySeverity,
		AttributeKeyJailed,
		AttributeKeyBlockHeight,
		AttributeKeyTimestamp,
		AttributeKeyNumSubmissions,
		AttributeKeyNumOutliers,
		AttributeKeyConfidence,
		AttributeKeySourceChain,
		AttributeKeyTargetChain,
		AttributeKeyStatus,
		AttributeKeyError,
		AttributeKeyActor,
		AttributeKeyPair,
		AttributeKeyFeedType,
		AttributeKeyChannelID,
		AttributeKeyPortID,
		AttributeKeyCounterpartyPortID,
		AttributeKeyCounterpartyChannelID,
		AttributeKeyPacketType,
		AttributeKeySequence,
		AttributeKeyAckSuccess,
		AttributeKeyPendingOperations,
		AttributeKeyPausedBy,
		AttributeKeyPauseReason,
		AttributeKeyResumeReason,
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, ak := range attributeKeys {
		if seen[ak] {
			t.Errorf("Duplicate attribute key: %s", ak)
		}
		seen[ak] = true
	}
}

func TestEventTypeNamingConvention(t *testing.T) {
	// All oracle event types should follow module_action format
	eventTypes := map[string]string{
		"price_update":           EventTypeOraclePriceUpdate,
		"price_submitted":        EventTypeOraclePriceSubmitted,
		"price_aggregated":       EventTypeOraclePriceAggregated,
		"vote":                   EventTypeOracleVote,
		"vote_aggregated":        EventTypeOracleVoteAggregated,
		"feeder_delegated":       EventTypeOracleFeederDelegated,
		"slash":                  EventTypeOracleSlash,
		"slash_outlier":          EventTypeOracleSlashOutlier,
		"jail":                   EventTypeOracleJail,
		"outlier":                EventTypeOracleOutlier,
		"cross_chain_price":      EventTypeOracleCrossChainPrice,
		"price_relay":            EventTypeOraclePriceRelay,
		"params_updated":         EventTypeOracleParamsUpdated,
		"circuit_breaker_open":   EventTypeCircuitBreakerOpen,
		"circuit_breaker_close":  EventTypeCircuitBreakerClose,
		"price_override":         EventTypePriceOverride,
		"price_override_clear":   EventTypePriceOverrideClear,
		"slashing_disabled":      EventTypeSlashingDisabled,
		"slashing_enabled":       EventTypeSlashingEnabled,
		"emergency_pause":        EventTypeEmergencyPause,
		"emergency_resume":       EventTypeEmergencyResume,
	}

	for action, eventType := range eventTypes {
		expected := "oracle_" + action
		if eventType != expected {
			t.Errorf("Event type mismatch: expected %q, got %q", expected, eventType)
		}
	}
}

func TestAttributeKeyNamingConvention(t *testing.T) {
	// Test that attribute keys follow snake_case convention
	tests := []struct {
		key       string
		shouldErr bool
	}{
		{"asset", false},
		{"voting_power", false},
		{"num_validators", false},
		{"deviation_percent", false},
		{"Asset", true},          // uppercase
		{"votingPower", true},    // camelCase
		{"num-validators", true}, // kebab-case
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			hasUppercase := tt.key != strings.ToLower(tt.key)
			hasHyphen := strings.Contains(tt.key, "-")
			hasCamelCase := false
			for i := 1; i < len(tt.key); i++ {
				if tt.key[i] >= 'A' && tt.key[i] <= 'Z' {
					hasCamelCase = true
					break
				}
			}

			isInvalid := hasUppercase || hasHyphen || hasCamelCase
			if isInvalid != tt.shouldErr {
				if tt.shouldErr {
					t.Errorf("Key %q should be invalid (snake_case violation)", tt.key)
				} else {
					t.Errorf("Key %q should be valid snake_case", tt.key)
				}
			}
		})
	}
}

func TestEventTypeCategories(t *testing.T) {
	// Verify event types are categorized correctly
	categories := map[string][]string{
		"price": {
			EventTypeOraclePriceUpdate,
			EventTypeOraclePriceSubmitted,
			EventTypeOraclePriceAggregated,
			EventTypeOracleCrossChainPrice,
			EventTypeOraclePriceRelay,
			EventTypePriceOverride,
			EventTypePriceOverrideClear,
		},
		"voting": {
			EventTypeOracleVote,
			EventTypeOracleVoteAggregated,
			EventTypeOracleFeederDelegated,
		},
		"security": {
			EventTypeOracleSlash,
			EventTypeOracleSlashOutlier,
			EventTypeOracleJail,
			EventTypeOracleOutlier,
		},
		"circuit_breaker": {
			EventTypeCircuitBreakerOpen,
			EventTypeCircuitBreakerClose,
			EventTypeSlashingDisabled,
			EventTypeSlashingEnabled,
		},
		"emergency": {
			EventTypeEmergencyPause,
			EventTypeEmergencyResume,
		},
	}

	for category, events := range categories {
		t.Run(category, func(t *testing.T) {
			for _, event := range events {
				if event == "" {
					t.Errorf("Empty event type in category %s", category)
				}
				if !strings.HasPrefix(event, "oracle_") {
					t.Errorf("Event %q in category %s should start with 'oracle_'", event, category)
				}
			}
		})
	}
}
