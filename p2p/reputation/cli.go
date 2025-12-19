package reputation

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"cosmossdk.io/log"
)

// CLI provides command-line interface for reputation management
type CLI struct {
	manager *Manager
	monitor *Monitor
	logger  log.Logger
}

// NewCLI creates a new CLI instance
func NewCLI(homeDir string, logger log.Logger) (*CLI, error) {
	// Load configuration
	config := DefaultConfig(homeDir)
	configPath := homeDir + "/config/p2p_security.toml"

	if cfg, err := LoadConfig(configPath); err == nil {
		config = *cfg
	}

	// Create storage
	storageConfig := DefaultFileStorageConfig(homeDir)
	storage, err := NewFileStorage(storageConfig, logger)
	if err != nil {
		return nil, err
	}

	// Create manager
	scoringConfig := config.Scoring.ToScoringConfig()
	scoreWeights := config.Scoring.ToScoreWeights()

	managerConfig := config.Manager.ToManagerConfig(
		&scoringConfig,
		&scoreWeights,
	)

	manager, err := NewManager(storage, &managerConfig, logger)
	if err != nil {
		return nil, err
	}

	// Create monitor
	metrics := NewMetrics()
	monitorConfig := DefaultMonitorConfig()
	monitor := NewMonitor(manager, metrics, monitorConfig, logger)

	return &CLI{
		manager: manager,
		monitor: monitor,
		logger:  logger,
	}, nil
}

// ListPeers displays all peers with their reputation scores
func (c *CLI) ListPeers(minScore float64, maxResults int) error {
	c.manager.peersMu.RLock()
	defer c.manager.peersMu.RUnlock()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PEER ID\tSCORE\tTRUST LEVEL\tSTATUS\tLAST SEEN\tADDRESS")
	fmt.Fprintln(w, "-------\t-----\t-----------\t------\t---------\t-------")

	count := 0
	for _, rep := range c.manager.peers {
		if rep.Score < minScore {
			continue
		}

		status := "Active"
		if rep.BanStatus.IsBanned {
			status = "Banned"
		}

		lastSeen := rep.LastSeen.Format("2006-01-02 15:04")

		fmt.Fprintf(w, "%s\t%.1f\t%s\t%s\t%s\t%s\n",
			truncatePeerID(string(rep.PeerID)),
			rep.Score,
			rep.TrustLevel.String(),
			status,
			lastSeen,
			rep.Address,
		)

		count++
		if maxResults > 0 && count >= maxResults {
			break
		}
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush peer list: %w", err)
	}
	return nil
}

// ShowPeer displays detailed information about a peer
func (c *CLI) ShowPeer(peerID string) error {
	rep, err := c.manager.GetReputation(PeerID(peerID))
	if err != nil {
		return err
	}

	if rep == nil {
		return fmt.Errorf("peer not found: %s", peerID)
	}

	fmt.Println("=== Peer Reputation ===")
	fmt.Printf("Peer ID: %s\n", rep.PeerID)
	fmt.Printf("Address: %s\n", rep.Address)
	fmt.Printf("Score: %.2f / 100\n", rep.Score)
	fmt.Printf("Trust Level: %s\n", rep.TrustLevel.String())
	fmt.Printf("First Seen: %s\n", rep.FirstSeen.Format(time.RFC3339))
	fmt.Printf("Last Seen: %s\n", rep.LastSeen.Format(time.RFC3339))

	fmt.Println("\n=== Network Info ===")
	fmt.Printf("IP Address: %s\n", rep.NetworkInfo.IPAddress)
	fmt.Printf("Subnet: %s\n", rep.NetworkInfo.Subnet)
	fmt.Printf("Country: %s\n", rep.NetworkInfo.Country)
	fmt.Printf("ASN: %d\n", rep.NetworkInfo.ASN)

	fmt.Println("\n=== Metrics ===")
	fmt.Printf("Connections: %d\n", rep.Metrics.ConnectionCount)
	fmt.Printf("Disconnections: %d\n", rep.Metrics.DisconnectionCount)
	fmt.Printf("Total Uptime: %s\n", rep.Metrics.TotalUptime)
	fmt.Printf("Valid Messages: %d / %d (%.1f%%)\n",
		rep.Metrics.ValidMessages,
		rep.Metrics.TotalMessages,
		rep.Metrics.ValidMessageRatio*100,
	)
	fmt.Printf("Avg Latency: %s\n", rep.Metrics.AvgResponseLatency)
	fmt.Printf("Blocks Propagated: %d (%d fast)\n",
		rep.Metrics.BlocksPropagated,
		rep.Metrics.FastBlockCount,
	)
	fmt.Printf("Protocol Violations: %d\n", rep.Metrics.ProtocolViolations)
	fmt.Printf("Spam Attempts: %d\n", rep.Metrics.SpamAttempts)

	fmt.Println("\n=== Ban Status ===")
	fmt.Printf("Banned: %t\n", rep.BanStatus.IsBanned)
	if rep.BanStatus.IsBanned {
		fmt.Printf("Ban Type: %s\n", rep.BanStatus.BanType.String())
		fmt.Printf("Banned At: %s\n", rep.BanStatus.BannedAt.Format(time.RFC3339))
		fmt.Printf("Ban Reason: %s\n", rep.BanStatus.BanReason)
		if rep.BanStatus.BanType == BanTypeTemporary {
			fmt.Printf("Ban Expires: %s\n", rep.BanStatus.BanExpires.Format(time.RFC3339))
		}
		fmt.Printf("Ban Count: %d\n", rep.BanStatus.BanCount)
	}
	fmt.Printf("Whitelisted: %t\n", rep.BanStatus.IsWhitelisted)

	fmt.Println("\n=== Recent Score History ===")
	if len(rep.Metrics.RecentScores) > 0 {
		count := len(rep.Metrics.RecentScores)
		if count > 10 {
			count = 10
		}
		for i := len(rep.Metrics.RecentScores) - count; i < len(rep.Metrics.RecentScores); i++ {
			snapshot := rep.Metrics.RecentScores[i]
			fmt.Printf("%s: %.1f (%s)\n",
				snapshot.Timestamp.Format("2006-01-02 15:04:05"),
				snapshot.Score,
				snapshot.Reason,
			)
		}
	}

	return nil
}

