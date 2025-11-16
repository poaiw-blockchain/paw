# Circuit Breaker CLI Examples

This document provides examples of how to interact with the circuit breaker system via CLI commands.

## Query Commands

### 1. Query Circuit Breaker Configuration

```bash
# Query the global circuit breaker configuration
pawd query dex circuit-breaker-config

# Example output:
# config:
#   threshold_1min: "0.100000000000000000"    # 10%
#   threshold_5min: "0.200000000000000000"    # 20%
#   threshold_15min: "0.250000000000000000"   # 25%
#   threshold_1hour: "0.300000000000000000"   # 30%
#   cooldown_period: 600                       # 10 minutes
#   enable_gradual_resume: true
#   resume_volume_factor: "0.500000000000000000"  # 50%
```

### 2. Query Circuit Breaker State for a Pool

```bash
# Query circuit breaker state for pool ID 1
pawd query dex circuit-breaker-state 1

# Example output (tripped):
# state:
#   pool_id: 1
#   is_tripped: true
#   trip_reason: "Price volatility exceeded 1 minute threshold: 12.50% change in 1 minute (threshold: 10.00%)"
#   tripped_at: 12345              # Block height
#   tripped_at_time: 1699876543    # Unix timestamp
#   price_at_trip: "1.250000000000000000"
#   can_resume_at: 1699877143      # Unix timestamp
#   gradual_resume: true
#   resume_started_at: 0

# Example output (not tripped):
# state:
#   pool_id: 1
#   is_tripped: false
#   trip_reason: ""
#   tripped_at: 0
#   tripped_at_time: 0
#   price_at_trip: "0"
#   can_resume_at: 0
#   gradual_resume: false
#   resume_started_at: 0
```

### 3. Query Circuit Breaker Status (Human-Readable)

```bash
# Query user-friendly status for pool ID 1
pawd query dex circuit-breaker-status 1

# Example output (in cooldown):
# pool_id: 1
# is_tripped: true
# status: "cooldown"
# trip_reason: "Price volatility exceeded 1 minute threshold"
# seconds_until_resume: 245
# in_gradual_resume: false
# max_swap_percentage: "0%"

# Example output (gradual resume):
# pool_id: 1
# is_tripped: false
# status: "gradual_resume"
# trip_reason: ""
# seconds_until_resume: 0
# in_gradual_resume: true
# max_swap_percentage: "50%"

# Example output (normal):
# pool_id: 1
# is_tripped: false
# status: "normal"
# trip_reason: ""
# seconds_until_resume: 0
# in_gradual_resume: false
# max_swap_percentage: "100%"
```

### 4. Query All Circuit Breaker States

```bash
# Query all circuit breaker states across all pools
pawd query dex all-circuit-breaker-states

# Example output:
# states:
# - pool_id: 1
#   is_tripped: true
#   trip_reason: "Price volatility exceeded threshold"
#   ...
# - pool_id: 2
#   is_tripped: false
#   ...
```

### 5. Query Active Circuit Breakers

```bash
# Query only pools with active (tripped) circuit breakers
pawd query dex active-circuit-breakers

# Example output:
# states:
# - pool_id: 1
#   is_tripped: true
#   trip_reason: "Price volatility exceeded 1 minute threshold"
#   tripped_at: 12345
#   can_resume_at: 1699877143
# - pool_id: 5
#   is_tripped: true
#   trip_reason: "Price volatility exceeded 5 minute threshold"
#   tripped_at: 12340
#   can_resume_at: 1699877100
```

## Governance Proposal Commands

### 1. Submit Circuit Breaker Configuration Update Proposal

```bash
# Create a proposal to update circuit breaker configuration
pawd tx gov submit-proposal circuit-breaker-config \
  --title="Update Circuit Breaker Thresholds" \
  --description="Increase thresholds for more volatile market conditions" \
  --threshold-1min="0.15" \
  --threshold-5min="0.25" \
  --threshold-15min="0.35" \
  --threshold-1hour="0.50" \
  --cooldown-period=900 \
  --enable-gradual-resume=true \
  --resume-volume-factor="0.30" \
  --deposit="10000000upaw" \
  --from=mykey \
  --chain-id=paw-1

# Or submit via JSON file:
cat > circuit-breaker-config-proposal.json <<EOF
{
  "title": "Update Circuit Breaker Thresholds",
  "description": "Increase thresholds for more volatile market conditions",
  "config": {
    "threshold_1min": "0.150000000000000000",
    "threshold_5min": "0.250000000000000000",
    "threshold_15min": "0.350000000000000000",
    "threshold_1hour": "0.500000000000000000",
    "cooldown_period": 900,
    "enable_gradual_resume": true,
    "resume_volume_factor": "0.300000000000000000"
  }
}
EOF

pawd tx gov submit-proposal circuit-breaker-config circuit-breaker-config-proposal.json \
  --deposit="10000000upaw" \
  --from=mykey \
  --chain-id=paw-1
```

