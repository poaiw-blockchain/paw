# Groth16 Proof System Integration Guide

## What is Groth16?

Groth16 is a zk-SNARK proving system offering constant-size proofs (~192 bytes) and fast verification. Widely used in production systems (Zcash, Filecoin, Tornado Cash).

### Key Properties
- **Proof Size**: 192 bytes (2 G1 points + 1 G2 point)
- **Verification**: Single pairing check, ~10ms
- **Security**: Relies on trusted setup
- **Trade-off**: Circuit-specific keys

## gnark Integration

PAW blockchain uses [gnark](https://github.com/Consensys/gnark) for Groth16 implementation.

### Installation
```bash
go get github.com/consensys/gnark@latest
go get github.com/consensys/gnark-crypto@latest
```

### Basic Workflow

#### 1. Define Circuit
```go
package circuits

import (
    "github.com/consensys/gnark/frontend"
    "github.com/consensys/gnark/std/hash/mimc"
)

type MyCircuit struct {
    // Public inputs (visible to verifier)
    PublicInput frontend.Variable `gnark:",public"`

    // Private inputs (witness - kept secret)
    SecretValue frontend.Variable `gnark:",secret"`
}

func (circuit *MyCircuit) Define(api frontend.API) error {
    // Define constraints
    mimc, _ := mimc.NewMiMC(api)
    mimc.Write(circuit.SecretValue)
    hash := mimc.Sum()

    api.AssertIsEqual(circuit.PublicInput, hash)
    return nil
}
```

#### 2. Compile Circuit
```go
import (
    "github.com/consensys/gnark/backend/groth16"
    "github.com/consensys/gnark/frontend"
    "github.com/consensys/gnark/frontend/cs/r1cs"
    "github.com/consensys/gnark-crypto/ecc"
)

func CompileCircuit() (groth16.ProvingKey, groth16.VerifyingKey, error) {
    var circuit MyCircuit

    // Compile to R1CS
    ccs, err := frontend.Compile(
        ecc.BN254.ScalarField(),
        r1cs.NewBuilder,
        &circuit,
    )
    if err != nil {
        return nil, nil, err
    }

    // Setup (trusted setup)
    pk, vk, err := groth16.Setup(ccs)
    if err != nil {
        return nil, nil, err
    }

    return pk, vk, nil
}
```

#### 3. Generate Proof
```go
func GenerateProof(pk groth16.ProvingKey,
    publicInput, secretValue *big.Int) (groth16.Proof, error) {

    // Create witness
    assignment := MyCircuit{
        PublicInput: publicInput,
        SecretValue: secretValue,
    }

    witness, err := frontend.NewWitness(&assignment, ecc.BN254.ScalarField())
    if err != nil {
        return nil, err
    }

    // Generate proof
    proof, err := groth16.Prove(ccs, pk, witness)
    if err != nil {
        return nil, err
    }

    return proof, nil
}
```

#### 4. Verify Proof
```go
func VerifyProof(vk groth16.VerifyingKey, proof groth16.Proof,
    publicInput *big.Int) (bool, error) {

    // Create public witness
    publicAssignment := MyCircuit{
        PublicInput: publicInput,
    }

    publicWitness, err := frontend.NewWitness(
        &publicAssignment,
        ecc.BN254.ScalarField(),
        frontend.PublicOnly(),
    )
    if err != nil {
        return false, err
    }

    // Verify
    err = groth16.Verify(proof, vk, publicWitness)
    return err == nil, err
}
```

## PAW Implementation

### Architecture

```
x/compute/
├── circuits/              # Circuit definitions
│   ├── compute_circuit.go
│   ├── escrow_circuit.go
│   └── result_circuit.go
├── keeper/
│   └── zk_verification.go # Keeper integration
├── setup/                 # Trusted setup
│   ├── mpc_ceremony.go
│   └── keygen.go
└── types/
    └── zk_types.go        # Type definitions
```

### Keeper Integration

```go
// In keeper/zk_verification.go
type ZKVerifier struct {
    keeper          *Keeper
    provingKeys     map[string]groth16.ProvingKey
    verifyingKeys   map[string]groth16.VerifyingKey
    circuits        map[string]frontend.Circuit
}

func (zv *ZKVerifier) GenerateProof(
    ctx sdk.Context,
    circuitID string,
    witness interface{},
) (*types.ZKProof, error) {
    pk := zv.provingKeys[circuitID]
    circuit := zv.circuits[circuitID]

    // Create witness
    w, err := frontend.NewWitness(witness, ecc.BN254.ScalarField())
    if err != nil {
        return nil, err
    }

    // Generate proof
    proof, err := groth16.Prove(circuit, pk, w)
    if err != nil {
        return nil, err
    }

    // Serialize
    proofBytes := proof.MarshalBinary()

    return &types.ZKProof{
        Proof:       proofBytes,
        CircuitId:   circuitID,
        GeneratedAt: ctx.BlockTime(),
    }, nil
}
```

## Proof Serialization

### Serialization Format
```go
// Groth16 proof structure
type Groth16Proof struct {
    Ar bn254.G1Affine  // 64 bytes
    Krs bn254.G1Affine // 64 bytes
    Bs bn254.G2Affine  // 128 bytes (compressed)
}
// Total: 256 bytes (uncompressed) or 192 bytes (compressed)
```

### Serialization Code
```go
func SerializeProof(proof groth16.Proof) ([]byte, error) {
    buf := new(bytes.Buffer)
    _, err := proof.WriteTo(buf)
    if err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}

func DeserializeProof(data []byte) (groth16.Proof, error) {
    proof := groth16.NewProof(ecc.BN254)
    _, err := proof.ReadFrom(bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    return proof, nil
}
```

## On-Chain Verification

### Gas Optimization
```go
// Store verifying key on-chain (one-time cost)
func (k Keeper) SetVerifyingKey(
    ctx sdk.Context,
    circuitID string,
    vk groth16.VerifyingKey,
) {
    store := ctx.KVStore(k.storeKey)
    vkBytes := SerializeVK(vk)
    store.Set(types.VKKey(circuitID), vkBytes)
}

// Verify proof (called per transaction)
func (k Keeper) VerifyProof(
    ctx sdk.Context,
    proof types.ZKProof,
    publicInputs []byte,
) (bool, error) {
    // Load VK from store
    vk := k.GetVerifyingKey(ctx, proof.CircuitId)

    // Deserialize proof
    grProof, err := DeserializeProof(proof.Proof)
    if err != nil {
        return false, err
    }

    // Parse public inputs
    publicWitness, err := ParsePublicInputs(publicInputs)
    if err != nil {
        return false, err
    }

    // Verify (pairing check)
    err = groth16.Verify(grProof, vk, publicWitness)

    // Emit metrics
    k.RecordVerification(ctx, proof.CircuitId, err == nil)

    return err == nil, err
}
```

## Key Management

### Proving Key Storage
```go
// Encrypt and store proving key
func StoreProvingKey(pk groth16.ProvingKey, password string) error {
    // Serialize
    pkBytes, err := SerializePK(pk)
    if err != nil {
        return err
    }

    // Encrypt with AES-256-GCM
    encrypted, err := EncryptAES256GCM(pkBytes, password)
    if err != nil {
        return err
    }

    // Store to disk
    return ioutil.WriteFile("pk.encrypted", encrypted, 0600)
}
```

### Verifying Key Storage
```go
// Store VK on-chain in genesis or upgrade
func (k Keeper) InitVerifyingKeys(ctx sdk.Context) {
    // Compute circuit
    vkCompute := LoadVK("compute_circuit.vk")
    k.SetVerifyingKey(ctx, "compute-v1", vkCompute)

    // Escrow circuit
    vkEscrow := LoadVK("escrow_circuit.vk")
    k.SetVerifyingKey(ctx, "escrow-v1", vkEscrow)

    // Result circuit
    vkResult := LoadVK("result_circuit.vk")
    k.SetVerifyingKey(ctx, "result-v1", vkResult)
}
```

## Performance Tuning

### Parallel Proof Generation
```go
func GenerateProofsConcurrent(
    requests []ProofRequest,
    workers int,
) []ProofResult {
    jobs := make(chan ProofRequest, len(requests))
    results := make(chan ProofResult, len(requests))

    // Start workers
    for w := 0; w < workers; w++ {
        go func() {
            for req := range jobs {
                proof, err := GenerateProof(req.PK, req.Witness)
                results <- ProofResult{
                    Proof: proof,
                    Error: err,
                }
            }
        }()
    }

    // Send jobs
    for _, req := range requests {
        jobs <- req
    }
    close(jobs)

    // Collect results
    var output []ProofResult
    for i := 0; i < len(requests); i++ {
        output = append(output, <-results)
    }

    return output
}
```

### Batch Verification
```go
// Verify multiple proofs more efficiently
func BatchVerify(
    vk groth16.VerifyingKey,
    proofs []groth16.Proof,
    publicInputs []frontend.Witness,
) (bool, error) {
    // gnark doesn't have native batch verify for Groth16
    // Implement using single verifications (still faster than separate calls)
    for i, proof := range proofs {
        err := groth16.Verify(proof, vk, publicInputs[i])
        if err != nil {
            return false, fmt.Errorf("proof %d failed: %w", i, err)
        }
    }
    return true, nil
}
```

## Error Handling

### Common Errors
```go
var (
    ErrInvalidProof       = errors.New("proof verification failed")
    ErrInvalidWitness     = errors.New("invalid witness data")
    ErrCircuitNotFound    = errors.New("circuit not found")
    ErrKeyNotFound        = errors.New("key not found")
    ErrConstraintMismatch = errors.New("constraint count mismatch")
)

func (k Keeper) SafeVerify(
    ctx sdk.Context,
    proof types.ZKProof,
) (valid bool, err error) {
    defer func() {
        if r := recover(); r != nil {
            valid = false
            err = fmt.Errorf("verification panic: %v", r)
        }
    }()

    return k.VerifyProof(ctx, proof)
}
```

## Metrics & Monitoring

### Track Verification Performance
```go
type VerificationMetrics struct {
    TotalVerifications    uint64
    SuccessfulVerifications uint64
    FailedVerifications   uint64
    AverageTimeMs         uint64
    TotalGasUsed          uint64
}

func (k Keeper) RecordVerification(
    ctx sdk.Context,
    circuitID string,
    success bool,
    duration time.Duration,
) {
    metrics := k.GetMetrics(ctx, circuitID)

    metrics.TotalVerifications++
    if success {
        metrics.SuccessfulVerifications++
    } else {
        metrics.FailedVerifications++
    }

    // Update average
    metrics.AverageTimeMs = (metrics.AverageTimeMs*
        (metrics.TotalVerifications-1) +
        uint64(duration.Milliseconds())) /
        metrics.TotalVerifications

    k.SetMetrics(ctx, circuitID, metrics)
}
```

## Testing

### Unit Tests
```go
func TestGroth16ProofGeneration(t *testing.T) {
    // Setup
    pk, vk, err := CompileCircuit()
    require.NoError(t, err)

    // Generate proof
    secretValue := big.NewInt(42)
    publicInput := HashSecret(secretValue)

    proof, err := GenerateProof(pk, publicInput, secretValue)
    require.NoError(t, err)

    // Verify
    valid, err := VerifyProof(vk, proof, publicInput)
    require.NoError(t, err)
    require.True(t, valid)
}
```

## Security Considerations

1. **Trusted Setup**: Must use MPC ceremony
2. **Key Protection**: Encrypt proving keys
3. **Input Validation**: Verify all public inputs
4. **Gas Limits**: Set maximum verification gas
5. **Replay Protection**: Include unique nonces

## References

- [Groth16 Paper](https://eprint.iacr.org/2016/260.pdf)
- [gnark Documentation](https://docs.gnark.consensys.net/)
- [BN254 Specification](https://neuromancer.sk/std/bn/bn254)
