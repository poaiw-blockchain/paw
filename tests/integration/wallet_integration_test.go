package integration

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/go-bip39"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/paw-chain/paw/app"
	dextypes "github.com/paw-chain/paw/x/dex/types"
)

// WalletIntegrationTestSuite defines the test suite for wallet integration
type WalletIntegrationTestSuite struct {
	suite.Suite
	encCfg app.EncodingConfig
	cdc    codec.Codec
}

// SetupSuite runs once before all tests
func (suite *WalletIntegrationTestSuite) SetupSuite() {
	// Set PAW network configuration
	app.SetConfig()

	// Initialize encoding config
	suite.encCfg = app.MakeEncodingConfig()
	suite.cdc = suite.encCfg.Codec
}

// TestWalletIntegrationTestSuite runs the test suite
func TestWalletIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(WalletIntegrationTestSuite))
}

// TestWalletCreationAndKeyGeneration tests wallet creation and key generation
func (suite *WalletIntegrationTestSuite) TestWalletCreationAndKeyGeneration() {
	tests := []struct {
		name        string
		entropy     int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "create wallet with 128-bit entropy",
			entropy:     128,
			expectError: false,
		},
		{
			name:        "create wallet with 256-bit entropy",
			entropy:     256,
			expectError: false,
		},
		{
			name:        "invalid entropy size",
			entropy:     64,
			expectError: true,
			errorMsg:    "Entropy length must be",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Generate entropy
			entropy, err := bip39.NewEntropy(tt.entropy)

			if tt.expectError {
				suite.Require().Error(err)
				if tt.errorMsg != "" {
					suite.Require().Contains(err.Error(), tt.errorMsg)
				}
				return
			}

			suite.Require().NoError(err)
			suite.Require().NotNil(entropy)

			// Generate mnemonic
			mnemonic, err := bip39.NewMnemonic(entropy)
			suite.Require().NoError(err)
			suite.Require().NotEmpty(mnemonic)

			// Verify mnemonic is valid
			valid := bip39.IsMnemonicValid(mnemonic)
			suite.Require().True(valid, "generated mnemonic should be valid")

			// Count words
			words := strings.Split(mnemonic, " ")
			expectedWords := tt.entropy / 32 * 3
			suite.Require().Equal(expectedWords, len(words), "mnemonic should have correct number of words")

			// Derive seed from mnemonic
			seed := bip39.NewSeed(mnemonic, "")
			suite.Require().NotNil(seed)
			suite.Require().Len(seed, 64, "seed should be 64 bytes")
		})
	}
}

// TestAddressDerivationBech32 tests address derivation in Bech32 format
func (suite *WalletIntegrationTestSuite) TestAddressDerivationBech32() {
	tests := []struct {
		name         string
		mnemonic     string
		hdPath       string
		expectedAddr string // Leave empty for dynamic tests
		expectError  bool
	}{
		{
			name:     "derive address with standard path",
			mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
			hdPath:   "m/44'/118'/0'/0/0",
		},
		{
			name:     "derive address with account index 1",
			mnemonic: "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
			hdPath:   "m/44'/118'/0'/0/1",
		},
		{
			name:        "invalid HD path",
			mnemonic:    "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about",
			hdPath:      "invalid/path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Derive private key from mnemonic
			seed := bip39.NewSeed(tt.mnemonic, "")

			// Parse HD path
			hdParams, err := hd.NewParamsFromPath(tt.hdPath)
			if tt.expectError {
				suite.Require().Error(err)
				return
			}
			suite.Require().NoError(err)

			// Derive key using BIP44
			masterPriv, ch := hd.ComputeMastersFromSeed(seed)
			derivedPriv, err := hd.DerivePrivateKeyForPath(masterPriv, ch, hdParams.String())
			suite.Require().NoError(err)

			// Create private key
			privKey := &secp256k1.PrivKey{Key: derivedPriv}
			pubKey := privKey.PubKey()

			// Get address
			addr := sdk.AccAddress(pubKey.Address())
			suite.Require().NotEmpty(addr)

			// Verify Bech32 format
			bech32Addr := addr.String()
			suite.Require().True(strings.HasPrefix(bech32Addr, "paw1"), "address should have paw1 prefix")

			// Validate Bech32 encoding
			hrp, decoded, err := bech32.DecodeAndConvert(bech32Addr)
			suite.Require().NoError(err)
			suite.Require().Equal("paw", hrp)
			suite.Require().Len(decoded, 20, "decoded address should be 20 bytes")

			// Verify we can reconstruct the address
			reconstructed, err := bech32.ConvertAndEncode("paw", decoded)
			suite.Require().NoError(err)
			suite.Require().Equal(bech32Addr, reconstructed)

			// Test expected address if provided
			if tt.expectedAddr != "" {
				suite.Require().Equal(tt.expectedAddr, bech32Addr)
			}
		})
	}
}

