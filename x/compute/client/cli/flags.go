package cli

// Flag constants for compute CLI commands
const (
	// Provider flags
	FlagMoniker         = "moniker"
	FlagEndpoint        = "endpoint"
	FlagProviderAddress = "provider-address"

	// Compute spec flags
	FlagCpuCores       = "cpu-cores"
	FlagMemoryMb       = "memory-mb"
	FlagDiskMb         = "disk-mb"
	FlagGpuUnits       = "gpu-units"
	FlagTimeoutSeconds = "timeout-seconds"

	// Pricing flags
	FlagCpuPrice     = "cpu-price"
	FlagMemoryPrice  = "memory-price"
	FlagGpuPrice     = "gpu-price"
	FlagStoragePrice = "storage-price"

	// Request flags
	FlagContainerImage    = "container-image"
	FlagCommand           = "command"
	FlagEnvVars           = "env-vars"
	FlagMaxPayment        = "max-payment"
	FlagPreferredProvider = "preferred-provider"

	// Result flags
	FlagOutputHash        = "output-hash"
	FlagOutputURL         = "output-url"
	FlagExitCode          = "exit-code"
	FlagLogsURL           = "logs-url"
	FlagVerificationProof = "verification-proof"

	// Dispute flags
	FlagReason        = "reason"
	FlagEvidence      = "evidence"
	FlagDepositAmount = "deposit-amount"

	// Vote flags
	FlagVote          = "vote"
	FlagJustification = "justification"
)
