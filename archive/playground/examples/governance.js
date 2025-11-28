// Governance Examples in Multiple Languages

export const governanceExamples = {
    javascript: `// JavaScript Example - Submit and Vote on Proposal
const { SigningStargateClient } = require('@cosmjs/stargate');

async function submitProposal() {
    // Connect to network
    const rpcEndpoint = 'https://rpc.paw.zone';
    const client = await SigningStargateClient.connectWithSigner(
        rpcEndpoint,
        offlineSigner
    );

    // Query governance parameters
    const paramsResponse = await fetch(
        'https://api.paw.zone/cosmos/gov/v1beta1/params/deposit'
    ).then(r => r.json());

    console.log('Gov params:', paramsResponse);

    const minDeposit = paramsResponse.deposit_params.min_deposit[0];
    console.log('Minimum deposit:', minDeposit);

    // Create proposal message
    const proposalMsg = {
        typeUrl: '/cosmos.gov.v1beta1.MsgSubmitProposal',
        value: {
            content: {
                typeUrl: '/cosmos.gov.v1beta1.TextProposal',
                value: {
                    title: 'Improve Network Performance',
                    description: 'Proposal to upgrade network infrastructure for better performance and reliability.'
                }
            },
            initialDeposit: [minDeposit],
            proposer: proposerAddress
        }
    };

    // Sign and broadcast
    const result = await client.signAndBroadcast(
        proposerAddress,
        [proposalMsg],
        {
            amount: [{ denom: 'upaw', amount: '5000' }],
            gas: '300000',
        }
    );

    console.log('Proposal transaction hash:', result.transactionHash);

    // Query proposals
    const proposalsUrl = 'https://api.paw.zone/cosmos/gov/v1beta1/proposals';
    const proposals = await fetch(proposalsUrl).then(r => r.json());
    console.log('Recent proposals:', proposals.proposals.slice(0, 5));

    return result;
}

async function voteOnProposal(proposalId) {
    const rpcEndpoint = 'https://rpc.paw.zone';
    const client = await SigningStargateClient.connectWithSigner(
        rpcEndpoint,
        offlineSigner
    );

    // Create vote message
    // Vote options: 1=Yes, 2=Abstain, 3=No, 4=NoWithVeto
    const voteMsg = {
        typeUrl: '/cosmos.gov.v1beta1.MsgVote',
        value: {
            proposalId: proposalId,
            voter: voterAddress,
            option: 1 // VOTE_OPTION_YES
        }
    };

    // Sign and broadcast
    const result = await client.signAndBroadcast(
        voterAddress,
        [voteMsg],
        {
            amount: [{ denom: 'upaw', amount: '5000' }],
            gas: '200000',
        }
    );

    console.log('Vote transaction hash:', result.transactionHash);

    // Query vote tally
    const tallyUrl = \`https://api.paw.zone/cosmos/gov/v1beta1/proposals/\${proposalId}/tally\`;
    const tally = await fetch(tallyUrl).then(r => r.json());
    console.log('Current tally:', tally);

    return result;
}`,

    python: `# Python Example - Submit and Vote on Proposal
import requests
import json

def submit_proposal():
    api_url = 'https://api.paw.zone'

    # Query governance parameters
    params_url = f'{api_url}/cosmos/gov/v1beta1/params/deposit'
    params = requests.get(params_url).json()

    print(f'Gov params: {params}')

    min_deposit = params['deposit_params']['min_deposit'][0]
    print(f'Minimum deposit: {min_deposit}')

    # Build proposal message
    proposal_msg = {
        '@type': '/cosmos.gov.v1beta1.MsgSubmitProposal',
        'content': {
            '@type': '/cosmos.gov.v1beta1.TextProposal',
            'title': 'Improve Network Performance',
            'description': 'Proposal to upgrade network infrastructure for better performance and reliability.'
        },
        'initial_deposit': [min_deposit],
        'proposer': proposer_address
    }

    # Create transaction
    tx = {
        'body': {
            'messages': [proposal_msg],
            'memo': 'Governance proposal',
            'timeout_height': '0'
        },
        'auth_info': {
            'fee': {
                'amount': [{'denom': 'upaw', 'amount': '5000'}],
                'gas_limit': '300000'
            }
        }
    }

    print(f'Proposal transaction:\\n{json.dumps(tx, indent=2)}')

    # Query recent proposals
    proposals_url = f'{api_url}/cosmos/gov/v1beta1/proposals'
    proposals = requests.get(proposals_url).json()
    print(f'Recent proposals: {len(proposals["proposals"])}')

    return tx

def vote_on_proposal(proposal_id):
    api_url = 'https://api.paw.zone'

    # Query proposal details
    proposal_url = f'{api_url}/cosmos/gov/v1beta1/proposals/{proposal_id}'
    proposal = requests.get(proposal_url).json()
    print(f'Voting on: {proposal["proposal"]["content"]["title"]}')

    # Build vote message
    # Vote options: 1=Yes, 2=Abstain, 3=No, 4=NoWithVeto
    vote_msg = {
        '@type': '/cosmos.gov.v1beta1.MsgVote',
        'proposal_id': str(proposal_id),
        'voter': voter_address,
        'option': 1  # VOTE_OPTION_YES
    }

    # Create transaction
    tx = {
        'body': {
            'messages': [vote_msg],
            'memo': 'Vote on proposal',
            'timeout_height': '0'
        },
        'auth_info': {
            'fee': {
                'amount': [{'denom': 'upaw', 'amount': '5000'}],
                'gas_limit': '200000'
            }
        }
    }

    # Query current tally
    tally_url = f'{api_url}/cosmos/gov/v1beta1/proposals/{proposal_id}/tally'
    tally = requests.get(tally_url).json()
    print(f'Current tally: {tally}')

    return tx`,

    go: `// Go Example - Submit and Vote on Proposal
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    "example.com/cosmos/cosmos-sdk/client"
    "example.com/cosmos/cosmos-sdk/types"
    govtypes "example.com/cosmos/cosmos-sdk/x/gov/types"
)

func submitProposal(clientCtx client.Context) error {
    // Query governance parameters
    paramsURL := "https://api.paw.zone/cosmos/gov/v1beta1/params/deposit"
    resp, err := http.Get(paramsURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    var paramsResp struct {
        DepositParams struct {
            MinDeposit []types.Coin \`json:"min_deposit"\`
        } \`json:"deposit_params"\`
    }
    json.Unmarshal(body, &paramsResp)

    fmt.Printf("Min deposit: %v\\n", paramsResp.DepositParams.MinDeposit)

    // Create text proposal
    content := govtypes.NewTextProposal(
        "Improve Network Performance",
        "Proposal to upgrade network infrastructure for better performance and reliability.",
    )

    // Create submit proposal message
    msg, err := govtypes.NewMsgSubmitProposal(
        content,
        paramsResp.DepositParams.MinDeposit,
        proposerAddr,
    )
    if err != nil {
        return err
    }

    // Build transaction
    txBuilder := clientCtx.TxConfig.NewTxBuilder()
    if err := txBuilder.SetMsgs(msg); err != nil {
        return err
    }

    // Set fee
    txBuilder.SetFeeAmount(types.NewCoins(types.NewInt64Coin("upaw", 5000)))
    txBuilder.SetGasLimit(300000)

    // Sign and broadcast
    txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
    if err != nil {
        return err
    }

    res, err := clientCtx.BroadcastTx(txBytes)
    if err != nil {
        return err
    }

    fmt.Printf("Proposal transaction hash: %s\\n", res.TxHash)
    return nil
}

func voteOnProposal(clientCtx client.Context, proposalID uint64) error {
    // Create vote message
    msg := govtypes.NewMsgVote(
        voterAddr,
        proposalID,
        govtypes.OptionYes,
    )

    // Build transaction
    txBuilder := clientCtx.TxConfig.NewTxBuilder()
    if err := txBuilder.SetMsgs(msg); err != nil {
        return err
    }

    // Set fee
    txBuilder.SetFeeAmount(types.NewCoins(types.NewInt64Coin("upaw", 5000)))
    txBuilder.SetGasLimit(200000)

    // Sign and broadcast
    txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
    if err != nil {
        return err
    }

    res, err := clientCtx.BroadcastTx(txBytes)
    if err != nil {
        return err
    }

    fmt.Printf("Vote transaction hash: %s\\n", res.TxHash)
    return nil
}`,

    shell: `# Shell (cURL) Example - Submit and Vote on Proposal

# Step 1: Query governance parameters
curl -X GET "https://api.paw.zone/cosmos/gov/v1beta1/params/deposit" \\
  -H "accept: application/json"

# Step 2: Query active proposals
curl -X GET "https://api.paw.zone/cosmos/gov/v1beta1/proposals?proposal_status=PROPOSAL_STATUS_VOTING_PERIOD" \\
  -H "accept: application/json"

# Step 3: Submit proposal transaction JSON
# {
#   "body": {
#     "messages": [{
#       "@type": "/cosmos.gov.v1beta1.MsgSubmitProposal",
#       "content": {
#         "@type": "/cosmos.gov.v1beta1.TextProposal",
#         "title": "Improve Network Performance",
#         "description": "Proposal to upgrade network infrastructure."
#       },
#       "initial_deposit": [{"denom": "upaw", "amount": "10000000"}],
#       "proposer": "paw1proposer..."
#     }],
#     "memo": "Governance proposal",
#     "timeout_height": "0"
#   },
#   "auth_info": {
#     "fee": {
#       "amount": [{"denom": "upaw", "amount": "5000"}],
#       "gas_limit": "300000"
#     }
#   }
# }

# Step 4: Vote on proposal
# {
#   "body": {
#     "messages": [{
#       "@type": "/cosmos.gov.v1beta1.MsgVote",
#       "proposal_id": "1",
#       "voter": "paw1voter...",
#       "option": 1
#     }],
#     "memo": "Vote on proposal",
#     "timeout_height": "0"
#   },
#   "auth_info": {
#     "fee": {
#       "amount": [{"denom": "upaw", "amount": "5000"}],
#       "gas_limit": "200000"
#     }
#   }
# }

# Step 5: Query proposal tally
curl -X GET "https://api.paw.zone/cosmos/gov/v1beta1/proposals/1/tally" \\
  -H "accept: application/json"

# Step 6: Query votes
curl -X GET "https://api.paw.zone/cosmos/gov/v1beta1/proposals/1/votes" \\
  -H "accept: application/json"`
};
