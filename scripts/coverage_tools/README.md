# Coverage Automation Tools for PAW (Go)

Tools for achieving and maintaining 98%+ test coverage in Go projects.

## Tools

### 1. go_test_generator.go

**Purpose**: Generate test scaffolds for Go functions.

**Features**:
- Parses Go source files
- Generates table-driven test templates
- Creates benchmark tests
- Provides fixture stubs
- Includes edge case test placeholders

**Build**:
```bash
go build -o go_test_generator go_test_generator.go
```

**Usage**:
```bash
./go_test_generator <go_file> [--output <test_file>]
```

**Example**:
```bash
./go_test_generator ../../cmd/main.go
./go_test_generator ../../x/types/types.go --output ../../x/types/types_test.go
```

**Generated Output**:
- Table-driven test template
- Benchmark test template
- Fixture setup/cleanup functions
- Edge case test stubs

---

### 2. coverage_diff.py

**Purpose**: Compare Go and Python coverage metrics.

**Features**:
- Parses Go coverage.out files
- Compares with Python coverage.json
- Identifies coverage gaps
- Generates comparative reports
- Tracks progress across projects

**Usage**:
```bash
python coverage_diff.py \
  --go-coverage coverage.out \
  --py-coverage ../Crypto/coverage.json \
  [--show-gaps] [--report] [--html <file>]
```

**Options**:
- `--go-coverage`: Path to Go coverage.out
- `--py-coverage`: Path to Python coverage.json
- `--show-gaps`: Show modules below threshold
- `--threshold`: Coverage threshold (default: 95%)
- `--report`: Generate text report
- `--html`: Generate HTML report
- `--output`: Save report to file

**Example**:
```bash
# Show coverage comparison
python coverage_diff.py --go-coverage coverage.out

# Identify gaps
python coverage_diff.py --go-coverage coverage.out --show-gaps --threshold 95

# Generate comparison report
python coverage_diff.py \
  --go-coverage coverage.out \
  --py-coverage ../Crypto/coverage.json \
  --report \
  --output comparison.txt

# Generate HTML report
python coverage_diff.py \
  --go-coverage coverage.out \
  --py-coverage ../Crypto/coverage.json \
  --html coverage_comparison.html
```

---

## Workflow

### Generate Tests for New Module

```bash
# 1. Create new Go module
mkdir -p x/mymodule
cat > x/mymodule/types.go << 'EOF'
package mymodule

func Calculate(x int) int {
    return x * 2
}
EOF

# 2. Build test generator
go build -o go_test_generator go_test_generator.go

# 3. Generate test scaffold
./go_test_generator ../../x/mymodule/types.go \
  --output ../../x/mymodule/types_test.go

# 4. Implement tests in x/mymodule/types_test.go
# Edit the generated _test.go file and implement test cases

# 5. Run tests
cd ../../
go test -v ./x/mymodule/...

# 6. Check coverage
go test -cover ./x/mymodule/...
```

### Check Coverage Over Time

```bash
# Generate coverage baseline
go test -coverprofile=coverage.out ./...

# Later, compare
go test -coverprofile=coverage_new.out ./...
python coverage_diff.py --go-coverage coverage_new.out

# Track trends
python coverage_diff.py --go-coverage coverage.out --report
```

### Cross-Project Analysis

```bash
# From PAW project root
cd scripts/coverage_tools

# Generate Go coverage
go test -coverprofile=../../coverage.out ./...

# Compare with Python project
python coverage_diff.py \
  --go-coverage ../../coverage.out \
  --py-coverage ../../Crypto/coverage.json \
  --show-gaps \
  --threshold 98
```

---

## Integration with CI/CD

###  Actions Example

```yaml
name: Test Coverage

on: [push, pull_request]

jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'

      - name: Run tests with coverage
        run: |
          go test -coverprofile=coverage.out ./...

      - name: Check coverage threshold
        run: |
          # Simple check: ensure coverage > 90%
          coverage=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
          if (( $(echo "$coverage < 90" | bc -l) )); then
            echo "Coverage $coverage% is below 90%"
            exit 1
          fi

      - name: Generate coverage report
        run: |
          go tool cover -html=coverage.out -o coverage.html

      - name: Upload coverage report
        uses: actions/upload-artifact@v3
        with:
          name: coverage-report
          path: coverage.html
```

---

## Best Practices

### 1. Table-Driven Tests

