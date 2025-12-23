// Package keeper implements the Compute module keeper for decentralized computation.
//
// The Compute module enables users to submit verifiable computation requests
// to a decentralized network of compute providers, with results verified using
// zero-knowledge proofs (ZK-SNARKs). Supports escrow-based payments, provider
// reputation management, and dispute resolution.
//
// # Core Functionality
//
// Compute Requests: Submit containerized computation jobs with resource specs
// (CPU, memory, storage), payment escrow, and optional provider preferences.
//
// Provider Management: Register compute providers with staked collateral,
// reputation scores, geographic diversity, and resource availability tracking.
//
// ZK Proof Verification: Verify computation correctness using ZK-SNARKs via
// CircuitManager. Three proof types: compute results, escrow releases, and
// result correctness. Lazy circuit initialization minimizes startup overhead.
//
// Escrow and Payments: Automatic escrow locking on request submission, release
// on verified result delivery, and refunds for failures or timeouts.
//
// Reputation System: Dynamic provider scoring based on successful completions,
// response times, and slashing events. Influences provider selection probability.
//
// Dispute Resolution: Challenge-response mechanism for incorrect results with
// validator voting, evidence submission, and slashing of dishonest providers.
//
// # Key Types
//
// Keeper: Main module keeper managing state, escrows, providers, and ZK circuits.
//
// ComputeRequest: User request specifying container image, resource requirements,
// payment, and execution parameters.
//
// Provider: Registered compute provider with collateral, reputation, and capacity.
//
// CircuitManager: ZK circuit compilation and proof verification engine.
//
// # Usage Patterns
//
// Submitting a request:
//
//	requestID, err := keeper.SubmitRequest(ctx, requester, specs, containerImage, command, envVars, maxPayment, preferredProvider)
//
// Submitting a result:
//
//	err := keeper.SubmitResult(ctx, requestID, provider, resultData, proofData)
//
// Verifying a proof:
//
//	valid, err := keeper.VerifyComputeProofWithCircuitManager(ctx, proofData, requestID, resultCommitment, providerCommitment, resourceCommitment)
//
// # IBC Port
//
// Binds to the "compute" IBC port for cross-chain compute request routing.
// Channels must be authorized via governance before packet acceptance.
//
// # Metrics
//
// Exposes Prometheus metrics for request counts, completion rates, proof
// verification times, and provider performance via ComputeMetrics.
package keeper
