package protocol

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/paw-chain/paw/p2p/reputation"
)

// GossipConfig configures gossip behavior
type GossipConfig struct {
	// Fanout for gossip propagation
	BlockFanout int
	TxFanout    int
	PeerFanout  int

	// Propagation intervals
	BlockPropagationInterval time.Duration
	TxPropagationInterval    time.Duration
	PeerPropagationInterval  time.Duration

	// Anti-spam settings
	MaxBlockGossipRate  int // blocks per second
	MaxTxGossipRate     int // transactions per second
	DuplicateExpiration time.Duration

	// Peer selection
	MinPeerReputation float64
	EnableDiversity   bool
}

// DefaultGossipConfig returns default gossip configuration
func DefaultGossipConfig() GossipConfig {
	return GossipConfig{
		BlockFanout:              8,
		TxFanout:                 4,
		PeerFanout:               3,
		BlockPropagationInterval: 100 * time.Millisecond,
		TxPropagationInterval:    50 * time.Millisecond,
		PeerPropagationInterval:  30 * time.Second,
		MaxBlockGossipRate:       10,
		MaxTxGossipRate:          100,
		DuplicateExpiration:      5 * time.Minute,
		MinPeerReputation:        30.0,
		EnableDiversity:          true,
	}
}

// GossipProtocol manages gossip-based message propagation
type GossipProtocol struct {
	config        GossipConfig
	reputationMgr *reputation.Manager
	logger        log.Logger

	// Peer connections
	peers   map[string]*PeerConnection
	peersMu sync.RWMutex

	// Duplicate detection
	seenBlocks map[string]time.Time
	seenTxs    map[string]time.Time
	seenMu     sync.RWMutex

	// Rate limiting
	blockRateLimiter *RateLimiter
	txRateLimiter    *RateLimiter

	// Propagation queues
	blockQueue chan *BlockMessage
	txQueue    chan *TxMessage
	peerQueue  chan *PeerListMessage

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Metrics
	metrics *GossipMetrics
}

// PeerConnection represents a connection to a peer
type PeerConnection struct {
	ID         string
	Address    string
	Outbound   bool
	LastSeen   time.Time
	Reputation float64
	SendChan   chan Message
}

// GossipMetrics tracks gossip statistics
type GossipMetrics struct {
	BlocksGossiped     int64
	TxsGossiped        int64
	PeersGossiped      int64
	DuplicatesFiltered int64
	RateLimited        int64
	mu                 sync.RWMutex
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	rate     int
	capacity int
	tokens   int
	lastFill time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate, capacity int) *RateLimiter {
	return &RateLimiter{
		rate:     rate,
		capacity: capacity,
		tokens:   capacity,
		lastFill: time.Now(),
	}
}

// Allow checks if an action is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastFill)

	// Refill tokens based on elapsed time
	tokensToAdd := int(elapsed.Seconds() * float64(rl.rate))
	if tokensToAdd > 0 {
		rl.tokens = min(rl.capacity, rl.tokens+tokensToAdd)
		rl.lastFill = now
	}

	if rl.tokens > 0 {
		rl.tokens--
		return true
	}

	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewGossipProtocol creates a new gossip protocol instance
func NewGossipProtocol(
	config GossipConfig,
	reputationMgr *reputation.Manager,
	logger log.Logger,
) *GossipProtocol {
	ctx, cancel := context.WithCancel(context.Background())

	gp := &GossipProtocol{
		config:           config,
		reputationMgr:    reputationMgr,
		logger:           logger,
		peers:            make(map[string]*PeerConnection),
		seenBlocks:       make(map[string]time.Time),
		seenTxs:          make(map[string]time.Time),
		blockRateLimiter: NewRateLimiter(config.MaxBlockGossipRate, config.MaxBlockGossipRate*2),
		txRateLimiter:    NewRateLimiter(config.MaxTxGossipRate, config.MaxTxGossipRate*2),
		blockQueue:       make(chan *BlockMessage, 100),
		txQueue:          make(chan *TxMessage, 1000),
		peerQueue:        make(chan *PeerListMessage, 10),
		ctx:              ctx,
		cancel:           cancel,
		metrics:          &GossipMetrics{},
	}

	// Start background workers
	gp.startWorkers()

	return gp
}

