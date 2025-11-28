package keeper

import (
	"fmt"
	"runtime/debug"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TASK 91: Panic recovery in all msg handlers

// RecoverPanic recovers from panics in message handlers and returns appropriate error
func RecoverPanic(ctx sdk.Context, handler string) error {
	if r := recover(); r != nil {
		// Capture stack trace
		stackTrace := string(debug.Stack())

		// Log the panic with full details
		ctx.Logger().Error("PANIC RECOVERED",
			"handler", handler,
			"panic", fmt.Sprintf("%v", r),
			"stack_trace", stackTrace,
		)

		// Emit critical event for monitoring
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"panic_recovered",
				sdk.NewAttribute("handler", handler),
				sdk.NewAttribute("error", fmt.Sprintf("%v", r)),
				sdk.NewAttribute("severity", "critical"),
			),
		)

		// Return error instead of propagating panic
		return fmt.Errorf("panic in %s: %v", handler, r)
	}
	return nil
}

// SafeExecute wraps a function with panic recovery
func SafeExecute(ctx sdk.Context, handler string, fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = RecoverPanic(ctx, handler)
		}
	}()

	return fn()
}

// SafeExecuteWithReturn wraps a function with return value and panic recovery
func SafeExecuteWithReturn[T any](ctx sdk.Context, handler string, fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = RecoverPanic(ctx, handler)
			var zero T
			result = zero
		}
	}()

	return fn()
}
