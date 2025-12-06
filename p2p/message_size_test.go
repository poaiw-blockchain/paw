package p2p

import (
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/paw-chain/paw/p2p/reputation"
	"github.com/stretchr/testify/require"
)

func TestMessageSizeLimits(t *testing.T) {
	// Create a test node
	config := DefaultNodeConfig()
	config.DataDir = t.TempDir()
	config.NodeID = "test-node-1"
	config.EnableReputation = false // Disable reputation for simpler test

	logger := log.NewNopLogger()
	node, err := NewNode(config, logger)
	require.NoError(t, err)
	defer node.Stop()

	// Track handler calls
	messageReceived := false
	node.RegisterMessageHandler("test", func(peerID reputation.PeerID, msg []byte) error {
		messageReceived = true
		return nil
	})

	t.Run("accept message under size limit", func(t *testing.T) {
		messageReceived = false
		testPeer := reputation.PeerID("peer1")

		// Create message just under the limit (10MB)
		validData := make([]byte, MaxP2PMessageSize-1024)
		node.handlePeerMessage(testPeer, "test", validData)

		// Message should be processed
		require.True(t, messageReceived, "Message under size limit should be processed")
	})

	t.Run("reject message over size limit", func(t *testing.T) {
		messageReceived = false
		testPeer := reputation.PeerID("peer2")

		// Create message over the limit (10MB + 1KB)
		oversizedData := make([]byte, MaxP2PMessageSize+1024)
		node.handlePeerMessage(testPeer, "test", oversizedData)

		// Message should be rejected
		require.False(t, messageReceived, "Oversized message should be rejected")
	})

	t.Run("accept block message under block size limit", func(t *testing.T) {
		messageReceived = false
		testPeer := reputation.PeerID("peer3")

		// Register block handler
		node.RegisterMessageHandler("block", func(peerID reputation.PeerID, msg []byte) error {
			messageReceived = true
			return nil
		})

		// Create block message just under the block limit (21MB)
		validBlockData := make([]byte, MaxBlockSize-1024)
		node.handlePeerMessage(testPeer, "block", validBlockData)

		// Message should be processed
		require.True(t, messageReceived, "Block under size limit should be processed")
	})

	t.Run("reject block message over block size limit", func(t *testing.T) {
		messageReceived = false
		testPeer := reputation.PeerID("peer4")

		// Create block message over the block limit (21MB + 1KB)
		oversizedBlockData := make([]byte, MaxBlockSize+1024)
		node.handlePeerMessage(testPeer, "block", oversizedBlockData)

		// Message should be rejected
		require.False(t, messageReceived, "Oversized block should be rejected")
	})

	t.Run("exact size limit boundary", func(t *testing.T) {
		messageReceived = false
		testPeer := reputation.PeerID("peer5")

		// Create message exactly at the limit
		exactSizeData := make([]byte, MaxP2PMessageSize)
		node.handlePeerMessage(testPeer, "test", exactSizeData)

		// Message at exact limit should be processed
		require.True(t, messageReceived, "Message exactly at size limit should be processed")
	})
}

func TestMessageSizeLimitsWithReputation(t *testing.T) {
	// Create a test node with reputation enabled
	config := DefaultNodeConfig()
	config.DataDir = t.TempDir()
	config.NodeID = "test-node-2"
	config.EnableReputation = true

	logger := log.NewNopLogger()
	node, err := NewNode(config, logger)
	require.NoError(t, err)
	defer node.Stop()

	err = node.Start()
	require.NoError(t, err)

	testPeer := reputation.PeerID("malicious-peer")

	// Track handler calls
	messageReceived := false
	node.RegisterMessageHandler("test", func(peerID reputation.PeerID, msg []byte) error {
		messageReceived = true
		return nil
	})

	t.Run("oversized message records reputation event", func(t *testing.T) {
		messageReceived = false

		// Send oversized message
		oversizedData := make([]byte, MaxP2PMessageSize+10000)
		node.handlePeerMessage(testPeer, "test", oversizedData)

		// Message should be rejected
		require.False(t, messageReceived, "Oversized message should be rejected")

		// Verify reputation event was recorded
		if node.repManager != nil {
			// Give some time for async event processing
			time.Sleep(10 * time.Millisecond)

			rep, err := node.repManager.GetReputation(testPeer)
			if err == nil && rep != nil {
				// Reputation should be negatively impacted
				require.True(t, rep.Metrics.OversizedMessages > 0,
					"Oversized message should be recorded in reputation metrics")
			}
		}
	})

	t.Run("multiple oversized messages increase penalty", func(t *testing.T) {
		messageReceived = false

		// Send multiple oversized messages
		for i := 0; i < 3; i++ {
			oversizedData := make([]byte, MaxP2PMessageSize+10000)
			node.handlePeerMessage(testPeer, "test", oversizedData)
			time.Sleep(5 * time.Millisecond)
		}

		// All messages should be rejected
		require.False(t, messageReceived, "All oversized messages should be rejected")

		// Verify cumulative reputation penalty
		if node.repManager != nil {
			time.Sleep(20 * time.Millisecond)

			rep, err := node.repManager.GetReputation(testPeer)
			if err == nil && rep != nil {
				require.True(t, rep.Metrics.OversizedMessages >= 3,
					"Multiple violations should accumulate")
			}
		}
	})
}

