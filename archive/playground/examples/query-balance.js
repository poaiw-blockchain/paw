// Query Balance Examples in Multiple Languages

export const queryBalanceExamples = {
    javascript: `// JavaScript Example - Query Account Balance
async function queryBalance() {
    const api_url = 'https://api.paw.zone';
    const address = 'paw1...'; // Replace with actual address

    // Query all balances
    const balancesUrl = \`\${api_url}/cosmos/bank/v1beta1/balances/\${address}\`;
    const response = await fetch(balancesUrl);
    const data = await response.json();

    console.log('All balances:', data.balances);

    // Query specific denom
    const pawUrl = \`\${api_url}/cosmos/bank/v1beta1/balances/\${address}/by_denom?denom=upaw\`;
    const pawResponse = await fetch(pawUrl);
    const pawData = await pawResponse.json();

    console.log('PAW balance:', pawData.balance);

    // Convert to human-readable format (upaw to PAW)
    const pawAmount = parseInt(pawData.balance.amount) / 1000000;
    console.log(\`Balance: \${pawAmount} PAW\`);

    return data;
}

// Query with error handling
async function queryBalanceWithRetry(address, maxRetries = 3) {
    for (let i = 0; i < maxRetries; i++) {
        try {
            const response = await fetch(
                \`https://api.paw.zone/cosmos/bank/v1beta1/balances/\${address}\`
            );

            if (!response.ok) {
                throw new Error(\`HTTP error! status: \${response.status}\`);
            }

            const data = await response.json();
            return data.balances;
        } catch (error) {
            console.error(\`Attempt \${i + 1} failed:\`, error.message);
            if (i === maxRetries - 1) throw error;
            await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)));
        }
    }
}`,

    python: `# Python Example - Query Account Balance
import requests
from decimal import Decimal

def query_balance(address):
    api_url = 'https://api.paw.zone'

    # Query all balances
    balances_url = f'{api_url}/cosmos/bank/v1beta1/balances/{address}'
    response = requests.get(balances_url)
    data = response.json()

    print(f'All balances: {data["balances"]}')

    # Query specific denom
    paw_url = f'{api_url}/cosmos/bank/v1beta1/balances/{address}/by_denom?denom=upaw'
    paw_response = requests.get(paw_url)
    paw_data = paw_response.json()

    print(f'PAW balance: {paw_data["balance"]}')

    # Convert to human-readable format
    paw_amount = Decimal(paw_data['balance']['amount']) / Decimal('1000000')
    print(f'Balance: {paw_amount} PAW')

    return data

def query_multiple_addresses(addresses):
    """Query balances for multiple addresses"""
    results = {}
    api_url = 'https://api.paw.zone'

    for address in addresses:
        try:
            url = f'{api_url}/cosmos/bank/v1beta1/balances/{address}'
            response = requests.get(url)
            response.raise_for_status()
            results[address] = response.json()['balances']
        except Exception as e:
            print(f'Error querying {address}: {e}')
            results[address] = None

    return results

# Example usage
if __name__ == '__main__':
    address = 'paw1...'
    balance = query_balance(address)
    print(f'\\nFinal balance data: {balance}')`,

    go: `// Go Example - Query Account Balance
package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "strconv"

    "example.com/cosmos/cosmos-sdk/types"
)

type BalancesResponse struct {
    Balances []types.Coin \`json:"balances"\`
}

type BalanceResponse struct {
    Balance types.Coin \`json:"balance"\`
}

func queryBalance(address string) error {
    apiURL := "https://api.paw.zone"

    // Query all balances
    balancesURL := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s", apiURL, address)
    resp, err := http.Get(balancesURL)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    var balancesResp BalancesResponse
    if err := json.Unmarshal(body, &balancesResp); err != nil {
        return err
    }

    fmt.Printf("All balances: %v\\n", balancesResp.Balances)

    // Query specific denom
    pawURL := fmt.Sprintf("%s/cosmos/bank/v1beta1/balances/%s/by_denom?denom=upaw", apiURL, address)
    pawResp, err := http.Get(pawURL)
    if err != nil {
        return err
    }
    defer pawResp.Body.Close()

    pawBody, _ := ioutil.ReadAll(pawResp.Body)

    var balanceResp BalanceResponse
    if err := json.Unmarshal(pawBody, &balanceResp); err != nil {
        return err
    }

    fmt.Printf("PAW balance: %v\\n", balanceResp.Balance)

    // Convert to human-readable format
    amount, _ := strconv.ParseFloat(balanceResp.Balance.Amount.String(), 64)
    pawAmount := amount / 1000000
    fmt.Printf("Balance: %.6f PAW\\n", pawAmount)

    return nil
}

func main() {
    address := "paw1..." // Replace with actual address
    if err := queryBalance(address); err != nil {
        fmt.Printf("Error: %v\\n", err)
    }
}`,

    shell: `# Shell (cURL) Example - Query Account Balance

# Query all balances for an address
curl -X GET "https://api.paw.zone/cosmos/bank/v1beta1/balances/paw1..." \\
  -H "accept: application/json" \\
  | jq '.'

# Query specific denom (PAW)
curl -X GET "https://api.paw.zone/cosmos/bank/v1beta1/balances/paw1.../by_denom?denom=upaw" \\
  -H "accept: application/json" \\
  | jq '.balance'

# Query and format output
curl -s "https://api.paw.zone/cosmos/bank/v1beta1/balances/paw1..." \\
  -H "accept: application/json" \\
  | jq -r '.balances[] | "\\(.denom): \\(.amount)"'

# Query total supply
curl -X GET "https://api.paw.zone/cosmos/bank/v1beta1/supply" \\
  -H "accept: application/json" \\
  | jq '.supply'

# Query supply of specific denom
curl -X GET "https://api.paw.zone/cosmos/bank/v1beta1/supply/by_denom?denom=upaw" \\
  -H "accept: application/json" \\
  | jq '.amount'

# Query denom metadata
curl -X GET "https://api.paw.zone/cosmos/bank/v1beta1/denoms_metadata/upaw" \\
  -H "accept: application/json" \\
  | jq '.metadata'

# Batch query multiple addresses (using shell script)
for addr in "paw1addr1..." "paw1addr2..." "paw1addr3..."; do
  echo "Querying $addr"
  curl -s "https://api.paw.zone/cosmos/bank/v1beta1/balances/$addr" \\
    -H "accept: application/json" \\
    | jq -r '.balances[] | "  \\(.denom): \\(.amount)"'
  echo ""
done`
};