// AddPeer adds a peer to the gossip network
func (gp *GossipProtocol) AddPeer(id, address string, outbound bool) error {
	gp.peersMu.Lock()
	defer gp.peersMu.Unlock()

	if _, exists := gp.peers[id]; exists {
		return fmt.Errorf("peer already exists: %s", id)
	}

	// Get reputation
	rep, err := gp.reputationMgr.GetReputation(reputation.PeerID(id))
	score := 50.0 // Default score for new peers
	if err == nil && rep != nil {
		score = rep.Score
	}

	peer := &PeerConnection{
		ID:         id,
		Address:    address,
		Outbound:   outbound,
		LastSeen:   time.Now(),
		Reputation: score,
		SendChan:   make(chan Message, 100),
	}

	gp.peers[id] = peer
	gp.logger.Info("peer added to gossip", "peer_id", id, "reputation", score)

	return nil
}

// RemovePeer removes a peer from the gossip network
func (gp *GossipProtocol) RemovePeer(id string) {
	gp.peersMu.Lock()
	defer gp.peersMu.Unlock()

	if peer, exists := gp.peers[id]; exists {
		close(peer.SendChan)
		delete(gp.peers, id)
		gp.logger.Info("peer removed from gossip", "peer_id", id)
	}
}

// GossipBlock gossips a block to connected peers
func (gp *GossipProtocol) GossipBlock(msg *BlockMessage) error {
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("invalid block message: %w", err)
	}

	// Check rate limit
	if !gp.blockRateLimiter.Allow() {
		gp.metrics.mu.Lock()
		gp.metrics.RateLimited++
		gp.metrics.mu.Unlock()
		return fmt.Errorf("block gossip rate limit exceeded")
	}

	// Check if already seen
	blockKey := fmt.Sprintf("%d:%x", msg.Height, msg.Hash)
	if gp.hasSeen(blockKey, true) {
		gp.metrics.mu.Lock()
		gp.metrics.DuplicatesFiltered++
		gp.metrics.mu.Unlock()
		return nil
	}

	// Queue for propagation
	select {
	case gp.blockQueue <- msg:
		return nil
	case <-gp.ctx.Done():
		return fmt.Errorf("gossip protocol stopped")
	default:
		return fmt.Errorf("block queue full")
	}
}

// GossipTx gossips a transaction to connected peers
func (gp *GossipProtocol) GossipTx(msg *TxMessage) error {
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("invalid tx message: %w", err)
	}

	// Check rate limit
	if !gp.txRateLimiter.Allow() {
		gp.metrics.mu.Lock()
		gp.metrics.RateLimited++
		gp.metrics.mu.Unlock()
		return fmt.Errorf("tx gossip rate limit exceeded")
	}

	// Check if already seen
	txKey := fmt.Sprintf("%x", msg.TxHash)
	if gp.hasSeen(txKey, false) {
		gp.metrics.mu.Lock()
		gp.metrics.DuplicatesFiltered++
		gp.metrics.mu.Unlock()
		return nil
	}

	// Queue for propagation
	select {
	case gp.txQueue <- msg:
		return nil
	case <-gp.ctx.Done():
		return fmt.Errorf("gossip protocol stopped")
	default:
		return fmt.Errorf("tx queue full")
	}
}

// GossipPeers gossips peer addresses
func (gp *GossipProtocol) GossipPeers(msg *PeerListMessage) error {
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("invalid peer list message: %w", err)
	}

	select {
	case gp.peerQueue <- msg:
		return nil
	case <-gp.ctx.Done():
		return fmt.Errorf("gossip protocol stopped")
	default:
		return fmt.Errorf("peer queue full")
	}
}

