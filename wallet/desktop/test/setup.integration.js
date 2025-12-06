// Integration test setup
process.env.NODE_ENV = 'test';

const { TextEncoder, TextDecoder } = require('util');
const { webcrypto } = require('crypto');

global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;
global.crypto = webcrypto;

const storeData = new Map();

const electronStoreMock = {
  get: jest.fn(async (key) => storeData.has(key) ? storeData.get(key) : null),
  set: jest.fn(async (key, value) => {
    storeData.set(key, value);
  }),
  delete: jest.fn(async (key) => {
    storeData.delete(key);
  }),
  clear: jest.fn(async () => {
    storeData.clear();
  })
};

// Mock window + electron bridge
global.window = {
  crypto: webcrypto,
  electron: {
    store: electronStoreMock,
    dialog: {
      showOpenDialog: jest.fn(() => Promise.resolve({ canceled: true })),
      showSaveDialog: jest.fn(() => Promise.resolve({ canceled: true })),
      showMessageBox: jest.fn(() => Promise.resolve({ response: 0 }))
    },
    app: {
      getVersion: jest.fn(() => Promise.resolve('1.0.0')),
      getPath: jest.fn(() => Promise.resolve('/tmp'))
    },
    onMenuAction: jest.fn(),
    removeMenuActionListener: jest.fn()
  }
};

// Mock localStorage
const localStorageStore = new Map();
global.localStorage = {
  getItem: jest.fn((key) => localStorageStore.has(key) ? localStorageStore.get(key) : null),
  setItem: jest.fn((key, value) => {
    localStorageStore.set(key, value);
  }),
  removeItem: jest.fn((key) => {
    localStorageStore.delete(key);
  }),
  clear: jest.fn(() => {
    localStorageStore.clear();
  })
};

jest.setTimeout(30000);
