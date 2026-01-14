package types

import (
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
)

// ============================================================================
// Module Constants Tests
// ============================================================================

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
	if StoreKey != ModuleName {
		t.Errorf("StoreKey %q should match ModuleName %q", StoreKey, ModuleName)
	}
	if RouterKey != ModuleName {
		t.Errorf("RouterKey %q should match ModuleName %q", RouterKey, ModuleName)
	}
	if QuerierRoute != ModuleName {
		t.Errorf("QuerierRoute %q should match ModuleName %q", QuerierRoute, ModuleName)
	}
	if PortID != ModuleName {
		t.Errorf("PortID %q should match ModuleName %q", PortID, ModuleName)
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

// ============================================================================
// IBC Event Type Constants Tests
// ============================================================================

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

			if strings.ToLower(ak.key) != ak.key {
				t.Errorf("%s = %q should be lowercase", ak.name, ak.key)
			}

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
			if key != strings.ToLower(key) {
				t.Errorf("Attribute key %q should be lowercase", key)
			}
			if strings.Contains(key, "-") {
				t.Errorf("Attribute key %q should use underscores, not hyphens", key)
			}
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

// ============================================================================
// State Types Tests (SlashingInfo, TWAPDataPoint, AggregateExchangeRate*)
// ============================================================================

func TestSlashingInfo_Creation(t *testing.T) {
	info := SlashingInfo{
		Validator:     "cosmosvaloper1abc...",
		MissCount:     5,
		SlashedAmount: math.LegacyNewDecWithPrec(100, 2), // 1.00
		SlashedHeight: 1000,
		JailedUntil:   2000,
	}

	if info.Validator == "" {
		t.Error("Validator should not be empty")
	}
	if info.MissCount != 5 {
		t.Errorf("Expected MissCount 5, got %d", info.MissCount)
	}
	if info.SlashedHeight != 1000 {
		t.Errorf("Expected SlashedHeight 1000, got %d", info.SlashedHeight)
	}
	if info.JailedUntil != 2000 {
		t.Errorf("Expected JailedUntil 2000, got %d", info.JailedUntil)
	}
}

func TestSlashingInfo_ZeroValues(t *testing.T) {
	info := SlashingInfo{}

	if info.Validator != "" {
		t.Error("Default Validator should be empty")
	}
	if info.MissCount != 0 {
		t.Error("Default MissCount should be 0")
	}
	if info.SlashedHeight != 0 {
		t.Error("Default SlashedHeight should be 0")
	}
	if info.JailedUntil != 0 {
		t.Error("Default JailedUntil should be 0")
	}
}

func TestTWAPDataPoint_Creation(t *testing.T) {
	now := time.Now().Unix()
	dataPoint := TWAPDataPoint{
		Price:       math.LegacyNewDec(50000),
		Timestamp:   now,
		Volume:      math.LegacyNewDec(1000000),
		BlockHeight: 100,
	}

	if !dataPoint.Price.Equal(math.LegacyNewDec(50000)) {
		t.Errorf("Expected price 50000, got %s", dataPoint.Price)
	}
	if dataPoint.Timestamp != now {
		t.Errorf("Expected timestamp %d, got %d", now, dataPoint.Timestamp)
	}
	if !dataPoint.Volume.Equal(math.LegacyNewDec(1000000)) {
		t.Errorf("Expected volume 1000000, got %s", dataPoint.Volume)
	}
	if dataPoint.BlockHeight != 100 {
		t.Errorf("Expected block height 100, got %d", dataPoint.BlockHeight)
	}
}

func TestTWAPDataPoint_ZeroValues(t *testing.T) {
	dp := TWAPDataPoint{}

	if dp.BlockHeight != 0 {
		t.Error("Default BlockHeight should be 0")
	}
	if dp.Timestamp != 0 {
		t.Error("Default Timestamp should be 0")
	}
}

func TestAggregateExchangeRatePrevote_Creation(t *testing.T) {
	prevote := AggregateExchangeRatePrevote{
		Hash:        "abc123hash",
		Voter:       "cosmosvaloper1voter...",
		SubmitBlock: 500,
	}

	if prevote.Hash != "abc123hash" {
		t.Errorf("Expected hash abc123hash, got %s", prevote.Hash)
	}
	if prevote.Voter == "" {
		t.Error("Voter should not be empty")
	}
	if prevote.SubmitBlock != 500 {
		t.Errorf("Expected SubmitBlock 500, got %d", prevote.SubmitBlock)
	}
}

func TestAggregateExchangeRatePrevote_EmptyHash(t *testing.T) {
	prevote := AggregateExchangeRatePrevote{
		Hash:        "",
		Voter:       "cosmosvaloper1voter...",
		SubmitBlock: 500,
	}

	if prevote.Hash != "" {
		t.Error("Hash should be empty")
	}
}

func TestAggregateExchangeRateVote_Creation(t *testing.T) {
	vote := AggregateExchangeRateVote{
		ExchangeRates: []math.LegacyDec{
			math.LegacyNewDec(50000),
			math.LegacyNewDec(3000),
			math.LegacyNewDec(1),
		},
		Voter: "cosmosvaloper1voter...",
	}

	if len(vote.ExchangeRates) != 3 {
		t.Errorf("Expected 3 exchange rates, got %d", len(vote.ExchangeRates))
	}
	if vote.Voter == "" {
		t.Error("Voter should not be empty")
	}
}

func TestAggregateExchangeRateVote_EmptyRates(t *testing.T) {
	vote := AggregateExchangeRateVote{
		ExchangeRates: []math.LegacyDec{},
		Voter:         "cosmosvaloper1voter...",
	}

	if len(vote.ExchangeRates) != 0 {
		t.Errorf("Expected 0 exchange rates, got %d", len(vote.ExchangeRates))
	}
}

func TestAggregateExchangeRateVote_NilRates(t *testing.T) {
	vote := AggregateExchangeRateVote{
		ExchangeRates: nil,
		Voter:         "cosmosvaloper1voter...",
	}

	if vote.ExchangeRates != nil {
		t.Error("ExchangeRates should be nil")
	}
}

// ============================================================================
// Protobuf Type Tests (Price, ValidatorPrice, ValidatorOracle, PriceSnapshot)
// ============================================================================

func TestPrice_Creation(t *testing.T) {
	price := Price{
		Asset:         "BTC",
		Price:         math.LegacyNewDec(50000),
		BlockHeight:   1000,
		BlockTime:     time.Now().Unix(),
		NumValidators: 10,
	}

	if price.Asset != "BTC" {
		t.Errorf("Expected asset BTC, got %s", price.Asset)
	}
	if price.BlockHeight != 1000 {
		t.Errorf("Expected block height 1000, got %d", price.BlockHeight)
	}
	if price.NumValidators != 10 {
		t.Errorf("Expected 10 validators, got %d", price.NumValidators)
	}
}

func TestPrice_GetterMethods(t *testing.T) {
	price := &Price{
		Asset:         "ETH",
		Price:         math.LegacyNewDec(3000),
		BlockHeight:   2000,
		BlockTime:     1234567890,
		NumValidators: 5,
	}

	if price.GetAsset() != "ETH" {
		t.Errorf("GetAsset() = %s, want ETH", price.GetAsset())
	}
	if price.GetBlockHeight() != 2000 {
		t.Errorf("GetBlockHeight() = %d, want 2000", price.GetBlockHeight())
	}
	if price.GetBlockTime() != 1234567890 {
		t.Errorf("GetBlockTime() = %d, want 1234567890", price.GetBlockTime())
	}
	if price.GetNumValidators() != 5 {
		t.Errorf("GetNumValidators() = %d, want 5", price.GetNumValidators())
	}
}

func TestPrice_NilReceiver(t *testing.T) {
	var price *Price

	if price.GetAsset() != "" {
		t.Error("GetAsset() on nil should return empty string")
	}
	if price.GetBlockHeight() != 0 {
		t.Error("GetBlockHeight() on nil should return 0")
	}
	if price.GetBlockTime() != 0 {
		t.Error("GetBlockTime() on nil should return 0")
	}
	if price.GetNumValidators() != 0 {
		t.Error("GetNumValidators() on nil should return 0")
	}
}

func TestValidatorPrice_Creation(t *testing.T) {
	vp := ValidatorPrice{
		ValidatorAddr: "cosmosvaloper1abc...",
		Asset:         "BTC",
		Price:         math.LegacyNewDec(50000),
		BlockHeight:   1000,
		VotingPower:   100,
	}

	if vp.ValidatorAddr == "" {
		t.Error("ValidatorAddr should not be empty")
	}
	if vp.Asset != "BTC" {
		t.Errorf("Expected asset BTC, got %s", vp.Asset)
	}
	if vp.VotingPower != 100 {
		t.Errorf("Expected voting power 100, got %d", vp.VotingPower)
	}
}

func TestValidatorPrice_GetterMethods(t *testing.T) {
	vp := &ValidatorPrice{
		ValidatorAddr: "cosmosvaloper1xyz...",
		Asset:         "ATOM",
		Price:         math.LegacyNewDec(10),
		BlockHeight:   500,
		VotingPower:   50,
	}

	if vp.GetValidatorAddr() != "cosmosvaloper1xyz..." {
		t.Errorf("GetValidatorAddr() = %s, want cosmosvaloper1xyz...", vp.GetValidatorAddr())
	}
	if vp.GetAsset() != "ATOM" {
		t.Errorf("GetAsset() = %s, want ATOM", vp.GetAsset())
	}
	if vp.GetBlockHeight() != 500 {
		t.Errorf("GetBlockHeight() = %d, want 500", vp.GetBlockHeight())
	}
	if vp.GetVotingPower() != 50 {
		t.Errorf("GetVotingPower() = %d, want 50", vp.GetVotingPower())
	}
}

func TestValidatorPrice_NilReceiver(t *testing.T) {
	var vp *ValidatorPrice

	if vp.GetValidatorAddr() != "" {
		t.Error("GetValidatorAddr() on nil should return empty string")
	}
	if vp.GetAsset() != "" {
		t.Error("GetAsset() on nil should return empty string")
	}
	if vp.GetBlockHeight() != 0 {
		t.Error("GetBlockHeight() on nil should return 0")
	}
	if vp.GetVotingPower() != 0 {
		t.Error("GetVotingPower() on nil should return 0")
	}
}

func TestValidatorOracle_Creation(t *testing.T) {
	vo := ValidatorOracle{
		ValidatorAddr:    "cosmosvaloper1abc...",
		MissCounter:      5,
		TotalSubmissions: 100,
		IsActive:         true,
		GeographicRegion: "na",
		IpAddress:        "192.168.1.1",
		Asn:              12345,
	}

	if vo.MissCounter != 5 {
		t.Errorf("Expected MissCounter 5, got %d", vo.MissCounter)
	}
	if vo.TotalSubmissions != 100 {
		t.Errorf("Expected TotalSubmissions 100, got %d", vo.TotalSubmissions)
	}
	if !vo.IsActive {
		t.Error("Expected IsActive to be true")
	}
	if vo.GeographicRegion != "na" {
		t.Errorf("Expected region na, got %s", vo.GeographicRegion)
	}
	if vo.Asn != 12345 {
		t.Errorf("Expected ASN 12345, got %d", vo.Asn)
	}
}

func TestValidatorOracle_GetterMethods(t *testing.T) {
	vo := &ValidatorOracle{
		ValidatorAddr:    "cosmosvaloper1test...",
		MissCounter:      10,
		TotalSubmissions: 200,
		IsActive:         false,
		GeographicRegion: "eu",
		IpAddress:        "10.0.0.1",
		Asn:              54321,
	}

	if vo.GetValidatorAddr() != "cosmosvaloper1test..." {
		t.Errorf("GetValidatorAddr() = %s, want cosmosvaloper1test...", vo.GetValidatorAddr())
	}
	if vo.GetMissCounter() != 10 {
		t.Errorf("GetMissCounter() = %d, want 10", vo.GetMissCounter())
	}
	if vo.GetTotalSubmissions() != 200 {
		t.Errorf("GetTotalSubmissions() = %d, want 200", vo.GetTotalSubmissions())
	}
	if vo.GetIsActive() != false {
		t.Error("GetIsActive() should be false")
	}
	if vo.GetGeographicRegion() != "eu" {
		t.Errorf("GetGeographicRegion() = %s, want eu", vo.GetGeographicRegion())
	}
	if vo.GetIpAddress() != "10.0.0.1" {
		t.Errorf("GetIpAddress() = %s, want 10.0.0.1", vo.GetIpAddress())
	}
	if vo.GetAsn() != 54321 {
		t.Errorf("GetAsn() = %d, want 54321", vo.GetAsn())
	}
}

func TestValidatorOracle_NilReceiver(t *testing.T) {
	var vo *ValidatorOracle

	if vo.GetValidatorAddr() != "" {
		t.Error("GetValidatorAddr() on nil should return empty string")
	}
	if vo.GetMissCounter() != 0 {
		t.Error("GetMissCounter() on nil should return 0")
	}
	if vo.GetTotalSubmissions() != 0 {
		t.Error("GetTotalSubmissions() on nil should return 0")
	}
	if vo.GetIsActive() != false {
		t.Error("GetIsActive() on nil should return false")
	}
	if vo.GetGeographicRegion() != "" {
		t.Error("GetGeographicRegion() on nil should return empty string")
	}
	if vo.GetIpAddress() != "" {
		t.Error("GetIpAddress() on nil should return empty string")
	}
	if vo.GetAsn() != 0 {
		t.Error("GetAsn() on nil should return 0")
	}
}

func TestAuthorizedChannel_Creation(t *testing.T) {
	channel := AuthorizedChannel{
		PortId:    "oracle",
		ChannelId: "channel-0",
	}

	if channel.PortId != "oracle" {
		t.Errorf("Expected port oracle, got %s", channel.PortId)
	}
	if channel.ChannelId != "channel-0" {
		t.Errorf("Expected channel channel-0, got %s", channel.ChannelId)
	}
}

func TestAuthorizedChannel_GetterMethods(t *testing.T) {
	channel := &AuthorizedChannel{
		PortId:    "transfer",
		ChannelId: "channel-1",
	}

	if channel.GetPortId() != "transfer" {
		t.Errorf("GetPortId() = %s, want transfer", channel.GetPortId())
	}
	if channel.GetChannelId() != "channel-1" {
		t.Errorf("GetChannelId() = %s, want channel-1", channel.GetChannelId())
	}
}

func TestAuthorizedChannel_NilReceiver(t *testing.T) {
	var channel *AuthorizedChannel

	if channel.GetPortId() != "" {
		t.Error("GetPortId() on nil should return empty string")
	}
	if channel.GetChannelId() != "" {
		t.Error("GetChannelId() on nil should return empty string")
	}
}

// ============================================================================
// Params Getter Tests
// ============================================================================

func TestParams_GetterMethods(t *testing.T) {
	params := &Params{
		VotePeriod:                 30,
		VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
		SlashFraction:              math.LegacyMustNewDecFromStr("0.05"),
		SlashWindow:                10000,
		MinValidPerWindow:          100,
		TwapLookbackWindow:         1000,
		AuthorizedChannels:         []AuthorizedChannel{{PortId: "oracle", ChannelId: "channel-0"}},
		AllowedRegions:             []string{"na", "eu", "apac"},
		MinGeographicRegions:       3,
		MaxValidatorsPerIp:         3,
		MaxValidatorsPerAsn:        5,
		RequireGeographicDiversity: true,
		NonceTtlSeconds:            604800,
		DiversityCheckInterval:     100,
		EnforceRuntimeDiversity:    false,
		EmergencyAdmin:             "cosmos1admin...",
		GeoipCacheTtlSeconds:       3600,
		GeoipCacheMaxEntries:       1000,
	}

	if params.GetVotePeriod() != 30 {
		t.Errorf("GetVotePeriod() = %d, want 30", params.GetVotePeriod())
	}
	if params.GetSlashWindow() != 10000 {
		t.Errorf("GetSlashWindow() = %d, want 10000", params.GetSlashWindow())
	}
	if params.GetMinValidPerWindow() != 100 {
		t.Errorf("GetMinValidPerWindow() = %d, want 100", params.GetMinValidPerWindow())
	}
	if params.GetTwapLookbackWindow() != 1000 {
		t.Errorf("GetTwapLookbackWindow() = %d, want 1000", params.GetTwapLookbackWindow())
	}
	if len(params.GetAuthorizedChannels()) != 1 {
		t.Errorf("GetAuthorizedChannels() length = %d, want 1", len(params.GetAuthorizedChannels()))
	}
	if len(params.GetAllowedRegions()) != 3 {
		t.Errorf("GetAllowedRegions() length = %d, want 3", len(params.GetAllowedRegions()))
	}
	if params.GetMinGeographicRegions() != 3 {
		t.Errorf("GetMinGeographicRegions() = %d, want 3", params.GetMinGeographicRegions())
	}
	if params.GetMaxValidatorsPerIp() != 3 {
		t.Errorf("GetMaxValidatorsPerIp() = %d, want 3", params.GetMaxValidatorsPerIp())
	}
	if params.GetMaxValidatorsPerAsn() != 5 {
		t.Errorf("GetMaxValidatorsPerAsn() = %d, want 5", params.GetMaxValidatorsPerAsn())
	}
	if params.GetRequireGeographicDiversity() != true {
		t.Error("GetRequireGeographicDiversity() should be true")
	}
	if params.GetNonceTtlSeconds() != 604800 {
		t.Errorf("GetNonceTtlSeconds() = %d, want 604800", params.GetNonceTtlSeconds())
	}
	if params.GetDiversityCheckInterval() != 100 {
		t.Errorf("GetDiversityCheckInterval() = %d, want 100", params.GetDiversityCheckInterval())
	}
	if params.GetEnforceRuntimeDiversity() != false {
		t.Error("GetEnforceRuntimeDiversity() should be false")
	}
	if params.GetEmergencyAdmin() != "cosmos1admin..." {
		t.Errorf("GetEmergencyAdmin() = %s, want cosmos1admin...", params.GetEmergencyAdmin())
	}
	if params.GetGeoipCacheTtlSeconds() != 3600 {
		t.Errorf("GetGeoipCacheTtlSeconds() = %d, want 3600", params.GetGeoipCacheTtlSeconds())
	}
	if params.GetGeoipCacheMaxEntries() != 1000 {
		t.Errorf("GetGeoipCacheMaxEntries() = %d, want 1000", params.GetGeoipCacheMaxEntries())
	}
}

func TestParams_NilReceiver(t *testing.T) {
	var params *Params

	if params.GetVotePeriod() != 0 {
		t.Error("GetVotePeriod() on nil should return 0")
	}
	if params.GetSlashWindow() != 0 {
		t.Error("GetSlashWindow() on nil should return 0")
	}
	if params.GetMinValidPerWindow() != 0 {
		t.Error("GetMinValidPerWindow() on nil should return 0")
	}
	if params.GetTwapLookbackWindow() != 0 {
		t.Error("GetTwapLookbackWindow() on nil should return 0")
	}
	if params.GetAuthorizedChannels() != nil {
		t.Error("GetAuthorizedChannels() on nil should return nil")
	}
	if params.GetAllowedRegions() != nil {
		t.Error("GetAllowedRegions() on nil should return nil")
	}
	if params.GetMinGeographicRegions() != 0 {
		t.Error("GetMinGeographicRegions() on nil should return 0")
	}
	if params.GetMaxValidatorsPerIp() != 0 {
		t.Error("GetMaxValidatorsPerIp() on nil should return 0")
	}
	if params.GetMaxValidatorsPerAsn() != 0 {
		t.Error("GetMaxValidatorsPerAsn() on nil should return 0")
	}
	if params.GetRequireGeographicDiversity() != false {
		t.Error("GetRequireGeographicDiversity() on nil should return false")
	}
	if params.GetNonceTtlSeconds() != 0 {
		t.Error("GetNonceTtlSeconds() on nil should return 0")
	}
	if params.GetDiversityCheckInterval() != 0 {
		t.Error("GetDiversityCheckInterval() on nil should return 0")
	}
	if params.GetEnforceRuntimeDiversity() != false {
		t.Error("GetEnforceRuntimeDiversity() on nil should return false")
	}
	if params.GetEmergencyAdmin() != "" {
		t.Error("GetEmergencyAdmin() on nil should return empty string")
	}
	if params.GetGeoipCacheTtlSeconds() != 0 {
		t.Error("GetGeoipCacheTtlSeconds() on nil should return 0")
	}
	if params.GetGeoipCacheMaxEntries() != 0 {
		t.Error("GetGeoipCacheMaxEntries() on nil should return 0")
	}
}

// ============================================================================
// Message Type Constants Tests
// ============================================================================

func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"TypeMsgSubmitPrice", TypeMsgSubmitPrice, "submit_price"},
		{"TypeMsgDelegateFeedConsent", TypeMsgDelegateFeedConsent, "delegate_feed_consent"},
		{"TypeMsgUpdateParams", TypeMsgUpdateParams, "update_params"},
		{"TypeMsgEmergencyPauseOracle", TypeMsgEmergencyPauseOracle, "emergency_pause_oracle"},
		{"TypeMsgResumeOracle", TypeMsgResumeOracle, "resume_oracle"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestMessageTypeConstants_LowercaseSnakeCase(t *testing.T) {
	typeConstants := []string{
		TypeMsgSubmitPrice,
		TypeMsgDelegateFeedConsent,
		TypeMsgUpdateParams,
		TypeMsgEmergencyPauseOracle,
		TypeMsgResumeOracle,
	}

	for _, tc := range typeConstants {
		t.Run(tc, func(t *testing.T) {
			if tc != strings.ToLower(tc) {
				t.Errorf("Type constant %q should be lowercase", tc)
			}
			if strings.Contains(tc, "-") {
				t.Errorf("Type constant %q should use underscores, not hyphens", tc)
			}
		})
	}
}

func TestMessageTypeConstants_Unique(t *testing.T) {
	typeConstants := []string{
		TypeMsgSubmitPrice,
		TypeMsgDelegateFeedConsent,
		TypeMsgUpdateParams,
		TypeMsgEmergencyPauseOracle,
		TypeMsgResumeOracle,
	}

	seen := make(map[string]bool)
	for _, tc := range typeConstants {
		if seen[tc] {
			t.Errorf("Duplicate message type constant: %s", tc)
		}
		seen[tc] = true
	}
}

// ============================================================================
// Price Type Boundary Tests
// ============================================================================

func TestPrice_LargeValues(t *testing.T) {
	// Test with very large price values
	largePrice := Price{
		Asset:         "LARGE",
		Price:         math.LegacyNewDec(1e18), // 10^18
		BlockHeight:   int64(1e12),             // Very large block height
		BlockTime:     int64(1e12),
		NumValidators: 1000,
	}

	if largePrice.GetBlockHeight() != int64(1e12) {
		t.Errorf("Large block height not preserved: got %d", largePrice.GetBlockHeight())
	}
	if largePrice.GetNumValidators() != 1000 {
		t.Errorf("NumValidators should be 1000, got %d", largePrice.GetNumValidators())
	}
}

func TestPrice_ZeroValues(t *testing.T) {
	zeroPrice := Price{}

	if zeroPrice.GetAsset() != "" {
		t.Error("Default asset should be empty")
	}
	if zeroPrice.GetBlockHeight() != 0 {
		t.Error("Default block height should be 0")
	}
	if zeroPrice.GetNumValidators() != 0 {
		t.Error("Default num validators should be 0")
	}
}

// ============================================================================
// ValidatorOracle State Tests
// ============================================================================

func TestValidatorOracle_ActiveInactive(t *testing.T) {
	tests := []struct {
		name     string
		isActive bool
	}{
		{"active validator", true},
		{"inactive validator", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vo := ValidatorOracle{
				ValidatorAddr: "cosmosvaloper1test...",
				IsActive:      tt.isActive,
			}

			if vo.GetIsActive() != tt.isActive {
				t.Errorf("IsActive = %v, want %v", vo.GetIsActive(), tt.isActive)
			}
		})
	}
}