### 2. Submit Circuit Breaker Resume Proposal

```bash
# Create a proposal to override circuit breaker and resume trading
pawd tx gov submit-proposal circuit-breaker-resume \
  --title="Resume Trading on Pool 1" \
  --description="Market conditions have stabilized after investigation. Safe to resume trading." \
  --pool-id=1 \
  --deposit="10000000upaw" \
  --from=mykey \
  --chain-id=paw-1

# Or submit via JSON file:
cat > circuit-breaker-resume-proposal.json <<EOF
{
  "title": "Resume Trading on Pool 1",
  "description": "Market conditions have stabilized after investigation. Safe to resume trading.",
  "pool_id": 1
}
EOF

pawd tx gov submit-proposal circuit-breaker-resume circuit-breaker-resume-proposal.json \
  --deposit="10000000upaw" \
  --from=mykey \
  --chain-id=paw-1
```

### 3. Vote on Circuit Breaker Proposals

```bash
# Vote yes on a proposal
pawd tx gov vote 1 yes \
  --from=mykey \
  --chain-id=paw-1

# Vote no on a proposal
pawd tx gov vote 1 no \
  --from=mykey \
  --chain-id=paw-1

# Vote abstain
pawd tx gov vote 1 abstain \
  --from=mykey \
  --chain-id=paw-1

# Vote no with veto
pawd tx gov vote 1 no_with_veto \
  --from=mykey \
  --chain-id=paw-1
```

## Monitoring Scripts

### 1. Monitor Circuit Breaker Status

```bash
#!/bin/bash
# monitor-circuit-breakers.sh
# Continuously monitor circuit breaker status

while true; do
  echo "=== Circuit Breaker Status at $(date) ==="

  # Get all active circuit breakers
  ACTIVE=$(pawd query dex active-circuit-breakers --output json)

  if [ "$(echo $ACTIVE | jq '.states | length')" -gt 0 ]; then
    echo "⚠️  ACTIVE CIRCUIT BREAKERS DETECTED!"
    echo $ACTIVE | jq '.states[] | {pool_id, trip_reason, tripped_at}'

    # Send alert (example with webhook)
    curl -X POST https://your-monitoring-system/alert \
      -H "Content-Type: application/json" \
      -d "$ACTIVE"
  else
    echo "✓ No active circuit breakers"
  fi

  echo ""
  sleep 60  # Check every minute
done
```

### 2. Check Pool Status Before Trading

```bash
#!/bin/bash
# check-pool-status.sh
# Check if a pool is safe to trade

POOL_ID=$1

if [ -z "$POOL_ID" ]; then
  echo "Usage: $0 <pool_id>"
  exit 1
fi

STATUS=$(pawd query dex circuit-breaker-status $POOL_ID --output json)

IS_TRIPPED=$(echo $STATUS | jq -r '.is_tripped')
STATUS_TEXT=$(echo $STATUS | jq -r '.status')
MAX_SWAP=$(echo $STATUS | jq -r '.max_swap_percentage')

echo "Pool $POOL_ID Status: $STATUS_TEXT"

if [ "$IS_TRIPPED" = "true" ]; then
  REASON=$(echo $STATUS | jq -r '.trip_reason')
  SECONDS_LEFT=$(echo $STATUS | jq -r '.seconds_until_resume')
  echo "❌ Trading is PAUSED"
  echo "Reason: $REASON"
  echo "Resume in: $SECONDS_LEFT seconds"
  exit 1
elif [ "$STATUS_TEXT" = "gradual_resume" ]; then
  echo "⚠️  Gradual resume mode - Limited to $MAX_SWAP of pool reserves"
  exit 0
else
  echo "✓ Normal trading - No restrictions"
  exit 0
fi
```

### 3. Get Circuit Breaker Summary

```bash
#!/bin/bash
# circuit-breaker-summary.sh
# Get a summary of all circuit breaker states

echo "=== Circuit Breaker Configuration ==="
pawd query dex circuit-breaker-config --output json | jq '{
  threshold_1min,
  threshold_5min,
  threshold_15min,
  threshold_1hour,
  cooldown_minutes: (.cooldown_period / 60),
  gradual_resume: .enable_gradual_resume,
  max_swap_on_resume: .resume_volume_factor
}'

echo ""
echo "=== Active Circuit Breakers ==="
ACTIVE=$(pawd query dex active-circuit-breakers --output json)
ACTIVE_COUNT=$(echo $ACTIVE | jq '.states | length')
echo "Total active: $ACTIVE_COUNT"

if [ "$ACTIVE_COUNT" -gt 0 ]; then
  echo $ACTIVE | jq '.states[] | {
    pool_id,
    status: (if .is_tripped then "TRIPPED" else "NORMAL" end),
    reason: .trip_reason,
    tripped_at_block: .tripped_at
  }'
fi

echo ""
echo "=== All Pools Summary ==="
pawd query dex all-pools --output json | jq -r '.pools[] | .id' | while read POOL_ID; do
  STATUS=$(pawd query dex circuit-breaker-status $POOL_ID --output json)
  STATUS_TEXT=$(echo $STATUS | jq -r '.status')
  MAX_SWAP=$(echo $STATUS | jq -r '.max_swap_percentage')
  printf "Pool %-3s: %-15s (Max swap: %s)\n" "$POOL_ID" "$STATUS_TEXT" "$MAX_SWAP"
done
```

