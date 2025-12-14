package test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/explorer/indexer/internal/database"
)

// ============================================================================
// MOCK DATA GENERATORS
// ============================================================================

// GenerateMockBlock creates a test block with realistic data
func GenerateMockBlock(height int64) database.Block {
	return database.Block{
		Height:          height,
		Hash:            RandomHash(),
		ProposerAddress: RandomValidatorAddress(),
		Time:            time.Now().UTC().Add(-time.Duration(height) * 6 * time.Second),
		TxCount:         rand.Intn(20),
		GasUsed:         int64(rand.Intn(1000000)),
		GasWanted:       int64(rand.Intn(1200000)),
		EvidenceCount:   0,
	}
}

// GenerateMockTransaction creates a test transaction
func GenerateMockTransaction(blockHeight int64, txIndex int) database.Transaction {
	return database.Transaction{
		Hash:        RandomTxHash(),
		BlockHeight: blockHeight,
		TxIndex:     txIndex,
		Type:        "/cosmos.bank.v1beta1.MsgSend",
		Sender:      RandomAddress(),
		Status:      "success",
		Code:        0,
		GasUsed:     int64(rand.Intn(100000)),
		GasWanted:   int64(rand.Intn(120000)),
		FeeAmount:   fmt.Sprintf("%d", rand.Intn(10000)),
		FeeDenom:    "upaw",
		Memo:        "test transaction",
		RawLog:      "[]",
		Time:        time.Now().UTC(),
		Messages:    json.RawMessage(`[{"@type":"/cosmos.bank.v1beta1.MsgSend"}]`),
		Events:      json.RawMessage(`[{"type":"transfer","attributes":[]}]`),
	}
}

// GenerateMockDEXPool creates a test DEX pool
func GenerateMockDEXPool(poolID string) database.DEXPool {
	return database.DEXPool{
		PoolID:        poolID,
		TokenA:        "upaw",
		TokenB:        "uusd",
		ReserveA:      float64(rand.Intn(1000000)),
		ReserveB:      float64(rand.Intn(1000000)),
		LPTokenSupply: float64(rand.Intn(100000)),
		SwapFeeRate:   0.003,
		TVL:           float64(rand.Intn(2000000)),
		CreatedHeight: int64(rand.Intn(10000)),
	}
}

// GenerateMockDEXSwap creates a test DEX swap
func GenerateMockDEXSwap(poolID string) database.DEXSwap {
	return database.DEXSwap{
		TxHash:    RandomTxHash(),
		PoolID:    poolID,
		Sender:    RandomAddress(),
		TokenIn:   "upaw",
		TokenOut:  "uusd",
		AmountIn:  float64(rand.Intn(10000)),
		AmountOut: float64(rand.Intn(10000)),
		Price:     1.0 + rand.Float64()*0.1,
		Fee:       float64(rand.Intn(100)),
		Time:      time.Now().UTC(),
	}
}

// GenerateMockValidator creates a test validator
func GenerateMockValidator(address string) database.Validator {
	return database.Validator{
		Address:                 address,
		OperatorAddress:         "pawvaloper" + address[3:],
		ConsensusPubkey:         RandomPubkey(),
		Moniker:                 "TestValidator",
		CommissionRate:          0.1,
		CommissionMaxRate:       0.2,
		CommissionMaxChangeRate: 0.01,
		VotingPower:             int64(rand.Intn(1000000)),
		Jailed:                  false,
		Status:                  "BOND_STATUS_BONDED",
		Tokens:                  int64(rand.Intn(10000000)),
		DelegatorShares:         float64(rand.Intn(10000000)),
	}
}

// ============================================================================
// RANDOM DATA GENERATORS
// ============================================================================

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func init() {
	rand.Seed(time.Now().UnixNano())
}

// RandomString generates a random string of specified length
func RandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// RandomHash generates a random block hash
func RandomHash() string {
	return RandomString(64)
}

// RandomTxHash generates a random transaction hash
func RandomTxHash() string {
	return RandomString(64)
}

// RandomAddress generates a random PAW address
func RandomAddress() string {
	return "paw1" + RandomString(38)
}

// RandomValidatorAddress generates a random validator address
func RandomValidatorAddress() string {
	return "pawvaloper1" + RandomString(38)
}

