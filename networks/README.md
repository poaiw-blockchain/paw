# Testnet/Mainnet Artifact Publishing

Use this directory to stage the public artifacts for every PAW network (testnet or mainnet). Each network should live under `networks/<chain-id>/` with the following layout:

```
networks/
  paw-testnet-1/
    README.md               # Notes, changelog, endpoint list
    genesis.json            # Canonical genesis file (copy from packaging script)
    genesis.sha256          # SHA256 checksum for validators to verify
    peers.txt               # Seeds + persistent peers shared publicly
    STATUS.md               # (optional) status log/action items for the network
    checkpoints/            # (optional) snapshot metadata
```

## Publishing Workflow

1. Run `./scripts/devnet/package-testnet-artifacts.sh <output-dir>` on the canonical node.
2. Copy the generated `*-genesis.json` and `*-genesis.sha256` files into the matching network directory.
3. Review the generated `*-peer-metadata.txt`, rename it to `peers.txt`, and prune/update entries as needed before sharing.
4. Run `./scripts/devnet/verify-network-artifacts.sh <chain-id>` to confirm the checksum and chain-id match.
5. Commit/publish the files (or sync them to CDN/object storage) so validators can fetch them.
   - Tip: `./scripts/devnet/publish-testnet-artifacts.sh` automates steps 1-3 and syncs into `networks/<chain-id>/`.

> Never place private keys, node DBs, or any confidential data under `networks/`. Only public bootstrap material belongs here.
