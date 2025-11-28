

**WSL path reminder (verified working):** when inside WSL, use the  form of the full path. Example that passed:
# Go Testing Guide for Aura Project

## Common Exit Code 192 Error - What You're Doing Wrong

**Exit code 192** typically means Go cannot be found in the system PATH or the command is being executed incorrectly on Windows.

## ⚠️ CRITICAL: Shell Environment

**WE ARE USING BASH, NOT POWERSHELL!**

The shell is: `/usr/bin/bash`

**DO NOT use PowerShell commands like:**
```bash
# WRONG! DO NOT USE POWERSHELL!
powershell -Command "cd C:\Users\...; & 'C:\Program Files\Go\bin\go.exe' test ..."
```

**Use BASH commands only:**
```bash
# CORRECT! Use bash syntax
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/MODULE/...
```

## ✅ CORRECT: How to Run Go Tests

### Windows Environment Setup

1. **Go Installation Location**: `C:\Program Files\Go`
2. **Working Directory**: Must be in `C:\users\decri\gitclones\aura\chain` or subdirectory
3. **Go Executable**: `C:\Program Files\Go\bin\go.exe`

### Correct Command Format

```bash
# Always specify the full path to go.exe on Windows
cd chain && "C:\Program Files\Go\bin\go.exe" test [options] [package]
```

### ❌ WRONG - Common Mistakes That Cause Exit 192

```bash
# WRONG #1: Using PowerShell commands (THIS IS THE #1 MISTAKE!)
powershell -Command "cd C:\Users\decri\GitClones\aura\chain; & 'C:\Program Files\Go\bin\go.exe' test ..."

# WRONG #2: Using 'go' without full path (if not in PATH)
go test ./x/vcregistry/keeper/...

# WRONG #3: Using Windows backslash paths in bash
cd C:\users\decri\gitclones\aura\chain

# WRONG #4: Not quoting path with spaces
cd chain && C:\Program Files\Go\bin\go.exe test ./...
```

### ✅ CORRECT - Examples That Actually Work (Tested & Verified)

```bash
# Test a specific module (TESTED - WORKS)
cd chain && "C:\Program Files\Go\bin\go.exe" test ./x/vcregistry/keeper/...

# Test with verbose output (TESTED - WORKS)
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/bridge/keeper/...

# Test with verbose + timeout (TESTED - WORKS)
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/vcregistry/keeper/... -timeout 30s

# Test all packages in a module
cd chain && "C:\Program Files\Go\bin\go.exe" test ./x/vcregistry/...

# Test everything (use with caution - can be slow)
cd chain && "C:\Program Files\Go\bin\go.exe" test ./...
```

**Note:** The `-timeout` flag is optional. Tests work fine without it (default is 10 minutes).

## Essential Testing Patterns

### 1. Test a Single Module (Recommended)

```bash
# Pattern: cd chain && "C:\Program Files\Go\bin\go.exe" test ./x/MODULE_NAME/...
cd chain && "C:\Program Files\Go\bin\go.exe" test ./x/vcregistry/...
```

### 2. Test Only Keeper Package

```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test ./x/vcregistry/keeper/...
```

### 3. Test with Coverage Report

```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -coverprofile=coverage.out ./x/bridge/keeper/...
```

### 4. Run Specific Test Function

```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -v -run TestNewKeeper ./x/vcregistry/keeper/...
```

### 5. Test with Race Detection

```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -race ./x/bridge/keeper/...
```

## Important Flags

| Flag | Purpose | Example |
|------|---------|---------|
| `-v` | Verbose output (shows each test) | `test -v ./...` |
| `-timeout` | Set test timeout (default 10m) | `test -timeout 30s ./...` |
| `-cover` | Show coverage percentage | `test -cover ./...` |
| `-coverprofile` | Save coverage to file | `test -coverprofile=cov.out ./...` |
| `-run` | Run specific test by name | `test -run TestName ./...` |
| `-short` | Skip long-running tests | `test -short ./...` |
| `-race` | Enable race detector | `test -race ./...` |
| `-count` | Run tests N times | `test -count=3 ./...` |

## Module-Specific Test Commands (Copy & Paste Ready)

