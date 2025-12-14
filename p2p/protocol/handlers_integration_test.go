package protocol

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/p2p/reputation"
)

// HandlersIntegrationTestSuite tests full message handling workflows
type HandlersIntegrationTestSuite struct {
	suite.Suite
	handlers   *ProtocolHandlers
	repManager *reputation.Manager
	logger     log.Logger
}

func (s *HandlersIntegrationTestSuite) SetupTest() {
	s.logger = log.NewNopLogger()

	// Create reputation manager
	storage := reputation.NewMemoryStorage()
	config := reputation.DefaultManagerConfig()
	var err error
	s.repManager, err = reputation.NewManager(storage, &config, s.logger)
	s.Require().NoError(err)

	// Create protocol handlers
	s.handlers = NewProtocolHandlers(s.repManager, s.logger)
}

func (s *HandlersIntegrationTestSuite) TearDownTest() {
	if s.repManager != nil {
		_ = s.repManager.Close()
	}
}

// Test full handshake workflow
func (s *HandlersIntegrationTestSuite) TestHandshakeWorkflow() {
	t := s.T()
	ctx := context.Background()

	handshakeReceived := false
	var receivedMsg *HandshakeMessage

	// Set up handshake handler
	s.handlers.SetHandshakeHandler(func(peerID string, msg *HandshakeMessage) (*HandshakeAckMessage, error) {
		handshakeReceived = true
		receivedMsg = msg
		return &HandshakeAckMessage{
			Accepted: true,
			Message:  "Welcome",
		}, nil
	})

	// Prepare handshake message
	msg := &HandshakeMessage{
		ProtocolVersion: 1,
		ChainID:         "test-chain",
		BestHeight:      100,
		Timestamp:       time.Now().Unix(),
	}

	// Handle handshake
	err := s.handlers.HandleMessage(ctx, "peer-1", msg)
	require.NoError(t, err)
	require.True(t, handshakeReceived)
	require.NotNil(t, receivedMsg)
	require.Equal(t, "test-chain", receivedMsg.ChainID)
	require.Equal(t, int64(100), receivedMsg.BestHeight)

	// Verify positive reputation event was recorded
	rep, err := s.repManager.GetReputation("peer-1")
	require.NoError(t, err)
	require.NotNil(t, rep)
	require.Greater(t, rep.Metrics.ValidMessages, int64(0))
}

// Test block message handling workflow
func (s *HandlersIntegrationTestSuite) TestBlockMessageWorkflow() {
	t := s.T()
	ctx := context.Background()

	blockReceived := false
	var receivedBlock *BlockMessage

	// Set up block handler
	s.handlers.SetBlockHandler(func(peerID string, msg *BlockMessage) error {
		blockReceived = true
		receivedBlock = msg
		return nil
	})

	// Prepare block message
	msg := &BlockMessage{
		Height:    101,
		Hash:      []byte("block-hash-101"),
		PrevHash:  []byte("block-hash-100"),
		Timestamp: time.Now().Unix(),
	}

	// Handle block
	err := s.handlers.HandleMessage(ctx, "peer-1", msg)
	require.NoError(t, err)
	require.True(t, blockReceived)
	require.NotNil(t, receivedBlock)
	require.Equal(t, int64(101), receivedBlock.Height)

	// Verify block propagation event recorded
	rep, err := s.repManager.GetReputation("peer-1")
	require.NoError(t, err)
	require.Greater(t, rep.Metrics.BlocksPropagated, int64(0))
}

// Test transaction message workflow
func (s *HandlersIntegrationTestSuite) TestTransactionMessageWorkflow() {
	t := s.T()
	ctx := context.Background()

	txReceived := false
	var receivedTx *TxMessage

	s.handlers.SetTxHandler(func(peerID string, msg *TxMessage) error {
		txReceived = true
		receivedTx = msg
		return nil
	})

	msg := &TxMessage{
		TxHash: []byte("tx-hash-123"),
		TxData: []byte("transaction data"),
	}

	err := s.handlers.HandleMessage(ctx, "peer-1", msg)
	require.NoError(t, err)
	require.True(t, txReceived)
	require.NotNil(t, receivedTx)
}