// ShowStats displays overall reputation statistics
func (c *CLI) ShowStats() error {
	stats := c.manager.GetStatistics()

	fmt.Println("=== Reputation System Statistics ===")
	fmt.Printf("Total Peers: %d\n", stats.TotalPeers)
	fmt.Printf("Banned Peers: %d\n", stats.BannedPeers)
	fmt.Printf("Whitelisted Peers: %d\n", stats.WhitelistedPeers)
	fmt.Printf("Average Score: %.2f\n", stats.AvgScore)

	fmt.Println("\n=== Score Distribution ===")
	for bucket, count := range stats.ScoreDistribution {
		fmt.Printf("%s: %d peers\n", bucket, count)
	}

	fmt.Println("\n=== Trust Distribution ===")
	for level, count := range stats.TrustDistribution {
		fmt.Printf("%s: %d peers\n", level, count)
	}

	health := c.monitor.GetHealth()
	fmt.Println("\n=== System Health ===")
	fmt.Printf("Status: %s\n", healthStatus(health.Healthy))
	fmt.Printf("Last Check: %s\n", health.LastCheck.Format(time.RFC3339))
	if len(health.Issues) > 0 {
		fmt.Println("Issues:")
		for _, issue := range health.Issues {
			fmt.Printf("  - %s\n", issue)
		}
	}

	return nil
}

// BanPeer bans a peer
func (c *CLI) BanPeer(peerID string, duration time.Duration, reason string) error {
	if reason == "" {
		reason = "Manual ban via CLI"
	}

	if err := c.manager.BanPeer(PeerID(peerID), duration, reason); err != nil {
		return err
	}

	banType := "permanent"
	if duration > 0 {
		banType = fmt.Sprintf("temporary (%s)", duration)
	}

	fmt.Printf("✓ Peer %s banned (%s)\n", truncatePeerID(peerID), banType)
	fmt.Printf("  Reason: %s\n", reason)

	return nil
}

// UnbanPeer unbans a peer
func (c *CLI) UnbanPeer(peerID string) error {
	if err := c.manager.UnbanPeer(PeerID(peerID)); err != nil {
		return err
	}

	fmt.Printf("✓ Peer %s unbanned\n", truncatePeerID(peerID))
	return nil
}

// WhitelistPeer adds a peer to whitelist
func (c *CLI) WhitelistPeer(peerID string) error {
	c.manager.AddToWhitelist(PeerID(peerID))
	fmt.Printf("✓ Peer %s added to whitelist\n", truncatePeerID(peerID))
	return nil
}

// UnwhitelistPeer removes a peer from whitelist
func (c *CLI) UnwhitelistPeer(peerID string) error {
	c.manager.RemoveFromWhitelist(PeerID(peerID))
	fmt.Printf("✓ Peer %s removed from whitelist\n", truncatePeerID(peerID))
	return nil
}

// ExportJSON exports reputation data to JSON
func (c *CLI) ExportJSON(outputPath string) error {
	c.manager.peersMu.RLock()
	defer c.manager.peersMu.RUnlock()

	data := make(map[string]*PeerReputation)
	for id, rep := range c.manager.peers {
		data[string(id)] = rep
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(outputPath, jsonData, 0600); err != nil {
		return err
	}

	fmt.Printf("✓ Exported reputation data to %s\n", outputPath)
	fmt.Printf("  Peers exported: %d\n", len(data))

	return nil
}

// ShowTopPeers displays top-ranked peers
func (c *CLI) ShowTopPeers(count int, minScore float64) error {
	peers := c.manager.GetTopPeers(count, minScore)

	if len(peers) == 0 {
		fmt.Println("No peers found matching criteria")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "RANK\tPEER ID\tSCORE\tTRUST LEVEL\tUPTIME\tMSGS (VALID%)\tLATENCY")
	fmt.Fprintln(w, "----\t-------\t-----\t-----------\t------\t------------\t-------")

	for i, rep := range peers {
		fmt.Fprintf(w, "%d\t%s\t%.1f\t%s\t%s\t%d (%.1f%%)\t%s\n",
			i+1,
			truncatePeerID(string(rep.PeerID)),
			rep.Score,
			rep.TrustLevel.String(),
			formatDuration(rep.Metrics.TotalUptime),
			rep.Metrics.TotalMessages,
			rep.Metrics.ValidMessageRatio*100,
			formatDuration(rep.Metrics.AvgResponseLatency),
		)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("failed to flush top peers table: %w", err)
	}
	return nil
}

// Close closes the CLI and underlying manager
func (c *CLI) Close() error {
	if err := c.monitor.Close(); err != nil {
		c.logger.Error("error closing monitor", "error", err)
	}
	return c.manager.Close()
}

// Helper functions

func truncatePeerID(peerID string) string {
	if len(peerID) <= 12 {
		return peerID
	}
	return peerID[:6] + "..." + peerID[len(peerID)-6:]
}

func healthStatus(healthy bool) string {
	if healthy {
		return "✓ Healthy"
	}
	return "✗ Unhealthy"
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0fs", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0fm", d.Minutes())
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	} else {
		return fmt.Sprintf("%.1fd", d.Hours()/24)
	}
}
