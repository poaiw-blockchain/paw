package types

import (
	"encoding/json"
	"strings"
	"testing"

	"cosmossdk.io/math"
)

func TestDiscoverProvidersPacketData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		packet  DiscoverProvidersPacketData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid packet",
			packet: DiscoverProvidersPacketData{
				Type:      DiscoverProvidersType,
				Nonce:     1,
				Timestamp: 1000,
				Requester: validAddress,
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			packet: DiscoverProvidersPacketData{
				Type:      "invalid_type",
				Nonce:     1,
				Timestamp: 1000,
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: DiscoverProvidersPacketData{
				Type:      DiscoverProvidersType,
				Nonce:     0,
				Timestamp: 1000,
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "nonce must be greater than zero",
		},
		{
			name: "zero timestamp",
			packet: DiscoverProvidersPacketData{
				Type:      DiscoverProvidersType,
				Nonce:     1,
				Timestamp: 0,
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
		{
			name: "negative timestamp",
			packet: DiscoverProvidersPacketData{
				Type:      DiscoverProvidersType,
				Nonce:     1,
				Timestamp: -1,
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
		{
			name: "invalid requester address",
			packet: DiscoverProvidersPacketData{
				Type:      DiscoverProvidersType,
				Nonce:     1,
				Timestamp: 1000,
				Requester: invalidAddress,
			},
			wantErr: true,
			errMsg:  "invalid requester address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("DiscoverProvidersPacketData.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("DiscoverProvidersPacketData.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestDiscoverProvidersPacketData_GetType(t *testing.T) {
	packet := DiscoverProvidersPacketData{Type: DiscoverProvidersType}
	if packet.GetType() != DiscoverProvidersType {
		t.Errorf("GetType() = %v, want %v", packet.GetType(), DiscoverProvidersType)
	}
}

func TestDiscoverProvidersPacketData_GetBytes(t *testing.T) {
	packet := DiscoverProvidersPacketData{
		Type:      DiscoverProvidersType,
		Nonce:     1,
		Timestamp: 1000,
		Requester: validAddress,
	}

	bytes, err := packet.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() error = %v", err)
	}

	// Verify we can unmarshal it back
	var unmarshaled DiscoverProvidersPacketData
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal bytes: %v", err)
	}

	if unmarshaled.Nonce != packet.Nonce {
		t.Errorf("Unmarshaled nonce = %v, want %v", unmarshaled.Nonce, packet.Nonce)
	}
}

func TestSubmitJobPacketData_ValidateBasic(t *testing.T) {
	validRequirements := JobRequirements{
		CPUCores:  4,
		MemoryMB:  8192,
		MaxDuration: 3600,
	}

	tests := []struct {
		name    string
		packet  SubmitJobPacketData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid packet",
			packet: SubmitJobPacketData{
				Type:         SubmitJobType,
				Nonce:        1,
				Timestamp:    1000,
				JobID:        "job-123",
				JobType:      "wasm",
				JobData:      []byte("data"),
				Requirements: validRequirements,
				Provider:     "provider-1",
				Requester:    validAddress,
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			packet: SubmitJobPacketData{
				Type:         "invalid",
				Nonce:        1,
				Timestamp:    1000,
				JobID:        "job-123",
				JobType:      "wasm",
				JobData:      []byte("data"),
				Requirements: validRequirements,
				Provider:     "provider-1",
				Requester:    validAddress,
			},
			wantErr: true,
			errMsg:  "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: SubmitJobPacketData{
				Type:         SubmitJobType,
				Nonce:        0,
				Timestamp:    1000,
				JobID:        "job-123",
				JobType:      "wasm",
				JobData:      []byte("data"),
				Requirements: validRequirements,
				Provider:     "provider-1",
				Requester:    validAddress,
			},
			wantErr: true,
			errMsg:  "nonce must be greater than zero",
		},
		{
			name: "zero timestamp",
			packet: SubmitJobPacketData{
				Type:         SubmitJobType,
				Nonce:        1,
				Timestamp:    0,
				JobID:        "job-123",
				JobType:      "wasm",
				JobData:      []byte("data"),
				Requirements: validRequirements,
				Provider:     "provider-1",
				Requester:    validAddress,
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
		{
			name: "empty job ID",
			packet: SubmitJobPacketData{
				Type:         SubmitJobType,
				Nonce:        1,
				Timestamp:    1000,
				JobID:        "",
				JobType:      "wasm",
				JobData:      []byte("data"),
				Requirements: validRequirements,
				Provider:     "provider-1",
				Requester:    validAddress,
			},
			wantErr: true,
			errMsg:  "job ID cannot be empty",
		},
		{
			name: "empty job type",
			packet: SubmitJobPacketData{
				Type:         SubmitJobType,
				Nonce:        1,
				Timestamp:    1000,
				JobID:        "job-123",
				JobType:      "",
				JobData:      []byte("data"),
				Requirements: validRequirements,
				Provider:     "provider-1",
				Requester:    validAddress,
			},
			wantErr: true,
			errMsg:  "job type cannot be empty",
		},
		{
			name: "empty job data",
			packet: SubmitJobPacketData{
				Type:         SubmitJobType,
				Nonce:        1,
				Timestamp:    1000,
				JobID:        "job-123",
				JobType:      "wasm",
				JobData:      []byte{},
				Requirements: validRequirements,
				Provider:     "provider-1",
				Requester:    validAddress,
			},
			wantErr: true,
			errMsg:  "job data cannot be empty",
		},
		{
			name: "empty provider",
			packet: SubmitJobPacketData{
				Type:         SubmitJobType,
				Nonce:        1,
				Timestamp:    1000,
				JobID:        "job-123",
				JobType:      "wasm",
				JobData:      []byte("data"),
				Requirements: validRequirements,
				Provider:     "",
				Requester:    validAddress,
			},
			wantErr: true,
			errMsg:  "provider cannot be empty",
		},
		{
			name: "invalid requester address",
			packet: SubmitJobPacketData{
				Type:         SubmitJobType,
				Nonce:        1,
				Timestamp:    1000,
				JobID:        "job-123",
				JobType:      "wasm",
				JobData:      []byte("data"),
				Requirements: validRequirements,
				Provider:     "provider-1",
				Requester:    invalidAddress,
			},
			wantErr: true,
			errMsg:  "invalid requester address",
		},
		{
			name: "zero CPU cores",
			packet: SubmitJobPacketData{
				Type:      SubmitJobType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				JobType:   "wasm",
				JobData:   []byte("data"),
				Requirements: JobRequirements{
					CPUCores: 0,
					MemoryMB: 8192,
				},
				Provider:  "provider-1",
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "CPU cores must be positive",
		},
		{
			name: "zero memory",
			packet: SubmitJobPacketData{
				Type:      SubmitJobType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				JobType:   "wasm",
				JobData:   []byte("data"),
				Requirements: JobRequirements{
					CPUCores: 4,
					MemoryMB: 0,
				},
				Provider:  "provider-1",
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "memory must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitJobPacketData.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("SubmitJobPacketData.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestJobResultPacketData_ValidateBasic(t *testing.T) {
	validResult := JobResult{
		ResultData: []byte("result"),
		ResultHash: "hash123",
		ComputeTime: 1000,
		Timestamp:  1000,
	}

	tests := []struct {
		name    string
		packet  JobResultPacketData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid packet",
			packet: JobResultPacketData{
				Type:      JobResultType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Result:    validResult,
				Provider:  "provider-1",
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			packet: JobResultPacketData{
				Type:      "invalid",
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Result:    validResult,
				Provider:  "provider-1",
			},
			wantErr: true,
			errMsg:  "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: JobResultPacketData{
				Type:      JobResultType,
				Nonce:     0,
				Timestamp: 1000,
				JobID:     "job-123",
				Result:    validResult,
				Provider:  "provider-1",
			},
			wantErr: true,
			errMsg:  "nonce must be greater than zero",
		},
		{
			name: "zero timestamp",
			packet: JobResultPacketData{
				Type:      JobResultType,
				Nonce:     1,
				Timestamp: 0,
				JobID:     "job-123",
				Result:    validResult,
				Provider:  "provider-1",
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
		{
			name: "empty job ID",
			packet: JobResultPacketData{
				Type:      JobResultType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "",
				Result:    validResult,
				Provider:  "provider-1",
			},
			wantErr: true,
			errMsg:  "job ID cannot be empty",
		},
		{
			name: "empty result data",
			packet: JobResultPacketData{
				Type:      JobResultType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Result: JobResult{
					ResultData: []byte{},
					ResultHash: "hash123",
				},
				Provider: "provider-1",
			},
			wantErr: true,
			errMsg:  "result data cannot be empty",
		},
		{
			name: "empty result hash",
			packet: JobResultPacketData{
				Type:      JobResultType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Result: JobResult{
					ResultData: []byte("result"),
					ResultHash: "",
				},
				Provider: "provider-1",
			},
			wantErr: true,
			errMsg:  "result hash cannot be empty",
		},
		{
			name: "empty provider",
			packet: JobResultPacketData{
				Type:      JobResultType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Result:    validResult,
				Provider:  "",
			},
			wantErr: true,
			errMsg:  "provider cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("JobResultPacketData.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("JobResultPacketData.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestJobStatusPacketData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		packet  JobStatusPacketData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid packet",
			packet: JobStatusPacketData{
				Type:      JobStatusType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Requester: validAddress,
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			packet: JobStatusPacketData{
				Type:      "invalid",
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: JobStatusPacketData{
				Type:      JobStatusType,
				Nonce:     0,
				Timestamp: 1000,
				JobID:     "job-123",
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "nonce must be greater than zero",
		},
		{
			name: "zero timestamp",
			packet: JobStatusPacketData{
				Type:      JobStatusType,
				Nonce:     1,
				Timestamp: 0,
				JobID:     "job-123",
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
		{
			name: "empty job ID",
			packet: JobStatusPacketData{
				Type:      JobStatusType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "",
				Requester: validAddress,
			},
			wantErr: true,
			errMsg:  "job ID cannot be empty",
		},
		{
			name: "invalid requester address",
			packet: JobStatusPacketData{
				Type:      JobStatusType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Requester: invalidAddress,
			},
			wantErr: true,
			errMsg:  "invalid requester address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("JobStatusPacketData.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("JobStatusPacketData.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestReleaseEscrowPacketData_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		packet  ReleaseEscrowPacketData
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid packet",
			packet: ReleaseEscrowPacketData{
				Type:      ReleaseEscrowType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Provider:  "provider-1",
				Amount:    math.NewInt(1000),
			},
			wantErr: false,
		},
		{
			name: "invalid type",
			packet: ReleaseEscrowPacketData{
				Type:      "invalid",
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Provider:  "provider-1",
				Amount:    math.NewInt(1000),
			},
			wantErr: true,
			errMsg:  "invalid packet type",
		},
		{
			name: "zero nonce",
			packet: ReleaseEscrowPacketData{
				Type:      ReleaseEscrowType,
				Nonce:     0,
				Timestamp: 1000,
				JobID:     "job-123",
				Provider:  "provider-1",
				Amount:    math.NewInt(1000),
			},
			wantErr: true,
			errMsg:  "nonce must be greater than zero",
		},
		{
			name: "zero timestamp",
			packet: ReleaseEscrowPacketData{
				Type:      ReleaseEscrowType,
				Nonce:     1,
				Timestamp: 0,
				JobID:     "job-123",
				Provider:  "provider-1",
				Amount:    math.NewInt(1000),
			},
			wantErr: true,
			errMsg:  "timestamp must be positive",
		},
		{
			name: "empty job ID",
			packet: ReleaseEscrowPacketData{
				Type:      ReleaseEscrowType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "",
				Provider:  "provider-1",
				Amount:    math.NewInt(1000),
			},
			wantErr: true,
			errMsg:  "job ID cannot be empty",
		},
		{
			name: "empty provider",
			packet: ReleaseEscrowPacketData{
				Type:      ReleaseEscrowType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Provider:  "",
				Amount:    math.NewInt(1000),
			},
			wantErr: true,
			errMsg:  "provider cannot be empty",
		},
		{
			name: "nil amount",
			packet: ReleaseEscrowPacketData{
				Type:      ReleaseEscrowType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Provider:  "provider-1",
				Amount:    math.Int{},
			},
			wantErr: true,
			errMsg:  "amount must be positive",
		},
		{
			name: "zero amount",
			packet: ReleaseEscrowPacketData{
				Type:      ReleaseEscrowType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Provider:  "provider-1",
				Amount:    math.NewInt(0),
			},
			wantErr: true,
			errMsg:  "amount must be positive",
		},
		{
			name: "negative amount",
			packet: ReleaseEscrowPacketData{
				Type:      ReleaseEscrowType,
				Nonce:     1,
				Timestamp: 1000,
				JobID:     "job-123",
				Provider:  "provider-1",
				Amount:    math.NewInt(-1000),
			},
			wantErr: true,
			errMsg:  "amount must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.packet.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReleaseEscrowPacketData.ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ReleaseEscrowPacketData.ValidateBasic() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestParsePacketData(t *testing.T) {
	tests := []struct {
		name       string
		packetJSON string
		wantType   string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "discover providers packet",
			packetJSON: `{"type":"discover_providers","nonce":1,"timestamp":1000,"requester":"` + validAddress + `"}`,
			wantType:   DiscoverProvidersType,
			wantErr:    false,
		},
		{
			name:       "submit job packet",
			packetJSON: `{"type":"submit_job","nonce":1,"timestamp":1000,"job_id":"job-123","job_type":"wasm","job_data":"ZGF0YQ==","requirements":{"cpu_cores":4,"memory_mb":8192},"provider":"provider-1","requester":"` + validAddress + `"}`,
			wantType:   SubmitJobType,
			wantErr:    false,
		},
		{
			name:       "job result packet",
			packetJSON: `{"type":"job_result","nonce":1,"timestamp":1000,"job_id":"job-123","result":{"result_data":"cmVzdWx0","result_hash":"hash123","compute_time":1000,"timestamp":1000},"provider":"provider-1"}`,
			wantType:   JobResultType,
			wantErr:    false,
		},
		{
			name:       "job status packet",
			packetJSON: `{"type":"job_status","nonce":1,"timestamp":1000,"job_id":"job-123","requester":"` + validAddress + `"}`,
			wantType:   JobStatusType,
			wantErr:    false,
		},
		{
			name:       "release escrow packet",
			packetJSON: `{"type":"release_escrow","nonce":1,"timestamp":1000,"job_id":"job-123","provider":"provider-1","amount":"1000"}`,
			wantType:   ReleaseEscrowType,
			wantErr:    false,
		},
		{
			name:       "unknown packet type",
			packetJSON: `{"type":"unknown_type","nonce":1}`,
			wantType:   "",
			wantErr:    true,
			errMsg:     "unknown packet type",
		},
		{
			name:       "invalid JSON",
			packetJSON: `{invalid json}`,
			wantType:   "",
			wantErr:    true,
			errMsg:     "failed to unmarshal packet data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet, err := ParsePacketData([]byte(tt.packetJSON))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePacketData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ParsePacketData() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if !tt.wantErr && packet != nil {
				if packet.GetType() != tt.wantType {
					t.Errorf("ParsePacketData().GetType() = %v, want %v", packet.GetType(), tt.wantType)
				}
			}
		})
	}
}

func TestIBCVersion(t *testing.T) {
	if IBCVersion != "paw-compute-1" {
		t.Errorf("IBCVersion = %v, want 'paw-compute-1'", IBCVersion)
	}
}

func TestPacketTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"DiscoverProvidersType", DiscoverProvidersType, "discover_providers"},
		{"SubmitJobType", SubmitJobType, "submit_job"},
		{"JobResultType", JobResultType, "job_result"},
		{"JobStatusType", JobStatusType, "job_status"},
		{"ReleaseEscrowType", ReleaseEscrowType, "release_escrow"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.constant, tt.expected)
			}
		})
	}
}

func TestAcknowledgementGetBytes(t *testing.T) {
	t.Run("DiscoverProvidersAcknowledgement", func(t *testing.T) {
		ack := DiscoverProvidersAcknowledgement{
			Nonce:   1,
			Success: true,
		}

		bytes, err := ack.GetBytes()
		if err != nil {
			t.Fatalf("GetBytes() error = %v", err)
		}

		var unmarshaled DiscoverProvidersAcknowledgement
		if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.Nonce != ack.Nonce {
			t.Errorf("Unmarshaled nonce = %v, want %v", unmarshaled.Nonce, ack.Nonce)
		}
	})

	t.Run("SubmitJobAcknowledgement", func(t *testing.T) {
		ack := SubmitJobAcknowledgement{
			Nonce:   1,
			Success: true,
			JobID:   "job-123",
		}

		bytes, err := ack.GetBytes()
		if err != nil {
			t.Fatalf("GetBytes() error = %v", err)
		}

		var unmarshaled SubmitJobAcknowledgement
		if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.JobID != ack.JobID {
			t.Errorf("Unmarshaled JobID = %v, want %v", unmarshaled.JobID, ack.JobID)
		}
	})

	t.Run("JobStatusAcknowledgement", func(t *testing.T) {
		ack := JobStatusAcknowledgement{
			Nonce:   1,
			Success: true,
			Status:  "completed",
		}

		bytes, err := ack.GetBytes()
		if err != nil {
			t.Fatalf("GetBytes() error = %v", err)
		}

		var unmarshaled JobStatusAcknowledgement
		if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.Status != ack.Status {
			t.Errorf("Unmarshaled Status = %v, want %v", unmarshaled.Status, ack.Status)
		}
	})

	t.Run("JobResultAcknowledgement", func(t *testing.T) {
		ack := JobResultAcknowledgement{
			Nonce:      1,
			Success:    true,
			ResultHash: "hash123",
		}

		bytes, err := ack.GetBytes()
		if err != nil {
			t.Fatalf("GetBytes() error = %v", err)
		}

		var unmarshaled JobResultAcknowledgement
		if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if unmarshaled.ResultHash != ack.ResultHash {
			t.Errorf("Unmarshaled ResultHash = %v, want %v", unmarshaled.ResultHash, ack.ResultHash)
		}
	})
}

func BenchmarkParsePacketData(b *testing.B) {
	packetJSON := []byte(`{"type":"submit_job","nonce":1,"timestamp":1000,"job_id":"job-123","job_type":"wasm","job_data":"ZGF0YQ==","requirements":{"cpu_cores":4,"memory_mb":8192},"provider":"provider-1","requester":"` + validAddress + `"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParsePacketData(packetJSON)
	}
}

func BenchmarkValidateBasic(b *testing.B) {
	packet := SubmitJobPacketData{
		Type:      SubmitJobType,
		Nonce:     1,
		Timestamp: 1000,
		JobID:     "job-123",
		JobType:   "wasm",
		JobData:   []byte("data"),
		Requirements: JobRequirements{
			CPUCores: 4,
			MemoryMB: 8192,
		},
		Provider:  "provider-1",
		Requester: validAddress,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = packet.ValidateBasic()
	}
}

// ============================================================================
// GetBytes Tests for Complete Coverage
// ============================================================================

func TestSubmitJobPacketData_GetBytesError(t *testing.T) {
	// Test that GetBytes returns valid JSON bytes
	packet := SubmitJobPacketData{
		Type:      SubmitJobType,
		Nonce:     1,
		Timestamp: 1000,
		JobID:     "job-123",
		JobType:   "wasm",
		JobData:   []byte("test data"),
		Requirements: JobRequirements{
			CPUCores: 4,
			MemoryMB: 8192,
		},
		Provider:  "provider-1",
		Requester: validAddress,
	}

	bytes, err := packet.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() returned unexpected error: %v", err)
	}

	if len(bytes) == 0 {
		t.Error("GetBytes() returned empty bytes")
	}

	// Verify it's valid JSON
	var unmarshaled SubmitJobPacketData
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Errorf("GetBytes() produced invalid JSON: %v", err)
	}
}

func TestJobResultPacketData_GetBytesError(t *testing.T) {
	// Test that GetBytes returns valid JSON bytes
	packet := JobResultPacketData{
		Type:      JobResultType,
		Nonce:     1,
		Timestamp: 1000,
		JobID:     "job-123",
		Provider:  "provider-1",
		Result: JobResult{
			ResultData: []byte("result data"),
			ResultHash: "hash123",
		},
	}

	bytes, err := packet.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() returned unexpected error: %v", err)
	}

	if len(bytes) == 0 {
		t.Error("GetBytes() returned empty bytes")
	}

	// Verify it's valid JSON
	var unmarshaled JobResultPacketData
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Errorf("GetBytes() produced invalid JSON: %v", err)
	}
}

func TestJobStatusPacketData_GetBytesError(t *testing.T) {
	// Test that GetBytes returns valid JSON bytes
	packet := JobStatusPacketData{
		Type:      JobStatusType,
		Nonce:     1,
		Timestamp: 1000,
		JobID:     "job-123",
		Requester: validAddress,
	}

	bytes, err := packet.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() returned unexpected error: %v", err)
	}

	if len(bytes) == 0 {
		t.Error("GetBytes() returned empty bytes")
	}

	// Verify it's valid JSON
	var unmarshaled JobStatusPacketData
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Errorf("GetBytes() produced invalid JSON: %v", err)
	}
}

func TestReleaseEscrowPacketData_GetBytesError(t *testing.T) {
	// Test that GetBytes returns valid JSON bytes
	packet := ReleaseEscrowPacketData{
		Type:      ReleaseEscrowType,
		Nonce:     1,
		Timestamp: 1000,
		JobID:     "job-123",
		Provider:  "provider-1",
		Amount:    math.NewInt(1000000),
	}

	bytes, err := packet.GetBytes()
	if err != nil {
		t.Fatalf("GetBytes() returned unexpected error: %v", err)
	}

	if len(bytes) == 0 {
		t.Error("GetBytes() returned empty bytes")
	}

	// Verify it's valid JSON
	var unmarshaled ReleaseEscrowPacketData
	if err := json.Unmarshal(bytes, &unmarshaled); err != nil {
		t.Errorf("GetBytes() produced invalid JSON: %v", err)
	}
}

// ============================================================================
// Additional GetType Tests for Complete Coverage
// ============================================================================

func TestAllPacketTypes_GetType(t *testing.T) {
	tests := []struct {
		name       string
		packet     interface{ GetType() string }
		expectType string
	}{
		{
			name: "DiscoverProvidersPacketData",
			packet: &DiscoverProvidersPacketData{
				Type: DiscoverProvidersType,
			},
			expectType: DiscoverProvidersType,
		},
		{
			name: "SubmitJobPacketData",
			packet: &SubmitJobPacketData{
				Type: SubmitJobType,
			},
			expectType: SubmitJobType,
		},
		{
			name: "JobResultPacketData",
			packet: &JobResultPacketData{
				Type: JobResultType,
			},
			expectType: JobResultType,
		},
		{
			name: "JobStatusPacketData",
			packet: &JobStatusPacketData{
				Type: JobStatusType,
			},
			expectType: JobStatusType,
		},
		{
			name: "ReleaseEscrowPacketData",
			packet: &ReleaseEscrowPacketData{
				Type: ReleaseEscrowType,
			},
			expectType: ReleaseEscrowType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.packet.GetType(); got != tt.expectType {
				t.Errorf("GetType() = %v, want %v", got, tt.expectType)
			}
		})
	}
}

// ============================================================================
// ParsePacketData Error Cases
// ============================================================================

func TestParsePacketData_InvalidJSON(t *testing.T) {
	invalidJSON := []byte("{invalid json")
	_, err := ParsePacketData(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestParsePacketData_UnknownType(t *testing.T) {
	unknownType := []byte(`{"type": "unknown_type"}`)
	_, err := ParsePacketData(unknownType)
	if err == nil {
		t.Error("Expected error for unknown packet type")
	}
}

func TestParsePacketData_AllTypes(t *testing.T) {
	tests := []struct {
		name       string
		packetJSON string
		expectType string
	}{
		{
			name:       "discover_providers",
			packetJSON: `{"type":"discover_providers","nonce":1,"timestamp":1000,"requester":"` + validAddress + `"}`,
			expectType: DiscoverProvidersType,
		},
		{
			name:       "submit_job",
			packetJSON: `{"type":"submit_job","nonce":1,"timestamp":1000,"job_id":"job-123","job_type":"wasm","job_data":"dGVzdA==","requirements":{"cpu_cores":4,"memory_mb":8192},"provider":"provider-1","requester":"` + validAddress + `"}`,
			expectType: SubmitJobType,
		},
		{
			name:       "job_result",
			packetJSON: `{"type":"job_result","nonce":1,"timestamp":1000,"job_id":"job-123","provider":"provider-1","result":{"result_data":"dGVzdA==","result_hash":"hash123"}}`,
			expectType: JobResultType,
		},
		{
			name:       "job_status",
			packetJSON: `{"type":"job_status","nonce":1,"timestamp":1000,"job_id":"job-123","requester":"` + validAddress + `"}`,
			expectType: JobStatusType,
		},
		{
			name:       "release_escrow",
			packetJSON: `{"type":"release_escrow","nonce":1,"timestamp":1000,"job_id":"job-123","provider":"provider-1","amount":"1000000"}`,
			expectType: ReleaseEscrowType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			packet, err := ParsePacketData([]byte(tt.packetJSON))
			if err != nil {
				t.Errorf("ParsePacketData() error = %v", err)
				return
			}
			if packet == nil {
				t.Error("ParsePacketData() returned nil packet")
			}
		})
	}
}
