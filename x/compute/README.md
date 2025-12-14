# Compute Module

## Overview

The Compute module provides a decentralized, verifiable off-chain computation marketplace on the PAW blockchain. It enables users to request computational tasks, providers to execute them, and ensures integrity through cryptographic verification, reputation systems, and economic incentives.

## Concepts

### Decentralized Compute Marketplace

The Compute module creates a trustless marketplace for off-chain computation:

- **Requesters**: Submit computation jobs with specifications and payment
- **Providers**: Execute jobs and submit verifiable results
- **Verification**: Results are verified using Merkle proofs, Ed25519 signatures, and optional ZK-SNARKs
- **Reputation**: Provider quality tracked via success rates and slashing
- **Escrow**: Automatic payment upon successful verification
- **Disputes**: On-chain governance for resolving conflicts

### Compute Request Lifecycle

```
1. Submit Request → 2. Assign Provider → 3. Execute Off-Chain → 4. Submit Result → 5. Verify → 6. Payment/Slash
```

**Phases:**

1. **Request Submission**: User submits job with specs, container image, command, and max payment
2. **Provider Assignment**: Module matches job to suitable provider based on specs, stake, and reputation
3. **Off-Chain Execution**: Provider runs containerized computation
4. **Result Submission**: Provider submits output hash, URL, and cryptographic proofs
5. **Verification**: Module verifies result integrity using proofs
6. **Settlement**: Payment released to provider or slashing applied for failures

### Verification Methods

**Three-tier verification system:**

#### 1. Merkle Proof Verification

Basic integrity verification using Merkle trees:
- Provider computes Merkle tree of output data
- Submits root hash and proof
- Module verifies proof against root hash
- Fast, lightweight verification

#### 2. Ed25519 Signature Verification

Cryptographic attestation of result:
- Provider signs result hash with private key
- Module verifies signature against registered public key
- Proves provider authenticity and non-repudiation

#### 3. ZK-SNARK Verification (Advanced)

Zero-knowledge proof of computation correctness:
- Provider generates ZK-SNARK proof using Groth16
- Proof attests to computation without revealing private data
- Module verifies proof on-chain
- Enables privacy-preserving computation

**Circuit Definition:**
```
Public Inputs: RequestID, ResultHash, ProviderAddressHash
Private Inputs: ComputationDataHash, ExecutionTimestamp, ExitCode, CpuCyclesUsed, MemoryBytesUsed

Constraint: MiMC(RequestID || ProviderAddressHash || ComputationDataHash || ExecutionTimestamp || ExitCode || CpuCycles || Memory) == ResultHash
```

### Reputation System

Provider quality is tracked and enforced:

- **Reputation Score**: 0-100 scale, starts at minimum required score
- **Success Tracking**: Increments on successful completion
- **Failure Tracking**: Decrements on failures, timeouts, disputes
- **Slashing**: Reputation and stake slashed for misbehavior
- **Job Matching**: Higher reputation providers get priority
- **Minimum Threshold**: Providers below minimum score cannot accept jobs

### Escrow and Payment

Automated trustless payment system:

- **Upfront Escrow**: Max payment escrowed when request submitted
- **Cost Estimation**: Module estimates cost based on specs and pricing
- **Actual Payment**: Based on actual resources used (within max)
- **Release Delay**: Configurable delay for dispute window (default: 3600s)
- **Refunds**: Automatic refund if provider fails or request cancelled

### Dispute Resolution

Decentralized governance for conflicts:

**Dispute Flow:**
1. Requester creates dispute with deposit
2. Evidence submission period (default: 24h)
3. Validator voting period (default: 24h)
4. Resolution: refund requester or penalize provider

**Appeal Process:**
- Slashed providers can appeal with deposit
- Validator voting on appeal
- Final resolution with stake adjustment

### Rate Limiting

Protection against spam and DoS attacks:

- **Per-Address Limits**: Max requests per block per address
- **Global Limits**: Max total requests per block
- **Dynamic Adjustment**: Limits based on network load
- **Exemptions**: High-reputation providers may have higher limits

### IBC Integration

