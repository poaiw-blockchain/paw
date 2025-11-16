package reputation

// This file provides example integration code for the PAW blockchain

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
)

// ExampleIntegration demonstrates how to integrate the reputation system
type ExampleIntegration struct {
	manager *Manager
	monitor *Monitor
	metrics *Metrics
	logger  log.Logger
}

// NewExampleIntegration creates a new example integration
func NewExampleIntegration(homeDir string, logger log.Logger) (*ExampleIntegration, error) {
	// 1. Load configuration
	config := DefaultConfig(homeDir)
	configPath := filepath.Join(homeDir, "config", "p2p_security.toml")

	// Try to load custom config
	if cfg, err := LoadConfig(configPath); err == nil {
		config = *cfg
		logger.Info("loaded custom p2p security config", "path", configPath)
	} else {
		logger.Info("using default p2p security config", "error", err)
		// Save default config for future editing
		if err := SaveConfig(&config, configPath); err != nil {
			logger.Error("failed to save default config", "error", err)
		}
	}

	// 2. Create storage
	storageConfig := FileStorageConfig{
		DataDir:       config.Storage.DataDir,
		CacheSize:     config.Storage.CacheSize,
		FlushInterval: config.Storage.FlushInterval,
		EnableCache:   config.Storage.EnableCache,
	}

	var storage Storage
	var err error

	if config.Storage.Type == "memory" {
		storage = NewMemoryStorage()
		logger.Info("using in-memory reputation storage")
	} else {
		storage, err = NewFileStorage(storageConfig, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create file storage: %w", err)
		}
		logger.Info("using file-based reputation storage", "dir", config.Storage.DataDir)
	}

	// 3. Create manager
	managerConfig := config.Manager.ToManagerConfig(
		config.Scoring.ToScoringConfig(),
		config.Scoring.ToScoreWeights(),
	)

	manager, err := NewManager(storage, managerConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create reputation manager: %w", err)
	}

	// 4. Apply whitelist from config
	for _, peerIDStr := range config.Whitelist {
		manager.AddToWhitelist(PeerID(peerIDStr))
		logger.Info("whitelisted peer", "peer_id", peerIDStr)
	}

	// 5. Create metrics
	metrics := NewMetrics()

	// 6. Create monitor
	monitorConfig := DefaultMonitorConfig()
	monitor := NewMonitor(manager, metrics, monitorConfig, logger)

	logger.Info("reputation system initialized successfully")

	return &ExampleIntegration{
		manager: manager,
		monitor: monitor,
		metrics: metrics,
		logger:  logger,
	}, nil
}

// HandlePeerConnected should be called when a new peer connects
func (e *ExampleIntegration) HandlePeerConnected(peerID string, address string) error {
	// Check if peer should be accepted
	shouldAccept, reason := e.manager.ShouldAcceptPeer(PeerID(peerID), address)
	if !shouldAccept {
		e.logger.Info("rejecting peer connection", "peer_id", peerID, "reason", reason)
		return fmt.Errorf("peer rejected: %s", reason)
	}

	// Record connection event
	event := PeerEvent{
		PeerID:    PeerID(peerID),
		EventType: EventTypeConnected,
		Timestamp: time.Now(),
	}

	if err := e.manager.RecordEvent(event); err != nil {
		e.logger.Error("failed to record peer connection", "peer_id", peerID, "error", err)
		return err
	}

	e.logger.Info("peer connected", "peer_id", peerID, "address", address)
	return nil
}

// HandlePeerDisconnected should be called when a peer disconnects
func (e *ExampleIntegration) HandlePeerDisconnected(peerID string) error {
	event := PeerEvent{
		PeerID:    PeerID(peerID),
		EventType: EventTypeDisconnected,
		Timestamp: time.Now(),
	}

	if err := e.manager.RecordEvent(event); err != nil {
		e.logger.Error("failed to record peer disconnection", "peer_id", peerID, "error", err)
		return err
	}

	e.logger.Debug("peer disconnected", "peer_id", peerID)
	return nil
}

// HandleMessageReceived should be called when a message is received from a peer
func (e *ExampleIntegration) HandleMessageReceived(peerID string, messageSize int64, valid bool) error {
	eventType := EventTypeValidMessage
	if !valid {
		eventType = EventTypeInvalidMessage
	}

	event := PeerEvent{
		PeerID:    PeerID(peerID),
		EventType: eventType,
		Timestamp: time.Now(),
		Data: EventData{
			MessageSize: messageSize,
		},
	}

	if err := e.manager.RecordEvent(event); err != nil {
		e.logger.Error("failed to record message event", "peer_id", peerID, "error", err)
		return err
	}

	return nil
}

// HandleBlockReceived should be called when a block is received from a peer
func (e *ExampleIntegration) HandleBlockReceived(peerID string, blockHeight int64, propagationTime time.Duration) error {
	event := PeerEvent{
		PeerID:    PeerID(peerID),
		EventType: EventTypeBlockPropagated,
		Timestamp: time.Now(),
		Data: EventData{
			BlockHeight: blockHeight,
			Latency:     propagationTime,
		},
	}

	if err := e.manager.RecordEvent(event); err != nil {
		e.logger.Error("failed to record block event", "peer_id", peerID, "error", err)
		return err
	}

	return nil
}

