# PAW Blockchain CLI Quick Reference

Quick reference for common CLI commands across all modules.

## Compute Module

### Provider Operations
```bash
# Register provider
pawd tx compute register-provider --moniker "Name" --endpoint "URL" --cpu-cores 4 --memory-mb 8192 --timeout-seconds 3600 --cpu-price 0.001 --memory-price 0.0005 --gpu-price 0.1 --storage-price 0.0001 --amount 1000000 --from key

# Update provider
pawd tx compute update-provider --moniker "NewName" --from key

# Deactivate provider
pawd tx compute deactivate-provider --from key

# Query provider
pawd query compute provider [address]
pawd query compute providers
pawd query compute active-providers
```

### Compute Jobs
```bash
# Submit job
pawd tx compute submit-request --container-image "ubuntu:22.04" --command "cmd" --cpu-cores 2 --memory-mb 4096 --timeout-seconds 1800 --max-payment 100000 --from key

# Cancel job
pawd tx compute cancel-request [request-id] --from key

# Submit result
pawd tx compute submit-result [request-id] --output-hash "hash" --output-url "url" --exit-code 0 --from key

# Query requests
pawd query compute request [id]
pawd query compute requests
pawd query compute requests-by-requester [address]
pawd query compute requests-by-provider [address]
pawd query compute requests-by-status [status]
pawd query compute result [request-id]
```

### Disputes & Appeals
```bash
# Create dispute
pawd tx compute create-dispute [request-id] --reason "text" --deposit-amount 1000000 --from key

# Vote on dispute
pawd tx compute vote-dispute [dispute-id] --vote favor_requester --justification "text" --from validator

# Appeal slash
pawd tx compute appeal-slashing [slash-id] --justification "text" --deposit-amount 1000000 --from key

# Query disputes
pawd query compute dispute [id]
pawd query compute disputes
pawd query compute slash-records
```

## DEX Module

### Pool Operations
```bash
# Create pool
pawd tx dex create-pool [token-a] [amount-a] [token-b] [amount-b] --from key

# Add liquidity
pawd tx dex add-liquidity [pool-id] [amount-a] [amount-b] --from key

# Remove liquidity
pawd tx dex remove-liquidity [pool-id] [shares] --from key

# Query pools
pawd query dex pool [pool-id]
pawd query dex pools
pawd query dex pool-by-tokens [token-a] [token-b]
pawd query dex liquidity [pool-id] [provider]
```

### Trading
```bash
# Simulate swap
pawd query dex simulate-swap [pool-id] [token-in] [token-out] [amount-in]

# Execute swap
pawd tx dex swap [pool-id] [token-in] [amount-in] [token-out] [min-amount-out] --from key
```

## Oracle Module

### Price Feeds
```bash
# Submit price
pawd tx oracle submit-price [validator] [asset] [price] --from key

# Delegate feeder
pawd tx oracle delegate-feeder [delegate-address] --from validator-key

# Query prices
pawd query oracle price [asset]
pawd query oracle prices
pawd query oracle validator-price [validator] [asset]
```

### Validator Info
```bash
# Query validators
pawd query oracle validator [validator-address]
pawd query oracle validators
```

## Common Patterns

### All Modules
```bash
# Query parameters
pawd query [module] params

# All queries support pagination
pawd query [module] [command] --limit 10 --offset 20
```

### Transaction Flags
```bash
--from [key-name]           # Signer key
--chain-id [chain-id]       # Chain identifier
--gas auto                  # Auto estimate gas
--gas-adjustment 1.5        # Gas adjustment multiplier
--fees 1000upaw            # Transaction fees
-y                         # Skip confirmation
```

### Query Flags
```bash
--output json              # JSON output
--height [height]          # Query at specific height
--node [url]              # RPC node URL
```

## Status Values

### Compute Request Status
- `pending` - Awaiting assignment
- `assigned` - Assigned to provider
- `processing` - Being executed
- `completed` - Finished successfully
- `failed` - Execution failed
- `cancelled` - Cancelled by requester
- `disputed` - Under dispute

### Dispute Status
- `pending` - Awaiting votes
- `voting` - Active voting period
- `resolved_favor_requester` - Requester won
- `resolved_favor_provider` - Provider won
- `cancelled` - Dispute cancelled

### Appeal Status
- `pending` - Awaiting votes
- `voting` - Active voting period
- `approved` - Appeal approved
- `rejected` - Appeal rejected
- `cancelled` - Appeal cancelled

## Examples

### Full Compute Workflow
```bash
# 1. Register as provider
pawd tx compute register-provider --moniker "MyProvider" --endpoint "https://api.github.com" --cpu-cores 8 --memory-mb 16384 --timeout-seconds 3600 --cpu-price 0.001 --memory-price 0.0005 --gpu-price 0.1 --storage-price 0.0001 --amount 1000000 --from provider

# 2. User submits job
pawd tx compute submit-request --container-image "ubuntu:22.04" --command "python,script.py" --cpu-cores 4 --memory-mb 8192 --timeout-seconds 1800 --max-payment 100000 --from user

# 3. Provider submits result
pawd tx compute submit-result 1 --output-hash "abc123" --output-url "https://storage.github.com/result.tar.gz" --exit-code 0 --from provider
```

### Full DEX Workflow
```bash
# 1. Create pool
pawd tx dex create-pool upaw 1000000 uusdt 2000000 --from creator

# 2. Simulate swap
pawd query dex simulate-swap 1 upaw uusdt 100000

# 3. Execute swap
pawd tx dex swap 1 upaw 100000 uusdt 195000 --from trader

# 4. Add liquidity
pawd tx dex add-liquidity 1 500000 1000000 --from provider
```

### Full Oracle Workflow
```bash
# 1. Delegate feeder
pawd tx oracle delegate-feeder $(pawd keys show feeder -a) --from validator

# 2. Submit prices
pawd tx oracle submit-price $(pawd keys show validator --bech val -a) BTC 50000 --from feeder
pawd tx oracle submit-price $(pawd keys show validator --bech val -a) ETH 3000 --from feeder

# 3. Query prices
pawd query oracle price BTC
pawd query oracle prices
```

## Tips

1. **Use `--help`** on any command for detailed information
2. **Use `--dry-run`** to test transactions without broadcasting
3. **Use `--generate-only`** to generate unsigned transactions
4. **Use `-y`** to skip confirmation prompts (be careful!)
5. **Use `--gas auto`** for automatic gas estimation
6. **Always simulate swaps** before executing to avoid slippage
7. **Check pool liquidity** before large swaps
8. **Monitor oracle miss counters** to avoid slashing
