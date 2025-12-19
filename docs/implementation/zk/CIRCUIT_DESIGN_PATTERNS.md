# ZK Circuit Design Patterns for PAW Blockchain

## Overview

This guide covers proven design patterns for building efficient, secure zero-knowledge circuits using Groth16 and gnark.

## Pattern 1: Commitment-Based Privacy

### Concept
Hide private data behind cryptographic commitments in public inputs.

### Implementation
```go
// Public: commitment = MiMC(secret_data || salt)
Commitment frontend.Variable `gnark:",public"`

// Private: actual data
SecretData frontend.Variable `gnark:",secret"`
Salt       frontend.Variable `gnark:",secret"`

// Constraint: verify commitment
func (circuit *MyCircuit) Define(api frontend.API) error {
    mimc, _ := mimc.NewMiMC(api)
    mimc.Write(circuit.SecretData)
    mimc.Write(circuit.Salt)
    computed := mimc.Sum()
    api.AssertIsEqual(circuit.Commitment, computed)
}
```

### When to Use
- Hiding user inputs/outputs
- Provider identity protection
- Result privacy

### Constraint Cost
~8,000 per MiMC hash

## Pattern 2: Range Constraints

### Concept
Prove values are within valid ranges without revealing exact value.

### Implementation
```go
// Prove: 0 <= value < 2^32
func constrainRange32(api frontend.API, value frontend.Variable) {
    bits := api.ToBinary(value, 32)
    // ToBinary enforces: sum(bits[i] * 2^i) == value
    // Each bit must be 0 or 1
}

// Prove: min <= value <= max
func constrainBounded(api frontend.API, value, min, max frontend.Variable) {
    // value >= min
    diff1 := api.Sub(value, min)
    constrainRange32(api, diff1)

    // value <= max
    diff2 := api.Sub(max, value)
    constrainRange32(api, diff2)
}
```

### When to Use
- Resource limits (CPU, memory)
- Monetary amounts
- Array indices
- Exit codes, status values

### Constraint Cost
~1 constraint per bit

## Pattern 3: Boolean Flags

### Concept
Encode multiple conditions as boolean variables.

### Implementation
```go
type Flags struct {
    ExecutionSuccess frontend.Variable `gnark:",secret"`
    VerificationPass frontend.Variable `gnark:",secret"`
    DeadlineMet      frontend.Variable `gnark:",secret"`
}

func (circuit *MyCircuit) Define(api frontend.API) error {
    // Constrain each flag to be boolean (0 or 1)
    api.AssertIsBoolean(circuit.Flags.ExecutionSuccess)
    api.AssertIsBoolean(circuit.Flags.VerificationPass)
    api.AssertIsBoolean(circuit.Flags.DeadlineMet)

    // ALL conditions must be true
    allTrue := api.Mul(circuit.Flags.ExecutionSuccess,
                       circuit.Flags.VerificationPass)
    allTrue = api.Mul(allTrue, circuit.Flags.DeadlineMet)
    api.AssertIsEqual(allTrue, 1)
}
```

### When to Use
- Multi-condition verification
- State machine transitions
- Access control checks

### Constraint Cost
~2 constraints per flag

## Pattern 4: Merkle Tree Membership

### Concept
Prove data is member of a merkle tree without revealing full tree.

### Implementation
```go
type MerkleProof struct {
    Root    frontend.Variable   `gnark:",public"`
    Leaf    frontend.Variable   `gnark:",secret"`
    Path    [4]frontend.Variable `gnark:",secret"` // siblings
    Indices [4]frontend.Variable `gnark:",secret"` // left/right
}

func verifyMerkle(api frontend.API, proof MerkleProof) {
    mimc, _ := mimc.NewMiMC(api)
    current := proof.Leaf

    for i := 0; i < 4; i++ {
        // Select hash order based on index (0=left, 1=right)
        isLeft := api.Sub(1, proof.Indices[i])

        mimc.Reset()
        left := api.Select(isLeft, current, proof.Path[i])
        right := api.Select(isLeft, proof.Path[i], current)
        mimc.Write(left, right)
        current = mimc.Sum()
    }

    api.AssertIsEqual(current, proof.Root)
}
```

### When to Use
- Proving result/input membership
- State verification
- Batch inclusion proofs

### Constraint Cost
~8,000 × tree_depth

## Pattern 5: Conditional Logic

### Concept
Implement if/else logic using Select operations.

### Implementation
```go
// if condition { result = trueValue } else { result = falseValue }
result := api.Select(condition, trueValue, falseValue)

// Complex conditions using AND/OR
// AND: condition1 AND condition2
andResult := api.Mul(condition1, condition2)

// OR: condition1 OR condition2
orResult := api.Add(condition1, condition2)
orResult = api.Sub(orResult, api.Mul(condition1, condition2))

// NOT: !condition
notResult := api.Sub(1, condition)
```

### When to Use
- State machine logic
- Conditional computations
- Error handling

### Constraint Cost
~2 constraints per Select

## Pattern 6: Array Processing

### Concept
Process arrays with dynamic lengths efficiently.

