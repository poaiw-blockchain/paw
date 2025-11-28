# Python SDK

Complete reference for the PAW Python SDK.

## Installation

```bash
pip install paw-sdk
```

## Quick Start

```python
from paw_sdk import PAWClient, Wallet

# Initialize client
client = PAWClient(
    rpc_url='https://rpc.paw.network',
    chain_id='paw-mainnet-1'
)

# Create wallet
wallet = Wallet.generate()
print(f"Address: {wallet.address}")
print(f"Mnemonic: {wallet.mnemonic}")

# Send tokens
tx = client.send_tokens(
    wallet,
    to_address='paw1...',
    amount='1000000upaw',
    fees='500upaw'
)
print(f"TX Hash: {tx['txhash']}")
```

## API Reference

Full reference at [Python SDK Docs](https://docs.paw.network/python-sdk)

---

**Previous:** [JavaScript SDK](/developer/javascript-sdk) | **Next:** [Go Development](/developer/go-development) →
