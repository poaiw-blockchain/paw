# Contributing to PAW Blockchain

Thank you for your interest in contributing to PAW! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Environment Setup](#development-environment-setup)
- [Code Style Guidelines](#code-style-guidelines)
- [Commit Message Format](#commit-message-format)
- [Pull Request Process](#pull-request-process)
- [Testing Requirements](#testing-requirements)
- [Code Review Guidelines](#code-review-guidelines)
- [Development Workflow](#development-workflow)
- [Issue Reporting](#issue-reporting)
- [Security Issues](#security-issues)

## Getting Started

### Prerequisites

Before you begin, ensure you have the following installed:

- **Go** 1.21 or higher
- **Node.js** 18.x or higher (for frontend development)
- **Python** 3.8 or higher (for utilities and scripts)
- **Git** for version control
- **Docker** (optional, for containerized development)

### Development Environment Setup

> **✅ All development tools are already installed!** See [TOOLS_SETUP.md](TOOLS_SETUP.md) for complete details on available linters, formatters, security scanners, and testing tools.

1. **Fork the repository** on GitHub
2. **Clone your fork** locally:

   ```bash
   git clone https://github.com/YOUR_USERNAME/paw.git
   cd paw
   ```

3. **Add upstream remote**:

   ```bash
   git remote add upstream https://github.com/ORIGINAL_OWNER/paw.git
   ```

4. **Install dependencies**:

   ```bash
   # Go dependencies
   go mod download

   # Node.js dependencies (if applicable)
   npm install

   # Python dependencies (if applicable)
   pip install -r requirements.txt
   ```

5. **Set up commit message template** (optional but recommended):

   ```bash
   git config commit.template .gitmessage
   ```

6. **Build the project**:

   ```bash
   go build -o bin/paw ./cmd/paw
   ```

7. **Run tests** to verify setup:
   ```bash
   go test ./...
   ```

## Code Style Guidelines

### Go Code Style

We follow standard Go conventions and use `gofmt` for formatting:

- **Formatting**: Run `gofmt -w .` or `goimports -w .` before committing
- **Linting**: Use `golangci-lint run` (v1.64.8 installed) to check for issues
- **Security**: Run `gosec ./...` (v2.dev installed) to scan for security vulnerabilities
- **Vulnerabilities**: Run `govulncheck ./...` (v1.1.4 installed) to check for known vulnerabilities
- **Naming conventions**:
  - Use camelCase for variables and functions
  - Use PascalCase for exported identifiers
  - Use descriptive names (avoid single-letter variables except in short loops)
- **Comments**:
  - All exported functions, types, and constants must have comments
  - Comments should be complete sentences
  - Use `//` for single-line comments
- **Error handling**:
  - Always check and handle errors
  - Wrap errors with context using `fmt.Errorf("context: %w", err)`
- **Package organization**:
  - Keep packages focused and cohesive
  - Avoid circular dependencies

**Example:**

```go
// ProcessTransaction validates and processes a blockchain transaction.
// It returns an error if the transaction is invalid or cannot be processed.
func ProcessTransaction(tx *Transaction) error {
    if err := tx.Validate(); err != nil {
        return fmt.Errorf("transaction validation failed: %w", err)
    }

    // Process transaction logic here
    return nil
}
```

### JavaScript Code Style

- **Formatting**: Use Prettier (v3.6.2 installed) with the project's `.prettierrc` configuration
- **Linting**: Use ESLint (v8.57.1 installed) with the project's `.eslintrc` configuration
- **Commit messages**: Commitlint (v18.6.1 installed) enforces conventional commit format
- **Naming conventions**:
  - Use camelCase for variables and functions
  - Use PascalCase for classes and React components
  - Use UPPER_SNAKE_CASE for constants
- **Modern syntax**: Use ES6+ features (arrow functions, destructuring, etc.)
- **Async/await**: Prefer async/await over raw promises

### Python Code Style

- **Formatting**: Follow PEP 8 guidelines
- **Tools**: Use `black` (v25.11.0 installed) for formatting and `pylint` (v4.0.2 installed) for linting
- **Type checking**: Use `mypy` (v1.18.2 installed) for static type checking
- **Type hints**: Use type hints for function parameters and return values
- **Docstrings**: Use Google-style docstrings

## Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/) format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Types

- **feat**: New feature
- **fix**: Bug fix
- **docs**: Documentation changes
- **style**: Code style changes (formatting, missing semicolons, etc.)
- **refactor**: Code refactoring without changing functionality
- **perf**: Performance improvements
- **test**: Adding or updating tests
- **chore**: Maintenance tasks, dependency updates, etc.
- **ci**: CI/CD configuration changes
- **build**: Build system or external dependency changes

### Examples

```
feat(wallet): add support for HD wallet derivation

Implement BIP32/BIP44 hierarchical deterministic wallet support
with secure key derivation and mnemonic phrase generation.

Closes #123
```

```
fix(consensus): prevent double-spending in edge case

Fixed a race condition in transaction validation that could
allow double-spending when transactions arrive simultaneously.

Fixes #456
```

```
docs(api): update REST API documentation for v2 endpoints
```

### Guidelines

- Use imperative mood ("add feature" not "added feature")
- Keep subject line under 50 characters
- Capitalize the subject line
- Don't end subject with a period
- Separate subject from body with a blank line
- Wrap body at 72 characters
- Use body to explain what and why, not how
- Reference issues and PRs in footer

## Pull Request Process

### Branch Naming

Use descriptive branch names with prefixes:

- `feature/` - New features (e.g., `feature/hd-wallet-support`)
- `fix/` - Bug fixes (e.g., `fix/consensus-race-condition`)
- `docs/` - Documentation updates (e.g., `docs/api-reference`)
- `refactor/` - Code refactoring (e.g., `refactor/transaction-validation`)
- `test/` - Test additions/updates (e.g., `test/wallet-integration`)

### Creating a Pull Request

1. **Update your fork** with the latest upstream changes:

   ```bash
   git fetch upstream
   git checkout master
   git merge upstream/master
   ```

2. **Create a new branch** for your changes:

   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes** and commit them following the commit message format

4. **Push to your fork**:

   ```bash
   git push origin feature/your-feature-name
   ```

5. **Open a Pull Request** on GitHub
   - Fill out the PR template completely
   - Link related issues
   - Add appropriate labels
   - Request reviews from relevant maintainers

### PR Description Requirements

- Clear description of what changes were made and why
- Screenshots for UI changes
- Performance implications (if any)
- Breaking changes clearly marked
- Migration guide for breaking changes

### Review Process

1. **Automated checks** must pass:
   - All tests must pass
   - Linting must pass
   - Code coverage must not decrease significantly

2. **Code review** from at least one maintainer required

3. **Address feedback** promptly and professionally

4. **Squash commits** if requested before merging

5. **Merge** will be performed by maintainers after approval

## Testing Requirements

All pull requests must include appropriate tests:

### Unit Tests

- **Go**: Place tests in `*_test.go` files alongside the code
- **JavaScript**: Place tests in `*.test.js` or `*.spec.js` files
- **Python**: Place tests in `tests/` directory following pytest conventions

### Test Coverage

- Aim for at least 80% code coverage for new code
- Critical paths (consensus, transaction validation) require 90%+ coverage
- All bug fixes must include regression tests

### Running Tests

```bash
# Run all Go tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./pkg/wallet/...

# Run with race detection
go test -race ./...

# Run JavaScript tests
npm test

# Run Python tests
pytest tests/
```

### Integration Tests

- Add integration tests for features that interact with multiple components
- Integration tests should be placed in `tests/integration/`
- Document any special setup requirements

### Manual Testing

For significant features, include manual testing steps in the PR description.

## Code Review Guidelines

### For Contributors

- Respond to review comments professionally
- Ask for clarification if feedback is unclear
- Make requested changes or explain why you disagree
- Keep discussions focused on the code, not personal

### For Reviewers

Review PRs for:

- **Correctness**: Does the code do what it claims?
- **Security**: Are there any security vulnerabilities?
- **Performance**: Will this impact performance negatively?
- **Testing**: Are there adequate tests?
- **Documentation**: Is new functionality documented?
- **Style**: Does code follow project conventions?
- **Maintainability**: Is the code readable and maintainable?

### Review Checklist

- [ ] Code follows project style guidelines
- [ ] All tests pass
- [ ] New code has appropriate test coverage
- [ ] Documentation updated (if needed)
- [ ] No unnecessary dependencies added
- [ ] Security considerations addressed
- [ ] Performance impact considered
- [ ] Breaking changes clearly documented

## Development Workflow

### Standard Workflow

1. **Check for existing issues** before starting work
2. **Create or comment on issue** to discuss approach
3. **Fork and clone** the repository
4. **Create feature branch** from `master`
5. **Make changes** following code style guidelines
6. **Write tests** for your changes
7. **Run tests** locally to ensure they pass
8. **Commit changes** using conventional commit format
9. **Push to your fork** and create PR
10. **Address review feedback** as needed
11. **Celebrate** when your PR is merged!

### Keeping Your Fork Updated

Regularly sync your fork with upstream:

```bash
git fetch upstream
git checkout master
git merge upstream/master
git push origin master
```

### Resolving Conflicts

If your branch has conflicts with master:

```bash
git fetch upstream
git checkout your-branch
git rebase upstream/master
# Resolve conflicts
git rebase --continue
git push --force-with-lease origin your-branch
```

## Issue Reporting

### Bug Reports

When reporting bugs, please use the bug report template and include:

- Clear, descriptive title
- Detailed description of the issue
- Steps to reproduce
- Expected vs actual behavior
- Environment details (OS, Go version, etc.)
- Relevant logs or error messages
- Screenshots (if applicable)

### Feature Requests

When requesting features, please:

- Check if the feature already exists or is planned
- Clearly describe the use case
- Explain why this feature would benefit the project
- Provide examples of how it would be used
- Consider implementation complexity

### Questions

For questions:

- Check existing documentation first
- Search closed issues for similar questions
- Use GitHub Discussions for general questions
- Use issues only for actionable bugs or feature requests

## Security Issues

**DO NOT** open public issues for security vulnerabilities.

Instead, please follow our [Security Policy](SECURITY.md):

1. Email security concerns to the maintainers (specified in SECURITY.md)
2. Include detailed description of the vulnerability
3. Provide steps to reproduce if possible
4. Allow time for the issue to be addressed before public disclosure

We take security seriously and will respond to reports promptly.

## Community Guidelines

- Be respectful and inclusive
- Follow our [Code of Conduct](CODE_OF_CONDUCT.md)
- Help others and share knowledge
- Provide constructive feedback
- Credit others for their contributions

## Recognition

Contributors will be recognized in:

- Project README
- Release notes
- Annual contributor highlights

## Getting Help

- **Documentation**: Check the `/docs` directory
- **Discussions**: Use GitHub Discussions for questions
- **Chat**: Join our community chat (if available)
- **Issues**: Search existing issues before creating new ones

## License

By contributing to PAW, you agree that your contributions will be licensed under the same license as the project (see LICENSE file).

---

Thank you for contributing to PAW! Your efforts help make blockchain technology more accessible and secure for everyone.
