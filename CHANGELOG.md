# Changelog

All notable changes to the PAW blockchain project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- Corrected keeper method signatures in DEX module tests
- Updated all SDK Int types to use cosmossdk.io/math.Int
- Fixed variable shadowing in app initialization tests
- Resolved GetPool return value handling in keeper tests
- Replaced deprecated bip39 API calls
- Fixed database and logger compatibility in test suites

### Changed
- Disabled incomplete test suites pending refactoring
- Updated test files to match current keeper API patterns
- Improved code formatting with gofmt and goimports
- Applied prettier formatting to documentation files

### Added
- Comprehensive pre-commit and pre-push hooks
- Circuit breaker configuration tests
- Property-based tests for DEX invariants

## [0.1.0] - 2025-01-16

### Added
- Initial blockchain implementation
- Native DEX with AMM functionality
- Compute aggregation module
- Oracle price feed module
- Multi-device wallet support
- Security audit tests
- Benchmark suite
- E2E test framework

[Unreleased]: https://github.com/decristofaroj/paw/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/decristofaroj/paw/releases/tag/v0.1.0
