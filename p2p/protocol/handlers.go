package protocol

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/paw-chain/paw/p2p/reputation"
)

// MessageHandler processes incoming messages
type MessageHandler func(ctx context.Context, peerID string, msg Message) error

// ProtocolHandlers manages message handling
type ProtocolHandlers struct {
	handlers      map[MessageType]MessageHandler
	reputationMgr *reputation.Manager
	logger        log.Logger

	// Handler stats
	stats   map[MessageType]*HandlerStats
	statsMu sync.RWMutex

	// Rate limiting per peer
	peerLimits map[string]*PeerRateLimits
	limitsLock sync.RWMutex

	// Callback hooks
	onHandshake     func(peerID string, msg *HandshakeMessage) (*HandshakeAckMessage, error)
	onBlock         func(peerID string, msg *BlockMessage) error
	onTx            func(peerID string, msg *TxMessage) error
	onSyncRequest   func(peerID string, fromHeight, toHeight int64) ([][]byte, error)
	onPeerRequest   func(peerID string) ([]PeerAddress, error)
	onStatusRequest func(peerID string) (*StatusMessage, error)

	mu sync.RWMutex
}

// HandlerStats tracks handler statistics
type HandlerStats struct {
	MessagesReceived int64
	MessagesHandled  int64
	Errors           int64
	LastProcessed    time.Time
	AvgProcessTime   time.Duration
	mu               sync.RWMutex
}

// PeerRateLimits tracks rate limits per peer
type PeerRateLimits struct {
	BlockLimit *RateLimiter
	TxLimit    *RateLimiter
	MsgLimit   *RateLimiter
	LastReset  time.Time
	Violations int
	mu         sync.Mutex
}

// NewProtocolHandlers creates a new protocol handlers instance
func NewProtocolHandlers(reputationMgr *reputation.Manager, logger log.Logger) *ProtocolHandlers {
	ph := &ProtocolHandlers{
		handlers:      make(map[MessageType]MessageHandler),
		reputationMgr: reputationMgr,
		logger:        logger,
		stats:         make(map[MessageType]*HandlerStats),
		peerLimits:    make(map[string]*PeerRateLimits),
	}

	// Register default handlers
	ph.registerDefaultHandlers()

	return ph
}

// RegisterHandler registers a custom message handler
func (ph *ProtocolHandlers) RegisterHandler(msgType MessageType, handler MessageHandler) {
	ph.mu.Lock()
	defer ph.mu.Unlock()

	ph.handlers[msgType] = handler
	ph.stats[msgType] = &HandlerStats{}
	ph.logger.Info("handler registered", "message_type", msgType.String())
}

// HandleMessage processes an incoming message
func (ph *ProtocolHandlers) HandleMessage(ctx context.Context, peerID string, msg Message) error {
	msgType := msg.Type()

	// Record message receipt
	ph.recordMessageReceived(msgType)

	// Check rate limits
	if !ph.checkRateLimit(peerID, msgType) {
		ph.recordRateLimitViolation(peerID)
		return fmt.Errorf("rate limit exceeded for peer %s", peerID)
	}

	// Get handler
	ph.mu.RLock()
	handler, exists := ph.handlers[msgType]
	ph.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no handler for message type: %s", msgType.String())
	}

	// Measure processing time
	startTime := time.Now()

	// Execute handler
	err := handler(ctx, peerID, msg)

	processingTime := time.Since(startTime)

	// Update stats
	if err != nil {
		ph.recordError(msgType)
		ph.logger.Error("message handling failed",
			"peer_id", peerID,
			"message_type", msgType.String(),
			"error", err,
		)

		// Record negative reputation event
		ph.recordReputationEvent(peerID, reputation.EventTypeInvalidMessage)
	} else {
		ph.recordMessageHandled(msgType, processingTime)
		ph.logger.Debug("message handled",
			"peer_id", peerID,
			"message_type", msgType.String(),
			"processing_time", processingTime,
		)

		// Record positive reputation event for valid messages
		ph.recordReputationEvent(peerID, reputation.EventTypeValidMessage)
	}

	return err
}