Use the generated structure for all test cases:

```go
func TestFunction(t *testing.T) {
    tests := []struct {
        name    string
        args    args
        want    interface{}
        wantErr bool
    }{
        {
            name: "basic case",
            args: args{param: "value"},
            want: expectedValue,
            wantErr: false,
        },
        {
            name: "empty input",
            args: args{param: ""},
            want: nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### 2. Benchmark Tests

Include benchmarks for performance-critical functions:

```go
func BenchmarkFunction(b *testing.B) {
    // Setup
    setup := prepareTestData()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        Function(setup)
    }
}
```

### 3. Coverage Targets

- **Core modules**: 95%+ coverage
- **Business logic**: 98%+ coverage
- **Security modules**: 99%+ coverage
- **Utils/helpers**: 90%+ coverage

### 4. Edge Cases to Test

- Empty values
- Nil/null inputs
- Maximum values
- Minimum values
- Boundary conditions
- Invalid types
- Concurrent access
- Error conditions

---

## Troubleshooting

### Coverage file not found

```bash
# Generate coverage
go test -coverprofile=coverage.out ./...

# Or use specific path
python coverage_diff.py --go-coverage ./path/to/coverage.out
```

### Test generator fails

```bash
# Ensure Go module is initialized
go mod init github.com/paw-chain/paw
go mod tidy

# Build generator explicitly
go build -v -o go_test_generator go_test_generator.go
```

### Coverage seems wrong

```bash
# Check specific module
go test -coverprofile=coverage.out ./x/mymodule/...
go tool cover -func=coverage.out

# Generate HTML report
go tool cover -html=coverage.out
```

### Python tools not available

```bash
# Install Python dependencies
pip install coverage pathlib

# Run from correct directory
cd scripts/coverage_tools
python coverage_diff.py --help
```

---

## File Structure

```
scripts/coverage_tools/
├── README.md                    # This file
├── go_test_generator.go         # Go test generation tool
├── coverage_diff.py             # Coverage comparison tool
└── (compiled binary)

├── Makefile (optional)          # Build commands
```

---

## Dependencies

### Go Test Generator
- Go 1.23+
- Standard library only

### Coverage Diff Tool
- Python 3.10+
- No external dependencies

### CI/CD Integration
- Go test suite
- Coverage tools
- (Optional)  Actions

---

## Examples

### Example 1: Generate Tests for New Keeper

```bash
# Create keeper module
cat > x/keeper/keeper.go << 'EOF'
package keeper

type Keeper struct {
    // ...
}

func (k Keeper) GetValue(key string) (interface{}, error) {
    // ...
}

func (k Keeper) SetValue(key string, value interface{}) error {
    // ...
}
EOF

# Generate test scaffold
go_test_generator x/keeper/keeper.go \
  --output x/keeper/keeper_test.go

# The test file will contain:
# - TestKeeperGetValue with table-driven structure
# - TestKeeperSetValue with table-driven structure
# - BenchmarkKeeperGetValue
# - BenchmarkKeeperSetValue

# Implement test cases in x/keeper/keeper_test.go
```

### Example 2: Compare Coverage Across Projects

```bash
# From PAW project
go test -coverprofile=coverage.out ./...

# Compare with XAI project
python scripts/coverage_tools/coverage_diff.py \
  --go-coverage coverage.out \
  --py-coverage ../Crypto/coverage.json \
  --show-gaps \
  --threshold 95

# Output shows:
# - Go modules below 95%
# - Python modules below 95%
# - Coverage comparison
# - Priority gaps to address
```

### Example 3: Monitor Coverage Over Time

```bash
# Baseline
go test -coverprofile=coverage.baseline.out ./...

# After development
go test -coverprofile=coverage.new.out ./...

# Compare
python coverage_diff.py --go-coverage coverage.new.out

# Generate trend
python coverage_diff.py --go-coverage coverage.new.out --report
```

---

## Resources

- [Go Testing](https://golang.org/pkg/testing/)
- [Go Coverage Tool](https://golang.org/cmd/cover/)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Benchmark Testing](https://dave.cheney.net/2013/06/30/how-to-write-benchmarks-in-go)

---

## Version History

- **1.0.0** (2025-11-18): Initial release
  - go_test_generator.go
  - coverage_diff.py
  - Comprehensive documentation

---

**Last Updated**: 2025-11-18
**Maintained By**: Coverage Automation Team
