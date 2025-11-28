package invariants

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/app"
	computetypes "github.com/paw-chain/paw/x/compute/types"
)

// ComputeInvariantTestSuite tests compute module invariants
// Critical for escrow balance conservation and request status consistency
type ComputeInvariantTestSuite struct {
	suite.Suite
	app *app.PAWApp
	ctx sdk.Context
}

// SetupTest initializes the test environment before each test
func (suite *ComputeInvariantTestSuite) SetupTest() {
	suite.app = app.Setup(suite.T(), false)
	suite.ctx = suite.app.BaseApp.NewContext(false)
}

// TestEscrowBalanceConservation verifies locked escrow equals sum of active escrows
// This is the most critical invariant - escrowed funds must be properly tracked
func (suite *ComputeInvariantTestSuite) TestEscrowBalanceConservation() {
	// Get all active requests (assigned, processing, completed but not finalized)
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	// Sum all escrowed amounts from active requests
	totalEscrowed := math.ZeroInt()
	for _, request := range allRequests {
		if request.Status != computetypes.RequestStatus_REQUEST_STATUS_FINALIZED &&
			request.Status != computetypes.RequestStatus_REQUEST_STATUS_CANCELLED &&
			request.Status != computetypes.RequestStatus_REQUEST_STATUS_EXPIRED {
			totalEscrowed = totalEscrowed.Add(request.EscrowedAmount)
		}
	}

	// Get module account balance
	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(computetypes.ModuleName)
	moduleBalance := suite.app.BankKeeper.GetBalance(suite.ctx, moduleAddr, "upaw")

	// Module balance should equal or exceed total escrowed
	// (may exceed due to fees or other module operations)
	suite.Require().True(
		moduleBalance.Amount.GTE(totalEscrowed),
		"Module balance less than escrowed: module=%s, escrowed=%s",
		moduleBalance.Amount.String(),
		totalEscrowed.String(),
	)
}

// TestEscrowStatusConsistency ensures escrow amounts match request status
func (suite *ComputeInvariantTestSuite) TestEscrowStatusConsistency() {
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	for _, request := range allRequests {
		switch request.Status {
		case computetypes.RequestStatus_REQUEST_STATUS_FINALIZED,
			computetypes.RequestStatus_REQUEST_STATUS_CANCELLED,
			computetypes.RequestStatus_REQUEST_STATUS_EXPIRED:
			// Finalized/cancelled/expired requests should have no escrowed funds
			suite.Require().True(
				request.EscrowedAmount.IsZero(),
				"Request %d has status %s but non-zero escrow: %s",
				request.Id,
				request.Status.String(),
				request.EscrowedAmount.String(),
			)

		case computetypes.RequestStatus_REQUEST_STATUS_ASSIGNED,
			computetypes.RequestStatus_REQUEST_STATUS_PROCESSING:
			// Active requests should have positive escrow
			suite.Require().True(
				request.EscrowedAmount.GT(math.ZeroInt()),
				"Request %d has active status %s but zero escrow",
				request.Id,
				request.Status.String(),
			)

			// Escrowed amount should not exceed max payment
			suite.Require().True(
				request.EscrowedAmount.LTE(request.MaxPayment),
				"Request %d escrow %s exceeds max payment %s",
				request.Id,
				request.EscrowedAmount.String(),
				request.MaxPayment.String(),
			)

		case computetypes.RequestStatus_REQUEST_STATUS_COMPLETED:
			// Completed requests may have escrow if not yet finalized
			suite.Require().True(
				request.EscrowedAmount.GTE(math.ZeroInt()),
				"Request %d has negative escrow: %s",
				request.Id,
				request.EscrowedAmount.String(),
			)
		}
	}
}

// TestRequestNonceUniqueness ensures all request IDs are unique
func (suite *ComputeInvariantTestSuite) TestRequestNonceUniqueness() {
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	seenIds := make(map[uint64]bool)

	for _, request := range allRequests {
		suite.Require().False(
			seenIds[request.Id],
			"Duplicate request ID found: %d",
			request.Id,
		)
		seenIds[request.Id] = true
	}
}

// TestMaxPaymentPositive ensures all requests have positive max payment
func (suite *ComputeInvariantTestSuite) TestMaxPaymentPositive() {
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	for _, request := range allRequests {
		suite.Require().True(
			request.MaxPayment.GT(math.ZeroInt()),
			"Request %d has non-positive max payment: %s",
			request.Id,
			request.MaxPayment.String(),
		)
	}
}

