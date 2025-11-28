#!/usr/bin/env node

/**
 * PAW Blockchain - Add Liquidity Example
 *
 * This example demonstrates how to add liquidity to a trading pool.
 *
 * Usage:
 *   node add-liquidity.js <tokenA> <tokenB> <amountA> <amountB>
 */

import { SigningStargateClient } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { stringToPath } from '@cosmjs/crypto';
import dotenv from 'dotenv';

dotenv.config();

const RPC_ENDPOINT = process.env.PAW_RPC_ENDPOINT || 'http://localhost:26657';
const WALLET_PREFIX = 'paw';
const GAS_PRICE = '0.025upaw';

async function addLiquidity(mnemonic, tokenA, tokenB, amountA, amountB) {
  console.log('Adding Liquidity to Pool...\n');

  try {
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: WALLET_PREFIX,
      hdPaths: [stringToPath("m/44'/118'/0'/0/0")]
    });

    const accounts = await wallet.getAccounts();
    const provider = accounts[0].address;

    console.log(`Provider: ${provider}`);
    console.log(`Pool: ${tokenA}/${tokenB}`);
    console.log(`Amount A: ${amountA} ${tokenA}`);
    console.log(`Amount B: ${amountB} ${tokenB}\n`);

    const client = await SigningStargateClient.connectWithSigner(
      RPC_ENDPOINT,
      wallet,
      { gasPrice: GAS_PRICE }
    );

    // Create add liquidity message
    const addLiquidityMsg = {
      typeUrl: '/paw.dex.v1.MsgAddLiquidity',
      value: {
        sender: provider,
        tokenA: { denom: tokenA, amount: amountA },
        tokenB: { denom: tokenB, amount: amountB },
        amountAMin: Math.floor(parseInt(amountA) * 0.99).toString(), // 1% slippage
        amountBMin: Math.floor(parseInt(amountB) * 0.99).toString(),
        deadline: Math.floor(Date.now() / 1000) + 300
      }
    };

    console.log('Broadcasting transaction...');

    const result = await client.signAndBroadcast(
      provider,
      [addLiquidityMsg],
      'auto',
      'Adding liquidity to PAW DEX pool'
    );

    if (result.code !== 0) {
      throw new Error(`Transaction failed: ${result.rawLog}`);
    }

    console.log('\n✓ Liquidity added successfully!\n');
    console.log(`Tx Hash: ${result.transactionHash}`);
    console.log(`Height: ${result.height}\n`);

    // Parse LP token amount from events
    const mintEvent = result.events?.find(e => e.type === 'mint_lp_tokens');
    if (mintEvent) {
      const lpAmount = mintEvent.attributes.find(a => a.key === 'amount')?.value;
      console.log(`LP Tokens Received: ${lpAmount}`);
    }

    client.disconnect();

    return {
      success: true,
      txHash: result.transactionHash
    };

  } catch (error) {
    console.error('\n✗ Error:', error.message);
    return {
      success: false,
      error: error.message
    };
  }
}

if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);
  if (args.length < 4) {
    console.error('Usage: node add-liquidity.js <tokenA> <tokenB> <amountA> <amountB>');
    process.exit(1);
  }

  const mnemonic = process.env.MNEMONIC;
  if (!mnemonic) {
    console.error('Error: MNEMONIC not set');
    process.exit(1);
  }

  addLiquidity(mnemonic, ...args).then(r => {
    if (!r.success) process.exit(1);
  });
}

export { addLiquidity };
