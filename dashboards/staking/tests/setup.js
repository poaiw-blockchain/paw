// Test setup file

// Mock window.keplr
global.window = global.window || {};
global.window.keplr = {
  enable: jest.fn(),
  getOfflineSigner: jest.fn(() => ({
    getAccounts: jest.fn(() => Promise.resolve([
      { address: 'paw1test123' }
    ]))
  })),
  experimentalSuggestChain: jest.fn()
};

// Mock fetch
global.fetch = jest.fn(() =>
  Promise.resolve({
    ok: true,
    json: () => Promise.resolve({
      validators: [],
      pool: { pool: { bonded_tokens: '1000000000' } },
      params: { params: { inflation_rate_change: '0.13' } },
      balances: [{ denom: 'upaw', amount: '1000000000' }],
      delegation_responses: [],
      unbonding_responses: [],
      total: [],
      rewards: []
    })
  })
);

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn(),
};
global.localStorage = localStorageMock;

// Mock console methods to reduce noise in tests
global.console = {
  ...console,
  error: jest.fn(),
  warn: jest.fn(),
};
