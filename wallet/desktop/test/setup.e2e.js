require('@testing-library/jest-dom');

process.env.NODE_ENV = 'test';

const electronStore = new Map();
const clipboardWriteMock = jest.fn().mockResolvedValue(undefined);

Object.defineProperty(navigator, 'clipboard', {
  value: {
    writeText: clipboardWriteMock,
  },
  configurable: true,
});

beforeEach(() => {
  clipboardWriteMock.mockClear();
  electronStore.clear();

  window.__menuAction = undefined;
  window.electron = {
    store: {
      get: jest.fn(async (key) => electronStore.get(key)),
      set: jest.fn(async (key, value) => {
        electronStore.set(key, value);
      }),
      delete: jest.fn(async (key) => {
        electronStore.delete(key);
      }),
    },
    dialog: {
      showMessageBox: jest.fn().mockResolvedValue({ response: 0 }),
    },
    app: {
      getVersion: jest.fn().mockResolvedValue('1.0.0-test'),
    },
    onMenuAction: jest.fn((callback) => {
      window.__menuAction = callback;
    }),
    removeMenuActionListener: jest.fn(() => {
      window.__menuAction = undefined;
    }),
  };
});

global.triggerMenuAction = (action) => {
  if (typeof window.__menuAction === 'function') {
    window.__menuAction(action);
  }
};