// registerDefaultHandlers registers built-in handlers
func (ph *ProtocolHandlers) registerDefaultHandlers() {
	// Handshake handler
	ph.RegisterHandler(MsgTypeHandshake, ph.handleHandshake)

	// Block handlers
	ph.RegisterHandler(MsgTypeNewBlock, ph.handleNewBlock)
	ph.RegisterHandler(MsgTypeBlockRequest, ph.handleBlockRequest)
	ph.RegisterHandler(MsgTypeBlockAnnounce, ph.handleBlockAnnounce)

	// Transaction handlers
	ph.RegisterHandler(MsgTypeNewTx, ph.handleNewTx)
	ph.RegisterHandler(MsgTypeTxRequest, ph.handleTxRequest)

	// Sync handlers
	ph.RegisterHandler(MsgTypeSyncRequest, ph.handleSyncRequest)

	// Peer discovery handlers
	ph.RegisterHandler(MsgTypePeerRequest, ph.handlePeerRequest)
	ph.RegisterHandler(MsgTypePeerAnnounce, ph.handlePeerAnnounce)

	// Status handlers
	ph.RegisterHandler(MsgTypePing, ph.handlePing)
	ph.RegisterHandler(MsgTypePong, ph.handlePong)
	ph.RegisterHandler(MsgTypeStatus, ph.handleStatus)

	// Error handler
	ph.RegisterHandler(MsgTypeError, ph.handleError)
}

// Built-in handlers

func (ph *ProtocolHandlers) handleHandshake(ctx context.Context, peerID string, msg Message) error {
	handshake, ok := msg.(*HandshakeMessage)
	if !ok {
		return fmt.Errorf("invalid message type for handshake")
	}

	ph.logger.Info("received handshake",
		"peer_id", peerID,
		"chain_id", handshake.ChainID,
		"best_height", handshake.BestHeight,
	)

	// Call custom handler if set
	ph.mu.RLock()
	handler := ph.onHandshake
	ph.mu.RUnlock()

	if handler != nil {
		ack, err := handler(peerID, handshake)
		if err != nil {
			return fmt.Errorf("handshake handler failed: %w", err)
		}
		ph.logger.Info("handshake processed", "peer_id", peerID, "accepted", ack.Accepted)
	}

	return nil
}

func (ph *ProtocolHandlers) handleNewBlock(ctx context.Context, peerID string, msg Message) error {
	block, ok := msg.(*BlockMessage)
	if !ok {
		return fmt.Errorf("invalid message type for block")
	}

	ph.logger.Debug("received block",
		"peer_id", peerID,
		"height", block.Height,
		"hash", fmt.Sprintf("%x", block.Hash[:min(len(block.Hash), 8)]),
	)

	// Call custom handler if set
	ph.mu.RLock()
	handler := ph.onBlock
	ph.mu.RUnlock()

	if handler != nil {
		if err := handler(peerID, block); err != nil {
			return fmt.Errorf("block handler failed: %w", err)
		}
	}

	// Record block propagation event
	ph.recordReputationEvent(peerID, reputation.EventTypeBlockPropagated)

	return nil
}

func (ph *ProtocolHandlers) handleBlockRequest(ctx context.Context, peerID string, msg Message) error {
	ph.logger.Debug("received block request", "peer_id", peerID)
	return nil
}

func (ph *ProtocolHandlers) handleBlockAnnounce(ctx context.Context, peerID string, msg Message) error {
	ph.logger.Debug("received block announce", "peer_id", peerID)
	return nil
}

func (ph *ProtocolHandlers) handleNewTx(ctx context.Context, peerID string, msg Message) error {
	tx, ok := msg.(*TxMessage)
	if !ok {
		return fmt.Errorf("invalid message type for transaction")
	}

	ph.logger.Debug("received transaction",
		"peer_id", peerID,
		"hash", fmt.Sprintf("%x", tx.TxHash[:min(len(tx.TxHash), 8)]),
	)

	// Call custom handler if set
	ph.mu.RLock()
	handler := ph.onTx
	ph.mu.RUnlock()

	if handler != nil {
		if err := handler(peerID, tx); err != nil {
			return fmt.Errorf("tx handler failed: %w", err)
		}
	}

	return nil
}

func (ph *ProtocolHandlers) handleTxRequest(ctx context.Context, peerID string, msg Message) error {
	ph.logger.Debug("received tx request", "peer_id", peerID)
	return nil
}

func (ph *ProtocolHandlers) handleSyncRequest(ctx context.Context, peerID string, msg Message) error {
	ph.logger.Debug("received sync request", "peer_id", peerID)

	// Call custom handler if set
	ph.mu.RLock()
	handler := ph.onSyncRequest
	ph.mu.RUnlock()

	if handler != nil {
		// Extract height range from message (would need proper message type)
		// For now, return empty
		_, err := handler(peerID, 0, 0)
		if err != nil {
			return fmt.Errorf("sync request handler failed: %w", err)
		}
	}

	return nil
}

