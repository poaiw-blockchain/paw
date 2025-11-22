# Development Workflow & Standards

## ⚠️ CRITICAL: Local Testing Policy

**ALL TESTING MUST BE DONE LOCALLY BEFORE PUSHING TO GITHUB**

This policy is in place to:
- ✅ Save GitHub Actions minutes (costs money!)
- ✅ Get faster feedback (local is faster than CI)
- ✅ Catch issues before they reach the repository
- ✅ Maintain professional development standards

## 🔒 Mandatory Pre-Push Checklist

### Before EVERY Push to GitHub

Run one of these commands locally:

```bash
# OPTION 1: Quick validation (1-2 minutes) - MINIMUM requirement
make quick
# or
.\local-ci.ps1 -Quick

# OPTION 2: Full CI pipeline (5-10 minutes) - RECOMMENDED
make ci
# or
.\local-ci.ps1
```

### ❌ DO NOT Push if:
- Any linting errors exist
- Any tests fail
- Security scans show critical issues
- Code coverage drops significantly

### ✅ Only Push When:
- All local tests pass
- Code is properly formatted (Black, isort)
- No security vulnerabilities detected
- Coverage is maintained or improved

## 🎯 Standard Development Workflow

### 1. Starting New Work
```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Ensure dependencies are up to date
pip install -r requirements.txt
pip install -r requirements-dev.txt
```

### 2. During Development
```bash
# Auto-format code frequently
make format

# Run quick checks periodically
make quick
```

### 3. Before Committing
```bash
# Run quick validation
make quick

# If all passes, commit
git add .
git commit -m "feat: your descriptive message"
```

### 4. Before Pushing (MANDATORY)
```bash
# Run FULL local CI pipeline
make ci
# OR at minimum
.\local-ci.ps1 -Quick

# If all passes, push
git push origin feature/your-feature-name
```

### 5. Creating Pull Request
```bash
# Verify one more time
make ci

# Create PR
gh pr create --title "Your PR Title" --body "Description"
```

## 🛠️ Available Commands

### Quick Reference
```bash
# FASTEST - Format code
make format                    # Auto-fix formatting issues

# FAST - Quick validation (1-2 min)
make quick                     # Formatting + linting + unit tests
.\local-ci.ps1 -Quick

# RECOMMENDED - Full CI (5-10 min)
make ci                        # Complete CI pipeline
.\local-ci.ps1

# COMPREHENSIVE - All checks
make all                       # Everything including integration tests

# SECURITY - Security only
make security                  # All security scans
.\local-ci.ps1 -SecurityOnly

# TESTING - Specific test suites
make test-unit                 # Unit tests only
make test-integration          # Integration tests only
make test-security            # Security tests only
make test-performance         # Performance tests only

# COVERAGE - Coverage reports
make coverage                  # Generate coverage report
```

## 🔄 Automatic Pre-Commit Hooks (RECOMMENDED)

Set up automatic checking before every commit:

```bash
# One-time setup
pip install pre-commit
pre-commit install

# Now every git commit automatically runs:
# ✓ Black formatting
# ✓ isort import sorting
# ✓ Flake8 linting
# ✓ Bandit security scanning
# ✓ MyPy type checking
# ✓ Secret detection
```

To skip hooks (ONLY in emergencies):
```bash
git commit --no-verify -m "emergency fix"
# ⚠️ WARNING: Still must run full CI before pushing!
```

## 📊 Understanding Test Results

### Success Output
```
✓ All checks passed! Safe to push to GitHub.
💰 You just saved GitHub Actions minutes!
```
**Action**: Safe to push!

### Failure Output
```
✗ Black formatting check
✗ Unit tests

⚠ Fix failed checks before pushing to GitHub!
```
**Action**: Fix issues, then re-run tests.

### Common Fixes
```bash
# Fix formatting
make format

# View detailed test failures
pytest tests/ -v --tb=short

# Run specific failing test
pytest tests/test_specific.py::test_function -v

# Check what changed
git diff

# Revert changes if needed
git checkout -- file.py
```

## 🚨 Emergency Procedures

### "I need to push NOW!"
**Minimum acceptable:**
```bash
.\local-ci.ps1 -Quick
# Must pass before pushing
```

