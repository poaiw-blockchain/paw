// Test Setup
import { jest } from '@jest/globals';

// Setup DOM environment
if (typeof document === 'undefined') {
    global.document = {
        getElementById: jest.fn(() => document.createElement('div')),
        createElement: jest.fn(() => ({})),
        querySelectorAll: jest.fn(() => []),
        addEventListener: jest.fn()
    };
}

// Mock localStorage
global.localStorage = {
    getItem: jest.fn(),
    setItem: jest.fn(),
    removeItem: jest.fn(),
    clear: jest.fn()
};

// Mock fetch
global.fetch = jest.fn();

// Mock window
global.window = {
    location: {
        origin: 'http://localhost',
        pathname: '/'
    },
    keplr: null
};

// Mock Monaco editor
global.monaco = {
    editor: {
        create: jest.fn(),
        setModelLanguage: jest.fn(),
        setTheme: jest.fn()
    }
};

// Mock highlight.js
global.hljs = {
    highlightElement: jest.fn()
};

// Suppress console errors in tests
global.console = {
    ...console,
    error: jest.fn(),
    warn: jest.fn()
};
