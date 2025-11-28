// API Client Tests
import { describe, test, expect, beforeEach, jest } from '@jest/globals';

// Mock fetch
global.fetch = jest.fn();

// API Client
class APIClient {
    constructor(network = 'testnet') {
        this.network = network;
        this.customEndpoint = null;
        this.endpoints = {
            local: 'http://localhost:1317',
            testnet: 'https://testnet-api.paw.zone',
            mainnet: 'https://api.paw.zone'
        };
    }

    getEndpoint() {
        return this.customEndpoint || this.endpoints[this.network];
    }

    setNetwork(network) {
        if (!this.endpoints[network]) {
            throw new Error(`Invalid network: ${network}`);
        }
        this.network = network;
        this.customEndpoint = null;
    }

    setCustomEndpoint(endpoint) {
        this.customEndpoint = endpoint;
    }

    async request(path, options = {}) {
        const endpoint = this.getEndpoint();
        const url = `${endpoint}${path}`;

        const response = await fetch(url, {
            ...options,
            headers: {
                'Content-Type': 'application/json',
                ...options.headers
            }
        });

        if (!response.ok) {
            throw new Error(`API request failed: ${response.status}`);
        }

        return await response.json();
    }

    async getBalance(address) {
        return await this.request(`/cosmos/bank/v1beta1/balances/${address}`);
    }

    async getValidators() {
        return await this.request('/cosmos/staking/v1beta1/validators');
    }
}

describe('APIClient', () => {
    let client;

    beforeEach(() => {
        client = new APIClient('testnet');
        fetch.mockClear();
    });

    describe('Network Management', () => {
        test('should initialize with testnet', () => {
            expect(client.network).toBe('testnet');
            expect(client.getEndpoint()).toBe('https://testnet-api.paw.zone');
        });

        test('should switch to mainnet', () => {
            client.setNetwork('mainnet');
            expect(client.network).toBe('mainnet');
            expect(client.getEndpoint()).toBe('https://api.paw.zone');
        });

        test('should switch to local', () => {
            client.setNetwork('local');
            expect(client.network).toBe('local');
            expect(client.getEndpoint()).toBe('http://localhost:1317');
        });

        test('should reject invalid network', () => {
            expect(() => {
                client.setNetwork('invalid');
            }).toThrow('Invalid network: invalid');
        });

        test('should set custom endpoint', () => {
            client.setCustomEndpoint('https://custom.api.com');
            expect(client.getEndpoint()).toBe('https://custom.api.com');
        });

        test('should clear custom endpoint when switching network', () => {
            client.setCustomEndpoint('https://custom.api.com');
            client.setNetwork('mainnet');
            expect(client.getEndpoint()).toBe('https://api.paw.zone');
        });
    });

    describe('API Requests', () => {
        test('should make GET request', async () => {
            const mockResponse = { balances: [] };
            fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => mockResponse
            });

            const result = await client.getBalance('paw1test');

            expect(fetch).toHaveBeenCalledWith(
                'https://testnet-api.paw.zone/cosmos/bank/v1beta1/balances/paw1test',
                expect.objectContaining({
                    headers: expect.objectContaining({
                        'Content-Type': 'application/json'
                    })
                })
            );
            expect(result).toEqual(mockResponse);
        });

        test('should handle API errors', async () => {
            fetch.mockResolvedValueOnce({
                ok: false,
                status: 404,
                text: async () => 'Not found'
            });

            await expect(client.getBalance('invalid')).rejects.toThrow('API request failed: 404');
        });

        test('should handle network errors', async () => {
            fetch.mockRejectedValueOnce(new Error('Network error'));

            await expect(client.getBalance('paw1test')).rejects.toThrow('Network error');
        });
    });

    describe('Bank Module Queries', () => {
        test('should query account balance', async () => {
            const mockBalance = {
                balances: [{ denom: 'upaw', amount: '1000000' }]
            };

            fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => mockBalance
            });

            const result = await client.getBalance('paw1test');
            expect(result).toEqual(mockBalance);
        });
    });

    describe('Staking Module Queries', () => {
        test('should query validators', async () => {
            const mockValidators = {
                validators: [
                    { operator_address: 'pawvaloper1...', status: 'BOND_STATUS_BONDED' }
                ]
            };

            fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => mockValidators
            });

            const result = await client.getValidators();
            expect(result).toEqual(mockValidators);
        });
    });

    describe('Request Headers', () => {
        test('should include Content-Type header', async () => {
            fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => ({})
            });

            await client.request('/test');

            expect(fetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        'Content-Type': 'application/json'
                    })
                })
            );
        });

        test('should allow custom headers', async () => {
            fetch.mockResolvedValueOnce({
                ok: true,
                json: async () => ({})
            });

            await client.request('/test', {
                headers: { 'X-Custom-Header': 'value' }
            });

            expect(fetch).toHaveBeenCalledWith(
                expect.any(String),
                expect.objectContaining({
                    headers: expect.objectContaining({
                        'Content-Type': 'application/json',
                        'X-Custom-Header': 'value'
                    })
                })
            );
        });
    });
});
