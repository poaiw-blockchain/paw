# Professional-Grade Readiness Checklist

- Security: threat model, SAST/DAST, fuzz/property tests (bank/DEX/keeper invariants), reproducible builds/signing, supply-chain SBOMs, secrets/key mgmt playbooks, incident response plan.
- Testing: multi-node devnet/CI smoke suite (bank/staking/DEX/gov), IBC/relayer flows, state sync/replay, upgrade/migration tests with real genesis snapshots, fuzz/load/soak benchmarks.
- Ops/Observability: hardened configs (RPC/p2p limits, anti-spam), metrics/logs/tracing, alerts/runbooks, backups/DR, chaos drills, resource sizing guidance.
- Protocol safety: invariant coverage in CI sims (fast+long), param sanity checks, downgrade/rollback plans, on-chain upgrade handlers validated.
- Tooling/CLI: polished key/tx/query/genesis/gentx UX, sample configs, deterministic Docker images.
- Networking/IBC: enable transfer + scoped keepers, relayer guides, connection/peer limits, p2p hardening.
- Documentation/Compliance: security policy and disclosure, versioned release notes, licenses/notices, contribution guidelines, code ownership.
- Economics/MEV: validate fee/MEV/TWAP params via simulations, publish economic assumptions and residual risks.
