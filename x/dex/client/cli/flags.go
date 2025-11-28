package cli

// Flag constants for dex CLI commands
const (
	// Pool creation flags
	FlagTokenA  = "token-a"
	FlagTokenB  = "token-b"
	FlagAmountA = "amount-a"
	FlagAmountB = "amount-b"

	// Liquidity flags
	FlagShares = "shares"

	// Swap flags
	FlagTokenIn      = "token-in"
	FlagTokenOut     = "token-out"
	FlagAmountIn     = "amount-in"
	FlagMinAmountOut = "min-amount-out"
	FlagSlippage     = "slippage"
	FlagDeadline     = "deadline"
)
