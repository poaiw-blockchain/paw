package discovery

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"cosmossdk.io/log"
	"github.com/paw-chain/paw/p2p/reputation"
)

// AddressBook manages known peer addresses using a bucket-based approach
// This is inspired by Bitcoin's addrman and libp2p's address book
type AddressBook struct {
	config  DiscoveryConfig
	logger  log.Logger
	dataDir string
	mu      sync.RWMutex

	// Address buckets
	newBucket   map[reputation.PeerID]*PeerAddr // Untried addresses
	triedBucket map[reputation.PeerID]*PeerAddr // Successfully connected addresses

	// Banned addresses
	banned map[reputation.PeerID]time.Time

	// Private peer IDs (won't be shared via PEX)
	privatePeers map[reputation.PeerID]bool

	// Statistics
	stats struct {
		totalAdded    uint64
		totalRemoved  uint64
		totalAttempts uint64
	}

	// Background tasks
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Entropy source (crypto/rand by default)
	randReader io.Reader
}

// NewAddressBook creates a new address book
func NewAddressBook(config DiscoveryConfig, dataDir string, logger log.Logger) (*AddressBook, error) {
	ab := &AddressBook{
		config:       config,
		logger:       logger,
		dataDir:      dataDir,
		newBucket:    make(map[reputation.PeerID]*PeerAddr),
		triedBucket:  make(map[reputation.PeerID]*PeerAddr),
		banned:       make(map[reputation.PeerID]time.Time),
		privatePeers: make(map[reputation.PeerID]bool),
		stopChan:     make(chan struct{}),
		randReader:   rand.Reader,
	}

	// Mark private peers
	for _, peerIDStr := range config.PrivatePeerIDs {
		ab.privatePeers[reputation.PeerID(peerIDStr)] = true
	}

	// Load existing address book
	if err := ab.load(); err != nil {
		logger.Warn("failed to load address book, starting fresh", "error", err)
	}

	// Start background persistence
	ab.wg.Add(1)
	go ab.backgroundPersistence()

	logger.Info("address book initialized",
		"new_addresses", len(ab.newBucket),
		"tried_addresses", len(ab.triedBucket))

	return ab, nil
}

// AddAddress adds a new peer address to the book
func (ab *AddressBook) AddAddress(addr *PeerAddr) error {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	// Check if banned
	if banExpiry, banned := ab.banned[addr.ID]; banned {
		if time.Now().Before(banExpiry) {
			return fmt.Errorf("peer is banned until %s", banExpiry)
		}
		// Ban expired, remove it
		delete(ab.banned, addr.ID)
	}

	// Check if already in tried bucket (already connected successfully)
	if existing, exists := ab.triedBucket[addr.ID]; exists {
		// Update last seen
		existing.LastSeen = time.Now()
		return nil
	}

	// Check if already in new bucket
	if existing, exists := ab.newBucket[addr.ID]; exists {
		// Update last seen
		existing.LastSeen = time.Now()
		existing.Source = addr.Source // Update source if better
		return nil
	}

	// Check capacity
	if len(ab.newBucket)+len(ab.triedBucket) >= ab.config.AddressBookSize {
		// Remove oldest from new bucket
		ab.evictOldest()
	}

	// Add to new bucket
	addr.FirstSeen = time.Now()
	addr.LastSeen = time.Now()
	ab.newBucket[addr.ID] = addr
	ab.stats.totalAdded++

	ab.logger.Debug("added address to book",
		"peer_id", addr.ID,
		"address", addr.NetAddr(),
		"source", addr.Source.String())

	return nil
}

// AddAddresses adds multiple addresses
func (ab *AddressBook) AddAddresses(addrs []*PeerAddr) {
	for _, addr := range addrs {
		if err := ab.AddAddress(addr); err != nil {
			ab.logger.Debug("failed to add address", "address", addr.NetAddr(), "error", err)
		}
	}
}