// RandomPubkey generates a random consensus pubkey
func RandomPubkey() string {
	return "pawvalconspub1" + RandomString(76)
}

// ============================================================================
// TEST DATABASE HELPERS
// ============================================================================

// TestDBConfig returns test database configuration
func TestDBConfig() database.Config {
	return database.Config{
		URL:            "postgres://postgres:postgres@localhost:5432/paw_explorer_test?sslmode=disable",
		MaxConnections: 10,
		MaxIdle:        5,
		ConnMaxLife:    time.Hour,
	}
}

// SetupTestDB creates and initializes a test database
func SetupTestDB(t *testing.T) *database.Database {
	db, err := database.New(TestDBConfig())
	require.NoError(t, err, "Failed to create test database")

	err = db.InitSchema()
	require.NoError(t, err, "Failed to initialize schema")

	CleanTestDB(t, db)
	return db
}

// CleanTestDB truncates all tables
func CleanTestDB(t *testing.T, db *database.Database) {
	tables := []string{
		"dex_trades", "dex_liquidity", "dex_pools", "dex_swaps",
		"dex_pool_price_history", "dex_pool_statistics", "dex_user_positions",
		"dex_analytics_cache", "oracle_prices", "compute_requests",
		"validator_uptime", "validators", "events", "transactions",
		"blocks", "accounts", "failed_blocks", "indexing_checkpoints",
		"indexing_metrics", "indexing_progress", "indexer_state",
	}

	for _, table := range tables {
		_, err := db.Exec("TRUNCATE TABLE " + table + " CASCADE")
		if err != nil {
			t.Logf("Warning: failed to truncate table %s: %v", table, err)
		}
	}
}

// TeardownTestDB closes database connection
func TeardownTestDB(t *testing.T, db *database.Database) {
	err := db.Close()
	require.NoError(t, err, "Failed to close database")
}

// SeedTestBlocks inserts test blocks into database
func SeedTestBlocks(t *testing.T, db *database.Database, count int) []database.Block {
	blocks := make([]database.Block, count)

	tx, err := db.BeginTx()
	require.NoError(t, err)

	for i := 0; i < count; i++ {
		block := GenerateMockBlock(int64(i + 1))
		blocks[i] = block
		err = db.InsertBlock(tx, block)
		require.NoError(t, err)
	}

	err = tx.Commit()
	require.NoError(t, err)

	return blocks
}

// SeedTestTransactions inserts test transactions
func SeedTestTransactions(t *testing.T, db *database.Database, blockHeight int64, count int) []database.Transaction {
	txs := make([]database.Transaction, count)

	tx, err := db.BeginTx()
	require.NoError(t, err)

	for i := 0; i < count; i++ {
		transaction := GenerateMockTransaction(blockHeight, i)
		txs[i] = transaction
		err = db.InsertTransaction(tx, transaction)
		require.NoError(t, err)
	}

	err = tx.Commit()
	require.NoError(t, err)

	return txs
}

// ============================================================================
// MOCK WEBSOCKET CLIENT
// ============================================================================

// MockWSClient represents a test WebSocket client
type MockWSClient struct {
	Conn     *websocket.Conn
	Messages [][]byte
	mu       sync.Mutex
}

// NewMockWSClient creates a mock WebSocket client
func NewMockWSClient(serverURL string) (*MockWSClient, error) {
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}

	client := &MockWSClient{
		Conn:     conn,
		Messages: make([][]byte, 0),
	}

	// Start message receiver
	go client.receiveMessages()

	return client, nil
}

// receiveMessages continuously receives messages
func (c *MockWSClient) receiveMessages() {
	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}

		c.mu.Lock()
		c.Messages = append(c.Messages, message)
		c.mu.Unlock()
	}
}

// Send sends a message to server
func (c *MockWSClient) Send(message []byte) error {
	return c.Conn.WriteMessage(websocket.TextMessage, message)
}

// GetMessages returns all received messages
func (c *MockWSClient) GetMessages() [][]byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	messages := make([][]byte, len(c.Messages))
	copy(messages, c.Messages)
	return messages
}

// Close closes the connection
func (c *MockWSClient) Close() error {
	return c.Conn.Close()
}

