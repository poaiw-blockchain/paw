# Compute Module

## Overview

The Compute module enables secure API key aggregation and compute task routing for the PAW blockchain. It allows users to request off-chain computation while protecting sensitive API keys through Trusted Execution Environments (TEEs), creating a decentralized compute marketplace.

## Features

- **Secure API Key Management**: TEE-protected key storage prevents extraction
- **Provider Registration**: Stake-based provider enrollment with quality guarantees
- **Task Routing**: Automatic assignment of compute requests to providers
- **Result Verification**: Cryptographic verification of computation results
- **Fee Management**: Automatic fee distribution and reward settlement
- **Time-Limited Execution**: Minute-level accounting with automatic key destruction

## Architecture

### Components

```
Compute Module
├── Keeper
│   ├── Provider Management
│   ├── Request Routing
│   ├── Task Tracking
│   ├── Result Verification
│   └── Fee Distribution
├── Types
│   ├── Provider
│   ├── ComputeTask
│   ├── ComputeRequest
│   └── Params
└── Messages
    ├── MsgRegisterProvider
    ├── MsgRequestCompute
    ├── MsgSubmitResult
    └── MsgUpdateParams
```

### State Storage

The compute module stores the following data:

| Key | Value | Description |
|-----|-------|-------------|
| `0x03 \| task_id` | ComputeTask | Legacy task record |
| `0x04 \| provider_address` | Provider | Registered compute provider |
| `0x05 \| request_id` | ComputeRequest | Active compute request |
| `0x06` | NextRequestID | Request ID counter |

## How It Works

### 1. Provider Registration

Compute providers register by staking tokens:

```
┌─────────────┐
│  Provider   │
│  Stakes PAW │
└──────┬──────┘
       │
       │ 1. MsgRegisterProvider(endpoint, stake)
       │
┌──────▼──────────────────────────┐
│     Compute Module Keeper       │
│  - Verify stake >= min_stake    │
│  - Store provider record        │
│  - Mark as active               │
└─────────────────────────────────┘
```

### 2. Compute Request Flow

```
┌─────────────┐
│  Requester  │
│ (User/DApp) │
└──────┬──────┘
       │
       │ 1. MsgRequestCompute(api_url, max_fee)
       │
┌──────▼──────────────────────────┐
│     Compute Module              │
│  - Generate request_id          │
│  - Store in PENDING             │
│  - Emit event                   │
└──────┬──────────────────────────┘
       │
       │ 2. Provider discovers request
       │
┌──────▼──────────┐
│  TEE Provider   │
│  - Load API key │
│  - Execute task │
│  - Generate     │
│    result       │
└──────┬──────────┘
       │
       │ 3. MsgSubmitResult(request_id, result)
       │
┌──────▼──────────────────────────┐
│     Compute Module              │
│  - Verify provider              │
│  - Validate result              │
│  - Mark COMPLETED               │
│  - Distribute fees              │
└─────────────────────────────────┘
```

### 3. TEE Security Model

```
┌────────────────────────────────────────┐
│      Trusted Execution Environment     │
│  ┌──────────────────────────────────┐  │
│  │     Encrypted Memory Region      │  │
│  │  ┌────────────────────────────┐  │  │
│  │  │   1. API Key Loaded        │  │  │
│  │  │   2. Request Executed      │  │  │
│  │  │   3. Result Generated      │  │  │
│  │  │   4. Key Destroyed         │  │  │
│  │  └────────────────────────────┘  │  │
│  │  API Key never leaves TEE        │  │
│  └──────────────────────────────────┘  │
│  Hardware-level isolation               │
└────────────────────────────────────────┘
```

## Usage Examples

### Register as Compute Provider

Providers must stake tokens to participate:

```bash
# Register with 1000 PAW stake
pawd tx compute register-provider \
  --endpoint "https://compute.provider.com/api" \
  --stake 1000000000upaw \
  --from provider-key \
  --chain-id paw-mainnet-1

# Verify registration
pawd query compute provider paw1xxxxx...
```

### Submit Compute Request

Users request compute tasks:

```bash
# Request computation
pawd tx compute request-compute \
  --api-url "https://api.openai.com/v1/chat/completions" \
  --max-fee 100000upaw \
  --from user-key

# Response includes request_id
# {
#   "request_id": 123
# }

# Check request status
pawd query compute request 123
```

### Submit Computation Result

Providers submit results:

```bash
# Submit result (provider only)
pawd tx compute submit-result \
  --request-id 123 \
  --result '{"response": "computation complete"}' \
  --from provider-key

# Verify completion
pawd query compute request 123
# {
#   "status": "COMPLETED",
#   "result": "..."
# }
```

## Parameters

The compute module has the following configurable parameters:

| Parameter | Default | Description |
|-----------|---------|-------------|
| `min_stake` | 1000 PAW | Minimum stake for provider registration |
| `max_task_duration` | 300s | Maximum task execution time |
| `fee_percentage` | 0.05 | Platform fee (5%) |
| `slash_fraction` | 0.10 | Slash amount for failed tasks (10%) |

### Update Parameters

