package integration

// PriceOverride represents a price override in the Oracle module
// This is a simplified version for the control center integration
// The actual proto definition would live in x/oracle/types/oracle.proto
type PriceOverride struct {
	Pair      string `json:"pair"`
	Price     string `json:"price"`
	ExpiresAt int64  `json:"expires_at"`
	Actor     string `json:"actor"`
	Reason    string `json:"reason"`
}

// JobCancellation represents a job cancellation in the Compute module
// This is a simplified version for the control center integration
// The actual proto definition would live in x/compute/types/compute.proto
type JobCancellation struct {
	JobID     string `json:"job_id"`
	Actor     string `json:"actor"`
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}

// ReputationOverride represents a reputation override in the Compute module
// This is a simplified version for the control center integration
// The actual proto definition would live in x/compute/types/compute.proto
type ReputationOverride struct {
	Provider  string `json:"provider"`
	Score     int64  `json:"score"`
	Actor     string `json:"actor"`
	Reason    string `json:"reason"`
	Timestamp int64  `json:"timestamp"`
}
