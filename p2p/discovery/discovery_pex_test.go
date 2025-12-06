package discovery

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/paw-chain/paw/p2p/reputation"
	"github.com/stretchr/testify/require"
)

func setupPEXService(t *testing.T) (*Service, func()) {
	t.Helper()

	cfg := DefaultDiscoveryConfig()
	cfg.EnablePEX = true
	cfg.MaxPeers = 10
	cfg.MaxInboundPeers = 5
	cfg.MaxOutboundPeers = 5
	cfg.MinOutboundPeers = 2

	svc, err := NewService(cfg, t.TempDir(), nil, log.NewNopLogger())
	require.NoError(t, err)

	cleanup := func() {
		require.NoError(t, svc.GetPeerManager().Close())
		require.NoError(t, svc.GetAddressBook().Close())
	}

	return svc, cleanup
}

func TestHandlePEXMessageAddsPeers(t *testing.T) {
	svc, cleanup := setupPEXService(t)
	defer cleanup()

	msg := pexMessage{
		Timestamp: time.Now(),
		Peers: []pexMessagePeer{
			{ID: "peer-one", Address: "203.0.113.10", Port: 26656},
			{ID: "peer-two", Address: "198.51.100.42", Port: 26657},
		},
	}

	payload, err := json.Marshal(msg)
	require.NoError(t, err)

	require.NoError(t, svc.HandlePEXMessage("seed-peer", payload))

	addrOne, exists := svc.addressBook.GetAddress(reputation.PeerID("peer-one"))
	require.True(t, exists)
	require.Equal(t, PeerSourcePEX, addrOne.Source)

	addrTwo, exists := svc.addressBook.GetAddress(reputation.PeerID("peer-two"))
	require.True(t, exists)
	require.Equal(t, PeerSourcePEX, addrTwo.Source)
}

func TestPerformPEXBroadcastsPayload(t *testing.T) {
	svc, cleanup := setupPEXService(t)
	defer cleanup()

	var (
		mu      sync.Mutex
		sent    = make(map[reputation.PeerID][]byte)
		sendErr error
	)

	svc.SetMessageSender(func(peerID reputation.PeerID, msgType string, data []byte) error {
		require.Equal(t, PEXMessageType, msgType)
		mu.Lock()
		sent[peerID] = append([]byte(nil), data...)
		err := sendErr
		mu.Unlock()
		return err
	})

	// Inject a connected peer
	peerID := reputation.PeerID("connected-peer")
	svc.peerManager.mu.Lock()
	svc.peerManager.peers[peerID] = &PeerConnection{
		PeerAddr: &PeerAddr{
			ID:      peerID,
			Address: "198.18.0.1",
			Port:    26656,
			Source:  PeerSourceBootstrap,
		},
	}
	svc.peerManager.mu.Unlock()

	// Seed a tried address for sharing
	sharedID := reputation.PeerID("shared-peer")
	svc.addressBook.mu.Lock()
	svc.addressBook.triedBucket[sharedID] = &PeerAddr{
		ID:       sharedID,
		Address:  "203.0.113.5",
		Port:     26657,
		Source:   PeerSourceBootstrap,
		LastSeen: time.Now(),
	}
	svc.addressBook.mu.Unlock()

	svc.performPEX()

	mu.Lock()
	payload, ok := sent[peerID]
	mu.Unlock()
	require.True(t, ok, "expected PEX payload to be broadcast")

	var decoded pexMessage
	require.NoError(t, json.Unmarshal(payload, &decoded))
	require.Len(t, decoded.Peers, 1)
	require.Equal(t, "shared-peer", decoded.Peers[0].ID)
}