// TestProviderQuotaLimits ensures providers don't exceed quota limits
func (suite *ComputeInvariantTestSuite) TestProviderQuotaLimits() {
	// Get all providers
	allProviders := suite.app.ComputeKeeper.GetAllProviders(suite.ctx)

	for _, provider := range allProviders {
		providerAddr, err := sdk.AccAddressFromBech32(provider.Address)
		suite.NoError(err)

		// Count active requests for this provider
		activeRequests := suite.app.ComputeKeeper.GetProviderActiveRequests(suite.ctx, providerAddr)

		// If provider has a quota, active requests should not exceed it
		if provider.MaxConcurrentRequests > 0 {
			suite.Require().True(
				uint64(len(activeRequests)) <= provider.MaxConcurrentRequests,
				"Provider %s exceeds quota: active=%d, max=%d",
				provider.Address,
				len(activeRequests),
				provider.MaxConcurrentRequests,
			)
		}
	}
}

// TestComputeSpecsValid ensures all requests have valid compute specs
func (suite *ComputeInvariantTestSuite) TestComputeSpecsValid() {
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	for _, request := range allRequests {
		specs := request.Specs

		// CPU should be positive
		suite.Require().True(
			specs.Cpu > 0,
			"Request %d has non-positive CPU: %d",
			request.Id,
			specs.Cpu,
		)

		// Memory should be positive
		suite.Require().True(
			specs.Memory > 0,
			"Request %d has non-positive memory: %d",
			request.Id,
			specs.Memory,
		)

		// Disk should be non-negative
		suite.Require().True(
			specs.Disk >= 0,
			"Request %d has negative disk: %d",
			request.Id,
			specs.Disk,
		)

		// Timeout should be positive
		suite.Require().True(
			specs.Timeout > 0,
			"Request %d has non-positive timeout: %d",
			request.Id,
			specs.Timeout,
		)
	}
}

// TestRequestTimestampsConsistent ensures timestamps are logical
func (suite *ComputeInvariantTestSuite) TestRequestTimestampsConsistent() {
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	for _, request := range allRequests {
		// CreatedAt should be set
		suite.Require().False(
			request.CreatedAt.IsZero(),
			"Request %d has zero CreatedAt timestamp",
			request.Id,
		)

		// If assigned, AssignedAt should be after CreatedAt
		if request.AssignedAt != nil {
			suite.Require().True(
				request.AssignedAt.After(request.CreatedAt) || request.AssignedAt.Equal(request.CreatedAt),
				"Request %d AssignedAt %s before CreatedAt %s",
				request.Id,
				request.AssignedAt.String(),
				request.CreatedAt.String(),
			)
		}

		// If completed, CompletedAt should be after CreatedAt
		if request.CompletedAt != nil {
			suite.Require().True(
				request.CompletedAt.After(request.CreatedAt),
				"Request %d CompletedAt %s before or equal to CreatedAt %s",
				request.Id,
				request.CompletedAt.String(),
				request.CreatedAt.String(),
			)

			// CompletedAt should be after AssignedAt
			if request.AssignedAt != nil {
				suite.Require().True(
					request.CompletedAt.After(*request.AssignedAt) || request.CompletedAt.Equal(*request.AssignedAt),
					"Request %d CompletedAt %s before AssignedAt %s",
					request.Id,
					request.CompletedAt.String(),
					request.AssignedAt.String(),
				)
			}
		}
	}
}

// TestProviderAddressesValid ensures all provider addresses are valid
func (suite *ComputeInvariantTestSuite) TestProviderAddressesValid() {
	allProviders := suite.app.ComputeKeeper.GetAllProviders(suite.ctx)

	for _, provider := range allProviders {
		_, err := sdk.AccAddressFromBech32(provider.Address)
		suite.Require().NoError(
			err,
			"Provider has invalid address: %s",
			provider.Address,
		)
	}
}

// TestProviderStakeConsistency ensures provider stakes are properly tracked
func (suite *ComputeInvariantTestSuite) TestProviderStakeConsistency() {
	allProviders := suite.app.ComputeKeeper.GetAllProviders(suite.ctx)

	for _, provider := range allProviders {
		// Stake should be non-negative
		suite.Require().False(
			provider.Stake.IsNegative(),
			"Provider %s has negative stake: %s",
			provider.Address,
			provider.Stake.String(),
		)

		// If provider is active, stake should meet minimum requirement
		params := suite.app.ComputeKeeper.GetParams(suite.ctx)
		if provider.Status == computetypes.ProviderStatus_PROVIDER_STATUS_ACTIVE {
			suite.Require().True(
				provider.Stake.GTE(params.MinProviderStake),
				"Active provider %s has stake %s below minimum %s",
				provider.Address,
				provider.Stake.String(),
				params.MinProviderStake.String(),
			)
		}
	}
}

