package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/go-bip39"
)

// Config holds the PAW client configuration
type Config struct {
	RPCEndpoint   string
	GRPCEndpoint  string
	ChainID       string
	Prefix        string
	GasPrice      string
	GasAdjustment float64
}

// Client provides helper methods for interacting with PAW blockchain
type Client struct {
	config     Config
	grpcConn   *grpc.ClientConn
	clientCtx  client.Context
	txFactory  tx.Factory
	keyring    keyring.Keyring
}

// NewClient creates a new PAW client
func NewClient(config Config) (*Client, error) {
	// Set default values
	if config.Prefix == "" {
		config.Prefix = "paw"
	}
	if config.GasPrice == "" {
		config.GasPrice = "0.025upaw"
	}
	if config.GasAdjustment == 0 {
		config.GasAdjustment = 1.5
	}

	// Set address prefix
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetBech32PrefixForAccount(config.Prefix, config.Prefix+"pub")
	sdkConfig.SetBech32PrefixForValidator(config.Prefix+"valoper", config.Prefix+"valoperpub")
	sdkConfig.SetBech32PrefixForConsensusNode(config.Prefix+"valcons", config.Prefix+"valconspub")

	// Create GRPC connection
	grpcConn, err := grpc.NewClient(
		config.GRPCEndpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to GRPC: %w", err)
	}

	// Create keyring
	kr := keyring.NewInMemory(MakeEncodingConfig().Codec)

	// Create client context
	clientCtx := client.Context{}.
		WithGRPCClient(grpcConn).
		WithChainID(config.ChainID).
		WithKeyring(kr).
		WithBroadcastMode(flags.BroadcastSync)

	// Create transaction factory
	txFactory := tx.Factory{}.
		WithChainID(config.ChainID).
		WithGas(flags.DefaultGasLimit).
		WithGasAdjustment(config.GasAdjustment).
		WithGasPrices(config.GasPrice).
		WithSignMode(signing.SignMode_SIGN_MODE_DIRECT).
		WithKeybase(kr)

	return &Client{
		config:    config,
		grpcConn:  grpcConn,
		clientCtx: clientCtx,
		txFactory: txFactory,
		keyring:   kr,
	}, nil
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.grpcConn != nil {
		return c.grpcConn.Close()
	}
	return nil
}

// ImportWalletFromMnemonic imports a wallet from mnemonic
func (c *Client) ImportWalletFromMnemonic(name, mnemonic, hdPath string) (sdk.AccAddress, error) {
	if hdPath == "" {
		hdPath = hd.CreateHDPath(118, 0, 0).String()
	}

	// Validate mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic")
	}

	// Import key
	info, err := c.keyring.NewAccount(name, mnemonic, "", hdPath, hd.Secp256k1)
	if err != nil {
		return nil, fmt.Errorf("failed to import account: %w", err)
	}

	addr, err := info.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	return addr, nil
}

// GetBalance gets the balance of an address
func (c *Client) GetBalance(ctx context.Context, address string, denom string) (*sdk.Coin, error) {
	queryClient := banktypes.NewQueryClient(c.grpcConn)

	res, err := queryClient.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: address,
		Denom:   denom,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query balance: %w", err)
	}

	return res.Balance, nil
}

// GetAllBalances gets all balances of an address
func (c *Client) GetAllBalances(ctx context.Context, address string) (sdk.Coins, error) {
	queryClient := banktypes.NewQueryClient(c.grpcConn)

	res, err := queryClient.AllBalances(ctx, &banktypes.QueryAllBalancesRequest{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query balances: %w", err)
	}

	return res.Balances, nil
}

// GetAccount gets account information
func (c *Client) GetAccount(ctx context.Context, address string) (authtypes.AccountI, error) {
	queryClient := authtypes.NewQueryClient(c.grpcConn)

	res, err := queryClient.Account(ctx, &authtypes.QueryAccountRequest{
		Address: address,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query account: %w", err)
	}

	var account authtypes.AccountI
	if err := c.clientCtx.InterfaceRegistry.UnpackAny(res.Account, &account); err != nil {
		return nil, fmt.Errorf("failed to unpack account: %w", err)
	}

	return account, nil
}

// BroadcastTx broadcasts a transaction
func (c *Client) BroadcastTx(ctx context.Context, txBytes []byte) (*sdk.TxResponse, error) {
	return c.clientCtx.BroadcastTx(txBytes)
}

// SignAndBroadcast signs and broadcasts messages
func (c *Client) SignAndBroadcast(ctx context.Context, fromName string, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	// Get account info
	fromInfo, err := c.keyring.Key(fromName)
	if err != nil {
		return nil, fmt.Errorf("failed to get key: %w", err)
	}

	fromAddr, err := fromInfo.GetAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	// Build transaction
	txBuilder := c.clientCtx.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msgs...); err != nil {
		return nil, fmt.Errorf("failed to set messages: %w", err)
	}

	// Set fee (simplified)
	txBuilder.SetGasLimit(flags.DefaultGasLimit)
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin("upaw", sdk.NewInt(5000))))

	// Get account number and sequence
	account, err := c.GetAccount(ctx, fromAddr.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get account: %w", err)
	}

	// Sign transaction
	signerData := authsigning.SignerData{
		ChainID:       c.config.ChainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
	}

	sigV2 := signing.SignatureV2{
		PubKey: fromInfo.GetPubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode_SIGN_MODE_DIRECT,
			Signature: nil,
		},
		Sequence: account.GetSequence(),
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, fmt.Errorf("failed to set signatures: %w", err)
	}

	// Generate signature
	sigV2, err = tx.SignWithPrivKey(
		ctx,
		signing.SignMode_SIGN_MODE_DIRECT,
		signerData,
		txBuilder,
		fromInfo.GetPubKey(),
		c.clientCtx.TxConfig.SignModeHandler(),
		account.GetSequence(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	if err := txBuilder.SetSignatures(sigV2); err != nil {
		return nil, fmt.Errorf("failed to set final signatures: %w", err)
	}

	// Encode and broadcast
	txBytes, err := c.clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to encode transaction: %w", err)
	}

	return c.BroadcastTx(ctx, txBytes)
}

// GetChainID returns the chain ID
func (c *Client) GetChainID() string {
	return c.config.ChainID
}

// GetClientContext returns the client context
func (c *Client) GetClientContext() client.Context {
	return c.clientCtx
}
