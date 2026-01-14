package websocket

import (
	"fmt"
	"log"
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
		return true // Allow all origins (configure properly in production)
	},
}

// Message represents a WebSocket message
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// Client represents a WebSocket client
type Client struct {
	conn     *websocket.Conn
	send     chan Message
	email    string
	role     string
	channels []string
}

// Server manages WebSocket connections
type Server struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewServer creates a new WebSocket server
func NewServer() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Start starts the WebSocket server
func (s *Server) Start(port int) {
	go s.run()
	log.Printf("WebSocket server started on port %d", port)
}

// run handles client registration, unregistration, and broadcasting
func (s *Server) run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("Client connected: %s (%s)", client.email, client.role)

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
				log.Printf("Client disconnected: %s", client.email)
			}
			s.mu.Unlock()

		case message := <-s.broadcast:
			message.Timestamp = time.Now()
			s.mu.RLock()
			for client := range s.clients {
				select {
				case client.send <- message:
				default:
					// Client's send channel is full, skip
				}
			}
			s.mu.RUnlock()
		}
	}
}

// HandleConnection handles new WebSocket connections
func (s *Server) HandleConnection(c *gin.Context) {
	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Get user info from context (set by auth middleware)
	email, _ := c.Get("user_email")
	role, _ := c.Get("user_role")

	// Create client
	client := &Client{
		conn:     conn,
		send:     make(chan Message, 256),
		email:    email.(string),
		role:     fmt.Sprintf("%v", role),
		channels: []string{"all"}, // Default to all channels
	}

	// Register client
	s.register <- client

	// Start goroutines for reading and writing
	go client.readPump(s)
	go client.writePump()
}

// Broadcast sends a message to all connected clients
func (s *Server) Broadcast(message Message) {
	s.broadcast <- message
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump(s *Server) {
	defer func() {
		s.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle client messages (e.g., subscribe to channels)
		if msg.Type == "subscribe" {
			if channels, ok := msg.Data.([]interface{}); ok {
				c.channels = make([]string, len(channels))
				for i, ch := range channels {
					c.channels[i] = ch.(string)
				}
				log.Printf("Client %s subscribed to channels: %v", c.email, c.channels)
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Send message as JSON
			err := c.conn.WriteJSON(message)
			if err != nil {
				return
			}

		case <-ticker.C:
			// Send ping to keep connection alive
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// GetStats returns WebSocket server statistics
func (s *Server) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"connected_clients": len(s.clients),
		"broadcast_queue":   len(s.broadcast),
	}
}
