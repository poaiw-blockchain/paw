# PAW Blockchain

## R2 Artifacts
- **Bucket**: `paw-testnet-artifacts`
- **URL**: https://artifacts.poaiw.org
- **Account ID**: `069b2e071fe1c5bea116a29786f2074c`

### Upload Artifacts
```bash
# Env vars pre-configured in ~/.bashrc
wrangler r2 object put paw-testnet-artifacts/genesis.json --file genesis.json --remote
wrangler r2 object put paw-testnet-artifacts/peers.txt --file peers.txt --remote
wrangler r2 object put paw-testnet-artifacts/addrbook.json --file addrbook.json --remote
```

### Delete
```bash
wrangler r2 object delete paw-testnet-artifacts/<path> --remote
```

## Testnet Server
```bash
ssh paw-testnet  # 54.39.103.49
```

## Chain Info
- Chain ID: `paw-testnet-1`
- Binary: `~/.paw/cosmovisor/genesis/bin/pawd`
- Home: `~/.paw`
