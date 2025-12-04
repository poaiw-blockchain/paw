module.exports = {
  testEnvironment: 'node',
  setupFilesAfterEnv: ['<rootDir>/test/setup.e2e.js'],
  testMatch: [
    '<rootDir>/test/e2e/**/*.test.js'
  ],
  testTimeout: 60000
};
