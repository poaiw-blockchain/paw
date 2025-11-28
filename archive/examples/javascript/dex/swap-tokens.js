#!/usr/bin/env node

/**
 * PAW Blockchain - Swap Tokens Example
 *
 * This example demonstrates how to swap tokens using the PAW DEX.
 *
 * Usage:
 *   node swap-tokens.js <tokenIn> <tokenOut> <amountIn> [slippage]
 */

import { SigningStargateClient } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { stringToPath } from '@cosmjs/crypto';
import dotenv from 'dotenv';

dotenv.config();

const RPC_ENDPOINT = process.env.PAW_RPC_ENDPOINT || 'http://localhost:26657';
const WALLET_PREFIX = 'paw';
const HD_PATH = "m/44'/118'/0'/0/0";
const GAS_PRICE = process.env.GAS_PRICE || '0.025upaw';
const DEFAULT_SLIPPAGE = 0.01; // 1%

/**
 * Swap tokens on PAW DEX
 * @param {string} mnemonic - Wallet mnemonic
 * @param {string} tokenIn - Input token denom
 * @param {string} tokenOut - Output token denom
 * @param {string} amountIn - Amount of input tokens
 * @param {number} slippage - Max acceptable slippage (0.01 = 1%)
 * @returns {Promise<Object>} Swap result
 */
async function swapTokens(mnemonic, tokenIn, tokenOut, amountIn, slippage = DEFAULT_SLIPPAGE) {
  console.log('PAW DEX - Token Swap\n');

  try {
    // Create wallet
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: WALLET_PREFIX,
      hdPaths: [stringToPath(HD_PATH)]
    });

    const accounts = await wallet.getAccounts();
    const trader = accounts[0].address;

    console.log(`Trader: ${trader}`);
    console.log(`Swap: ${amountIn} ${tokenIn} → ${tokenOut}`);
    console.log(`Max Slippage: ${(slippage * 100).toFixed(2)}%\n`);

    // Connect to network
    const client = await SigningStargateClient.connectWithSigner(
      RPC_ENDPOINT,
      wallet,
      { gasPrice: GAS_PRICE }
    );

    // Query current pool price for route calculation
    const poolInfo = await queryPoolInfo(client, tokenIn, tokenOut);
    console.log('Pool Information:');
    console.log(`  Pool ID: ${poolInfo.poolId}`);
    console.log(`  Reserves: ${poolInfo.reserveIn} ${tokenIn}, ${poolInfo.reserveOut} ${tokenOut}`);
    console.log(`  Price: 1 ${tokenIn} = ${poolInfo.price.toFixed(6)} ${tokenOut}\n`);

    // Calculate expected output
    const expectedOut = calculateSwapOutput(
      amountIn,
      poolInfo.reserveIn,
      poolInfo.reserveOut
    );

    // Calculate minimum output with slippage
    const minAmountOut = Math.floor(expectedOut * (1 - slippage));

    console.log('Swap Calculation:');
    console.log(`  Expected Output: ${expectedOut} ${tokenOut}`);
    console.log(`  Minimum Output: ${minAmountOut} ${tokenOut}`);
    console.log(`  Price Impact: ${calculatePriceImpact(amountIn, poolInfo.reserveIn).toFixed(4)}%\n`);

    // Create swap message
    const swapMsg = {
      typeUrl: '/paw.dex.v1.MsgSwapExactTokensForTokens',
      value: {
        sender: trader,
        amountIn: amountIn,
        amountOutMin: minAmountOut.toString(),
        path: [tokenIn, tokenOut],
        deadline: Math.floor(Date.now() / 1000) + 300 // 5 minutes
      }
    };

    console.log('Broadcasting swap transaction...');

    // Execute swap
    const result = await client.signAndBroadcast(
      trader,
      [swapMsg],
      'auto',
      'Token swap via PAW DEX'
    );

    if (result.code !== 0) {
      throw new Error(`Swap failed: ${result.rawLog}`);
    }

    console.log('\n✓ Swap successful!\n');
    console.log('Transaction Details:');
    console.log(`  Tx Hash: ${result.transactionHash}`);
    console.log(`  Height: ${result.height}`);
    console.log(`  Gas Used: ${result.gasUsed}\n`);

    // Parse swap events to get actual amounts
    const swapEvent = result.events?.find(e => e.type === 'swap');
    if (swapEvent) {
      const actualOut = swapEvent.attributes.find(a => a.key === 'amount_out')?.value;
      if (actualOut) {
        console.log(`Actual Output: ${actualOut} ${tokenOut}`);
        const slippageUsed = ((expectedOut - parseInt(actualOut)) / expectedOut * 100).toFixed(4);
        console.log(`Actual Slippage: ${slippageUsed}%`);
      }
    }

    client.disconnect();

    return {
      success: true,
      txHash: result.transactionHash,
      expectedOut,
      minAmountOut
    };

  } catch (error) {
    console.error('\n✗ Swap failed:', error.message);
    return {
      success: false,
      error: error.message
    };
  }
}

/**
 * Query pool information
 */
async function queryPoolInfo(client, tokenA, tokenB) {
  // Mock implementation - in production, query actual pool
  return {
    poolId: '1',
    reserveIn: '1000000000',
    reserveOut: '5000000000',
    price: 5.0
  };
}

/**
 * Calculate swap output using constant product formula (x * y = k)
 * @param {string} amountIn - Input amount
 * @param {string} reserveIn - Reserve of input token
 * @param {string} reserveOut - Reserve of output token
 * @param {number} fee - Trading fee (default 0.003 = 0.3%)
 * @returns {number} Expected output amount
 */
function calculateSwapOutput(amountIn, reserveIn, reserveOut, fee = 0.003) {
  const amountInNum = parseInt(amountIn);
  const reserveInNum = parseInt(reserveIn);
  const reserveOutNum = parseInt(reserveOut);

  // Apply fee
  const amountInWithFee = amountInNum * (1 - fee);

  // Constant product formula: (x + Δx) * (y - Δy) = x * y
  // Δy = (y * Δx) / (x + Δx)
  const numerator = amountInWithFee * reserveOutNum;
  const denominator = reserveInNum + amountInWithFee;

  return Math.floor(numerator / denominator);
}

/**
 * Calculate price impact percentage
 */
function calculatePriceImpact(amountIn, reserve) {
  return (parseInt(amountIn) / parseInt(reserve)) * 100;
}

// Run if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  const args = process.argv.slice(2);

  if (args.length < 3) {
    console.error('Usage: node swap-tokens.js <tokenIn> <tokenOut> <amountIn> [slippage]');
    console.log('\nExample:');
    console.log('  node swap-tokens.js upaw uatom 1000000 0.01');
    process.exit(1);
  }

  const mnemonic = process.env.MNEMONIC;
  if (!mnemonic) {
    console.error('Error: MNEMONIC not set');
    process.exit(1);
  }

  const [tokenIn, tokenOut, amountIn, slippageStr] = args;
  const slippage = slippageStr ? parseFloat(slippageStr) : DEFAULT_SLIPPAGE;

  swapTokens(mnemonic, tokenIn, tokenOut, amountIn, slippage)
    .then(result => {
      if (!result.success) process.exit(1);
    });
}

export { swapTokens, calculateSwapOutput, calculatePriceImpact };
