import { PawClient, PawWallet } from '@paw-chain/sdk';

async function main() {
  // 1. Create a new wallet
  const mnemonic = PawWallet.generateMnemonic();
  console.log('Generated mnemonic:', mnemonic);

  const wallet = new PawWallet('paw');
  await wallet.fromMnemonic(mnemonic);

  const address = await wallet.getAddress();
  console.log('Wallet address:', address);

  // 2. Connect to the blockchain
  const client = new PawClient({
    rpcEndpoint: 'http://localhost:26657',
    restEndpoint: 'http://localhost:1317',
    chainId: 'paw-mvp-1'
  });

  await client.connectWithWallet(wallet);
  console.log('Connected to chain:', await client.getChainId());

  // 3. Check balance
  const balance = await client.bank.getBalance(address, 'upaw');
  console.log('Balance:', balance);

  // 4. Get all balances
  const allBalances = await client.bank.getAllBalances(address);
  console.log('All balances:', allBalances);

  // 5. Disconnect
  await client.disconnect();
  console.log('Disconnected');
}

main().catch(console.error);
