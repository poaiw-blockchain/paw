module.exports = {
  preset: 'react-native',
  testEnvironment: 'jsdom',
  transformIgnorePatterns: [
    'node_modules/(?!(@react-native|react-native|react-native-ble-plx|@react-native-async-storage)/)',
  ],
};
