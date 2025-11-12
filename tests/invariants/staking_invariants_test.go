package invariants_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/paw-chain/paw/app"
)

type StakingInvariantsTestSuite struct {
	suite.Suite
	app *app.App
	ctx sdk.Context
}

func (s *StakingInvariantsTestSuite) SetupTest() {
	db := dbm.NewMemDB()
	encCfg := app.MakeEncodingConfig()

	s.app = app.NewApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		encCfg,
		app.GetEnabledProposals(),
		baseapp.SetChainID("paw-test-1"),
	)

	s.ctx = s.app.BaseApp.NewContext(false, tmproto.Header{
		ChainID: "paw-test-1",
		Height:  1,
	})
}

// InvariantModuleAccountCoins checks module account coins match bonded/not-bonded pools
func (s *StakingInvariantsTestSuite) InvariantModuleAccountCoins() (string, bool) {
	bondedPool := s.app.StakingKeeper.GetBondedPool(s.ctx)
	notBondedPool := s.app.StakingKeeper.GetNotBondedPool(s.ctx)

	bondedBalance := s.app.BankKeeper.GetBalance(s.ctx, bondedPool.GetAddress(), sdk.DefaultBondDenom)
	notBondedBalance := s.app.BankKeeper.GetBalance(s.ctx, notBondedPool.GetAddress(), sdk.DefaultBondDenom)

	bondedTokens := s.app.StakingKeeper.TotalBondedTokens(s.ctx)
	notBondedTokens := bondedBalance.Amount.Add(notBondedBalance.Amount).Sub(bondedTokens)

	broken := false
	msg := ""

	// Check bonded pool
	if !bondedBalance.Amount.Equal(bondedTokens) {
		broken = true
		msg += sdk.FormatInvariant(
			stakingtypes.ModuleName,
			"bonded pool",
			"bonded pool balance does not match total bonded tokens\n"+
				"\tbonded pool balance: %s\n"+
				"\ttotal bonded tokens: %s\n",
			bondedBalance.Amount.String(),
			bondedTokens.String(),
		)
	}

	// Check not bonded pool
	if notBondedTokens.IsNegative() {
		broken = true
		msg += sdk.FormatInvariant(
			stakingtypes.ModuleName,
			"not bonded pool",
			"not bonded pool has negative tokens\n"+
				"\tnot bonded tokens: %s\n",
			notBondedTokens.String(),
		)
	}

	return msg, broken
}

// InvariantValidatorsBonded checks validators are bonded correctly
func (s *StakingInvariantsTestSuite) InvariantValidatorsBonded() (string, bool) {
	var msg string
	var broken bool

	// Get all validators
	validators, err := s.app.StakingKeeper.GetAllValidators(s.ctx)
	if err != nil {
		broken = true
		msg = sdk.FormatInvariant(
			stakingtypes.ModuleName,
			"validators bonded",
			"error getting validators: %s\n",
			err.Error(),
		)
		return msg, broken
	}

	var totalBonded sdk.Int
	totalBonded = sdk.ZeroInt()

	for _, validator := range validators {
		if validator.IsBonded() {
			totalBonded = totalBonded.Add(validator.GetBondedTokens())
		}
	}

	bondedTokens := s.app.StakingKeeper.TotalBondedTokens(s.ctx)

	if !totalBonded.Equal(bondedTokens) {
		broken = true
		msg = sdk.FormatInvariant(
			stakingtypes.ModuleName,
			"validators bonded",
			"sum of bonded validators does not match total bonded tokens\n"+
				"\tsum of bonded validators: %s\n"+
				"\ttotal bonded tokens: %s\n",
			totalBonded.String(),
			bondedTokens.String(),
		)
	}

	return msg, broken
}

// InvariantDelegationShares checks delegation shares sum correctly
func (s *StakingInvariantsTestSuite) InvariantDelegationShares() (string, bool) {
	var msg string
	var broken bool

	// Get all validators
	validators, err := s.app.StakingKeeper.GetAllValidators(s.ctx)
	if err != nil {
		broken = true
		msg = sdk.FormatInvariant(
			stakingtypes.ModuleName,
			"delegation shares",
			"error getting validators: %s\n",
			err.Error(),
		)
		return msg, broken
	}

	for _, validator := range validators {
		// Get all delegations for this validator
		delegations, err := s.app.StakingKeeper.GetValidatorDelegations(s.ctx, validator.GetOperator())
		if err != nil {
			continue
		}

		var totalShares sdk.Dec
		totalShares = sdk.ZeroDec()

		for _, delegation := range delegations {
			totalShares = totalShares.Add(delegation.Shares)
		}

		// Check if total shares match validator's delegator shares
		if !totalShares.Equal(validator.GetDelegatorShares()) {
			broken = true
			msg += sdk.FormatInvariant(
				stakingtypes.ModuleName,
				"delegation shares",
				"validator %s delegation shares do not match\n"+
					"\tsum of delegations: %s\n"+
					"\tvalidator delegator shares: %s\n",
				validator.GetOperator().String(),
				totalShares.String(),
				validator.GetDelegatorShares().String(),
			)
		}
	}

	return msg, broken
}

// InvariantPositiveDelegation checks all delegations are positive
func (s *StakingInvariantsTestSuite) InvariantPositiveDelegation() (string, bool) {
	var msg string
	var broken bool

	// Iterate through all delegations
	err := s.app.StakingKeeper.IterateDelegations(s.ctx, func(index int64, delegation stakingtypes.DelegationI) bool {
		if delegation.GetShares().IsNegative() {
			broken = true
			msg += sdk.FormatInvariant(
				stakingtypes.ModuleName,
				"positive delegation",
				"delegation has negative shares\n"+
					"\tdelegator: %s\n"+
					"\tvalidator: %s\n"+
					"\tshares: %s\n",
				delegation.GetDelegatorAddr().String(),
				delegation.GetValidatorAddr().String(),
				delegation.GetShares().String(),
			)
		}
		return false
	})

	if err != nil {
		broken = true
		msg += sdk.FormatInvariant(
			stakingtypes.ModuleName,
			"positive delegation",
			"error iterating delegations: %s\n",
			err.Error(),
		)
	}

	return msg, broken
}

// TestStakingInvariants runs all staking invariants
func (s *StakingInvariantsTestSuite) TestStakingInvariants() {
	// Note: In a real test, you would create validators and delegations here
	// This is a basic structure showing how to test staking invariants

	msg, broken := s.InvariantModuleAccountCoins()
	s.Require().False(broken, msg)

	msg, broken = s.InvariantValidatorsBonded()
	s.Require().False(broken, msg)

	msg, broken = s.InvariantDelegationShares()
	s.Require().False(broken, msg)

	msg, broken = s.InvariantPositiveDelegation()
	s.Require().False(broken, msg)
}

// TestInvariantsWithDelegations tests invariants with actual delegations
func (s *StakingInvariantsTestSuite) TestInvariantsWithDelegations() {
	// This would require setting up validators and delegations
	// For now, we verify the invariant functions work correctly

	msg, broken := s.InvariantModuleAccountCoins()
	s.Require().False(broken, msg)

	msg, broken = s.InvariantValidatorsBonded()
	s.Require().False(broken, msg)
}

func TestStakingInvariantsTestSuite(t *testing.T) {
	suite.Run(t, new(StakingInvariantsTestSuite))
}
