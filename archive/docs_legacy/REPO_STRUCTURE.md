Target Professional Repository Structure

Core
- cmd/: CLI entrypoints (`pawd`, `pawcli`)
- app/: Cosmos SDK application wiring
- x/: Modules (compute, dex, oracle, privacy)
- proto/: Protobuf definitions and options
- scripts/: Essential build, localnet, and release scripts
- config/: Sample configs (app.toml, config.toml) and genesis helpers
- docker/ + compose/: Containerization for local/dev/test
- Makefile, go.mod, go.sum: Build and dependency control

Optional (kept minimal)
- docs/: Focused documentation (README, WHITEPAPER, architecture)
- ibc/: Relayer config samples

Archived (moved to archive/)
- dashboards/, explorer/, examples/, faucet/, monitoring/, playground/
- status/, testing-dashboard/
- expansive docs portals, bug-bounty templates, long-form internal guides

Rationale
- Keep the chain repository focused on the node, modules, and protocol.
- Provide minimal, accurate docs and scripts for operating the chain.
- Move ancillary assets to `archive/` for later extraction into separate repos.