Cross-chain compute orchestration:

- **Job Packets**: Send compute requests to remote chains
- **Result Packets**: Receive results from remote providers
- **Authorized Channels**: Whitelist of trusted IBC channels
- **Timeout Handling**: Automatic refunds on packet timeout

## State

The module stores the following data:

### Requests

- **Key**: `requests/{requestId}`
- **Value**: Request struct
- **Description**: All compute requests and their status

```go
type Request struct {
    Id             uint64
    Requester      string
    Provider       string
    Specs          ComputeSpec
    ContainerImage string
    Command        []string
    EnvVars        map[string]string
    Status         RequestStatus  // PENDING, ASSIGNED, COMPLETED, FAILED, CANCELLED
    MaxPayment     math.Int
    EscrowedAmount math.Int
    ActualPayment  math.Int
    CreatedAt      time.Time
    AssignedAt     *time.Time
    CompletedAt    *time.Time
}
```

### Providers

- **Key**: `providers/{address}`
- **Value**: Provider struct
- **Description**: Registered compute providers

```go
type Provider struct {
    Address                string
    Moniker                string
    Endpoint               string          // API endpoint for job submission
    AvailableSpecs         ComputeSpec     // Hardware capabilities
    Pricing                Pricing         // Pricing structure
    Stake                  math.Int        // Staked amount
    Reputation             uint64          // 0-100 score
    TotalRequestsCompleted uint64
    TotalRequestsFailed    uint64
    Active                 bool
    RegisteredAt           time.Time
    LastActiveAt           time.Time
}
```

### Results

- **Key**: `results/{requestId}`
- **Value**: Result struct
- **Description**: Computation results with verification data

```go
type Result struct {
    RequestId      uint64
    Provider       string
    OutputHash     string          // Hash of output data
    OutputUrl      string          // URL to download full output
    ExitCode       int32
    ExecutionTime  uint64          // Milliseconds
    MerkleRoot     []byte
    MerkleProof    []byte
    Signature      []byte          // Ed25519 signature
    ZkProof        *ZKProof        // Optional ZK-SNARK proof
    VerifiedAt     time.Time
}
```

### Disputes

- **Key**: `disputes/{disputeId}`
- **Value**: Dispute struct
- **Description**: Active and resolved disputes

```go
type Dispute struct {
    Id              uint64
    RequestId       uint64
    Requester       string
    Provider        string
    Reason          string
    DepositAmount   math.Int
    Status          DisputeStatus  // OPEN, VOTING, RESOLVED
    Evidence        []Evidence
    Votes           map[string]bool  // validator -> vote
    Resolution      string
    CreatedAt       time.Time
    ResolvedAt      *time.Time
}
```

### Parameters

- **Key**: `params`
- **Value**: Params struct

```go
type Params struct {
    MinProviderStake           math.Int        // 1000000 upaw (1 PAW)
    VerificationTimeoutSeconds uint64          // 300 seconds
    MaxRequestTimeoutSeconds   uint64          // 3600 seconds
    ReputationSlashPercentage  uint64          // 10%
    StakeSlashPercentage       uint64          // 1%
    MinReputationScore         uint64          // 50
    EscrowReleaseDelaySeconds  uint64          // 3600 seconds
    AuthorizedChannels         []AuthorizedChannel
}

type GovernanceParams struct {
    DisputeDeposit          math.Int              // 1000000 upaw
    EvidencePeriodSeconds   uint64                // 86400 (24h)
    VotingPeriodSeconds     uint64                // 86400 (24h)
    QuorumPercentage        math.LegacyDec        // 0.334 (33.4%)
    ConsensusThreshold      math.LegacyDec        // 0.5 (50%)
    SlashPercentage         math.LegacyDec        // 0.1 (10%)
    AppealDepositPercentage math.LegacyDec        // 0.05 (5%)
    MaxEvidenceSize         uint64                // 10 MB
}
```

## Messages

### MsgRegisterProvider

Register as a compute provider.

