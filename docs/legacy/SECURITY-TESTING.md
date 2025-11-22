# PAW Blockchain Security Testing Guide

## Security Testing Philosophy

### Why 100% Coverage Matters for Security

Security testing is fundamentally different from functional testing. While functional tests verify "does it work?", security tests ask "can it break?" and "can someone attack it?"

For security-critical code, even a single untested line could be an exploitable vulnerability:

- **Validation logic**: An untested edge case could accept malicious input
- **Authorization checks**: An untested branch could allow unauthorized access
- **State transitions**: An untested path could corrupt blockchain state
- **Cryptographic operations**: An untested fallback could compromise keys
- **Economic incentives**: An untested edge case could enable theft

**100% coverage for security code is not optional—it's mandatory.**

### Security Testing Principles

1. **Assume attacker controls inputs** - Test all possible input combinations
2. **Test negative cases first** - What could go wrong?
3. **Verify invariants** - What should always be true?
4. **Test boundaries** - What happens at limits?
5. **Check error paths** - Are errors handled securely?
6. **Validate all paths** - Can any code path be skipped?

---

## Critical Security Modules

### Module Coverage Requirements

| Module | Type | Coverage Required | Reason |
|--------|------|-------------------|--------|
| x/oracle/keeper/validation | Critical | 100% | Oracle price feeds affect all DEX operations |
| x/oracle/keeper/price.go | Critical | 100% | Incorrect pricing = theft risk |
| x/dex/keeper/swap.go | Critical | 100% | Swap logic = economic correctness |
| x/dex/keeper/circuit_breaker | Critical | 100% | Emergency safety mechanism |
| x/dex/types/validation.go | Critical | 100% | Input validation = attack prevention |
| x/compute/keeper/task_execution | Critical | 100% | Task execution = resource isolation |
| api/validation.go | High | 95% | API input validation |
| api/middleware.go | High | 95% | Authentication and authorization |
| p2p/reputation/manager.go | High | 95% | Network security and peer trust |

### Why These Modules Are Critical

**x/oracle/keeper/validation**
- Prices feed all DEX operations
- Invalid prices could authorize theft
- Must validate: sources, staleness, deviation bounds
- **Test every validation rule and every error case**

**x/oracle/keeper/price.go**
- Calculates median of price submissions
- Must handle: duplicates, outliers, edge cases
- Incorrect calculation = unfair trading
- **Test with adversarial price feeds**

**x/dex/keeper/swap.go**
- Executes all token exchanges
- Must prevent: slippage abuse, MEV extraction, flash loans
- Incorrect calculation = theft of liquidity
- **Test with extreme amounts, low liquidity, edge ratios**

**x/dex/keeper/circuit_breaker**
- Emergency halt mechanism
- Must trigger on: abnormal trading, price crashes
- Failure to trigger = catastrophic losses
- **Test all trigger conditions and their interactions**

**x/dex/types/validation.go**
- First line of defense against malformed messages
- Must reject: invalid amounts, addresses, ratios
- Bypass = potential for exploits
- **Test every validation rule with malicious inputs**

**x/compute/keeper/task_execution**
- Executes user-supplied logic
- Must isolate: compute resources, storage access
- Failure = system compromise
- **Test resource limits and access controls**

---

## Attack Scenarios to Test

### Oracle Attack Scenarios

#### 1. Price Feed Manipulation

```go
@pytest.mark.security
func TestOraclePriceFeedManipulation(t *testing.T) {
    tests := []struct {
        name        string
        prices      []math.Int
        shouldPass  bool
        description string
    }{
        {
            name:       "legitimate prices",
            prices:     []math.Int{math.NewInt(100), math.NewInt(101), math.NewInt(99)},
            shouldPass: true,
            description: "Normal market conditions",
        },
        {
            name:       "single outlier",
            prices:     []math.Int{math.NewInt(100), math.NewInt(101), math.NewInt(1000)},
            shouldPass: true,
            description: "One bad feed should be filtered",
        },
        {
            name:       "two extreme outliers",
            prices:     []math.Int{math.NewInt(1), math.NewInt(100000), math.NewInt(100)},
            shouldPass: false,
            description: "Multiple outliers should trigger error",
        },
        {
            name:       "zero price",
            prices:     []math.Int{math.NewInt(0), math.NewInt(100), math.NewInt(101)},
            shouldPass: false,
            description: "Zero price is invalid",
        },
        {
            name:       "negative price",
            prices:     []math.Int{math.NewInt(-100), math.NewInt(100), math.NewInt(101)},
            shouldPass: false,
            description: "Negative prices are impossible",
        },
        {
            name:       "all same price",
            prices:     []math.Int{math.NewInt(100), math.NewInt(100), math.NewInt(100)},
            shouldPass: true,
            description: "Unanimous pricing is valid",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            app := setupTestApp()
            ctx := app.BaseApp.NewContext(false)

            // Submit prices
            for i, price := range tt.prices {
                msg := &types.MsgSubmitPrice{
                    Feeder: setupFeeder(app, i),
                    Price: price,
                    Token: "AAPL",
                }
                app.OracleKeeper.SubmitPrice(ctx, msg)
            }

            // Get aggregated price
            price, err := app.OracleKeeper.GetMedianPrice(ctx, "AAPL")

            if tt.shouldPass {
                require.NoError(t, err, tt.description)
                require.NotNil(t, price)
            } else {
                require.Error(t, err, tt.description)
            }
        })
    }
}
```

