package cometmock

import (
	"errors"
)

var (
	// Configuration errors
	ErrInvalidValidatorCount = errors.New("validator count must be at least 1")
	ErrInvalidBlockTime      = errors.New("block time must be at least 1ms")
	ErrInvalidBlockSize      = errors.New("max block size must be positive")
	ErrInvalidMaxGas         = errors.New("max gas must be positive")
	ErrMissingChainID        = errors.New("chain ID cannot be empty")
	ErrInvalidInitialHeight  = errors.New("initial height must be at least 1")

	// Runtime errors
	ErrBlockNotStarted     = errors.New("block not started - call BeginBlock first")
	ErrBlockAlreadyStarted = errors.New("block already started - call EndBlock first")
	ErrInvalidTransaction  = errors.New("invalid transaction format")
	ErrValidatorNotFound   = errors.New("validator not found")
)