```protobuf
message MsgRegisterProvider {
  string provider = 1;
  string moniker = 2;
  string endpoint = 3;
  ComputeSpec available_specs = 4;
  Pricing pricing = 5;
  string stake = 6 [(cosmos_proto.scalar) = "cosmos.Int"];
}
```

**CLI Command:**
```bash
pawd tx compute register-provider \
  --moniker "My Compute Node" \
  --endpoint "https://compute.example.com" \
  --cpu-cores 16 \
  --memory-mb 32768 \
  --gpu-count 2 \
  --storage-gb 1000 \
  --cpu-price 1000 \
  --memory-price 500 \
  --gpu-price 50000 \
  --storage-price 100 \
  --stake 1000000upaw \
  --from provider \
  --chain-id paw-1
```

**Validation:**
- Provider must not already be registered
- Stake must meet minimum requirement
- Specs and pricing must be valid
- Endpoint must be reachable (future)

**Effects:**
- Transfers stake to module escrow
- Creates provider record with initial reputation
- Adds to active provider index
- Emits `compute_provider_registered` event

### MsgSubmitRequest

Submit a compute job request.

```protobuf
message MsgSubmitRequest {
  string requester = 1;
  ComputeSpec specs = 2;
  string container_image = 3;
  repeated string command = 4;
  map<string, string> env_vars = 5;
  string max_payment = 6 [(cosmos_proto.scalar) = "cosmos.Int"];
  string preferred_provider = 7;
}
```

**CLI Command:**
```bash
pawd tx compute submit-request \
  --cpu-cores 4 \
  --memory-mb 8192 \
  --timeout-seconds 600 \
  --container-image "ubuntu:22.04" \
  --command "python3,script.py" \
  --env-vars "API_KEY=secret,ENV=prod" \
  --max-payment 100000upaw \
  --from alice \
  --chain-id paw-1
```

**Validation:**
- Requester must have sufficient balance for max payment
- Specs must be valid and within limits
- Container image must be valid format
- Commands and env vars must pass security checks

**Effects:**
- Escrows max payment from requester
- Matches suitable provider
- Creates request record with ASSIGNED status
- Updates provider last active time
- Emits `compute_request` and `compute_request_accepted` events

### MsgSubmitResult

Provider submits computation result.

```protobuf
message MsgSubmitResult {
  string provider = 1;
  uint64 request_id = 2;
  string output_hash = 3;
  string output_url = 4;
  int32 exit_code = 5;
  uint64 execution_time = 6;
  bytes merkle_root = 7;
  bytes merkle_proof = 8;
  bytes signature = 9;
  ZKProof zk_proof = 10;
}
```

**CLI Command:**
```bash
pawd tx compute submit-result \
  --request-id 42 \
  --output-hash "sha256:abc123..." \
  --output-url "https://storage.example.com/result-42.tar.gz" \
  --exit-code 0 \
  --execution-time 45000 \
  --merkle-root <base64> \
  --merkle-proof <base64> \
  --signature <base64> \
  --from provider \
  --chain-id paw-1
```

**Validation:**
- Request must exist and be assigned to provider
- Output hash and URL must be valid
- Merkle proof must verify against root
- Signature must verify against provider's public key
- ZK proof must verify if present

**Effects:**
- Verifies Merkle proof
- Verifies Ed25519 signature
- Verifies ZK proof if present
- Updates request status to COMPLETED or FAILED
- Calculates actual payment based on resources used
- Schedules payment release after delay
- Updates provider reputation
- Emits `compute_result` and `compute_result_verified` events

### MsgCreateDispute

Create a dispute for a request result.

```protobuf
message MsgCreateDispute {
  string requester = 1;
  uint64 request_id = 2;
  string reason = 3;
  string deposit_amount = 4 [(cosmos_proto.scalar) = "cosmos.Int"];
}
```

**CLI Command:**
```bash
pawd tx compute create-dispute \
  --request-id 42 \
  --reason "Result verification failed, incorrect output" \
  --deposit 1000000upaw \
  --from alice \
  --chain-id paw-1
```

**Validation:**
- Request must exist and be completed
- Requester must be original requester
- Deposit must meet minimum
- Reason must be provided

