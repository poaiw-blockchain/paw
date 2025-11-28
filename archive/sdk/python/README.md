# PAW Python SDK

Official Python SDK for interacting with the PAW blockchain.

## Features

- **Async/Await Support**: Built with modern async Python
- **Type Hints**: Full type annotations for better IDE support
- **Wallet Management**: Create, import, and manage wallets with BIP39 mnemonics
- **Module Support**:
  - **Bank**: Send tokens, query balances
  - **DEX**: Create pools, swap tokens, add/remove liquidity
  - **Staking**: Delegate, undelegate, redelegate, withdraw rewards
  - **Governance**: Submit proposals, vote, deposit
- **Production Ready**: Comprehensive error handling and validation

## Installation

```bash
pip install paw-sdk
```

## Quick Start

### Create a Wallet

```python
from paw import PawWallet

# Generate a new mnemonic
mnemonic = PawWallet.generate_mnemonic()
print(f"Save this mnemonic: {mnemonic}")

# Create wallet from mnemonic
wallet = PawWallet("paw")
wallet.from_mnemonic(mnemonic)

# Get address
print(f"Address: {wallet.address}")
```

### Connect to PAW Blockchain

```python
import asyncio
from paw import PawClient, PawWallet, ChainConfig

async def main():
    # Initialize client
    config = ChainConfig(
        rpc_endpoint="http://localhost:26657",
        rest_endpoint="http://localhost:1317",
        chain_id="paw-testnet-1"
    )

    async with PawClient(config) as client:
        # Connect wallet
        await client.connect_wallet(wallet)

        # Query balance
        balance = await client.bank.get_balance(wallet.address, "upaw")
        print(f"Balance: {balance}")

asyncio.run(main())
```

### Send Tokens

```python
from paw.types import SendParams

result = await client.bank.send(
    sender=wallet.address,
    params=SendParams(
        recipient="paw1...",
        amount="1000000",  # 1 PAW (6 decimals)
        denom="upaw"
    )
)

print(f"Transaction hash: {result.transaction_hash}")
```

## Module Examples

### DEX Trading

```python
from paw.types import SwapParams, PoolParams

# Get all pools
pools = await client.dex.get_all_pools()

# Create pool
await client.dex.create_pool(
    creator=wallet.address,
    params=PoolParams(
        token_a="upaw",
        token_b="uatom",
        amount_a="10000000",
        amount_b="5000000"
    )
)

# Swap tokens
await client.dex.swap(
    sender=wallet.address,
    params=SwapParams(
        pool_id=pool.id,
        token_in="upaw",
        amount_in="1000000",
        min_amount_out="900000"
    )
)
```

### Staking

```python
from paw.types import DelegateParams

# Get validators
validators = await client.staking.get_validators()

# Delegate
await client.staking.delegate(
    delegator=wallet.address,
    params=DelegateParams(
        validator_address="pawvaloper1...",
        amount="1000000"
    )
)

# Withdraw rewards
await client.staking.withdraw_rewards(
    delegator=wallet.address,
    validator_address="pawvaloper1..."
)
```

### Governance

```python
from paw.types import VoteParams, VoteOption

# Get proposals
proposals = await client.governance.get_proposals()

# Vote
await client.governance.vote(
    voter=wallet.address,
    params=VoteParams(
        proposal_id="1",
        option=VoteOption.YES
    )
)
```

## Testing

```bash
# Install dev dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Run with coverage
pytest --cov=paw --cov-report=html
```

## Development

```bash
# Install in development mode
pip install -e ".[dev]"

# Format code
black paw tests examples

# Type checking
mypy paw

# Linting
pylint paw
```

## License

MIT License - see [LICENSE](../../LICENSE) for details.

## Support

- Documentation: https://docs.paw.network
- : https://github.com/paw-chain/paw
- Discord: https://discord.gg/paw
