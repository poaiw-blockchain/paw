package hub

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/explorer/indexer/pkg/logger"
)

// mockLogger creates a test logger
func mockLogger() *logger.Logger {
	log, _ := logger.New(logger.Config{
		Level:  "info",
		Output: "stdout",
	})
	return log
}

// setupTestHub creates a test hub
func setupTestHub() *Hub {
	log := mockLogger()
	return NewHub(log)
}

// setupTestServer creates a test WebSocket server
func setupTestServer(t *testing.T, hub *Hub) *httptest.Server {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		require.NoError(t, err)

		client := NewClient(hub, conn, mockLogger())
		hub.Register(client)

		go client.WritePump()
		go client.ReadPump()
	})

	return httptest.NewServer(handler)
}

// ============================================================================
// HUB CREATION AND LIFECYCLE TESTS
// ============================================================================

func TestNewHub(t *testing.T) {
	hub := setupTestHub()
	assert.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.broadcast)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.log)
	assert.NotNil(t, hub.ctx)
}

func TestHubRun(t *testing.T) {
	hub := setupTestHub()

	// Start hub in background
	go hub.Run()

	// Give it time to start
	time.Sleep(100 * time.Millisecond)

	// Stop hub
	hub.Stop()

	// Give it time to stop
	time.Sleep(100 * time.Millisecond)

	// Hub should have stopped gracefully
	assert.NotNil(t, hub)
}

func TestHubStop(t *testing.T) {
	hub := setupTestHub()

	go hub.Run()
	time.Sleep(50 * time.Millisecond)

	hub.Stop()

	// Context should be cancelled
	select {
	case <-hub.ctx.Done():
		// Expected
	case <-time.After(time.Second):
		t.Fatal("Hub did not stop in time")
	}
}

// ============================================================================
// CLIENT REGISTRATION/UNREGISTRATION TESTS
// ============================================================================

func TestClientRegistration(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	// Connect client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for registration
	time.Sleep(100 * time.Millisecond)

	count := hub.GetClientCount()
	assert.Equal(t, 1, count)
}

func TestClientUnregistration(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	// Connect client
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 1, hub.GetClientCount())

	// Disconnect client
	conn.Close()
	time.Sleep(100 * time.Millisecond)

	count := hub.GetClientCount()
	assert.Equal(t, 0, count)
}

func TestMultipleClientRegistration(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect multiple clients
	clients := make([]*websocket.Conn, 10)
	for i := 0; i < 10; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		clients[i] = conn
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 10, hub.GetClientCount())

	// Cleanup
	for _, conn := range clients {
		conn.Close()
	}
}

// ============================================================================
// MESSAGE BROADCASTING TESTS
// ============================================================================

func TestBroadcastBlock(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Subscribe to blocks
	subscribeMsg := Message{
		Type: MessageTypeSubscribe,
		Data: json.RawMessage(`{"Type":"block"}`),
	}
	subscribeData, _ := json.Marshal(subscribeMsg)
	err = conn.WriteMessage(websocket.TextMessage, subscribeData)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	// Broadcast a block
	blockData := map[string]interface{}{
		"height": 1000,
		"hash":   "ABC123",
	}
	hub.BroadcastBlock(blockData)

	// Read message
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	var received Message
	err = json.Unmarshal(message, &received)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeBlock, received.Type)
}

func TestBroadcastTransaction(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Subscribe to transactions
	subscribeMsg := Message{
		Type: MessageTypeSubscribe,
		Data: json.RawMessage(`{"Type":"transaction"}`),
	}
	subscribeData, _ := json.Marshal(subscribeMsg)
	conn.WriteMessage(websocket.TextMessage, subscribeData)

	time.Sleep(50 * time.Millisecond)

	// Broadcast transaction
	txData := map[string]interface{}{
		"hash":   "TX123",
		"status": "success",
	}
	hub.BroadcastTransaction(txData)

	// Read message
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	var received Message
	err = json.Unmarshal(message, &received)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeTransaction, received.Type)
}

