import '@testing-library/jest-dom';
import { TextEncoder, TextDecoder } from 'util';
import { webcrypto } from 'crypto';

// Polyfill TextEncoder/TextDecoder for Node.js
global.TextEncoder = TextEncoder;
global.TextDecoder = TextDecoder;
global.crypto = webcrypto;

// Mock electron APIs
global.window = global.window || {};
global.window.crypto = webcrypto;
global.window.electron = {
  store: {
    get: jest.fn(() => Promise.resolve(null)),
    set: jest.fn(() => Promise.resolve()),
    delete: jest.fn(() => Promise.resolve()),
    clear: jest.fn(() => Promise.resolve())
  },
  dialog: {
    showOpenDialog: jest.fn(),
    showSaveDialog: jest.fn(),
    showMessageBox: jest.fn()
  },
  app: {
    getVersion: jest.fn(() => Promise.resolve('1.0.0')),
    getPath: jest.fn()
  },
  onMenuAction: jest.fn(),
  removeMenuActionListener: jest.fn()
};

// Mock localStorage
const localStorageMock = {
  getItem: jest.fn(),
  setItem: jest.fn(),
  removeItem: jest.fn(),
  clear: jest.fn()
};
global.localStorage = localStorageMock;

// Mock navigator.clipboard
Object.defineProperty(navigator, 'clipboard', {
  value: {
    writeText: jest.fn(() => Promise.resolve())
  },
  writable: true
});

global.atob =
  global.atob ||
  function atobPolyfill(data) {
    return Buffer.from(data, 'base64').toString('binary');
  };

global.btoa =
  global.btoa ||
  function btoaPolyfill(data) {
    return Buffer.from(data, 'binary').toString('base64');
  };