func TestValidatorOracle_RegionValues(t *testing.T) {
	validRegions := []string{"global", "na", "eu", "apac", "latam", "africa"}

	for _, region := range validRegions {
		t.Run(region, func(t *testing.T) {
			vo := ValidatorOracle{
				ValidatorAddr:    "cosmosvaloper1test...",
				GeographicRegion: region,
			}

			if vo.GetGeographicRegion() != region {
				t.Errorf("GeographicRegion = %s, want %s", vo.GetGeographicRegion(), region)
			}
		})
	}
}

// ============================================================================
// AuthorizedChannel Edge Cases
// ============================================================================

func TestAuthorizedChannel_EmptyValues(t *testing.T) {
	tests := []struct {
		name      string
		portId    string
		channelId string
	}{
		{"empty port", "", "channel-0"},
		{"empty channel", "oracle", ""},
		{"both empty", "", ""},
		{"valid", "oracle", "channel-0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := AuthorizedChannel{
				PortId:    tt.portId,
				ChannelId: tt.channelId,
			}

			if channel.GetPortId() != tt.portId {
				t.Errorf("GetPortId() = %s, want %s", channel.GetPortId(), tt.portId)
			}
			if channel.GetChannelId() != tt.channelId {
				t.Errorf("GetChannelId() = %s, want %s", channel.GetChannelId(), tt.channelId)
			}
		})
	}
}