// MarkAttempt marks a dial attempt for an address
func (ab *AddressBook) MarkAttempt(peerID reputation.PeerID) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.stats.totalAttempts++

	// Check new bucket first
	if addr, exists := ab.newBucket[peerID]; exists {
		addr.Attempts++
		addr.LastDialed = time.Now()

		// Remove if too many failed attempts
		if addr.Attempts > 10 {
			delete(ab.newBucket, peerID)
			ab.stats.totalRemoved++
			ab.logger.Debug("removed address after too many attempts", "peer_id", peerID)
		}
		return
	}

	// Check tried bucket
	if addr, exists := ab.triedBucket[peerID]; exists {
		addr.Attempts++
		addr.LastDialed = time.Now()
	}
}

// MarkGood marks an address as successfully connected (moves to tried bucket)
func (ab *AddressBook) MarkGood(peerID reputation.PeerID) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	var addr *PeerAddr
	var exists bool

	// Remove from new bucket if present
	addr, exists = ab.newBucket[peerID]
	if exists {
		delete(ab.newBucket, peerID)
	} else {
		// Check if already in tried bucket
		addr, exists = ab.triedBucket[peerID]
		if !exists {
			// Not in either bucket, can't mark good
			return
		}
	}

	// Add to tried bucket
	addr.LastSeen = time.Now()
	addr.Attempts = 0 // Reset attempts
	ab.triedBucket[peerID] = addr

	ab.logger.Debug("marked address as good", "peer_id", peerID, "address", addr.NetAddr())
}

// MarkBad marks an address as bad (increases attempt count)
func (ab *AddressBook) MarkBad(peerID reputation.PeerID) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	// Check new bucket
	if addr, exists := ab.newBucket[peerID]; exists {
		addr.Attempts++
		if addr.Attempts > 5 {
			delete(ab.newBucket, peerID)
			ab.stats.totalRemoved++
		}
		return
	}

	// Check tried bucket
	if addr, exists := ab.triedBucket[peerID]; exists {
		addr.Attempts++
		// Don't remove from tried bucket easily, just track attempts
	}
}

// Ban bans a peer for a duration
func (ab *AddressBook) Ban(peerID reputation.PeerID, duration time.Duration) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.banned[peerID] = time.Now().Add(duration)

	// Remove from both buckets
	delete(ab.newBucket, peerID)
	delete(ab.triedBucket, peerID)

	ab.logger.Info("banned peer", "peer_id", peerID, "duration", duration)
}

// Unban removes a ban
func (ab *AddressBook) Unban(peerID reputation.PeerID) {
	ab.mu.Lock()
	defer ab.mu.Unlock()

	delete(ab.banned, peerID)
	ab.logger.Info("unbanned peer", "peer_id", peerID)
}

// IsBanned checks if a peer is banned
func (ab *AddressBook) IsBanned(peerID reputation.PeerID) bool {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	banExpiry, banned := ab.banned[peerID]
	if !banned {
		return false
	}

	if time.Now().After(banExpiry) {
		// Ban expired
		return false
	}

	return true
}

// GetAddress retrieves an address by peer ID
func (ab *AddressBook) GetAddress(peerID reputation.PeerID) (*PeerAddr, bool) {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	// Check tried bucket first
	if addr, exists := ab.triedBucket[peerID]; exists {
		return addr, true
	}

	// Check new bucket
	if addr, exists := ab.newBucket[peerID]; exists {
		return addr, true
	}

	return nil, false
}

