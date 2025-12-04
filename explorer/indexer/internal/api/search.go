package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SearchResult represents a search result
type SearchResult struct {
	Type        string                 `json:"type"` // "block", "transaction", "address", "validator", "pool"
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Score       float64                `json:"score"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp,omitempty"`
}

// SearchResponse represents the search response
type SearchResponse struct {
	Query      string         `json:"query"`
	Results    []SearchResult `json:"results"`
	TotalCount int            `json:"total_count"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	Took       int64          `json:"took_ms"`
}

// handleAdvancedSearch handles advanced search requests
func (s *Server) handleAdvancedSearch(c *gin.Context) {
	startTime := time.Now()

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "search query is required",
		})
		return
	}

	searchType := c.Query("type") // Optional: "all", "block", "transaction", "address", "validator"
	page := parseQueryInt(c, "page", 1)
	limit := parseQueryInt(c, "limit", 20)
	if limit > 100 {
		limit = 100
	}

	// Check cache
	cacheKey := fmt.Sprintf("search:%s:%s:%d:%d", query, searchType, page, limit)
	if cached, err := s.cache.Get(c.Request.Context(), cacheKey); err == nil {
		var response SearchResponse
		if err := json.Unmarshal(cached, &response); err == nil {
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Perform search
	results, totalCount, err := s.performSearch(c.Request.Context(), query, searchType, page, limit)
	if err != nil {
		s.log.Error("Search failed", "query", query, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "search failed",
		})
		return
	}

	response := SearchResponse{
		Query:      query,
		Results:    results,
		TotalCount: totalCount,
		Page:       page,
		Limit:      limit,
		Took:       time.Since(startTime).Milliseconds(),
	}

	// Cache results
	if data, err := json.Marshal(response); err == nil {
		s.cache.Set(c.Request.Context(), cacheKey, data, 5*time.Minute)
	}

	c.JSON(http.StatusOK, response)
}

// performSearch performs the actual search operation
func (s *Server) performSearch(ctx context.Context, query, searchType string, page, limit int) ([]SearchResult, int, error) {
	results := make([]SearchResult, 0)
	query = strings.TrimSpace(query)

	// Determine what type of search to perform based on the query format
	if searchType == "" || searchType == "all" {
		// Auto-detect query type
		if isBlockHeight(query) {
			searchType = "block"
		} else if isTxHash(query) {
			searchType = "transaction"
		} else if isAddress(query) {
			searchType = "address"
		} else {
			// Full-text search across all types
			searchType = "all"
		}
	}

	var totalCount int
	var err error

	switch searchType {
	case "block":
		results, totalCount, err = s.searchBlocks(ctx, query, page, limit)
	case "transaction":
		results, totalCount, err = s.searchTransactions(ctx, query, page, limit)
	case "address":
		results, totalCount, err = s.searchAddresses(ctx, query, page, limit)
	case "validator":
		results, totalCount, err = s.searchValidators(ctx, query, page, limit)
	case "pool":
		results, totalCount, err = s.searchPools(ctx, query, page, limit)
	case "all":
		results, totalCount, err = s.searchAll(ctx, query, page, limit)
	default:
		return nil, 0, fmt.Errorf("invalid search type: %s", searchType)
	}

	return results, totalCount, err
}

// searchBlocks searches for blocks
func (s *Server) searchBlocks(ctx context.Context, query string, page, limit int) ([]SearchResult, int, error) {
	results := make([]SearchResult, 0)

	// Try to parse as block height
	var height int64
	if _, err := fmt.Sscanf(query, "%d", &height); err == nil {
		block, err := s.db.GetBlockByHeight(height)
		if err == nil {
			results = append(results, SearchResult{
				Type:        "block",
				ID:          fmt.Sprintf("%d", block.Height),
				Title:       fmt.Sprintf("Block #%d", block.Height),
				Description: fmt.Sprintf("Hash: %s, Transactions: %d", block.Hash, block.TxCount),
				Score:       1.0,
				Data: map[string]interface{}{
					"height":    block.Height,
					"hash":      block.Hash,
					"tx_count":  block.TxCount,
					"gas_used":  block.GasUsed,
				},
				Timestamp: block.Time,
			})
		}
	}

	// Search by hash prefix
	if len(query) >= 4 {
		blocks, total, err := s.db.SearchBlocksByHash(query, (page-1)*limit, limit)
		if err == nil {
			for _, block := range blocks {
				results = append(results, SearchResult{
					Type:        "block",
					ID:          fmt.Sprintf("%d", block.Height),
					Title:       fmt.Sprintf("Block #%d", block.Height),
					Description: fmt.Sprintf("Hash: %s", block.Hash),
					Score:       0.8,
					Data: map[string]interface{}{
						"height":   block.Height,
						"hash":     block.Hash,
						"tx_count": block.TxCount,
					},
					Timestamp: block.Time,
				})
			}
			return results, total, nil
		}
	}

	return results, len(results), nil
}

// searchTransactions searches for transactions
func (s *Server) searchTransactions(ctx context.Context, query string, page, limit int) ([]SearchResult, int, error) {
	results := make([]SearchResult, 0)

	// Try exact hash match
	tx, err := s.db.GetTransactionByHash(query)
	if err == nil {
		results = append(results, SearchResult{
			Type:        "transaction",
			ID:          tx.Hash,
			Title:       fmt.Sprintf("Transaction %s", tx.Hash[:16]+"..."),
			Description: fmt.Sprintf("Type: %s, Status: %s", tx.Type, tx.Status),
			Score:       1.0,
			Data: map[string]interface{}{
				"hash":         tx.Hash,
				"type":         tx.Type,
				"status":       tx.Status,
				"block_height": tx.BlockHeight,
				"sender":       tx.Sender,
			},
			Timestamp: tx.Time,
		})
		return results, 1, nil
	}

	// Search by hash prefix
	if len(query) >= 4 {
		txs, total, err := s.db.SearchTransactionsByHash(query, (page-1)*limit, limit)
		if err == nil {
			for _, tx := range txs {
				results = append(results, SearchResult{
					Type:        "transaction",
					ID:          tx.Hash,
					Title:       fmt.Sprintf("Tx %s", tx.Hash[:16]+"..."),
					Description: fmt.Sprintf("Type: %s, Status: %s", tx.Type, tx.Status),
					Score:       0.8,
					Data: map[string]interface{}{
						"hash":   tx.Hash,
						"type":   tx.Type,
						"status": tx.Status,
					},
					Timestamp: tx.Time,
				})
			}
			return results, total, nil
		}
	}

	return results, len(results), nil
}

// searchAddresses searches for addresses
func (s *Server) searchAddresses(ctx context.Context, query string, page, limit int) ([]SearchResult, int, error) {
	results := make([]SearchResult, 0)

	// Try exact address match
	account, err := s.db.GetAccount(query)
	if err == nil {
		results = append(results, SearchResult{
			Type:        "address",
			ID:          query,
			Title:       fmt.Sprintf("Address %s", query[:16]+"..."),
			Description: "Account address",
			Score:       1.0,
			Data: map[string]interface{}{
				"address": query,
			},
		})
	}

	// Search by address prefix
	if len(query) >= 4 {
		accounts, total, err := s.db.SearchAccountsByAddress(query, (page-1)*limit, limit)
		if err == nil {
			for _, acc := range accounts {
				results = append(results, SearchResult{
					Type:        "address",
					ID:          acc.Address,
					Title:       fmt.Sprintf("Address %s", acc.Address[:16]+"..."),
					Description: "Account address",
					Score:       0.8,
					Data: map[string]interface{}{
						"address": acc.Address,
					},
				})
			}
			return results, total, nil
		}
	}

	return results, len(results), nil
}

// searchValidators searches for validators
func (s *Server) searchValidators(ctx context.Context, query string, page, limit int) ([]SearchResult, int, error) {
	results := make([]SearchResult, 0)

	validators, total, err := s.db.SearchValidators(query, (page-1)*limit, limit)
	if err != nil {
		return results, 0, err
	}

	for _, val := range validators {
		results = append(results, SearchResult{
			Type:        "validator",
			ID:          val.Address,
			Title:       val.Moniker,
			Description: fmt.Sprintf("Voting Power: %d, Status: %s", val.VotingPower, val.Status),
			Score:       0.9,
			Data: map[string]interface{}{
				"address":      val.Address,
				"moniker":      val.Moniker,
				"voting_power": val.VotingPower,
				"status":       val.Status,
			},
		})
	}

	return results, total, nil
}

// searchPools searches for DEX pools
func (s *Server) searchPools(ctx context.Context, query string, page, limit int) ([]SearchResult, int, error) {
	results := make([]SearchResult, 0)

	pools, total, err := s.db.SearchDEXPools(query, (page-1)*limit, limit)
	if err != nil {
		return results, 0, err
	}

	for _, pool := range pools {
		results = append(results, SearchResult{
			Type:        "pool",
			ID:          pool.PoolID,
			Title:       fmt.Sprintf("Pool %s/%s", pool.TokenA, pool.TokenB),
			Description: fmt.Sprintf("TVL: %.2f", pool.TVL),
			Score:       0.9,
			Data: map[string]interface{}{
				"pool_id": pool.PoolID,
				"token_a": pool.TokenA,
				"token_b": pool.TokenB,
				"tvl":     pool.TVL,
			},
		})
	}

	return results, total, nil
}

// searchAll performs a comprehensive search across all types
func (s *Server) searchAll(ctx context.Context, query string, page, limit int) ([]SearchResult, int, error) {
	allResults := make([]SearchResult, 0)

	// Search blocks
	blockResults, _, _ := s.searchBlocks(ctx, query, 1, 5)
	allResults = append(allResults, blockResults...)

	// Search transactions
	txResults, _, _ := s.searchTransactions(ctx, query, 1, 5)
	allResults = append(allResults, txResults...)

	// Search addresses
	addrResults, _, _ := s.searchAddresses(ctx, query, 1, 5)
	allResults = append(allResults, addrResults...)

	// Search validators
	valResults, _, _ := s.searchValidators(ctx, query, 1, 5)
	allResults = append(allResults, valResults...)

	// Search pools
	poolResults, _, _ := s.searchPools(ctx, query, 1, 5)
	allResults = append(allResults, poolResults...)

	// Sort by score
	// In production, use a proper sorting algorithm

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit
	if start >= len(allResults) {
		return []SearchResult{}, len(allResults), nil
	}
	if end > len(allResults) {
		end = len(allResults)
	}

	return allResults[start:end], len(allResults), nil
}

// Helper functions

func isBlockHeight(s string) bool {
	var height int64
	_, err := fmt.Sscanf(s, "%d", &height)
	return err == nil && height >= 0
}

func isTxHash(s string) bool {
	return len(s) == 64 || (len(s) > 32 && len(s) < 128)
}

func isAddress(s string) bool {
	return strings.HasPrefix(s, "paw1") && len(s) >= 40
}
