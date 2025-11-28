#!/usr/bin/env node

/**
 * PAW Blockchain - Delegate Tokens Example
 *
 * This example demonstrates how to delegate tokens to a validator.
 *
 * Usage:
 *   node delegate.js <validator_address> <amount>
 */

import { SigningStargateClient } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { stringToPath } from '@cosmjs/crypto';
import dotenv from 'dotenv';

dotenv.config();

const RPC_ENDPOINT = process.env.PAW_RPC_ENDPOINT || 'http://localhost:26657';
const WALLET_PREFIX = 'paw';
const GAS_PRICE = '0.025upaw';

async function delegate(mnemonic, validatorAddress, amount, denom = 'upaw') {
  console.log('Delegating Tokens to Validator...\n');

  try {
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(mnemonic, {
      prefix: WALLET_PREFIX,
      hdPaths: [stringToPath("m/44'/118'/0'/0/0")]
    });

    const accounts = await wallet.getAccounts();
    const delegator = accounts[0].address;

    console.log(`Delegator: ${delegator}`);
    console.log(`Validator: ${validatorAddress}`);
    console.log(`Amount: ${parseInt(amount) / 1_000_000} PAW\n`);

    const client = await SigningStargateClient.connectWithSigner(
      RPC_ENDPOINT,
      wallet,
      { gasPrice: GAS_PRICE }
    );

    // Query validator info
    const validators = await client.getValidators(await client.getHeight());
    const validator = validators.find(v => v.address === validatorAddress);

    if (validator) {
      console.log('Validator Info:');
      console.log(`  Moniker: ${validator.description?.moniker || 'Unknown'}`);
      console.log(`  Commission: ${(parseFloat(validator.commission?.commissionRates?.rate || '0') * 100).toFixed(2)}%`);
      console.log(`  Status: ${validator.status}\n`);
    }

    // Create delegation message
    const delegateMsg = {
      typeUrl: '/cosmos.staking.v1beta1.MsgDelegate',
      value: {
        delegatorAddress: delegator,
        validatorAddress: validatorAddress,
        amount: { denom, amount }
      }
    };

    console.log('Broadcasting delegation...');

    const result = await client.signAndBroadcast(
      delegator,
      [delegateMsg],
      'auto',
      'Delegating tokens to validator'
    );

    if (result.code !== 0) {
      throw new Error(`Delegation failed: ${result.rawLog}`);
    }

    console.log('\n✓ Delegation successful!\n');
    console.log(`Tx Hash: ${result.transactionHash}`);
    console.log(`Height: ${result.height}`);
    console.log(`Gas Used: ${result.gasUsed}\n`);

    // Query new delegation
    const delegation = await client.queryContractSmart(
      delegator,
      { delegation: { delegator_address: delegator, validator_address: validatorAddress } }
    ).catch(() => null);

    if (delegation) {
      console.log('New Delegation Status:');
      console.log(`  Total Delegated: ${delegation.balance?.amount || 'N/A'} ${denom}`);
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
  if (args.length < 2) {
    console.error('Usage: node delegate.js <validator_address> <amount>');
    console.log('\nExample:');
    console.log('  node delegate.js pawvaloper1abc...xyz 1000000');
    process.exit(1);
  }

  const mnemonic = process.env.MNEMONIC;
  if (!mnemonic) {
    console.error('Error: MNEMONIC not set');
    process.exit(1);
  }

  delegate(mnemonic, args[0], args[1]).then(r => {
    if (!r.success) process.exit(1);
  });
}

export { delegate };