Parameters can be updated via governance:

```bash
# Submit parameter change proposal
pawd tx gov submit-proposal param-change proposal.json \
  --from proposer-key

# proposal.json
{
  "title": "Update Compute Min Stake",
  "description": "Increase minimum stake to 5000 PAW",
  "changes": [
    {
      "subspace": "compute",
      "key": "MinStake",
      "value": "5000000000"
    }
  ]
}
```

## API Integration

### For DApp Developers

Integrate compute requests in your application:

```javascript
// JavaScript/TypeScript example
import { SigningStargateClient } from "@cosmjs/stargate";

async function requestComputation(apiUrl, maxFee) {
  const client = await SigningStargateClient.connectWithSigner(
    "https://rpc.paw.network",
    signer
  );

  const msg = {
    typeUrl: "/paw.compute.MsgRequestCompute",
    value: {
      requester: userAddress,
      apiUrl: apiUrl,
      maxFee: maxFee,
    },
  };

  const result = await client.signAndBroadcast(
    userAddress,
    [msg],
    "auto"
  );

  const requestId = result.events
    .find(e => e.type === "compute_requested")
    .attributes.find(a => a.key === "request_id").value;

  return requestId;
}

async function getComputeResult(requestId) {
  const response = await fetch(
    `https://api.paw.network/compute/requests/${requestId}`
  );
  return await response.json();
}

// Usage
const requestId = await requestComputation(
  "https://api.openai.com/v1/completions",
  "100000"
);

// Poll for result
const result = await getComputeResult(requestId);
console.log(result);
```

### For Compute Providers

Run a compute provider node:

```bash
# Install provider software
go install github.com/paw-chain/compute-provider@latest

# Configure provider
cat > provider-config.yaml <<EOF
chain_id: paw-mainnet-1
provider_address: paw1xxxxx...
endpoint: https://compute.provider.com/api
tee_enabled: true
tee_attestation_url: https://attestation.provider.com
stake_amount: 5000000000upaw
max_concurrent_tasks: 10
supported_apis:
  - openai
  - anthropic
  - google-cloud
EOF

# Start provider
compute-provider start --config provider-config.yaml
```

### TEE Configuration

Configure Trusted Execution Environment:

```yaml
# tee-config.yaml
tee_type: sgx  # Intel SGX
enclave_path: /usr/local/lib/compute-enclave.signed.so
attestation:
  enabled: true
  endpoint: https://attestation.intel.com
  mrenclave: "abc123..."  # Expected enclave measurement
security:
  key_rotation_interval: 24h
  max_key_age: 1h
  auto_destroy: true
monitoring:
  enabled: true
  metrics_endpoint: :9090
```

## Request Lifecycle

### State Transitions

```
PENDING → ASSIGNED → EXECUTING → COMPLETED
   ↓         ↓          ↓            ↓
CANCELLED  FAILED    TIMEOUT      SUCCESS
```

### Status Codes

| Status | Code | Description |
|--------|------|-------------|
| PENDING | 0 | Request created, awaiting provider |
| ASSIGNED | 1 | Provider assigned, not started |
| EXECUTING | 2 | Computation in progress |
| COMPLETED | 3 | Successfully completed |
| FAILED | 4 | Execution failed |
| TIMEOUT | 5 | Exceeded max duration |
| CANCELLED | 6 | Cancelled by requester |

## Fee Structure

### Fee Distribution

```
Total Fee: 100 PAW
├─ Provider: 90 PAW (90%)
├─ Protocol: 5 PAW (5%)
└─ Validator Rewards: 5 PAW (5%)
```

### Fee Calculation Example

```go
// Request with 100 PAW max fee
maxFee := math.NewInt(100_000_000) // 100 PAW in upaw

// Provider receives 90%
providerFee := maxFee.MulRaw(90).QuoRaw(100) // 90 PAW

// Protocol receives 5%
protocolFee := maxFee.MulRaw(5).QuoRaw(100)  // 5 PAW

// Validators receive 5%
validatorFee := maxFee.MulRaw(5).QuoRaw(100) // 5 PAW
```

## Security Considerations

### TEE Attestation

Providers must prove TEE authenticity:

```bash
# Provider generates attestation
compute-provider attest \
  --enclave-path /path/to/enclave.so \
  --output attestation.json

# Submit attestation to chain
pawd tx compute submit-attestation \
  --attestation attestation.json \
  --from provider-key
