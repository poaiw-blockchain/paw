/**
 * PAW Testing Control Panel - Configuration
 * Network configurations for local, testnet, and mainnet
 */

export const CONFIG = {
    networks: {
        local: {
            name: 'Local Testnet',
            rpcUrl: 'http://localhost:26657',
            restUrl: 'http://localhost:1317',
            chainId: 'paw-local-1',
            faucetUrl: 'http://localhost:8000',
            explorerUrl: null,
            features: {
                reset: true,
                fullControl: true,
                faucet: true
            }
        },
        testnet: {
            name: 'Public Testnet',
            rpcUrl: 'https://testnet-rpc.paw.network',
            restUrl: 'https://testnet-api.paw.network',
            chainId: 'paw-testnet-1',
            faucetUrl: 'https://faucet.paw.network',
            explorerUrl: 'https://testnet-explorer.paw.network',
            features: {
                reset: false,
                fullControl: false,
                faucet: true
            }
        },
        mainnet: {
            name: 'Mainnet (Read-Only)',
            rpcUrl: 'https://rpc.paw.network',
            restUrl: 'https://api.paw.network',
            chainId: 'paw-1',
            faucetUrl: null,
            explorerUrl: 'https://explorer.paw.network',
            features: {
                reset: false,
                fullControl: false,
                faucet: false,
                readOnly: true
            }
        }
    },

    // Update intervals (in milliseconds)
    updateIntervals: {
        blockUpdates: 3000,      // 3 seconds
        metricsUpdates: 5000,    // 5 seconds
        logsUpdates: 2000,       // 2 seconds
        eventsUpdates: 3000      // 3 seconds
    },

    // Test data for quick testing
    testData: {
        wallets: [
            {
                name: 'Test Wallet 1',
                address: 'paw1testaddress1xxxxxxxxxxxxxxxxxx',
                mnemonic: 'test mnemonic phrase for demonstration purposes only not real keys'
            },
            {
                name: 'Test Wallet 2',
                address: 'paw1testaddress2xxxxxxxxxxxxxxxxxx',
                mnemonic: 'another test mnemonic phrase for demonstration purposes only'
            }
        ],
        transactions: {
            amount: '1000000',
            denom: 'upaw',
            gas: '200000',
            memo: 'Test transaction from PAW Testing Control Panel'
        },
        staking: {
            amount: '100000000',
            denom: 'upaw'
        },
        governance: {
            deposit: '10000000upaw',
            votingPeriod: '172800s'
        },
        dex: {
            slippage: '0.5',
            deadline: '600'
        }
    },

    // UI settings
    ui: {
        logsMaxEntries: 100,
        eventsMaxEntries: 50,
        tablePageSize: 10,
        autoRefresh: true,
        theme: 'light'
    },

    // API endpoints
    api: {
        blocks: '/cosmos/base/tendermint/v1beta1/blocks/latest',
        transactions: '/cosmos/tx/v1beta1/txs',
        validators: '/cosmos/staking/v1beta1/validators',
        proposals: '/cosmos/gov/v1beta1/proposals',
        balance: '/cosmos/bank/v1beta1/balances',
        pools: '/paw/dex/v1/pools'
    }
};

export default CONFIG;
