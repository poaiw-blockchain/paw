import PawAPI from '../PawAPI';
import { signWithLedger } from '../LedgerHardwareSigner';
import { assertBech32Prefix } from './guards';

const DEFAULT_FEE = { amount: [{ denom: 'upaw', amount: '4000' }], gas: '250000' };

async function loadAccountMeta(address: string) {
  const nodeInfo = await PawAPI.getNodeInfo();
  const chainId = nodeInfo?.default_node_info?.network || 'paw-mvp-1';
  const account = await PawAPI.getAccount(address);
  const base = account?.base_account || account?.base_vesting_account?.base_account || account;
  const accountNumber = String(base?.account_number || '');
  const sequence = String(base?.sequence || '');
  if (!accountNumber || !sequence) {
    throw new Error('Unable to load account_number/sequence for hardware signing');
  }
  return { chainId, accountNumber, sequence };
}

export async function signLedgerSend({
  fromAddress,
  toAddress,
  amount,
  denom = 'upaw',
  memo = '',
  accountIndex = 0,
}: {
  fromAddress: string;
  toAddress: string;
  amount: string;
  denom?: string;
  memo?: string;
  accountIndex?: number;
}) {
  assertBech32Prefix(fromAddress, 'paw');
  assertBech32Prefix(toAddress, 'paw');
  const meta = await loadAccountMeta(fromAddress);
  const signDoc = {
    chain_id: meta.chainId,
    account_number: meta.accountNumber,
    sequence: meta.sequence,
    fee: DEFAULT_FEE,
    msgs: [
      {
        type: 'cosmos-sdk/MsgSend',
        value: {
          from_address: fromAddress,
          to_address: toAddress,
          amount: [{ denom, amount }],
        },
      },
    ],
    memo,
  };
  const signed = await signWithLedger(signDoc, accountIndex);
  return { ...signed, signDoc };
}

export async function signLedgerDelegate({
  delegatorAddress,
  validatorAddress,
  amount,
  denom = 'upaw',
  accountIndex = 0,
}: {
  delegatorAddress: string;
  validatorAddress: string;
  amount: string;
  denom?: string;
  accountIndex?: number;
}) {
  assertBech32Prefix(delegatorAddress, 'paw');
  assertBech32Prefix(validatorAddress, 'paw');
  const meta = await loadAccountMeta(delegatorAddress);
  const signDoc = {
    chain_id: meta.chainId,
    account_number: meta.accountNumber,
    sequence: meta.sequence,
    fee: DEFAULT_FEE,
    msgs: [
      {
        type: 'cosmos-sdk/MsgDelegate',
        value: {
          delegator_address: delegatorAddress,
          validator_address: validatorAddress,
          amount: { denom, amount },
        },
      },
    ],
    memo: '',
  };
  const signed = await signWithLedger(signDoc, accountIndex);
  return { ...signed, signDoc };
}

export async function signLedgerVote({
  voter,
  proposalId,
  option,
  accountIndex = 0,
}: {
  voter: string;
  proposalId: string;
  option: string;
  accountIndex?: number;
}) {
  assertBech32Prefix(voter, 'paw');
  const meta = await loadAccountMeta(voter);
  const signDoc = {
    chain_id: meta.chainId,
    account_number: meta.accountNumber,
    sequence: meta.sequence,
    fee: DEFAULT_FEE,
    msgs: [
      {
        type: 'cosmos-sdk/MsgVote',
        value: {
          voter,
          proposal_id: proposalId,
          option,
        },
      },
    ],
    memo: '',
  };
  const signed = await signWithLedger(signDoc, accountIndex);
  return { ...signed, signDoc };
}

export async function signLedgerIbcTransfer({
  sender,
  receiver,
  sourceChannel,
  sourcePort = 'transfer',
  amount,
  denom = 'upaw',
  timeoutHeight = { revision_number: '0', revision_height: '0' },
  timeoutTimestamp = '0',
  accountIndex = 0,
}: {
  sender: string;
  receiver: string;
  sourceChannel: string;
  sourcePort?: string;
  amount: string;
  denom?: string;
  timeoutHeight?: { revision_number: string; revision_height: string };
  timeoutTimestamp?: string;
  accountIndex?: number;
}) {
  assertBech32Prefix(sender, 'paw');
  const meta = await loadAccountMeta(sender);
  const signDoc = {
    chain_id: meta.chainId,
    account_number: meta.accountNumber,
    sequence: meta.sequence,
    fee: { amount: [{ denom: 'upaw', amount: '6000' }], gas: '350000' },
    msgs: [
      {
        type: 'cosmos-sdk/MsgTransfer',
        value: {
          source_port: sourcePort,
          source_channel: sourceChannel,
          sender,
          receiver,
          token: { denom, amount },
          timeout_height: timeoutHeight,
          timeout_timestamp: timeoutTimestamp,
        },
      },
    ],
    memo: '',
  };
  const signed = await signWithLedger(signDoc, accountIndex);
  return { ...signed, signDoc };
}
