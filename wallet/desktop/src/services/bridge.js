import { SigningStargateClient, GasPrice } from '@cosmjs/stargate';
import { DirectSecp256k1HdWallet } from '@cosmjs/proto-signing';
import { coins } from '@cosmjs/amino';
import { ApiService } from './api';
import { KeystoreService } from './keystore';

const DEFAULT_GAS_PRICE = '0.025upaw';
const DEFAULT_PORT = 'transfer';
const DEFAULT_CHANNEL = 'channel-0';
const DEFAULT_TIMEOUT_NS = 15n * 60n * 1_000_000_000n; // 15 minutes

export class BridgeService {
  constructor() {
    this.api = new ApiService();
    this.keystore = new KeystoreService();
  }

  async buildSigner(password) {
    const unlocked = await this.keystore.unlockWallet(password);
    const wallet = await DirectSecp256k1HdWallet.fromMnemonic(unlocked.privateKey, {
      prefix: 'paw',
    });
    const [account] = await wallet.getAccounts();
    if (!account) {
      throw new Error('Unable to derive account for signing');
    }
    return { wallet, address: account.address };
  }

  async bridgeTokens({
    password,
    amount,
    denom,
    destAddress,
    sourceChannel = DEFAULT_CHANNEL,
    sourcePort = DEFAULT_PORT,
    memo = '',
    timeoutSeconds = 900,
    offlineSigner,
    fromAddress,
  }) {
    if (!offlineSigner && (!password || password.length < 8)) {
      throw new Error('Password required to sign bridge transaction');
    }
    if (!amount || Number(amount) <= 0) {
      throw new Error('Amount must be greater than zero');
    }
    if (!destAddress) {
      throw new Error('Destination address required');
    }

    let signer = offlineSigner;
    let address = fromAddress;
    if (!signer) {
      const built = await this.buildSigner(password);
      signer = built.wallet;
      address = built.address;
    } else if (!address) {
      const accounts = await signer.getAccounts();
      address = accounts?.[0]?.address;
    }

    if (!address) {
      throw new Error('Unable to determine sender address for bridge');
    }

    const restEndpoint = await this.api.getEndpoint();
    const rpcEndpoint = restEndpoint.replace('1317', '26657').replace(/\/cosmos.*/, '');

    const client = await SigningStargateClient.connectWithSigner(rpcEndpoint, signer, {
      gasPrice: GasPrice.fromString(DEFAULT_GAS_PRICE),
    });

    const timeoutTimestamp = BigInt(Date.now()) * 1_000_000n + BigInt(timeoutSeconds) * 1_000_000_000n;

    const res = await client.sendIbcTokens(
      address,
      destAddress,
      coins(amount, denom),
      sourcePort,
      sourceChannel,
      undefined,
      timeoutTimestamp,
      memo,
    );

    return res;
  }
}

export default BridgeService;