```

### Key Protection

API keys are protected through:

1. **Hardware Isolation**: Keys only exist in TEE memory
2. **Time Limits**: Keys auto-destruct after task completion
3. **Encrypted Communication**: All key transfers encrypted
4. **Audit Logging**: All key accesses logged immutably

### Slashing Protection

Providers are slashed for:

- Failed task execution
- Timeout violations
- Invalid results
- TEE attestation failures

```
Slash Amount = stake_amount * slash_fraction
Example: 1000 PAW * 0.10 = 100 PAW slashed
```

## Events

### RequestCreated

```json
{
  "type": "compute_requested",
  "attributes": [
    {"key": "request_id", "value": "123"},
    {"key": "requester", "value": "paw1xxx..."},
    {"key": "api_url", "value": "https://api.example.com"},
    {"key": "max_fee", "value": "100000000"}
  ]
}
```

### ResultSubmitted

```json
{
  "type": "result_submitted",
  "attributes": [
    {"key": "request_id", "value": "123"},
    {"key": "provider", "value": "paw1yyy..."},
    {"key": "status", "value": "COMPLETED"}
  ]
}
```

### ProviderRegistered

```json
{
  "type": "provider_registered",
  "attributes": [
    {"key": "provider", "value": "paw1yyy..."},
    {"key": "endpoint", "value": "https://compute.provider.com"},
    {"key": "stake", "value": "1000000000"}
  ]
}
```

## Testing

### Unit Tests

```bash
# Run all compute tests
go test ./x/compute/...

# Run with coverage
go test -cover ./x/compute/...

# Run keeper tests
go test ./x/compute/keeper/...
```

### Test Coverage

| Component | Coverage |
|-----------|----------|
| Keeper | 85% |
| Types | 80% |
| Overall | 82% |

### Example Tests

```go
func TestProviderRegistration(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    msg := &types.MsgRegisterProvider{
        Provider: "paw1xxx...",
        Endpoint: "https://provider.com",
        Stake:    math.NewInt(1000_000_000),
    }

    _, err := app.ComputeKeeper.RegisterProvider(ctx, msg)
    require.NoError(t, err)

    // Verify provider stored
    provider, found := app.ComputeKeeper.GetProvider(ctx, msg.Provider)
    require.True(t, found)
    require.Equal(t, msg.Endpoint, provider.Endpoint)
    require.True(t, provider.Active)
}

func TestComputeRequestFlow(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Register provider
    RegisterTestProvider(
        &app.ComputeKeeper,
        ctx,
        "paw1provider...",
        "https://provider.com",
        math.NewInt(1000_000_000),
    )

    // Submit request
    requestId, err := SubmitTestRequest(
        &app.ComputeKeeper,
        ctx,
        "paw1user...",
        "https://api.example.com",
        math.NewInt(100_000_000),
    )
    require.NoError(t, err)

    // Submit result
    submitMsg := &types.MsgSubmitResult{
        RequestId: requestId,
        Provider:  "paw1provider...",
        Result:    "computation complete",
    }

    _, err = app.ComputeKeeper.SubmitResult(ctx, submitMsg)
    require.NoError(t, err)

    // Verify completed
    request, found := app.ComputeKeeper.GetRequest(ctx, requestId)
    require.True(t, found)
    require.Equal(t, types.RequestStatus_COMPLETED, request.Status)
}
```

## CLI Reference

### Transactions

```bash
# Register provider
pawd tx compute register-provider [endpoint] [stake] [flags]

# Request compute
pawd tx compute request-compute [api-url] [max-fee] [flags]

# Submit result (provider only)
pawd tx compute submit-result [request-id] [result] [flags]

# Update params (governance)
pawd tx compute update-params [params-json] [flags]
```

### Queries

```bash
# Get provider info
pawd query compute provider [address]

# Get request details
pawd query compute request [request-id]

# Get all active providers
pawd query compute providers

# Get all pending requests
pawd query compute pending-requests

# Get module params
pawd query compute params
```

## REST API

### Endpoints

```
GET  /compute/providers              # List providers
GET  /compute/providers/{address}    # Get provider
GET  /compute/requests               # List requests
GET  /compute/requests/{id}          # Get request
POST /compute/requests               # Create request
GET  /compute/params                 # Get params
```

### Example API Calls

```bash
# Get provider details
curl http://localhost:1317/compute/providers/paw1xxx...

# Get request status
curl http://localhost:1317/compute/requests/123

# List pending requests
curl http://localhost:1317/compute/requests?status=pending
```

## Future Enhancements

### Planned Features

1. **Multi-Provider Consensus**
   - Multiple providers execute same task
   - Results aggregated via consensus
   - Enhanced reliability and security

2. **zkML Verification**
   - Zero-knowledge proofs for computations
   - Verifiable computation results
   - Privacy-preserving execution

3. **Resource Marketplace**
   - GPU/CPU resource allocation
   - Dynamic pricing based on demand
   - Resource reservation system

4. **Cross-Chain Compute**
   - IBC compute request routing
   - Multi-chain provider network
   - Unified compute marketplace

## Resources

- [Compute Keeper Documentation](./keeper/)
- [Provider Setup Guide](./docs/provider-setup.md)
- [TEE Integration Guide](./docs/tee-integration.md)
- [PAW Architecture](../../ARCHITECTURE.md)
- [Testing Guide](../../TESTING.md)

## Support

- **GitHub Issues**: [Report compute issues](https://github.com/decristofaroj/paw/issues)
- **Documentation**: [Full PAW docs](../../README.md)
- **Provider Chat**: [Discord #compute-providers](https://discord.gg/paw)

---

**Module Version**: 1.0
**Test Coverage**: 82%
**Status**: Beta
**Maintainer**: PAW Development Team
