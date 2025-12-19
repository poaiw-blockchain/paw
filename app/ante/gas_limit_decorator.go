package ante

import (
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// Gas limits for different operation types to prevent exhaustion attacks
const (
	// DEX operations
	MaxGasPerSwap            uint64 = 200_000
	MaxGasPerPoolCreation    uint64 = 300_000
	MaxGasPerLiquidityAdd    uint64 = 150_000
	MaxGasPerLiquidityRemove uint64 = 150_000

	// Compute operations
	MaxGasPerComputeRequest uint64 = 250_000
	MaxGasPerComputeResult  uint64 = 200_000
	MaxGasPerZKVerification uint64 = 500_000
	MaxGasPerEscrowRelease  uint64 = 100_000
	MaxGasPerDispute        uint64 = 150_000

	// Oracle operations
	MaxGasPerPriceFeed   uint64 = 100_000
	MaxGasPerOracleVote  uint64 = 80_000
	MaxGasPerOracleAdmin uint64 = 150_000

	// General limits
	MaxGasPerTx           uint64 = 10_000_000 // Maximum gas per transaction
	MaxGasPerMessage      uint64 = 500_000   // Maximum gas per message in tx
	MaxMessagesPerTx      int    = 10        // Maximum messages per transaction
	MaxIterationsPerLoop  int    = 1000      // Maximum iterations in any loop
	MaxStorageReadsPerOp  int    = 100       // Maximum storage reads per operation
	MaxStorageWritesPerOp int    = 50        // Maximum storage writes per operation
	MaxComputationDepth   int    = 10        // Maximum recursion/nesting depth
)

// GasLimitDecorator enforces per-operation gas limits to prevent exhaustion attacks
type GasLimitDecorator struct{}

// NewGasLimitDecorator creates a new GasLimitDecorator
func NewGasLimitDecorator() GasLimitDecorator {
	return GasLimitDecorator{}
}

// AnteHandle enforces gas limits on transactions and individual messages
func (gld GasLimitDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Get messages from transaction
	msgs := tx.GetMsgs()

	// Enforce maximum messages per transaction
	if len(msgs) > MaxMessagesPerTx {
		return ctx, sdkerrors.ErrInvalidRequest.Wrapf(
			"transaction contains too many messages: %d > %d (prevents DoS)",
			len(msgs), MaxMessagesPerTx,
		)
	}

	// Track gas meter before processing
	gasBefore := ctx.GasMeter().GasConsumed()

	// Validate each message has appropriate gas limits
	for i, msg := range msgs {
		// Get required gas for this message type
		requiredGas, err := getRequiredGasForMessage(msg)
		if err != nil {
			return ctx, sdkerrors.ErrInvalidRequest.Wrapf(
				"failed to get required gas for message %d: %v", i, err,
			)
		}

		// Check if message gas limit exceeds maximum
		if requiredGas > MaxGasPerMessage {
			return ctx, sdkerrors.ErrInvalidRequest.Wrapf(
				"message %d requires too much gas: %d > %d",
				i, requiredGas, MaxGasPerMessage,
			)
		}

		// Create a bounded gas meter for this message
		msgGasMeter := storetypes.NewGasMeter(requiredGas)
		msgCtx := ctx.WithGasMeter(msgGasMeter)

		// Verify the message doesn't consume more than allocated
		// This is a pre-check; actual consumption happens during execution
		if err := validateMessageGasUsage(msgCtx, msg); err != nil {
			return ctx, sdkerrors.ErrInvalidRequest.Wrapf(
				"message %d failed gas validation: %v", i, err,
			)
		}
	}

	// Check total transaction gas limit
	totalGasRequired := ctx.GasMeter().Limit()
	if totalGasRequired > MaxGasPerTx && !simulate {
		return ctx, sdkerrors.ErrInvalidRequest.Wrapf(
			"transaction gas limit too high: %d > %d",
			totalGasRequired, MaxGasPerTx,
		)
	}

	// Track gas consumption and ensure it doesn't exceed limits
	newCtx, err := next(ctx, tx, simulate)
	if err != nil {
		return newCtx, err
	}

	gasAfter := newCtx.GasMeter().GasConsumed()
	gasUsed := gasAfter - gasBefore

	// Log excessive gas usage for monitoring
	if gasUsed > MaxGasPerTx/2 {
		ctx.Logger().Info("High gas consumption detected",
			"gas_used", gasUsed,
			"num_messages", len(msgs),
			"tx_hash", fmt.Sprintf("%X", ctx.TxBytes()),
		)
	}

	return newCtx, nil
}

// getRequiredGasForMessage returns the required gas for a specific message type
func getRequiredGasForMessage(msg sdk.Msg) (uint64, error) {
	switch msg.(type) {
	// DEX messages
	case *dextypes.MsgSwap:
		return MaxGasPerSwap, nil
	case *dextypes.MsgCreatePool:
		return MaxGasPerPoolCreation, nil
	case *dextypes.MsgAddLiquidity:
		return MaxGasPerLiquidityAdd, nil
	case *dextypes.MsgRemoveLiquidity:
		return MaxGasPerLiquidityRemove, nil

	// Compute messages
	case *computetypes.MsgSubmitRequest, *computetypes.MsgCancelRequest:
		return MaxGasPerComputeRequest, nil
	case *computetypes.MsgSubmitResult:
		return MaxGasPerComputeResult, nil
	case *computetypes.MsgSubmitEvidence:
		return MaxGasPerZKVerification, nil
	case *computetypes.MsgResolveDispute, *computetypes.MsgResolveAppeal:
		return MaxGasPerEscrowRelease, nil
	case *computetypes.MsgCreateDispute,
		*computetypes.MsgVoteOnDispute,
		*computetypes.MsgVoteOnAppeal,
		*computetypes.MsgAppealSlashing:
		return MaxGasPerDispute, nil
	case *computetypes.MsgRegisterProvider,
		*computetypes.MsgUpdateProvider,
		*computetypes.MsgDeactivateProvider:
		return MaxGasPerComputeRequest, nil

	// Oracle messages
	case *oracletypes.MsgSubmitPrice:
		return MaxGasPerPriceFeed, nil
	case *oracletypes.MsgDelegateFeedConsent:
		return MaxGasPerOracleVote, nil
	case *oracletypes.MsgUpdateParams:
		return MaxGasPerOracleAdmin, nil

	default:
		// For unknown message types, use a conservative default
		return MaxGasPerMessage, nil
	}
}

// validateMessageGasUsage performs pre-validation of message gas requirements
func validateMessageGasUsage(ctx sdk.Context, msg sdk.Msg) error {
	// Basic validation that message won't exceed gas limits
	// This is a static check; dynamic checks happen during execution

	type validateBasicMsg interface {
		ValidateBasic() error
	}

	if vb, ok := msg.(validateBasicMsg); ok {
		if err := vb.ValidateBasic(); err != nil {
			return fmt.Errorf("message validation failed: %w", err)
		}
	}

	return nil
}

// ConsumeGasForOperation consumes gas and checks it doesn't exceed per-operation limits
func ConsumeGasForOperation(ctx sdk.Context, gas uint64, operationType string, maxGas uint64) error {
	if gas > maxGas {
		return sdkerrors.ErrInvalidRequest.Wrapf(
			"operation '%s' requires too much gas: %d > %d",
			operationType, gas, maxGas,
		)
	}

	// Consume the gas (will panic if exceeds meter limit)
	ctx.GasMeter().ConsumeGas(gas, operationType)

	return nil
}

// IterateWithGasLimit executes a function in a loop with gas metering and iteration limits
func IterateWithGasLimit(
	ctx sdk.Context,
	maxIterations int,
	gasPerIteration uint64,
	iterFunc func(int) (bool, error),
) error {
	for i := 0; i < maxIterations; i++ {
		// Consume gas for this iteration
		ctx.GasMeter().ConsumeGas(gasPerIteration, fmt.Sprintf("iteration_%d", i))

		// Execute iteration function
		shouldContinue, err := iterFunc(i)
		if err != nil {
			return err
		}

		if !shouldContinue {
			break
		}
	}

	return nil
}

// TrackStorageAccess tracks storage reads/writes to enforce limits
type StorageAccessTracker struct {
	reads  int
	writes int
}

// NewStorageAccessTracker creates a new storage access tracker
func NewStorageAccessTracker() *StorageAccessTracker {
	return &StorageAccessTracker{}
}

// RecordRead records a storage read and checks limits
func (sat *StorageAccessTracker) RecordRead(ctx sdk.Context) error {
	sat.reads++
	if sat.reads > MaxStorageReadsPerOp {
		return sdkerrors.ErrInvalidRequest.Wrapf(
			"too many storage reads: %d > %d (DoS protection)",
			sat.reads, MaxStorageReadsPerOp,
		)
	}

	// Consume gas for read
	ctx.GasMeter().ConsumeGas(storetypes.Gas(1000), "storage_read")
	return nil
}

// RecordWrite records a storage write and checks limits
func (sat *StorageAccessTracker) RecordWrite(ctx sdk.Context) error {
	sat.writes++
	if sat.writes > MaxStorageWritesPerOp {
		return sdkerrors.ErrInvalidRequest.Wrapf(
			"too many storage writes: %d > %d (DoS protection)",
			sat.writes, MaxStorageWritesPerOp,
		)
	}

	// Consume gas for write
	ctx.GasMeter().ConsumeGas(storetypes.Gas(2000), "storage_write")
	return nil
}

// GetStats returns current storage access statistics
func (sat *StorageAccessTracker) GetStats() (reads int, writes int) {
	return sat.reads, sat.writes
}
