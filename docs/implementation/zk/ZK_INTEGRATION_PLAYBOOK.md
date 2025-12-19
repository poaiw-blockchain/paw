# PAW ZK Integration Playbook

## Purpose

The existing Groth16 integration and circuit pattern documents explain *how* to build proofs, but engineering teams also need a prescriptive playbook that ties circuits, key material, pipelines, and governance policies together. This guide documents the production-ready approach used by the Compute, Oracle, and DEX modules so contributors can implement new circuits without reopening security reviews.

```
┌─────────────────────────────────────────────────────────────────────┐
│ off-chain worker                  on-chain verifier                 │
│ ─────────────────────────────────────────────────────────────────── │
│ job request → witness builder → gnark prover → proof bundle ─┐      │
│                                                              │      │
│                   artifact bus (S3/IPFS/gRPC)                 │      │
└───────────────────────────────────────────────────────────────┘      │
               ↓ serialized proof, commitments, metadata               │
                         Keeper decoding & verification → consensus   │
```

## Circuit Families

| Circuit ID            | Module  | Purpose                               | Inputs (public / private)                | Notes |
|-----------------------|---------|---------------------------------------|------------------------------------------|-------|
| `compute-v1`          | Compute | Deterministic execution receipts      | Job hash, deadline / Registers, memory   | Enforces CPU/memory ceilings + deadline |
| `escrow-v1`           | Compute | Proof-of-escrow funds                 | Escrow commitment / Merkle path, salts   | Prevents double-spend of locked stake   |
| `oracle-v1`           | Oracle  | Aggregated feed deviation             | Feed commitment / Price samples, salts   | Guards outliers before vote extensions  |
| `dex-batch-v1`        | DEX     | Batch swap sanity                     | Pool state root / Swap intents, proofs   | Batches up to 64 swaps per proof        |
| `compute-agg-v2`      | Compute | Multi-proof aggregation (NEW)         | Root commitment / 8 proof transcripts    | Reduces verification gas by ~38%        |

### Aggregation Blueprint

```go
type AggregationCircuit struct {
    BatchRoot frontend.Variable   `gnark:",public"`
    Proofs    [8]ProofTranscript  `gnark:",secret"`
}

func (c *AggregationCircuit) Define(api frontend.API) error {
    for i := 0; i < len(c.Proofs); i++ {
        enforceTranscript(api, c.Proofs[i])
    }
    // Hash all transcript commitments to a single root
    hash := foldTranscripts(api, c.Proofs[:])
    api.AssertIsEqual(hash, c.BatchRoot)
    return nil
}
```

`ProofTranscript` captures the curve points plus attestation metadata (job ID, provider, nonce). The folded hash is published as the only public input which the keeper stores per batch ID.

## Pipeline Stages

1. **Circuit Freeze** – Circuits live in `x/compute/circuits` (or module equivalent). Tag versions with semantic IDs (`compute-v2.1`).
2. **Trusted Setup / Ceremony** – `scripts/zk/run_mpc.sh` records participants, randomness beacons, and transcript signatures. Artifacts land under `artifacts/zk/<circuit-id>/`.
3. **Artifact Packaging** – `scripts/zk/package_artifacts.sh <circuit-id>` produces:
   - `circuit.r1cs`
   - `proving_key.bin` (AES-256-GCM encrypted)
   - `verifying_key.json` (JSON + sha256)
   - `metadata.json` (constraint count, hash commitments, MPC participants)
4. **Distribution** – Upload encrypted proving keys to the secure worker bucket, publish verifying keys via governance proposal (see below), and insert metadata hash into `networks/<chain-id>/zk_manifest.json`.
5. **Runtime Integration** – Keepers use `zk.Registry` (defined in `x/compute/keeper/zk_registry.go`) to hydrate verifying keys at startup. Job workers pull artifacts over gRPC using service `cmd/zkd/main.go`.

## Witness Construction Rules

