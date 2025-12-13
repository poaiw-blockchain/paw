# PAW Project Guidelines

**Read `../CLAUDE.md` first** - contains all general instructions.

## Project-Specific

**Node:** `~/.paw/` (not in repo)
**Binary:** `go build -o pawd ./cmd/...`
**Init:** `./pawd init <moniker> --chain-id paw-testnet-1`
**Proto:** `make proto-gen` or `buf generate` after modifying `.proto` files
