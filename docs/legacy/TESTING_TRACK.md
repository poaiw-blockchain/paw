# Testing & Hardening Progress

## Current Status
- Unit tests: `go test ./...` (passing)
- Targeted modules: compute + dex keeper/types (passing)
- Devnet: not yet created
- Smoke tests: not yet created

## Next Steps
- Create docker-compose devnet (2–4 nodes) with sample genesis and funded accounts.
- Add smoke test script: status, bank send, staking (create/join validator), DEX create pool/add liquidity/swap, governance proposal lifecycle.
- Wire devnet + smoke into CI job.
- Add fuzz/property tests for DEX invariants and keeper methods.
- Add IBC/relayer flow test once IBC is enabled.

## Notes
- Track test run outputs in `unit.log` (current passing).
- Update this file as new test suites or hardening steps land.
 - Devnet + smoke workflow added: .github/workflows/devnet-smoke.yml (docker-compose uses scripts/devnet/init_node.sh and scripts/devnet/smoke.sh).