func TestAuthorizedChannel_ChannelNaming(t *testing.T) {
	// Test various channel ID formats
	channelIds := []string{
		"channel-0",
		"channel-1",
		"channel-100",
		"channel-999999",
	}

	for _, channelId := range channelIds {
		t.Run(channelId, func(t *testing.T) {
			channel := AuthorizedChannel{
				PortId:    "oracle",
				ChannelId: channelId,
			}

			if channel.GetChannelId() != channelId {
				t.Errorf("GetChannelId() = %s, want %s", channel.GetChannelId(), channelId)
			}
		})
	}
}

// ============================================================================
// SlashingInfo Edge Cases
// ============================================================================

func TestSlashingInfo_NegativeValues(t *testing.T) {
	// Test with negative slashed height (should be allowed by struct)
	info := SlashingInfo{
		Validator:     "cosmosvaloper1test...",
		SlashedHeight: -1, // Edge case
		JailedUntil:   -1, // Edge case
	}

	if info.SlashedHeight != -1 {
		t.Errorf("SlashedHeight should be -1, got %d", info.SlashedHeight)
	}
	if info.JailedUntil != -1 {
		t.Errorf("JailedUntil should be -1, got %d", info.JailedUntil)
	}
}

func TestSlashingInfo_LargeSlashAmount(t *testing.T) {
	largeAmount := math.LegacyNewDec(1e18)
	info := SlashingInfo{
		Validator:     "cosmosvaloper1test...",
		SlashedAmount: largeAmount,
	}

	if !info.SlashedAmount.Equal(largeAmount) {
		t.Errorf("SlashedAmount = %s, want %s", info.SlashedAmount, largeAmount)
	}
}

