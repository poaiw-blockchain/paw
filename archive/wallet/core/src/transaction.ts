/**
 * Transaction building and signing for PAW blockchain
 * Supports Cosmos SDK standard messages and PAW custom modules
 */

import { makeSignDoc, makeAuthInfoBytes, TxBodyEncodeObject, Registry } from '@cosmjs/proto-signing';
import { SignMode } from 'cosmjs-types/cosmos/tx/signing/v1beta1/signing';
import { TxRaw } from 'cosmjs-types/cosmos/tx/v1beta1/tx';
import { PubKey } from 'cosmjs-types/cosmos/crypto/secp256k1/keys';
import { Any } from 'cosmjs-types/google/protobuf/any';
import { toBase64, fromBase64 } from '@cosmjs/encoding';
import { Int53 } from '@cosmjs/math';
import {
  TransactionOptions,
  SignedTransaction,
  MsgSend,
  MsgDelegate,
  MsgUndelegate,
  MsgBeginRedelegate,
  MsgWithdrawDelegatorReward,
  MsgVote,
  MsgSwap,
  MsgCreatePool,
  MsgAddLiquidity,
  MsgRemoveLiquidity,
  GasEstimation,
} from './types';
import { signData } from './crypto';

const DEFAULT_GAS_PRICE = '0.025';
const DEFAULT_GAS_DENOM = 'upaw';

/**
 * Create a registry for message types
 */
export function createRegistry(): Registry {
  const registry = new Registry();

  // Register standard Cosmos messages
  registry.register('/cosmos.bank.v1beta1.MsgSend', {} as any);
  registry.register('/cosmos.staking.v1beta1.MsgDelegate', {} as any);
  registry.register('/cosmos.staking.v1beta1.MsgUndelegate', {} as any);
  registry.register('/cosmos.staking.v1beta1.MsgBeginRedelegate', {} as any);
  registry.register('/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward', {} as any);
  registry.register('/cosmos.gov.v1beta1.MsgVote', {} as any);

  // Register PAW custom messages
  registry.register('/paw.dex.v1.MsgSwap', {} as any);
  registry.register('/paw.dex.v1.MsgCreatePool', {} as any);
  registry.register('/paw.dex.v1.MsgAddLiquidity', {} as any);
  registry.register('/paw.dex.v1.MsgRemoveLiquidity', {} as any);

  return registry;
}

/**
 * Build transaction body
 * @param messages - Array of messages to include
 * @param memo - Optional memo
 * @param timeoutHeight - Optional timeout height
 * @returns Encoded transaction body
 */
export function buildTxBody(
  messages: Array<{ typeUrl: string; value: any }>,
  memo: string = '',
  timeoutHeight?: string
): TxBodyEncodeObject {
  return {
    typeUrl: '/cosmos.tx.v1beta1.TxBody',
    value: {
      messages: messages.map((msg) => ({
        typeUrl: msg.typeUrl,
        value: msg.value,
      })),
      memo,
      timeoutHeight: timeoutHeight ? BigInt(timeoutHeight) : BigInt(0),
    },
  };
}

/**
 * Build auth info
 * @param pubkey - Public key
 * @param sequence - Account sequence number
 * @param gasLimit - Gas limit
 * @param feeAmount - Fee amount
 * @param feeDenom - Fee denomination
 * @returns Encoded auth info bytes
 */
export function buildAuthInfo(
  pubkey: Uint8Array,
  sequence: number,
  gasLimit: number,
  feeAmount: string,
  feeDenom: string = DEFAULT_GAS_DENOM
): Uint8Array {
  const pubkeyProto: PubKey = {
    key: pubkey,
  };

  const pubkeyAny: Any = {
    typeUrl: '/cosmos.crypto.secp256k1.PubKey',
    value: PubKey.encode(pubkeyProto).finish(),
  };

  return makeAuthInfoBytes(
    [{ pubkey: pubkeyAny, sequence: BigInt(sequence) }],
    [{ amount: feeAmount, denom: feeDenom }],
    gasLimit,
    undefined,
    undefined,
    SignMode.SIGN_MODE_DIRECT
  );
}

/**
 * Sign transaction
 * @param txBody - Transaction body
 * @param authInfo - Auth info bytes
 * @param chainId - Chain ID
 * @param accountNumber - Account number
 * @param privateKey - Private key for signing
 * @returns Signed transaction
 */
export async function signTransaction(
  txBody: TxBodyEncodeObject,
  authInfo: Uint8Array,
  chainId: string,
  accountNumber: number,
  privateKey: Uint8Array
): Promise<SignedTransaction> {
  const registry = createRegistry();
  const bodyBytes = registry.encode(txBody);

  const signDoc = makeSignDoc(
    bodyBytes,
    authInfo,
    chainId,
    accountNumber
  );

  const signDocBytes = new Uint8Array([
    ...signDoc.bodyBytes,
    ...signDoc.authInfoBytes,
    ...Buffer.from(signDoc.chainId),
    ...new Uint8Array(new BigUint64Array([BigInt(signDoc.accountNumber)]).buffer),
  ]);

  const signature = signData(signDocBytes, privateKey);

  return {
    bodyBytes,
    authInfoBytes: authInfo,
    signatures: [signature],
  };
}

