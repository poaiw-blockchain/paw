package integration

import (
	"encoding/json"
	"os"
	"path/filepath"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ContractInfo holds information about a deployed contract
type ContractInfo struct {
	CodeID   uint64
	Address  sdk.AccAddress
	Label    string
	Admin    sdk.AccAddress
	InitMsg  string
	WasmFile string
}

// ContractManager manages test contracts
type ContractManager struct {
	contracts     map[string]*ContractInfo
	codeIDCounter uint64
}

// NewContractManager creates a new contract manager
func NewContractManager() *ContractManager {
	return &ContractManager{
		contracts:     make(map[string]*ContractInfo),
		codeIDCounter: 1,
	}
}

// LoadWasmFile loads a wasm file from disk
func LoadWasmFile(path string) ([]byte, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	wasmCode, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}

	return wasmCode, nil
}

// StoreContract stores a contract and returns the code ID
func (m *ContractManager) StoreContract(label string, wasmPath string) (*ContractInfo, error) {
	wasmCode, err := LoadWasmFile(wasmPath)
	if err != nil {
		return nil, err
	}
	// wasmCode is loaded for validation but not used in mock implementation
	_ = wasmCode

	info := &ContractInfo{
		CodeID:   m.codeIDCounter,
		Label:    label,
		WasmFile: wasmPath,
	}

	m.codeIDCounter++
	m.contracts[label] = info

	return info, nil
}

// InstantiateContract instantiates a stored contract
func (m *ContractManager) InstantiateContract(
	label string,
	admin sdk.AccAddress,
	initMsg interface{},
) (*ContractInfo, error) {
	info, ok := m.contracts[label]
	if !ok {
		// If not stored yet, return a new contract info
		info = &ContractInfo{
			CodeID: m.codeIDCounter,
			Label:  label,
			Admin:  admin,
		}
		m.codeIDCounter++
	}

	// Set admin and init message
	info.Admin = admin

	initMsgBytes, err := json.Marshal(initMsg)
	if err != nil {
		return nil, err
	}
	info.InitMsg = string(initMsgBytes)

	// Generate a deterministic address based on code ID and label
	// In a real implementation, this would come from the blockchain
	info.Address = sdk.AccAddress([]byte(label))

	m.contracts[label] = info
	return info, nil
}

// GetContract retrieves contract info by label
func (m *ContractManager) GetContract(label string) (*ContractInfo, error) {
	info, ok := m.contracts[label]
	if !ok {
		return nil, wasmtypes.ErrNotFound
	}
	return info, nil
}

// Common contract initialization messages

// CW20InitMsg is the initialization message for CW20 tokens
type CW20InitMsg struct {
	Name            string         `json:"name"`
	Symbol          string         `json:"symbol"`
	Decimals        uint8          `json:"decimals"`
	InitialBalances []CW20Balance  `json:"initial_balances"`
	Mint            *CW20Minter    `json:"mint,omitempty"`
	Marketing       *CW20Marketing `json:"marketing,omitempty"`
}

// CW20Balance represents a balance in CW20 init
type CW20Balance struct {
	Address string `json:"address"`
	Amount  string `json:"amount"`
}

// CW20Minter represents minter configuration
type CW20Minter struct {
	Minter string  `json:"minter"`
	Cap    *string `json:"cap,omitempty"`
}

// CW20Marketing represents marketing info
type CW20Marketing struct {
	Project     string `json:"project,omitempty"`
	Description string `json:"description,omitempty"`
	Marketing   string `json:"marketing,omitempty"`
	Logo        string `json:"logo,omitempty"`
}

// NewCW20InitMsg creates a basic CW20 initialization message
func NewCW20InitMsg(
	name, symbol string,
	decimals uint8,
	balances []CW20Balance,
) CW20InitMsg {
	return CW20InitMsg{
		Name:            name,
		Symbol:          symbol,
		Decimals:        decimals,
		InitialBalances: balances,
	}
}

// CW721InitMsg is the initialization message for CW721 NFTs
type CW721InitMsg struct {
	Name   string `json:"name"`
	Symbol string `json:"symbol"`
	Minter string `json:"minter"`
}