// ============================================================================
// TWAPDataPoint Edge Cases
// ============================================================================

func TestTWAPDataPoint_VolumeZero(t *testing.T) {
	dp := TWAPDataPoint{
		Price:       math.LegacyNewDec(50000),
		Timestamp:   time.Now().Unix(),
		Volume:      math.LegacyZeroDec(),
		BlockHeight: 100,
	}

	if !dp.Volume.IsZero() {
		t.Errorf("Volume should be zero, got %s", dp.Volume)
	}
}

func TestTWAPDataPoint_NegativeTimestamp(t *testing.T) {
	// Edge case: negative timestamp (pre-epoch)
	dp := TWAPDataPoint{
		Price:       math.LegacyNewDec(50000),
		Timestamp:   -1,
		BlockHeight: 100,
	}

	if dp.Timestamp != -1 {
		t.Errorf("Timestamp should be -1, got %d", dp.Timestamp)
	}
}

// ============================================================================
// AggregateExchangeRateVote Precision Tests
// ============================================================================

func TestAggregateExchangeRateVote_HighPrecision(t *testing.T) {
	// Test with high precision decimal values
	vote := AggregateExchangeRateVote{
		ExchangeRates: []math.LegacyDec{
			math.LegacyMustNewDecFromStr("0.123456789012345678"), // 18 decimals
			math.LegacyMustNewDecFromStr("50000.000000000000001"),
		},
		Voter: "cosmosvaloper1voter...",
	}

	if len(vote.ExchangeRates) != 2 {
		t.Errorf("Expected 2 exchange rates, got %d", len(vote.ExchangeRates))
	}

	// Verify precision is preserved
	expected := math.LegacyMustNewDecFromStr("0.123456789012345678")
	if !vote.ExchangeRates[0].Equal(expected) {
		t.Errorf("Precision not preserved: got %s, want %s", vote.ExchangeRates[0], expected)
	}
}

// ============================================================================
// MaxAssetLen Tests
// ============================================================================

func TestMaxAssetLen_Constant(t *testing.T) {
	if maxAssetLen != 128 {
		t.Errorf("maxAssetLen = %d, want 128", maxAssetLen)
	}
}
