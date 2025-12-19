module.exports = {
  env: {
    browser: true,
    es2021: true,
    webextensions: true,
    commonjs: true,
  },
  extends: ['eslint:recommended', 'prettier'],
  parserOptions: {
    ecmaVersion: 2021,
    sourceType: 'module',
  },
  plugins: [],
  ignorePatterns: ['dist/**', 'node_modules/**'],
  rules: {
    'no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
    'no-console': 'off',
  },
}