**Effects:**
- Escrows dispute deposit
- Creates dispute record
- Pauses payment release
- Starts evidence period
- Emits `compute_dispute` event

### MsgVoteOnDispute

Validator votes on a dispute.

```protobuf
message MsgVoteOnDispute {
  string validator = 1;
  uint64 dispute_id = 2;
  bool vote = 3;  // true = favor requester, false = favor provider
}
```

**Access Control:** Only active validators can vote.

### MsgCancelRequest

Cancel a pending request.

```protobuf
message MsgCancelRequest {
  string requester = 1;
  uint64 request_id = 2;
}
```

**Validation:**
- Request must be pending (not yet completed)
- Only original requester can cancel

**Effects:**
- Refunds escrowed payment to requester
- Updates request status to CANCELLED
- Emits `compute_escrow_refunded` event

## Queries

### Request

Get request details by ID.

```bash
pawd query compute request 42
```

### Requests

List all requests with filtering and pagination.

```bash
pawd query compute requests \
  --requester paw1abc...xyz \
  --status completed \
  --page 1 \
  --limit 50
```

### Provider

Get provider details.

```bash
pawd query compute provider paw1provider...xyz
```

### Providers

List all providers.

```bash
pawd query compute providers \
  --active true \
  --min-reputation 80 \
  --page 1 \
  --limit 50
```

### Result

Get result for a request.

```bash
pawd query compute result 42
```

### Dispute

Get dispute details.

```bash
pawd query compute dispute 7
```

### EstimateCost

Estimate cost for a compute request.

```bash
pawd query compute estimate-cost \
  --provider paw1provider...xyz \
  --cpu-cores 4 \
  --memory-mb 8192 \
  --timeout-seconds 600
```

**Response:**
```json
{
  "estimated_cost": "95000",
  "breakdown": {
    "cpu_cost": "40000",
    "memory_cost": "30000",
    "storage_cost": "15000",
    "base_fee": "10000"
  }
}
```

### Params

Get module parameters.

```bash
pawd query compute params
```

## Events

### compute_request

Emitted when a new request is submitted.

**Attributes:**
- `request_id`: Request identifier
- `requester`: Requester address
- `provider`: Assigned provider
- `max_payment`: Maximum payment amount

### compute_result_verified

Emitted when a result is verified successfully.

**Attributes:**
- `request_id`: Request identifier
- `provider`: Provider address
- `output_hash`: Result hash
- `verification_status`: Verification result
- `actual_payment`: Actual payment amount

### compute_provider_slashed

Emitted when a provider is slashed.

**Attributes:**
- `provider`: Provider address
- `request_id`: Related request
- `slash_amount`: Stake amount slashed
- `reputation_delta`: Reputation decrease
- `reason`: Slashing reason

### compute_dispute

Emitted when a dispute is created.

**Attributes:**
- `dispute_id`: Dispute identifier
- `request_id`: Disputed request
- `requester`: Dispute creator
- `provider`: Disputed provider
- `deposit_amount`: Dispute deposit

### compute_zk_proof_verified

Emitted when a ZK proof is verified.

**Attributes:**
- `request_id`: Request identifier
- `provider`: Provider address
- `proof_type`: ZK proof type
- `verification_status`: Success or failure

## Parameters

Module parameters can be updated via governance.

### Governance Update Example

```bash
# Create parameter change proposal
pawd tx gov submit-proposal param-change proposal.json \
  --from validator \
  --chain-id paw-1

# proposal.json
{
  "title": "Reduce Minimum Provider Stake",
  "description": "Lower barrier to entry for compute providers",
  "changes": [
    {
      "subspace": "compute",
      "key": "MinProviderStake",
      "value": "500000"
    }
  ],
  "deposit": "10000000upaw"
}
```

## Security Features

### Cryptographic Verification

**Three-Layer Verification:**

1. **Merkle Proofs**: Integrity of output data
2. **Ed25519 Signatures**: Provider authenticity
3. **ZK-SNARKs**: Computation correctness (optional)

### Rate Limiting

Prevents spam and DoS attacks:
- Per-address request limits (default: 10/block)
- Global request limits (default: 100/block)
- Dynamic adjustment based on load