### "Tests are taking too long"
**Quick validation:**
```bash
make quick          # 1-2 minutes
# or
pytest tests/unit/ -x --maxfail=3  # Stop on first 3 failures
```

### "I only changed documentation"
**Still run quick check:**
```bash
make quick
# Ensures no accidental code changes
```

## 📈 CI/CD Pipeline Details

### What Runs Locally (FREE)
1. **Code Quality**
   - Black (formatting)
   - isort (import sorting)
   - Flake8 (style guide)
   - Pylint (code analysis)
   - MyPy (type checking)

2. **Security**
   - Bandit (security vulnerabilities)
   - Safety (dependency vulnerabilities)
   - pip-audit (dependency auditing)
   - Semgrep (SAST analysis)

3. **Testing**
   - Unit tests (pytest)
   - Integration tests
   - Security tests
   - Performance tests
   - Coverage measurement

### What Runs on GitHub (COSTS MONEY)
- Same checks as above (redundant validation)
- Multiple Python versions (3.10, 3.11, 3.12)
- Multiple OS (Ubuntu, Windows)
- CodeQL advanced analysis
- Dependency review
- Automated benchmarking

**Therefore**: Running locally FIRST saves money and time!

## 💡 Best Practices

### 1. Test Frequently
```bash
# Don't wait until the end
# Test as you develop
make quick  # Every 15-30 minutes
```

### 2. Commit Small Changes
```bash
# Smaller commits = easier debugging
# Test each logical change
git add specific_file.py
make quick
git commit -m "feat: add specific feature"
```

### 3. Use Coverage Reports
```bash
# Generate HTML coverage report
make coverage

# Open in browser
open htmlcov/index.html  # Mac/Linux
start htmlcov/index.html # Windows
```

### 4. Keep Dependencies Updated
```bash
# Check for updates weekly
make deps-update

# Audit for vulnerabilities
make security
```

## 🎓 Learning Resources

### Understanding Test Failures
- Read pytest output carefully
- Use `-v` for verbose output
- Use `--tb=short` for shorter tracebacks
- Use `-x` to stop on first failure

### Improving Coverage
- Aim for 80%+ coverage minimum
- 90%+ for critical modules
- 100% for security-critical code

### Security Best Practices
- Fix Bandit HIGH severity issues immediately
- Update vulnerable dependencies within 24 hours
- Never commit secrets (pre-commit will catch this)

## 🔧 Troubleshooting

### "Local tests pass but CI fails"
```bash
# Check Python version
python --version

# Update dependencies
pip install -r requirements.txt --upgrade

# Clear cache
make clean
```

### "Tests are slow"
```bash
# Run in parallel
pytest -n auto

# Run only changed tests
pytest --testmon

# Skip slow tests
pytest -m "not slow"
```

### "Out of disk space"
```bash
# Clean up
make clean
make clean-all

# Remove old coverage reports
rm -rf htmlcov/ .coverage
```

## 📞 Getting Help

### Issues with Testing
1. Check test output carefully
2. Run individual test: `pytest tests/test_file.py::test_name -v`
3. Check recent changes: `git diff`
4. Ask team for help (include error output)

### Issues with CI Scripts
1. Check script output
2. Ensure dependencies installed: `make install-dev`
3. Check virtual environment is activated
4. Try running commands individually

## ✅ Success Metrics

Track your progress:
- ✅ All commits pass `make quick`
- ✅ All pushes pass `make ci`
- ✅ Zero failed CI runs on GitHub
- ✅ Coverage maintained above 80%
- ✅ No critical security issues
- ✅ Code review approvals without major issues

## 🎯 Goals

### Short-term
- [ ] Run `make quick` before every commit
- [ ] Run `make ci` before every push
- [ ] Install pre-commit hooks
- [ ] Achieve 80%+ test coverage

### Long-term
- [ ] Zero failed GitHub Actions runs
- [ ] 90%+ test coverage
- [ ] All security scans clean
- [ ] Sub-2-minute test suite

---

## 📝 Summary

**REMEMBER**:
1. Always run `make ci` or `.\local-ci.ps1` before pushing
2. Fix all issues locally before pushing to GitHub
3. Use pre-commit hooks for automatic checking
4. Save money and time by testing locally first

**Questions?** See `RUN-TESTS-LOCALLY.md` for detailed instructions.
