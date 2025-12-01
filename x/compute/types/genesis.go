package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
)

// DefaultGenesis returns the default genesis state for the compute module.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		GovernanceParams: DefaultGovernanceParams(),
		Providers:        []Provider{},
		Requests:         []Request{},
		Results:          []Result{},
		Disputes:         []Dispute{},
		SlashRecords:     []SlashRecord{},
		Appeals:          []Appeal{},
		NextRequestId:    1,
		NextDisputeId:    1,
		NextSlashId:      1,
		NextAppealId:     1,
	}
}

// Validate ensures the genesis state is well-formed.
func (gs GenesisState) Validate() error {
	p := gs.Params
	if !p.MinProviderStake.IsPositive() {
		return fmt.Errorf("min provider stake must be positive")
	}
	if p.VerificationTimeoutSeconds == 0 || p.MaxRequestTimeoutSeconds == 0 {
		return fmt.Errorf("timeouts must be non-zero")
	}
	if p.ReputationSlashPercentage < 0 || p.StakeSlashPercentage < 0 {
		return fmt.Errorf("slash percentages must be non-negative")
	}
	if p.MinReputationScore < 0 {
		return fmt.Errorf("min reputation score must be non-negative")
	}
	if p.EscrowReleaseDelaySeconds == 0 {
		return fmt.Errorf("escrow release delay must be non-zero")
	}

	gov := gs.GovernanceParams
	if !gov.DisputeDeposit.IsPositive() {
		return fmt.Errorf("dispute deposit must be positive")
	}
	if gov.EvidencePeriodSeconds == 0 || gov.VotingPeriodSeconds == 0 {
		return fmt.Errorf("evidence and voting periods must be non-zero")
	}
	if gov.QuorumPercentage.LTE(sdkmath.LegacyZeroDec()) || gov.QuorumPercentage.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("quorum percentage must be (0,1]")
	}
	if gov.ConsensusThreshold.LTE(sdkmath.LegacyZeroDec()) || gov.ConsensusThreshold.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("consensus threshold must be (0,1]")
	}
	if gov.SlashPercentage.LT(sdkmath.LegacyZeroDec()) || gov.SlashPercentage.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("slash percentage must be between 0 and 1")
	}
	if gov.AppealDepositPercentage.LT(sdkmath.LegacyZeroDec()) || gov.AppealDepositPercentage.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("appeal deposit percentage must be between 0 and 1")
	}
	if gov.MaxEvidenceSize <= 0 {
		return fmt.Errorf("max evidence size must be positive")
	}

	if gs.NextRequestId == 0 || gs.NextDisputeId == 0 || gs.NextSlashId == 0 || gs.NextAppealId == 0 {
		return fmt.Errorf("next ids must be positive")
	}

	return nil
}
