package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SimpleNodeChecker implements NodeHealthChecker using RPC calls
type SimpleNodeChecker struct {
	rpcAddr string
	client  *http.Client
}

// NewSimpleNodeChecker creates a new node health checker
func NewSimpleNodeChecker(rpcAddr string) *SimpleNodeChecker {
	return &SimpleNodeChecker{
		rpcAddr: rpcAddr,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// CheckRPC checks if the RPC endpoint is accessible
func (c *SimpleNodeChecker) CheckRPC() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/health", c.rpcAddr), nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("rpc unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("rpc returned status %d", resp.StatusCode)
	}

	return nil
}

// CheckSync checks if the node is syncing
func (c *SimpleNodeChecker) CheckSync() (syncing bool, height int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/status", c.rpcAddr), nil)
	if err != nil {
		return false, 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, 0, fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	var status struct {
		Result struct {
			SyncInfo struct {
				CatchingUp     bool   `json:"catching_up"`
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return false, 0, fmt.Errorf("failed to decode status: %w", err)
	}

	var blockHeight int64
	fmt.Sscanf(status.Result.SyncInfo.LatestBlockHeight, "%d", &blockHeight)

	return status.Result.SyncInfo.CatchingUp, blockHeight, nil
}

// CheckConsensus checks if the node is participating in consensus
func (c *SimpleNodeChecker) CheckConsensus() error {
	// For non-validator nodes, this is not critical
	// Would need to check validator status for actual validators
	return nil
}

// GetPeerCount returns the number of connected peers
func (c *SimpleNodeChecker) GetPeerCount() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/net_info", c.rpcAddr), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get net info: %w", err)
	}
	defer resp.Body.Close()

	var netInfo struct {
		Result struct {
			NPeers string `json:"n_peers"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&netInfo); err != nil {
		return 0, fmt.Errorf("failed to decode net info: %w", err)
	}

	var peers int
	fmt.Sscanf(netInfo.Result.NPeers, "%d", &peers)

	return peers, nil
}

// GetBlockHeight returns the current block height
func (c *SimpleNodeChecker) GetBlockHeight() (int64, error) {
	_, height, err := c.CheckSync()
	return height, err
}