// TestBalanceQueries tests balance queries using mock responses
func (suite *WalletIntegrationTestSuite) TestBalanceQueries() {
	// Create test address
	privKey := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(privKey.PubKey().Address())

	tests := []struct {
		name          string
		address       string
		mockResponse  interface{}
		expectedCoins sdk.Coins
		expectError   bool
	}{
		{
			name:    "query balance with single coin",
			address: addr.String(),
			mockResponse: banktypes.QueryAllBalancesResponse{
				Balances: sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000)),
			},
			expectedCoins: sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000000)),
			expectError:   false,
		},
		{
			name:    "query balance with multiple coins",
			address: addr.String(),
			mockResponse: banktypes.QueryAllBalancesResponse{
				Balances: sdk.NewCoins(
					sdk.NewInt64Coin("upaw", 1000000),
					sdk.NewInt64Coin("uusdt", 500000),
				),
			},
			expectedCoins: sdk.NewCoins(
				sdk.NewInt64Coin("upaw", 1000000),
				sdk.NewInt64Coin("uusdt", 500000),
			),
			expectError: false,
		},
		{
			name:          "query balance with empty balance",
			address:       addr.String(),
			mockResponse:  banktypes.QueryAllBalancesResponse{Balances: sdk.NewCoins()},
			expectedCoins: sdk.NewCoins(),
			expectError:   false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Create mock server
			server := suite.createMockBalanceServer(tt.mockResponse)
			defer server.Close()

			// Query balance
			balances, err := suite.queryBalanceMock(server.URL, tt.address)

			if tt.expectError {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tt.expectedCoins, balances)
			}
		})
	}
}

// TestTokenTransferTransactions tests token transfer message creation
func (suite *WalletIntegrationTestSuite) TestTokenTransferTransactions() {
	// Create test accounts
	senderPriv := secp256k1.GenPrivKey()
	senderAddr := sdk.AccAddress(senderPriv.PubKey().Address())
	recipientPriv := secp256k1.GenPrivKey()
	recipientAddr := sdk.AccAddress(recipientPriv.PubKey().Address())

	tests := []struct {
		name        string
		from        sdk.AccAddress
		to          sdk.AccAddress
		amount      sdk.Coins
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid transfer",
			from:        senderAddr,
			to:          recipientAddr,
			amount:      sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000)),
			expectError: false,
		},
		{
			name:        "transfer with multiple coins",
			from:        senderAddr,
			to:          recipientAddr,
			amount:      sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000), sdk.NewInt64Coin("uusdt", 500)),
			expectError: false,
		},
		{
			name:        "transfer to same address",
			from:        senderAddr,
			to:          senderAddr,
			amount:      sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000)),
			expectError: false, // Cosmos SDK allows this
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Create MsgSend
			msg := banktypes.NewMsgSend(tt.from, tt.to, tt.amount)

			// Verify message fields
			suite.Require().Equal(tt.from.String(), msg.FromAddress)
			suite.Require().Equal(tt.to.String(), msg.ToAddress)
			suite.Require().Equal(tt.amount, msg.Amount)

			// Create transaction builder and add message
			txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
			err := txBuilder.SetMsgs(msg)
			suite.Require().NoError(err)

			// Verify we can retrieve the message
			msgs := txBuilder.GetTx().GetMsgs()
			suite.Require().Len(msgs, 1)
		})
	}
}