/**
 * Serialize signed transaction to bytes
 * @param signedTx - Signed transaction
 * @returns Transaction bytes
 */
export function serializeSignedTx(signedTx: SignedTransaction): Uint8Array {
  const txRaw: TxRaw = {
    bodyBytes: signedTx.bodyBytes,
    authInfoBytes: signedTx.authInfoBytes,
    signatures: signedTx.signatures,
  };

  return TxRaw.encode(txRaw).finish();
}

/**
 * Encode signed transaction to base64
 * @param signedTx - Signed transaction
 * @returns Base64 encoded transaction
 */
export function encodeTxBase64(signedTx: SignedTransaction): string {
  const txBytes = serializeSignedTx(signedTx);
  return toBase64(txBytes);
}

/**
 * Decode transaction from base64
 * @param base64Tx - Base64 encoded transaction
 * @returns Decoded transaction
 */
export function decodeTxBase64(base64Tx: string): SignedTransaction {
  const txBytes = fromBase64(base64Tx);
  const txRaw = TxRaw.decode(txBytes);

  return {
    bodyBytes: txRaw.bodyBytes,
    authInfoBytes: txRaw.authInfoBytes,
    signatures: txRaw.signatures,
  };
}

/**
 * Estimate gas for transaction
 * @param messages - Transaction messages
 * @param gasMultiplier - Gas multiplier (default: 1.3 for safety margin)
 * @returns Gas estimation
 */
export function estimateGas(messages: any[], gasMultiplier: number = 1.3): GasEstimation {
  // Base gas cost
  let gasLimit = 80000;

  // Add gas per message
  messages.forEach((msg) => {
    switch (msg.typeUrl) {
      case '/cosmos.bank.v1beta1.MsgSend':
        gasLimit += 100000;
        break;
      case '/cosmos.staking.v1beta1.MsgDelegate':
      case '/cosmos.staking.v1beta1.MsgUndelegate':
        gasLimit += 200000;
        break;
      case '/cosmos.staking.v1beta1.MsgBeginRedelegate':
        gasLimit += 250000;
        break;
      case '/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward':
        gasLimit += 150000;
        break;
      case '/cosmos.gov.v1beta1.MsgVote':
        gasLimit += 100000;
        break;
      case '/paw.dex.v1.MsgSwap':
        gasLimit += 300000;
        break;
      case '/paw.dex.v1.MsgCreatePool':
        gasLimit += 400000;
        break;
      case '/paw.dex.v1.MsgAddLiquidity':
      case '/paw.dex.v1.MsgRemoveLiquidity':
        gasLimit += 250000;
        break;
      default:
        gasLimit += 150000;
    }
  });

  // Apply multiplier for safety
  gasLimit = Math.ceil(gasLimit * gasMultiplier);

  // Calculate fee
  const feeAmount = Math.ceil(gasLimit * parseFloat(DEFAULT_GAS_PRICE)).toString();

  return {
    gasLimit: gasLimit.toString(),
    feeAmount,
    feeDenom: DEFAULT_GAS_DENOM,
  };
}

/**
 * Create MsgSend message
 * @param fromAddress - Sender address
 * @param toAddress - Recipient address
 * @param amount - Amount to send
 * @param denom - Token denomination
 * @returns Message object
 */
export function createMsgSend(
  fromAddress: string,
  toAddress: string,
  amount: string,
  denom: string
): { typeUrl: string; value: MsgSend } {
  return {
    typeUrl: '/cosmos.bank.v1beta1.MsgSend',
    value: {
      fromAddress,
      toAddress,
      amount: [{ denom, amount }],
    },
  };
}

/**
 * Create MsgDelegate message
 * @param delegatorAddress - Delegator address
 * @param validatorAddress - Validator address
 * @param amount - Amount to delegate
 * @param denom - Token denomination
 * @returns Message object
 */
export function createMsgDelegate(
  delegatorAddress: string,
  validatorAddress: string,
  amount: string,
  denom: string
): { typeUrl: string; value: MsgDelegate } {
  return {
    typeUrl: '/cosmos.staking.v1beta1.MsgDelegate',
    value: {
      delegatorAddress,
      validatorAddress,
      amount: { denom, amount },
    },
  };
}

/**
 * Create MsgUndelegate message
 * @param delegatorAddress - Delegator address
 * @param validatorAddress - Validator address
 * @param amount - Amount to undelegate
 * @param denom - Token denomination
 * @returns Message object
 */
export function createMsgUndelegate(
  delegatorAddress: string,
  validatorAddress: string,
  amount: string,
  denom: string
): { typeUrl: string; value: MsgUndelegate } {
  return {
    typeUrl: '/cosmos.staking.v1beta1.MsgUndelegate',
    value: {
      delegatorAddress,
      validatorAddress,
      amount: { denom, amount },
    },
  };
}