#### 2. Stale Price Attack

```go
@pytest.mark.security
func TestStaleOraclePrice(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Submit price with timestamp
    msg := &types.MsgSubmitPrice{
        Feeder: setupFeeder(app, 0),
        Price: math.NewInt(100),
        Timestamp: ctx.BlockTime(),
    }
    app.OracleKeeper.SubmitPrice(ctx, msg)

    // Fast forward 1 hour
    newCtx := ctx.WithBlockTime(ctx.BlockTime().Add(time.Hour))

    // Try to use stale price in swap
    swapMsg := &dextypes.MsgSwap{
        Trader: "paw1user",
        TokenIn: sdk.NewCoin("AAPL", math.NewInt(100)),
    }

    // Should reject due to staleness
    _, err := app.DEXKeeper.Swap(newCtx, swapMsg)
    require.Error(t, err)
    require.Contains(t, err.Error(), "stale")
}
```

#### 3. Price Deviation Attack

```go
@pytest.mark.security
func TestPriceDeviationBounds(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Establish baseline price
    for i := 0; i < 3; i++ {
        msg := &types.MsgSubmitPrice{
            Feeder: setupFeeder(app, i),
            Price: math.NewInt(100),
        }
        app.OracleKeeper.SubmitPrice(ctx, msg)
    }

    // Try to submit price with >10% deviation
    tests := []struct {
        name     string
        price    int64
        shouldErr bool
    }{
        {"101 (1% increase)", 101, false},
        {"110 (10% increase)", 110, false},
        {"115 (15% increase)", 115, true},  // Exceeds max deviation
        {"90 (10% decrease)", 90, false},
        {"85 (15% decrease)", 85, true},   // Exceeds max deviation
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            msg := &types.MsgSubmitPrice{
                Feeder: setupFeeder(app, 3),
                Price: math.NewInt(tt.price),
            }

            _, err := app.OracleKeeper.SubmitPrice(ctx, msg)
            if tt.shouldErr {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### DEX Attack Scenarios

#### 1. Flash Loan Attack

```go
@pytest.mark.security
func TestFlashLoanAttack(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Create liquidity pool
    setupPool(app, "USDC", "USDT", math.NewInt(1000000), math.NewInt(1000000))

    // Attempt flash loan attack:
    // 1. Borrow huge amount at start of block
    // 2. Manipulate price with borrowed funds
    // 3. Exploit price to gain profit
    // 4. Repay loan

    flashMsg := &dextypes.MsgFlashLoan{
        Borrower: "paw1attacker",
        Amount: math.NewInt(10000000),  // Huge amount
        Token: "USDC",
    }

    _, err := app.DEXKeeper.ExecuteFlashLoan(ctx, flashMsg)

    // Flash loans should require guarantee/fee for profit extraction
    require.Error(t, err)
    require.Contains(t, err.Error(), "insufficient fee")
}
```

#### 2. Slippage Attack

```go
@pytest.mark.security
func TestSlippageProtection(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Create imbalanced pool
    setupPool(app, "USDC", "USDT", math.NewInt(1000000), math.NewInt(100000))

    tests := []struct {
        name         string
        amountIn     int64
        minAmountOut int64
        shouldPass   bool
    }{
        {
            name:         "reasonable slippage",
            amountIn:     1000,
            minAmountOut: 900,  // 10% slippage
            shouldPass:   true,
        },
        {
            name:         "excessive slippage",
            amountIn:     1000,
            minAmountOut: 950,  // 5% slippage, but actual is 20%
            shouldPass:   false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            swapMsg := &dextypes.MsgSwap{
                TokenIn: sdk.NewCoin("USDC", math.NewInt(tt.amountIn)),
                MinAmountOut: math.NewInt(tt.minAmountOut),
            }

            _, err := app.DEXKeeper.Swap(ctx, swapMsg)
            if tt.shouldPass {
                require.NoError(t, err)
            } else {
                require.Error(t, err)
                require.Contains(t, err.Error(), "slippage")
            }
        })
    }
}
```

#### 3. Circuit Breaker Activation

```go
@pytest.mark.security
func TestCircuitBreakerProtection(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    setupPool(app, "USDC", "USDT", math.NewInt(1000000), math.NewInt(1000000))

    tests := []struct {
        name          string
        swapAmount    int64
        shouldTrigger bool
        description   string
    }{
        {
            name:          "normal swap",
            swapAmount:    10000,
            shouldTrigger: false,
            description:   "Small swap should succeed",
        },
        {
            name:          "large swap",
            swapAmount:    100000,  // 10% of pool
            shouldTrigger: false,
            description:   "Large but within limits",
        },
        {
            name:          "extreme swap",
            swapAmount:    500000,  // 50% of pool
            shouldTrigger: true,
            description:   "Extreme swap should trigger circuit breaker",
        },
        {
            name:          "multiple swaps",
            swapAmount:    50000,   // Each small, but together breach
            shouldTrigger: true,
            description:   "Multiple swaps causing excess volume",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            swapMsg := &dextypes.MsgSwap{
                TokenIn: sdk.NewCoin("USDC", math.NewInt(tt.swapAmount)),
            }

            _, err := app.DEXKeeper.Swap(ctx, swapMsg)

            if tt.shouldTrigger {
                require.Error(t, err)
                require.Contains(t, err.Error(), "circuit breaker")
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Compute Attack Scenarios

#### 1. Resource Exhaustion

```go
@pytest.mark.security
func TestComputeResourceLimits(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    tests := []struct {
        name        string
        cpuLimit    uint64
        memLimit    uint64
        timeLimit   uint64
        shouldFail  bool
    }{
        {
            name:       "normal computation",
            cpuLimit:   1000,
            memLimit:   100000,
            timeLimit:  30,  // 30 seconds
            shouldFail: false,
        },
        {
            name:       "infinite loop attempt",
            cpuLimit:   1000,
            memLimit:   100000,
            timeLimit:  1,   // 1 second timeout
            shouldFail: true,
        },
        {
            name:       "memory bomb",
            cpuLimit:   1000,
            memLimit:   100000,
            timeLimit:  30,
            shouldFail: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            taskMsg := &types.MsgSubmitTask{
                Code: getTestCode(tt.name),
                CpuLimit: tt.cpuLimit,
                MemLimit: tt.memLimit,
                TimeLimit: tt.timeLimit,
            }

            _, err := app.ComputeKeeper.SubmitTask(ctx, taskMsg)
            if tt.shouldFail {
                require.Error(t, err)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

### Network Attack Scenarios

#### 1. Sybil Attack Resistance

```go
@pytest.mark.security
func TestSybilAttackResistance(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Attempt to create many identities
    for i := 0; i < 1000; i++ {
        addr := fmt.Sprintf("paw1attacker%d", i)

        // Each identity tries to stake
        stakeMsg := &stakingtypes.MsgDelegate{
            DelegatorAddress: addr,
            ValidatorAddress: "pawvaloper1...",
            Amount: sdk.NewCoin("upaw", math.NewInt(100)),
        }

        _, err := app.StakingKeeper.Delegate(ctx, addr, math.NewInt(100))

        // Should be limited by minimum stake or other mechanism
        // Should not allow unlimited cheap identities
    }

    // Verify attacker doesn't control governance
    votingPower := app.StakingKeeper.GetValidatorSet(ctx).GetTotalVotingPower()
    attackerPower := app.StakingKeeper.GetDelegatorValidator(ctx, "paw1attacker0")

    require.Less(t, float64(attackerPower), float64(votingPower)/3)
}
```

---

## Testing Tools

### Go Security Tools

```bash
# gosec - Static security analysis
go install github.com/securego/gosec/v2/cmd/gosec@latest
gosec ./...

# golangci-lint - Comprehensive linter with security checks
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
golangci-lint run ./... --enable-all

# nancy - Vulnerability scanner
go install github.com/sonatype-nexus-community/nancy@latest
go list -json -m all | nancy sleuth

# go-fuzz - Fuzzing for security issues
go install github.com/dvyukov/go-fuzz/go-fuzz@latest
go test -fuzz=FuzzSwapCalculation ./x/dex/keeper

# staticcheck - Advanced static analysis
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

### Python Security Tools

```bash
# bandit - Python security analysis
pip install bandit
bandit -r wallet/

# safety - Dependency vulnerability checker
pip install safety
safety check

# pip-audit - Comprehensive dependency audit
pip install pip-audit
pip-audit

# semgrep - Pattern-based security scanner
pip install semgrep
semgrep --config=p/security-audit wallet/
```

### Integrated Testing

```bash
# Run all security checks
make security-check

# Run all security tests
make test-security

# Generate security report
make security-report
```

---

## Vulnerability Categories to Cover

### 1. Input Validation

**What to test**:
- Empty inputs
- Null/nil pointers
- Out of bounds values
- Invalid character sequences
- Oversized inputs
- Type mismatches

**Example**:
```go
func TestInputValidation(t *testing.T) {
    invalidInputs := []interface{}{
        "",                    // Empty
        nil,                  // Nil
        math.NewInt(-1),      // Negative
        math.NewInt(0),       // Zero (if invalid)
        math.NewInt(1 << 63), // Overflow
        "'; DROP TABLE;",     // Injection
        "x" * 10000,          // Oversized
    }

    for _, input := range invalidInputs {
        err := Validate(input)
        require.Error(t, err, fmt.Sprintf("Should reject: %v", input))
    }
}
```

### 2. Authorization & Authentication

**What to test**:
- Missing credentials
- Invalid signatures
- Expired tokens
- Insufficient permissions
- Role escalation attempts
- Cross-user access

**Example**:
```go
func TestAuthorizationBypass(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // User A tries to access User B's wallet
    walletMsg := &types.MsgTransferFromWallet{
        From: "paw1user_b",  // Different user
        To: "paw1attacker",
    }

    // Must verify signature from User A, not B
    // Should fail
    _, err := app.WalletKeeper.TransferFromWallet(ctx, walletMsg)
    require.Error(t, err)
}
```

### 3. Cryptographic Operations

**What to test**:
- Key generation randomness
- Signature verification
- Hash collisions
- Random number quality
- Key derivation correctness
- Encryption/decryption symmetry

**Example**:
```go
func TestCryptographicSecurity(t *testing.T) {
    // Test key generation randomness
    key1 := GenerateKey()
    key2 := GenerateKey()
    require.NotEqual(t, key1, key2)

    // Test signature verification
    msg := []byte("test")
    sig := SignMessage(key1, msg)

    require.True(t, VerifySignature(key1.Public, msg, sig))
    require.False(t, VerifySignature(key2.Public, msg, sig))

    // Test tampering detection
    msg[0] ^= 0xFF
    require.False(t, VerifySignature(key1.Public, msg, sig))
}
```

### 4. State Management

**What to test**:
- Concurrent state updates
- State rollback on errors
- Invariant violations
- Race conditions
- Atomicity of operations

**Example**:
```go
func TestStateInvariants(t *testing.T) {
    app := setupTestApp()
    ctx := app.BaseApp.NewContext(false)

    // Invariant: Total supply = sum of all balances
    initialSupply := app.BankKeeper.GetSupply(ctx, "upaw")

    // Perform operations
    app.BankKeeper.SendCoins(ctx, addr1, addr2, sdk.NewCoins(coin))

    // Verify invariant
    finalSupply := app.BankKeeper.GetSupply(ctx, "upaw")
    require.Equal(t, initialSupply, finalSupply)
}
```

### 5. Economic Incentives

**What to test**:
- Fair pricing calculations
- Fee calculations
- Reward distributions
- Inflation limits
- Deflation mechanics

**Example**:
```go
func TestEconomicSecurity(t *testing.T) {
    // Test slippage calculation is fair
    swapMsg := &dextypes.MsgSwap{
        TokenIn: sdk.NewCoin("USDC", math.NewInt(1000)),
    }

    expectedOutput := CalculateSwapOutput(1000, pool)
    actualOutput := app.DEXKeeper.Swap(ctx, swapMsg)

    // Output should be deterministic and fair
    require.Equal(t, expectedOutput, actualOutput)
}
```

### 6. Denial of Service (DoS)

**What to test**:
- Resource exhaustion
- Unbounded loops
- Memory leaks
- Large message handling
- Rate limiting

**Example**:
```go
func TestDoSResistance(t *testing.T) {
    // Test handling large message
    largeMsg := &types.MsgLargePayload{
        Data: make([]byte, 100*1024*1024),  // 100MB
    }

    _, err := app.Handler.HandleMsg(ctx, largeMsg)
    // Should either reject or handle gracefully
    require.NotNil(t, err) // Should fail safely
}
```

---

## Security Test Checklist

Before committing security-critical code:

- [ ] Input validation tests for all parameters
- [ ] Negative test cases (what shouldn't work)
- [ ] Boundary condition tests
- [ ] Authorization tests for each permission level
- [ ] Cryptographic correctness tests
- [ ] State invariant verification
- [ ] Concurrency tests (race detection)
- [ ] Error handling tests (no panic, secure failure)
- [ ] Performance degradation under attack
- [ ] Fuzzing with random inputs
- [ ] Code review for common vulnerabilities
- [ ] Coverage >95% for critical paths
- [ ] Security tool scans pass (gosec, bandit, etc.)

---

## Security Testing Best Practices

### 1. Think Like an Attacker

```go
// Bad test - checks happy path
func TestSwap(t *testing.T) {
    result := Swap(1000, pool)
    require.NotNil(t, result)
}

// Good test - checks attack scenarios
func TestSwap(t *testing.T) {
    tests := []struct {
        name  string
        setup func(*App)
        input *types.MsgSwap
        check func(*testing.T, error)
    }{
        {
            name: "normal swap",
            input: &types.MsgSwap{...},
            check: func(t *testing.T, err error) {
                require.NoError(t, err)
            },
        },
        {
            name: "overflow attempt",
            input: &types.MsgSwap{Amount: math.MaxInt64},
            check: func(t *testing.T, err error) {
                require.Error(t, err)  // Should reject
            },
        },
        // ... more adversarial cases
    }
}
```

### 2. Test All Error Paths

```go
func TestErrorHandling(t *testing.T) {
    // Each error case should be tested
    tests := []struct {
        name      string
        setupErr  error
        checkErr  bool
        errorType string
    }{
        {name: "insufficient balance", checkErr: true},
        {name: "invalid address", checkErr: true},
        {name: "expired transaction", checkErr: true},
        {name: "authorization failed", checkErr: true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup to trigger specific error
            // Verify correct error is returned (not panic, not nil)
        })
    }
}
```

### 3. Use Fuzzing for Security

```go
// Fuzzing test - automatically finds edge cases
func FuzzSwapCalculation(f *testing.F) {
    f.Add(int64(1), int64(1000000), int64(1000000))

    f.Fuzz(func(t *testing.T, amountIn, reserve1, reserve2 int64) {
        if amountIn < 0 || reserve1 < 0 || reserve2 < 0 {
            t.Skip()
        }

        output := CalculateSwapOutput(
            math.NewInt(amountIn),
            math.NewInt(reserve1),
            math.NewInt(reserve2),
        )

        // Invariants that must always hold
        require.True(t, output.GTE(math.ZeroInt()))
        require.True(t, output.LTE(math.NewInt(reserve2)))
    })
}
```

---

## Continuous Security Monitoring

### Automated Scanning

```bash
# GitHub Actions security workflow
name: Security
on: [push, pull_request]

jobs:
  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Run gosec
        run: gosec ./...

      - name: Run bandit
        run: bandit -r wallet/

      - name: Check dependencies
        run: nancy sleuth

      - name: OWASP dependency check
        uses: dependency-check/Dependency-Check_Action@main
```

### Regular Security Audits

```bash
# Scheduled weekly security review
make security-audit

# Generates comprehensive report:
# - Code scanning results
# - Dependency vulnerabilities
# - Test coverage gaps
# - Performance issues
```

---

## Incident Response

If a security vulnerability is discovered:

1. **Do not disclose publicly** - Follow responsible disclosure
2. **Create private security advisory** - GitHub Security Advisory
3. **Fix vulnerability** - Create security patch
4. **Add regression test** - Prevent reoccurrence
5. **Update documentation** - Security implications
6. **Notify affected parties** - If applicable
7. **Release patch** - Follow semantic versioning

See `SECURITY.md` for vulnerability reporting process.

---

## References

- **[OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)** - Comprehensive security testing methodology
- **[CWE: Common Weakness Enumeration](https://cwe.mitre.org/)** - Security vulnerability catalog
- **[Cosmos SDK Security](https://docs.cosmos.network/main/architecture/modules/intro)** - Module security considerations
- **[Go Security Tools](https://golang.org/wiki/CodeReviewComments)** - Security in Go best practices

---

**Document Version**: 1.0
**Last Updated**: November 2024
**Classification**: Security-Critical
**Maintainer**: PAW Security Team