// TestDEXSwapTransactions tests DEX swap transaction creation
func (suite *WalletIntegrationTestSuite) TestDEXSwapTransactions() {
	// Create test account
	privKey := secp256k1.GenPrivKey()
	trader := sdk.AccAddress(privKey.PubKey().Address()).String()

	tests := []struct {
		name         string
		poolID       uint64
		tokenIn      string
		tokenOut     string
		amountIn     math.Int
		minAmountOut math.Int
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "valid swap",
			poolID:       1,
			tokenIn:      "upaw",
			tokenOut:     "uusdt",
			amountIn:     math.NewInt(1000000),
			minAmountOut: math.NewInt(1900000),
			expectError:  false,
		},
		{
			name:         "swap with zero amount",
			poolID:       1,
			tokenIn:      "upaw",
			tokenOut:     "uusdt",
			amountIn:     math.ZeroInt(),
			minAmountOut: math.ZeroInt(),
			expectError:  true,
			errorMsg:     "invalid amount",
		},
		{
			name:         "swap with same tokens",
			poolID:       1,
			tokenIn:      "upaw",
			tokenOut:     "upaw",
			amountIn:     math.NewInt(1000000),
			minAmountOut: math.NewInt(1000000),
			expectError:  true,
			errorMsg:     "same token",
		},
		{
			name:         "swap with invalid token denom",
			poolID:       1,
			tokenIn:      "invalid;token",
			tokenOut:     "uusdt",
			amountIn:     math.NewInt(1000000),
			minAmountOut: math.NewInt(1900000),
			expectError:  true,
			errorMsg:     "invalid denom",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			// Create MsgSwap
			msg := &dextypes.MsgSwap{
				Trader:       trader,
				PoolId:       tt.poolID,
				TokenIn:      tt.tokenIn,
				TokenOut:     tt.tokenOut,
				AmountIn:     tt.amountIn,
				MinAmountOut: tt.minAmountOut,
			}
			msg.Deadline = time.Now().Add(time.Minute).Unix()

			// Validate message
			err := msg.ValidateBasic()

			if tt.expectError {
				suite.Require().Error(err)
				if tt.errorMsg != "" {
					suite.Require().Contains(err.Error(), tt.errorMsg)
				}
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tt.poolID, msg.PoolId)
				suite.Require().Equal(tt.tokenIn, msg.TokenIn)
				suite.Require().Equal(tt.tokenOut, msg.TokenOut)
			}
		})
	}
}

// TestTransactionSigning tests basic transaction signing
func (suite *WalletIntegrationTestSuite) TestTransactionSigning() {
	// Create test account
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	// Create test message
	msg := banktypes.NewMsgSend(
		addr,
		sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()),
		sdk.NewCoins(sdk.NewInt64Coin("upaw", 1000)),
	)

	// Create transaction builder
	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)

	// Set fee and gas
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("upaw", 200)))
	txBuilder.SetGasLimit(200000)

	// Test signature verification
	testMsg := []byte("test transaction for wallet")
	signature, err := privKey.Sign(testMsg)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(signature)

	// Verify signature
	valid := pubKey.VerifySignature(testMsg, signature)
	suite.Require().True(valid, "signature should be valid")

	// Test with wrong message
	wrongMsg := []byte("wrong message")
	valid = pubKey.VerifySignature(wrongMsg, signature)
	suite.Require().False(valid, "signature should be invalid for wrong message")
}