// HandleProtocolViolation should be called when a peer violates protocol
func (e *ExampleIntegration) HandleProtocolViolation(peerID string, violationType string, details string) error {
	var eventType EventType

	// Map violation types to events
	switch violationType {
	case "double_sign":
		eventType = EventTypeDoubleSign
	case "invalid_block":
		eventType = EventTypeInvalidBlock
	case "spam":
		eventType = EventTypeSpam
	default:
		eventType = EventTypeProtocolViolation
	}

	event := PeerEvent{
		PeerID:    PeerID(peerID),
		EventType: eventType,
		Timestamp: time.Now(),
		Data: EventData{
			ViolationType: violationType,
			Details:       details,
		},
	}

	if err := e.manager.RecordEvent(event); err != nil {
		e.logger.Error("failed to record violation", "peer_id", peerID, "error", err)
		return err
	}

	e.logger.Warn("protocol violation recorded",
		"peer_id", peerID,
		"type", violationType,
		"details", details,
	)

	return nil
}

// MeasureLatency should be called to record peer response latency
func (e *ExampleIntegration) MeasureLatency(peerID string, latency time.Duration) error {
	event := PeerEvent{
		PeerID:    PeerID(peerID),
		EventType: EventTypeLatencyMeasured,
		Timestamp: time.Now(),
		Data: EventData{
			Latency: latency,
		},
	}

	if err := e.manager.RecordEvent(event); err != nil {
		e.logger.Error("failed to record latency", "peer_id", peerID, "error", err)
		return err
	}

	return nil
}

// SelectPeersForBlockRequest selects best peers for requesting a block
func (e *ExampleIntegration) SelectPeersForBlockRequest(count int) []string {
	// Get top-scoring peers
	peers := e.manager.GetTopPeers(count, 50.0)

	peerIDs := make([]string, len(peers))
	for i, peer := range peers {
		peerIDs[i] = string(peer.PeerID)
	}

	e.logger.Debug("selected peers for block request", "count", len(peerIDs), "requested", count)
	return peerIDs
}

// SelectDiversePeers selects geographically diverse peers
func (e *ExampleIntegration) SelectDiversePeers(count int) []string {
	// Get diverse peers
	peers := e.manager.GetDiversePeers(count, 40.0)

	peerIDs := make([]string, len(peers))
	for i, peer := range peers {
		peerIDs[i] = string(peer.PeerID)
	}

	e.logger.Debug("selected diverse peers", "count", len(peerIDs), "requested", count)
	return peerIDs
}

// GetPeerReputation returns reputation info for a peer
func (e *ExampleIntegration) GetPeerReputation(peerID string) (*PeerReputation, error) {
	return e.manager.GetReputation(PeerID(peerID))
}

// GetSystemHealth returns current system health
func (e *ExampleIntegration) GetSystemHealth() HealthStatus {
	return e.monitor.GetHealth()
}

// GetStatistics returns current reputation statistics
func (e *ExampleIntegration) GetStatistics() Statistics {
	return e.manager.GetStatistics()
}

// BanPeer manually bans a peer
func (e *ExampleIntegration) BanPeer(peerID string, duration time.Duration, reason string) error {
	return e.manager.BanPeer(PeerID(peerID), duration, reason)
}

// UnbanPeer manually unbans a peer
func (e *ExampleIntegration) UnbanPeer(peerID string) error {
	return e.manager.UnbanPeer(PeerID(peerID))
}

// Shutdown gracefully shuts down the reputation system
func (e *ExampleIntegration) Shutdown(ctx context.Context) error {
	e.logger.Info("shutting down reputation system")

	// Close monitor
	if err := e.monitor.Close(); err != nil {
		e.logger.Error("error closing monitor", "error", err)
	}

	// Close manager (saves final snapshot)
	if err := e.manager.Close(); err != nil {
		e.logger.Error("error closing manager", "error", err)
		return err
	}

	e.logger.Info("reputation system shut down successfully")
	return nil
}

// Example usage in main application:
//
// func main() {
//     logger := log.NewLogger(os.Stdout)
//     homeDir := "/home/user/.paw"
//
//     // Initialize reputation system
//     repSystem, err := NewExampleIntegration(homeDir, logger)
//     if err != nil {
//         logger.Error("failed to initialize reputation system", "error", err)
//         return
//     }
//     defer repSystem.Shutdown(context.Background())
//
//     // In your P2P connection handler:
//     if err := repSystem.HandlePeerConnected(peerID, address); err != nil {
//         // Reject connection
//         return err
//     }
//
//     // In your message handler:
//     valid := validateMessage(msg)
//     repSystem.HandleMessageReceived(peerID, len(msg), valid)
//
//     // When selecting peers for operations:
//     peers := repSystem.SelectPeersForBlockRequest(10)
//
//     // Setup HTTP API:
//     handlers := NewHTTPHandlers(repSystem.manager, repSystem.monitor, repSystem.metrics)
//     mux := http.NewServeMux()
//     handlers.RegisterRoutes(mux)
//     go http.ListenAndServe(":8080", mux)
// }
