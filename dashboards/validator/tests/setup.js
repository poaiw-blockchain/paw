// Test setup file for Jest

// Mock console methods to reduce test output noise
global.console = {
    ...console,
    log: jest.fn(),
    debug: jest.fn(),
    info: jest.fn(),
    warn: jest.fn(),
    error: jest.fn()
};

// Mock fetch for API calls
global.fetch = jest.fn();

// Mock WebSocket
global.WebSocket = jest.fn().mockImplementation(() => ({
    send: jest.fn(),
    close: jest.fn(),
    addEventListener: jest.fn(),
    removeEventListener: jest.fn(),
    readyState: 1
}));

// Mock localStorage
const localStorageMock = {
    data: {},
    getItem: function(key) {
        return this.data[key] || null;
    },
    setItem: function(key, value) {
        this.data[key] = value;
    },
    removeItem: function(key) {
        delete this.data[key];
    },
    clear: function() {
        this.data = {};
    }
};

global.localStorage = localStorageMock;

// Mock document if not available
if (typeof document === 'undefined') {
    global.document = {
        getElementById: jest.fn(),
        querySelector: jest.fn(),
        querySelectorAll: jest.fn(() => []),
        createElement: jest.fn(() => ({
            textContent: '',
            innerHTML: '',
            appendChild: jest.fn(),
            addEventListener: jest.fn()
        })),
        addEventListener: jest.fn(),
        body: {
            innerHTML: '',
            appendChild: jest.fn()
        }
    };
}

// Reset mocks before each test
beforeEach(() => {
    jest.clearAllMocks();
    localStorage.clear();
    if (global.fetch.mockClear) {
        global.fetch.mockClear();
    }
});

// Clean up after each test
afterEach(() => {
    jest.restoreAllMocks();
});
