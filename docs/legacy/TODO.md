# TODO

- [x] Register codecs/interfaces for custom modules (dex, compute, oracle).
- [ ] Wire Msg/Query services + gRPC Gateway + tx/query CLI + begin/end block + simulation hooks for dex (CLI/GRPC wired; run keeper/invariant tests to close out).
- [ ] Wire Msg/Query services + gRPC Gateway + tx/query CLI + invariants/simulation for compute (query server registered; add sims/tests).
- [ ] Wire Msg/Query services + gRPC Gateway + tx/query CLI + invariants/simulation for oracle.
- [x] Initialize SimulationManager in app so simulations run.
- [ ] Enable IBC/transfer scoped keepers and initialize CosmWasm or remove wasm store key appropriately.
- [ ] Restore SDK v0.50 CLI/genesis commands (gentx collect/migrate/config).
- [ ] Unskip and fix compute/oracle/e2e/app tests and boost coverage per roadmap.
