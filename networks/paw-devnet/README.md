# PAW Devnet Network Configuration

This directory contains the network configuration for the PAW blockchain 4-validator devnet.

## Network Information

- **Chain ID**: `paw-devnet`
- **Number of Validators**: 4
- **Genesis Time**: See `genesis.json`
- **Bond Denom**: `upaw`

## Files

### genesis.json
The genesis file for the PAW devnet. This file defines the initial state of the blockchain including:
- Initial validator set (4 validators)
- Genesis accounts and balances
- Module parameters
- Network configuration

### genesis.sha256
SHA-256 checksum of the genesis file for integrity verification.

To verify:
```bash
sha256sum -c genesis.sha256
```

### peers.txt
Peer connection information including:
- Node IDs for all 4 validators
- Seed node configuration (if applicable)
- Persistent peer examples

## Joining the Network

To join this network as a validator or full node:

1. **Initialize your node**:
   ```bash
   pawd init <your-moniker> --chain-id paw-devnet
   ```

2. **Replace genesis**:
   ```bash
   cp genesis.json ~/.paw/config/genesis.json
   ```

3. **Verify genesis**:
   ```bash
   sha256sum ~/.paw/config/genesis.json
   # Should match the hash in genesis.sha256
   ```

4. **Configure peers**:
   Edit `~/.paw/config/config.toml` and add persistent peers from `peers.txt`

5. **Start your node**:
   ```bash
   pawd start
   ```

## Validator Information

All 4 genesis validators are active and signing blocks. See the blockchain explorer or use CLI commands to query validator details:

```bash
pawd query staking validators
pawd query staking validator <validator-address>
```

## Support

For issues or questions about joining the network, see the main repository documentation.