func (ph *ProtocolHandlers) handlePeerRequest(ctx context.Context, peerID string, msg Message) error {
	ph.logger.Debug("received peer request", "peer_id", peerID)

	// Call custom handler if set
	ph.mu.RLock()
	handler := ph.onPeerRequest
	ph.mu.RUnlock()

	if handler != nil {
		_, err := handler(peerID)
		if err != nil {
			return fmt.Errorf("peer request handler failed: %w", err)
		}
	}

	return nil
}

func (ph *ProtocolHandlers) handlePeerAnnounce(ctx context.Context, peerID string, msg Message) error {
	peerList, ok := msg.(*PeerListMessage)
	if !ok {
		return fmt.Errorf("invalid message type for peer announce")
	}

	ph.logger.Debug("received peer announce",
		"peer_id", peerID,
		"peers_count", len(peerList.Peers),
	)

	return nil
}

func (ph *ProtocolHandlers) handlePing(ctx context.Context, peerID string, msg Message) error {
	ph.logger.Debug("received ping", "peer_id", peerID)

	// Record latency measurement
	ph.recordReputationEvent(peerID, reputation.EventTypeLatencyMeasured)

	return nil
}

func (ph *ProtocolHandlers) handlePong(ctx context.Context, peerID string, msg Message) error {
	ph.logger.Debug("received pong", "peer_id", peerID)
	return nil
}

func (ph *ProtocolHandlers) handleStatus(ctx context.Context, peerID string, msg Message) error {
	status, ok := msg.(*StatusMessage)
	if !ok {
		return fmt.Errorf("invalid message type for status")
	}

	ph.logger.Debug("received status",
		"peer_id", peerID,
		"height", status.Height,
		"syncing", status.Syncing,
	)

	// Call custom handler if set
	ph.mu.RLock()
	handler := ph.onStatusRequest
	ph.mu.RUnlock()

	if handler != nil {
		_, err := handler(peerID)
		if err != nil {
			return fmt.Errorf("status handler failed: %w", err)
		}
	}

	return nil
}

func (ph *ProtocolHandlers) handleError(ctx context.Context, peerID string, msg Message) error {
	errMsg, ok := msg.(*ErrorMessage)
	if !ok {
		return fmt.Errorf("invalid message type for error")
	}

	ph.logger.Warn("received error from peer",
		"peer_id", peerID,
		"code", errMsg.Code,
		"message", errMsg.Message,
	)

	return nil
}

// Callback setters

func (ph *ProtocolHandlers) SetHandshakeHandler(handler func(string, *HandshakeMessage) (*HandshakeAckMessage, error)) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.onHandshake = handler
}

func (ph *ProtocolHandlers) SetBlockHandler(handler func(string, *BlockMessage) error) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.onBlock = handler
}

func (ph *ProtocolHandlers) SetTxHandler(handler func(string, *TxMessage) error) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.onTx = handler
}

func (ph *ProtocolHandlers) SetSyncHandler(handler func(string, int64, int64) ([][]byte, error)) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.onSyncRequest = handler
}

func (ph *ProtocolHandlers) SetPeerRequestHandler(handler func(string) ([]PeerAddress, error)) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.onPeerRequest = handler
}

func (ph *ProtocolHandlers) SetStatusHandler(handler func(string) (*StatusMessage, error)) {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	ph.onStatusRequest = handler
}

// Rate limiting

func (ph *ProtocolHandlers) checkRateLimit(peerID string, msgType MessageType) bool {
	ph.limitsLock.Lock()
	defer ph.limitsLock.Unlock()

	limits, exists := ph.peerLimits[peerID]
	if !exists {
		limits = &PeerRateLimits{
			BlockLimit: NewRateLimiter(10, 20),   // 10 blocks/sec
			TxLimit:    NewRateLimiter(100, 200), // 100 txs/sec
			MsgLimit:   NewRateLimiter(50, 100),  // 50 msgs/sec
			LastReset:  time.Now(),
		}
		ph.peerLimits[peerID] = limits
	}

	limits.mu.Lock()
	defer limits.mu.Unlock()

	// Reset counters if needed (every minute)
	if time.Since(limits.LastReset) > time.Minute {
		limits.BlockLimit = NewRateLimiter(10, 20)
		limits.TxLimit = NewRateLimiter(100, 200)
		limits.MsgLimit = NewRateLimiter(50, 100)
		limits.LastReset = time.Now()
		limits.Violations = 0
	}

	// Check message-specific limit
	switch msgType {
	case MsgTypeNewBlock, MsgTypeBlockRequest, MsgTypeBlockResponse, MsgTypeBlockAnnounce:
		return limits.BlockLimit.Allow()
	case MsgTypeNewTx, MsgTypeTxRequest, MsgTypeTxResponse:
		return limits.TxLimit.Allow()
	default:
		return limits.MsgLimit.Allow()
	}
}

