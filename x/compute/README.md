# Compute Module

## Purpose

The Compute module provides a decentralized marketplace for off-chain computation with zero-knowledge verification. It enables users to request compute jobs, providers to execute them, and ensures integrity through cryptographic proofs, reputation systems, and economic incentives.

## Key Features

- **Provider Registration**: Compute providers stake tokens and register available resources (CPU, memory, GPU, storage)
- **Compute Requests**: Users submit containerized workloads with resource specifications
- **Escrow System**: Automatic payment handling with dispute protection
- **ZK Verification**: Optional zero-knowledge proofs for privacy-preserving computation
- **Reputation System**: Multi-dimensional provider scoring (reliability, speed, accuracy, availability)
- **Dispute Resolution**: On-chain governance for resolving conflicts with validator voting
- **Rate Limiting**: Per-account quotas and request cooldowns to prevent spam
- **IBC Integration**: Cross-chain compute requests via authorized IBC channels

## Key Types

### Provider
- `address`: Provider's blockchain address
- `moniker`: Human-readable name
- `endpoint`: API endpoint for job submission
- `available_specs`: Hardware capabilities (CPU, memory, GPU, storage)
- `pricing`: Cost per resource unit (CPU/hour, memory/hour, GPU/hour, storage/hour)
- `stake`: Staked tokens for security
- `reputation`: Score from 0-100 based on performance history

### Request
- `id`: Unique request identifier
- `requester`: User submitting the request
- `provider`: Assigned compute provider
- `specs`: Required resources (ComputeSpec)
- `container_image`: Docker image to execute
- `command`: Command and arguments
- `env_vars`: Environment variables
- `status`: PENDING, ASSIGNED, PROCESSING, COMPLETED, FAILED, CANCELLED
- `max_payment`: Maximum amount user will pay

### Result
- `request_id`: Associated request
- `output_hash`: Hash of computation output
- `output_url`: URL to download results
- `exit_code`: Process exit code
- `logs_url`: Execution logs location
- `verification_proof`: Cryptographic proof (Merkle, Ed25519, or ZK-SNARK)

## Key Messages

- **MsgRegisterProvider**: Register as compute provider with stake and resource specs
- **MsgUpdateProvider**: Update provider information (endpoint, specs, pricing)
- **MsgDeactivateProvider**: Deactivate provider registration
- **MsgSubmitRequest**: Submit compute job with specs and max payment
- **MsgCancelRequest**: Cancel pending request and refund escrow
- **MsgSubmitResult**: Provider submits result with verification proof
- **MsgCreateDispute**: Create dispute with deposit for incorrect results
- **MsgVoteOnDispute**: Validator votes on dispute (provider fault, requester fault, no fault)
- **MsgSubmitEvidence**: Submit evidence for active dispute
- **MsgAppealSlashing**: Appeal slashing event with deposit
- **MsgVoteOnAppeal**: Validator votes on slashing appeal

## Configuration Parameters

### Core Parameters
- `min_provider_stake`: Minimum stake to register (default: 1000000 upaw)
- `verification_timeout_seconds`: Timeout for result verification (default: 300s)
- `max_request_timeout_seconds`: Maximum execution time (default: 3600s)
- `escrow_release_delay_seconds`: Delay before releasing payment (default: 3600s)
- `stake_slash_percentage`: Percentage slashed for misbehavior (default: 1%)
- `reputation_slash_percentage`: Reputation penalty for failures (default: 10%)

### Rate Limiting
- `max_requests_per_address_per_day`: Daily request limit per address (default: 100)
- `request_cooldown_blocks`: Minimum blocks between requests (default: 10)

### Advanced
- `nonce_retention_blocks`: Replay protection window (default: 17280 blocks ≈ 24 hours)
- `circuit_param_hashes`: Governance-approved ZK circuit parameters
- `provider_cache_size`: Top providers cached by reputation (default: 10)
- `provider_cache_refresh_interval`: Cache refresh frequency (default: 100 blocks)

### Governance Parameters
- `dispute_deposit`: Required deposit to file dispute (default: 1000000 upaw)
- `evidence_period_seconds`: Evidence submission window (default: 86400s)
- `voting_period_seconds`: Validator voting period (default: 86400s)
- `quorum_percentage`: Minimum voting power required (default: 33.4%)
- `consensus_threshold`: Percentage for decision (default: 50%)

---

**Module Path:** `github.com/paw-chain/paw/x/compute`
**Maintainers:** PAW Core Development Team
**Last Updated:** 2025-12-25
