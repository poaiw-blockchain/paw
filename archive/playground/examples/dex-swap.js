// DEX Swap Examples in Multiple Languages

export const dexSwapExamples = {
    javascript: `// JavaScript Example - Swap Tokens on PAW DEX
const { SigningStargateClient } = require('@cosmjs/stargate');

async function swapTokens() {
    // Connect to network
    const rpcEndpoint = 'https://rpc.paw.zone';
    const client = await SigningStargateClient.connectWithSigner(
        rpcEndpoint,
        offlineSigner
    );

    // Get pool information
    const poolId = 1;
    const poolInfo = await fetch('https://api.paw.zone/paw/dex/v1/pools/1')
        .then(r => r.json());

    console.log('Pool info:', poolInfo);

    // Estimate swap
    const estimateResponse = await fetch(
        'https://api.paw.zone/paw/dex/v1/estimate_swap',
        {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                pool_id: poolId,
                token_in: 'upaw',
                amount_in: '1000000'
            })
        }
    ).then(r => r.json());

    console.log('Estimated output:', estimateResponse);

    // Create swap message
    const swapMsg = {
        typeUrl: '/paw.dex.v1.MsgSwap',
        value: {
            sender: senderAddress,
            poolId: poolId,
            tokenIn: { denom: 'upaw', amount: '1000000' },
            minTokenOut: estimateResponse.amount_out
        }
    };

    // Sign and broadcast
    const result = await client.signAndBroadcast(
        senderAddress,
        [swapMsg],
        {
            amount: [{ denom: 'upaw', amount: '5000' }],
            gas: '300000',
        }
    );

    console.log('Swap transaction hash:', result.transactionHash);
    return result;
}`,

    python: `# Python Example - Swap Tokens on PAW DEX
import requests
import json

def swap_tokens():
    api_url = 'https://api.paw.zone'

    # Get pool information
    pool_id = 1
    pool_url = f'{api_url}/paw/dex/v1/pools/{pool_id}'
    pool_info = requests.get(pool_url).json()
    print(f'Pool info: {pool_info}')

    # Estimate swap
    estimate_data = {
        'pool_id': pool_id,
        'token_in': 'upaw',
        'amount_in': '1000000'
    }
    estimate_url = f'{api_url}/paw/dex/v1/estimate_swap'
    estimate = requests.post(estimate_url, json=estimate_data).json()
    print(f'Estimated output: {estimate}')

    # Build swap message
    swap_msg = {
        '@type': '/paw.dex.v1.MsgSwap',
        'sender': sender_address,
        'pool_id': str(pool_id),
        'token_in': {
            'denom': 'upaw',
            'amount': '1000000'
        },
        'min_token_out': estimate['amount_out']
    }

    # Create transaction (requires signing)
    tx = {
        'body': {
            'messages': [swap_msg],
            'memo': 'DEX Swap',
            'timeout_height': '0'
        },
        'auth_info': {
            'fee': {
                'amount': [{'denom': 'upaw', 'amount': '5000'}],
                'gas_limit': '300000'
            }
        }
    }

    print(f'Transaction to sign: {json.dumps(tx, indent=2)}')
    return tx`,

    go: `// Go Example - Swap Tokens on PAW DEX
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"

    "example.com/cosmos/cosmos-sdk/client"
    "example.com/cosmos/cosmos-sdk/types"
    dextypes "example.com/paw/x/dex/types"
)

func swapTokens(clientCtx client.Context) error {
    // Query pool information
    poolID := uint64(1)
    poolURL := fmt.Sprintf("https://api.paw.zone/paw/dex/v1/pools/%d", poolID)

    resp, err := http.Get(poolURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    fmt.Printf("Pool info: %s\\n", string(body))

    // Create swap message
    tokenIn := types.NewInt64Coin("upaw", 1000000)
    msg := dextypes.NewMsgSwap(
        senderAddr,
        poolID,
        tokenIn,
        types.NewInt(900000), // Min token out (with slippage)
    )

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

    fmt.Printf("Swap transaction hash: %s\\n", res.TxHash)
    return nil
}`,

    shell: `# Shell (cURL) Example - DEX Swap
# Step 1: Query pool information
curl -X GET "https://api.paw.zone/paw/dex/v1/pools/1" \\
  -H "accept: application/json"

# Step 2: Estimate swap output
curl -X POST "https://api.paw.zone/paw/dex/v1/estimate_swap" \\
  -H "accept: application/json" \\
  -H "Content-Type: application/json" \\
  -d '{
    "pool_id": "1",
    "token_in": "upaw",
    "amount_in": "1000000"
  }'

# Step 3: Build swap transaction JSON
# {
#   "body": {
#     "messages": [{
#       "@type": "/paw.dex.v1.MsgSwap",
#       "sender": "paw1sender...",
#       "pool_id": "1",
#       "token_in": {"denom": "upaw", "amount": "1000000"},
#       "min_token_out": "900000"
#     }],
#     "memo": "DEX Swap",
#     "timeout_height": "0"
#   },
#   "auth_info": {
#     "fee": {
#       "amount": [{"denom": "upaw", "amount": "5000"}],
#       "gas_limit": "300000"
#     }
#   }
# }

# Step 4: Broadcast signed transaction
curl -X POST "https://api.paw.zone/cosmos/tx/v1beta1/txs" \\
  -H "accept: application/json" \\
  -H "Content-Type: application/json" \\
  -d @signed_swap_tx.json`
};
