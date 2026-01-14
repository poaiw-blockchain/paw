"""Basic usage example for PAW Python SDK."""

import asyncio
from paw import PawClient, PawWallet, ChainConfig


async def main():
    """Main example function."""
    # 1. Create a new wallet
    mnemonic = PawWallet.generate_mnemonic()
    print(f"Generated mnemonic: {mnemonic}")

    wallet = PawWallet("paw")
    wallet.from_mnemonic(mnemonic)
    print(f"Wallet address: {wallet.address}")

    # 2. Connect to the blockchain
    config = ChainConfig(
        rpc_endpoint="http://localhost:26657",
        rest_endpoint="http://localhost:1317",
        chain_id="paw-mvp-1"
    )

    async with PawClient(config) as client:
        await client.connect_wallet(wallet)
        print(f"Connected to chain: {await client.get_chain_id()}")

        # 3. Check balance
        balance = await client.bank.get_balance(wallet.address, "upaw")
        if balance:
            print(f"Balance: {client.bank.format_balance(balance)}")

        # 4. Get all balances
        all_balances = await client.bank.get_all_balances(wallet.address)
        print(f"All balances: {len(all_balances)} tokens")


if __name__ == "__main__":
    asyncio.run(main())
