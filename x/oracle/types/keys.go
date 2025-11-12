package types

const (
	// ModuleName defines the module name
	ModuleName = "oracle"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// RouterKey is the message route for oracle
	RouterKey = ModuleName

	// QuerierRoute defines the module's query routing key
	QuerierRoute = ModuleName

	// MemStoreKey defines the in-memory store key
	MemStoreKey = "mem_oracle"
)

func KeyPrefix(p string) []byte {
	return []byte(p)
}

const (
	// PriceFeedKeyPrefix is the prefix for price feed store
	PriceFeedKeyPrefix = "PriceFeed/value/"

	// ValidatorKeyPrefix is the prefix for oracle validator store
	ValidatorKeyPrefix = "Validator/value/"
)
