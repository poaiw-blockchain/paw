// Package e2e_testnet provides end-to-end testing against live AURA testnet infrastructure.
package e2e_testnet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Client provides methods to interact with testnet nodes
type Client struct {
	config  *TestnetConfig
	httpCli *http.Client
}

// NewClient creates a new testnet client
func NewClient(cfg *TestnetConfig) *Client {
	return &Client{
		config: cfg,
		httpCli: &http.Client{
			Timeout: cfg.Timeout,
		},
	}
}

// RPCStatus represents the status response from a node
type RPCStatus struct {
	NodeInfo struct {
		Network string `json:"network"`
		Moniker string `json:"moniker"`
	} `json:"node_info"`
	SyncInfo struct {
		LatestBlockHeight string `json:"latest_block_height"`
		LatestBlockTime   string `json:"latest_block_time"`
		CatchingUp        bool   `json:"catching_up"`
	} `json:"sync_info"`
	ValidatorInfo struct {
		Address     string `json:"address"`
		VotingPower string `json:"voting_power"`
	} `json:"validator_info"`
}

// GetStatus fetches the status of a validator node
func (c *Client) GetStatus(ctx context.Context, val *ValidatorConfig) (*RPCStatus, error) {
	endpoint := val.GetRPCEndpoint() + "/status"

	// For remote hosts, use SSH tunnel
	if val.Host != "127.0.0.1" && val.Host != "localhost" {
		return c.getStatusViaSSH(ctx, val)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result RPCStatus `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result.Result, nil
}

// getStatusViaSSH fetches status from remote node via SSH
func (c *Client) getStatusViaSSH(ctx context.Context, val *ValidatorConfig) (*RPCStatus, error) {
	cmd := exec.CommandContext(ctx, "ssh", "-o", "ConnectTimeout=5", "-o", "BatchMode=yes",
		val.Host, fmt.Sprintf("curl -s http://127.0.0.1:%d/status", val.RPCPort))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("SSH command failed: %w", err)
	}

	var result struct {
		Result RPCStatus `json:"result"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return &result.Result, nil
}

// RESTQueryResult represents a generic REST query response
type RESTQueryResult map[string]interface{}

// QueryREST makes a REST API query
func (c *Client) QueryREST(ctx context.Context, val *ValidatorConfig, path string) (RESTQueryResult, error) {
	endpoint := val.GetRESTEndpoint() + path

	// For remote hosts, use SSH tunnel
	if val.Host != "127.0.0.1" && val.Host != "localhost" {
		return c.queryRESTViaSSH(ctx, val, path)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result RESTQueryResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// queryRESTViaSSH fetches REST data from remote node via SSH
func (c *Client) queryRESTViaSSH(ctx context.Context, val *ValidatorConfig, path string) (RESTQueryResult, error) {
	cmd := exec.CommandContext(ctx, "ssh", "-o", "ConnectTimeout=5", "-o", "BatchMode=yes",
		val.Host, fmt.Sprintf("curl -s http://127.0.0.1:%d%s", val.RESTPort, path))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("SSH command failed: %w", err)
	}

	var result RESTQueryResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// GetBlockHeight returns the current block height from a validator
func (c *Client) GetBlockHeight(ctx context.Context, val *ValidatorConfig) (int64, error) {
	status, err := c.GetStatus(ctx, val)
	if err != nil {
		return 0, err
	}

	var height int64
	fmt.Sscanf(status.SyncInfo.LatestBlockHeight, "%d", &height)
	return height, nil
}

// WaitForBlocks waits for n blocks to be produced
func (c *Client) WaitForBlocks(ctx context.Context, val *ValidatorConfig, n int) error {
	startHeight, err := c.GetBlockHeight(ctx, val)
	if err != nil {
		return err
	}

	targetHeight := startHeight + int64(n)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			height, err := c.GetBlockHeight(ctx, val)
			if err != nil {
				continue
			}
			if height >= targetHeight {
				return nil
			}
		}
	}
}

// RunCommand executes a command on the appropriate server
func (c *Client) RunCommand(ctx context.Context, val *ValidatorConfig, cmd string) (string, error) {
	if val.Host == "127.0.0.1" || val.Host == "localhost" {
		// Local execution
		command := exec.CommandContext(ctx, "bash", "-c", cmd)
		output, err := command.CombinedOutput()
		return strings.TrimSpace(string(output)), err
	}

	// Remote execution via SSH
	command := exec.CommandContext(ctx, "ssh", "-o", "ConnectTimeout=5", "-o", "BatchMode=yes",
		val.Host, cmd)
	output, err := command.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// GetBankSupply queries the total supply from the bank module
func (c *Client) GetBankSupply(ctx context.Context, val *ValidatorConfig) (RESTQueryResult, error) {
	return c.QueryREST(ctx, val, "/cosmos/bank/v1beta1/supply")
}

// GetValidators queries the staking validators
func (c *Client) GetValidators(ctx context.Context, val *ValidatorConfig) (RESTQueryResult, error) {
	return c.QueryREST(ctx, val, "/cosmos/staking/v1beta1/validators")
}

// GetNodeInfo queries the node info
func (c *Client) GetNodeInfo(ctx context.Context, val *ValidatorConfig) (RESTQueryResult, error) {
	return c.QueryREST(ctx, val, "/cosmos/base/tendermint/v1beta1/node_info")
}

// GetLatestBlock queries the latest block
func (c *Client) GetLatestBlock(ctx context.Context, val *ValidatorConfig) (RESTQueryResult, error) {
	return c.QueryREST(ctx, val, "/cosmos/base/tendermint/v1beta1/blocks/latest")
}