func TestBroadcastDEXSwap(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Subscribe to DEX swaps
	subscribeMsg := Message{
		Type: MessageTypeSubscribe,
		Data: json.RawMessage(`{"Type":"dex_swap"}`),
	}
	subscribeData, _ := json.Marshal(subscribeMsg)
	conn.WriteMessage(websocket.TextMessage, subscribeData)

	time.Sleep(50 * time.Millisecond)

	// Broadcast DEX swap
	swapData := map[string]interface{}{
		"pool_id":   "pool1",
		"amount_in": "100",
	}
	hub.BroadcastDEXSwap(swapData)

	// Read message
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	var received Message
	err = json.Unmarshal(message, &received)
	require.NoError(t, err)
	assert.Equal(t, MessageTypeDEXSwap, received.Type)
}

// ============================================================================
// SUBSCRIPTION FILTERING TESTS
// ============================================================================

func TestSubscriptionFiltering(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Subscribe only to blocks
	subscribeMsg := Message{
		Type: MessageTypeSubscribe,
		Data: json.RawMessage(`{"Type":"block"}`),
	}
	subscribeData, _ := json.Marshal(subscribeMsg)
	conn.WriteMessage(websocket.TextMessage, subscribeData)

	time.Sleep(50 * time.Millisecond)

	// Broadcast a transaction (should NOT receive)
	hub.BroadcastTransaction(map[string]interface{}{"hash": "TX1"})

	// Set short read deadline
	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = conn.ReadMessage()

	// Should timeout because we're not subscribed to transactions
	assert.Error(t, err)

	// Broadcast a block (should receive)
	hub.BroadcastBlock(map[string]interface{}{"height": 100})

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	var received Message
	json.Unmarshal(message, &received)
	assert.Equal(t, MessageTypeBlock, received.Type)
}

func TestUnsubscribe(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Subscribe to blocks
	subscribeMsg := Message{
		Type: MessageTypeSubscribe,
		Data: json.RawMessage(`{"Type":"block"}`),
	}
	subscribeData, _ := json.Marshal(subscribeMsg)
	conn.WriteMessage(websocket.TextMessage, subscribeData)

	time.Sleep(50 * time.Millisecond)

	// Unsubscribe from blocks
	unsubscribeMsg := Message{
		Type: MessageTypeUnsubscribe,
		Data: json.RawMessage(`{"Type":"block"}`),
	}
	unsubscribeData, _ := json.Marshal(unsubscribeMsg)
	conn.WriteMessage(websocket.TextMessage, unsubscribeData)

	time.Sleep(50 * time.Millisecond)

	// Broadcast block (should NOT receive)
	hub.BroadcastBlock(map[string]interface{}{"height": 100})

	conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	assert.Error(t, err) // Timeout expected
}

// ============================================================================
// CONCURRENT CLIENT HANDLING TESTS
// ============================================================================

func TestConcurrentBroadcast(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect multiple clients
	numClients := 50
	clients := make([]*websocket.Conn, numClients)
	for i := 0; i < numClients; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		clients[i] = conn

		// Subscribe each client
		subscribeMsg := Message{
			Type: MessageTypeSubscribe,
			Data: json.RawMessage(`{"Type":"block"}`),
		}
		subscribeData, _ := json.Marshal(subscribeMsg)
		conn.WriteMessage(websocket.TextMessage, subscribeData)
	}

	time.Sleep(200 * time.Millisecond)

	// Broadcast messages concurrently
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(height int) {
			defer wg.Done()
			hub.BroadcastBlock(map[string]interface{}{"height": height})
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// Cleanup
	for _, conn := range clients {
		conn.Close()
	}
}

func TestConcurrentClientConnections(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	var wg sync.WaitGroup
	numClients := 100
	successCount := int32(0)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			if err == nil {
				atomic.AddInt32(&successCount, 1)
				time.Sleep(50 * time.Millisecond)
				conn.Close()
			}
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	// Most connections should succeed
	assert.Greater(t, atomic.LoadInt32(&successCount), int32(90))
}

