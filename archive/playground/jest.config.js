export default {
    testEnvironment: 'jsdom',
    transform: {
        '^.+\\.js$': 'babel-jest',
    },
    moduleFileExtensions: ['js', 'json'],
    testMatch: [
        '**/tests/**/*.test.js'
    ],
    collectCoverageFrom: [
        'components/**/*.js',
        'services/**/*.js',
        'app.js',
        '!**/node_modules/**',
        '!**/tests/**'
    ],
    setupFilesAfterEnv: ['<rootDir>/tests/setup.js'],
    verbose: true
};
