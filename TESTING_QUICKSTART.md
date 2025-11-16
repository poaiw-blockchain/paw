# Testing Quick Start Guide

Quick reference for running tests in the PAW blockchain project.

## Prerequisites

```bash
# Install dependencies
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

## Quick Commands

### Run Everything

```bash
make test                 # All tests with coverage
```

### Development Workflow

```bash
make test-unit            # Fast unit tests only
make test-keeper          # Test all keepers
make lint                 # Run linter
```

### Before Committing

```bash
make test-coverage        # Generate coverage report
make lint                 # Check code quality
make format              # Format code
```

### Debugging

```bash
# Run specific test with verbose output
go test -v -run TestCreatePool ./x/dex/keeper/...

# Run with race detection
go test -race ./x/dex/...

# Run single test file
go test -v ./x/dex/types/msg_test.go
```

## Test Patterns

### Unit Test Example

```go
func TestFeature(t *testing.T) {
    k, ctx := keepertest.DexKeeper(t)

    msg := &types.MsgCreatePool{
        Creator: "paw1test",
        TokenA:  "upaw",
        TokenB:  "uusdt",
        AmountA: sdk.NewInt(1000000),
        AmountB: sdk.NewInt(2000000),
    }

    resp, err := k.CreatePool(ctx, msg)
    require.NoError(t, err)
    require.NotNil(t, resp)
}
```

### Table-Driven Test Example

```go
tests := []struct {
    name    string
    msg     *types.MsgSwap
    wantErr bool
}{
    {
        name: "valid swap",
        msg: &types.MsgSwap{...},
        wantErr: false,
    },
    {
        name: "invalid pool",
        msg: &types.MsgSwap{...},
        wantErr: true,
    },
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test logic
    })
}
```

## Coverage Goals

- Unit Tests: 80%+
- Integration Tests: Key workflows
- E2E Tests: User journeys

## CI/CD

Tests run automatically on:

- Push to master/main/develop
- Pull requests
- Multiple Go versions

## Common Issues

### Import Errors

```bash
go mod tidy
go mod download
```

### Race Conditions

```bash
go test -race ./...
```

### Slow Tests

```bash
# Run only fast tests
go test -short ./...
```

## Need Help?

See [TESTING.md](./TESTING.md) for comprehensive documentation.