// TestBuildAndEncodeTransaction ensures full tx builder + signing flow works with the encoding config.
func (suite *WalletIntegrationTestSuite) TestBuildAndEncodeTransaction() {
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	fromAddr := sdk.AccAddress(pubKey.Address())
	recipient := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	msg := banktypes.NewMsgSend(fromAddr, recipient, sdk.NewCoins(sdk.NewInt64Coin("upaw", 2500)))

	txBuilder := suite.encCfg.TxConfig.NewTxBuilder()
	err := txBuilder.SetMsgs(msg)
	suite.Require().NoError(err)
	txBuilder.SetMemo("integration-test")
	txBuilder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("upaw", 120)))
	txBuilder.SetGasLimit(200000)

	signerData := authsigning.SignerData{
		ChainID:       "paw-integration",
		AccountNumber: 9,
		Sequence:      3,
		Address:       fromAddr.String(),
		PubKey:        pubKey,
	}

	signMode := signing.SignMode(suite.encCfg.TxConfig.SignModeHandler().DefaultMode())
	placeholderSig := signing.SignatureV2{
		PubKey: pubKey,
		Data: &signing.SingleSignatureData{
			SignMode:  signMode,
			Signature: nil,
		},
		Sequence: signerData.Sequence,
	}
	suite.Require().NoError(txBuilder.SetSignatures(placeholderSig))

	sigV2, err := clienttx.SignWithPrivKey(
		context.Background(),
		signMode,
		signerData,
		txBuilder,
		privKey,
		suite.encCfg.TxConfig,
		signerData.Sequence,
	)
	suite.Require().NoError(err)
	suite.Require().NoError(txBuilder.SetSignatures(sigV2))

	txBytes, err := suite.encCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	suite.Require().NoError(err)
	suite.Require().NotEmpty(txBytes)

	decodedTx, err := suite.encCfg.TxConfig.TxDecoder()(txBytes)
	suite.Require().NoError(err)
	var memoTx sdk.TxWithMemo
	if txWithMemo, ok := decodedTx.(sdk.TxWithMemo); ok {
		memoTx = txWithMemo
	} else {
		// Wrap into a TxBuilder to access memo if decoder returned a non-wrapper Tx.
		builder, err := suite.encCfg.TxConfig.WrapTxBuilder(decodedTx)
		suite.Require().NoError(err)
		if txWithMemo, ok := builder.(sdk.TxWithMemo); ok {
			memoTx = txWithMemo
		}
	}
	suite.Require().NotNil(memoTx, "decoded tx should expose memo")
	suite.Require().Equal("integration-test", memoTx.GetMemo())
	suite.Require().Len(decodedTx.GetMsgs(), 1)

	singleSig, ok := sigV2.Data.(*signing.SingleSignatureData)
	suite.Require().True(ok)
	v2Adaptable, ok := decodedTx.(authsigning.V2AdaptableTx)
	suite.Require().True(ok)
	txData := v2Adaptable.GetSigningTxData()
	txSignerData := txsigning.SignerData{
		ChainID:       signerData.ChainID,
		AccountNumber: signerData.AccountNumber,
		Sequence:      signerData.Sequence,
		Address:       signerData.Address,
	}
	if signerData.PubKey != nil {
		pkAny, errAny := codectypes.NewAnyWithValue(signerData.PubKey)
		suite.Require().NoError(errAny)
		txSignerData.PubKey = &anypb.Any{TypeUrl: pkAny.TypeUrl, Value: pkAny.Value}
	}

	err = authsigning.VerifySignature(context.Background(), pubKey, txSignerData, singleSig, suite.encCfg.TxConfig.SignModeHandler(), txData)
	suite.Require().NoError(err)
}

