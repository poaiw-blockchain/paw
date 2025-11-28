module.exports = {
    testEnvironment: 'jsdom',
    coverageDirectory: 'coverage',
    collectCoverageFrom: [
        '**/*.js',
        '!tests/**',
        '!coverage/**',
        '!node_modules/**',
        '!jest.config.js'
    ],
    testMatch: [
        '**/tests/**/*.test.js'
    ],
    moduleFileExtensions: ['js'],
    coverageThreshold: {
        global: {
            branches: 70,
            functions: 70,
            lines: 70,
            statements: 70
        }
    },
    setupFilesAfterEnv: ['<rootDir>/tests/setup.js'],
    testTimeout: 10000
};
