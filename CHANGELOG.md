# Changelog

All notable changes to the PAW Blockchain project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- GCP 3-node testnet setup and deployment automation
- Comprehensive security hardening across all modules
- 100% test pass rate achievement
- Docker Compose configurations organized in `/compose/` directory
- Scripts reorganized into logical subdirectories

### Changed
- Archived legacy documentation and workflows
- Hardened husky pre-push checks
- Removed pawd binary from  tracking
- Moved Docker Compose files to dedicated directory
- Consolidated scripts into organized structure

### Security
- **Oracle Module**: Enhanced statistical outlier detection in `x/oracle/keeper/security.go`
  - Added comprehensive error logging for sqrt computation failures with diagnostic information
  - Added Prometheus metrics (`anomalous_patterns_detected_total`) to monitor fallback frequency
  - Added detailed security risk documentation explaining liveness vs accuracy tradeoff
  - Modified `calculateStdDev()` to accept context and asset parameters for better observability

## [2025-11-16] - Compute Module Implementation

### Added
- `x/compute/keeper/keeper_methods.go` (240 lines)
  - `RegisterProvider()` method with stake validation
  - `RequestCompute()` method with ID generation
  - `SubmitResult()` method with status updates
  - `GetProvider()` and `GetRequest()` retrieval methods
  - `SetProvider()` and `SetRequest()` storage methods
  - `GetNextRequestID()` and `SetNextRequestID()` helpers
  - Test helper functions

### Changed
- `testutil/keeper/setup.go`
  - CRITICAL FIX: Added CommitMultiStore initialization
  - Added proper genesis state generation
  - Added InitChain, FinalizeBlock, and Commit sequence
  - Changed from 31 lines to 67 lines

- `testutil/keeper/compute.go`
  - Updated RegisterTestProvider to use actual keeper method
  - Updated SubmitTestRequest to use actual keeper method
  - Removed skip statements

- `x/compute/types/tx.pb.go`
  - Added MsgRequestCompute message type (+31 lines)
  - Added MsgRequestComputeResponse (+7 lines)
  - Added MsgSubmitResult message type (+34 lines)
  - Added MsgSubmitResultResponse (+5 lines)

- `x/compute/types/compute.pb.go`
  - Added RequestStatus enum (4 values)
  - Added Provider struct (+11 lines)
  - Added ComputeRequest struct (+13 lines)

- `tests/security/auth_test.go`
  - Fixed CreatePool method calls (3 locations)
  - Fixed Swap method calls (2 locations)
  - Fixed SubmitPrice method call (1 location)
  - Fixed GetPool return value handling (2 locations)
  - Added TokenOut field to msgSwap structs (2 locations)

- `tests/security/adversarial_test.go`
  - Removed SetupTestApp skip statement

### Fixed
- Critical SetupTestApp blocker
- Security test compilation errors
- Multiple method signature mismatches

### Tests
- Overall: 83% pass rate (64 PASS / 13 FAIL / 6 SKIP)
- Passing: api, tests/property, x/dex/types, x/oracle/keeper
- Known issues: interface registry, test address validation
