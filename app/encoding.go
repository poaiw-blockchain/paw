package app

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/gogoproto/proto"

	tendermintclient "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"

	"cosmossdk.io/x/tx/signing"
)

// EncodingConfig specifies the concrete encoding types to use for a given app.
type EncodingConfig struct {
	InterfaceRegistry codectypes.InterfaceRegistry
	Codec             codec.Codec
	TxConfig          client.TxConfig
	Amino             *codec.LegacyAmino
}

// makeBaseEncodingConfig builds the encoding config with only the standard SDK
// interfaces registered. Module interfaces are added later once the module
// manager is initialized to ensure module basics have their codec wiring set.
func makeBaseEncodingConfig() EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry, err := codectypes.NewInterfaceRegistryWithOptions(codectypes.InterfaceRegistryOptions{
		ProtoFiles: proto.HybridResolver,
		SigningOptions: signing.Options{
			AddressCodec:          address.NewBech32Codec(Bech32PrefixAccAddr),
			ValidatorAddressCodec: address.NewBech32Codec(Bech32PrefixValAddr),
		},
	})
	if err != nil {
		panic(err)
	}

	// Register standard interfaces first (includes crypto types)
	std.RegisterInterfaces(interfaceRegistry)
	tendermintclient.RegisterInterfaces(interfaceRegistry)

	cdc := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(cdc, tx.DefaultSignModes)

	return EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             cdc,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}

// MakeEncodingConfig creates an EncodingConfig for an amino based test configuration.
func MakeEncodingConfig() EncodingConfig {
	cfg := makeBaseEncodingConfig()
	bm := GetBasicManager()

	// Register module interfaces after the module basics have been built with
	// the correct codec/address codec wiring.
	bm.RegisterInterfaces(cfg.InterfaceRegistry)
	RegisterLegacyAminoCodecWithManager(cfg.Amino, bm)

	return cfg
}

// RegisterLegacyAminoCodec registers the sdk message type using the default
// module basic manager.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	RegisterLegacyAminoCodecWithManager(cdc, GetBasicManager())
}

// RegisterLegacyAminoCodecWithManager registers amino codecs using the provided
// module basic manager. This is useful when the basic manager is constructed
// lazily after module wiring.
func RegisterLegacyAminoCodecWithManager(cdc *codec.LegacyAmino, bm module.BasicManager) {
	std.RegisterLegacyAminoCodec(cdc)
	bm.RegisterLegacyAminoCodec(cdc)
}
