# Running  Actions Tests Locally (FREE!)

Stop paying for  Actions minutes! This guide shows you how to run all the same tests locally before pushing.

## 🚀 Quick Start

### Go (PAW blockchain)
- One-shot local suite: `make test-local` (uses `GOTOOLCHAIN=go1.23.1`, cleans caches, runs `go test ./...`)
- On Windows PowerShell: `pwsh -File scripts/local-test.ps1`

### Devnet Smoke Tests (Phase 1)
- Bring up the 4-node dockerized devnet and run on-chain smoke tests:  
  `./scripts/devnet/smoke_tests.sh`
- The script waits for RPC readiness, sends a bank transfer, creates a DEX pool, and executes a swap.  
  Set `PAW_SMOKE_KEEP_STACK=1` to leave the containers running for manual debugging.

### Windows (PowerShell)
```powershell
# Run complete CI pipeline
.\local-ci.ps1

# Quick checks only (faster)
.\local-ci.ps1 -Quick

# Skip tests (linting + security only)
.\local-ci.ps1 -SkipTests

# Security scans only
.\local-ci.ps1 -SecurityOnly
```

### Linux/Mac/WSL (Bash)
```bash
# Make script executable
chmod +x local-ci.sh

# Run complete CI pipeline
./local-ci.sh

# Quick checks only (faster)
./local-ci.sh --quick

# Skip tests (linting + security only)
./local-ci.sh --skip-tests

# Security scans only
./local-ci.sh --security-only
```

### Using Make (Cross-platform)
```bash
# See all available commands
make help

# Run everything
make all

# Quick checks
make quick

# Just run tests
make test

# Just security scans
make security

# Format code automatically
make format

# Generate coverage report
make coverage
```

## 📋 What Gets Checked

### 1. Code Quality (Linting)
- ✅ **Black** - Code formatting
- ✅ **isort** - Import sorting
- ✅ **Flake8** - Style guide enforcement
- ✅ **Pylint** - Code analysis
- ✅ **MyPy** - Type checking

### 2. Security Scanning
- ✅ **Bandit** - Python security linter
- ✅ **Safety** - Dependency vulnerability scanner
- ✅ **pip-audit** - Dependency auditing
- ✅ **Semgrep** - SAST (Static Application Security Testing)

### 3. Testing
- ✅ **Unit Tests** - Fast, isolated tests
- ✅ **Integration Tests** - Component interaction tests
- ✅ **Security Tests** - Security-focused test cases
- ✅ **Performance Tests** - Benchmarking

### 4. Coverage
- ✅ **pytest-cov** - Code coverage measurement
- ✅ **HTML Reports** - Visual coverage reports

## ⚡ Pre-Commit Hooks (Automatic Checks)

Set up automatic checks before every commit:

```bash
# Install pre-commit
pip install pre-commit

# Setup hooks (one-time)
make pre-commit-setup
# OR
pre-commit install

# Test it (optional)
pre-commit run --all-files
```

Now every time you ` commit`, these checks run automatically:
- Code formatting (Black, isort)
- Linting (Flake8)
- Security (Bandit)
- Type checking (MyPy)
- Secret detection
- YAML/JSON validation

## 💡 Recommended Workflow

### Before Every Commit
```bash
# Quick checks (30 seconds - 1 minute)
make quick
```

### Before Every Push
```bash
# Full CI pipeline (5-10 minutes)
make all
# OR
.\local-ci.ps1  # Windows
./local-ci.sh   # Linux/Mac
```

### Daily/Weekly
```bash
# Full security audit
make security
```

## 📊 Reports Generated

All reports are saved locally (add to `ignore`):

- `bandit-report.json` - Security vulnerabilities
- `safety-report.json` - Dependency vulnerabilities
- `pip-audit-report.json` - Dependency audit results
- `semgrep-report.json` - SAST findings
- `coverage.xml` - Coverage data (XML)
- `htmlcov/` - Coverage report (HTML) - Open `htmlcov/index.html` in browser

## 🔧 Installation

### First Time Setup
```bash
# Install development dependencies
make install

# OR manually
pip install -r requirements-dev.txt
pip install black isort flake8 pylint mypy
pip install bandit safety semgrep pip-audit
pip install pytest pytest-cov pytest-xdist pytest-timeout
```

### Virtual Environment (Recommended)
```bash
# Create virtual environment
python -m venv .venv

# Activate it
# Windows:
.\.venv\Scripts\Activate.ps1
# Linux/Mac:
source .venv/bin/activate

# Install dependencies
make install
```

### CosmoPy Protobuf Bindings
The Python SDK now depends on the `cosmospy-protobuf` wheel bundled in this repository.  
Run the helper script once (or anytime you recreate `.venv-pytest`) to install both wheels in the correct order:

```bash
scripts/python/setup_paw_sdk.sh
```

This bootstraps `.venv-pytest`, installs `sdk/python/cosmospy-protobuf`, and finally installs `archive/sdk/python[dev]` so `pip` never reaches out to PyPI for unpublished artifacts.

## 💰 Cost Savings

###  Actions Costs
- Free tier: 2,000 minutes/month
- Paid: $0.008/minute (Linux), $0.016/minute (Windows), $0.08/minute (macOS)

### Your Comprehensive CI Pipeline
- **Full run**: ~15-20 minutes/push
- **Quick run**: ~3-5 minutes/push
- **Multiple Python versions**: 3x time
- **Multiple OS**: 2-3x time

**Monthly savings** (assuming 5 pushes/day):
- Full CI: 5 pushes × 20 min × 3 versions × 2 OS = **600 minutes/day**
- Monthly: 600 × 30 = **18,000 minutes** = $144-$1,440/month saved!

## 🎯 Tips

### Speed Up Tests
```bash
# Run tests in parallel
pytest -n auto

# Run only failed tests from last run
pytest --lf

# Stop on first failure
pytest -x
```

### Auto-fix Issues
```bash
# Auto-format code
make format

# Auto-fix linting issues
black src/ tests/
isort src/ tests/
```

### IDE Integration
Most IDEs can run these tools automatically:
- **VS Code**: Install Python, Pylint, Black extensions
- **PyCharm**: Built-in support for most tools
- **Vim/Neovim**: Use ALE or coc.nvim

## 🐛 Troubleshooting

### "Command not found"
Install the missing tool:
```bash
pip install <tool-name>
```

### "Permission denied" (Linux/Mac)
Make script executable:
```bash
chmod +x local-ci.sh
```

### Tests fail locally but pass in CI
- Check Python version: `python --version`
- Check dependencies: `pip freeze`
- Clean cache: `make clean`

### Out of memory during tests
```bash
# Reduce parallel workers
pytest -n 2  # instead of -n auto
```

## 📝 Configuration Files

- `.pre-commit-config.yaml` - Pre-commit hooks configuration
- `.flake8` / `setup.cfg` - Flake8 configuration
- `.pylintrc` - Pylint configuration
- `pyproject.toml` - Black, isort, MyPy configuration
- `pytest.ini` - Pytest configuration
- `.bandit` - Bandit security configuration

## 🔗 Related Commands

```bash
# Check what would be pushed
 log origin/main..HEAD

# Run tests for specific file
pytest tests/test_specific.py -v

# Check coverage for specific module
pytest --cov=src.module tests/ --cov-report=term-missing

# Profile slow tests
pytest --durations=10
```

## 🎉 Success!

When you see:
```
✓ All checks passed! Safe to push to .
💰 You just saved  Actions minutes!
```

You're good to push! 🚀

---

**Questions?** Check the [ Actions workflows](hub/workflows/) to see what CI runs in the cloud.