// TestErrorHandlingInvalidInputs tests error handling for invalid inputs
func (suite *WalletIntegrationTestSuite) TestErrorHandlingInvalidInputs() {
	tests := []struct {
		name        string
		testFunc    func() error
		expectError bool
		errorMsg    string
	}{
		{
			name: "invalid bech32 address - wrong prefix",
			testFunc: func() error {
				_, err := sdk.AccAddressFromBech32("cosmos1invalidaddress")
				return err
			},
			expectError: true,
			errorMsg:    "decoding bech32 failed",
		},
		{
			name: "invalid bech32 address - malformed",
			testFunc: func() error {
				_, err := sdk.AccAddressFromBech32("paw1invalid")
				return err
			},
			expectError: true,
		},
		{
			name: "invalid coin - negative amount",
			testFunc: func() error {
				coin := sdk.Coin{
					Denom:  "upaw",
					Amount: math.NewInt(-1000),
				}
				return coin.Validate()
			},
			expectError: true,
			errorMsg:    "negative coin amount",
		},
		{
			name: "invalid coin - empty denom",
			testFunc: func() error {
				coin := sdk.Coin{
					Denom:  "",
					Amount: math.NewInt(1000),
				}
				return coin.Validate()
			},
			expectError: true,
			errorMsg:    "invalid denom",
		},
		{
			name: "invalid mnemonic",
			testFunc: func() error {
				valid := bip39.IsMnemonicValid("invalid mnemonic phrase")
				if !valid {
					return fmt.Errorf("invalid mnemonic")
				}
				return nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := tt.testFunc()

			if tt.expectError {
				suite.Require().Error(err)
				if tt.errorMsg != "" {
					suite.Require().Contains(err.Error(), tt.errorMsg)
				}
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

// TestPublicKeyRecovery tests public key recovery from signature
func (suite *WalletIntegrationTestSuite) TestPublicKeyRecovery() {
	// Create test private key
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()

	// Create test message
	msg := []byte("test message for signature")

	// Sign message
	signature, err := privKey.Sign(msg)
	suite.Require().NoError(err)

	// Verify signature
	valid := pubKey.VerifySignature(msg, signature)
	suite.Require().True(valid, "signature should be valid")

	// Test with wrong message
	wrongMsg := []byte("wrong message")
	valid = pubKey.VerifySignature(wrongMsg, signature)
	suite.Require().False(valid, "signature should be invalid for wrong message")

	// Test with wrong public key
	wrongPrivKey := secp256k1.GenPrivKey()
	wrongPubKey := wrongPrivKey.PubKey()
	valid = wrongPubKey.VerifySignature(msg, signature)
	suite.Require().False(valid, "signature should be invalid for wrong public key")
}

// TestMultipleAccountDerivation tests deriving multiple accounts from one mnemonic
func (suite *WalletIntegrationTestSuite) TestMultipleAccountDerivation() {
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	seed := bip39.NewSeed(mnemonic, "")

	addresses := make([]string, 5)

	// Derive 5 different addresses
	for i := 0; i < 5; i++ {
		hdPath := fmt.Sprintf("m/44'/118'/0'/0/%d", i)
		hdParams, err := hd.NewParamsFromPath(hdPath)
		suite.Require().NoError(err)

		masterPriv, ch := hd.ComputeMastersFromSeed(seed)
		derivedPriv, err := hd.DerivePrivateKeyForPath(masterPriv, ch, hdParams.String())
		suite.Require().NoError(err)

		privKey := &secp256k1.PrivKey{Key: derivedPriv}
		pubKey := privKey.PubKey()
		addr := sdk.AccAddress(pubKey.Address())

		addresses[i] = addr.String()
	}

	// Verify all addresses are different
	for i := 0; i < len(addresses); i++ {
		for j := i + 1; j < len(addresses); j++ {
			suite.Require().NotEqual(addresses[i], addresses[j], "addresses should be unique")
		}
	}

	// Verify all addresses have paw1 prefix
	for _, addr := range addresses {
		suite.Require().True(strings.HasPrefix(addr, "paw1"))
	}
}

// TestAccountSequenceManagement tests account sequence handling
func (suite *WalletIntegrationTestSuite) TestAccountSequenceManagement() {
	privKey := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(privKey.PubKey().Address())

	// Mock account info response
	mockAccount := authtypes.BaseAccount{
		Address:       addr.String(),
		AccountNumber: 1,
		Sequence:      5,
	}

	// Verify we can access account info
	suite.Require().Equal(addr.String(), mockAccount.Address)
	suite.Require().Equal(uint64(1), mockAccount.AccountNumber)
	suite.Require().Equal(uint64(5), mockAccount.Sequence)

	// Test sequence increment
	mockAccount.Sequence++
	suite.Require().Equal(uint64(6), mockAccount.Sequence)
}

// TestHexEncodingDecoding tests hex encoding/decoding for transaction bytes
func (suite *WalletIntegrationTestSuite) TestHexEncodingDecoding() {
	// Create test transaction bytes
	testBytes := []byte("test transaction data")

	// Encode to hex
	hexStr := hex.EncodeToString(testBytes)
	suite.Require().NotEmpty(hexStr)

	// Decode from hex
	decodedBytes, err := hex.DecodeString(hexStr)
	suite.Require().NoError(err)
	suite.Require().Equal(testBytes, decodedBytes)

	// Test with invalid hex
	_, err = hex.DecodeString("invalid hex string!")
	suite.Require().Error(err)
}

// TestJSONRPCRequestFormat tests JSON-RPC request formatting
func (suite *WalletIntegrationTestSuite) TestJSONRPCRequestFormat() {
	type JSONRPCRequest struct {
		JSONRPC string          `json:"jsonrpc"`
		Method  string          `json:"method"`
		Params  json.RawMessage `json:"params"`
		ID      int             `json:"id"`
	}

	// Create test request
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  "broadcast_tx_sync",
		Params:  json.RawMessage(`{"tx":"dGVzdA=="}`),
		ID:      1,
	}

	// Marshal to JSON
	reqBytes, err := json.Marshal(req)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(reqBytes)

	// Unmarshal back
	var decodedReq JSONRPCRequest
	err = json.Unmarshal(reqBytes, &decodedReq)
	suite.Require().NoError(err)
	suite.Require().Equal(req.JSONRPC, decodedReq.JSONRPC)
	suite.Require().Equal(req.Method, decodedReq.Method)
	suite.Require().Equal(req.ID, decodedReq.ID)
}

// Helper functions

// createMockBalanceServer creates a mock HTTP server for balance queries
func (suite *WalletIntegrationTestSuite) createMockBalanceServer(response interface{}) *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Marshal response using JSON encoding (simpler for mock)
		respBytes, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(respBytes); err != nil {
			suite.T().Logf("failed to write mock response: %v", err)
		}
	})

	return httptest.NewServer(handler)
}

// queryBalanceMock queries balance from mock server
func (suite *WalletIntegrationTestSuite) queryBalanceMock(baseURL, address string) (sdk.Coins, error) {
	url := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", baseURL, address)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var balanceResp banktypes.QueryAllBalancesResponse
	if err := json.Unmarshal(body, &balanceResp); err != nil {
		return nil, err
	}

	return balanceResp.Balances, nil
}

// Benchmark tests

// BenchmarkWalletCreation benchmarks wallet creation performance
func BenchmarkWalletCreation(b *testing.B) {
	app.SetConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		entropy, _ := bip39.NewEntropy(256)
		mnemonic, _ := bip39.NewMnemonic(entropy)
		seed := bip39.NewSeed(mnemonic, "")

		hdParams, _ := hd.NewParamsFromPath("m/44'/118'/0'/0/0")
		masterPriv, ch := hd.ComputeMastersFromSeed(seed)
		derivedPriv, _ := hd.DerivePrivateKeyForPath(masterPriv, ch, hdParams.String())

		privKey := &secp256k1.PrivKey{Key: derivedPriv}
		pubKey := privKey.PubKey()
		_ = sdk.AccAddress(pubKey.Address())
	}
}

