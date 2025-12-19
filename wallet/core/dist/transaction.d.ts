/**
 * Transaction building and signing for PAW blockchain
 * Supports Cosmos SDK standard messages and PAW custom modules
 */
import { TxBodyEncodeObject, Registry } from '@cosmjs/proto-signing';
import { SignedTransaction, MsgSend, MsgDelegate, MsgUndelegate, MsgBeginRedelegate, MsgWithdrawDelegatorReward, MsgVote, MsgSwap, MsgCreatePool, MsgAddLiquidity, MsgRemoveLiquidity, GasEstimation } from './types';
/**
 * Create a registry for message types
 */
export declare function createRegistry(): Registry;
/**
 * Build transaction body
 * @param messages - Array of messages to include
 * @param memo - Optional memo
 * @param timeoutHeight - Optional timeout height
 * @returns Encoded transaction body
 */
export declare function buildTxBody(messages: Array<{
    typeUrl: string;
    value: any;
}>, memo?: string, timeoutHeight?: string): TxBodyEncodeObject;
/**
 * Build auth info
 * @param pubkey - Public key
 * @param sequence - Account sequence number
 * @param gasLimit - Gas limit
 * @param feeAmount - Fee amount
 * @param feeDenom - Fee denomination
 * @returns Encoded auth info bytes
 */
export declare function buildAuthInfo(pubkey: Uint8Array, sequence: number, gasLimit: number, feeAmount: string, feeDenom?: string): Uint8Array;
/**
 * Sign transaction
 * @param txBody - Transaction body
 * @param authInfo - Auth info bytes
 * @param chainId - Chain ID
 * @param accountNumber - Account number
 * @param privateKey - Private key for signing
 * @returns Signed transaction
 */
export declare function signTransaction(txBody: TxBodyEncodeObject, authInfo: Uint8Array, chainId: string, accountNumber: number, privateKey: Uint8Array): Promise<SignedTransaction>;
/**
 * Serialize signed transaction to bytes
 * @param signedTx - Signed transaction
 * @returns Transaction bytes
 */
export declare function serializeSignedTx(signedTx: SignedTransaction): Uint8Array;
/**
 * Encode signed transaction to base64
 * @param signedTx - Signed transaction
 * @returns Base64 encoded transaction
 */
export declare function encodeTxBase64(signedTx: SignedTransaction): string;
/**
 * Decode transaction from base64
 * @param base64Tx - Base64 encoded transaction
 * @returns Decoded transaction
 */
export declare function decodeTxBase64(base64Tx: string): SignedTransaction;
/**
 * Estimate gas for transaction
 * @param messages - Transaction messages
 * @param gasMultiplier - Gas multiplier (default: 1.3 for safety margin)
 * @returns Gas estimation
 */
export declare function estimateGas(messages: any[], gasMultiplier?: number): GasEstimation;
/**
 * Create MsgSend message
 * @param fromAddress - Sender address
 * @param toAddress - Recipient address
 * @param amount - Amount to send
 * @param denom - Token denomination
 * @returns Message object
 */
export declare function createMsgSend(fromAddress: string, toAddress: string, amount: string, denom: string): {
    typeUrl: string;
    value: MsgSend;
};
/**
 * Create MsgDelegate message
 * @param delegatorAddress - Delegator address
 * @param validatorAddress - Validator address
 * @param amount - Amount to delegate
 * @param denom - Token denomination
 * @returns Message object
 */
export declare function createMsgDelegate(delegatorAddress: string, validatorAddress: string, amount: string, denom: string): {
    typeUrl: string;
    value: MsgDelegate;
};
/**
 * Create MsgUndelegate message
 * @param delegatorAddress - Delegator address
 * @param validatorAddress - Validator address
 * @param amount - Amount to undelegate
 * @param denom - Token denomination
 * @returns Message object
 */
export declare function createMsgUndelegate(delegatorAddress: string, validatorAddress: string, amount: string, denom: string): {
    typeUrl: string;
    value: MsgUndelegate;
};
/**
 * Create MsgBeginRedelegate message
 * @param delegatorAddress - Delegator address
 * @param validatorSrcAddress - Source validator address
 * @param validatorDstAddress - Destination validator address
 * @param amount - Amount to redelegate
 * @param denom - Token denomination
 * @returns Message object
 */
export declare function createMsgRedelegate(delegatorAddress: string, validatorSrcAddress: string, validatorDstAddress: string, amount: string, denom: string): {
    typeUrl: string;
    value: MsgBeginRedelegate;
};
/**
 * Create MsgWithdrawDelegatorReward message
 * @param delegatorAddress - Delegator address
 * @param validatorAddress - Validator address
 * @returns Message object
 */
export declare function createMsgWithdrawReward(delegatorAddress: string, validatorAddress: string): {
    typeUrl: string;
    value: MsgWithdrawDelegatorReward;
};
/**
 * Create MsgVote message
 * @param proposalId - Proposal ID
 * @param voter - Voter address
 * @param option - Vote option (1: Yes, 2: Abstain, 3: No, 4: NoWithVeto)
 * @returns Message object
 */
export declare function createMsgVote(proposalId: string, voter: string, option: 1 | 2 | 3 | 4): {
    typeUrl: string;
    value: MsgVote;
};
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
export declare function createMsgSwap(trader: string, poolId: number, tokenIn: string, tokenOut: string, amountIn: string, minAmountOut: string): {
    typeUrl: string;
    value: MsgSwap;
};
/**
 * Create MsgCreatePool message for DEX
 * @param creator - Creator address
 * @param tokenA - First token denom
 * @param tokenB - Second token denom
 * @param amountA - Amount of first token
 * @param amountB - Amount of second token
 * @returns Message object
 */
export declare function createMsgCreatePool(creator: string, tokenA: string, tokenB: string, amountA: string, amountB: string): {
    typeUrl: string;
    value: MsgCreatePool;
};
/**
 * Create MsgAddLiquidity message for DEX
 * @param provider - Provider address
 * @param poolId - Pool ID
 * @param amountA - Amount of first token
 * @param amountB - Amount of second token
 * @returns Message object
 */
export declare function createMsgAddLiquidity(provider: string, poolId: number, amountA: string, amountB: string): {
    typeUrl: string;
    value: MsgAddLiquidity;
};
/**
 * Create MsgRemoveLiquidity message for DEX
 * @param provider - Provider address
 * @param poolId - Pool ID
 * @param shares - LP shares to remove
 * @returns Message object
 */
export declare function createMsgRemoveLiquidity(provider: string, poolId: number, shares: string): {
    typeUrl: string;
    value: MsgRemoveLiquidity;
};
/**
 * Calculate transaction hash
 * @param signedTx - Signed transaction
 * @returns Transaction hash (hex)
 */
export declare function calculateTxHash(signedTx: SignedTransaction): string;
/**
 * Simulate transaction (dry run)
 * @param messages - Transaction messages
 * @returns Estimated gas and fee
 */
export declare function simulateTransaction(messages: any[]): GasEstimation;
//# sourceMappingURL=transaction.d.ts.map