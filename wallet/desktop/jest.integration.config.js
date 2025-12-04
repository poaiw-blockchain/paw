module.exports = {
  testEnvironment: 'node',
  setupFilesAfterEnv: ['<rootDir>/test/setup.integration.js'],
  testMatch: [
    '<rootDir>/test/integration/**/*.test.js'
  ],
  testTimeout: 30000
};