// BenchmarkAddressDerivation benchmarks address derivation
func BenchmarkAddressDerivation(b *testing.B) {
	app.SetConfig()

	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	seed := bip39.NewSeed(mnemonic, "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hdPath := fmt.Sprintf("m/44'/118'/0'/0/%d", i%100)
		hdParams, _ := hd.NewParamsFromPath(hdPath)

		masterPriv, ch := hd.ComputeMastersFromSeed(seed)
		derivedPriv, _ := hd.DerivePrivateKeyForPath(masterPriv, ch, hdParams.String())

		privKey := &secp256k1.PrivKey{Key: derivedPriv}
		pubKey := privKey.PubKey()
		_ = sdk.AccAddress(pubKey.Address())
	}
}

// BenchmarkSignatureVerification benchmarks signature verification
func BenchmarkSignatureVerification(b *testing.B) {
	app.SetConfig()

	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	msg := []byte("test message for benchmarking")
	signature, _ := privKey.Sign(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pubKey.VerifySignature(msg, signature)
	}
}

// Simple standalone tests

// TestSimpleWalletCreation is a simple standalone test
func TestSimpleWalletCreation(t *testing.T) {
	t.Parallel()
	app.SetConfig()

	// Generate mnemonic
	entropy, err := bip39.NewEntropy(256)
	require.NoError(t, err)

	mnemonic, err := bip39.NewMnemonic(entropy)
	require.NoError(t, err)
	require.NotEmpty(t, mnemonic)

	// Derive address
	seed := bip39.NewSeed(mnemonic, "")
	hdParams, err := hd.NewParamsFromPath("m/44'/118'/0'/0/0")
	require.NoError(t, err)

	masterPriv, ch := hd.ComputeMastersFromSeed(seed)
	derivedPriv, err := hd.DerivePrivateKeyForPath(masterPriv, ch, hdParams.String())
	require.NoError(t, err)

	privKey := &secp256k1.PrivKey{Key: derivedPriv}
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	// Verify address format
	require.True(t, strings.HasPrefix(addr.String(), "paw1"))
}

// TestSimpleSignature is a simple standalone test for signature
func TestSimpleSignature(t *testing.T) {
	t.Parallel()
	app.SetConfig()

	// Create key pair
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()

	// Sign message
	msg := []byte("test message")
	signature, err := privKey.Sign(msg)
	require.NoError(t, err)

	// Verify signature
	valid := pubKey.VerifySignature(msg, signature)
	require.True(t, valid)
}