### Implementation
```go
type ArrayCircuit struct {
    Elements [64]frontend.Variable `gnark:",secret"`
    Count    frontend.Variable     `gnark:",secret"`
    Sum      frontend.Variable     `gnark:",public"`
}

func (c *ArrayCircuit) Define(api frontend.API) error {
    sum := frontend.Variable(0)

    for i := 0; i < 64; i++ {
        // Only include element if i < Count
        isValid := api.IsZero(api.Sub(c.Count, i+1))
        isValid = api.Sub(1, isValid) // invert
        contribution := api.Mul(c.Elements[i], isValid)
        sum = api.Add(sum, contribution)
    }

    api.AssertIsEqual(sum, c.Sum)
    return nil
}
```

### When to Use
- Variable-length data processing
- Batch operations
- Resource aggregation

### Constraint Cost
~5 constraints per element

## Pattern 7: Non-Repudiation with Signatures

### Concept
Bind provider identity to computation using signature verification.

### Implementation
```go
import "github.com/consensys/gnark/std/signature/eddsa"

type SignedCircuit struct {
    Message   frontend.Variable   `gnark:",public"`
    PublicKey eddsa.PublicKey     `gnark:",public"`
    Signature eddsa.Signature     `gnark:",secret"`
}

func (c *SignedCircuit) Define(api frontend.API) error {
    curve, _ := twistededwards.NewEdCurve(api, twistededwards.BN254)
    verifier := eddsa.NewEdDSA(curve)

    // Verify signature
    err := verifier.Verify(c.Signature, c.Message, c.PublicKey)
    if err != nil {
        return err
    }
    return nil
}
```

### When to Use
- Provider authentication
- Result authorization
- Audit trails

### Constraint Cost
~40,000 constraints

## Pattern 8: Resource Aggregation

### Concept
Sum multiple resource metrics into single commitment.

### Implementation
```go
func computeResourceCommitment(api frontend.API,
    cpu, mem, disk, net frontend.Variable) frontend.Variable {

    mimc, _ := mimc.NewMiMC(api)
    mimc.Write(cpu)
    mimc.Write(mem)
    mimc.Write(disk)
    mimc.Write(net)
    return mimc.Sum()
}

// In circuit:
computed := computeResourceCommitment(api,
    circuit.CPUUsed, circuit.MemUsed,
    circuit.DiskUsed, circuit.NetUsed)
api.AssertIsEqual(circuit.ResourceCommitment, computed)
```

### When to Use
- Multi-metric commitments
- Cost calculation
- Quota enforcement

### Constraint Cost
~8,000 per hash

## Pattern 9: Timing Constraints

### Concept
Prove execution completed within time bounds.

### Implementation
```go
func (c *TimingCircuit) Define(api frontend.API) error {
    // Execution time = EndTime - StartTime
    execTime := api.Sub(c.EndTime, c.StartTime)

    // Must complete before deadline
    beforeDeadline := api.Sub(c.Deadline, c.EndTime)
    constrainRange32(api, beforeDeadline) // >= 0

    // Must take reasonable time (not instant)
    constrainBounded(api, execTime, minTime, maxTime)

    return nil
}
```

### When to Use
- Deadline enforcement
- DOS prevention
- Execution verification

### Constraint Cost
~64 constraints

## Pattern 10: Batch Verification

### Concept
Verify multiple proofs efficiently using aggregation.

### Implementation
```go
type BatchCircuit struct {
    Commitments [8]frontend.Variable `gnark:",public"`
    Proofs      [8]ProofData         `gnark:",secret"`
}

func (c *BatchCircuit) Define(api frontend.API) error {
    for i := 0; i < 8; i++ {
        verifyProof(api, c.Commitments[i], c.Proofs[i])
    }
    return nil
}
```

### When to Use
- High-throughput verification
- Gas optimization
- Scalability

### Constraint Cost
Depends on individual proof complexity

## Best Practices

### 1. Minimize Constraint Count
- Use lookup tables for common operations
- Batch similar operations
- Avoid redundant constraints

### 2. Optimize Hash Usage
- Hash only when necessary
- Combine multiple values in single hash
- Use MiMC instead of SHA-256

### 3. Constant-Time Operations
- No conditional branches on secrets
- Fixed loop counts
- Uniform memory access

### 4. Clear Separation
- Public inputs: only what verifier needs
- Private inputs: everything else
- Document security assumptions

### 5. Input Validation
- Range constraints on all inputs
- Boolean constraints on flags
- Non-zero constraints where needed

## Performance Comparison

| Pattern | Constraints | Proving Time | Use Case |
|---------|-------------|--------------|----------|
| Commitment | 8,000 | ~100ms | Privacy |
| Range (32-bit) | 32 | <1ms | Bounds |
| Merkle (depth 4) | 32,000 | ~400ms | Membership |
| EdDSA | 40,000 | ~500ms | Authentication |
| Boolean | 2 | <1ms | Conditions |

## Common Pitfalls

### 1. Over-Constraining
Don't add redundant constraints - it slows proving.

### 2. Under-Constraining
Ensure all invariants are enforced in circuit.

### 3. Timing Attacks
Avoid secret-dependent execution paths.

### 4. Hash Collisions
Use proper salts in all commitments.

### 5. Integer Overflow
Always use range constraints on arithmetic.
