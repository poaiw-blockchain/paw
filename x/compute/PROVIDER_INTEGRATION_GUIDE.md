# Provider Integration Guide: Cryptographic Verification

This guide explains how compute providers integrate with PAW's institutional-grade cryptographic verification system.

## Overview

Providers must submit cryptographic proofs with every computation result. These proofs enable trustless verification of correct execution without requiring validators to re-execute computations.

## Proof Requirements

### Components Required

1. **Ed25519 Signature** (64 bytes)
   - Sign the canonical message hash
   - Use a secure, persistent key pair
   - Keep private key secure

2. **Merkle Tree** (variable size)
   - Build from execution trace
   - Maximum 32 levels
   - Minimum 1 proof node

3. **State Commitment** (32 bytes)
   - Hash of final computation state
   - Binds inputs to outputs

4. **Execution Trace** (32 bytes)
   - Deterministic log hash
   - Proves execution path

5. **Nonce** (8 bytes)
   - Unique per submission
   - Prevents replay attacks
   - Must increment or randomize

6. **Timestamp** (8 bytes)
   - Unix timestamp
   - Proof generation time

## Implementation Steps

### Step 1: Initialize Provider Keys

```go
package provider

import (
    "crypto/ed25519"
    "crypto/rand"
)

type Provider struct {
    privateKey ed25519.PrivateKey
    publicKey  ed25519.PublicKey
    nonce      uint64
}

func NewProvider() (*Provider, error) {
    pub, priv, err := ed25519.GenerateKey(rand.Reader)
    if err != nil {
        return nil, err
    }

    return &Provider{
        privateKey: priv,
        publicKey:  pub,
        nonce:      1, // Start at 1 (0 is invalid)
    }, nil
}

func (p *Provider) GetNextNonce() uint64 {
    nonce := p.nonce
    p.nonce++
    return nonce
}
```

### Step 2: Execute Computation with Logging

```go
type ExecutionLog struct {
    Steps []ExecutionStep
}

type ExecutionStep struct {
    Timestamp    int64
    Operation    string
    Input        []byte
    Output       []byte
    StateHash    []byte
}

func (p *Provider) ExecuteComputation(request ComputeRequest) (*Result, *ExecutionLog, error) {
    log := &ExecutionLog{}

    // Execute each step
    for _, cmd := range request.Commands {
        step := ExecutionStep{
            Timestamp: time.Now().Unix(),
            Operation: cmd.Name,
        }

        // Execute command
        output, err := executeCommand(cmd)
        if err != nil {
            return nil, nil, err
        }

        step.Output = output

        // Compute state hash after this step
        step.StateHash = computeStateHash(getCurrentState())

        log.Steps = append(log.Steps, step)
    }

    result := &Result{
        OutputHash: computeOutputHash(log.Steps[len(log.Steps)-1].Output),
        OutputURL:  uploadOutput(log.Steps[len(log.Steps)-1].Output),
        ExitCode:   0,
        LogsURL:    uploadLogs(log),
    }

    return result, log, nil
}
```

### Step 3: Build Merkle Tree from Execution Log

