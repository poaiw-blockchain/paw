# paw-testnet-1 Publish Checklist

Use this checklist when promoting the staged artifacts to the public CDN/bucket.

1. Replace `peers.txt` hostnames with public sentry/validator endpoints (no docker hostnames). Keep seeds empty unless real sentries are online.
2. Re-run packaging to refresh hashes and manifest (ensures `paw-testnet-1-manifest.json.peers_file` points to `peers.txt`):
   ```bash
   PAW_HOME=/root/.paw/node1 PAWD_BIN=$(pwd)/pawd CHAIN_ID=paw-testnet-1 ./scripts/devnet/publish-testnet-artifacts.sh
   ./scripts/devnet/verify-network-artifacts.sh paw-testnet-1
   ./scripts/devnet/bundle-testnet-artifacts.sh
   ```
3. Upload to CDN/bucket (adjust path):
   - `networks/paw-testnet-1/genesis.json`
   - `networks/paw-testnet-1/genesis.sha256`
   - `networks/paw-testnet-1/paw-testnet-1-manifest.json`
   - `networks/paw-testnet-1/peers.txt`
   - `networks/paw-testnet-1/paw-testnet-1-artifacts.sha256`
   - `artifacts/paw-testnet-1-artifacts.tar.gz`
   - Helper: `ARTIFACTS_DEST=s3://<bucket>/paw-testnet-1 ./scripts/devnet/upload-artifacts.sh` (requires AWS CLI)
4. Post-upload verification:
   - Download from CDN and verify `sha256sum` matches `genesis.sha256`.
   - Run `./scripts/devnet/validate-remote-artifacts.sh https://networks.paw.xyz/paw-testnet-1`.
   - Spot-check `peers.txt` resolves (dig/curl) and ports accept TCP on 26656/26657.
   - Hit status page endpoint to confirm live probes show green for RPC/REST/gRPC/Explorer/Faucet/Metrics.
5. Announce external onboarding:
   - Publish CDN URLs + checksum in docs/Discord.
   - Open faucet and monitoring links (RPC/REST/gRPC/Explorer/Status).
