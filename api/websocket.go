package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development
		// In production, implement proper origin checking
		return true
	},
}

// WebSocketHub manages WebSocket connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan WSMessage
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
}

// WebSocketClient represents a WebSocket client
type WebSocketClient struct {
	hub          *WebSocketHub
	conn         *websocket.Conn
	send         chan WSMessage
	subscriptions map[string]bool
	mu           sync.RWMutex
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan WSMessage, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			fmt.Printf("WebSocket client connected. Total: %d\n", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			fmt.Printf("WebSocket client disconnected. Total: %d\n", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				// Check if client is subscribed to this channel
				if message.Channel != "" {
					client.mu.RLock()
					subscribed := client.subscriptions[message.Channel]
					client.mu.RUnlock()
					if !subscribed {
						continue
					}
				}

				select {
				case client.send <- message:
				default:
					// Client's send channel is full, close the connection
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client)
					close(client.send)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *WebSocketHub) Broadcast(message WSMessage) {
	select {
	case h.broadcast <- message:
	default:
		fmt.Println("Warning: Broadcast channel is full")
	}
}

// Close closes the WebSocket hub
func (h *WebSocketHub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		client.conn.Close()
		delete(h.clients, client)
	}

	close(h.broadcast)
	close(h.register)
	close(h.unregister)
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade error: %v\n", err)
		return
	}

	client := &WebSocketClient{
		hub:           s.wsHub,
		conn:          conn,
		send:          make(chan WSMessage, 256),
		subscriptions: make(map[string]bool),
	}

	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// readPump pumps messages from the WebSocket connection to the hub
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
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
				fmt.Printf("WebSocket error: %v\n", err)
			}
			break
		}

		// Parse message
		var msg WSSubscribeMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			fmt.Printf("Error parsing WebSocket message: %v\n", err)
			continue
		}

		// Handle subscription/unsubscription
		c.handleMessage(msg)
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *WebSocketClient) writePump() {
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

			// Send message as JSON
			if err := c.conn.WriteJSON(message); err != nil {
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

// handleMessage handles incoming WebSocket messages
func (c *WebSocketClient) handleMessage(msg WSSubscribeMessage) {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch msg.Type {
	case "subscribe":
		c.subscriptions[msg.Channel] = true
		fmt.Printf("Client subscribed to channel: %s\n", msg.Channel)

		// Send confirmation
		c.sendMessage(WSMessage{
			Type:    "subscribed",
			Channel: msg.Channel,
			Data: map[string]interface{}{
				"channel": msg.Channel,
				"status":  "subscribed",
			},
		})

	case "unsubscribe":
		delete(c.subscriptions, msg.Channel)
		fmt.Printf("Client unsubscribed from channel: %s\n", msg.Channel)

		// Send confirmation
		c.sendMessage(WSMessage{
			Type:    "unsubscribed",
			Channel: msg.Channel,
			Data: map[string]interface{}{
				"channel": msg.Channel,
				"status":  "unsubscribed",
			},
		})

	default:
		fmt.Printf("Unknown message type: %s\n", msg.Type)
	}
}

// sendMessage sends a message to this specific client
func (c *WebSocketClient) sendMessage(msg WSMessage) {
	select {
	case c.send <- msg:
	default:
		fmt.Println("Warning: Client send channel is full")
	}
}

// GetConnectedClients returns the number of connected clients
func (h *WebSocketHub) GetConnectedClients() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// BroadcastToChannel sends a message to all clients subscribed to a specific channel
func (h *WebSocketHub) BroadcastToChannel(channel string, data interface{}) {
	message := WSMessage{
		Type:    channel + "_update",
		Channel: channel,
		Data:    data,
	}
	h.Broadcast(message)
}
