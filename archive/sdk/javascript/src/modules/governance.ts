import { PawClient } from '../client';
import {
  Proposal,
  VoteParams,
  DepositParams,
  VoteOption,
  TxResult,
  GasOptions
} from '../types';

export class GovernanceModule {
  constructor(private client: PawClient) {}

  /**
   * Submit a text proposal
   */
  async submitTextProposal(
    proposer: string,
    title: string,
    description: string,
    initialDeposit: string,
    denom: string = 'upaw',
    options?: GasOptions
  ): Promise<TxResult> {
    const message = {
      typeUrl: '/cosmos.gov.v1beta1.MsgSubmitProposal',
      value: {
        content: {
          typeUrl: '/cosmos.gov.v1beta1.TextProposal',
          value: {
            title,
            description
          }
        },
        initialDeposit: [{ denom, amount: initialDeposit }],
        proposer
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(proposer, [message], options);
  }

  /**
   * Vote on a proposal
   */
  async vote(
    voter: string,
    params: VoteParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const message = {
      typeUrl: '/cosmos.gov.v1beta1.MsgVote',
      value: {
        proposalId: params.proposalId,
        voter,
        option: params.option,
        metadata: params.metadata || ''
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(voter, [message], options);
  }

  /**
   * Deposit to a proposal
   */
  async deposit(
    depositor: string,
    params: DepositParams,
    options?: GasOptions
  ): Promise<TxResult> {
    const denom = params.denom || 'upaw';
    const message = {
      typeUrl: '/cosmos.gov.v1beta1.MsgDeposit',
      value: {
        proposalId: params.proposalId,
        depositor,
        amount: [{ denom, amount: params.amount }]
      }
    };

    const txBuilder = this.client.getTxBuilder();
    return await txBuilder.signAndBroadcast(depositor, [message], options);
  }

  /**
   * Get all proposals
   */
  async getProposals(status?: number): Promise<Proposal[]> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const statusParam = status !== undefined ? `?proposal_status=${status}` : '';
      const response = await fetch(`${restEndpoint}/cosmos/gov/v1beta1/proposals${statusParam}`);
      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.proposals || [];
    } catch (error) {
      console.error('Error fetching proposals:', error);
      return [];
    }
  }

  /**
   * Get proposal by ID
   */
  async getProposal(proposalId: string): Promise<Proposal | null> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/gov/v1beta1/proposals/${proposalId}`);
      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return data.proposal || null;
    } catch (error) {
      console.error('Error fetching proposal:', error);
      return null;
    }
  }

  /**
   * Get votes for a proposal
   */
  async getVotes(proposalId: string): Promise<any[]> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/gov/v1beta1/proposals/${proposalId}/votes`);
      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.votes || [];
    } catch (error) {
      console.error('Error fetching votes:', error);
      return [];
    }
  }

  /**
   * Get vote for a specific voter
   */
  async getVote(proposalId: string, voter: string): Promise<any | null> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(
        `${restEndpoint}/cosmos/gov/v1beta1/proposals/${proposalId}/votes/${voter}`
      );
      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return data.vote || null;
    } catch (error) {
      console.error('Error fetching vote:', error);
      return null;
    }
  }

  /**
   * Get deposits for a proposal
   */
  async getDeposits(proposalId: string): Promise<any[]> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/gov/v1beta1/proposals/${proposalId}/deposits`);
      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.deposits || [];
    } catch (error) {
      console.error('Error fetching deposits:', error);
      return [];
    }
  }

  /**
   * Get tally for a proposal
   */
  async getTally(proposalId: string): Promise<any | null> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/gov/v1beta1/proposals/${proposalId}/tally`);
      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return data.tally || null;
    } catch (error) {
      console.error('Error fetching tally:', error);
      return null;
    }
  }

  /**
   * Get governance parameters
   */
  async getParams(paramsType: 'voting' | 'tallying' | 'deposit'): Promise<any | null> {
    try {
      const config = this.client.getConfig();
      const restEndpoint = config.restEndpoint || config.rpcEndpoint.replace(':26657', ':1317');

      const response = await fetch(`${restEndpoint}/cosmos/gov/v1beta1/params/${paramsType}`);
      if (!response.ok) {
        return null;
      }

      const data = await response.json();
      return data[`${paramsType}_params`] || null;
    } catch (error) {
      console.error('Error fetching params:', error);
      return null;
    }
  }

  /**
   * Check if proposal has passed quorum
   */
  async hasQuorum(proposalId: string): Promise<boolean> {
    const tally = await this.getTally(proposalId);
    const tallyParams = await this.getParams('tallying');

    if (!tally || !tallyParams) {
      return false;
    }

    const totalVotes = BigInt(tally.yes) + BigInt(tally.no) + BigInt(tally.abstain) + BigInt(tally.no_with_veto);
    const quorum = BigInt(tallyParams.quorum);

    // Simplified quorum check (actual implementation depends on total bonded tokens)
    return totalVotes > quorum;
  }

  /**
   * Get vote option name
   */
  getVoteOptionName(option: VoteOption): string {
    switch (option) {
      case VoteOption.YES:
        return 'Yes';
      case VoteOption.NO:
        return 'No';
      case VoteOption.ABSTAIN:
        return 'Abstain';
      case VoteOption.NO_WITH_VETO:
        return 'No with Veto';
      default:
        return 'Unspecified';
    }
  }
}
