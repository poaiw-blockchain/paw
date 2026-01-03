# PAW Project Guidelines

**Read `../CLAUDE.md` first** - contains all general instructions.

## Project-Specific

**Node:** `~/.paw/` (not in repo)
**Binary:** `go build -o pawd ./cmd/...`
**Init:** `./pawd init <moniker> --chain-id paw-testnet-1`
**Proto:** `make proto-gen` or `buf generate` after modifying `.proto` files
- no summaries longer than 50 lines!!!!
## Testnet Access
- **Server**: `ssh paw-testnet` (54.39.103.49)
- **Chain ID**: paw-testnet-1
- **Binary**: `~/.paw/cosmovisor/genesis/bin/pawd`
- **Home**: `~/.paw`
- **VPN**: 10.10.0.2

### Quick Commands
```bash
pawd status --home ~/.paw
pawd query compute params --home ~/.paw
pawd query dex pools --home ~/.paw
```

**Full docs**: See `TESTNET_INFRASTRUCTURE.md`

## Testnet SSH Access

Use the pre-configured SSH alias:

```bash
ssh paw-testnet  # 54.39.103.49
```

The SSH config is in `~/.ssh/config`. Never store SSH keys in repositories.
