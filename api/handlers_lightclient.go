package api

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// handleGetHeaders returns block headers for light client sync
func (s *Server) handleGetHeaders(c *gin.Context) {
	// Parse query parameters
	startHeight := int64(1)
	limit := 20

	if h := c.Query("start_height"); h != "" {
		if parsed, err := strconv.ParseInt(h, 10, 64); err == nil {
			startHeight = parsed
		}
	}

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	// Get headers from blockchain
	headers, err := s.getBlockHeaders(startHeight, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to fetch headers",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"headers":      headers,
		"start_height": startHeight,
		"count":        len(headers),
	})
}

// handleGetHeaderByHeight returns a specific block header
func (s *Server) handleGetHeaderByHeight(c *gin.Context) {
	heightStr := c.Param("height")
	height, err := strconv.ParseInt(heightStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid height",
			Details: err.Error(),
		})
		return
	}

	header, err := s.getBlockHeader(height)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Header not found",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, header)
}

// handleGetCheckpoint returns a trusted checkpoint for light client initialization
func (s *Server) handleGetCheckpoint(c *gin.Context) {
	// Get latest height or specific height
	var height int64
	if h := c.Query("height"); h != "" {
		if parsed, err := strconv.ParseInt(h, 10, 64); err == nil {
			height = parsed
		}
	}

	// If no height specified, get latest
	if height == 0 {
		latestHeight, err := s.getLatestHeight()
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "Failed to get latest height",
				Details: err.Error(),
			})
			return
		}
		height = latestHeight
	}

	checkpoint, err := s.getCheckpoint(height)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to get checkpoint",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, checkpoint)
}

// handleGetTxProof returns a Merkle proof for a transaction
func (s *Server) handleGetTxProof(c *gin.Context) {
	txHash := c.Param("txid")

	if txHash == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Transaction hash is required",
		})
		return
	}

	proof, err := s.getTxProof(txHash)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Transaction proof not found",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, proof)
}

// handleVerifyProof verifies a transaction proof
func (s *Server) handleVerifyProof(c *gin.Context) {
	var req VerifyProofRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Details: err.Error(),
		})
		return
	}

	// Verify the proof
	verified, err := s.verifyTxProof(req.TxHash, req.Height, req.Proof, req.BlockHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Proof verification failed",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"verified":   verified,
		"tx_hash":    req.TxHash,
		"height":     req.Height,
		"block_hash": req.BlockHash,
	})
}

// getBlockHeaders retrieves multiple block headers
func (s *Server) getBlockHeaders(startHeight int64, limit int) ([]HeaderResponse, error) {
	headers := make([]HeaderResponse, 0, limit)

	// In production, query from CometBFT node
	// For now, generate mock headers
	for i := int64(0); i < int64(limit); i++ {
		height := startHeight + i
		headers = append(headers, HeaderResponse{
			Height:   height,
			Hash:     fmt.Sprintf("0x%064d", height),
			Time:     time.Now().Add(-time.Duration(limit-int(i)) * 4 * time.Second),
			ChainID:  s.config.ChainID,
			Proposer: "pawvaloper1abc...",
			LastHash: fmt.Sprintf("0x%064d", height-1),
			DataHash: fmt.Sprintf("0xdata%060d", height),
		})
	}

	return headers, nil
}

// getBlockHeader retrieves a single block header
func (s *Server) getBlockHeader(height int64) (*HeaderResponse, error) {
	// In production, query from CometBFT node using:
	// client := tmrpc.NewHTTP(s.config.NodeURI, "/websocket")
	// result, err := client.Block(context.Background(), &height)

	// For now, return mock header
	return &HeaderResponse{
		Height:   height,
		Hash:     fmt.Sprintf("0x%064d", height),
		Time:     time.Now(),
		ChainID:  s.config.ChainID,
		Proposer: "pawvaloper1abc...",
		LastHash: fmt.Sprintf("0x%064d", height-1),
		DataHash: fmt.Sprintf("0xdata%060d", height),
	}, nil
}

// getLatestHeight gets the latest block height
func (s *Server) getLatestHeight() (int64, error) {
	// In production, query from node
	// For now, return mock height
	return 1000, nil
}

// getCheckpoint retrieves a checkpoint for light client
func (s *Server) getCheckpoint(height int64) (*CheckpointResponse, error) {
	// In production, this would:
	// 1. Get block at height
	// 2. Get validator set
	// 3. Compute validator set hash
	// 4. Return checkpoint with signatures

	header, err := s.getBlockHeader(height)
	if err != nil {
		return nil, err
	}

	return &CheckpointResponse{
		Height:           height,
		Hash:             header.Hash,
		ValidatorSetHash: fmt.Sprintf("0xvalset%057d", height),
		Timestamp:        header.Time,
		TrustedHeight:    height - 100, // Previous trusted checkpoint
	}, nil
}

// getTxProof retrieves a Merkle proof for a transaction
func (s *Server) getTxProof(txHash string) (*TxProofResponse, error) {
	// In production, this would:
	// 1. Query transaction by hash
	// 2. Get block containing the transaction
	// 3. Generate Merkle proof
	// 4. Return proof with transaction data

	// Decode hex hash
	_, err := hex.DecodeString(txHash)
	if err != nil {
		return nil, fmt.Errorf("invalid transaction hash: %w", err)
	}

	// Mock proof for demonstration
	return &TxProofResponse{
		TxHash:    txHash,
		Height:    100,
		Index:     0,
		Proof:     []string{"0xproof1", "0xproof2", "0xproof3"},
		Data:      "0x" + txHash,
		BlockHash: "0x" + fmt.Sprintf("%064d", 100),
		Verified:  true,
	}, nil
}

// verifyTxProof verifies a transaction Merkle proof
func (s *Server) verifyTxProof(txHash string, height int64, proof []string, blockHash string) (bool, error) {
	// In production, this would:
	// 1. Get block header at height
	// 2. Verify block hash matches
	// 3. Verify Merkle proof against data hash in header
	// 4. Return verification result

	// For demonstration, simple validation
	if len(proof) == 0 {
		return false, fmt.Errorf("empty proof")
	}

	if blockHash == "" {
		return false, fmt.Errorf("empty block hash")
	}

	// Mock verification - in production, implement proper Merkle proof verification
	return true, nil
}

// Helper function to compute Merkle root (simplified)
func computeMerkleRoot(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		return nil
	}

	if len(leaves) == 1 {
		return leaves[0]
	}

	// Use CometBFT's Merkle tree implementation in production
	// For now, return mock root
	return []byte("mock_merkle_root")
}
