package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/paw-chain/paw/explorer/indexer/internal/database"
)

// TraceStep represents a single step in a transaction trace
type TraceStep struct {
	Step      int                    `json:"step"`
	Type      string                 `json:"type"`
	Module    string                 `json:"module"`
	Action    string                 `json:"action"`
	Inputs    map[string]interface{} `json:"inputs"`
	Outputs   map[string]interface{} `json:"outputs"`
	GasUsed   int64                  `json:"gas_used"`
	GasCost   int64                  `json:"gas_cost"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	SubTraces []TraceStep            `json:"sub_traces,omitempty"`
}

// TransactionTrace represents a complete transaction trace
type TransactionTrace struct {
	TxHash       string                 `json:"tx_hash"`
	BlockHeight  int64                  `json:"block_height"`
	Success      bool                   `json:"success"`
	TotalGasUsed int64                  `json:"total_gas_used"`
	CallStack    []TraceStep            `json:"call_stack"`
	Events       []database.Event       `json:"events"`
	StateChanges []StateChange          `json:"state_changes"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// StateChange represents a state change during transaction execution
type StateChange struct {
	Address   string      `json:"address"`
	Key       string      `json:"key"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
	Operation string      `json:"operation"` // "create", "update", "delete"
}

// handleGetTransactionTrace handles transaction tracing requests
func (s *Server) handleGetTransactionTrace(c *gin.Context) {
	txHash := c.Param("hash")
	if txHash == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "transaction hash is required",
		})
		return
	}

	// Check cache first
	cacheKey := fmt.Sprintf("trace:%s", txHash)
	if cached, err := s.cache.Get(c.Request.Context(), cacheKey); err == nil {
		var trace TransactionTrace
		if err := json.Unmarshal(cached, &trace); err == nil {
			c.JSON(http.StatusOK, gin.H{
				"trace":  trace,
				"cached": true,
			})
			return
		}
	}

	// Get transaction from database
	tx, err := s.db.GetTransactionByHash(txHash)
	if err != nil {
		s.log.Error("Failed to get transaction", "hash", txHash, "error", err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "transaction not found",
		})
		return
	}

	// Build transaction trace
	trace, err := s.buildTransactionTrace(c.Request.Context(), tx)
	if err != nil {
		s.log.Error("Failed to build transaction trace", "hash", txHash, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to build transaction trace",
		})
		return
	}

	// Cache the trace
	if data, err := json.Marshal(trace); err == nil {
		s.cache.Set(c.Request.Context(), cacheKey, data, time.Hour)
	}

	c.JSON(http.StatusOK, gin.H{
		"trace": trace,
	})
}

// buildTransactionTrace builds a complete trace for a transaction
func (s *Server) buildTransactionTrace(ctx context.Context, tx database.Transaction) (*TransactionTrace, error) {
	// Get events for the transaction
	events, err := s.db.GetEventsByTxHash(tx.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %w", err)
	}

	// Parse messages to build call stack
	callStack, err := s.buildCallStack(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to build call stack: %w", err)
	}

	// Extract state changes from events
	stateChanges := s.extractStateChanges(events)

	// Build metadata
	metadata := map[string]interface{}{
		"sender":     tx.Sender,
		"type":       tx.Type,
		"fee_amount": tx.FeeAmount,
		"fee_denom":  tx.FeeDenom,
		"gas_wanted": tx.GasWanted,
		"gas_used":   tx.GasUsed,
		"memo":       tx.Memo,
	}

	trace := &TransactionTrace{
		TxHash:       tx.Hash,
		BlockHeight:  tx.BlockHeight,
		Success:      tx.Status == "success",
		TotalGasUsed: tx.GasUsed,
		CallStack:    callStack,
		Events:       events,
		StateChanges: stateChanges,
		Metadata:     metadata,
	}

	return trace, nil
}

// buildCallStack builds the call stack from transaction messages
func (s *Server) buildCallStack(tx database.Transaction) ([]TraceStep, error) {
	var messages []map[string]interface{}
	if err := json.Unmarshal(tx.Messages, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	callStack := make([]TraceStep, 0, len(messages))
	totalGas := int64(0)

	for i, msg := range messages {
		msgType, _ := msg["@type"].(string)

		// Calculate gas for this message (simplified)
		gasUsed := tx.GasUsed / int64(len(messages))
		if i == len(messages)-1 {
			gasUsed = tx.GasUsed - totalGas // remainder for last message
		}
		totalGas += gasUsed

		step := TraceStep{
			Step:      i + 1,
			Type:      "message",
			Module:    extractModuleFromType(msgType),
			Action:    extractActionFromType(msgType),
			Inputs:    msg,
			Outputs:   make(map[string]interface{}),
			GasUsed:   gasUsed,
			GasCost:   gasUsed,
			Timestamp: tx.Time,
		}

		// Add error if transaction failed
		if tx.Status != "success" && i == 0 {
			step.Error = tx.RawLog
		}

		callStack = append(callStack, step)
	}

	return callStack, nil
}

// extractStateChanges extracts state changes from events
func (s *Server) extractStateChanges(events []database.Event) []StateChange {
	stateChanges := make([]StateChange, 0)

	for _, event := range events {
		var attrs []map[string]interface{}
		if err := json.Unmarshal(event.Attributes, &attrs); err != nil {
			continue
		}

		// Extract state changes based on event type
		switch event.EventType {
		case "coin_spent":
			change := s.extractCoinSpentChange(attrs)
			if change != nil {
				stateChanges = append(stateChanges, *change)
			}
		case "coin_received":
			change := s.extractCoinReceivedChange(attrs)
			if change != nil {
				stateChanges = append(stateChanges, *change)
			}
		case "transfer":
			change := s.extractTransferChange(attrs)
			if change != nil {
				stateChanges = append(stateChanges, *change)
			}
		}
	}

	return stateChanges
}

// extractCoinSpentChange extracts state change from coin_spent event
func (s *Server) extractCoinSpentChange(attrs []map[string]interface{}) *StateChange {
	var spender, amount string
	for _, attr := range attrs {
		if key, ok := attr["key"].(string); ok {
			if key == "spender" {
				spender, _ = attr["value"].(string)
			}
			if key == "amount" {
				amount, _ = attr["value"].(string)
			}
		}
	}

	if spender != "" && amount != "" {
		return &StateChange{
			Address:   spender,
			Key:       "balance",
			OldValue:  nil, // Would need to query historical state
			NewValue:  amount,
			Operation: "update",
		}
	}

	return nil
}

// extractCoinReceivedChange extracts state change from coin_received event
func (s *Server) extractCoinReceivedChange(attrs []map[string]interface{}) *StateChange {
	var receiver, amount string
	for _, attr := range attrs {
		if key, ok := attr["key"].(string); ok {
			if key == "receiver" {
				receiver, _ = attr["value"].(string)
			}
			if key == "amount" {
				amount, _ = attr["value"].(string)
			}
		}
	}

	if receiver != "" && amount != "" {
		return &StateChange{
			Address:   receiver,
			Key:       "balance",
			OldValue:  nil,
			NewValue:  amount,
			Operation: "update",
		}
	}

	return nil
}

// extractTransferChange extracts state change from transfer event
func (s *Server) extractTransferChange(attrs []map[string]interface{}) *StateChange {
	var recipient, sender, amount string
	for _, attr := range attrs {
		if key, ok := attr["key"].(string); ok {
			if key == "recipient" {
				recipient, _ = attr["value"].(string)
			}
			if key == "sender" {
				sender, _ = attr["value"].(string)
			}
			if key == "amount" {
				amount, _ = attr["value"].(string)
			}
		}
	}

	if recipient != "" && amount != "" {
		return &StateChange{
			Address:   recipient,
			Key:       "balance",
			OldValue:  sender,
			NewValue:  amount,
			Operation: "transfer",
		}
	}

	return nil
}

// extractModuleFromType extracts module name from message type
func extractModuleFromType(msgType string) string {
	// e.g., "/cosmos.bank.v1beta1.MsgSend" -> "bank"
	parts := []rune{}
	inModule := false
	for _, c := range msgType {
		if c == '.' {
			if !inModule {
				inModule = true
				continue
			} else {
				break
			}
		}
		if inModule {
			parts = append(parts, c)
		}
	}
	if len(parts) > 0 {
		return string(parts)
	}
	return "unknown"
}

// extractActionFromType extracts action name from message type
func extractActionFromType(msgType string) string {
	// e.g., "/cosmos.bank.v1beta1.MsgSend" -> "Send"
	parts := []rune{}
	foundMsg := false
	for _, c := range msgType {
		if foundMsg {
			parts = append(parts, c)
		} else if c == 'M' && len(parts) == 0 {
			foundMsg = true
			continue
		}
	}
	if len(parts) > 3 && string(parts[:3]) == "sg" {
		return string(parts[2:])
	}
	if len(parts) > 0 {
		return string(parts)
	}
	return "unknown"
}
