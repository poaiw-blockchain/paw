package keeper

import (
	"cosmossdk.io/errors"
	"github.com/paw-chain/paw/x/dex/types"
)

var (
	// ErrInvalidRequest is returned when a request is invalid
	ErrInvalidRequest = errors.Register(types.ModuleName, 100, "invalid request")
)
