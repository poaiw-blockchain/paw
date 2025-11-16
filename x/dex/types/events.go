package types

// Event types for the DEX module
const (
	// Circuit Breaker Events
	EventTypeCircuitBreakerTripped = "circuit_breaker_tripped"
	EventTypeCircuitBreakerResumed = "circuit_breaker_resumed"

	// MEV Protection Events
	EventTypeSandwichAttack      = "sandwich_attack_detected"
	EventTypeFrontRunning        = "front_running_detected"
	EventTypePriceImpactExceeded = "price_impact_exceeded"
	EventTypeTimestampOrdering   = "timestamp_ordering_enforced"
	EventTypeSandwichPattern     = "sandwich_pattern_recorded"
	EventTypeMEVBlocked          = "mev_attack_blocked"
)
