"use strict";
/**
 * Transaction building and signing for PAW blockchain
 * Supports Cosmos SDK standard messages and PAW custom modules
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.createRegistry = createRegistry;
exports.buildTxBody = buildTxBody;
exports.buildAuthInfo = buildAuthInfo;
exports.signTransaction = signTransaction;
exports.serializeSignedTx = serializeSignedTx;
exports.encodeTxBase64 = encodeTxBase64;
exports.decodeTxBase64 = decodeTxBase64;
exports.estimateGas = estimateGas;
exports.createMsgSend = createMsgSend;
exports.createMsgDelegate = createMsgDelegate;
exports.createMsgUndelegate = createMsgUndelegate;
exports.createMsgRedelegate = createMsgRedelegate;
exports.createMsgWithdrawReward = createMsgWithdrawReward;
exports.createMsgVote = createMsgVote;
exports.createMsgSwap = createMsgSwap;
exports.createMsgCreatePool = createMsgCreatePool;
exports.createMsgAddLiquidity = createMsgAddLiquidity;
exports.createMsgRemoveLiquidity = createMsgRemoveLiquidity;
exports.calculateTxHash = calculateTxHash;
exports.simulateTransaction = simulateTransaction;
const proto_signing_1 = require("@cosmjs/proto-signing");
const signing_1 = require("cosmjs-types/cosmos/tx/signing/v1beta1/signing");
const tx_1 = require("cosmjs-types/cosmos/tx/v1beta1/tx");
const keys_1 = require("cosmjs-types/cosmos/crypto/secp256k1/keys");
const encoding_1 = require("@cosmjs/encoding");
const crypto_1 = require("./crypto");
const DEFAULT_GAS_PRICE = '0.025';
const DEFAULT_GAS_DENOM = 'upaw';
/**
 * Create a registry for message types
 */
function createRegistry() {
    const registry = new proto_signing_1.Registry();
    // Register standard Cosmos messages
    registry.register('/cosmos.bank.v1beta1.MsgSend', {});
    registry.register('/cosmos.staking.v1beta1.MsgDelegate', {});
    registry.register('/cosmos.staking.v1beta1.MsgUndelegate', {});
    registry.register('/cosmos.staking.v1beta1.MsgBeginRedelegate', {});
    registry.register('/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward', {});
    registry.register('/cosmos.gov.v1beta1.MsgVote', {});
    // Register PAW custom messages
    registry.register('/paw.dex.v1.MsgSwap', {});
    registry.register('/paw.dex.v1.MsgCreatePool', {});
    registry.register('/paw.dex.v1.MsgAddLiquidity', {});
    registry.register('/paw.dex.v1.MsgRemoveLiquidity', {});
    return registry;
}
/**
 * Build transaction body
 * @param messages - Array of messages to include
 * @param memo - Optional memo
 * @param timeoutHeight - Optional timeout height
 * @returns Encoded transaction body
 */
function buildTxBody(messages, memo = '', timeoutHeight) {
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
function buildAuthInfo(pubkey, sequence, gasLimit, feeAmount, feeDenom = DEFAULT_GAS_DENOM) {
    const pubkeyProto = {
        key: pubkey,
    };
    const pubkeyAny = {
        typeUrl: '/cosmos.crypto.secp256k1.PubKey',
        value: keys_1.PubKey.encode(pubkeyProto).finish(),
    };
    return (0, proto_signing_1.makeAuthInfoBytes)([{ pubkey: pubkeyAny, sequence: BigInt(sequence) }], [{ amount: feeAmount, denom: feeDenom }], gasLimit, undefined, undefined, signing_1.SignMode.SIGN_MODE_DIRECT);
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
async function signTransaction(txBody, authInfo, chainId, accountNumber, privateKey) {
    const registry = createRegistry();
    const bodyBytes = registry.encode(txBody);
    const signDoc = (0, proto_signing_1.makeSignDoc)(bodyBytes, authInfo, chainId, accountNumber);
    const signDocBytes = new Uint8Array([
        ...signDoc.bodyBytes,
        ...signDoc.authInfoBytes,
        ...Buffer.from(signDoc.chainId),
        ...new Uint8Array(new BigUint64Array([BigInt(signDoc.accountNumber)]).buffer),
    ]);
    const signature = (0, crypto_1.signData)(signDocBytes, privateKey);
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
function serializeSignedTx(signedTx) {
    const txRaw = {
        bodyBytes: signedTx.bodyBytes,
        authInfoBytes: signedTx.authInfoBytes,
        signatures: signedTx.signatures,
    };
    return tx_1.TxRaw.encode(txRaw).finish();
}
/**
 * Encode signed transaction to base64
 * @param signedTx - Signed transaction
 * @returns Base64 encoded transaction
 */
function encodeTxBase64(signedTx) {
    const txBytes = serializeSignedTx(signedTx);
    return (0, encoding_1.toBase64)(txBytes);
}
/**
 * Decode transaction from base64
 * @param base64Tx - Base64 encoded transaction
 * @returns Decoded transaction
 */
function decodeTxBase64(base64Tx) {
    const txBytes = (0, encoding_1.fromBase64)(base64Tx);
    const txRaw = tx_1.TxRaw.decode(txBytes);
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
function estimateGas(messages, gasMultiplier = 1.3) {
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
function createMsgSend(fromAddress, toAddress, amount, denom) {
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
function createMsgDelegate(delegatorAddress, validatorAddress, amount, denom) {
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
function createMsgUndelegate(delegatorAddress, validatorAddress, amount, denom) {
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
function createMsgRedelegate(delegatorAddress, validatorSrcAddress, validatorDstAddress, amount, denom) {
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
function createMsgWithdrawReward(delegatorAddress, validatorAddress) {
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
function createMsgVote(proposalId, voter, option) {
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
function createMsgSwap(trader, poolId, tokenIn, tokenOut, amountIn, minAmountOut) {
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
function createMsgCreatePool(creator, tokenA, tokenB, amountA, amountB) {
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
function createMsgAddLiquidity(provider, poolId, amountA, amountB) {
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
function createMsgRemoveLiquidity(provider, poolId, shares) {
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
function calculateTxHash(signedTx) {
    const txBytes = serializeSignedTx(signedTx);
    const crypto = require('crypto');
    return crypto.createHash('sha256').update(txBytes).digest('hex').toUpperCase();
}
/**
 * Simulate transaction (dry run)
 * @param messages - Transaction messages
 * @returns Estimated gas and fee
 */
function simulateTransaction(messages) {
    return estimateGas(messages, 1.5); // Higher multiplier for simulation
}
//# sourceMappingURL=transaction.js.map