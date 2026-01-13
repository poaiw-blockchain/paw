package faucet

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/paw-chain/paw/faucet/pkg/config"
	"github.com/paw-chain/paw/faucet/pkg/database"
)

// Service handles faucet operations
type Service struct {
	cfg    *config.Config
	db     *database.DB
	client *http.Client
}

// SendRequest represents a token send request
type SendRequest struct {
	Recipient string
	Amount    int64
	IPAddress string
}

// SendResponse represents a token send response
type SendResponse struct {
	TxHash    string
	Recipient string
	Amount    int64
}

// NodeStatus represents blockchain node status
type NodeStatus struct {
	NodeInfo struct {
		Network string `json:"network"`
		Version string `json:"version"`
	} `json:"node_info"`
	SyncInfo struct {
		LatestBlockHeight string `json:"latest_block_height"`
		CatchingUp        bool   `json:"catching_up"`
	} `json:"sync_info"`
}

// Balance represents account balance
type Balance struct {
	Balances []struct {
		Denom  string `json:"denom"`
		Amount string `json:"amount"`
	} `json:"balances"`
}

// NewService creates a new faucet service
func NewService(cfg *config.Config, db *database.DB) (*Service, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &Service{
		cfg:    cfg,
		db:     db,
		client: client,
	}, nil
}

const (
	// MaxRequestAmount is the maximum amount that can be requested in a single transaction
	// This prevents abuse and ensures fair distribution
	MaxRequestAmount = 10000000000 // 10,000 tokens (assuming 6 decimal places)

	// MinRequestAmount is the minimum amount that can be requested
	MinRequestAmount = 1000000 // 1 token (assuming 6 decimal places)
)

// SendTokens sends tokens to a recipient
func (s *Service) SendTokens(req *SendRequest) (*SendResponse, error) {
	log.WithFields(log.Fields{
		"recipient": req.Recipient,
		"amount":    req.Amount,
		"ip":        req.IPAddress,
	}).Info("Sending tokens")

	// Validate request amount bounds
	if req.Amount < MinRequestAmount {
		return nil, fmt.Errorf("request amount %d is below minimum %d", req.Amount, MinRequestAmount)
	}

	if req.Amount > MaxRequestAmount {
		return nil, fmt.Errorf("request amount %d exceeds maximum %d", req.Amount, MaxRequestAmount)
	}

	// Additional validation: ensure amount matches configured amount per request
	// This prevents clients from manipulating the amount field
	if req.Amount != s.cfg.AmountPerRequest {
		log.WithFields(log.Fields{
			"requested": req.Amount,
			"expected":  s.cfg.AmountPerRequest,
		}).Warn("Request amount does not match configured amount")
		return nil, fmt.Errorf("request amount must be %d", s.cfg.AmountPerRequest)
	}

	// Create database record
	dbReq, err := s.db.CreateRequest(req.Recipient, req.IPAddress, req.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to create request record: %w", err)
	}

	// Prepare transaction
	txData := map[string]interface{}{
		"chain_id": s.cfg.ChainID,
		"from":     s.cfg.FaucetAddress,
		"to":       req.Recipient,
		"amount": []map[string]string{
			{
				"denom":  s.cfg.Denom,
				"amount": fmt.Sprintf("%d", req.Amount),
			},
		},
		"gas":       fmt.Sprintf("%d", s.cfg.GasLimit),
		"gas_price": s.cfg.GasPrice,
		"memo":      s.cfg.TransactionMemo,
	}

	// Send transaction to node
	txHash, err := s.broadcastTransaction(txData)
	if err != nil {
		// Update request as failed
		if updateErr := s.db.UpdateRequestFailed(dbReq.ID, err.Error()); updateErr != nil {
			log.WithError(updateErr).Error("Failed to update request status")
		}
		return nil, fmt.Errorf("failed to broadcast transaction: %w", err)
	}

	// Update request as successful
	if err := s.db.UpdateRequestSuccess(dbReq.ID, txHash); err != nil {
		log.WithError(err).Error("Failed to update request status")
	}

	log.WithFields(log.Fields{
		"tx_hash":   txHash,
		"recipient": req.Recipient,
		"amount":    req.Amount,
	}).Info("Tokens sent successfully")

	return &SendResponse{
		TxHash:    txHash,
		Recipient: req.Recipient,
		Amount:    req.Amount,
	}, nil
}

