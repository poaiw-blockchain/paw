package types

const (
	// ModuleName defines the module name
	ModuleName = "compute"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for compute
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_compute"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	// TaskKeyPrefix is the prefix for task store
	TaskKeyPrefix = "Task/value/"

	// ProviderKeyPrefix is the prefix for compute provider store
	ProviderKeyPrefix = "Provider/value/"
)
