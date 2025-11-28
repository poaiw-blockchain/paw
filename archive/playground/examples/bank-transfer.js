// Bank Transfer Examples in Multiple Languages

export const bankTransferExamples = {
    javascript: `// JavaScript Example - Send PAW Tokens
const { SigningStargateClient } = require('@cosmjs/stargate');

async function sendTokens() {
    // Connect to network
    const rpcEndpoint = 'https://rpc.paw.zone';
    const client = await SigningStargateClient.connectWithSigner(
        rpcEndpoint,
        offlineSigner
    );

    // Send transaction
    const result = await client.sendTokens(
        senderAddress,
        recipientAddress,
        [{ denom: 'upaw', amount: '1000000' }],
        {
            amount: [{ denom: 'upaw', amount: '5000' }],
            gas: '200000',
        }
    );

    console.log('Transaction hash:', result.transactionHash);
    return result;
}`,

    python: `# Python Example - Send PAW Tokens
import requests
from cosmospy import BIP32DerivationError, Transaction

def send_tokens():
    # API endpoint
    api_url = 'https://api.paw.zone'

    # Get account info
    account_url = f'{api_url}/cosmos/auth/v1beta1/accounts/{sender_address}'
    account_info = requests.get(account_url).json()

    # Build transaction
    tx = Transaction(
        privkey=private_key,
        account_num=account_info['account']['account_number'],
        sequence=account_info['account']['sequence'],
        fee=5000,
        gas=200000,
        memo='',
        chain_id='paw-1',
        sync_mode='sync'
    )

    # Add send message
    tx.add_msg(
        tx_type='transfer',
        sender=sender_address,
        recipient=recipient_address,
        amount=1000000,
        denom='upaw'
    )

    # Sign and broadcast
    result = tx.broadcast()
    print(f'Transaction hash: {result["txhash"]}')
    return result`,

    go: `// Go Example - Send PAW Tokens
package main

import (
    "context"
    "fmt"

    "example.com/cosmos/cosmos-sdk/client"
    "example.com/cosmos/cosmos-sdk/types"
    banktypes "example.com/cosmos/cosmos-sdk/x/bank/types"
)

func sendTokens(clientCtx client.Context) error {
    // Create send message
    msg := banktypes.NewMsgSend(
        senderAddr,
        recipientAddr,
        types.NewCoins(types.NewInt64Coin("upaw", 1000000)),
    )

    // Build and sign transaction
    txBuilder := clientCtx.TxConfig.NewTxBuilder()
    if err := txBuilder.SetMsgs(msg); err != nil {
        return err
    }

    // Set fee
    txBuilder.SetFeeAmount(types.NewCoins(types.NewInt64Coin("upaw", 5000)))
    txBuilder.SetGasLimit(200000)

    // Broadcast transaction
    txBytes, err := clientCtx.TxConfig.TxEncoder()(txBuilder.GetTx())
    if err != nil {
        return err
    }

    res, err := clientCtx.BroadcastTx(txBytes)
    if err != nil {
        return err
    }

    fmt.Printf("Transaction hash: %s\\n", res.TxHash)
    return nil
}`,

    shell: `# Shell (cURL) Example - Query Balance and Build Transaction
# Step 1: Query sender account
curl -X GET "https://api.paw.zone/cosmos/auth/v1beta1/accounts/paw1sender..." \\
  -H "accept: application/json"

# Step 2: Query balance
curl -X GET "https://api.paw.zone/cosmos/bank/v1beta1/balances/paw1sender..." \\
  -H "accept: application/json"

# Step 3: Build transaction (requires signing offline)
# Create transaction JSON:
# {
#   "body": {
#     "messages": [{
#       "@type": "/cosmos.bank.v1beta1.MsgSend",
#       "from_address": "paw1sender...",
#       "to_address": "paw1recipient...",
#       "amount": [{"denom": "upaw", "amount": "1000000"}]
#     }],
#     "memo": "",
#     "timeout_height": "0",
#     "extension_options": [],
#     "non_critical_extension_options": []
#   },
#   "auth_info": {
#     "signer_infos": [],
#     "fee": {
#       "amount": [{"denom": "upaw", "amount": "5000"}],
#       "gas_limit": "200000",
#       "payer": "",
#       "granter": ""
#     }
#   },
#   "signatures": []
# }

# Step 4: Broadcast signed transaction
curl -X POST "https://api.paw.zone/cosmos/tx/v1beta1/txs" \\
  -H "accept: application/json" \\
  -H "Content-Type: application/json" \\
  -d @signed_tx.json`
};