// NewCW721InitMsg creates a basic CW721 initialization message
func NewCW721InitMsg(name, symbol, minter string) CW721InitMsg {
	return CW721InitMsg{
		Name:   name,
		Symbol: symbol,
		Minter: minter,
	}
}

// AMM Pool Contract Messages

// AMMPoolInitMsg initializes an AMM pool
type AMMPoolInitMsg struct {
	TokenADenom string `json:"token_a_denom"`
	TokenBDenom string `json:"token_b_denom"`
	FeePercent  string `json:"fee_percent"`
	Admin       string `json:"admin,omitempty"`
}

// NewAMMPoolInitMsg creates an AMM pool initialization message
func NewAMMPoolInitMsg(tokenA, tokenB, feePercent string) AMMPoolInitMsg {
	return AMMPoolInitMsg{
		TokenADenom: tokenA,
		TokenBDenom: tokenB,
		FeePercent:  feePercent,
	}
}

// AMMSwapMsg is a swap execution message
type AMMSwapMsg struct {
	Swap *SwapParams `json:"swap"`
}

// SwapParams defines swap parameters
type SwapParams struct {
	OfferAsset string `json:"offer_asset"`
	AskAsset   string `json:"ask_asset"`
	Amount     string `json:"amount"`
	MinReceive string `json:"min_receive,omitempty"`
}

// NewAMMSwapMsg creates a swap message
func NewAMMSwapMsg(offerAsset, askAsset, amount string) AMMSwapMsg {
	return AMMSwapMsg{
		Swap: &SwapParams{
			OfferAsset: offerAsset,
			AskAsset:   askAsset,
			Amount:     amount,
		},
	}
}

// AMMAddLiquidityMsg adds liquidity to a pool
type AMMAddLiquidityMsg struct {
	AddLiquidity *AddLiquidityParams `json:"add_liquidity"`
}

// AddLiquidityParams defines liquidity addition parameters
type AddLiquidityParams struct {
	TokenAAmount string `json:"token_a_amount"`
	TokenBAmount string `json:"token_b_amount"`
	MinLPTokens  string `json:"min_lp_tokens,omitempty"`
}

// NewAMMAddLiquidityMsg creates an add liquidity message
func NewAMMAddLiquidityMsg(amountA, amountB string) AMMAddLiquidityMsg {
	return AMMAddLiquidityMsg{
		AddLiquidity: &AddLiquidityParams{
			TokenAAmount: amountA,
			TokenBAmount: amountB,
		},
	}
}

// Common contract query messages

// BalanceQuery queries token balance
type BalanceQuery struct {
	Balance *BalanceParams `json:"balance"`
}

// BalanceParams defines balance query parameters
type BalanceParams struct {
	Address string `json:"address"`
}

// NewBalanceQuery creates a balance query
func NewBalanceQuery(address string) BalanceQuery {
	return BalanceQuery{
		Balance: &BalanceParams{
			Address: address,
		},
	}
}

// PoolInfoQuery queries pool information
type PoolInfoQuery struct {
	PoolInfo *struct{} `json:"pool_info"`
}

// NewPoolInfoQuery creates a pool info query
func NewPoolInfoQuery() PoolInfoQuery {
	return PoolInfoQuery{
		PoolInfo: &struct{}{},
	}
}

// ContractExecuteHelper helps build contract execute messages
type ContractExecuteHelper struct {
	Contract sdk.AccAddress
	Sender   sdk.AccAddress
	Funds    sdk.Coins
}

// NewContractExecuteHelper creates a new execute helper
func NewContractExecuteHelper(contract, sender sdk.AccAddress, funds sdk.Coins) *ContractExecuteHelper {
	return &ContractExecuteHelper{
		Contract: contract,
		Sender:   sender,
		Funds:    funds,
	}
}

// ExecuteMsg builds an execute message
func (h *ContractExecuteHelper) ExecuteMsg(msg interface{}) ([]byte, error) {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return msgBytes, nil
}
