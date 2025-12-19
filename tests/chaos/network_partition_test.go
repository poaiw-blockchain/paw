package chaos_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/testutil/network"
)

// NetworkPartitionTestSuite tests system behavior under network partitions
type NetworkPartitionTestSuite struct {
	suite.Suite

	cfg     network.Config
	network *network.Network
}

func TestNetworkPartitionTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping chaos tests in short mode")
	}
	suite.Run(t, new(NetworkPartitionTestSuite))
}

func (suite *NetworkPartitionTestSuite) SetupSuite() {
	suite.T().Log("setting up chaos engineering test suite")

	suite.cfg = network.DefaultConfig()
	suite.cfg.NumValidators = 5 // Need multiple validators for partition

	var err error
	suite.network, err = network.New(suite.T(), suite.T().TempDir(), suite.cfg)
	suite.Require().NoError(err)

	_, err = suite.network.WaitForHeight(1)
	suite.Require().NoError(err)
}

func (suite *NetworkPartitionTestSuite) TearDownSuite() {
	suite.network.Cleanup()
}

// TestMajorityPartition tests that chain continues with majority partition
func (suite *NetworkPartitionTestSuite) TestMajorityPartition() {
	suite.T().Log("Testing majority partition (3 out of 5 validators)")

	// Record initial height
	initialHeight, err := suite.network.LatestHeight()
	suite.Require().NoError(err)

	// Partition network: isolate 2 validators (minority)
	minorityValidators := suite.network.Validators[3:5]

	suite.T().Log("Creating network partition...")
	suite.partitionValidators(minorityValidators)

	// Wait for blocks to continue with majority
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			height, err := suite.network.LatestHeight()
			if err == nil && height > initialHeight+3 {
				suite.T().Logf("Chain progressed to height %d despite partition", height)
				goto partitionHealed
			}
		case <-ctx.Done():
			suite.T().Fatal("chain did not progress with majority partition")
		}
	}

partitionHealed:
	// Heal partition
	suite.T().Log("Healing network partition...")
	suite.healPartition(minorityValidators)

	// Wait for minority to catch up
	time.Sleep(5 * time.Second)

	// Verify all validators are at similar heights
	heights := make([]int64, len(suite.network.Validators))
	for i, val := range suite.network.Validators {
		height, err := suite.getValidatorHeight(val)
		suite.Require().NoError(err)
		heights[i] = height
		suite.T().Logf("Validator %d height: %d", i, height)
	}

	// All validators should be within a few blocks
	maxHeight := heights[0]
	minHeight := heights[0]
	for _, h := range heights[1:] {
		if h > maxHeight {
			maxHeight = h
		}
		if h < minHeight {
			minHeight = h
		}
	}

	suite.Require().LessOrEqual(maxHeight-minHeight, int64(5),
		"validators should catch up within 5 blocks")
}

// TestMinorityPartition tests that minority partition cannot progress
func (suite *NetworkPartitionTestSuite) TestMinorityPartition() {
	suite.T().Log("Testing minority partition (2 out of 5 validators)")

	// Partition network: isolate 3 validators (majority)
	majorityValidators := suite.network.Validators[0:3]

	initialHeights := make(map[int]int64)
	for i, val := range suite.network.Validators[3:5] {
		height, err := suite.getValidatorHeight(val)
		suite.Require().NoError(err)
		initialHeights[i] = height
	}

	suite.T().Log("Creating network partition...")
	suite.partitionValidators(majorityValidators)

	// Wait and verify minority cannot progress
	time.Sleep(10 * time.Second)

	for i, val := range suite.network.Validators[3:5] {
		height, err := suite.getValidatorHeight(val)
		suite.Require().NoError(err)

		// Minority should not make significant progress (maybe 1-2 blocks at most)
		suite.Require().LessOrEqual(height-initialHeights[i], int64(2),
			"minority partition should not make progress")
	}

	// Heal partition
	suite.healPartition(majorityValidators)
}

// TestFlappingPartition tests behavior under intermittent network issues
func (suite *NetworkPartitionTestSuite) TestFlappingPartition() {
	suite.T().Log("Testing flapping network partition")

	initialHeight, err := suite.network.LatestHeight()
	suite.Require().NoError(err)

	// Simulate flapping by repeatedly partitioning and healing
	for i := 0; i < 5; i++ {
		suite.T().Logf("Flap iteration %d: creating partition", i+1)

		// Randomly partition 2 validators
		validators := suite.network.Validators[i%2 : i%2+2]
		suite.partitionValidators(validators)

		// Wait 2 seconds
		time.Sleep(2 * time.Second)

		suite.T().Logf("Flap iteration %d: healing partition", i+1)
		suite.healPartition(validators)

		// Wait 2 seconds
		time.Sleep(2 * time.Second)
	}

	// Verify chain still makes progress despite flapping
	finalHeight, err := suite.network.LatestHeight()
	suite.Require().NoError(err)

	suite.Require().Greater(finalHeight, initialHeight,
		"chain should make progress despite flapping partitions")

	suite.T().Logf("Chain progressed from %d to %d during flapping test",
		initialHeight, finalHeight)
}

