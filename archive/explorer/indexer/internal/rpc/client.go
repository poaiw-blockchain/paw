package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Client provides RPC access to the blockchain node
type Client struct {
	rpcURL      string
	httpClient  *http.Client
	timeout     time.Duration
	maxRetries  int
	rateLimiter *time.Ticker
	mu          sync.Mutex
}

// Config holds RPC client configuration
type Config struct {
	RPCURL         string
	Timeout        time.Duration
	MaxRetries     int
	RequestsPerSec int
}

// BlockResponse represents an RPC block response
type BlockResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		BlockID struct {
			Hash  string `json:"hash"`
			Parts struct {
				Total int    `json:"total"`
				Hash  string `json:"hash"`
			} `json:"parts"`
		} `json:"block_id"`
		Block struct {
			Header struct {
				Version struct {
					Block string `json:"block"`
					App   string `json:"app"`
				} `json:"version"`
				ChainID     string    `json:"chain_id"`
				Height      string    `json:"height"`
				Time        time.Time `json:"time"`
				LastBlockID struct {
					Hash  string `json:"hash"`
					Parts struct {
						Total int    `json:"total"`
						Hash  string `json:"hash"`
					} `json:"parts"`
				} `json:"last_block_id"`
				LastCommitHash     string `json:"last_commit_hash"`
				DataHash           string `json:"data_hash"`
				ValidatorsHash     string `json:"validators_hash"`
				NextValidatorsHash string `json:"next_validators_hash"`
				ConsensusHash      string `json:"consensus_hash"`
				AppHash            string `json:"app_hash"`
				LastResultsHash    string `json:"last_results_hash"`
				EvidenceHash       string `json:"evidence_hash"`
				ProposerAddress    string `json:"proposer_address"`
			} `json:"header"`
			Data struct {
				Txs []string `json:"txs"`
			} `json:"data"`
			Evidence struct {
				Evidence []interface{} `json:"evidence"`
			} `json:"evidence"`
			LastCommit struct {
				Height     string        `json:"height"`
				Round      int           `json:"round"`
				BlockID    interface{}   `json:"block_id"`
				Signatures []interface{} `json:"signatures"`
			} `json:"last_commit"`
		} `json:"block"`
	} `json:"result"`
}

// BlockResultsResponse represents block execution results
type BlockResultsResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		Height                string        `json:"height"`
		TxsResults            []TxResult    `json:"txs_results"`
		BeginBlockEvents      []Event       `json:"begin_block_events"`
		EndBlockEvents        []Event       `json:"end_block_events"`
		ValidatorUpdates      []interface{} `json:"validator_updates"`
		ConsensusParamUpdates interface{}   `json:"consensus_param_updates"`
	} `json:"result"`
}

// TxResult represents a transaction execution result
type TxResult struct {
	Code      int     `json:"code"`
	Data      string  `json:"data"`
	Log       string  `json:"log"`
	Info      string  `json:"info"`
	GasWanted string  `json:"gas_wanted"`
	GasUsed   string  `json:"gas_used"`
	Events    []Event `json:"events"`
	Codespace string  `json:"codespace"`
}

// Event represents a blockchain event
type Event struct {
	Type       string      `json:"type"`
	Attributes []Attribute `json:"attributes"`
}

// Attribute represents an event attribute
type Attribute struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Index bool   `json:"index,omitempty"`
}