func TestMessageSizeDoSProtection(t *testing.T) {
	// This test verifies that the message size check prevents DoS attacks
	// by rejecting messages before they consume excessive memory

	config := DefaultNodeConfig()
	config.DataDir = t.TempDir()
	config.NodeID = "test-node-3"
	config.EnableReputation = false

	logger := log.NewNopLogger()
	node, err := NewNode(config, logger)
	require.NoError(t, err)
	defer node.Stop()

	// Track memory-intensive handler that should never be called for oversized messages
	expensiveHandlerCalled := false
	node.RegisterMessageHandler("expensive", func(peerID reputation.PeerID, msg []byte) error {
		// Simulate expensive processing
		_ = make([]byte, len(msg)*2) // Would double memory usage
		expensiveHandlerCalled = true
		return nil
	})

	t.Run("prevent memory exhaustion attack", func(t *testing.T) {
		expensiveHandlerCalled = false
		testPeer := reputation.PeerID("attacker")

		// Attempt to send extremely large message (100MB)
		// This should be rejected before allocating memory for processing
		attackData := make([]byte, 100*1024*1024)
		node.handlePeerMessage(testPeer, "expensive", attackData)

		// Handler should never be called, preventing memory exhaustion
		require.False(t, expensiveHandlerCalled,
			"Expensive handler should not be called for oversized message")
	})

	t.Run("validate size before handler lookup", func(t *testing.T) {
		// Even messages without handlers should be size-checked
		testPeer := reputation.PeerID("unknown-peer")

		// Large message for nonexistent handler
		largeData := make([]byte, MaxP2PMessageSize+1)

		// Should not panic or cause issues
		require.NotPanics(t, func() {
			node.handlePeerMessage(testPeer, "nonexistent", largeData)
		})
	})
}

func TestMessageTypeSizeVariation(t *testing.T) {
	config := DefaultNodeConfig()
	config.DataDir = t.TempDir()
	config.NodeID = "test-node-4"
	config.EnableReputation = false

	logger := log.NewNopLogger()
	node, err := NewNode(config, logger)
	require.NoError(t, err)
	defer node.Stop()

	messageTypes := []struct {
		msgType     string
		shouldAllow bool
		size        int
	}{
		// Regular messages - 10MB limit
		{"tx", true, MaxP2PMessageSize - 1024},
		{"tx", false, MaxP2PMessageSize + 1024},

		// Block messages - 21MB limit
		{"block", true, MaxBlockSize - 1024},
		{"block", false, MaxBlockSize + 1024},
		{"block_announce", true, MaxBlockSize - 1024},
		{"block_announce", false, MaxBlockSize + 1024},
		{"block_response", true, MaxBlockSize - 1024},
		{"block_response", false, MaxBlockSize + 1024},

		// Other messages default to 10MB
		{"unknown", true, MaxP2PMessageSize - 1024},
		{"unknown", false, MaxP2PMessageSize + 1024},
	}

	for _, tc := range messageTypes {
		t.Run(tc.msgType, func(t *testing.T) {
			messageReceived := false

			// Register handler
			node.RegisterMessageHandler(tc.msgType, func(peerID reputation.PeerID, msg []byte) error {
				messageReceived = true
				return nil
			})

			// Send message
			data := make([]byte, tc.size)
			testPeer := reputation.PeerID("peer-" + tc.msgType)
			node.handlePeerMessage(testPeer, tc.msgType, data)

			// Verify expected behavior
			if tc.shouldAllow {
				require.True(t, messageReceived,
					"Message type %s with size %d should be allowed", tc.msgType, tc.size)
			} else {
				require.False(t, messageReceived,
					"Message type %s with size %d should be rejected", tc.msgType, tc.size)
			}
		})
	}
}