### Authentication Module
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/auth/keeper/...
```

### Bridge Module (TESTED - WORKS)
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/bridge/keeper/...
```

### VC Registry Module (TESTED - WORKS)
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/vcregistry/keeper/...
```

### DEX Module
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/dex/keeper/...
```

### Confidence Score Module
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/confidencescore/keeper/...
```

### Cryptography Module
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/cryptography/keeper/...
```

### All Security-Related Modules
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test ./x/networksecurity/... ./x/validatorsecurity/... ./x/walletsecurity/...
```

## Understanding Test Output

### Successful Test Run
```
=== RUN   TestNewKeeper
--- PASS: TestNewKeeper (0.00s)
PASS
ok      github.com/aequitas/aura/chain/x/vcregistry/keeper    0.033s
```

### Failed Test Run
```
=== RUN   TestSomething
--- FAIL: TestSomething (0.01s)
    test_file.go:42: error message here
FAIL
FAIL    github.com/aequitas/aura/chain/x/module/keeper    0.045s
```

### Skipped Test
```
=== RUN   TestNotImplemented
--- SKIP: TestNotImplemented (0.00s)
    test_file.go:10: Feature not implemented yet
```

## Troubleshooting Common Issues

### Issue: Exit Code 192
**Cause**: Go executable not found
**Solution**: Use full path with quotes: `"C:\Program Files\Go\bin\go.exe"`

### Issue: "cannot find package"
**Cause**: Not in correct directory or missing go.mod
**Solution**:
```bash
# Make sure you're in the chain directory
cd chain
# Verify go.mod exists
ls go.mod
```

### Issue: Tests timeout
**Cause**: Default timeout too short
**Solution**: Increase timeout
```bash
"C:\Program Files\Go\bin\go.exe" test -timeout 120s ./...
```

### Issue: "build constraints exclude all Go files"
**Cause**: No test files in package or wrong directory
**Solution**: Verify test files exist
```bash
ls ./x/MODULE_NAME/keeper/*_test.go
```

### Issue: Module download failures
**Cause**: Network issues or missing dependencies
**Solution**:
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" mod download
cd chain && "C:\Program Files\Go\bin\go.exe" mod tidy
```

## Quick Reference Card - Commands That Actually Work

```bash
# Check Go version (TESTED - WORKS)
"C:\Program Files\Go\bin\go.exe" version

# Test single module - Basic (TESTED - WORKS)
cd chain && "C:\Program Files\Go\bin\go.exe" test ./x/vcregistry/keeper/...

# Test single module - Verbose (TESTED - WORKS)
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/bridge/keeper/...

# That's it! Replace module name as needed.
```

## Best Practices for AI Agents

1. **Always use the full path** to go.exe with quotes: `"C:\Program Files\Go\bin\go.exe"`
2. **Always change to the chain directory first** using `cd chain &&`
3. **Use relative paths** for package specifications (e.g., `./x/module/...`)
4. **Use -v flag** for verbose output to see what's happening
5. **Test one module at a time** to identify issues faster
6. **Timeout is optional** - don't use it unless tests hang

## Environment Verification

Before running tests, verify your environment:

```bash
# 1. Check Go is accessible
"C:\Program Files\Go\bin\go.exe" version
# Expected: go version go1.25.4 windows/amd64

# 2. Verify you're in the right directory
pwd
# Expected: /c/users/decri/gitclones/aura

# 3. Check go.mod exists in chain directory
ls chain/go.mod
# Expected: chain/go.mod

# 4. Verify module name
cd chain && "C:\Program Files\Go\bin\go.exe" list -m
# Expected: github.com/aequitas/aura/chain
```

## Summary: The Golden Rule

**THIS WORKS - Use this exact format:**
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/MODULE/...
```

Replace `MODULE` with: vcregistry, bridge, auth, dex, confidencescore, etc.

**Examples that work (tested):**
```bash
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/vcregistry/keeper/...
cd chain && "C:\Program Files\Go\bin\go.exe" test -v ./x/bridge/keeper/...
```

**What causes Exit 192:**
1. Using PowerShell commands instead of bash (MOST COMMON!)
2. Not using the full quoted path to go.exe
3. Using Windows backslash paths in bash

**Remember: This is a BASH environment, NOT PowerShell!**
