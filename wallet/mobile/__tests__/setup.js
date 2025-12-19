/**
 * Test setup file
 * Configures Jest and mocks for React Native components
 */

import 'react-native-gesture-handler/jestSetup';

// Mock react-native-keychain
jest.mock('react-native-keychain', () => ({
  setGenericPassword: jest.fn(() => Promise.resolve(true)),
  getGenericPassword: jest.fn(() =>
    Promise.resolve({
      username: 'paw_wallet',
      password: JSON.stringify({
        privateKey: 'encrypted_private_key',
        mnemonic: 'encrypted_mnemonic',
      }),
    }),
  ),
  resetGenericPassword: jest.fn(() => Promise.resolve(true)),
  ACCESSIBLE: {
    WHEN_UNLOCKED_THIS_DEVICE_ONLY: 'WhenUnlockedThisDeviceOnly',
  },
}));

// Mock AsyncStorage
jest.mock('@react-native-async-storage/async-storage', () => ({
  setItem: jest.fn(() => Promise.resolve()),
  getItem: jest.fn(() => Promise.resolve(null)),
  removeItem: jest.fn(() => Promise.resolve()),
  clear: jest.fn(() => Promise.resolve()),
}));

// Mock react-native-biometrics
jest.mock('react-native-biometrics', () => ({
  __esModule: true,
  default: jest.fn().mockImplementation(() => ({
    isSensorAvailable: jest.fn(() =>
      Promise.resolve({available: true, biometryType: 'TouchID'}),
    ),
    simplePrompt: jest.fn(() => Promise.resolve({success: true})),
    createKeys: jest.fn(() => Promise.resolve({publicKey: 'mock_public_key'})),
    deleteKeys: jest.fn(() => Promise.resolve({keysDeleted: true})),
    biometricKeysExist: jest.fn(() => Promise.resolve({keysExist: true})),
  })),
  TouchID: 'TouchID',
  FaceID: 'FaceID',
  Biometrics: 'Biometrics',
}));

// Mock react-native-vector-icons
jest.mock('react-native-vector-icons/MaterialIcons', () => 'Icon');

// Mock react-native-qrcode-svg
jest.mock('react-native-qrcode-svg', () => 'QRCode');

// Mock axios
jest.mock('axios', () => ({
  create: jest.fn(() => ({
    get: jest.fn(),
    post: jest.fn(),
    defaults: {
      baseURL: 'http://localhost:1317',
    },
  })),
}));

// Mock react-native-push-notification
jest.mock('react-native-push-notification', () => ({
  createChannel: jest.fn((_, cb) => cb(true)),
  requestPermissions: jest.fn(() =>
    Promise.resolve({alert: true, authorizationStatus: 2}),
  ),
  localNotification: jest.fn(),
  cancelAllLocalNotifications: jest.fn(),
  removeAllDeliveredNotifications: jest.fn(),
  checkPermissions: jest.fn(cb => cb({alert: true, sound: true})),
  abandonPermissions: jest.fn(() => Promise.resolve()),
}));

// Silence console warnings in tests
global.console = {
  ...console,
  warn: jest.fn(),
  error: jest.fn(),
};
