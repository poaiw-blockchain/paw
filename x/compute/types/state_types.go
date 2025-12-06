package types

import (
	"cosmossdk.io/math"
)

// ComputeProvider represents a compute resource provider
type ComputeProvider struct {
	Address       string   `json:"address"`
	Stake         math.Int `json:"stake"`
	Active        bool     `json:"active"`
	Reputation    float64  `json:"reputation"`
	TotalJobs     uint64   `json:"total_jobs"`
	CompletedJobs uint64   `json:"completed_jobs"`
	FailedJobs    uint64   `json:"failed_jobs"`
	LastActive    int64    `json:"last_active"`
}

// ComputeRequest represents a compute job request
type ComputeRequest struct {
	ID              uint64   `json:"id"`
	Requester       string   `json:"requester"`
	Provider        string   `json:"provider,omitempty"`
	ContainerImage  string   `json:"container_image"`
	InputData       []byte   `json:"input_data,omitempty"`
	ResourceSpec    Resource `json:"resource_spec"`
	EscrowAmount    math.Int `json:"escrow_amount"`
	Status          string   `json:"status"` // pending, assigned, running, completed, failed, cancelled
	SubmittedHeight int64    `json:"submitted_height"`
	Timeout         int64    `json:"timeout"`
}

// Resource specifies computational resource requirements
type Resource struct {
	CPUCores  uint32 `json:"cpu_cores"`
	MemoryMB  uint32 `json:"memory_mb"`
	StorageGB uint32 `json:"storage_gb"`
	GPUs      uint32 `json:"gpus,omitempty"`
}

// ComputeResult represents the result of a compute job
type ComputeResult struct {
	RequestID    uint64 `json:"request_id"`
	Provider     string `json:"provider"`
	ResultData   []byte `json:"result_data"`
	ResultHash   string `json:"result_hash"`
	Verified     bool   `json:"verified"`
	SubmittedAt  int64  `json:"submitted_at"`
	VerifiedAt   int64  `json:"verified_at,omitempty"`
}