// ============================================================================
// MOCK RPC SERVER
// ============================================================================

// MockRPCServer represents a test RPC server
type MockRPCServer struct {
	Server       *httptest.Server
	BlockHeight  int64
	Blocks       map[int64]interface{}
	Transactions map[string]interface{}
}

// NewMockRPCServer creates a mock blockchain RPC server
func NewMockRPCServer() *MockRPCServer {
	mock := &MockRPCServer{
		BlockHeight:  100,
		Blocks:       make(map[int64]interface{}),
		Transactions: make(map[string]interface{}),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mock.handleRPCRequest(w, r)
	})

	mock.Server = httptest.NewServer(handler)
	return mock
}

// handleRPCRequest handles RPC requests
func (m *MockRPCServer) handleRPCRequest(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	method := req["method"].(string)

	switch method {
	case "status":
		m.handleStatus(w)
	case "block":
		m.handleBlock(w, req)
	case "tx":
		m.handleTx(w, req)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleStatus returns blockchain status
func (m *MockRPCServer) handleStatus(w http.ResponseWriter) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result": map[string]interface{}{
			"sync_info": map[string]interface{}{
				"latest_block_height": fmt.Sprintf("%d", m.BlockHeight),
			},
		},
	}
	json.NewEncoder(w).Encode(response)
}

// handleBlock returns block data
func (m *MockRPCServer) handleBlock(w http.ResponseWriter, req map[string]interface{}) {
	params := req["params"].(map[string]interface{})
	height := int64(params["height"].(float64))

	if block, ok := m.Blocks[height]; ok {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result": map[string]interface{}{
				"block": block,
			},
		}
		json.NewEncoder(w).Encode(response)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// handleTx returns transaction data
func (m *MockRPCServer) handleTx(w http.ResponseWriter, req map[string]interface{}) {
	params := req["params"].(map[string]interface{})
	hash := params["hash"].(string)

	if tx, ok := m.Transactions[hash]; ok {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      req["id"],
			"result": map[string]interface{}{
				"tx": tx,
			},
		}
		json.NewEncoder(w).Encode(response)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// AddBlock adds a block to mock server
func (m *MockRPCServer) AddBlock(height int64, block interface{}) {
	m.Blocks[height] = block
	if height > m.BlockHeight {
		m.BlockHeight = height
	}
}

// AddTransaction adds a transaction to mock server
func (m *MockRPCServer) AddTransaction(hash string, tx interface{}) {
	m.Transactions[hash] = tx
}

// Close closes the mock server
func (m *MockRPCServer) Close() {
	m.Server.Close()
}

// ============================================================================
// ASSERTION HELPERS
// ============================================================================

// AssertBlockEqual asserts two blocks are equal
func AssertBlockEqual(t *testing.T, expected, actual database.Block) {
	require.Equal(t, expected.Height, actual.Height, "Block height mismatch")
	require.Equal(t, expected.Hash, actual.Hash, "Block hash mismatch")
	require.Equal(t, expected.ProposerAddress, actual.ProposerAddress, "Proposer mismatch")
	require.Equal(t, expected.TxCount, actual.TxCount, "TX count mismatch")
}

// AssertTransactionEqual asserts two transactions are equal
func AssertTransactionEqual(t *testing.T, expected, actual database.Transaction) {
	require.Equal(t, expected.Hash, actual.Hash, "TX hash mismatch")
	require.Equal(t, expected.BlockHeight, actual.BlockHeight, "Block height mismatch")
	require.Equal(t, expected.Sender, actual.Sender, "Sender mismatch")
	require.Equal(t, expected.Status, actual.Status, "Status mismatch")
}

// AssertWithinDuration asserts times are within duration
func AssertWithinDuration(t *testing.T, expected, actual time.Time, delta time.Duration) {
	diff := actual.Sub(expected)
	if diff < 0 {
		diff = -diff
	}
	require.LessOrEqual(t, diff, delta, "Time difference exceeds delta")
}

// ============================================================================
// CONTEXT HELPERS
// ============================================================================

// TestContext returns a context with timeout
func TestContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// TestContextWithDeadline returns a context with deadline
func TestContextWithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), deadline)
}