// GetBalance returns the faucet account balance
func (s *Service) GetBalance() (int64, error) {
	return s.getBalanceForAddress(s.cfg.FaucetAddress)
}

// GetAddressBalance returns the balance for a specific address
func (s *Service) GetAddressBalance(address string) (int64, error) {
	return s.getBalanceForAddress(address)
}

func (s *Service) getBalanceForAddress(address string) (int64, error) {
	url := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", s.cfg.NodeREST, address)

	resp, err := s.client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to get balance: status %d, body: %s", resp.StatusCode, string(body))
	}

	var balance Balance
	if err := json.NewDecoder(resp.Body).Decode(&balance); err != nil {
		return 0, fmt.Errorf("failed to decode balance response: %w", err)
	}

	// Find the balance for our denom
	for _, b := range balance.Balances {
		if b.Denom == s.cfg.Denom {
			var amount int64
			fmt.Sscanf(b.Amount, "%d", &amount)
			return amount, nil
		}
	}

	return 0, nil
}

// GetNodeStatus returns the blockchain node status
func (s *Service) GetNodeStatus() (*NodeStatus, error) {
	url := fmt.Sprintf("%s/status", s.cfg.NodeRPC)

	resp, err := s.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get node status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get node status: status %d, body: %s", resp.StatusCode, string(body))
	}

	var status NodeStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &status, nil
}

// broadcastTransaction broadcasts a transaction using the pawd CLI
func (s *Service) broadcastTransaction(txData map[string]interface{}) (string, error) {
	// Extract transaction parameters
	recipient, ok := txData["to"].(string)
	if !ok {
		return "", fmt.Errorf("invalid recipient address")
	}

	amounts, ok := txData["amount"].([]map[string]string)
	if !ok || len(amounts) == 0 {
		return "", fmt.Errorf("invalid amount")
	}

	amount := amounts[0]["amount"] + amounts[0]["denom"]

	// Build pawd command
	args := []string{
		"tx", "bank", "send",
		s.cfg.FaucetKeyName,
		recipient,
		amount,
		"--keyring-backend", s.cfg.KeyringBackend,
		"--home", s.cfg.PawdHome,
		"--chain-id", s.cfg.ChainID,
		"--node", s.cfg.NodeRPC,
		"--fees", fmt.Sprintf("%d%s", s.cfg.GasLimit/10, s.cfg.Denom), // 10% of gas limit as fee
		"-y", // Auto-confirm
	}

	log.WithFields(log.Fields{
		"binary": s.cfg.PawdBinary,
		"args":   args,
	}).Debug("Executing pawd transaction")

	cmd := exec.Command(s.cfg.PawdBinary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute pawd: %w, output: %s", err, string(output))
	}

	// Parse output to extract txhash
	outputStr := string(output)
	log.WithField("output", outputStr).Debug("pawd output")

	// Look for txhash in output (format: txhash: HASH)
	txHashRegex := regexp.MustCompile(`txhash:\s*([A-Fa-f0-9]{64})`)
	matches := txHashRegex.FindStringSubmatch(outputStr)
	if len(matches) >= 2 {
		return matches[1], nil
	}

	// If no txhash found, check for error code
	if strings.Contains(outputStr, "code: 0") || strings.Contains(outputStr, `"code":0`) {
		// Transaction succeeded but couldn't extract hash
		// Try to find any 64-char hex string
		hexRegex := regexp.MustCompile(`[A-Fa-f0-9]{64}`)
		hexMatches := hexRegex.FindAllString(outputStr, -1)
		if len(hexMatches) > 0 {
			return hexMatches[len(hexMatches)-1], nil
		}
	}

	return "", fmt.Errorf("transaction may have failed or txhash not found in output: %s", outputStr)
}

// ValidateAddress validates a PAW address
func (s *Service) ValidateAddress(address string) error {
	if len(address) < 40 || len(address) > 65 {
		return fmt.Errorf("invalid address length")
	}

	// Support both mainnet (paw1) and testnet (pawtest1) prefixes
	validPrefixes := []string{"pawtest1", "paw1"}
	hasValidPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(address, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return fmt.Errorf("address must start with paw1 or pawtest1")
	}

	// Additional validation could be added here
	// For example, Bech32 validation

	return nil
}