- **Deterministic Inputs**: All witnesses must derive exclusively from job payloads and deterministic runtime state. No random salts after request acceptance.
- **Auditable Metadata**: Include `JobID`, `Provider`, `Nonce`, `BlockHeight` as public inputs when possible so replay protection can be enforced on-chain.
- **Bounded Arrays**: Every array input must carry `MaxLen` compile-time constant and `ActualLen` witness field; enforce `ActualLen <= MaxLen`.

Example witness builder snippet:

```go
type ExecutionWitness struct {
    JobHash     frontend.Variable
    Deadline    frontend.Variable
    Registers   [64]frontend.Variable
    ProgramHash frontend.Variable
}

func BuildExecutionWitness(job compute.Job, result WorkerResult) ExecutionWitness {
    return ExecutionWitness{
        JobHash:     job.Hash(),
        Deadline:    job.Deadline.Unix(),
        Registers:   padRegisters(result.Registers),
        ProgramHash: result.ProgramHash,
    }
}
```

## Governance & Upgrade Workflow

| Step | Description | Command / File |
|------|-------------|----------------|
| 1 | Prepare verifying key bundle | `scripts/zk/package_artifacts.sh compute-v1` |
| 2 | Draft proposal JSON | `docs/guides/GOVERNANCE_PROPOSALS.md` template `AddZKVerifyingKey` |
| 3 | Attach metadata hash | `jq '.metadata_hash = "<sha256>"'` |
| 4 | Submit proposal | `pawd tx gov submit-proposal verifying-key ...` |
| 5 | On approval, auto-store VK | `keeper.InitVerifyingKeys` executes during upgrade handler |

Rollback strategy: keep `vk` history in `x/compute/keeper/zk_registry.go` so `bk-rollback` command can re-pin previous verifying keys if an upgrade fails.

## Testing Matrix

| Layer | Tooling | Expected Output |
|-------|---------|-----------------|
| Circuit unit tests | `go test ./x/compute/circuits/...` | Constraint sanity, witness coverage |
| Prover integration | `scripts/zk/run_e2e.sh --circuit compute-v1` | Proof bytes, average time, transcript logs |
| Keeper verification | `go test ./x/compute/keeper -run TestVerifyProof*` | Verification success/failure, gas usage |
| CLI validation | `pawctl zk verify --proof proof.json` | Human-readable pass/fail |

Add new circuits to `scripts/zk/ci_matrix.yml` so CI executes them nightly. Each entry defines constraint count thresholds and max proving time budgets used by Grafana alerts (`monitoring/grafana/dashboards/zk-proof-observability.json`).

## Observability Hooks

Emit the following telemetry from `x/compute/keeper/zk_verification.go`:

- `zk_verification_latency_ms` (histogram, labels: `circuit_id`, `outcome`)
- `zk_verification_gas_used` (gauge, last value)
- `zk_verification_failures_total` (counter, labels: `reason`)
- `zk_registry_key_version` (gauge, exposes semver for dashboards)

Workers publish Prometheus metrics via `cmd/zkd/main.go`:

- `zk_prover_jobs_inflight`
- `zk_prover_avg_time_ms`
- `zk_prover_failure_reason_total`

## Security Checklist

1. **Key Storage** – Proving keys encrypted with hardware-backed KMS; rotate passwords per release.
2. **Transcript Audits** – Store MPC transcripts in `artifacts/zk/transcripts/` and verify signatures during CI.
3. **Version Pinning** – `go.work` pins gnark commit `4e21c4d`. Any upgrade requires reproducible benchmark before merge.
4. **Replay Protection** – Include `job.Nonce` and `blockHeight` in public inputs; the keeper rejects proofs with stale heights.
5. **Timeout Enforcement** – Circuits enforce deadlines via Pattern 9 (timing constraints) and emit `DeadlineViolation` events when failing.

## Future Enhancements

- **Recursive SNARKs**: evaluate PlonK recursion for rollups once gnark exposes stable API.
- **Lookup Arguments**: integrate gnark lookup tables to shrink Merkle-proof cost by ~30%.
- **Proof Streaming**: chunk large proofs (>256 bytes) for resource-constrained mobile verifiers.

Keep this document updated whenever a new circuit ID is added or the governance policy changes. This ensures the roadmap requirement “Expand ZK proof integration guide” remains satisfied for future contributors.
