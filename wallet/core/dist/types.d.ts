/**
 * Core type definitions for PAW Wallet SDK
 */
export interface WalletAccount {
    address: string;
    pubkey: Uint8Array;
    algo: 'secp256k1';
}
export interface KeyPair {
    privateKey: Uint8Array;
    publicKey: Uint8Array;
}
export interface HDPath {
    coinType: number;
    account: number;
    change: number;
    addressIndex: number;
}
export interface Keystore {
    version: number;
    crypto: {
        cipher: string;
        ciphertext: string;
        cipherparams: {
            iv: string;
        };
        kdf: string;
        kdfparams: {
            salt: string;
            iterations: number;
            keylen: number;
            digest: string;
        };
        mac: string;
    };
    id: string;
    address: string;
    meta?: {
        name?: string;
        timestamp?: number;
    };
}
export interface TransactionOptions {
    memo?: string;
    fee?: {
        amount: Array<{
            denom: string;
            amount: string;
        }>;
        gas: string;
    };
    timeoutHeight?: string;
}
export interface SignedTransaction {
    bodyBytes: Uint8Array;
    authInfoBytes: Uint8Array;
    signatures: Uint8Array[];
}
export interface BroadcastResult {
    code: number;
    transactionHash: string;
    rawLog?: string;
    height?: number;
    gasUsed?: number;
    gasWanted?: number;
}
export interface Balance {
    denom: string;
    amount: string;
}
export interface Validator {
    operatorAddress: string;
    consensusPubkey: string;
    jailed: boolean;
    status: string;
    tokens: string;
    delegatorShares: string;
    description: {
        moniker: string;
        identity: string;
        website: string;
        securityContact: string;
        details: string;
    };
    commission: {
        commissionRates: {
            rate: string;
            maxRate: string;
            maxChangeRate: string;
        };
        updateTime: string;
    };
}
export interface Delegation {
    delegatorAddress: string;
    validatorAddress: string;
    shares: string;
}
export interface Pool {
    id: string;
    tokenA: string;
    tokenB: string;
    reserveA: string;
    reserveB: string;
    totalShares: string;
}
export interface SwapRoute {
    poolId: string;
    tokenIn: string;
    tokenOut: string;
}
export interface GasEstimation {
    gasLimit: string;
    feeAmount: string;
    feeDenom: string;
}
export interface ChainInfo {
    chainId: string;
    chainName: string;
    rpc: string;
    rest: string;
    bech32Prefix: string;
    coinType: number;
    stakeCurrency: {
        coinDenom: string;
        coinMinimalDenom: string;
        coinDecimals: number;
    };
    feeCurrencies: Array<{
        coinDenom: string;
        coinMinimalDenom: string;
        coinDecimals: number;
        gasPriceStep?: {
            low: number;
            average: number;
            high: number;
        };
    }>;
}
export interface PriceData {
    asset: string;
    price: string;
    timestamp: number;
    sources: number;
}
export interface MsgCreatePool {
    creator: string;
    tokenA: string;
    tokenB: string;
    amountA: string;
    amountB: string;
}
export interface MsgAddLiquidity {
    provider: string;
    poolId: number;
    amountA: string;
    amountB: string;
}
export interface MsgRemoveLiquidity {
    provider: string;
    poolId: number;
    shares: string;
}
export interface MsgSwap {
    trader: string;
    poolId: number;
    tokenIn: string;
    tokenOut: string;
    amountIn: string;
    minAmountOut: string;
}
export interface MsgSubmitPrice {
    validator: string;
    feeder: string;
    asset: string;
    price: string;
}
export interface MsgDelegateFeedConsent {
    validator: string;
    delegate: string;
}
export interface MsgSend {
    fromAddress: string;
    toAddress: string;
    amount: Array<{
        denom: string;
        amount: string;
    }>;
}
export interface MsgDelegate {
    delegatorAddress: string;
    validatorAddress: string;
    amount: {
        denom: string;
        amount: string;
    };
}
export interface MsgUndelegate {
    delegatorAddress: string;
    validatorAddress: string;
    amount: {
        denom: string;
        amount: string;
    };
}
export interface MsgBeginRedelegate {
    delegatorAddress: string;
    validatorSrcAddress: string;
    validatorDstAddress: string;
    amount: {
        denom: string;
        amount: string;
    };
}
export interface MsgWithdrawDelegatorReward {
    delegatorAddress: string;
    validatorAddress: string;
}
export interface MsgVote {
    proposalId: string;
    voter: string;
    option: number;
}
export interface Transaction {
    hash: string;
    height: number;
    timestamp: string;
    success: boolean;
    memo?: string;
    fee: {
        amount: Balance[];
        gas: string;
    };
    messages: Array<{
        type: string;
        value: any;
    }>;
}
export interface NodeInfo {
    nodeId: string;
    listenAddr: string;
    network: string;
    version: string;
    channels: string;
    moniker: string;
    other: {
        txIndex: string;
        rpcAddress: string;
    };
}
export interface BlockInfo {
    chainId: string;
    height: number;
    time: string;
    lastBlockId: {
        hash: string;
    };
    proposerAddress: string;
}
export interface EncryptedData {
    ciphertext: string;
    salt: string;
    iv: string;
    algorithm: 'AES-256-GCM';
    kdf: 'PBKDF2' | 'Argon2id';
    iterations: number;
}
export interface SecureKeystore {
    version: number;
    crypto: {
        cipher: 'AES-256-GCM';
        ciphertext: string;
        kdf: 'PBKDF2' | 'Argon2id';
        kdfparams: {
            salt: string;
            iterations: number;
            dklen: number;
        };
        mac: string;
        iv: string;
    };
    address: string;
    id: string;
    meta?: {
        name?: string;
        timestamp?: number;
    };
}
export interface KeyDerivationParams {
    password: string;
    salt?: Uint8Array;
    iterations?: number;
    keyLength?: number;
}
export interface PasswordStrength {
    valid: boolean;
    strength: 'weak' | 'medium' | 'strong';
    errors: string[];
    score?: number;
}
//# sourceMappingURL=types.d.ts.map