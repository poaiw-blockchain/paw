# Cosmos SDK v0.50 Migration (PAW)

This note captures the required changes and operational expectations after migrating PAW to Cosmos SDK v0.50. It is written for module authors and operators.

## Encoding & Genesis Discipline
- Protobuf is the single source of truth. No custom JSON unmarshalling anywhere in CLI, IBC, or genesis helpers.
- Genesis must be canonical Comet/Cosmos JSON: integers serialized as strings, stable ordering, non-null `app_hash`.
- Invalid genesis is rejected. Operators must fix/normalize offline (CLI tooling canonicalizes during `init`/`gentx`/`collect-gentxs`; runtime never auto-heals).
- IBC acknowledgements and module payloads use proto codecs (see DEX/Oracle/Compute acks via `channeltypes.AcknowledgementFromBz`).

## CLI / Node Boot Flow (strict)
Validated single-node sequence (Phase 0.2):
1) `./build/pawd init <moniker> --chain-id paw-test-1 --home <home>`
2) `./build/pawd add-genesis-account <addr> 1000000000upaw --home <home> --keyring-backend test`
3) `./build/pawd gentx <moniker> 700000000upaw --chain-id paw-test-1 --home <home> --keyring-backend test`
4) `./build/pawd collect-gentxs --home <home>`
5) `./build/pawd start --home <home> --minimum-gas-prices 0.001upaw --grpc.address 127.0.0.1:19090 --api.address tcp://127.0.0.1:1318 --rpc.laddr tcp://127.0.0.1:26658`

Notes:
- Bond denom is `upaw` everywhere (staking, mint, gov, crisis, fees).
- Ports above avoid local conflicts; update only if you know they are free.
- Genesis initial height is stored as a string; canonicalization enforces this.

## Module / Keeper Changes
- Error handling: removed `sdkerrors.Wrap*`; use `fmt.Errorf`/`errors.Join` or module errors.
- Updated query/tx response types to v0.50 signatures; regenerated protobufs via `make proto-gen`.
- Simulation, ibctesting, and testutil network updated to v0.50 harness with PAW keepers.
- Ante handlers and address codec wiring aligned to v0.50 (staking address codec enforced).

## Testing Expectations
- Build: `make build`
- Unit tests: `make test-unit` (some security suites may be skipped pending staking/genesis fixtures)
- Protobuf conformance: regenerate on .proto changes (`make proto-gen`); ensure no manual JSON paths creep back in.
- For any new feature, add round-trip tests on proto JSON and binary encodings; reject lenient parsing in test fixtures.

## Operator Checklist
- Never run with a “fixed” genesis produced by ad-hoc scripts; always canonicalize via the CLI steps above.
- Set minimum gas price (default: `0.001upaw`) before start.
- If changing ports, adjust `--grpc.address`, `--api.address`, and `--rpc.laddr` together to avoid conflicts.