```go
import "crypto/sha256"

type MerkleTree struct {
    Root   []byte
    Levels [][][]byte
}

func BuildMerkleTree(log *ExecutionLog) (*MerkleTree, error) {
    // Hash all execution steps (leaves)
    leaves := make([][]byte, len(log.Steps))
    for i, step := range log.Steps {
        hasher := sha256.New()
        hasher.Write([]byte(step.Operation))
        hasher.Write(step.Input)
        hasher.Write(step.Output)
        hasher.Write(step.StateHash)
        leaves[i] = hasher.Sum(nil)
    }

    tree := &MerkleTree{
        Levels: make([][][]byte, 0),
    }
    tree.Levels = append(tree.Levels, leaves)

    // Build tree bottom-up
    currentLevel := leaves
    for len(currentLevel) > 1 {
        nextLevel := make([][]byte, 0)

        for i := 0; i < len(currentLevel); i += 2 {
            hasher := sha256.New()
            hasher.Write(currentLevel[i])

            if i+1 < len(currentLevel) {
                hasher.Write(currentLevel[i+1])
            } else {
                // Odd number of nodes - duplicate last
                hasher.Write(currentLevel[i])
            }

            nextLevel = append(nextLevel, hasher.Sum(nil))
        }

        tree.Levels = append(tree.Levels, nextLevel)
        currentLevel = nextLevel
    }

    tree.Root = currentLevel[0]
    return tree, nil
}

func (mt *MerkleTree) GenerateProof(leafIndex int) [][]byte {
    proof := make([][]byte, 0)
    index := leafIndex

    for level := 0; level < len(mt.Levels)-1; level++ {
        levelNodes := mt.Levels[level]

        // Determine sibling index
        var siblingIndex int
        if index%2 == 0 {
            siblingIndex = index + 1
        } else {
            siblingIndex = index - 1
        }

        // Add sibling to proof (if exists)
        if siblingIndex < len(levelNodes) {
            proof = append(proof, levelNodes[siblingIndex])
        }

        // Move to parent index
        index = index / 2
    }

    return proof
}
```

### Step 4: Compute State Commitment

```go
func ComputeStateCommitment(request ComputeRequest, result *Result, log *ExecutionLog) []byte {
    hasher := sha256.New()

    // Include container image
    hasher.Write([]byte(request.ContainerImage))

    // Include all commands
    for _, cmd := range request.Commands {
        hasher.Write([]byte(cmd))
    }

    // Include output hash
    hasher.Write([]byte(result.OutputHash))

    // Include execution trace
    executionTrace := computeExecutionTrace(log)
    hasher.Write(executionTrace)

    return hasher.Sum(nil)
}

func computeExecutionTrace(log *ExecutionLog) []byte {
    hasher := sha256.New()

    for _, step := range log.Steps {
        hasher.Write([]byte(step.Operation))
        hasher.Write(step.Output)
        hasher.Write(step.StateHash)
    }

    return hasher.Sum(nil)
}
```

### Step 5: Create Verification Proof

```go
import (
    "encoding/binary"
    "time"
)

func (p *Provider) CreateVerificationProof(
    requestID uint64,
    result *Result,
    log *ExecutionLog,
    request ComputeRequest,
) ([]byte, error) {
    // Build merkle tree
    tree, err := BuildMerkleTree(log)
    if err != nil {
        return nil, err
    }

    // Generate proof for last execution step
    merkleProof := tree.GenerateProof(len(log.Steps) - 1)

    // Compute state commitment
    stateCommitment := ComputeStateCommitment(request, result, log)

    // Compute execution trace hash
    executionTrace := computeExecutionTrace(log)

    // Get nonce
    nonce := p.GetNextNonce()

    // Get timestamp
    timestamp := time.Now().Unix()

    // Compute message hash for signature
    messageHash := computeMessageHash(
        requestID,
        result.OutputHash,
        tree.Root,
        stateCommitment,
        nonce,
        timestamp,
    )

    // Sign the message
    signature := ed25519.Sign(p.privateKey, messageHash)

    // Serialize proof
    return serializeProof(VerificationProof{
        Signature:       signature,
        PublicKey:       p.publicKey,
        MerkleRoot:      tree.Root,
        MerkleProof:     merkleProof,
        StateCommitment: stateCommitment,
        ExecutionTrace:  executionTrace,
        Nonce:           nonce,
        Timestamp:       timestamp,
    }), nil
}

func computeMessageHash(
    requestID uint64,
    resultHash string,
    merkleRoot []byte,
    stateCommitment []byte,
    nonce uint64,
    timestamp int64,
) []byte {
    hasher := sha256.New()

    // Request ID
    reqIDBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(reqIDBytes, requestID)
    hasher.Write(reqIDBytes)

    // Result hash
    hasher.Write([]byte(resultHash))

    // Merkle root
    hasher.Write(merkleRoot)

    // State commitment
    hasher.Write(stateCommitment)

    // Nonce
    nonceBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(nonceBytes, nonce)
    hasher.Write(nonceBytes)

    // Timestamp
    timestampBytes := make([]byte, 8)
    binary.BigEndian.PutUint64(timestampBytes, uint64(timestamp))
    hasher.Write(timestampBytes)

    return hasher.Sum(nil)
}
```