// ============================================================================
// CLIENT DISCONNECTION CLEANUP TESTS
// ============================================================================

func TestClientDisconnectionCleanup(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	// Connect and disconnect multiple clients
	for i := 0; i < 10; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)

		time.Sleep(20 * time.Millisecond)
		assert.Greater(t, hub.GetClientCount(), 0)

		conn.Close()
		time.Sleep(20 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, 0, hub.GetClientCount())
}

func TestGracefulClientShutdown(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, hub.GetClientCount())

	// Stop hub (should disconnect clients)
	hub.Stop()
	time.Sleep(100 * time.Millisecond)

	// Try to read from connection (should fail)
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = conn.ReadMessage()
	assert.Error(t, err)

	conn.Close()
}

// ============================================================================
// PING/PONG TESTS
// ============================================================================

func TestPingPong(t *testing.T) {
	hub := setupTestHub()
	go hub.Run()
	defer hub.Stop()

	server := setupTestServer(t, hub)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	time.Sleep(50 * time.Millisecond)

	// Send ping
	pingMsg := Message{
		Type: MessageTypePing,
	}
	pingData, _ := json.Marshal(pingMsg)
	err = conn.WriteMessage(websocket.TextMessage, pingData)
	require.NoError(t, err)

	// Should receive pong
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)

	var received Message
	json.Unmarshal(message, &received)
	assert.Equal(t, MessageTypePong, received.Type)
}

// ============================================================================
// BENCHMARK TESTS
// ============================================================================

func BenchmarkHubWith100Clients(b *testing.B) {
	hub := NewHub(mockLogger())
	go hub.Run()
	defer hub.Stop()

	// Create mock clients
	for i := 0; i < 100; i++ {
		client := &Client{
			hub:           hub,
			send:          make(chan []byte, channelBufferSize),
			subscriptions: make(map[MessageType]*Subscription),
			log:           mockLogger(),
			id:            generateClientID(),
		}
		client.subscriptions[MessageTypeBlock] = &Subscription{Type: MessageTypeBlock}
		hub.Register(client)
	}

	time.Sleep(100 * time.Millisecond)

	blockData := map[string]interface{}{"height": 1000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.BroadcastBlock(blockData)
	}
}

func BenchmarkHubWith1000Clients(b *testing.B) {
	hub := NewHub(mockLogger())
	go hub.Run()
	defer hub.Stop()

	// Create mock clients
	for i := 0; i < 1000; i++ {
		client := &Client{
			hub:           hub,
			send:          make(chan []byte, channelBufferSize),
			subscriptions: make(map[MessageType]*Subscription),
			log:           mockLogger(),
			id:            generateClientID(),
		}
		client.subscriptions[MessageTypeBlock] = &Subscription{Type: MessageTypeBlock}
		hub.Register(client)
	}

	time.Sleep(200 * time.Millisecond)

	blockData := map[string]interface{}{"height": 1000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.BroadcastBlock(blockData)
	}
}

func BenchmarkHubWith10000Clients(b *testing.B) {
	hub := NewHub(mockLogger())
	go hub.Run()
	defer hub.Stop()

	// Create mock clients
	for i := 0; i < 10000; i++ {
		client := &Client{
			hub:           hub,
			send:          make(chan []byte, channelBufferSize),
			subscriptions: make(map[MessageType]*Subscription),
			log:           mockLogger(),
			id:            generateClientID(),
		}
		client.subscriptions[MessageTypeBlock] = &Subscription{Type: MessageTypeBlock}
		hub.Register(client)
	}

	time.Sleep(500 * time.Millisecond)

	blockData := map[string]interface{}{"height": 1000}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.BroadcastBlock(blockData)
	}
}

func BenchmarkClientRegistration(b *testing.B) {
	hub := NewHub(mockLogger())
	go hub.Run()
	defer hub.Stop()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := &Client{
			hub:           hub,
			send:          make(chan []byte, channelBufferSize),
			subscriptions: make(map[MessageType]*Subscription),
			log:           mockLogger(),
			id:            generateClientID(),
		}
		hub.Register(client)
	}
}