### Slashing and Reputation

Economic security through:
- Stake slashing for failed jobs (1% default)
- Reputation penalties (10% default)
- Minimum reputation threshold (50 default)
- Jailing for severe violations

### Escrow Protection

Trustless payment system:
- Upfront escrow of max payment
- Release only after verification
- Configurable delay for dispute window
- Automatic refunds on failure

### Access Control

- Provider registration requires stake
- Only assigned provider can submit results
- Only validators can vote on disputes
- Only requester can create disputes

### Nonce Cleanup and Replay Protection

Prevents replay attacks while managing state growth:

- **Replay Protection**: Each result submission requires a unique, monotonically increasing nonce
- **Automatic Cleanup**: Old nonces are automatically cleaned up to prevent unbounded state growth
- **Configurable Retention**: `nonce_retention_blocks` parameter controls how long nonces are kept
  - Default: 17,280 blocks (~24 hours at 5-second block time)
  - Nonces older than retention period are eligible for cleanup
- **Batched Processing**: Cleanup processes 100 blocks worth of nonces per EndBlocker to manage gas consumption
- **Height-Indexed Storage**: Nonces are indexed by block height for efficient cleanup
- **Metrics**: Cleanup operations are tracked via Prometheus metrics
  - `paw_compute_nonce_cleanups_total`: Number of cleanup operations executed
  - `paw_compute_nonces_cleaned_total`: Total number of nonces cleaned up

**How it works:**

1. When a provider submits a result, a nonce is recorded with the current block height
2. Every block, EndBlocker runs cleanup for nonces older than `nonce_retention_blocks`
3. Cleanup processes the oldest 100 blocks of expired nonces per cycle
4. Old nonces are removed from both the main store and height index
5. Events are emitted with cleanup statistics

**Configuration:**

```json
{
  "params": {
    "nonce_retention_blocks": 17280  // ~24 hours at 5s block time
  }
}
```

### Circuit Breaker

Emergency halt mechanism:
- Module-level circuit breaker
- Per-provider circuit breaker
- Automatic tripping on anomalies
- Manual reset by governance

## Integration Examples

### JavaScript/TypeScript

```typescript
import { SigningStargateClient } from "@cosmjs/stargate";

const client = await SigningStargateClient.connectWithSigner(
  "https://rpc.paw.network",
  signer
);

// Submit a compute request
const msg = {
  typeUrl: "/paw.compute.v1.MsgSubmitRequest",
  value: {
    requester: "paw1abc...xyz",
    specs: {
      cpuCores: 4,
      memoryMb: 8192,
      timeoutSeconds: 600
    },
    containerImage: "ubuntu:22.04",
    command: ["python3", "script.py"],
    envVars: { API_KEY: "secret" },
    maxPayment: "100000"
  }
};

const result = await client.signAndBroadcast(address, [msg], "auto");
```

### Python

```python
from cosmospy import Transaction, BroadcastMode

# Submit compute request
submit_msg = {
    "type": "paw/compute/MsgSubmitRequest",
    "value": {
        "requester": "paw1abc...xyz",
        "specs": {
            "cpu_cores": 4,
            "memory_mb": 8192,
            "timeout_seconds": 600
        },
        "container_image": "ubuntu:22.04",
        "command": ["python3", "script.py"],
        "max_payment": "100000"
    }
}

tx = Transaction(...)
result = tx.broadcast(mode=BroadcastMode.SYNC)
```

### Go

```go
import (
    computetypes "github.com/paw-chain/paw/x/compute/types"
    sdk "github.com/cosmos/cosmos-sdk/types"
)

// Submit compute request
msg := &computetypes.MsgSubmitRequest{
    Requester: "paw1abc...xyz",
    Specs: computetypes.ComputeSpec{
        CpuCores:       4,
        MemoryMb:       8192,
        TimeoutSeconds: 600,
    },
    ContainerImage: "ubuntu:22.04",
    Command:        []string{"python3", "script.py"},
    EnvVars:        map[string]string{"API_KEY": "secret"},
    MaxPayment:     sdk.NewInt(100000),
}

// Broadcast via client...
```

