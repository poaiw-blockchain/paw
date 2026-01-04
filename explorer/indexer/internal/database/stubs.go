package database

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Query implementations for explorer API endpoints.

func (db *Database) GetBlockByHeight(height int64) (*Block, error) {
	var block Block
	err := db.QueryRow(
		`SELECT height, hash, proposer_address, time, tx_count, gas_used, gas_wanted, evidence_count
		 FROM blocks WHERE height = $1`,
		height,
	).Scan(
		&block.Height,
		&block.Hash,
		&block.ProposerAddress,
		&block.Time,
		&block.TxCount,
		&block.GasUsed,
		&block.GasWanted,
		&block.EvidenceCount,
	)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

func (db *Database) GetBlocks(offset, limit int) ([]Block, int, error) {
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM blocks").Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT height, hash, proposer_address, time, tx_count, gas_used, gas_wanted, evidence_count
		 FROM blocks ORDER BY height DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	blocks := make([]Block, 0)
	for rows.Next() {
		var block Block
		if err := rows.Scan(
			&block.Height,
			&block.Hash,
			&block.ProposerAddress,
			&block.Time,
			&block.TxCount,
			&block.GasUsed,
			&block.GasWanted,
			&block.EvidenceCount,
		); err != nil {
			return nil, 0, err
		}
		blocks = append(blocks, block)
	}

	return blocks, total, rows.Err()
}

func (db *Database) SearchBlocksByHash(query string, offset, limit int) ([]Block, int, error) {
	pattern := fmt.Sprintf("%s%%", query)
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM blocks WHERE hash ILIKE $1", pattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT height, hash, proposer_address, time, tx_count, gas_used, gas_wanted, evidence_count
		 FROM blocks WHERE hash ILIKE $1 ORDER BY height DESC LIMIT $2 OFFSET $3`,
		pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	blocks := make([]Block, 0)
	for rows.Next() {
		var block Block
		if err := rows.Scan(
			&block.Height,
			&block.Hash,
			&block.ProposerAddress,
			&block.Time,
			&block.TxCount,
			&block.GasUsed,
			&block.GasWanted,
			&block.EvidenceCount,
		); err != nil {
			return nil, 0, err
		}
		blocks = append(blocks, block)
	}

	return blocks, total, rows.Err()
}

func (db *Database) GetTransactions(offset, limit int, status, txType string) ([]Transaction, int, error) {
	filters := []string{"1=1"}
	args := []interface{}{}

	if status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	if txType != "" {
		args = append(args, txType)
		filters = append(filters, fmt.Sprintf("type = $%d", len(args)))
	}

	where := strings.Join(filters, " AND ")
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM transactions WHERE %s", where)
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`SELECT hash, block_height, tx_index, type, sender, status, code, gas_used, gas_wanted,
		fee_amount, fee_denom, memo, raw_log, time, messages, events
		FROM transactions WHERE %s ORDER BY time DESC LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	transactions := make([]Transaction, 0)
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(
			&tx.Hash,
			&tx.BlockHeight,
			&tx.TxIndex,
			&tx.Type,
			&tx.Sender,
			&tx.Status,
			&tx.Code,
			&tx.GasUsed,
			&tx.GasWanted,
			&tx.FeeAmount,
			&tx.FeeDenom,
			&tx.Memo,
			&tx.RawLog,
			&tx.Time,
			&tx.Messages,
			&tx.Events,
		); err != nil {
			return nil, 0, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, total, rows.Err()
}

func (db *Database) GetTransactionByHash(hash string) (*Transaction, error) {
	var tx Transaction
	err := db.QueryRow(
		`SELECT hash, block_height, tx_index, type, sender, status, code, gas_used, gas_wanted,
		 fee_amount, fee_denom, memo, raw_log, time, messages, events
		 FROM transactions WHERE hash = $1`,
		hash,
	).Scan(
		&tx.Hash,
		&tx.BlockHeight,
		&tx.TxIndex,
		&tx.Type,
		&tx.Sender,
		&tx.Status,
		&tx.Code,
		&tx.GasUsed,
		&tx.GasWanted,
		&tx.FeeAmount,
		&tx.FeeDenom,
		&tx.Memo,
		&tx.RawLog,
		&tx.Time,
		&tx.Messages,
		&tx.Events,
	)
	if err != nil {
		return nil, err
	}
	return &tx, nil
}

func (db *Database) SearchTransactionsByHash(query string, offset, limit int) ([]Transaction, int, error) {
	pattern := fmt.Sprintf("%s%%", query)
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM transactions WHERE hash ILIKE $1", pattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT hash, block_height, tx_index, type, sender, status, code, gas_used, gas_wanted,
		 fee_amount, fee_denom, memo, raw_log, time, messages, events
		 FROM transactions WHERE hash ILIKE $1 ORDER BY time DESC LIMIT $2 OFFSET $3`,
		pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	transactions := make([]Transaction, 0)
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(
			&tx.Hash,
			&tx.BlockHeight,
			&tx.TxIndex,
			&tx.Type,
			&tx.Sender,
			&tx.Status,
			&tx.Code,
			&tx.GasUsed,
			&tx.GasWanted,
			&tx.FeeAmount,
			&tx.FeeDenom,
			&tx.Memo,
			&tx.RawLog,
			&tx.Time,
			&tx.Messages,
			&tx.Events,
		); err != nil {
			return nil, 0, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, total, rows.Err()
}

func (db *Database) GetTransactionsByHeight(height int64) ([]Transaction, error) {
	rows, err := db.Query(
		`SELECT hash, block_height, tx_index, type, sender, status, code, gas_used, gas_wanted,
		 fee_amount, fee_denom, memo, raw_log, time, messages, events
		 FROM transactions WHERE block_height = $1 ORDER BY tx_index ASC`,
		height,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := make([]Transaction, 0)
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(
			&tx.Hash,
			&tx.BlockHeight,
			&tx.TxIndex,
			&tx.Type,
			&tx.Sender,
			&tx.Status,
			&tx.Code,
			&tx.GasUsed,
			&tx.GasWanted,
			&tx.FeeAmount,
			&tx.FeeDenom,
			&tx.Memo,
			&tx.RawLog,
			&tx.Time,
			&tx.Messages,
			&tx.Events,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, rows.Err()
}

func (db *Database) GetTransactionsByAddress(address string, offset, limit int) ([]Transaction, int, error) {
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM transactions WHERE sender = $1", address).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT hash, block_height, tx_index, type, sender, status, code, gas_used, gas_wanted,
		 fee_amount, fee_denom, memo, raw_log, time, messages, events
		 FROM transactions WHERE sender = $1 ORDER BY time DESC LIMIT $2 OFFSET $3`,
		address, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	transactions := make([]Transaction, 0)
	for rows.Next() {
		var tx Transaction
		if err := rows.Scan(
			&tx.Hash,
			&tx.BlockHeight,
			&tx.TxIndex,
			&tx.Type,
			&tx.Sender,
			&tx.Status,
			&tx.Code,
			&tx.GasUsed,
			&tx.GasWanted,
			&tx.FeeAmount,
			&tx.FeeDenom,
			&tx.Memo,
			&tx.RawLog,
			&tx.Time,
			&tx.Messages,
			&tx.Events,
		); err != nil {
			return nil, 0, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, total, rows.Err()
}

func (db *Database) GetEventsByTxHash(hash string) ([]Event, error) {
	rows, err := db.Query(
		`SELECT tx_hash, block_height, event_type, module, attributes
		 FROM events WHERE tx_hash = $1 ORDER BY id ASC`,
		hash,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := make([]Event, 0)
	for rows.Next() {
		var event Event
		if err := rows.Scan(
			&event.TxHash,
			&event.BlockHeight,
			&event.EventType,
			&event.Module,
			&event.Attributes,
		); err != nil {
			return nil, err
		}
		events = append(events, event)
	}

	return events, rows.Err()
}

func (db *Database) GetAccount(address string) (*Account, error) {
	var account Account
	if err := db.QueryRow("SELECT address FROM accounts WHERE address = $1", address).Scan(&account.Address); err != nil {
		return nil, err
	}
	return &account, nil
}

func (db *Database) GetAccountBalances(address string) ([]map[string]interface{}, error) {
	rows, err := db.Query(
		`SELECT address, denom, amount, last_updated_height, last_updated_at
		 FROM account_balances WHERE address = $1 ORDER BY denom`,
		address,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	balances := make([]map[string]interface{}, 0)
	for rows.Next() {
		var entry struct {
			Address           string
			Denom             string
			Amount            string
			LastUpdatedHeight int64
			LastUpdatedAt     time.Time
		}
		if err := rows.Scan(&entry.Address, &entry.Denom, &entry.Amount, &entry.LastUpdatedHeight, &entry.LastUpdatedAt); err != nil {
			return nil, err
		}
		balances = append(balances, map[string]interface{}{
			"address":             entry.Address,
			"denom":               entry.Denom,
			"amount":              entry.Amount,
			"last_updated_height": entry.LastUpdatedHeight,
			"last_updated_at":     entry.LastUpdatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(balances) > 0 {
		return balances, nil
	}

	var defaultBalance sql.NullString
	if err := db.QueryRow("SELECT balance FROM accounts WHERE address = $1", address).Scan(&defaultBalance); err != nil {
		if err == sql.ErrNoRows {
			return balances, nil
		}
		return nil, err
	}
	if !defaultBalance.Valid {
		return balances, nil
	}
	var fallback []map[string]interface{}
	if err := json.Unmarshal([]byte(defaultBalance.String), &fallback); err != nil {
		return balances, nil
	}
	return fallback, nil
}

func (db *Database) GetAccountTokens(address string) ([]map[string]interface{}, error) {
	rows, err := db.Query(
		`SELECT address, token_denom, token_name, token_symbol, amount, ibc_trace, last_updated_height, last_updated_at
		 FROM account_tokens WHERE address = $1 ORDER BY token_denom`,
		address,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]map[string]interface{}, 0)
	for rows.Next() {
		var entry struct {
			Address           string
			TokenDenom        string
			TokenName         sql.NullString
			TokenSymbol       sql.NullString
			Amount            string
			IbcTrace          sql.NullString
			LastUpdatedHeight int64
			LastUpdatedAt     time.Time
		}
		if err := rows.Scan(
			&entry.Address,
			&entry.TokenDenom,
			&entry.TokenName,
			&entry.TokenSymbol,
			&entry.Amount,
			&entry.IbcTrace,
			&entry.LastUpdatedHeight,
			&entry.LastUpdatedAt,
		); err != nil {
			return nil, err
		}
		var trace interface{}
		if entry.IbcTrace.Valid {
			_ = json.Unmarshal([]byte(entry.IbcTrace.String), &trace)
		}
		tokens = append(tokens, map[string]interface{}{
			"address":             entry.Address,
			"token_denom":         entry.TokenDenom,
			"token_name":          entry.TokenName.String,
			"token_symbol":        entry.TokenSymbol.String,
			"amount":              entry.Amount,
			"ibc_trace":           trace,
			"last_updated_height": entry.LastUpdatedHeight,
			"last_updated_at":     entry.LastUpdatedAt,
		})
	}

	return tokens, rows.Err()
}

func (db *Database) SearchAccountsByAddress(query string, offset, limit int) ([]Account, int, error) {
	pattern := fmt.Sprintf("%s%%", query)
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM accounts WHERE address ILIKE $1", pattern).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query("SELECT address FROM accounts WHERE address ILIKE $1 ORDER BY address LIMIT $2 OFFSET $3", pattern, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	accounts := make([]Account, 0)
	for rows.Next() {
		var account Account
		if err := rows.Scan(&account.Address); err != nil {
			return nil, 0, err
		}
		accounts = append(accounts, account)
	}

	return accounts, total, rows.Err()
}

func (db *Database) GetValidators(offset, limit int, status string) ([]Validator, int, error) {
	args := []interface{}{}
	filters := []string{"1=1"}
	if status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	where := strings.Join(filters, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM validators WHERE %s", where)
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`SELECT address, operator_address, consensus_pubkey, moniker,
		commission_rate, commission_max_rate, commission_max_change_rate, voting_power,
		jailed, status, tokens, delegator_shares
		FROM validators WHERE %s ORDER BY voting_power DESC LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	validators := make([]Validator, 0)
	for rows.Next() {
		var validator Validator
		if err := rows.Scan(
			&validator.Address,
			&validator.OperatorAddress,
			&validator.ConsensusPubkey,
			&validator.Moniker,
			&validator.CommissionRate,
			&validator.CommissionMaxRate,
			&validator.CommissionMaxChangeRate,
			&validator.VotingPower,
			&validator.Jailed,
			&validator.Status,
			&validator.Tokens,
			&validator.DelegatorShares,
		); err != nil {
			return nil, 0, err
		}
		validators = append(validators, validator)
	}

	return validators, total, rows.Err()
}

func (db *Database) GetValidator(address string) (*Validator, error) {
	var validator Validator
	err := db.QueryRow(
		`SELECT address, operator_address, consensus_pubkey, moniker,
		 commission_rate, commission_max_rate, commission_max_change_rate, voting_power,
		 jailed, status, tokens, delegator_shares
		 FROM validators WHERE address = $1 OR operator_address = $1`,
		address,
	).Scan(
		&validator.Address,
		&validator.OperatorAddress,
		&validator.ConsensusPubkey,
		&validator.Moniker,
		&validator.CommissionRate,
		&validator.CommissionMaxRate,
		&validator.CommissionMaxChangeRate,
		&validator.VotingPower,
		&validator.Jailed,
		&validator.Status,
		&validator.Tokens,
		&validator.DelegatorShares,
	)
	if err != nil {
		return nil, err
	}
	return &validator, nil
}

func (db *Database) SearchValidators(query string, offset, limit int) ([]Validator, int, error) {
	pattern := fmt.Sprintf("%%%s%", query)
	var total int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM validators WHERE moniker ILIKE $1 OR operator_address ILIKE $1 OR address ILIKE $1`,
		pattern,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT address, operator_address, consensus_pubkey, moniker,
		 commission_rate, commission_max_rate, commission_max_change_rate, voting_power,
		 jailed, status, tokens, delegator_shares
		 FROM validators WHERE moniker ILIKE $1 OR operator_address ILIKE $1 OR address ILIKE $1
		 ORDER BY voting_power DESC LIMIT $2 OFFSET $3`,
		pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	validators := make([]Validator, 0)
	for rows.Next() {
		var validator Validator
		if err := rows.Scan(
			&validator.Address,
			&validator.OperatorAddress,
			&validator.ConsensusPubkey,
			&validator.Moniker,
			&validator.CommissionRate,
			&validator.CommissionMaxRate,
			&validator.CommissionMaxChangeRate,
			&validator.VotingPower,
			&validator.Jailed,
			&validator.Status,
			&validator.Tokens,
			&validator.DelegatorShares,
		); err != nil {
			return nil, 0, err
		}
		validators = append(validators, validator)
	}

	return validators, total, rows.Err()
}

func (db *Database) GetActiveValidators() ([]Validator, error) {
	rows, err := db.Query(
		`SELECT address, operator_address, consensus_pubkey, moniker,
		 commission_rate, commission_max_rate, commission_max_change_rate, voting_power,
		 jailed, status, tokens, delegator_shares
		 FROM validators WHERE status = 'BOND_STATUS_BONDED' ORDER BY voting_power DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	validators := make([]Validator, 0)
	for rows.Next() {
		var validator Validator
		if err := rows.Scan(
			&validator.Address,
			&validator.OperatorAddress,
			&validator.ConsensusPubkey,
			&validator.Moniker,
			&validator.CommissionRate,
			&validator.CommissionMaxRate,
			&validator.CommissionMaxChangeRate,
			&validator.VotingPower,
			&validator.Jailed,
			&validator.Status,
			&validator.Tokens,
			&validator.DelegatorShares,
		); err != nil {
			return nil, err
		}
		validators = append(validators, validator)
	}

	return validators, rows.Err()
}

func (db *Database) GetValidatorUptime(address string, days int) ([]map[string]interface{}, error) {
	rows, err := db.Query(
		`SELECT height, signed, timestamp FROM validator_uptime
		 WHERE validator_address = $1 AND timestamp >= NOW() - ($2::text || ' days')::interval
		 ORDER BY height DESC`,
		address, days,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]map[string]interface{}, 0)
	for rows.Next() {
		var height int64
		var signed bool
		var timestamp time.Time
		if err := rows.Scan(&height, &signed, &timestamp); err != nil {
			return nil, err
		}
		entries = append(entries, map[string]interface{}{
			"height":    height,
			"signed":    signed,
			"timestamp": timestamp,
		})
	}

	return entries, rows.Err()
}

func (db *Database) GetValidatorRewards(address string, offset, limit int) ([]map[string]interface{}, int, error) {
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM validator_rewards WHERE validator_address = $1", address).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT amount, denom, height, timestamp
		 FROM validator_rewards WHERE validator_address = $1
		 ORDER BY height DESC LIMIT $2 OFFSET $3`,
		address, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	rewards := make([]map[string]interface{}, 0)
	for rows.Next() {
		var amount string
		var denom string
		var height int64
		var timestamp time.Time
		if err := rows.Scan(&amount, &denom, &height, &timestamp); err != nil {
			return nil, 0, err
		}
		rewards = append(rewards, map[string]interface{}{
			"amount":    amount,
			"denom":     denom,
			"height":    height,
			"timestamp": timestamp,
		})
	}

	return rewards, total, rows.Err()
}

func (db *Database) GetDEXPools(offset, limit int, sortBy string) ([]DEXPool, int, error) {
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM dex_pools").Scan(&total); err != nil {
		return nil, 0, err
	}

	order := "tvl DESC"
	if sortBy == "volume" {
		order = "total_volume_24h DESC"
	}

	rows, err := db.Query(
		fmt.Sprintf(`SELECT pool_id, token_a, token_b, reserve_a, reserve_b, lp_token_supply,
		 swap_fee_rate, tvl, created_height FROM dex_pools ORDER BY %s LIMIT $1 OFFSET $2`, order),
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	pools := make([]DEXPool, 0)
	for rows.Next() {
		var pool DEXPool
		if err := rows.Scan(
			&pool.PoolID,
			&pool.TokenA,
			&pool.TokenB,
			&pool.ReserveA,
			&pool.ReserveB,
			&pool.LPTokenSupply,
			&pool.SwapFeeRate,
			&pool.TVL,
			&pool.CreatedHeight,
		); err != nil {
			return nil, 0, err
		}
		pools = append(pools, pool)
	}

	return pools, total, rows.Err()
}

func (db *Database) GetDEXPool(poolID string) (*DEXPool, error) {
	var pool DEXPool
	err := db.QueryRow(
		`SELECT pool_id, token_a, token_b, reserve_a, reserve_b, lp_token_supply,
		 swap_fee_rate, tvl, created_height FROM dex_pools WHERE pool_id = $1`,
		poolID,
	).Scan(
		&pool.PoolID,
		&pool.TokenA,
		&pool.TokenB,
		&pool.ReserveA,
		&pool.ReserveB,
		&pool.LPTokenSupply,
		&pool.SwapFeeRate,
		&pool.TVL,
		&pool.CreatedHeight,
	)
	if err != nil {
		return nil, err
	}
	return &pool, nil
}


func (db *Database) SearchDEXPools(query string, offset, limit int) ([]DEXPool, int, error) {
	pattern := fmt.Sprintf("%%%s%", query)
	var total int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM dex_pools WHERE pool_id ILIKE $1 OR token_a ILIKE $1 OR token_b ILIKE $1`,
		pattern,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT pool_id, token_a, token_b, reserve_a, reserve_b, lp_token_supply,
		 swap_fee_rate, tvl, created_height FROM dex_pools
		 WHERE pool_id ILIKE $1 OR token_a ILIKE $1 OR token_b ILIKE $1
		 ORDER BY tvl DESC LIMIT $2 OFFSET $3`,
		pattern, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	pools := make([]DEXPool, 0)
	for rows.Next() {
		var pool DEXPool
		if err := rows.Scan(
			&pool.PoolID,
			&pool.TokenA,
			&pool.TokenB,
			&pool.ReserveA,
			&pool.ReserveB,
			&pool.LPTokenSupply,
			&pool.SwapFeeRate,
			&pool.TVL,
			&pool.CreatedHeight,
		); err != nil {
			return nil, 0, err
		}
		pools = append(pools, pool)
	}

	return pools, total, rows.Err()
}

func (db *Database) GetDEXTrades(offset, limit int) ([]DEXSwap, int, error) {
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM dex_swaps").Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT tx_hash, pool_id, sender, token_in, token_out, amount_in, amount_out, price, fee, time
		 FROM dex_swaps ORDER BY time DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	trades := make([]DEXSwap, 0)
	for rows.Next() {
		var trade DEXSwap
		if err := rows.Scan(
			&trade.TxHash,
			&trade.PoolID,
			&trade.Sender,
			&trade.TokenIn,
			&trade.TokenOut,
			&trade.AmountIn,
			&trade.AmountOut,
			&trade.Price,
			&trade.Fee,
			&trade.Time,
		); err != nil {
			return nil, 0, err
		}
		trades = append(trades, trade)
	}

	return trades, total, rows.Err()
}

func (db *Database) GetLatestDEXTrades(limit int) ([]DEXSwap, error) {
	if limit <= 0 {
		limit = 10
	}
	rows, err := db.Query(
		`SELECT tx_hash, pool_id, sender, token_in, token_out, amount_in, amount_out, price, fee, time
		 FROM dex_swaps ORDER BY time DESC LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trades := make([]DEXSwap, 0)
	for rows.Next() {
		var trade DEXSwap
		if err := rows.Scan(
			&trade.TxHash,
			&trade.PoolID,
			&trade.Sender,
			&trade.TokenIn,
			&trade.TokenOut,
			&trade.AmountIn,
			&trade.AmountOut,
			&trade.Price,
			&trade.Fee,
			&trade.Time,
		); err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}

	return trades, rows.Err()
}

func (db *Database) GetPoolDepth(poolID string) (map[string]interface{}, error) {
	pool, err := db.GetDEXPool(poolID)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"pool_id":   pool.PoolID,
		"token_a":   pool.TokenA,
		"token_b":   pool.TokenB,
		"reserve_a": pool.ReserveA,
		"reserve_b": pool.ReserveB,
		"tvl":       pool.TVL,
	}, nil
}

func (db *Database) Ping() error {
	return db.DB.Ping()
}

func (db *Database) GetPoolTrades(poolID string, offset, limit int) ([]DEXSwap, int, error) {
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM dex_swaps WHERE pool_id = $1", poolID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT tx_hash, pool_id, sender, token_in, token_out, amount_in, amount_out, price, fee, time
		 FROM dex_swaps WHERE pool_id = $1 ORDER BY time DESC LIMIT $2 OFFSET $3`,
		poolID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	trades := make([]DEXSwap, 0)
	for rows.Next() {
		var trade DEXSwap
		if err := rows.Scan(
			&trade.TxHash,
			&trade.PoolID,
			&trade.Sender,
			&trade.TokenIn,
			&trade.TokenOut,
			&trade.AmountIn,
			&trade.AmountOut,
			&trade.Price,
			&trade.Fee,
			&trade.Time,
		); err != nil {
			return nil, 0, err
		}
		trades = append(trades, trade)
	}

	return trades, total, rows.Err()
}

func (db *Database) GetPoolLiquidity(poolID string, offset, limit int) ([]map[string]interface{}, int, error) {
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM dex_liquidity_events WHERE pool_id = $1", poolID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT tx_hash, sender, event_type, amount_a, amount_b, lp_tokens, time
		 FROM dex_liquidity_events WHERE pool_id = $1 ORDER BY time DESC LIMIT $2 OFFSET $3`,
		poolID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	events := make([]map[string]interface{}, 0)
	for rows.Next() {
		var entry struct {
			TxHash    string
			Sender    string
			EventType string
			AmountA   float64
			AmountB   float64
			LPTokens  float64
			Time      time.Time
		}
		if err := rows.Scan(&entry.TxHash, &entry.Sender, &entry.EventType, &entry.AmountA, &entry.AmountB, &entry.LPTokens, &entry.Time); err != nil {
			return nil, 0, err
		}
		events = append(events, map[string]interface{}{
			"tx_hash":   entry.TxHash,
			"sender":    entry.Sender,
			"event_type": entry.EventType,
			"amount_a":  entry.AmountA,
			"amount_b":  entry.AmountB,
			"lp_tokens": entry.LPTokens,
			"time":      entry.Time,
		})
	}

	return events, total, rows.Err()
}

func (db *Database) GetPoolChartData(poolID, period string) (map[string]interface{}, error) {
	start, end := parsePeriodRange(period)
	priceHistory, err := db.GetPoolPriceHistory(context.Background(), poolID, start, end, "1h")
	if err != nil {
		return nil, err
	}
	volumeHistory, err := db.GetPoolVolumeHistory(context.Background(), poolID, start, end, "1h")
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"price_history":  priceHistory,
		"volume_history": volumeHistory,
	}, nil
}

func (db *Database) GetPoolTradesSummary(poolID string) (map[string]interface{}, error) {
	var totalTrades int64
	if err := db.QueryRow("SELECT COUNT(*) FROM dex_swaps WHERE pool_id = $1", poolID).Scan(&totalTrades); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"pool_id":      poolID,
		"total_trades": totalTrades,
	}, nil
}

func (db *Database) GetDEXTradesSummary() (map[string]interface{}, error) {
	var totalTrades int64
	if err := db.QueryRow("SELECT COUNT(*) FROM dex_swaps").Scan(&totalTrades); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"total_trades": totalTrades,
	}, nil
}

func (db *Database) GetLatestOraclePrices() ([]OraclePrice, error) {
	rows, err := db.Query(
		`SELECT asset, price, timestamp, block_height, source
		 FROM oracle_prices ORDER BY timestamp DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := make([]OraclePrice, 0)
	for rows.Next() {
		var price OraclePrice
		if err := rows.Scan(&price.Asset, &price.Price, &price.Timestamp, &price.BlockHeight, &price.Source); err != nil {
			return nil, err
		}
		prices = append(prices, price)
	}

	return prices, rows.Err()
}

func (db *Database) GetAssetPrice(asset string) (*OraclePrice, error) {
	var price OraclePrice
	err := db.QueryRow(
		`SELECT asset, price, timestamp, block_height, source
		 FROM oracle_prices WHERE asset = $1 ORDER BY timestamp DESC LIMIT 1`,
		asset,
	).Scan(&price.Asset, &price.Price, &price.Timestamp, &price.BlockHeight, &price.Source)
	if err != nil {
		return nil, err
	}
	return &price, nil
}

func (db *Database) GetAssetPriceHistory(asset, period string) ([]OraclePrice, error) {
	start, _ := parsePeriodRange(period)
	rows, err := db.Query(
		`SELECT asset, price, timestamp, block_height, source
		 FROM oracle_prices WHERE asset = $1 AND timestamp >= $2 ORDER BY timestamp ASC`,
		asset, start,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prices := make([]OraclePrice, 0)
	for rows.Next() {
		var price OraclePrice
		if err := rows.Scan(&price.Asset, &price.Price, &price.Timestamp, &price.BlockHeight, &price.Source); err != nil {
			return nil, err
		}
		prices = append(prices, price)
	}

	return prices, rows.Err()
}

func (db *Database) GetAssetPriceChart(asset, period, interval string) ([]map[string]interface{}, error) {
	start, _ := parsePeriodRange(period)
	rows, err := db.Query(
		`SELECT timestamp, price, block_height
		 FROM oracle_prices WHERE asset = $1 AND timestamp >= $2 ORDER BY timestamp ASC`,
		asset, start,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chart := make([]map[string]interface{}, 0)
	for rows.Next() {
		var timestamp time.Time
		var price float64
		var height int64
		if err := rows.Scan(&timestamp, &price, &height); err != nil {
			return nil, err
		}
		chart = append(chart, map[string]interface{}{
			"timestamp": timestamp,
			"price":     price,
			"height":    height,
		})
	}

	return chart, rows.Err()
}

func (db *Database) GetOracleSubmissions(offset, limit int, asset string) ([]map[string]interface{}, int, error) {
	filters := []string{"1=1"}
	args := []interface{}{}
	if asset != "" {
		args = append(args, asset)
		filters = append(filters, fmt.Sprintf("asset = $%d", len(args)))
	}
	where := strings.Join(filters, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM oracle_submissions WHERE %s", where)
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`SELECT validator_address, asset, price, timestamp, block_height, included, deviation
		FROM oracle_submissions WHERE %s ORDER BY timestamp DESC LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	submissions := make([]map[string]interface{}, 0)
	for rows.Next() {
		var entry struct {
			Validator string
			Asset     string
			Price     float64
			Timestamp time.Time
			Height    int64
			Included  bool
			Deviation sql.NullFloat64
		}
		if err := rows.Scan(&entry.Validator, &entry.Asset, &entry.Price, &entry.Timestamp, &entry.Height, &entry.Included, &entry.Deviation); err != nil {
			return nil, 0, err
		}
		submissions = append(submissions, map[string]interface{}{
			"validator_address": entry.Validator,
			"asset":             entry.Asset,
			"price":             entry.Price,
			"timestamp":         entry.Timestamp,
			"block_height":      entry.Height,
			"included":          entry.Included,
			"deviation":         entry.Deviation.Float64,
		})
	}

	return submissions, total, rows.Err()
}

func (db *Database) GetOracleSlashes(offset, limit int) ([]map[string]interface{}, int, error) {
	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM oracle_slashes").Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := db.Query(
		`SELECT validator_address, slash_amount, reason, height, timestamp
		 FROM oracle_slashes ORDER BY height DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var entry struct {
			Validator string
			Amount    string
			Reason    string
			Height    int64
			Timestamp time.Time
		}
		if err := rows.Scan(&entry.Validator, &entry.Amount, &entry.Reason, &entry.Height, &entry.Timestamp); err != nil {
			return nil, 0, err
		}
		items = append(items, map[string]interface{}{
			"validator_address": entry.Validator,
			"slash_amount":      entry.Amount,
			"reason":            entry.Reason,
			"height":            entry.Height,
			"timestamp":         entry.Timestamp,
		})
	}

	return items, total, rows.Err()
}

func (db *Database) GetComputeRequests(offset, limit int, status string) ([]ComputeRequest, int, error) {
	filters := []string{"1=1"}
	args := []interface{}{}
	if status != "" {
		args = append(args, status)
		filters = append(filters, fmt.Sprintf("status = $%d", len(args)))
	}
	where := strings.Join(filters, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM compute_requests WHERE %s", where)
	var total int
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := fmt.Sprintf(`SELECT request_id, requester, provider, status, task_type, payment_amount,
		 payment_denom, escrow_amount, result_hash, verification_status, created_height, completed_height,
		 created_at
		 FROM compute_requests WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, len(args)-1, len(args))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	requests := make([]ComputeRequest, 0)
	for rows.Next() {
		var request ComputeRequest
		if err := rows.Scan(
			&request.RequestID,
			&request.Requester,
			&request.Provider,
			&request.Status,
			&request.TaskType,
			&request.PaymentAmount,
			&request.PaymentDenom,
			&request.EscrowAmount,
			&request.ResultHash,
			&request.VerificationStatus,
			&request.CreatedHeight,
			&request.CompletedHeight,
			&request.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		requests = append(requests, request)
	}

	return requests, total, rows.Err()
}

func (db *Database) GetComputeRequest(requestID string) (*ComputeRequest, error) {
	var request ComputeRequest
	err := db.QueryRow(
		`SELECT request_id, requester, provider, status, task_type, payment_amount,
		 payment_denom, escrow_amount, result_hash, verification_status, created_height, completed_height,
		 created_at
		 FROM compute_requests WHERE request_id = $1`,
		requestID,
	).Scan(
		&request.RequestID,
		&request.Requester,
		&request.Provider,
		&request.Status,
		&request.TaskType,
		&request.PaymentAmount,
		&request.PaymentDenom,
		&request.EscrowAmount,
		&request.ResultHash,
		&request.VerificationStatus,
		&request.CreatedHeight,
		&request.CompletedHeight,
		&request.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &request, nil
}

func (db *Database) GetComputeResults(requestID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(
		`SELECT provider, result_hash, output_data_hash, status, height, timestamp
		 FROM compute_results WHERE request_id = $1 ORDER BY timestamp DESC`,
		requestID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		var entry struct {
			Provider   string
			ResultHash string
			OutputHash sql.NullString
			Status     string
			Height     int64
			Timestamp  time.Time
		}
		if err := rows.Scan(&entry.Provider, &entry.ResultHash, &entry.OutputHash, &entry.Status, &entry.Height, &entry.Timestamp); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"provider":         entry.Provider,
			"result_hash":      entry.ResultHash,
			"output_data_hash": entry.OutputHash.String,
			"status":           entry.Status,
			"height":           entry.Height,
			"timestamp":        entry.Timestamp,
		})
	}

	return results, rows.Err()
}

func (db *Database) GetComputeVerifications(requestID string) ([]map[string]interface{}, error) {
	rows, err := db.Query(
		`SELECT verifier, status, score, height, timestamp
		 FROM compute_verifications WHERE request_id = $1 ORDER BY timestamp DESC`,
		requestID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var entry struct {
			Verifier  string
			Status    string
			Score     sql.NullString
			Height    int64
			Timestamp time.Time
		}
		if err := rows.Scan(&entry.Verifier, &entry.Status, &entry.Score, &entry.Height, &entry.Timestamp); err != nil {
			return nil, err
		}
		items = append(items, map[string]interface{}{
			"verifier":  entry.Verifier,
			"status":    entry.Status,
			"score":     entry.Score.String,
			"height":    entry.Height,
			"timestamp": entry.Timestamp,
		})
	}

	return items, rows.Err()
}

func (db *Database) GetComputeProviders() ([]map[string]interface{}, error) {
	rows, err := db.Query(
		`SELECT address, stake, active, reputation, total_jobs, completed_jobs, failed_jobs,
		 uptime_30d, avg_completion_time, slash_count
		 FROM compute_providers ORDER BY stake DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]map[string]interface{}, 0)
	for rows.Next() {
		var entry struct {
			Address          string
			Stake            string
			Active           bool
			Reputation       int
			TotalJobs        int64
			CompletedJobs    int64
			FailedJobs       int64
			Uptime30D        float64
			AvgCompletion    float64
			SlashCount       int64
		}
		if err := rows.Scan(
			&entry.Address,
			&entry.Stake,
			&entry.Active,
			&entry.Reputation,
			&entry.TotalJobs,
			&entry.CompletedJobs,
			&entry.FailedJobs,
			&entry.Uptime30D,
			&entry.AvgCompletion,
			&entry.SlashCount,
		); err != nil {
			return nil, err
		}
		items = append(items, map[string]interface{}{
			"address":            entry.Address,
			"stake":              entry.Stake,
			"active":             entry.Active,
			"reputation":         entry.Reputation,
			"total_jobs":         entry.TotalJobs,
			"completed_jobs":     entry.CompletedJobs,
			"failed_jobs":        entry.FailedJobs,
			"uptime_30d":          entry.Uptime30D,
			"avg_completion_time": entry.AvgCompletion,
			"slash_count":        entry.SlashCount,
		})
	}

	return items, rows.Err()
}

func (db *Database) GetComputeProvider(address string) (map[string]interface{}, error) {
	row := db.QueryRow(
		`SELECT address, stake, active, reputation, total_jobs, completed_jobs, failed_jobs,
		 uptime_30d, avg_completion_time, slash_count
		 FROM compute_providers WHERE address = $1`,
		address,
	)
	var entry struct {
		Address       string
		Stake         string
		Active        bool
		Reputation    int
		TotalJobs     int64
		CompletedJobs int64
		FailedJobs    int64
		Uptime30D     float64
		AvgCompletion float64
		SlashCount    int64
	}
	if err := row.Scan(
		&entry.Address,
		&entry.Stake,
		&entry.Active,
		&entry.Reputation,
		&entry.TotalJobs,
		&entry.CompletedJobs,
		&entry.FailedJobs,
		&entry.Uptime30D,
		&entry.AvgCompletion,
		&entry.SlashCount,
	); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"address":            entry.Address,
		"stake":              entry.Stake,
		"active":             entry.Active,
		"reputation":         entry.Reputation,
		"total_jobs":         entry.TotalJobs,
		"completed_jobs":     entry.CompletedJobs,
		"failed_jobs":        entry.FailedJobs,
		"uptime_30d":          entry.Uptime30D,
		"avg_completion_time": entry.AvgCompletion,
		"slash_count":        entry.SlashCount,
	}, nil
}

func (db *Database) GetNetworkStats() (map[string]interface{}, error) {
	var stats struct {
		Date             time.Time
		TotalTxs         int64
		UniqueAccounts   int64
		TotalVolume      float64
		DexTVL           float64
		ActiveValidators int
		AvgBlockTime     sql.NullFloat64
	}
	row := db.QueryRow(
		`SELECT date, total_txs, unique_accounts, total_volume, dex_tvl, active_validators, avg_block_time
		 FROM network_stats ORDER BY date DESC LIMIT 1`,
	)
	if err := row.Scan(
		&stats.Date,
		&stats.TotalTxs,
		&stats.UniqueAccounts,
		&stats.TotalVolume,
		&stats.DexTVL,
		&stats.ActiveValidators,
		&stats.AvgBlockTime,
	); err != nil {
		if err != sql.ErrNoRows {
			return nil, err
		}
	}

	var totalBlocks int64
	_ = db.QueryRow("SELECT COALESCE(MAX(height), 0) FROM blocks").Scan(&totalBlocks)

	return map[string]interface{}{
		"totalBlocks":       totalBlocks,
		"totalTransactions": stats.TotalTxs,
		"activeValidators":  stats.ActiveValidators,
		"averageBlockTime":  stats.AvgBlockTime.Float64,
		"tps":               calculateTPS(stats.TotalTxs, stats.AvgBlockTime.Float64),
		"tvl":               stats.DexTVL,
		"dexVolume24h":      stats.TotalVolume,
		"activeAccounts24h": stats.UniqueAccounts,
	}, nil
}

func (db *Database) GetTransactionChart(period string) ([]map[string]interface{}, error) {
	start, _ := parsePeriodRange(period)
	rows, err := db.Query(
		`SELECT DATE(time) as date, COUNT(*) as tx_count
		 FROM transactions WHERE time >= $1 GROUP BY DATE(time) ORDER BY date ASC`,
		start,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chart := make([]map[string]interface{}, 0)
	for rows.Next() {
		var date time.Time
		var count int64
		if err := rows.Scan(&date, &count); err != nil {
			return nil, err
		}
		chart = append(chart, map[string]interface{}{
			"date":     date,
			"tx_count": count,
		})
	}

	return chart, rows.Err()
}

func (db *Database) GetAddressChart(period string) ([]map[string]interface{}, error) {
	start, _ := parsePeriodRange(period)
	rows, err := db.Query(
		`SELECT DATE(updated_at) as date, COUNT(*) as count
		 FROM accounts WHERE updated_at >= $1 GROUP BY DATE(updated_at) ORDER BY date ASC`,
		start,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chart := make([]map[string]interface{}, 0)
	for rows.Next() {
		var date time.Time
		var count int64
		if err := rows.Scan(&date, &count); err != nil {
			return nil, err
		}
		chart = append(chart, map[string]interface{}{
			"date":  date,
			"count": count,
		})
	}

	return chart, rows.Err()
}

func (db *Database) GetVolumeChart(period string) ([]map[string]interface{}, error) {
	start, _ := parsePeriodRange(period)
	rows, err := db.Query(
		`SELECT DATE(time) as date, SUM(fee_amount::numeric) as volume
		 FROM transactions WHERE time >= $1 GROUP BY DATE(time) ORDER BY date ASC`,
		start,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chart := make([]map[string]interface{}, 0)
	for rows.Next() {
		var date time.Time
		var volume sql.NullFloat64
		if err := rows.Scan(&date, &volume); err != nil {
			return nil, err
		}
		chart = append(chart, map[string]interface{}{
			"date":   date,
			"volume": volume.Float64,
		})
	}

	return chart, rows.Err()
}

func (db *Database) GetGasChart(period string) ([]map[string]interface{}, error) {
	start, _ := parsePeriodRange(period)
	rows, err := db.Query(
		`SELECT DATE(time) as date, SUM(gas_used) as gas_used
		 FROM transactions WHERE time >= $1 GROUP BY DATE(time) ORDER BY date ASC`,
		start,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chart := make([]map[string]interface{}, 0)
	for rows.Next() {
		var date time.Time
		var gasUsed int64
		if err := rows.Scan(&date, &gasUsed); err != nil {
			return nil, err
		}
		chart = append(chart, map[string]interface{}{
			"date":     date,
			"gas_used": gasUsed,
		})
	}

	return chart, rows.Err()
}

func (db *Database) Search(query string) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0)
	if block, err := db.GetBlockByHeight(parseHeight(query)); err == nil {
		results = append(results, map[string]interface{}{"type": "block", "id": block.Height})
	}
	if tx, err := db.GetTransactionByHash(query); err == nil {
		results = append(results, map[string]interface{}{"type": "transaction", "id": tx.Hash})
	}
	if acc, err := db.GetAccount(query); err == nil {
		results = append(results, map[string]interface{}{"type": "address", "id": acc.Address})
	}
	return results, nil
}

func (db *Database) ExportTransactions(address, format, startDate, endDate string) ([]byte, error) {
	query := `SELECT hash, block_height, tx_index, type, sender, status, code, gas_used, gas_wanted,
	 fee_amount, fee_denom, memo, raw_log, time
	 FROM transactions WHERE sender = $1`
	args := []interface{}{address}
	if startDate != "" {
		args = append(args, startDate)
		query += fmt.Sprintf(" AND time >= $%d", len(args))
	}
	if endDate != "" {
		args = append(args, endDate)
		query += fmt.Sprintf(" AND time <= $%d", len(args))
	}
	query += " ORDER BY time DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if format == "json" {
		items := make([]map[string]interface{}, 0)
		for rows.Next() {
			var entry Transaction
			if err := rows.Scan(
				&entry.Hash,
				&entry.BlockHeight,
				&entry.TxIndex,
				&entry.Type,
				&entry.Sender,
				&entry.Status,
				&entry.Code,
				&entry.GasUsed,
				&entry.GasWanted,
				&entry.FeeAmount,
				&entry.FeeDenom,
				&entry.Memo,
				&entry.RawLog,
				&entry.Time,
			); err != nil {
				return nil, err
			}
			items = append(items, map[string]interface{}{
				"hash":         entry.Hash,
				"block_height": entry.BlockHeight,
				"tx_index":     entry.TxIndex,
				"type":         entry.Type,
				"sender":       entry.Sender,
				"status":       entry.Status,
				"code":         entry.Code,
				"gas_used":     entry.GasUsed,
				"gas_wanted":   entry.GasWanted,
				"fee_amount":   entry.FeeAmount,
				"fee_denom":    entry.FeeDenom,
				"memo":         entry.Memo,
				"raw_log":      entry.RawLog,
				"time":         entry.Time,
			})
		}
		return json.Marshal(items)
	}

	builder := &strings.Builder{}
	writer := csv.NewWriter(builder)
	_ = writer.Write([]string{"hash", "block_height", "tx_index", "type", "sender", "status", "code", "gas_used", "gas_wanted", "fee_amount", "fee_denom", "memo", "raw_log", "time"})
	for rows.Next() {
		var entry Transaction
		if err := rows.Scan(
			&entry.Hash,
			&entry.BlockHeight,
			&entry.TxIndex,
			&entry.Type,
			&entry.Sender,
			&entry.Status,
			&entry.Code,
			&entry.GasUsed,
			&entry.GasWanted,
			&entry.FeeAmount,
			&entry.FeeDenom,
			&entry.Memo,
			&entry.RawLog,
			&entry.Time,
		); err != nil {
			return nil, err
		}
		_ = writer.Write([]string{
			entry.Hash,
			fmt.Sprintf("%d", entry.BlockHeight),
			fmt.Sprintf("%d", entry.TxIndex),
			entry.Type,
			entry.Sender,
			entry.Status,
			fmt.Sprintf("%d", entry.Code),
			fmt.Sprintf("%d", entry.GasUsed),
			fmt.Sprintf("%d", entry.GasWanted),
			entry.FeeAmount,
			entry.FeeDenom,
			entry.Memo,
			entry.RawLog,
			entry.Time.Format(time.RFC3339),
		})
	}
	writer.Flush()
	return []byte(builder.String()), writer.Error()
}

func (db *Database) ExportTrades(poolID, format, startDate, endDate string) ([]byte, error) {
	query := `SELECT tx_hash, pool_id, sender, token_in, token_out, amount_in, amount_out, price, fee, time
	 FROM dex_swaps WHERE 1=1`
	args := []interface{}{}
	if poolID != "" {
		args = append(args, poolID)
		query += fmt.Sprintf(" AND pool_id = $%d", len(args))
	}
	if startDate != "" {
		args = append(args, startDate)
		query += fmt.Sprintf(" AND time >= $%d", len(args))
	}
	if endDate != "" {
		args = append(args, endDate)
		query += fmt.Sprintf(" AND time <= $%d", len(args))
	}
	query += " ORDER BY time DESC"

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if format == "json" {
		items := make([]map[string]interface{}, 0)
		for rows.Next() {
			var entry DEXSwap
			if err := rows.Scan(
				&entry.TxHash,
				&entry.PoolID,
				&entry.Sender,
				&entry.TokenIn,
				&entry.TokenOut,
				&entry.AmountIn,
				&entry.AmountOut,
				&entry.Price,
				&entry.Fee,
				&entry.Time,
			); err != nil {
				return nil, err
			}
			items = append(items, map[string]interface{}{
				"tx_hash":    entry.TxHash,
				"pool_id":    entry.PoolID,
				"sender":     entry.Sender,
				"token_in":   entry.TokenIn,
				"token_out":  entry.TokenOut,
				"amount_in":  entry.AmountIn,
				"amount_out": entry.AmountOut,
				"price":      entry.Price,
				"fee":        entry.Fee,
				"time":       entry.Time,
			})
		}
		return json.Marshal(items)
	}

	builder := &strings.Builder{}
	writer := csv.NewWriter(builder)
	_ = writer.Write([]string{"tx_hash", "pool_id", "sender", "token_in", "token_out", "amount_in", "amount_out", "price", "fee", "time"})
	for rows.Next() {
		var entry DEXSwap
		if err := rows.Scan(
			&entry.TxHash,
			&entry.PoolID,
			&entry.Sender,
			&entry.TokenIn,
			&entry.TokenOut,
			&entry.AmountIn,
			&entry.AmountOut,
			&entry.Price,
			&entry.Fee,
			&entry.Time,
		); err != nil {
			return nil, err
		}
		_ = writer.Write([]string{
			entry.TxHash,
			entry.PoolID,
			entry.Sender,
			entry.TokenIn,
			entry.TokenOut,
			fmt.Sprintf("%f", entry.AmountIn),
			fmt.Sprintf("%f", entry.AmountOut),
			fmt.Sprintf("%f", entry.Price),
			fmt.Sprintf("%f", entry.Fee),
			entry.Time.Format(time.RFC3339),
		})
	}
	writer.Flush()
	return []byte(builder.String()), writer.Error()
}

func parseHeight(query string) int64 {
	var height int64
	_, _ = fmt.Sscanf(query, "%d", &height)
	return height
}

func parsePeriodRange(period string) (time.Time, time.Time) {
	now := time.Now().UTC()
	switch period {
	case "24h":
		return now.Add(-24 * time.Hour), now
	case "7d":
		return now.Add(-7 * 24 * time.Hour), now
	case "30d":
		return now.Add(-30 * 24 * time.Hour), now
	default:
		return now.Add(-24 * time.Hour), now
	}
}

func calculateTPS(totalTxs int64, avgBlockTime float64) float64 {
	if avgBlockTime <= 0 {
		return 0
	}
	return float64(totalTxs) / (avgBlockTime * 86400)
}