func (ph *ProtocolHandlers) recordRateLimitViolation(peerID string) {
	ph.limitsLock.Lock()
	defer ph.limitsLock.Unlock()

	if limits, exists := ph.peerLimits[peerID]; exists {
		limits.mu.Lock()
		limits.Violations++
		violations := limits.Violations
		limits.mu.Unlock()

		// Ban peer if too many violations
		if violations > 10 {
			ph.logger.Warn("peer exceeded rate limits", "peer_id", peerID, "violations", violations)
			ph.recordReputationEvent(peerID, reputation.EventTypeSpam)
		}
	}
}

// Stats tracking

func (ph *ProtocolHandlers) recordMessageReceived(msgType MessageType) {
	ph.statsMu.Lock()
	defer ph.statsMu.Unlock()

	if stats, exists := ph.stats[msgType]; exists {
		stats.mu.Lock()
		stats.MessagesReceived++
		stats.mu.Unlock()
	}
}

func (ph *ProtocolHandlers) recordMessageHandled(msgType MessageType, processingTime time.Duration) {
	ph.statsMu.Lock()
	defer ph.statsMu.Unlock()

	if stats, exists := ph.stats[msgType]; exists {
		stats.mu.Lock()
		stats.MessagesHandled++
		stats.LastProcessed = time.Now()

		// Update average processing time
		if stats.AvgProcessTime == 0 {
			stats.AvgProcessTime = processingTime
		} else {
			stats.AvgProcessTime = (stats.AvgProcessTime + processingTime) / 2
		}
		stats.mu.Unlock()
	}
}

func (ph *ProtocolHandlers) recordError(msgType MessageType) {
	ph.statsMu.Lock()
	defer ph.statsMu.Unlock()

	if stats, exists := ph.stats[msgType]; exists {
		stats.mu.Lock()
		stats.Errors++
		stats.mu.Unlock()
	}
}

func (ph *ProtocolHandlers) recordReputationEvent(peerID string, eventType reputation.EventType) {
	if ph.reputationMgr == nil {
		return
	}

	event := reputation.PeerEvent{
		PeerID:    reputation.PeerID(peerID),
		EventType: eventType,
		Timestamp: time.Now(),
	}

	if err := ph.reputationMgr.RecordEvent(event); err != nil {
		ph.logger.Error("failed to record reputation event",
			"peer_id", peerID,
			"event_type", eventType.String(),
			"error", err,
		)
	}
}

// GetStats returns handler statistics
func (ph *ProtocolHandlers) GetStats(msgType MessageType) *HandlerStats {
	ph.statsMu.RLock()
	defer ph.statsMu.RUnlock()

	if stats, exists := ph.stats[msgType]; exists {
		stats.mu.RLock()
		defer stats.mu.RUnlock()

		// Return a copy
		return &HandlerStats{
			MessagesReceived: stats.MessagesReceived,
			MessagesHandled:  stats.MessagesHandled,
			Errors:           stats.Errors,
			LastProcessed:    stats.LastProcessed,
			AvgProcessTime:   stats.AvgProcessTime,
		}
	}

	return nil
}

// GetAllStats returns all handler statistics
func (ph *ProtocolHandlers) GetAllStats() map[MessageType]*HandlerStats {
	ph.statsMu.RLock()
	defer ph.statsMu.RUnlock()

	result := make(map[MessageType]*HandlerStats)
	for msgType, stats := range ph.stats {
		stats.mu.RLock()
		result[msgType] = &HandlerStats{
			MessagesReceived: stats.MessagesReceived,
			MessagesHandled:  stats.MessagesHandled,
			Errors:           stats.Errors,
			LastProcessed:    stats.LastProcessed,
			AvgProcessTime:   stats.AvgProcessTime,
		}
		stats.mu.RUnlock()
	}

	return result
}

// CleanupPeer removes rate limit tracking for a disconnected peer
func (ph *ProtocolHandlers) CleanupPeer(peerID string) {
	ph.limitsLock.Lock()
	defer ph.limitsLock.Unlock()

	delete(ph.peerLimits, peerID)
}