## Event Monitoring with WebSocket

### Subscribe to Circuit Breaker Events

```bash
# Subscribe to circuit breaker tripped events
pawd query txs --events 'circuit_breaker_tripped.pool_id=1' --limit 100

# Subscribe to circuit breaker resumed events
pawd query txs --events 'circuit_breaker_resumed.pool_id=1' --limit 100

# WebSocket subscription (using wscat or similar)
wscat -c ws://localhost:26657/websocket

# Then send:
{
  "jsonrpc": "2.0",
  "method": "subscribe",
  "id": 1,
  "params": {
    "query": "tm.event='Tx' AND circuit_breaker_tripped.pool_id EXISTS"
  }
}
```

## Alerting Integration

### 1. Slack Alert on Circuit Breaker Trip

```bash
#!/bin/bash
# slack-alert.sh
# Send Slack notification when circuit breaker trips

WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

POOL_ID=$1
REASON=$2

MESSAGE="{
  \"text\": \"🚨 Circuit Breaker Tripped!\",
  \"blocks\": [
    {
      \"type\": \"section\",
      \"text\": {
        \"type\": \"mrkdwn\",
        \"text\": \"*Circuit Breaker Alert*\n\nPool ID: $POOL_ID\nReason: $REASON\"
      }
    }
  ]
}"

curl -X POST $WEBHOOK_URL \
  -H 'Content-Type: application/json' \
  -d "$MESSAGE"
```

### 2. PagerDuty Integration

```bash
#!/bin/bash
# pagerduty-alert.sh
# Create PagerDuty incident on circuit breaker trip

INTEGRATION_KEY="your-integration-key"
POOL_ID=$1
REASON=$2

PAYLOAD="{
  \"routing_key\": \"$INTEGRATION_KEY\",
  \"event_action\": \"trigger\",
  \"payload\": {
    \"summary\": \"Circuit Breaker Tripped - Pool $POOL_ID\",
    \"severity\": \"critical\",
    \"source\": \"paw-dex\",
    \"custom_details\": {
      \"pool_id\": \"$POOL_ID\",
      \"reason\": \"$REASON\"
    }
  }
}"

curl -X POST https://events.pagerduty.com/v2/enqueue \
  -H 'Content-Type: application/json' \
  -d "$PAYLOAD"
```

## Testing Commands

### 1. Simulate Circuit Breaker Scenarios

```bash
# Test 1: Check configuration is correct
pawd query dex circuit-breaker-config

# Test 2: Create a test pool
pawd tx dex create-pool token_a token_b 1000000 1000000 \
  --from=testkey \
  --chain-id=paw-testnet-1

# Test 3: Check initial state (should be normal)
pawd query dex circuit-breaker-status 1

# Test 4: Perform swaps to trigger volatility
# (Multiple large swaps in quick succession)

# Test 5: Verify circuit breaker tripped
pawd query dex circuit-breaker-state 1

# Test 6: Try to swap (should fail)
pawd tx dex swap 1 token_a token_b 1000 1 \
  --from=testkey \
  --chain-id=paw-testnet-1
# Expected: Error containing "circuit breaker"

# Test 7: Submit resume proposal
pawd tx gov submit-proposal circuit-breaker-resume \
  --pool-id=1 \
  --from=testkey

# Test 8: Vote and pass proposal
pawd tx gov vote 1 yes --from=testkey

# Test 9: Verify trading resumed
pawd query dex circuit-breaker-status 1
```

## Troubleshooting

### Circuit Breaker Not Triggering

```bash
# Check if price observations are being recorded
pawd query dex twap 1 --window=60

# Check circuit breaker configuration
pawd query dex circuit-breaker-config

# Verify thresholds are reasonable for the price movement
```

### Unable to Resume Trading

```bash
# Check current state
pawd query dex circuit-breaker-state 1

# Check if cooldown period has passed
STATUS=$(pawd query dex circuit-breaker-status 1 --output json)
echo $STATUS | jq '.seconds_until_resume'

# If still in cooldown, submit governance proposal to override
```

### Queries Failing

```bash
# Verify node is synced
pawd status

# Check if module is enabled
pawd query dex params

# Verify pool exists
pawd query dex pool 1
```
