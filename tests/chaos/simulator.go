package chaos

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

// NetworkSimulator simulates a network of blockchain nodes with partition capabilities
type NetworkSimulator struct {
	nodes       map[string]*Node
	connections map[string]map[string]bool // node1 -> node2 -> connected
	mu          sync.RWMutex
	latency     time.Duration
	packetLoss  float64
}

func NewNetworkSimulator() *NetworkSimulator {
	return &NetworkSimulator{
		nodes:       make(map[string]*Node),
		connections: make(map[string]map[string]bool),
		latency:     10 * time.Millisecond,
		packetLoss:  0.0,
	}
}

func (ns *NetworkSimulator) AddNode(node *Node) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.nodes[node.ID] = node
	ns.connections[node.ID] = make(map[string]bool)
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) ConnectAll() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for id1 := range ns.nodes {
		for id2 := range ns.nodes {
			if id1 != id2 {
				ns.connections[id1][id2] = true
			}
		}
	}
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) ConnectNodes(nodes ...*Node) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for _, n1 := range nodes {
		for _, n2 := range nodes {
			if n1.ID != n2.ID {
				ns.connections[n1.ID][n2.ID] = true
				ns.connections[n2.ID][n1.ID] = true
			}
		}
	}
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) DisconnectNodes(n1, n2 *Node) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.connections[n1.ID][n2.ID] = false
	ns.connections[n2.ID][n1.ID] = false
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) IsolateNode(node *Node) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for id := range ns.nodes {
		ns.connections[node.ID][id] = false
		ns.connections[id][node.ID] = false
	}
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) ReconnectNode(node *Node) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for id := range ns.nodes {
		if id != node.ID {
			ns.connections[node.ID][id] = true
			ns.connections[id][node.ID] = true
		}
	}
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) CreatePartition(partition1, partition2 []*Node) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	// Disconnect cross-partition links
	for _, n1 := range partition1 {
		for _, n2 := range partition2 {
			ns.connections[n1.ID][n2.ID] = false
			ns.connections[n2.ID][n1.ID] = false
		}
	}
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) CreateThreeWayPartition(p1, p2, p3 []*Node) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	partitions := [][]*Node{p1, p2, p3}

	for i := 0; i < len(partitions); i++ {
		for j := i + 1; j < len(partitions); j++ {
			for _, n1 := range partitions[i] {
				for _, n2 := range partitions[j] {
					ns.connections[n1.ID][n2.ID] = false
					ns.connections[n2.ID][n1.ID] = false
				}
			}
		}
	}
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) HealPartition() {
	ns.ConnectAll()
}

func (ns *NetworkSimulator) CreateAsymmetricLink(from, to *Node) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.connections[from.ID][to.ID] = true
	ns.connections[to.ID][from.ID] = false
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) RepairAsymmetricLink(from, to *Node) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	ns.connections[from.ID][to.ID] = true
	ns.connections[to.ID][from.ID] = true
	ns.refreshPeersLocked()
}

func (ns *NetworkSimulator) CrashNode(node *Node) {
	atomic.StoreUint32(&node.crashed, 1)
	ns.IsolateNode(node)
}

func (ns *NetworkSimulator) RecoverNode(node *Node) {
	atomic.StoreUint32(&node.crashed, 0)
	ns.ReconnectNode(node)
}

func (ns *NetworkSimulator) StabilizeNetwork(duration time.Duration) {
	ns.ConnectAll()
	time.Sleep(duration)
}

func (ns *NetworkSimulator) Shutdown() {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for _, node := range ns.nodes {
		atomic.StoreUint32(&node.crashed, 1)
	}
}

func (ns *NetworkSimulator) SetLatency(latency time.Duration) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	ns.latency = latency
}

func (ns *NetworkSimulator) SetPacketLoss(probability float64) {
	ns.mu.Lock()
	defer ns.mu.Unlock()
	switch {
	case probability < 0:
		ns.packetLoss = 0
	case probability > 1:
		ns.packetLoss = 1
	default:
		ns.packetLoss = probability
	}
}

func (ns *NetworkSimulator) IsConnected(from, to string) bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	return ns.connections[from][to]
}

func (ns *NetworkSimulator) refreshPeersLocked() {
	for id, node := range ns.nodes {
		connections := ns.connections[id]
		peers := make([]*Node, 0, len(connections))
		for otherID, connected := range connections {
			if connected {
				if peerNode, ok := ns.nodes[otherID]; ok {
					peers = append(peers, peerNode)
				}
			}
		}

		node.mu.Lock()
		node.peers = peers
		node.mu.Unlock()
	}
}

// Node represents a blockchain node in the simulation
type Node struct {
	ID           string
	network      *NetworkSimulator
	blockHeight  uint64
	transactions map[string]*Transaction
	blocks       []*Block
	state        map[string][]byte
	peers        []*Node
	crashed      uint32
	mu           sync.RWMutex
}

func NewNode(id string, network *NetworkSimulator) *Node {
	return &Node{
		ID:           id,
		network:      network,
		transactions: make(map[string]*Transaction),
		blocks:       make([]*Block, 0),
		state:        make(map[string][]byte),
		peers:        make([]*Node, 0),
	}
}

func (n *Node) SubmitTransaction(ctx context.Context, tx *Transaction) error {
	if atomic.LoadUint32(&n.crashed) == 1 {
		return fmt.Errorf("node crashed")
	}

	n.mu.Lock()
	n.transactions[tx.ID] = tx
	n.mu.Unlock()

	// Broadcast to peers
	go n.broadcastTransaction(ctx, tx)

	return nil
}

func (n *Node) HasTransaction(txID string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	_, exists := n.transactions[txID]
	return exists
}

func (n *Node) GetStateHash() string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	hasher := sha256.New()

	// Sort keys for determinism
	for key, value := range n.state {
		hasher.Write([]byte(key))
		hasher.Write(value)
	}

	return hex.EncodeToString(hasher.Sum(nil))
}

func (n *Node) broadcastTransaction(ctx context.Context, tx *Transaction) {
	n.network.mu.RLock()
	latency := n.network.latency
	packetLoss := n.network.packetLoss
	n.network.mu.RUnlock()

	for _, peer := range n.peers {
		if n.network.IsConnected(n.ID, peer.ID) {
			go func(p *Node) {
				if packetLoss > 0 && rand.Float64() < packetLoss {
					return
				}
				select {
				case <-ctx.Done():
					return
				case <-time.After(latency):
					p.receiveTransaction(tx)
				}
			}(peer)
		}
	}
}

func (n *Node) receiveTransaction(tx *Transaction) {
	if atomic.LoadUint32(&n.crashed) == 1 {
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	if _, exists := n.transactions[tx.ID]; !exists {
		n.transactions[tx.ID] = tx
	}
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string
	Data      []byte
	Timestamp time.Time
	Nonce     uint64
}

// Block represents a blockchain block
type Block struct {
	Height       uint64
	PreviousHash string
	Transactions []*Transaction
	Timestamp    time.Time
	Proposer     string
	StateHash    string
}
