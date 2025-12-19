const baseConfig = require('./jest.config');

module.exports = {
  ...baseConfig,
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/test/setup.e2e.js'],
  testMatch: [
    '<rootDir>/test/e2e/**/*.test.(js|jsx|ts|tsx)'
  ],
  testTimeout: 60000,
  collectCoverageFrom: undefined,
  coverageDirectory: undefined,
};
