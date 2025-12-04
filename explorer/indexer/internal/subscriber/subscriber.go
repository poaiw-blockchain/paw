package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// BlockEvent represents a new block event from the blockchain
type BlockEvent struct {
	Height    int64                  `json:"height"`
	Hash      string                 `json:"hash"`
	Time      time.Time              `json:"time"`
	Proposer  string                 `json:"proposer"`
	Txs       []TransactionResult    `json:"txs"`
	RawBlock  map[string]interface{} `json:"raw_block"`
}

// TransactionResult represents a transaction result
type TransactionResult struct {
	Hash      string                   `json:"hash"`
	Code      int                      `json:"code"`
	GasUsed   int64                    `json:"gas_used"`
	GasWanted int64                    `json:"gas_wanted"`
	Events    []Event                  `json:"events"`
	Log       string                   `json:"log"`
	RawTx     map[string]interface{}   `json:"raw_tx"`
}

// Event represents a blockchain event
type Event struct {
	Type       string                 `json:"type"`
	Attributes []Attribute            `json:"attributes"`
}

// Attribute represents an event attribute
type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Subscriber manages WebSocket connection to blockchain node
type Subscriber struct {
	wsURL      string
	conn       *websocket.Conn
	eventChan  chan BlockEvent
	reconnect  bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewSubscriber creates a new blockchain subscriber
func NewSubscriber(wsURL string, bufferSize int) *Subscriber {
	ctx, cancel := context.WithCancel(context.Background())
	return &Subscriber{
		wsURL:     wsURL,
		eventChan: make(chan BlockEvent, bufferSize),
		reconnect: true,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start begins subscribing to blockchain events
func (s *Subscriber) Start() error {
	if err := s.connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Subscribe to new block events
	if err := s.subscribe(); err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	// Start listening for events
	go s.listen()

	log.Info().Msg("Blockchain subscriber started successfully")
	return nil
}

// connect establishes WebSocket connection to the blockchain node
func (s *Subscriber) connect() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(s.wsURL, nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}

	s.conn = conn
	log.Info().Str("url", s.wsURL).Msg("Connected to blockchain node")
	return nil
}

// subscribe sends subscription request for new block events
func (s *Subscriber) subscribe() error {
	subscribeMsg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "subscribe",
		"id":      1,
		"params": map[string]interface{}{
			"query": "tm.event='NewBlock'",
		},
	}

	if err := s.conn.WriteJSON(subscribeMsg); err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	log.Info().Msg("Subscribed to NewBlock events")
	return nil
}

// listen continuously reads events from WebSocket connection
func (s *Subscriber) listen() {
	defer close(s.eventChan)

	for {
		select {
		case <-s.ctx.Done():
			log.Info().Msg("Subscriber stopped by context")
			return
		default:
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				log.Error().Err(err).Msg("Failed to read message from websocket")

				if s.reconnect {
					log.Info().Msg("Attempting to reconnect...")
					if err := s.reconnectWithRetry(); err != nil {
						log.Error().Err(err).Msg("Failed to reconnect, stopping subscriber")
						return
					}
					continue
				}
				return
			}

			// Parse the message
			var wsMsg map[string]interface{}
			if err := json.Unmarshal(message, &wsMsg); err != nil {
				log.Error().Err(err).Msg("Failed to unmarshal message")
				continue
			}

			// Check if this is a block event
			if result, ok := wsMsg["result"].(map[string]interface{}); ok {
				if data, ok := result["data"].(map[string]interface{}); ok {
					if blockEvent := s.parseBlockEvent(data); blockEvent != nil {
						select {
						case s.eventChan <- *blockEvent:
						case <-s.ctx.Done():
							return
						}
					}
				}
			}
		}
	}
}

// parseBlockEvent parses raw block event data
func (s *Subscriber) parseBlockEvent(data map[string]interface{}) *BlockEvent {
	blockValue, ok := data["value"].(map[string]interface{})
	if !ok {
		return nil
	}

	block, ok := blockValue["block"].(map[string]interface{})
	if !ok {
		return nil
	}

	header, ok := block["header"].(map[string]interface{})
	if !ok {
		return nil
	}

	// Extract basic block info
	height, _ := header["height"].(string)
	blockTime, _ := header["time"].(string)
	proposer, _ := header["proposer_address"].(string)

	// Parse time
	parsedTime, err := time.Parse(time.RFC3339, blockTime)
	if err != nil {
		parsedTime = time.Now()
	}

	// Parse height
	var heightInt int64
	fmt.Sscanf(height, "%d", &heightInt)

	event := &BlockEvent{
		Height:   heightInt,
		Time:     parsedTime,
		Proposer: proposer,
		RawBlock: block,
	}

	// Extract transactions if present
	if txsData, ok := block["data"].(map[string]interface{}); ok {
		if txs, ok := txsData["txs"].([]interface{}); ok {
			event.Txs = make([]TransactionResult, len(txs))
			// Note: Full transaction parsing would happen in the indexer
		}
	}

	return event
}

// reconnectWithRetry attempts to reconnect with exponential backoff
func (s *Subscriber) reconnectWithRetry() error {
	maxRetries := 10
	baseDelay := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		delay := baseDelay * time.Duration(1<<uint(i))
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}

		log.Info().
			Int("attempt", i+1).
			Dur("delay", delay).
			Msg("Reconnecting to blockchain node")

		time.Sleep(delay)

		if err := s.connect(); err != nil {
			log.Error().Err(err).Msg("Reconnection failed")
			continue
		}

		if err := s.subscribe(); err != nil {
			log.Error().Err(err).Msg("Resubscription failed")
			s.conn.Close()
			continue
		}

		log.Info().Msg("Reconnected successfully")
		return nil
	}

	return fmt.Errorf("failed to reconnect after %d attempts", maxRetries)
}

// Events returns the channel for receiving block events
func (s *Subscriber) Events() <-chan BlockEvent {
	return s.eventChan
}

// Stop stops the subscriber
func (s *Subscriber) Stop() {
	log.Info().Msg("Stopping blockchain subscriber")
	s.reconnect = false
	s.cancel()
	if s.conn != nil {
		s.conn.Close()
	}
}