// hasSeen checks if we've seen this item before
func (gp *GossipProtocol) hasSeen(key string, isBlock bool) bool {
	gp.seenMu.Lock()
	defer gp.seenMu.Unlock()

	now := time.Now()

	if isBlock {
		if seenTime, exists := gp.seenBlocks[key]; exists {
			if now.Sub(seenTime) < gp.config.DuplicateExpiration {
				return true
			}
			delete(gp.seenBlocks, key)
		}
		gp.seenBlocks[key] = now
	} else {
		if seenTime, exists := gp.seenTxs[key]; exists {
			if now.Sub(seenTime) < gp.config.DuplicateExpiration {
				return true
			}
			delete(gp.seenTxs, key)
		}
		gp.seenTxs[key] = now
	}

	return false
}

// selectPeers selects peers for gossip based on reputation and diversity
func (gp *GossipProtocol) selectPeers(fanout int, excludePeer string) []*PeerConnection {
	gp.peersMu.RLock()
	defer gp.peersMu.RUnlock()

	// Filter eligible peers
	eligible := make([]*PeerConnection, 0, len(gp.peers))
	for _, peer := range gp.peers {
		if peer.ID == excludePeer {
			continue
		}
		if peer.Reputation >= gp.config.MinPeerReputation {
			eligible = append(eligible, peer)
		}
	}

	if len(eligible) == 0 {
		return nil
	}

	// If we have fewer peers than fanout, return all
	if len(eligible) <= fanout {
		return eligible
	}

	// Select peers
	if gp.config.EnableDiversity {
		return gp.selectDiversePeers(eligible, fanout)
	}

	return gp.selectTopPeers(eligible, fanout)
}

// selectTopPeers selects peers with highest reputation
func (gp *GossipProtocol) selectTopPeers(peers []*PeerConnection, n int) []*PeerConnection {
	// Simple bubble sort for top N (good enough for small lists)
	sorted := make([]*PeerConnection, len(peers))
	copy(sorted, peers)

	for i := 0; i < len(sorted)-1 && i < n; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].Reputation < sorted[j+1].Reputation {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	if n > len(sorted) {
		n = len(sorted)
	}

	return sorted[:n]
}

// selectDiversePeers selects peers ensuring diversity
func (gp *GossipProtocol) selectDiversePeers(peers []*PeerConnection, n int) []*PeerConnection {
	// For simplicity, use reputation-based selection
	// In production, this would consider geographic diversity
	return gp.selectTopPeers(peers, n)
}

// propagateBlock propagates a block to selected peers
func (gp *GossipProtocol) propagateBlock(msg *BlockMessage) {
	peers := gp.selectPeers(gp.config.BlockFanout, msg.Source)
	if len(peers) == 0 {
		return
	}

	gp.logger.Debug("propagating block",
		"height", msg.Height,
		"hash", fmt.Sprintf("%x", msg.Hash[:min(len(msg.Hash), 8)]),
		"peers", len(peers),
	)

	for _, peer := range peers {
		select {
		case peer.SendChan <- msg:
			gp.metrics.mu.Lock()
			gp.metrics.BlocksGossiped++
			gp.metrics.mu.Unlock()
		default:
			gp.logger.Warn("peer send channel full", "peer_id", peer.ID)
		}
	}
}

// propagateTx propagates a transaction to selected peers
func (gp *GossipProtocol) propagateTx(msg *TxMessage) {
	peers := gp.selectPeers(gp.config.TxFanout, msg.Source)
	if len(peers) == 0 {
		return
	}

	gp.logger.Debug("propagating tx",
		"hash", fmt.Sprintf("%x", msg.TxHash[:min(len(msg.TxHash), 8)]),
		"peers", len(peers),
	)

	for _, peer := range peers {
		select {
		case peer.SendChan <- msg:
			gp.metrics.mu.Lock()
			gp.metrics.TxsGossiped++
			gp.metrics.mu.Unlock()
		default:
			gp.logger.Warn("peer send channel full", "peer_id", peer.ID)
		}
	}
}

// propagatePeers propagates peer addresses
func (gp *GossipProtocol) propagatePeers(msg *PeerListMessage) {
	peers := gp.selectPeers(gp.config.PeerFanout, "")
	if len(peers) == 0 {
		return
	}

	gp.logger.Debug("propagating peers", "count", len(msg.Peers), "to", len(peers))

	for _, peer := range peers {
		select {
		case peer.SendChan <- msg:
			gp.metrics.mu.Lock()
			gp.metrics.PeersGossiped++
			gp.metrics.mu.Unlock()
		default:
			gp.logger.Warn("peer send channel full", "peer_id", peer.ID)
		}
	}
}

