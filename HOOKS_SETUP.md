# Git Hooks Setup Guide

This document provides comprehensive instructions for setting up Git hooks in the PAW blockchain project.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation Methods](#installation-methods)
  - [Method 1: Pre-Commit Framework (Recommended)](#method-1-pre-commit-framework-recommended)
  - [Method 2: Husky (Node.js)](#method-2-husky-nodejs)
  - [Method 3: Both Methods](#method-3-both-methods)
- [What Gets Checked](#what-gets-checked)
- [Platform-Specific Instructions](#platform-specific-instructions)
- [Usage](#usage)
- [Troubleshooting](#troubleshooting)
- [Configuration](#configuration)

## Overview

PAW blockchain uses Git hooks to ensure code quality and consistency across the codebase. The hooks automatically:

- Format code (Go, JavaScript, Python)
- Run linters and static analysis
- Execute fast unit tests
- Check for sensitive data
- Validate commit message format
- Prevent commits to protected branches

## Prerequisites

### Required Tools

**For Go Development:**
- Go 1.21+ (required)
- `gofmt` (included with Go)
- `goimports`: `go install golang.org/x/tools/cmd/goimports@latest`
- `golangci-lint`: See [installation guide](https://golangci-lint.run/usage/install/)

**For JavaScript Development:**
- Node.js 18+ (required for Husky method)
- npm 9+

**For Python Development:**
- Python 3.11+
- `pip3`

### Installation Tools

Choose one or both:

**Pre-Commit Framework (Python-based):**
```bash
pip3 install pre-commit
```

**Husky (Node.js-based):**
```bash
npm install
```

## Installation Methods

### Method 1: Pre-Commit Framework (Recommended)

The pre-commit framework provides comprehensive hooks with automatic dependency management.

#### Linux/Mac

```bash
# Install pre-commit
pip3 install --user pre-commit

# Run installation script
bash scripts/install-hooks.sh --method=pre-commit

# Or install manually
pre-commit install
pre-commit install --hook-type commit-msg
```

#### Windows (PowerShell)

```powershell
# Install pre-commit
pip install --user pre-commit

# Run installation script
.\scripts\install-hooks.ps1 -Method pre-commit

# Or install manually
pre-commit install
pre-commit install --hook-type commit-msg
```

#### Windows (Git Bash)

```bash
# Install pre-commit
pip install --user pre-commit

# Run installation script
bash scripts/install-hooks.sh --method=pre-commit
```

### Method 2: Husky (Node.js)

Husky provides lightweight Git hooks through npm.

#### Linux/Mac/Windows

```bash
# Install Node.js dependencies
npm install

# Husky will be automatically configured via package.json prepare script
# Or manually:
npx husky install
```

### Method 3: Both Methods

For maximum flexibility, install both:

#### Linux/Mac

```bash
bash scripts/install-hooks.sh --method=both
```

#### Windows (PowerShell)

```powershell
.\scripts\install-hooks.ps1 -Method both
```

## What Gets Checked

### Pre-Commit Checks

#### Go Files (`.go`)

- **gofmt**: Format code with `-s` flag for simplification
- **goimports**: Organize imports with local prefix `github.com/paw`
- **go vet**: Static analysis for common issues
- **golangci-lint**: Comprehensive linting (30+ linters enabled)
  - bodyclose, errcheck, goconst, gocritic, gosec, etc.
- **go test**: Fast unit tests with `-short -race -cover`
- **go mod tidy**: Ensure dependencies are clean

#### JavaScript Files (`.js`, `.jsx`, `.ts`, `.tsx`)

- **ESLint**: Code linting with auto-fix
- **Prettier**: Code formatting

#### Python Files (`.py`)

- **Black**: Code formatting (line length: 100)
- **Pylint**: Code linting
- **MyPy**: Static type checking

#### General Checks

- **Trailing whitespace**: Removed automatically
- **End-of-file fixer**: Ensures files end with newline
- **YAML/JSON validation**: Syntax checking
- **Merge conflict detection**: Prevents accidental commits
- **Large file detection**: Warns on files > 1MB
- **Private key detection**: Prevents credential leaks
- **Branch protection**: Blocks direct commits to `master`/`main`
- **Mixed line endings**: Normalizes to LF
- **Markdown linting**: Style checking for documentation

### Commit Message Validation

Commit messages must follow [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Valid types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding or updating tests
- `build`: Build system changes
- `ci`: CI/CD changes
- `chore`: Other changes (dependencies, etc.)
- `revert`: Revert a previous commit

**Examples:**

```
feat(wallet): add Fernet encryption support

Implements secure wallet storage using Fernet symmetric encryption.

Closes #123
```

```
fix(api): resolve connection timeout issue

Increased timeout from 30s to 60s for slow networks.
```

```
docs: update README with installation steps
```

### Pre-Push Checks

- **Full test suite**: All unit tests with race detection
- **Build verification**: Ensures code compiles
- **go.mod/go.sum validation**: Ensures dependencies are tidy
- **Branch protection warning**: Confirms before pushing to protected branches

## Platform-Specific Instructions

### Windows

#### Using PowerShell

```powershell
# Clone and navigate to repository
cd C:\users\decri\gitclones\paw

# Install pre-commit (choose one)
pip install --user pre-commit

# Run installation script
.\scripts\install-hooks.ps1

# Or use Node.js method
npm install
```

#### Using Git Bash (Recommended for Windows)

```bash
# Navigate to repository
cd /c/users/decri/gitclones/paw

# Install pre-commit
pip install --user pre-commit

# Run installation script
bash scripts/install-hooks.sh

# Or use Node.js method
npm install
```

#### Notes for Windows Users

- Pre-commit framework works best in Git Bash on Windows
- If using PowerShell, ensure Python is in your PATH
- Husky works equally well in PowerShell, CMD, and Git Bash
- Some hooks may run slower on Windows due to filesystem performance

### Linux

```bash
# Install pre-commit
pip3 install --user pre-commit

# Or via system package manager (Ubuntu/Debian)
sudo apt install pre-commit

# Run installation script
bash scripts/install-hooks.sh
```

### macOS

```bash
# Install pre-commit via Homebrew
brew install pre-commit

# Or via pip
pip3 install --user pre-commit

# Run installation script
bash scripts/install-hooks.sh
```

## Usage

### Running Hooks Manually

```bash
# Run all hooks on all files
pre-commit run --all-files

# Run specific hook
pre-commit run golangci-lint --all-files

# Run hooks on specific files
pre-commit run --files app/app.go cmd/pawd/main.go
```

### Updating Hooks

```bash
# Update pre-commit hooks to latest versions
pre-commit autoupdate

# Update npm dependencies
npm update
```

### Bypassing Hooks (Use Sparingly!)

```bash
# Skip pre-commit hooks
git commit --no-verify -m "emergency fix"

# Skip pre-push hooks
git push --no-verify
```

**⚠️ Warning**: Only bypass hooks when absolutely necessary (e.g., emergency hotfixes).

### Testing Commit Message Format

```bash
# This will trigger commit-msg hook validation
git commit --allow-empty -m "test: verify hooks work"

# Invalid format (will be rejected)
git commit -m "updated readme"  # Missing type

# Valid format
git commit -m "docs: update README with hook instructions"
```

## Troubleshooting

### Common Issues

#### "pre-commit: command not found"

**Solution**: Ensure pre-commit is installed and in your PATH

```bash
# Linux/Mac
pip3 install --user pre-commit
export PATH="$HOME/.local/bin:$PATH"

# Windows
pip install --user pre-commit
# Add %USERPROFILE%\AppData\Local\Programs\Python\Python3XX\Scripts to PATH
```

#### "golangci-lint: not found"

**Solution**: Install golangci-lint

```bash
# Linux/Mac
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Or via curl
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Windows
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

#### "npm: command not found"

**Solution**: Install Node.js

- Download from [nodejs.org](https://nodejs.org/)
- Or use a version manager: `nvm`, `fnm`, etc.

#### Hooks Running Slowly

**Solutions**:

1. **Run hooks only on changed files** (default behavior)
2. **Skip slow hooks during development**:
   ```bash
   SKIP=golangci-lint git commit -m "feat: quick commit"
   ```
3. **Disable specific hooks**:
   Edit `.pre-commit-config.yaml` and comment out slow hooks
4. **Use Husky instead** (lighter weight)

#### Hooks Failing on CI/CD

**Solution**: Ensure CI environment has all required tools installed

```yaml
# Example GitHub Actions
- name: Set up pre-commit
  run: |
    pip install pre-commit
    pre-commit run --all-files
```

#### Windows: "bad interpreter" Error

**Solution**: Ensure scripts have Unix line endings (LF, not CRLF)

```bash
# Fix line endings
git config core.autocrlf false
git rm --cached -r .
git reset --hard
```

### Getting Help

If you encounter issues:

1. Check this documentation
2. Review `.pre-commit-config.yaml` for hook configuration
3. Run with verbose output: `pre-commit run --all-files --verbose`
4. Check individual tool documentation
5. Open an issue in the repository

## Configuration

### Pre-Commit Configuration

Edit `.pre-commit-config.yaml` to customize hook behavior:

```yaml
repos:
  - repo: https://github.com/golangci/golangci-lint
    rev: v1.55.2
    hooks:
      - id: golangci-lint
        args:
          - --config=.golangci.yml
          - --timeout=10m
          # Add custom args here
```

### golangci-lint Configuration

Edit `.golangci.yml` to customize Go linting:

```yaml
linters:
  enable:
    - bodyclose
    - errcheck
    # Add more linters
```

### ESLint Configuration

Edit `.eslintrc.json` for JavaScript linting rules:

```json
{
  "rules": {
    "semi": ["error", "always"],
    "quotes": ["error", "single"]
  }
}
```

### Prettier Configuration

Edit `.prettierrc.json` for JavaScript formatting:

```json
{
  "semi": true,
  "singleQuote": true,
  "printWidth": 100
}
```

### Commitlint Configuration

Edit `.commitlintrc.json` for commit message rules:

```json
{
  "rules": {
    "type-enum": [2, "always", ["feat", "fix", "docs"]],
    "header-max-length": [2, "always", 100]
  }
}
```

### Disabling Specific Hooks

To disable a hook temporarily:

```bash
# Environment variable
SKIP=golangci-lint git commit -m "message"

# Or edit .pre-commit-config.yaml and comment out the hook
```

### Custom Local Hooks

Add custom validation in `scripts/hooks/`:

```bash
# scripts/hooks/my-custom-check.sh
#!/bin/bash
# Your custom validation logic
```

Then reference in `.pre-commit-config.yaml`:

```yaml
- repo: local
  hooks:
    - id: my-custom-check
      name: My Custom Check
      entry: scripts/hooks/my-custom-check.sh
      language: script
```

## Best Practices

1. **Commit frequently**: Small, focused commits are easier to review
2. **Write meaningful commit messages**: Follow conventional commits format
3. **Fix hook failures immediately**: Don't bypass hooks unless absolutely necessary
4. **Keep hooks fast**: Test hooks should complete in < 60 seconds
5. **Update regularly**: Run `pre-commit autoupdate` monthly
6. **Document exceptions**: If you must bypass hooks, document why

## Additional Resources

- [Pre-Commit Framework Documentation](https://pre-commit.com/)
- [Husky Documentation](https://typicode.github.io/husky/)
- [Conventional Commits](https://www.conventionalcommits.org/)
- [golangci-lint Documentation](https://golangci-lint.run/)
- [ESLint Documentation](https://eslint.org/)
- [Prettier Documentation](https://prettier.io/)

## Support

For issues or questions:

- Check the [Troubleshooting](#troubleshooting) section
- Review PAW project documentation
- Open an issue in the repository
- Contact the development team

---

**Remember**: Git hooks are here to help maintain code quality. If a hook is causing problems, there's usually a good reason—investigate before bypassing!