// TestResultHashesPresent ensures completed requests have result hashes
func (suite *ComputeInvariantTestSuite) TestResultHashesPresent() {
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	for _, request := range allRequests {
		if request.Status == computetypes.RequestStatus_REQUEST_STATUS_COMPLETED ||
			request.Status == computetypes.RequestStatus_REQUEST_STATUS_FINALIZED {
			// Completed/finalized requests should have result hash
			suite.Require().NotEmpty(
				request.ResultHash,
				"Request %d is %s but has no result hash",
				request.Id,
				request.Status.String(),
			)
		}
	}
}

// TestProviderReputationBounds ensures reputation scores are within valid range
func (suite *ComputeInvariantTestSuite) TestProviderReputationBounds() {
	allProviders := suite.app.ComputeKeeper.GetAllProviders(suite.ctx)

	for _, provider := range allProviders {
		// Reputation should be between 0 and 100
		suite.Require().True(
			provider.Reputation >= 0 && provider.Reputation <= 100,
			"Provider %s has reputation %d outside valid range [0,100]",
			provider.Address,
			provider.Reputation,
		)
	}
}

// TestRequestRequesterValid ensures all requesters have valid addresses
func (suite *ComputeInvariantTestSuite) TestRequestRequesterValid() {
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	for _, request := range allRequests {
		_, err := sdk.AccAddressFromBech32(request.Requester)
		suite.Require().NoError(
			err,
			"Request %d has invalid requester address: %s",
			request.Id,
			request.Requester,
		)

		_, err = sdk.AccAddressFromBech32(request.Provider)
		suite.Require().NoError(
			err,
			"Request %d has invalid provider address: %s",
			request.Id,
			request.Provider,
		)
	}
}

// TestModuleAccountIsModuleAccount ensures compute module account is properly configured
func (suite *ComputeInvariantTestSuite) TestModuleAccountIsModuleAccount() {
	moduleAddr := suite.app.AccountKeeper.GetModuleAddress(computetypes.ModuleName)
	suite.Require().NotNil(moduleAddr, "Compute module address not found")

	moduleAcc := suite.app.AccountKeeper.GetAccount(suite.ctx, moduleAddr)
	suite.Require().NotNil(moduleAcc, "Compute module account not found")

	// Verify it's a module account
	_, ok := moduleAcc.(sdk.ModuleAccountI)
	suite.Require().True(ok, "Compute account is not a module account")
}

// TestEscrowNonNegative ensures all escrowed amounts are non-negative
func (suite *ComputeInvariantTestSuite) TestEscrowNonNegative() {
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	for _, request := range allRequests {
		suite.Require().False(
			request.EscrowedAmount.IsNegative(),
			"Request %d has negative escrowed amount: %s",
			request.Id,
			request.EscrowedAmount.String(),
		)
	}
}

// TestProviderCapacityConsistency ensures provider capacity values are reasonable
func (suite *ComputeInvariantTestSuite) TestProviderCapacityConsistency() {
	allProviders := suite.app.ComputeKeeper.GetAllProviders(suite.ctx)

	for _, provider := range allProviders {
		// CPU capacity should be positive if provider is active
		if provider.Status == computetypes.ProviderStatus_PROVIDER_STATUS_ACTIVE {
			suite.Require().True(
				provider.Cpu > 0,
				"Active provider %s has non-positive CPU: %d",
				provider.Address,
				provider.Cpu,
			)

			suite.Require().True(
				provider.Memory > 0,
				"Active provider %s has non-positive memory: %d",
				provider.Address,
				provider.Memory,
			)

			suite.Require().True(
				provider.Disk >= 0,
				"Active provider %s has negative disk: %d",
				provider.Address,
				provider.Disk,
			)
		}
	}
}

// TestContainerImagePresent ensures all requests have container images
func (suite *ComputeInvariantTestSuite) TestContainerImagePresent() {
	allRequests := suite.app.ComputeKeeper.GetAllRequests(suite.ctx)

	for _, request := range allRequests {
		suite.Require().NotEmpty(
			request.ContainerImage,
			"Request %d has no container image",
			request.Id,
		)
	}
}

func TestComputeInvariantTestSuite(t *testing.T) {
	suite.Run(t, new(ComputeInvariantTestSuite))
}