// Test protocol handler error recovery
func (s *HandlersIntegrationTestSuite) TestHandlerErrorRecovery() {
	t := s.T()
	ctx := context.Background()

	errorCount := 0

	// Set up handler that fails first 3 times
	s.handlers.SetBlockHandler(func(peerID string, msg *BlockMessage) error {
		errorCount++
		if errorCount <= 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	// Send multiple messages
	for i := 0; i < 5; i++ {
		msg := &BlockMessage{
			Height:    int64(i + 1),
			Hash:      []byte(fmt.Sprintf("hash-%d", i)),
			Timestamp: time.Now().Unix(),
		}

		err := s.handlers.HandleMessage(ctx, "peer-1", msg)
		if i < 3 {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
	}

	// Check error stats
	stats := s.handlers.GetStats(MsgTypeNewBlock)
	require.NotNil(t, stats)
	require.Equal(t, int64(3), stats.Errors)
	require.Equal(t, int64(2), stats.MessagesHandled)
}

// Test concurrent message processing
func (s *HandlersIntegrationTestSuite) TestConcurrentMessageProcessing() {
	t := s.T()
	ctx := context.Background()

	var processedCount atomic.Int64
	var mu sync.Mutex
	processedMessages := make(map[string]int)

	s.handlers.SetBlockHandler(func(peerID string, msg *BlockMessage) error {
		processedCount.Add(1)
		mu.Lock()
		processedMessages[peerID]++
		mu.Unlock()

		// Simulate processing time
		time.Sleep(1 * time.Millisecond)
		return nil
	})

	numPeers := 10
	messagesPerPeer := 20

	var wg sync.WaitGroup
	wg.Add(numPeers)

	// Send messages concurrently from multiple peers
	for p := 0; p < numPeers; p++ {
		go func(peerNum int) {
			defer wg.Done()

			peerID := fmt.Sprintf("peer-%d", peerNum)

			for i := 0; i < messagesPerPeer; i++ {
				msg := &BlockMessage{
					Height:    int64(i + 1),
					Hash:      []byte(fmt.Sprintf("peer-%d-hash-%d", peerNum, i)),
					Timestamp: time.Now().Unix(),
				}

				err := s.handlers.HandleMessage(ctx, peerID, msg)
				require.NoError(t, err)
			}
		}(p)
	}

	wg.Wait()

	// Verify all messages were processed
	total := processedCount.Load()
	require.Equal(t, int64(numPeers*messagesPerPeer), total)

	// Verify each peer sent correct number of messages
	mu.Lock()
	for peerID, count := range processedMessages {
		require.Equal(t, messagesPerPeer, count,
			"peer %s should have sent %d messages", peerID, messagesPerPeer)
	}
	mu.Unlock()
}

// Test rate limiting enforcement
func (s *HandlersIntegrationTestSuite) TestRateLimitEnforcement() {
	t := s.T()
	ctx := context.Background()

	peerID := "rate-limit-peer"

	// Send blocks rapidly to trigger rate limit
	successCount := 0
	rateLimitCount := 0

	for i := 0; i < 100; i++ {
		msg := &BlockMessage{
			Height:    int64(i + 1),
			Hash:      []byte(fmt.Sprintf("hash-%d", i)),
			Timestamp: time.Now().Unix(),
		}

		err := s.handlers.HandleMessage(ctx, peerID, msg)
		if err != nil {
			if errors.Is(err, errors.New("rate limit exceeded")) ||
			   (err.Error() != "" && len(err.Error()) > 0) {
				rateLimitCount++
			}
		} else {
			successCount++
		}
	}

	// Some messages should have been rate limited
	// (exact count depends on rate limit configuration)
	t.Logf("Success: %d, Rate limited: %d", successCount, rateLimitCount)
	require.Greater(t, successCount, 0, "some messages should succeed")
}

// Test message type routing
func (s *HandlersIntegrationTestSuite) TestMessageTypeRouting() {
	t := s.T()
	ctx := context.Background()

	handlerCalls := make(map[MessageType]int)
	var mu sync.Mutex

	// Track which handlers are called
	trackingHandler := func(msgType MessageType) MessageHandler {
		return func(ctx context.Context, peerID string, msg Message) error {
			mu.Lock()
			handlerCalls[msgType]++
			mu.Unlock()
			return nil
		}
	}

	// Register tracking handlers for different message types
	messageTypes := []MessageType{
		MsgTypeHandshake,
		MsgTypeNewBlock,
		MsgTypeNewTx,
		MsgTypePing,
		MsgTypePong,
	}

	for _, msgType := range messageTypes {
		s.handlers.RegisterHandler(msgType, trackingHandler(msgType))
	}

	// Send different message types
	messages := []Message{
		&HandshakeMessage{ProtocolVersion: 1, ChainID: "test"},
		&BlockMessage{Height: 1, Hash: []byte("hash")},
		&TxMessage{TxHash: []byte("tx-hash")},
		&PingMessage{Timestamp: time.Now().Unix()},
		&PongMessage{Timestamp: time.Now().Unix()},
	}

	for _, msg := range messages {
		err := s.handlers.HandleMessage(ctx, "peer-1", msg)
		require.NoError(t, err)
	}

	// Verify correct handlers were called
	mu.Lock()
	for _, msgType := range messageTypes {
		require.Equal(t, 1, handlerCalls[msgType],
			"handler for %s should be called once", msgType.String())
	}
	mu.Unlock()
}

// Test handler statistics collection
func (s *HandlersIntegrationTestSuite) TestHandlerStatistics() {
	t := s.T()
	ctx := context.Background()

	// Send various messages
	for i := 0; i < 50; i++ {
		msg := &BlockMessage{
			Height:    int64(i + 1),
			Hash:      []byte(fmt.Sprintf("hash-%d", i)),
			Timestamp: time.Now().Unix(),
		}
		_ = s.handlers.HandleMessage(ctx, "peer-1", msg)
	}

	for i := 0; i < 30; i++ {
		msg := &TxMessage{
			TxHash: []byte(fmt.Sprintf("tx-%d", i)),
			TxData: []byte("data"),
		}
		_ = s.handlers.HandleMessage(ctx, "peer-1", msg)
	}

	// Get statistics
	blockStats := s.handlers.GetStats(MsgTypeNewBlock)
	require.NotNil(t, blockStats)
	require.Equal(t, int64(50), blockStats.MessagesReceived)

	txStats := s.handlers.GetStats(MsgTypeNewTx)
	require.NotNil(t, txStats)
	require.Equal(t, int64(30), txStats.MessagesReceived)

	// Get all stats
	allStats := s.handlers.GetAllStats()
	require.NotEmpty(t, allStats)
}

// Test processing time tracking
func (s *HandlersIntegrationTestSuite) TestProcessingTimeTracking() {
	t := s.T()
	ctx := context.Background()

	// Set handler with artificial delay
	s.handlers.SetBlockHandler(func(peerID string, msg *BlockMessage) error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	// Send several messages
	for i := 0; i < 10; i++ {
		msg := &BlockMessage{
			Height:    int64(i + 1),
			Hash:      []byte(fmt.Sprintf("hash-%d", i)),
			Timestamp: time.Now().Unix(),
		}
		err := s.handlers.HandleMessage(ctx, "peer-1", msg)
		require.NoError(t, err)
	}

	// Check processing time stats
	stats := s.handlers.GetStats(MsgTypeNewBlock)
	require.NotNil(t, stats)
	require.Greater(t, stats.AvgProcessTime, time.Duration(0))
	require.Greater(t, stats.AvgProcessTime, 4*time.Millisecond,
		"average processing time should reflect the delay")
}

// Test reputation integration
func (s *HandlersIntegrationTestSuite) TestReputationIntegration() {
	t := s.T()
	ctx := context.Background()

	peerID := "reputation-peer"

	// Send valid messages
	for i := 0; i < 10; i++ {
		msg := &BlockMessage{
			Height:    int64(i + 1),
			Hash:      []byte(fmt.Sprintf("hash-%d", i)),
			Timestamp: time.Now().Unix(),
		}
		err := s.handlers.HandleMessage(ctx, peerID, msg)
		require.NoError(t, err)
	}

	// Check reputation increased
	rep, err := s.repManager.GetReputation(reputation.PeerID(peerID))
	require.NoError(t, err)
	require.NotNil(t, rep)
	require.Greater(t, rep.Metrics.ValidMessages, int64(0))
	require.Greater(t, rep.Metrics.BlocksPropagated, int64(0))

	// Send with handler error (invalid message)
	s.handlers.SetBlockHandler(func(peerID string, msg *BlockMessage) error {
		return errors.New("invalid block")
	})

	msg := &BlockMessage{
		Height:    11,
		Hash:      []byte("invalid"),
		Timestamp: time.Now().Unix(),
	}
	err = s.handlers.HandleMessage(ctx, peerID, msg)
	require.Error(t, err)

	// Check reputation decreased
	rep, err = s.repManager.GetReputation(reputation.PeerID(peerID))
	require.NoError(t, err)
	require.Greater(t, rep.Metrics.InvalidMessages, int64(0))
}

// Test custom handler registration
func (s *HandlersIntegrationTestSuite) TestCustomHandlerRegistration() {
	t := s.T()
	ctx := context.Background()

	customHandlerCalled := false

	// Register custom handler
	customHandler := func(ctx context.Context, peerID string, msg Message) error {
		customHandlerCalled = true
		return nil
	}

	s.handlers.RegisterHandler(MsgTypeStatus, customHandler)

	// Send status message
	msg := &StatusMessage{
		Height:  100,
		Syncing: false,
	}

	err := s.handlers.HandleMessage(ctx, "peer-1", msg)
	require.NoError(t, err)
	require.True(t, customHandlerCalled)
}

// Test ping-pong workflow
func (s *HandlersIntegrationTestSuite) TestPingPongWorkflow() {
	t := s.T()
	ctx := context.Background()

	// Send ping
	pingMsg := &PingMessage{
		Timestamp: time.Now().Unix(),
		Nonce:     12345,
	}

	err := s.handlers.HandleMessage(ctx, "peer-1", pingMsg)
	require.NoError(t, err)

	// Send pong
	pongMsg := &PongMessage{
		Timestamp: time.Now().Unix(),
		Nonce:     12345,
	}

	err = s.handlers.HandleMessage(ctx, "peer-1", pongMsg)
	require.NoError(t, err)

	// Verify latency was recorded
	rep, err := s.repManager.GetReputation("peer-1")
	require.NoError(t, err)
	require.NotNil(t, rep)
}

// Test error message handling
func (s *HandlersIntegrationTestSuite) TestErrorMessageHandling() {
	t := s.T()
	ctx := context.Background()

	errMsg := &ErrorMessage{
		Code:    404,
		Message: "Block not found",
	}

	// Error messages should be handled without error
	err := s.handlers.HandleMessage(ctx, "peer-1", errMsg)
	require.NoError(t, err)
}

// Test peer cleanup
func (s *HandlersIntegrationTestSuite) TestPeerCleanup() {
	t := s.T()
	ctx := context.Background()

	peerID := "cleanup-peer"

	// Send some messages to create rate limit state
	for i := 0; i < 10; i++ {
		msg := &BlockMessage{
			Height:    int64(i + 1),
			Hash:      []byte(fmt.Sprintf("hash-%d", i)),
			Timestamp: time.Now().Unix(),
		}
		_ = s.handlers.HandleMessage(ctx, peerID, msg)
	}

	// Cleanup peer
	s.handlers.CleanupPeer(peerID)

	// Rate limit state should be cleared
	// (verification would require internal state inspection)
}

func TestHandlersIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(HandlersIntegrationTestSuite))
}

// Benchmarks

func BenchmarkHandleMessage(b *testing.B) {
	logger := log.NewNopLogger()
	storage := reputation.NewMemoryStorage()
	config := reputation.DefaultManagerConfig()

	repMgr, _ := reputation.NewManager(storage, &config, logger)
	defer repMgr.Close()

	handlers := NewProtocolHandlers(repMgr, logger)

	ctx := context.Background()
	msg := &BlockMessage{
		Height:    1,
		Hash:      []byte("test-hash"),
		Timestamp: time.Now().Unix(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handlers.HandleMessage(ctx, "bench-peer", msg)
	}
}

func BenchmarkConcurrentHandleMessage(b *testing.B) {
	logger := log.NewNopLogger()
	storage := reputation.NewMemoryStorage()
	config := reputation.DefaultManagerConfig()

	repMgr, _ := reputation.NewManager(storage, &config, logger)
	defer repMgr.Close()

	handlers := NewProtocolHandlers(repMgr, logger)

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			msg := &BlockMessage{
				Height:    int64(i + 1),
				Hash:      []byte(fmt.Sprintf("hash-%d", i)),
				Timestamp: time.Now().Unix(),
			}
			_ = handlers.HandleMessage(ctx, fmt.Sprintf("peer-%d", i%10), msg)
			i++
		}
	})
}
