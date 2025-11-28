// Staking Examples in Multiple Languages

export const stakingExamples = {
    javascript: `// JavaScript Example - Delegate to Validator
const { SigningStargateClient } = require('@cosmjs/stargate');

async function delegateTokens() {
    // Connect to network
    const rpcEndpoint = 'https://rpc.paw.zone';
    const client = await SigningStargateClient.connectWithSigner(
        rpcEndpoint,
        offlineSigner
    );

    // Query validators
    const validatorsResponse = await fetch(
        'https://api.paw.zone/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED'
    ).then(r => r.json());

    console.log('Active validators:', validatorsResponse.validators.length);

    // Select a validator (example: first one)
    const validator = validatorsResponse.validators[0];
    console.log('Delegating to:', validator.description.moniker);

    // Create delegation message
    const delegateMsg = {
        typeUrl: '/cosmos.staking.v1beta1.MsgDelegate',
        value: {
            delegatorAddress: delegatorAddress,
            validatorAddress: validator.operator_address,
            amount: { denom: 'upaw', amount: '1000000' }
        }
    };

    // Sign and broadcast
    const result = await client.signAndBroadcast(
        delegatorAddress,
        [delegateMsg],
        {
            amount: [{ denom: 'upaw', amount: '5000' }],
            gas: '250000',
        }
    );

    console.log('Delegation transaction hash:', result.transactionHash);

    // Query delegation after
    const delegationUrl = \`https://api.paw.zone/cosmos/staking/v1beta1/delegations/\${delegatorAddress}\`;
    const delegations = await fetch(delegationUrl).then(r => r.json());
    console.log('Current delegations:', delegations);

    return result;
}`,

    python: `# Python Example - Delegate to Validator
import requests
import json

def delegate_tokens():
    api_url = 'https://api.paw.zone'

    # Query active validators
    validators_url = f'{api_url}/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED'
    validators = requests.get(validators_url).json()

    print(f'Active validators: {len(validators["validators"])}')

    # Select validator
    validator = validators['validators'][0]
    print(f'Delegating to: {validator["description"]["moniker"]}')

    # Build delegation message
    delegate_msg = {
        '@type': '/cosmos.staking.v1beta1.MsgDelegate',
        'delegator_address': delegator_address,
        'validator_address': validator['operator_address'],
        'amount': {
            'denom': 'upaw',
            'amount': '1000000'
        }
    }

    # Create transaction
    tx = {
        'body': {
            'messages': [delegate_msg],
            'memo': 'Stake delegation',
            'timeout_height': '0'
        },
        'auth_info': {
            'fee': {
                'amount': [{'denom': 'upaw', 'amount': '5000'}],
                'gas_limit': '250000'
            }
        }
    }

    print(f'Transaction to sign:\\n{json.dumps(tx, indent=2)}')

    # After broadcasting, query delegations
    delegations_url = f'{api_url}/cosmos/staking/v1beta1/delegations/{delegator_address}'
    delegations = requests.get(delegations_url).json()
    print(f'Current delegations: {delegations}')

    return tx`,

    go: `// Go Example - Delegate to Validator
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    "example.com/cosmos/cosmos-sdk/client"
    "example.com/cosmos/cosmos-sdk/types"
    stakingtypes "example.com/cosmos/cosmos-sdk/x/staking/types"
)

func delegateTokens(clientCtx client.Context) error {
    // Query validators
    validatorsURL := "https://api.paw.zone/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED"
    resp, err := http.Get(validatorsURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    var validatorsResp struct {
        Validators []stakingtypes.Validator \`json:"validators"\`
    }
    json.Unmarshal(body, &validatorsResp)

    fmt.Printf("Active validators: %d\\n", len(validatorsResp.Validators))

    // Select validator
    validator := validatorsResp.Validators[0]
    fmt.Printf("Delegating to: %s\\n", validator.Description.Moniker)

    // Parse validator address
    valAddr, err := types.ValAddressFromBech32(validator.OperatorAddress)
    if err != nil {
        return err
    }

    // Create delegation message
    amount := types.NewInt64Coin("upaw", 1000000)
    msg := stakingtypes.NewMsgDelegate(delegatorAddr, valAddr, amount)

    // Build transaction
    txBuilder := clientCtx.TxConfig.NewTxBuilder()
    if err := txBuilder.SetMsgs(msg); err != nil {
        return err
    }

    // Set fee
    txBuilder.SetFeeAmount(types.NewCoins(types.NewInt64Coin("upaw", 5000)))
    txBuilder.SetGasLimit(250000)

    // Sign and broadcast
    txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
    if err != nil {
        return err
    }

    res, err := clientCtx.BroadcastTx(txBytes)
    if err != nil {
        return err
    }

    fmt.Printf("Delegation transaction hash: %s\\n", res.TxHash)
    return nil
}`,

    shell: `# Shell (cURL) Example - Delegate to Validator
# Step 1: Query active validators
curl -X GET "https://api.paw.zone/cosmos/staking/v1beta1/validators?status=BOND_STATUS_BONDED" \\
  -H "accept: application/json"

# Step 2: Query current delegations (before)
curl -X GET "https://api.paw.zone/cosmos/staking/v1beta1/delegations/paw1delegator..." \\
  -H "accept: application/json"

# Step 3: Query pending rewards
curl -X GET "https://api.paw.zone/cosmos/distribution/v1beta1/delegators/paw1delegator.../rewards" \\
  -H "accept: application/json"

# Step 4: Build delegation transaction JSON
# {
#   "body": {
#     "messages": [{
#       "@type": "/cosmos.staking.v1beta1.MsgDelegate",
#       "delegator_address": "paw1delegator...",
#       "validator_address": "pawvaloper1validator...",
#       "amount": {"denom": "upaw", "amount": "1000000"}
#     }],
#     "memo": "Stake delegation",
#     "timeout_height": "0"
#   },
#   "auth_info": {
#     "fee": {
#       "amount": [{"denom": "upaw", "amount": "5000"}],
#       "gas_limit": "250000"
#     }
#   }
# }

# Step 5: Broadcast signed transaction
curl -X POST "https://api.paw.zone/cosmos/tx/v1beta1/txs" \\
  -H "accept: application/json" \\
  -H "Content-Type: application/json" \\
  -d @signed_delegate_tx.json

# Step 6: Query delegations (after)
curl -X GET "https://api.paw.zone/cosmos/staking/v1beta1/delegations/paw1delegator..." \\
  -H "accept: application/json"`
};
