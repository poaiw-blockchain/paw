package types

import (
	"encoding/json"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// IBC Packet Types for Compute Module
//
// This file defines all IBC packet structures for cross-chain compute operations.

const (
	// Compute IBC version
	IBCVersion = "paw-compute-1"

	// Packet types
	DiscoverProvidersType = "discover_providers"
	SubmitJobType         = "submit_job"
	JobResultType         = "job_result"
	JobStatusType         = "job_status"
	ReleaseEscrowType     = "release_escrow"
)

// IBCPacketData is the base interface for all Compute IBC packets
type IBCPacketData interface {
	ValidateBasic() error
	GetType() string
}

// DiscoverProvidersPacketData discovers compute providers
type DiscoverProvidersPacketData struct {
	Type         string   `json:"type"`
	Capabilities []string `json:"capabilities,omitempty"`
	MaxPrice     math.LegacyDec  `json:"max_price,omitempty"`
	Requester    string   `json:"requester"`
}

func (p DiscoverProvidersPacketData) ValidateBasic() error {
	if p.Type != DiscoverProvidersType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if _, err := sdk.AccAddressFromBech32(p.Requester); err != nil {
		return errors.Wrapf(ErrInvalidPacket, "invalid requester address: %s", err)
	}
	return nil
}

func (p DiscoverProvidersPacketData) GetType() string {
	return p.Type
}

func (p DiscoverProvidersPacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// ProviderInfo contains provider information
type ProviderInfo struct {
	ProviderID   string   `json:"provider_id"`
	Address      string   `json:"address"`
	Capabilities []string `json:"capabilities"`
	PricePerUnit math.LegacyDec  `json:"price_per_unit"`
	Reputation   math.LegacyDec  `json:"reputation"`
}

// DiscoverProvidersAcknowledgement returns discovered providers
type DiscoverProvidersAcknowledgement struct {
	Success   bool           `json:"success"`
	Providers []ProviderInfo `json:"providers,omitempty"`
	Error     string         `json:"error,omitempty"`
}

func (a DiscoverProvidersAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// JobRequirements specifies computational requirements
type JobRequirements struct {
	CPUCores    uint32 `json:"cpu_cores"`
	MemoryMB    uint32 `json:"memory_mb"`
	StorageGB   uint32 `json:"storage_gb"`
	GPURequired bool   `json:"gpu_required"`
	TEERequired bool   `json:"tee_required"`
	MaxDuration uint64 `json:"max_duration"` // seconds
}

// SubmitJobPacketData submits a compute job
type SubmitJobPacketData struct {
	Type         string          `json:"type"`
	JobID        string          `json:"job_id"`
	JobType      string          `json:"job_type"` // "wasm", "docker", "tee"
	JobData      []byte          `json:"job_data"`
	Requirements JobRequirements `json:"requirements"`
	Provider     string          `json:"provider"`
	Requester    string          `json:"requester"`
	EscrowProof  []byte          `json:"escrow_proof"`
	Timeout      uint64          `json:"timeout"`
}

func (p SubmitJobPacketData) ValidateBasic() error {
	if p.Type != SubmitJobType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.JobID == "" {
		return errors.Wrap(ErrInvalidPacket, "job ID cannot be empty")
	}
	if p.JobType == "" {
		return errors.Wrap(ErrInvalidPacket, "job type cannot be empty")
	}
	if len(p.JobData) == 0 {
		return errors.Wrap(ErrInvalidPacket, "job data cannot be empty")
	}
	if p.Provider == "" {
		return errors.Wrap(ErrInvalidPacket, "provider cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(p.Requester); err != nil {
		return errors.Wrapf(ErrInvalidPacket, "invalid requester address: %s", err)
	}
	if p.Requirements.CPUCores == 0 {
		return errors.Wrap(ErrInvalidPacket, "CPU cores must be positive")
	}
	if p.Requirements.MemoryMB == 0 {
		return errors.Wrap(ErrInvalidPacket, "memory must be positive")
	}
	return nil
}

func (p SubmitJobPacketData) GetType() string {
	return p.Type
}

func (p SubmitJobPacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// SubmitJobAcknowledgement acknowledges job submission
type SubmitJobAcknowledgement struct {
	Success       bool   `json:"success"`
	JobID         string `json:"job_id,omitempty"`
	Status        string `json:"status,omitempty"`
	EstimatedTime uint64 `json:"estimated_time,omitempty"` // seconds
	Error         string `json:"error,omitempty"`
}

func (a SubmitJobAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// JobResult contains computation result
type JobResult struct {
	ResultData      []byte   `json:"result_data"`
	ResultHash      string   `json:"result_hash"`
	ComputeTime     uint64   `json:"compute_time"` // milliseconds
	ZKProof         []byte   `json:"zk_proof,omitempty"`
	AttestationSigs [][]byte `json:"attestation_sigs,omitempty"`
	Timestamp       int64    `json:"timestamp"`
}

// JobResultPacketData contains computation result
type JobResultPacketData struct {
	Type     string    `json:"type"`
	JobID    string    `json:"job_id"`
	Result   JobResult `json:"result"`
	Provider string    `json:"provider"`
}

func (p JobResultPacketData) ValidateBasic() error {
	if p.Type != JobResultType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.JobID == "" {
		return errors.Wrap(ErrInvalidPacket, "job ID cannot be empty")
	}
	if len(p.Result.ResultData) == 0 {
		return errors.Wrap(ErrInvalidPacket, "result data cannot be empty")
	}
	if p.Result.ResultHash == "" {
		return errors.Wrap(ErrInvalidPacket, "result hash cannot be empty")
	}
	if p.Provider == "" {
		return errors.Wrap(ErrInvalidPacket, "provider cannot be empty")
	}
	return nil
}

func (p JobResultPacketData) GetType() string {
	return p.Type
}

func (p JobResultPacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// JobStatusPacketData queries job status
type JobStatusPacketData struct {
	Type      string `json:"type"`
	JobID     string `json:"job_id"`
	Requester string `json:"requester"`
}

func (p JobStatusPacketData) ValidateBasic() error {
	if p.Type != JobStatusType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.JobID == "" {
		return errors.Wrap(ErrInvalidPacket, "job ID cannot be empty")
	}
	if _, err := sdk.AccAddressFromBech32(p.Requester); err != nil {
		return errors.Wrapf(ErrInvalidPacket, "invalid requester address: %s", err)
	}
	return nil
}

func (p JobStatusPacketData) GetType() string {
	return p.Type
}

func (p JobStatusPacketData) GetBytes() ([]byte, error) {
	return json.Marshal(p)
}

// JobStatusAcknowledgement returns job status
type JobStatusAcknowledgement struct {
	Success  bool   `json:"success"`
	JobID    string `json:"job_id,omitempty"`
	Status   string `json:"status,omitempty"` // "pending", "running", "completed", "failed"
	Progress uint32 `json:"progress,omitempty"` // 0-100
	Error    string `json:"error,omitempty"`
}

func (a JobStatusAcknowledgement) GetBytes() ([]byte, error) {
	return json.Marshal(a)
}

// ReleaseEscrowPacketData releases escrowed funds
type ReleaseEscrowPacketData struct {
	Type     string   `json:"type"`
	JobID    string   `json:"job_id"`
	Provider string   `json:"provider"`
	Amount   math.Int `json:"amount"`
}

func (p ReleaseEscrowPacketData) ValidateBasic() error {
	if p.Type != ReleaseEscrowType {
		return errors.Wrapf(ErrInvalidPacket, "invalid packet type: %s", p.Type)
	}
	if p.JobID == "" {
		return errors.Wrap(ErrInvalidPacket, "job ID cannot be empty")
	}
	if p.Provider == "" {
		return errors.Wrap(ErrInvalidPacket, "provider cannot be empty")
	}
	if p.Amount.IsNil() || !p.Amount.IsPositive() {
		return errors.Wrap(ErrInvalidPacket, "amount must be positive")
	}
	return nil
}

func (p ReleaseEscrowPacketData) GetType() string {
	return p.Type
}

func (p ReleaseEscrowPacketData) GetBytes() ([]byte, error) {
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
	case DiscoverProvidersType:
		var packet DiscoverProvidersPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal discover providers packet")
		}
		return packet, nil

	case SubmitJobType:
		var packet SubmitJobPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal submit job packet")
		}
		return packet, nil

	case JobResultType:
		var packet JobResultPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal job result packet")
		}
		return packet, nil

	case JobStatusType:
		var packet JobStatusPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal job status packet")
		}
		return packet, nil

	case ReleaseEscrowType:
		var packet ReleaseEscrowPacketData
		if err := json.Unmarshal(data, &packet); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal release escrow packet")
		}
		return packet, nil

	default:
		return nil, errors.Wrapf(ErrInvalidPacket, "unknown packet type: %s", basePacket.Type)
	}
}

// ErrInvalidPacket is defined in errors.go