/**
 * Create MsgBeginRedelegate message
 * @param delegatorAddress - Delegator address
 * @param validatorSrcAddress - Source validator address
 * @param validatorDstAddress - Destination validator address
 * @param amount - Amount to redelegate
 * @param denom - Token denomination
 * @returns Message object
 */
export function createMsgRedelegate(
  delegatorAddress: string,
  validatorSrcAddress: string,
  validatorDstAddress: string,
  amount: string,
  denom: string
): { typeUrl: string; value: MsgBeginRedelegate } {
  return {
    typeUrl: '/cosmos.staking.v1beta1.MsgBeginRedelegate',
    value: {
      delegatorAddress,
      validatorSrcAddress,
      validatorDstAddress,
      amount: { denom, amount },
    },
  };
}

/**
 * Create MsgWithdrawDelegatorReward message
 * @param delegatorAddress - Delegator address
 * @param validatorAddress - Validator address
 * @returns Message object
 */
export function createMsgWithdrawReward(
  delegatorAddress: string,
  validatorAddress: string
): { typeUrl: string; value: MsgWithdrawDelegatorReward } {
  return {
    typeUrl: '/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward',
    value: {
      delegatorAddress,
      validatorAddress,
    },
  };
}

/**
 * Create MsgVote message
 * @param proposalId - Proposal ID
 * @param voter - Voter address
 * @param option - Vote option (1: Yes, 2: Abstain, 3: No, 4: NoWithVeto)
 * @returns Message object
 */
export function createMsgVote(
  proposalId: string,
  voter: string,
  option: 1 | 2 | 3 | 4
): { typeUrl: string; value: MsgVote } {
  return {
    typeUrl: '/cosmos.gov.v1beta1.MsgVote',
    value: {
      proposalId,
      voter,
      option,
    },
  };
}

/**
 * Create MsgSwap message for DEX
 * @param trader - Trader address
 * @param poolId - Pool ID
 * @param tokenIn - Input token denom
 * @param tokenOut - Output token denom
 * @param amountIn - Input amount
 * @param minAmountOut - Minimum output amount (slippage protection)
 * @returns Message object
 */
export function createMsgSwap(
  trader: string,
  poolId: number,
  tokenIn: string,
  tokenOut: string,
  amountIn: string,
  minAmountOut: string
): { typeUrl: string; value: MsgSwap } {
  return {
    typeUrl: '/paw.dex.v1.MsgSwap',
    value: {
      trader,
      poolId,
      tokenIn,
      tokenOut,
      amountIn,
      minAmountOut,
    },
  };
}

/**
 * Create MsgCreatePool message for DEX
 * @param creator - Creator address
 * @param tokenA - First token denom
 * @param tokenB - Second token denom
 * @param amountA - Amount of first token
 * @param amountB - Amount of second token
 * @returns Message object
 */
export function createMsgCreatePool(
  creator: string,
  tokenA: string,
  tokenB: string,
  amountA: string,
  amountB: string
): { typeUrl: string; value: MsgCreatePool } {
  return {
    typeUrl: '/paw.dex.v1.MsgCreatePool',
    value: {
      creator,
      tokenA,
      tokenB,
      amountA,
      amountB,
    },
  };
}

/**
 * Create MsgAddLiquidity message for DEX
 * @param provider - Provider address
 * @param poolId - Pool ID
 * @param amountA - Amount of first token
 * @param amountB - Amount of second token
 * @returns Message object
 */
export function createMsgAddLiquidity(
  provider: string,
  poolId: number,
  amountA: string,
  amountB: string
): { typeUrl: string; value: MsgAddLiquidity } {
  return {
    typeUrl: '/paw.dex.v1.MsgAddLiquidity',
    value: {
      provider,
      poolId,
      amountA,
      amountB,
    },
  };
}

/**
 * Create MsgRemoveLiquidity message for DEX
 * @param provider - Provider address
 * @param poolId - Pool ID
 * @param shares - LP shares to remove
 * @returns Message object
 */
export function createMsgRemoveLiquidity(
  provider: string,
  poolId: number,
  shares: string
): { typeUrl: string; value: MsgRemoveLiquidity } {
  return {
    typeUrl: '/paw.dex.v1.MsgRemoveLiquidity',
    value: {
      provider,
      poolId,
      shares,
    },
  };
}

/**
 * Calculate transaction hash
 * @param signedTx - Signed transaction
 * @returns Transaction hash (hex)
 */
export function calculateTxHash(signedTx: SignedTransaction): string {
  const txBytes = serializeSignedTx(signedTx);
  const crypto = require('crypto');
  return crypto.createHash('sha256').update(txBytes).digest('hex').toUpperCase();
}

/**
 * Simulate transaction (dry run)
 * @param messages - Transaction messages
 * @returns Estimated gas and fee
 */
export function simulateTransaction(messages: any[]): GasEstimation {
  return estimateGas(messages, 1.5); // Higher multiplier for simulation
}