// cleanupSeen periodically cleans up expired entries
func (gp *GossipProtocol) cleanupSeen() {
	gp.seenMu.Lock()
	defer gp.seenMu.Unlock()

	now := time.Now()
	expiration := gp.config.DuplicateExpiration

	// Cleanup blocks
	for key, seenTime := range gp.seenBlocks {
		if now.Sub(seenTime) > expiration {
			delete(gp.seenBlocks, key)
		}
	}

	// Cleanup txs
	for key, seenTime := range gp.seenTxs {
		if now.Sub(seenTime) > expiration {
			delete(gp.seenTxs, key)
		}
	}
}

// startWorkers starts background workers
func (gp *GossipProtocol) startWorkers() {
	// Block propagation worker
	gp.wg.Add(1)
	go func() {
		defer gp.wg.Done()
		ticker := time.NewTicker(gp.config.BlockPropagationInterval)
		defer ticker.Stop()

		for {
			select {
			case msg := <-gp.blockQueue:
				gp.propagateBlock(msg)
			case <-ticker.C:
				// Periodic processing if needed
			case <-gp.ctx.Done():
				return
			}
		}
	}()

	// Tx propagation worker
	gp.wg.Add(1)
	go func() {
		defer gp.wg.Done()
		ticker := time.NewTicker(gp.config.TxPropagationInterval)
		defer ticker.Stop()

		for {
			select {
			case msg := <-gp.txQueue:
				gp.propagateTx(msg)
			case <-ticker.C:
				// Periodic processing if needed
			case <-gp.ctx.Done():
				return
			}
		}
	}()

	// Peer propagation worker
	gp.wg.Add(1)
	go func() {
		defer gp.wg.Done()
		ticker := time.NewTicker(gp.config.PeerPropagationInterval)
		defer ticker.Stop()

		for {
			select {
			case msg := <-gp.peerQueue:
				gp.propagatePeers(msg)
			case <-ticker.C:
				// Periodic processing if needed
			case <-gp.ctx.Done():
				return
			}
		}
	}()

	// Cleanup worker
	gp.wg.Add(1)
	go func() {
		defer gp.wg.Done()
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				gp.cleanupSeen()
			case <-gp.ctx.Done():
				return
			}
		}
	}()
}

// GetMetrics returns current gossip metrics
func (gp *GossipProtocol) GetMetrics() GossipMetrics {
	gp.metrics.mu.RLock()
	defer gp.metrics.mu.RUnlock()
	return GossipMetrics{
		BlocksGossiped:     gp.metrics.BlocksGossiped,
		TxsGossiped:        gp.metrics.TxsGossiped,
		PeersGossiped:      gp.metrics.PeersGossiped,
		DuplicatesFiltered: gp.metrics.DuplicatesFiltered,
		RateLimited:        gp.metrics.RateLimited,
	}
}

// GetPeerCount returns the number of connected peers
func (gp *GossipProtocol) GetPeerCount() int {
	gp.peersMu.RLock()
	defer gp.peersMu.RUnlock()
	return len(gp.peers)
}

// UpdatePeerReputation updates a peer's reputation score
func (gp *GossipProtocol) UpdatePeerReputation(peerID string, score float64) {
	gp.peersMu.Lock()
	defer gp.peersMu.Unlock()

	if peer, exists := gp.peers[peerID]; exists {
		peer.Reputation = score
		peer.LastSeen = time.Now()
	}
}

// Stop stops the gossip protocol
func (gp *GossipProtocol) Stop() {
	gp.logger.Info("stopping gossip protocol")
	gp.cancel()
	gp.wg.Wait()

	// Close all peer channels
	gp.peersMu.Lock()
	for _, peer := range gp.peers {
		close(peer.SendChan)
	}
	gp.peers = make(map[string]*PeerConnection)
	gp.peersMu.Unlock()

	gp.logger.Info("gossip protocol stopped")
}
