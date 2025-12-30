// Package abci provides shared utilities for ABCI error handling across modules.
// ARCH-4: Standardizes error handling with severity classification and metrics.
package abci

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ErrorSeverity classifies the severity of ABCI blocker errors.
// Higher severity errors may warrant operator attention.
type ErrorSeverity int

const (
	// SeverityLow indicates minor errors that don't affect core functionality.
	// Examples: cleanup failures, cache refresh issues
	SeverityLow ErrorSeverity = iota

	// SeverityMedium indicates errors that degrade functionality but don't break core operations.
	// Examples: fee distribution failure, reputation update issues
	SeverityMedium

	// SeverityHigh indicates errors affecting important operations.
	// Examples: escrow processing failure, order matching issues
	SeverityHigh

	// SeverityCritical indicates errors that may affect chain integrity.
	// Examples: consensus-related failures (should be rare in proper ABCI handlers)
	SeverityCritical
)

// String returns the string representation of the severity level.
func (s ErrorSeverity) String() string {
	switch s {
	case SeverityLow:
		return "low"
	case SeverityMedium:
		return "medium"
	case SeverityHigh:
		return "high"
	case SeverityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// BlockerErrorHandler provides standardized error handling for ABCI blockers.
// It logs errors with severity, emits monitoring events, and optionally records metrics.
type BlockerErrorHandler struct {
	moduleName string
	ctx        sdk.Context
}

// NewBlockerErrorHandler creates a new error handler for the given module.
func NewBlockerErrorHandler(ctx sdk.Context, moduleName string) *BlockerErrorHandler {
	return &BlockerErrorHandler{
		moduleName: moduleName,
		ctx:        ctx,
	}
}

// HandleError logs and emits an event for an error with the given severity.
// ABCI blockers should NOT return errors (to avoid halting the chain),
// so this method logs, emits events, and returns - callers should continue.
func (h *BlockerErrorHandler) HandleError(operation string, severity ErrorSeverity, err error) {
	if err == nil {
		return
	}

	// Log with severity level
	switch severity {
	case SeverityCritical:
		h.ctx.Logger().Error("CRITICAL ABCI error",
			"module", h.moduleName,
			"operation", operation,
			"severity", severity.String(),
			"error", err.Error(),
		)
	case SeverityHigh:
		h.ctx.Logger().Error("ABCI blocker error",
			"module", h.moduleName,
			"operation", operation,
			"severity", severity.String(),
			"error", err.Error(),
		)
	case SeverityMedium:
		h.ctx.Logger().Warn("ABCI blocker warning",
			"module", h.moduleName,
			"operation", operation,
			"severity", severity.String(),
			"error", err.Error(),
		)
	default:
		h.ctx.Logger().Debug("ABCI blocker minor issue",
			"module", h.moduleName,
			"operation", operation,
			"severity", severity.String(),
			"error", err.Error(),
		)
	}

	// Emit structured event for monitoring
	h.ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"abci_blocker_error",
			sdk.NewAttribute("module", h.moduleName),
			sdk.NewAttribute("operation", operation),
			sdk.NewAttribute("severity", severity.String()),
			sdk.NewAttribute("error", err.Error()),
			sdk.NewAttribute("height", fmt.Sprintf("%d", h.ctx.BlockHeight())),
		),
	)
}

// WrapError is a convenience method for handling errors inline.
// Returns true if there was an error (for use in if statements).
// Example:
//
//	if handler.WrapError("cleanup", SeverityLow, k.Cleanup(ctx)) {
//	    // error was handled, continue to next operation
//	}
func (h *BlockerErrorHandler) WrapError(operation string, severity ErrorSeverity, err error) bool {
	if err != nil {
		h.HandleError(operation, severity, err)
		return true
	}
	return false
}