// GetRandomAddresses returns N random addresses for dialing
// Prioritizes tried addresses over new addresses (85/15 split)
func (ab *AddressBook) GetRandomAddresses(n int, filter PeerFilter) []*PeerAddr {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	result := make([]*PeerAddr, 0, n)

	// Calculate split (85% tried, 15% new)
	nTried := (n * 85) / 100
	nNew := n - nTried

	// Get from tried bucket
	triedAddrs := make([]*PeerAddr, 0, len(ab.triedBucket))
	for _, addr := range ab.triedBucket {
		if filter != nil && !filter(addr) {
			continue
		}
		triedAddrs = append(triedAddrs, addr)
	}

	// Shuffle and take nTried
	ab.shuffleAddrs(triedAddrs)
	for i := 0; i < nTried && i < len(triedAddrs); i++ {
		result = append(result, triedAddrs[i])
	}

	// Get from new bucket
	newAddrs := make([]*PeerAddr, 0, len(ab.newBucket))
	for _, addr := range ab.newBucket {
		if filter != nil && !filter(addr) {
			continue
		}
		newAddrs = append(newAddrs, addr)
	}

	// Shuffle and take nNew
	ab.shuffleAddrs(newAddrs)
	for i := 0; i < nNew && i < len(newAddrs); i++ {
		result = append(result, newAddrs[i])
	}

	return result
}

// GetBestAddresses returns the best N addresses based on score
func (ab *AddressBook) GetBestAddresses(n int, filter PeerFilter) []*PeerAddr {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	now := time.Now()

	// Collect all addresses with scores
	type scoredAddr struct {
		addr  *PeerAddr
		score float64
	}

	scored := make([]scoredAddr, 0, len(ab.triedBucket)+len(ab.newBucket))

	for _, addr := range ab.triedBucket {
		if filter != nil && !filter(addr) {
			continue
		}
		scored = append(scored, scoredAddr{addr, addr.PeerScore(now)})
	}

	for _, addr := range ab.newBucket {
		if filter != nil && !filter(addr) {
			continue
		}
		scored = append(scored, scoredAddr{addr, addr.PeerScore(now)})
	}

	// Sort by score (bubble sort for small lists)
	for i := 0; i < len(scored)-1; i++ {
		for j := 0; j < len(scored)-i-1; j++ {
			if scored[j].score < scored[j+1].score {
				scored[j], scored[j+1] = scored[j+1], scored[j]
			}
		}
	}

	// Take top N
	result := make([]*PeerAddr, 0, n)
	for i := 0; i < n && i < len(scored); i++ {
		result = append(result, scored[i].addr)
	}

	return result
}

// GetAddressesForSharing returns addresses suitable for sharing via PEX
// Excludes private peers and local addresses
func (ab *AddressBook) GetAddressesForSharing(n int) []*PeerAddr {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	candidates := make([]*PeerAddr, 0)

	// Only share from tried bucket (successfully connected peers)
	for _, addr := range ab.triedBucket {
		// Skip private peers
		if ab.privatePeers[addr.ID] {
			continue
		}

		// Skip non-routable addresses
		if !addr.IsRoutable() {
			continue
		}

		candidates = append(candidates, addr)
	}

	// Shuffle and take N
	ab.shuffleAddrs(candidates)

	if n > len(candidates) {
		n = len(candidates)
	}

	return candidates[:n]
}

// Size returns the total number of addresses
func (ab *AddressBook) Size() (new, tried int) {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	return len(ab.newBucket), len(ab.triedBucket)
}

// Stats returns address book statistics
func (ab *AddressBook) Stats() map[string]interface{} {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	return map[string]interface{}{
		"new_addresses":   len(ab.newBucket),
		"tried_addresses": len(ab.triedBucket),
		"total_addresses": len(ab.newBucket) + len(ab.triedBucket),
		"banned_peers":    len(ab.banned),
		"total_added":     ab.stats.totalAdded,
		"total_removed":   ab.stats.totalRemoved,
		"total_attempts":  ab.stats.totalAttempts,
	}
}

// Close shuts down the address book
func (ab *AddressBook) Close() error {
	ab.logger.Info("closing address book")

	// Stop background tasks
	close(ab.stopChan)
	ab.wg.Wait()

	// Final save
	return ab.save()
}

// Internal methods

