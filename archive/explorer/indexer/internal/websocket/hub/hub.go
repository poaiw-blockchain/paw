package hub

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/paw-chain/paw/explorer/indexer/pkg/logger"
)

var (
	wsConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_ws_connections_total",
		Help: "Total number of active WebSocket connections",
	})

	wsMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "explorer_ws_messages_total",
			Help: "Total number of WebSocket messages sent",
		},
		[]string{"type"},
	)

	wsErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_ws_errors_total",
		Help: "Total number of WebSocket errors",
	})
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512

	// Buffer size for channels
	channelBufferSize = 256
)

// MessageType represents the type of WebSocket message
type MessageType string

const (
	MessageTypeBlock       MessageType = "block"
	MessageTypeTransaction MessageType = "transaction"
	MessageTypeEvent       MessageType = "event"
	MessageTypeDEXSwap     MessageType = "dex_swap"
	MessageTypeOraclePrice MessageType = "oracle_price"
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypePing        MessageType = "ping"
	MessageTypePong        MessageType = "pong"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType     `json:"type"`
	Data      json.RawMessage `json:"data"`
	Timestamp time.Time       `json:"timestamp"`
}

// Subscription represents a client subscription
type Subscription struct {
	Type   MessageType
	Filter map[string]string // Optional filters (e.g., address, pool_id)
}

// Client represents a WebSocket client
type Client struct {
	hub           *Hub
	conn          *websocket.Conn
	send          chan []byte
	subscriptions map[MessageType]*Subscription
	mu            sync.RWMutex
	log           *logger.Logger
	id            string
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan *Message

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Logger
	log *logger.Logger

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// NewHub creates a new Hub
func NewHub(log *logger.Logger) *Hub {
	ctx, cancel := context.WithCancel(context.Background())
	return &Hub{
		broadcast:  make(chan *Message, channelBufferSize),
		register:   make(chan *Client, channelBufferSize),
		unregister: make(chan *Client, channelBufferSize),
		clients:    make(map[*Client]bool),
		log:        log,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Run starts the hub
func (h *Hub) Run() {
	h.log.Info("WebSocket hub starting")

	for {
		select {
		case <-h.ctx.Done():
			h.log.Info("WebSocket hub stopping")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			wsConnections.Inc()
			h.log.Info("Client registered", "client_id", client.id, "total_clients", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			wsConnections.Dec()
			h.log.Info("Client unregistered", "client_id", client.id, "total_clients", len(h.clients))

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// broadcastMessage sends a message to all subscribed clients
func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	messageBytes, err := json.Marshal(message)
	if err != nil {
		h.log.Error("Failed to marshal message", "error", err)
		wsErrors.Inc()
		return
	}

	successCount := 0
	for _, client := range clients {
		if client.isSubscribed(message.Type) {
			select {
			case client.send <- messageBytes:
				successCount++
			default:
				// Client's send buffer is full, skip
				h.log.Warn("Client send buffer full, skipping message", "client_id", client.id)
			}
		}
	}

	wsMessagesTotal.WithLabelValues(string(message.Type)).Add(float64(successCount))
}

// BroadcastBlock broadcasts a new block
func (h *Hub) BroadcastBlock(block interface{}) {
	data, err := json.Marshal(block)
	if err != nil {
		h.log.Error("Failed to marshal block", "error", err)
		return
	}

	message := &Message{
		Type:      MessageTypeBlock,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
		h.log.Warn("Broadcast channel full, dropping block message")
	}
}

// BroadcastTransaction broadcasts a new transaction
func (h *Hub) BroadcastTransaction(tx interface{}) {
	data, err := json.Marshal(tx)
	if err != nil {
		h.log.Error("Failed to marshal transaction", "error", err)
		return
	}

	message := &Message{
		Type:      MessageTypeTransaction,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
		h.log.Warn("Broadcast channel full, dropping transaction message")
	}
}

// BroadcastEvent broadcasts an event
func (h *Hub) BroadcastEvent(event interface{}) {
	data, err := json.Marshal(event)
	if err != nil {
		h.log.Error("Failed to marshal event", "error", err)
		return
	}

	message := &Message{
		Type:      MessageTypeEvent,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
		h.log.Warn("Broadcast channel full, dropping event message")
	}
}

// BroadcastDEXSwap broadcasts a DEX swap
func (h *Hub) BroadcastDEXSwap(swap interface{}) {
	data, err := json.Marshal(swap)
	if err != nil {
		h.log.Error("Failed to marshal DEX swap", "error", err)
		return
	}

	message := &Message{
		Type:      MessageTypeDEXSwap,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
		h.log.Warn("Broadcast channel full, dropping DEX swap message")
	}
}

// BroadcastOraclePrice broadcasts an oracle price update
func (h *Hub) BroadcastOraclePrice(price interface{}) {
	data, err := json.Marshal(price)
	if err != nil {
		h.log.Error("Failed to marshal oracle price", "error", err)
		return
	}

	message := &Message{
		Type:      MessageTypeOraclePrice,
		Data:      data,
		Timestamp: time.Now(),
	}

	select {
	case h.broadcast <- message:
	default:
		h.log.Warn("Broadcast channel full, dropping oracle price message")
	}
}

// Register registers a client with the hub
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister unregisters a client from the hub
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Stop stops the hub
func (h *Hub) Stop() {
	h.cancel()
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, log *logger.Logger) *Client {
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan []byte, channelBufferSize),
		subscriptions: make(map[MessageType]*Subscription),
		log:           log,
		id:            generateClientID(),
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.log.Error("WebSocket error", "error", err)
				wsErrors.Inc()
			}
			break
		}

		// Handle client messages (subscriptions, unsubscriptions, pings)
		c.handleMessage(message)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes messages from the client
func (c *Client) handleMessage(message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		c.log.Error("Failed to unmarshal client message", "error", err)
		return
	}

	switch msg.Type {
	case MessageTypeSubscribe:
		c.subscribe(&msg)
	case MessageTypeUnsubscribe:
		c.unsubscribe(&msg)
	case MessageTypePing:
		c.sendPong()
	}
}

// subscribe handles subscription requests
func (c *Client) subscribe(msg *Message) {
	var sub Subscription
	if err := json.Unmarshal(msg.Data, &sub); err != nil {
		c.log.Error("Failed to unmarshal subscription", "error", err)
		return
	}

	c.mu.Lock()
	c.subscriptions[sub.Type] = &sub
	c.mu.Unlock()

	c.log.Info("Client subscribed", "client_id", c.id, "type", sub.Type)
}

// unsubscribe handles unsubscription requests
func (c *Client) unsubscribe(msg *Message) {
	var sub Subscription
	if err := json.Unmarshal(msg.Data, &sub); err != nil {
		c.log.Error("Failed to unmarshal unsubscription", "error", err)
		return
	}

	c.mu.Lock()
	delete(c.subscriptions, sub.Type)
	c.mu.Unlock()

	c.log.Info("Client unsubscribed", "client_id", c.id, "type", sub.Type)
}

// sendPong sends a pong message to the client
func (c *Client) sendPong() {
	pongMsg := Message{
		Type:      MessageTypePong,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(pongMsg)
	if err != nil {
		return
	}

	select {
	case c.send <- data:
	default:
		// Send buffer full, skip
	}
}

// isSubscribed checks if the client is subscribed to a message type
func (c *Client) isSubscribed(msgType MessageType) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.subscriptions[msgType]
	return ok
}

// generateClientID generates a unique client ID
func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
