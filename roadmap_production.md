# Production Readiness Roadmap (Crypto-Grade Expectations)

### Phase 1: Chain Configuration Baseline
- [x] Harden configuration: lock chain-id, base denom, bech32 prefix, gas prices/fees, min-gas prices, evidence params, and pruning strategy in `config/app.toml` and associated app wiring.
- [x] Document default config values and fee policy, ensuring module params in `app/app.go` match the intended production baseline.

### Phase 2: Genesis + Init Pipeline
- [x] Define a reproducible genesis template (staking params, governance min deposit/voting period, IBC transfer enablement, initial balances).
- [x] Add `scripts/init-testnet.sh` that builds `pawd`, initializes the home (`~/.paw` by default), sets seeds/persistent peers, and applies config patches deterministically.
- [x] Ensure no `~/.paw` data or keys are tracked; document `PAW_HOME` overrides and backups.

### Phase 3: Key Management
- [x] Select keyring backend (`os`/`file`), create validator/faucet key flows, and document backup/restore and distribution procedures.

### Phase 4: Networking/P2P Hardening
- [x] Populate and distribute seeds/persistent_peers; tune `config/config.toml` (addr book strictness, inbound/outbound limits, gossip intervals); enable/verify address book and anti-spam settings; secure Prometheus/pprof exposure.

### Phase 5: Observability
- [x] Add monitoring dashboards/alerts (Prometheus/Grafana), structured logging defaults, and log rotation guidance; script `/status`, `/net_info`, and `/validators` health checks.

### Phase 6: Security + Static Analysis
- [x] Run and gate `gosec`, `staticcheck`, `govulncheck`, and dependency review; address findings and ensure no unsafe feature flags in custom modules/ante/IBC middleware.

### Phase 7: Testing + Quality Gates
- [ ] Run `go test ./...`, `-race`, and coverage gates; add integration/e2e tests for tx flow, IBC handshakes, slashing, and governance.
- [ ] Introduce fuzz/property tests for invariant functions (dispute/appeal indexes, escrow accounting) to catch edge-case regressions.
- [ ] Add size/gas guard tests for ZK/IBC inputs (proof size, public inputs) to ensure DOS protections remain enforced.

### Phase 8: Protobuf/API Surface
- [ ] Regenerate protos (`make proto-gen`), verify gogoproto options, version RPC/API docs in `docs/`, and pin module interface versions.

### Immediate Next Tasks
- [x] Clean remaining staticcheck findings (unused err/ctx in compute tests, replace last WrapSDKContext, suppress deprecated DelegatorAddress/ScalarMult with explicit lint directives).
- [x] Re-run staticcheck on app/cmd/p2p/compute after fixes.
- [x] Run `govulncheck ./...` with extended timeout and capture results.
- [x] Finish gosec sweep by annotating/documenting p2p/app findings (file access, permissions, HTTP writes) and re-run gosec for full coverage. (`gosec -conf .security/.gosec.yml ./app/... ./p2p/...` is clean; full `./...` run still hits upstream SSA panics in explorer/archive packages, tracked for follow-up.)
- [ ] Kick off Phase 7 testing: run `go test ./...`, `go test -race ./...`, capture baseline coverage, and identify missing e2e/IBC/slashing/governance suites. (`go test ./...` passes when integration suite is given `-timeout 30m`; baseline `go test -race ./app/... ./p2p/...` clean—still need race + coverage for `x/compute`, `tests/integration`, governance/e2e flows.)
- [ ] Design fuzz/property tests and size guard suites for compute disputes/escrow/IBC proof inputs to enforce the DOS protections outlined in Phase 7.

### Phase 9: Release Engineering
- [ ] Add reproducible build pipeline (`Makefile` or `scripts/release.sh`) with ldflags (version/commit/chain-id), checksums/signatures, and Docker image build/push.

### Phase 10: Deployment
- [ ] Prepare `k8s/` or systemd manifests with `~/.paw` volumes, resource limits, liveness/readiness probes, upgrade/migration steps, and snapshot strategy documentation.
