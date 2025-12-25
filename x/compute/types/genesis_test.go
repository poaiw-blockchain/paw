package types

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
)

func TestDefaultGenesis(t *testing.T) {
	genesis := DefaultGenesis()

	if genesis == nil {
		t.Fatal("DefaultGenesis() returned nil")
	}

	if genesis.Params.MinProviderStake.IsNil() {
		t.Error("DefaultGenesis().Params should be initialized")
	}

	if genesis.GovernanceParams.DisputeDeposit.IsNil() {
		t.Error("DefaultGenesis().GovernanceParams should be initialized")
	}

	if genesis.Providers == nil {
		t.Error("DefaultGenesis().Providers should be initialized")
	}

	if genesis.Requests == nil {
		t.Error("DefaultGenesis().Requests should be initialized")
	}

	if genesis.Results == nil {
		t.Error("DefaultGenesis().Results should be initialized")
	}

	if genesis.Disputes == nil {
		t.Error("DefaultGenesis().Disputes should be initialized")
	}

	if genesis.SlashRecords == nil {
		t.Error("DefaultGenesis().SlashRecords should be initialized")
	}

	if genesis.Appeals == nil {
		t.Error("DefaultGenesis().Appeals should be initialized")
	}

	if genesis.EscrowStates == nil {
		t.Error("DefaultGenesis().EscrowStates should be initialized")
	}

	if genesis.NextRequestId != 1 {
		t.Errorf("DefaultGenesis().NextRequestId = %v, want 1", genesis.NextRequestId)
	}

	if genesis.NextDisputeId != 1 {
		t.Errorf("DefaultGenesis().NextDisputeId = %v, want 1", genesis.NextDisputeId)
	}

	if genesis.NextSlashId != 1 {
		t.Errorf("DefaultGenesis().NextSlashId = %v, want 1", genesis.NextSlashId)
	}

	if genesis.NextAppealId != 1 {
		t.Errorf("DefaultGenesis().NextAppealId = %v, want 1", genesis.NextAppealId)
	}

	if genesis.NextEscrowNonce != 1 {
		t.Errorf("DefaultGenesis().NextEscrowNonce = %v, want 1", genesis.NextEscrowNonce)
	}
}

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		name    string
		genesis GenesisState
		wantErr bool
		errMsg  string
	}{
		{
			name:    "default genesis is valid",
			genesis: *DefaultGenesis(),
			wantErr: false,
		},
		{
			name: "zero min provider stake",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(0),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       1,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "min provider stake must be positive",
		},
		{
			name: "negative min provider stake",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(-1000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       1,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "min provider stake must be positive",
		},
		{
			name: "zero verification timeout",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 0,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       1,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "timeouts must be non-zero",
		},
		{
			name: "zero max request timeout",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   0,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       1,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "timeouts must be non-zero",
		},
		{
			name: "zero reputation slash percentage",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  0,
					StakeSlashPercentage:       1,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "slash percentages must be positive",
		},
		{
			name: "zero stake slash percentage",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       0,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "slash percentages must be positive",
		},
		{
			name: "zero min reputation score",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       1,
					MinReputationScore:         0,
					EscrowReleaseDelaySeconds:  3600,
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "min reputation score must be positive",
		},
		{
			name: "zero escrow release delay",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       1,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  0,
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "escrow release delay must be non-zero",
		},
		{
			name: "empty authorized channel port_id",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       1,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
					AuthorizedChannels: []AuthorizedChannel{
						{PortId: "", ChannelId: "channel-0"},
					},
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "authorized channel port_id cannot be empty",
		},
		{
			name: "empty authorized channel channel_id",
			genesis: GenesisState{
				Params: Params{
					MinProviderStake:           math.NewInt(1000000),
					VerificationTimeoutSeconds: 300,
					MaxRequestTimeoutSeconds:   3600,
					ReputationSlashPercentage:  10,
					StakeSlashPercentage:       1,
					MinReputationScore:         50,
					EscrowReleaseDelaySeconds:  3600,
					AuthorizedChannels: []AuthorizedChannel{
						{PortId: "compute", ChannelId: ""},
					},
				},
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "authorized channel channel_id cannot be empty",
		},
		{
			name: "zero dispute deposit",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(0),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "dispute deposit must be positive",
		},
		{
			name: "zero evidence period",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   0,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "evidence and voting periods must be non-zero",
		},
		{
			name: "zero voting period",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     0,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "evidence and voting periods must be non-zero",
		},
		{
			name: "quorum percentage zero",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyZeroDec(),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "quorum percentage must be (0,1]",
		},
		{
			name: "quorum percentage greater than 1",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("1.1"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "quorum percentage must be (0,1]",
		},
		{
			name: "consensus threshold zero",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyZeroDec(),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "consensus threshold must be (0,1]",
		},
		{
			name: "consensus threshold greater than 1",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("1.1"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "consensus threshold must be (0,1]",
		},
		{
			name: "slash percentage negative",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("-0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "slash percentage must be between 0 and 1",
		},
		{
			name: "slash percentage greater than 1",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("1.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "slash percentage must be between 0 and 1",
		},
		{
			name: "appeal deposit percentage negative",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("-0.05"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "appeal deposit percentage must be between 0 and 1",
		},
		{
			name: "appeal deposit percentage greater than 1",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("1.1"),
					MaxEvidenceSize:         10 * 1024 * 1024,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "appeal deposit percentage must be between 0 and 1",
		},
		{
			name: "zero max evidence size",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         0,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "max evidence size must be positive",
		},
		{
			name: "negative max evidence size",
			genesis: GenesisState{
				Params: DefaultParams(),
				GovernanceParams: GovernanceParams{
					DisputeDeposit:          math.NewInt(1_000_000),
					EvidencePeriodSeconds:   86400,
					VotingPeriodSeconds:     86400,
					QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
					ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
					SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
					AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
					MaxEvidenceSize:         -1,
				},
				NextRequestId: 1,
				NextDisputeId: 1,
				NextSlashId:   1,
				NextAppealId:  1,
			},
			wantErr: true,
			errMsg:  "max evidence size must be positive",
		},
		{
			name: "zero next request id",
			genesis: GenesisState{
				Params:           DefaultParams(),
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    0,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "next ids must be positive",
		},
		{
			name: "zero next dispute id",
			genesis: GenesisState{
				Params:           DefaultParams(),
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    0,
				NextSlashId:      1,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "next ids must be positive",
		},
		{
			name: "zero next slash id",
			genesis: GenesisState{
				Params:           DefaultParams(),
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      0,
				NextAppealId:     1,
			},
			wantErr: true,
			errMsg:  "next ids must be positive",
		},
		{
			name: "zero next appeal id",
			genesis: GenesisState{
				Params:           DefaultParams(),
				GovernanceParams: DefaultGovernanceParams(),
				NextRequestId:    1,
				NextDisputeId:    1,
				NextSlashId:      1,
				NextAppealId:     0,
			},
			wantErr: true,
			errMsg:  "next ids must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.genesis.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenesisState.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("GenesisState.Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
			}
		})
	}
}