// StatusResponse represents the status RPC response
type StatusResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  struct {
		NodeInfo struct {
			ProtocolVersion struct {
				P2P   string `json:"p2p"`
				Block string `json:"block"`
				App   string `json:"app"`
			} `json:"protocol_version"`
			ID         string `json:"id"`
			ListenAddr string `json:"listen_addr"`
			Network    string `json:"network"`
			Version    string `json:"version"`
			Channels   string `json:"channels"`
			Moniker    string `json:"moniker"`
			Other      struct {
				TxIndex    string `json:"tx_index"`
				RPCAddress string `json:"rpc_address"`
			} `json:"other"`
		} `json:"node_info"`
		SyncInfo struct {
			LatestBlockHash     string    `json:"latest_block_hash"`
			LatestAppHash       string    `json:"latest_app_hash"`
			LatestBlockHeight   string    `json:"latest_block_height"`
			LatestBlockTime     time.Time `json:"latest_block_time"`
			EarliestBlockHash   string    `json:"earliest_block_hash"`
			EarliestAppHash     string    `json:"earliest_app_hash"`
			EarliestBlockHeight string    `json:"earliest_block_height"`
			EarliestBlockTime   time.Time `json:"earliest_block_time"`
			CatchingUp          bool      `json:"catching_up"`
		} `json:"sync_info"`
		ValidatorInfo struct {
			Address string `json:"address"`
			PubKey  struct {
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"pub_key"`
			VotingPower string `json:"voting_power"`
		} `json:"validator_info"`
	} `json:"result"`
}

// NewClient creates a new RPC client
func NewClient(cfg Config) (*Client, error) {
	if cfg.RPCURL == "" {
		return nil, fmt.Errorf("RPC URL is required")
	}

	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 3
	}

	if cfg.RequestsPerSec == 0 {
		cfg.RequestsPerSec = 10 // Default rate limit
	}

	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}

	rateLimiter := time.NewTicker(time.Second / time.Duration(cfg.RequestsPerSec))

	return &Client{
		rpcURL:      cfg.RPCURL,
		httpClient:  httpClient,
		timeout:     cfg.Timeout,
		maxRetries:  cfg.MaxRetries,
		rateLimiter: rateLimiter,
	}, nil
}

// GetChainHeight returns the current blockchain height
func (c *Client) GetChainHeight(ctx context.Context) (int64, error) {
	status, err := c.GetStatus(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get status: %w", err)
	}

	var height int64
	fmt.Sscanf(status.Result.SyncInfo.LatestBlockHeight, "%d", &height)

	return height, nil
}

// GetStatus returns the node status
func (c *Client) GetStatus(ctx context.Context) (*StatusResponse, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "status",
		"params":  []interface{}{},
	}

	var response StatusResponse
	if err := c.doRequest(ctx, reqBody, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetBlock fetches a block at the specified height
func (c *Client) GetBlock(ctx context.Context, height int64) (*BlockResponse, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "block",
		"params":  map[string]interface{}{"height": fmt.Sprintf("%d", height)},
	}

	var response BlockResponse
	if err := c.doRequest(ctx, reqBody, &response); err != nil {
		return nil, fmt.Errorf("failed to get block at height %d: %w", height, err)
	}

	return &response, nil
}

// GetBlockResults fetches block execution results
func (c *Client) GetBlockResults(ctx context.Context, height int64) (*BlockResultsResponse, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "block_results",
		"params":  map[string]interface{}{"height": fmt.Sprintf("%d", height)},
	}

	var response BlockResultsResponse
	if err := c.doRequest(ctx, reqBody, &response); err != nil {
		return nil, fmt.Errorf("failed to get block results at height %d: %w", height, err)
	}

	return &response, nil
}

// GetBlockBatch fetches multiple blocks in parallel
func (c *Client) GetBlockBatch(ctx context.Context, startHeight, endHeight int64) ([]*BlockData, error) {
	count := endHeight - startHeight + 1
	blocks := make([]*BlockData, count)
	errors := make([]error, count)

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrent requests

	for i := int64(0); i < count; i++ {
		wg.Add(1)
		go func(index int64) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			height := startHeight + index
			blockData, err := c.GetBlockWithResults(ctx, height)
			if err != nil {
				errors[index] = err
				log.Error().
					Err(err).
					Int64("height", height).
					Msg("Failed to fetch block in batch")
				return
			}

			blocks[index] = blockData
		}(i)
	}

	wg.Wait()

	// Check for errors
	for _, err := range errors {
		if err != nil {
			return nil, fmt.Errorf("batch fetch failed: %w", err)
		}
	}

	return blocks, nil
}

// GetBlockWithResults fetches both block and results in parallel
func (c *Client) GetBlockWithResults(ctx context.Context, height int64) (*BlockData, error) {
	var block *BlockResponse
	var results *BlockResultsResponse
	var blockErr, resultsErr error

	var wg sync.WaitGroup
	wg.Add(2)

	// Fetch block
	go func() {
		defer wg.Done()
		block, blockErr = c.GetBlock(ctx, height)
	}()

	// Fetch block results
	go func() {
		defer wg.Done()
		results, resultsErr = c.GetBlockResults(ctx, height)
	}()

	wg.Wait()

	if blockErr != nil {
		return nil, blockErr
	}
	if resultsErr != nil {
		return nil, resultsErr
	}

	// Combine block and results into BlockData
	return &BlockData{
		Block:   block,
		Results: results,
	}, nil
}

// BlockData combines block and execution results
type BlockData struct {
	Block   *BlockResponse
	Results *BlockResultsResponse
}

// doRequest executes an RPC request with retry logic
func (c *Client) doRequest(ctx context.Context, reqBody interface{}, response interface{}) error {
	var lastErr error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			log.Debug().
				Int("attempt", attempt+1).
				Dur("backoff", backoff).
				Msg("Retrying RPC request")

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Rate limiting
		select {
		case <-c.rateLimiter.C:
		case <-ctx.Done():
			return ctx.Err()
		}

		// Marshal request body
		jsonData, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", c.rpcURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		// Execute request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Msg("RPC request failed")
			continue
		}

		// Read response body
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = err
			log.Warn().
				Err(err).
				Int("attempt", attempt+1).
				Msg("Failed to read response body")
			continue
		}

		// Check HTTP status code
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
			log.Warn().
				Int("status", resp.StatusCode).
				Str("body", string(body)).
				Int("attempt", attempt+1).
				Msg("Non-200 status code")
			continue
		}

		// Parse response
		if err := json.Unmarshal(body, response); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}

		return nil
	}

	return fmt.Errorf("request failed after %d attempts: %w", c.maxRetries, lastErr)
}

// Close closes the RPC client
func (c *Client) Close() {
	if c.rateLimiter != nil {
		c.rateLimiter.Stop()
	}
	if c.httpClient != nil {
		c.httpClient.CloseIdleConnections()
	}
}

// Health checks if the RPC endpoint is healthy
func (c *Client) Health(ctx context.Context) error {
	_, err := c.GetStatus(ctx)
	return err
}