### Step 6: Serialize Proof

```go
func serializeProof(proof VerificationProof) []byte {
    buf := make([]byte, 0)

    // Signature (64 bytes)
    buf = append(buf, proof.Signature...)

    // Public key (32 bytes)
    buf = append(buf, proof.PublicKey...)

    // Merkle root (32 bytes)
    buf = append(buf, proof.MerkleRoot...)

    // Merkle proof count (1 byte)
    buf = append(buf, byte(len(proof.MerkleProof)))

    // Merkle proof nodes (32 bytes each)
    for _, node := range proof.MerkleProof {
        buf = append(buf, node...)
    }

    // State commitment (32 bytes)
    buf = append(buf, proof.StateCommitment...)

    // Execution trace (32 bytes)
    buf = append(buf, proof.ExecutionTrace...)

    // Nonce (8 bytes)
    nonceBz := make([]byte, 8)
    binary.BigEndian.PutUint64(nonceBz, proof.Nonce)
    buf = append(buf, nonceBz...)

    // Timestamp (8 bytes)
    timestampBz := make([]byte, 8)
    binary.BigEndian.PutUint64(timestampBz, uint64(proof.Timestamp))
    buf = append(buf, timestampBz...)

    return buf
}
```

### Step 7: Submit Result with Proof

```go
func (p *Provider) SubmitComputeResult(
    client *CosmosClient,
    requestID uint64,
    result *Result,
    log *ExecutionLog,
    request ComputeRequest,
) error {
    // Create verification proof
    proofBytes, err := p.CreateVerificationProof(requestID, result, log, request)
    if err != nil {
        return fmt.Errorf("failed to create proof: %w", err)
    }

    // Submit to blockchain
    msg := &types.MsgSubmitResult{
        Provider:          p.address,
        RequestId:         requestID,
        OutputHash:        result.OutputHash,
        OutputUrl:         result.OutputURL,
        ExitCode:          result.ExitCode,
        LogsUrl:           result.LogsURL,
        VerificationProof: proofBytes,
    }

    resp, err := client.BroadcastTx(msg)
    if err != nil {
        return fmt.Errorf("failed to submit result: %w", err)
    }

    // Check if verified
    if resp.Events["verification_completed"] != nil {
        score := resp.Events["verification_completed"]["total_score"]
        verified := resp.Events["verification_completed"]["verified"]

        log.Printf("Result verified: %s (score: %s/100)", verified, score)
    }

    return nil
}
```

## Complete Example

```go
package main

import (
    "fmt"
    "log"
)

func main() {
    // Initialize provider
    provider, err := NewProvider()
    if err != nil {
        log.Fatal(err)
    }

    // Get compute request from blockchain
    request, err := getComputeRequest(100)
    if err != nil {
        log.Fatal(err)
    }

    // Execute computation
    result, executionLog, err := provider.ExecuteComputation(request)
    if err != nil {
        log.Fatal(err)
    }

    // Submit result with cryptographic proof
    err = provider.SubmitComputeResult(
        cosmosClient,
        request.ID,
        result,
        executionLog,
        request,
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Result submitted successfully with cryptographic proof")
}
```

## Security Best Practices

### 1. Key Management

- Store private keys in secure hardware (HSM, TPM)
- Use key rotation policies
- Never expose private keys in logs or errors
- Implement key backup and recovery procedures

### 2. Nonce Management

- Use atomic increments for nonce
- Persist nonce state across restarts
- Monitor for nonce gaps (indicates potential issues)
- Consider using timestamp + counter hybrid

### 3. Execution Logging

- Log all operations deterministically
- Include timestamps with microsecond precision
- Hash intermediate states
- Keep logs for audit trail

### 4. Proof Validation (Self-Check)

Before submitting, validate your own proof:

```go
func (p *Provider) ValidateProof(proof []byte) error {
    // Parse proof
    parsed, err := parseProof(proof)
    if err != nil {
        return err
    }

    // Validate structure
    if len(parsed.Signature) != 64 {
        return fmt.Errorf("invalid signature length")
    }

    if len(parsed.PublicKey) != 32 {
        return fmt.Errorf("invalid public key length")
    }

    if parsed.Nonce == 0 {
        return fmt.Errorf("nonce cannot be zero")
    }

    // Validate merkle proof
    if len(parsed.MerkleProof) == 0 {
        return fmt.Errorf("merkle proof required")
    }

    return nil
}
```

### 5. Error Handling

```go
// Check for replay attack detection
if resp.Events["replay_attack_detected"] != nil {
    // Nonce was reused - critical error
    // Investigate potential compromise
    log.Error("SECURITY ALERT: Replay attack detected")
    notifySecurityTeam()
}

// Check verification score
score, _ := strconv.Atoi(resp.Events["verification_completed"]["total_score"])
if score < 80 {
    log.Warn("Low verification score: %d/100", score)
    // Investigate what component failed
}
```

## Performance Optimization

### 1. Pre-compute Merkle Trees

```go
// Build merkle tree incrementally during execution
type IncrementalMerkleTree struct {
    leaves [][]byte
}

func (t *IncrementalMerkleTree) AddLeaf(data []byte) {
    hash := sha256.Sum256(data)
    t.leaves = append(t.leaves, hash[:])
}

func (t *IncrementalMerkleTree) Finalize() *MerkleTree {
    return BuildMerkleTreeFromLeaves(t.leaves)
}
```

### 2. Batch Proof Generation

For multiple results:

```go
func (p *Provider) SubmitBatchResults(requests []ComputeRequest) error {
    proofs := make([][]byte, len(requests))

    // Generate all proofs in parallel
    var wg sync.WaitGroup
    for i, req := range requests {
        wg.Add(1)
        go func(idx int, r ComputeRequest) {
            defer wg.Done()
            result, log, _ := p.ExecuteComputation(r)
            proofs[idx], _ = p.CreateVerificationProof(r.ID, result, log, r)
        }(i, req)
    }
    wg.Wait()

    // Submit all results
    return p.submitBatch(requests, proofs)
}
```

## Monitoring and Debugging

### Metrics to Track

```go
type ProviderMetrics struct {
    TotalSubmissions       int64
    SuccessfulVerifications int64
    FailedVerifications    int64
    AverageScore           float64
    ReplayAttemptsDetected int64
}

func (p *Provider) RecordMetrics(resp TxResponse) {
    p.metrics.TotalSubmissions++

    if resp.Events["verification_completed"]["verified"] == "true" {
        p.metrics.SuccessfulVerifications++
    } else {
        p.metrics.FailedVerifications++
    }

    score, _ := strconv.ParseFloat(
        resp.Events["verification_completed"]["total_score"],
        64,
    )
    p.metrics.AverageScore =
        (p.metrics.AverageScore + score) / 2
}
```

### Debug Logging

```go
func (p *Provider) DebugProof(proof []byte) {
    parsed, _ := parseProof(proof)

    log.Debug("Proof Details:")
    log.Debug("  Signature: %x", parsed.Signature[:8])
    log.Debug("  Public Key: %x", parsed.PublicKey[:8])
    log.Debug("  Merkle Root: %x", parsed.MerkleRoot)
    log.Debug("  Merkle Proof Nodes: %d", len(parsed.MerkleProof))
    log.Debug("  Nonce: %d", parsed.Nonce)
    log.Debug("  Timestamp: %d", parsed.Timestamp)
}
```

## Troubleshooting

### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Signature verification fails | Wrong message hash | Verify hash computation matches spec |
| Merkle proof invalid | Tree construction error | Check tree building algorithm |
| Replay attack detected | Nonce reused | Ensure nonce increments properly |
| Low verification score | Missing components | Include all proof elements |
| State commitment mismatch | Hash order wrong | Follow canonical ordering |

## Support and Resources

- Documentation: https://docs.paw.chain/compute/verification
- Example Provider: https://github.com/paw-chain/provider-reference
- Security Audits: https://audits.paw.chain/verification
- Community: https://discord.gg/paw-chain
