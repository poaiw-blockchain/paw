PAW Blockchain Whitepaper

Abstract
PAW is a Cosmos SDK–based Layer‑1 blockchain purpose‑built to verifiably coordinate AI compute. The chain integrates three first‑class modules: a DEX for exchange and liquidity, a Compute module for job escrow and verification, and an Oracle for data feeds and price aggregation. Inter‑chain routing is enabled through IBC (ICS‑20 for tokens and custom ports for compute/oracle/DEX messages). Consensus is provided by CometBFT with a target ~4s block time. The economic model uses the native token (uPAW) for gas and module fees.

1. Motivation
- Verifiable AI workloads need an accountable coordination layer: escrow of payments, assignment to workers, verification of results, and dispute resolution.
- Native liquidity (DEX) reduces reliance on external venues and enables on‑chain incentives for compute markets.
- Secure data ingress (Oracle) supports pricing and task selection while preserving safety via slashing hooks.

2. Architecture Overview
- Base Layer: Cosmos SDK v0.50+, CometBFT, IAVL stores, Module framework.
- App Wiring: `app/` sets Bech32 prefix `paw`, min gas price `0.001upaw`, configures keepers, routes, and the AnteHandler.
- Consensus: CometBFT with tuned timeouts (propose/prevote/precommit) targeting ~4s blocks.
- Interoperability: IBC core (channels/ports/clients) plus ICS‑20 transfers. Custom IBC modules registered for `compute`, `dex`, and `oracle` ports.
- APIs: gRPC, gRPC‑Gateway (REST), Tendermint RPC; Swagger enabled when configured.

3. Modules
3.1 DEX (`x/dex`)
- Responsibilities: liquidity pools, swaps, fees, and settlement.
- State: pool records, liquidity positions, fee accrual.
- IBC: ICS‑20 token flow for cross‑chain liquidity; custom IBC hooks wired via a module router.

3.2 Compute (`x/compute`)
- Responsibilities: job creation and funding (escrow), assignment to workers/validators, submission and verification of results, challenge/dispute flows.
- State: jobs, assignments, results, challenges; types include protobuf‑generated state and query/tx bindings.
- Security: integration with staking and slashing keepers; authority‑gated operations where required.
- IBC: dedicated port for cross‑chain compute requests and results; packet types defined in `proto/paw/compute/v1`.
- ZK Hooks: types scaffolded for zero‑knowledge proof metadata (`zk_types.go`) for future proof‑based verification.

3.3 Oracle (`x/oracle`)
- Responsibilities: data and price reporting, aggregation, voting windows, and finalization.
- State: price feeds, voting records, slashing metadata for dishonest reporting.
- IBC: custom port for cross‑chain requests/updates; protobuf definitions in `proto/paw/oracle/v1`.

4. Application Wiring (`app/`)
- Keepers: Account, Bank, Staking, Slashing, Mint, Distribution, FeeGrant, Params, Evidence, Governance, Upgrade, ConsensusParams; plus IBC (core/transfer) and capability scoping.
- Module Manager: deterministic begin/end block ordering across core and custom modules.
- AnteHandler: augmented with IBC, Compute, DEX, and Oracle hooks for fee handling and message validation.
- Upgrades: `UpgradeKeeper` configured for in‑place migrations; store loader and handlers stubbed for chain updates.

5. Inter‑Blockchain Communication
- ICS‑20 Transfers: token bridging enabled via `x/ibc-transfer` keeper wiring.
- Custom Ports: routers attach compute/dex/oracle IBC modules for domain‑specific packets.
- Relayer: Hermes config and security guidance available under `ibc/`.

6. Token and Fees
- Denom: `upaw` (micro‑unit). Min gas price defaulted to `0.001upaw` in app config.
- Fee Sources: transaction gas, DEX swap/LP fees, compute job fees, and oracle reporting fees.
- Distribution: follows Cosmos SDK distribution and staking economics; module specific fees feed into community pool or module accounts per governance.

7. Security Model
- Consensus: CometBFT with validator set slashing for downtime and double‑signing.
- Module Invariants: crisis module registered; invariants can halt on violation in configured environments.
- Slashing: oracle dishonesty and compute misbehavior integrated via `x/slashing` and module hooks.
- Key Management: Bech32 `paw` prefix; hardware wallet workflows via standard Cosmos signing.
- TLS: See `docs/security/` for certificate guidance; never use self‑signed certs in production.

8. P2P and State Sync
- Core P2P is provided by CometBFT. The repo additionally contains a scoped `p2p/` package implementing a reputation‑aware discovery and a state‑sync prototype. This is currently experimental and not wired into the app’s consensus path.

9. Status and Roadmap
- Implementation: DEX/Compute/Oracle modules and IBC wiring are present with protobuf bindings and keepers. ZK types scaffolded for future proof‑based verification.
- Testnet: beta configuration with ~4s block time; default min‑gas prices enabled.
- Upcoming: finalize module message handlers and queries, expand keeper logic, add migrations, formalize governance params, and integrate the experimental P2P where applicable.

10. References
- Cosmos SDK: https://docs.cosmos.network
- IBC: https://ibcprotocol.org/
- CometBFT: https://cometbft.com/