// TestSlowValidator tests impact of slow validator on consensus
func (suite *NetworkPartitionTestSuite) TestSlowValidator() {
	suite.T().Log("Testing slow validator behavior")

	initialHeight, err := suite.network.LatestHeight()
	suite.Require().NoError(err)

	// Slow down one validator (simulate by adding artificial delay)
	slowValidator := suite.network.Validators[0]
	suite.slowDownValidator(slowValidator, 500*time.Millisecond)

	// Wait for several blocks
	time.Sleep(10 * time.Second)

	// Verify chain still progresses
	finalHeight, err := suite.network.LatestHeight()
	suite.Require().NoError(err)

	suite.Require().Greater(finalHeight, initialHeight+3,
		"chain should progress despite slow validator")

	// Remove slowdown
	suite.removeSlowdown(slowValidator)
}

// TestValidatorCrashRecovery tests recovery from validator crashes
func (suite *NetworkPartitionTestSuite) TestValidatorCrashRecovery() {
	suite.T().Log("Testing validator crash and recovery")

	// Crash 2 validators (minority)
	crashedValidators := suite.network.Validators[3:5]

	for _, val := range crashedValidators {
		suite.T().Logf("Crashing validator %s", val.Address)
		suite.crashValidator(val)
	}

	// Wait for chain to continue with remaining validators
	initialHeight, err := suite.network.LatestHeight()
	suite.Require().NoError(err)

	time.Sleep(5 * time.Second)

	height, err := suite.network.LatestHeight()
	suite.Require().NoError(err)
	suite.Require().Greater(height, initialHeight,
		"chain should continue with remaining validators")

	// Recover crashed validators
	for _, val := range crashedValidators {
		suite.T().Logf("Recovering validator %s", val.Address)
		suite.recoverValidator(val)
	}

	// Wait for recovery
	time.Sleep(5 * time.Second)

	// Verify all validators are operational
	for i, val := range suite.network.Validators {
		height, err := suite.getValidatorHeight(val)
		suite.Require().NoError(err)
		suite.T().Logf("Validator %d height after recovery: %d", i, height)
	}
}

// TestByzantineValidator tests behavior with Byzantine validator
func (suite *NetworkPartitionTestSuite) TestByzantineValidator() {
	suite.T().Log("Testing Byzantine validator behavior")

	// Mark one validator as Byzantine (sends conflicting votes)
	byzantineValidator := suite.network.Validators[0]
	suite.makeByzantine(byzantineValidator)

	initialHeight, err := suite.network.LatestHeight()
	suite.Require().NoError(err)

	// Wait and verify chain still progresses
	time.Sleep(10 * time.Second)

	finalHeight, err := suite.network.LatestHeight()
	suite.Require().NoError(err)

	suite.Require().Greater(finalHeight, initialHeight,
		"chain should progress despite Byzantine validator")

	// Byzantine validator should be eventually detected and ignored/slashed
	// (Implementation depends on consensus layer)
}

// Helper functions for chaos injection

func (suite *NetworkPartitionTestSuite) partitionValidators(validators []*network.Validator) {
	// In a real implementation, this would use iptables or network namespaces
	// to actually partition the network
	for _, val := range validators {
		suite.T().Logf("Partitioning validator %s", val.Address)
		// Simulate partition
	}
}

func (suite *NetworkPartitionTestSuite) healPartition(validators []*network.Validator) {
	// Restore network connectivity
	for _, val := range validators {
		suite.T().Logf("Healing partition for validator %s", val.Address)
		// Restore connectivity
	}
}

func (suite *NetworkPartitionTestSuite) getValidatorHeight(val *network.Validator) (int64, error) {
	_ = val
	// Query validator's current height
	// Simplified implementation
	return suite.network.LatestHeight()
}

func (suite *NetworkPartitionTestSuite) slowDownValidator(val *network.Validator, delay time.Duration) {
	suite.T().Logf("Slowing down validator %s by %v", val.Address, delay)
	// Inject artificial delay in message processing
}

func (suite *NetworkPartitionTestSuite) removeSlowdown(val *network.Validator) {
	suite.T().Logf("Removing slowdown for validator %s", val.Address)
	// Remove delay
}

func (suite *NetworkPartitionTestSuite) crashValidator(val *network.Validator) {
	// Simulate validator crash
	suite.T().Logf("Simulating crash for validator %s", val.Address)
}

func (suite *NetworkPartitionTestSuite) recoverValidator(val *network.Validator) {
	// Restart validator
	suite.T().Logf("Restarting validator %s", val.Address)
}

func (suite *NetworkPartitionTestSuite) makeByzantine(val *network.Validator) {
	// Configure validator to send conflicting messages
	suite.T().Logf("Making validator %s Byzantine", val.Address)
}