## Provider Implementation Guide

### Setting Up a Compute Provider

**1. Register on-chain:**

```bash
pawd tx compute register-provider \
  --moniker "My Compute Node" \
  --endpoint "https://compute.example.com" \
  --cpu-cores 16 \
  --memory-mb 32768 \
  --stake 1000000upaw \
  --from provider
```

**2. Run provider daemon:**

```bash
# Provider daemon polls for assigned jobs
compute-provider-daemon \
  --chain-rpc https://rpc.paw.network \
  --provider-key provider.key \
  --work-dir /var/compute \
  --log-level info
```

**3. Execute jobs:**

Provider daemon should:
- Poll chain for assigned requests
- Download container image
- Execute with resource limits
- Capture output and generate proofs
- Submit result on-chain

### Example Provider Implementation

```python
import hashlib
import requests
from merkle_tree import MerkleTree

def execute_job(request):
    # Download and verify container image
    image = download_container(request.container_image)

    # Execute with resource limits
    result = execute_container(
        image=image,
        command=request.command,
        env_vars=request.env_vars,
        cpu_limit=request.specs.cpu_cores,
        memory_limit=request.specs.memory_mb,
        timeout=request.specs.timeout_seconds
    )

    # Generate Merkle tree of output
    merkle_tree = MerkleTree(result.output)
    merkle_root = merkle_tree.root()
    merkle_proof = merkle_tree.proof(result.output)

    # Sign result hash
    output_hash = hashlib.sha256(result.output).hexdigest()
    signature = sign_with_ed25519(output_hash, provider_privkey)

    # Generate ZK proof (optional)
    zk_proof = generate_zk_proof(request, result) if use_zk else None

    # Submit result on-chain
    submit_result(
        request_id=request.id,
        output_hash=output_hash,
        output_url=upload_to_storage(result.output),
        exit_code=result.exit_code,
        execution_time=result.execution_time_ms,
        merkle_root=merkle_root,
        merkle_proof=merkle_proof,
        signature=signature,
        zk_proof=zk_proof
    )
```

## Testing

### Unit Tests

```bash
# Run all compute module tests
go test ./x/compute/...

# Test with coverage
go test -cover ./x/compute/...

# Test specific functionality
go test ./x/compute/keeper -run TestRequestLifecycle -v
```

### Integration Tests

```bash
# Run integration tests
go test ./x/compute/keeper -run TestIntegration -v
```

### ZK Proof Tests

```bash
# Test ZK proof generation and verification
go test ./x/compute/keeper -run TestZKVerification -v
```

## Monitoring

### Key Metrics

- Total active providers
- Total active requests
- Average completion time
- Success rate per provider
- Slashing events
- Dispute rate

### Prometheus Metrics

```
compute_provider_count: Number of registered providers
compute_active_requests: Current active requests
compute_total_completed: Total completed requests
compute_provider_reputation{provider}: Reputation score per provider
compute_slashing_events: Total slashing events
compute_dispute_count: Active disputes
```

## Future Enhancements

### Planned Features

- Multi-provider redundancy (parallel execution)
- Trusted Execution Environment (TEE) integration
- GPU workload support
- Distributed storage integration (IPFS, Arweave)
- Advanced ZK circuits for specific computation types
- Provider SLAs and automatic penalties
- Dynamic pricing based on demand

### Research Areas

- Fully homomorphic encryption (FHE) for private computation
- Secure multi-party computation (MPC)
- Verifiable delay functions (VDF) for fairness
- On-chain WASM execution for verification

## References

- [Cosmos SDK Documentation](https://docs.cosmos.network/)
- [Groth16 ZK-SNARKs](https://eprint.iacr.org/2016/260.pdf)
- [gnark ZK Framework](https://github.com/consensys/gnark)
- [Docker Containerization](https://docs.docker.com/)
- [Ed25519 Signatures](https://ed25519.cr.yp.to/)

---

**Module Maintainers:** PAW Core Development Team
**Last Updated:** 2025-12-06