// evictOldest removes the oldest entry from new bucket
func (ab *AddressBook) evictOldest() {
	var oldest *PeerAddr
	var oldestID reputation.PeerID

	for id, addr := range ab.newBucket {
		if oldest == nil || addr.FirstSeen.Before(oldest.FirstSeen) {
			oldest = addr
			oldestID = id
		}
	}

	if oldest != nil {
		delete(ab.newBucket, oldestID)
		ab.stats.totalRemoved++
		ab.logger.Debug("evicted oldest address", "peer_id", oldestID)
	}
}

// shuffleAddrs shuffles a slice of peer addresses using cryptographic randomness.
func (ab *AddressBook) shuffleAddrs(addrs []*PeerAddr) {
	for i := len(addrs) - 1; i > 0; i-- {
		j, err := ab.secureIntn(i + 1)
		if err != nil {
			ab.logger.Error("failed to get secure randomness for shuffle", "error", err)
			// Fallback: swap with self to avoid biased ordering when entropy is unavailable.
			j = i
		}
		addrs[i], addrs[j] = addrs[j], addrs[i]
	}
}

// secureIntn returns a cryptographically secure random integer in [0, n).
func (ab *AddressBook) secureIntn(n int) (int, error) {
	if n <= 0 {
		return 0, nil
	}

	max := big.NewInt(int64(n))
	r, err := rand.Int(ab.randReader, max)
	if err != nil {
		return 0, fmt.Errorf("crypto rand failure: %w", err)
	}

	return int(r.Int64()), nil
}

// save persists the address book to disk
func (ab *AddressBook) save() error {
	ab.mu.RLock()
	defer ab.mu.RUnlock()

	data := struct {
		NewBucket   map[reputation.PeerID]*PeerAddr `json:"new_bucket"`
		TriedBucket map[reputation.PeerID]*PeerAddr `json:"tried_bucket"`
		Banned      map[reputation.PeerID]time.Time `json:"banned"`
		SavedAt     time.Time                       `json:"saved_at"`
	}{
		NewBucket:   ab.newBucket,
		TriedBucket: ab.triedBucket,
		Banned:      ab.banned,
		SavedAt:     time.Now(),
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal address book: %w", err)
	}

	filePath := filepath.Join(ab.dataDir, "address_book.json")

	// Ensure directory exists
	if err := os.MkdirAll(ab.dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to temporary file first
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write address book: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, filePath); err != nil {
		return fmt.Errorf("failed to rename address book: %w", err)
	}

	return nil
}

// load loads the address book from disk
func (ab *AddressBook) load() error {
	filePath := filepath.Join(ab.dataDir, "address_book.json")

	data, err := os.ReadFile(filePath) // #nosec G304 - address book path resides under node data directory
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing address book
		}
		return fmt.Errorf("failed to read address book: %w", err)
	}

	var loaded struct {
		NewBucket   map[reputation.PeerID]*PeerAddr `json:"new_bucket"`
		TriedBucket map[reputation.PeerID]*PeerAddr `json:"tried_bucket"`
		Banned      map[reputation.PeerID]time.Time `json:"banned"`
		SavedAt     time.Time                       `json:"saved_at"`
	}

	if err := json.Unmarshal(data, &loaded); err != nil {
		return fmt.Errorf("failed to unmarshal address book: %w", err)
	}

	ab.mu.Lock()
	defer ab.mu.Unlock()

	ab.newBucket = loaded.NewBucket
	ab.triedBucket = loaded.TriedBucket
	ab.banned = loaded.Banned

	// Clean expired bans
	now := time.Now()
	for peerID, expiry := range ab.banned {
		if now.After(expiry) {
			delete(ab.banned, peerID)
		}
	}

	ab.logger.Info("loaded address book",
		"new", len(ab.newBucket),
		"tried", len(ab.triedBucket),
		"saved_at", loaded.SavedAt)

	return nil
}

// backgroundPersistence periodically saves the address book
func (ab *AddressBook) backgroundPersistence() {
	defer ab.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := ab.save(); err != nil {
				ab.logger.Error("failed to save address book", "error", err)
			}

		case <-ab.stopChan:
			return
		}
	}
}
