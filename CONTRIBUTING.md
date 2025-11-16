# Contributing to PAW Blockchain

Thank you for considering contributing to PAW! This document provides guidelines and instructions for contributing.

## Development Setup

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/paw.git`
3. Install dependencies: `go mod download`
4. Run tests: `go test ./...`
5. Run formatters: `gofmt -w . && goimports -w .`

## Commit Guidelines

We follow conventional commits. Format: `type: description`

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting changes
- `refactor`: Code restructuring
- `test`: Adding tests
- `chore`: Maintenance tasks

Example: `fix: correct DEX pool calculation`

## Pull Request Process

1. Update tests for your changes
2. Run `go test ./...` and ensure all tests pass
3. Run `gofmt -w .` and `goimports -w .`
4. Update CHANGELOG.md with your changes
5. Create PR with clear description of changes
6. Wait for review and address feedback

## Code Standards

- Follow Go best practices
- Write tests for new functionality
- Keep functions focused and small
- Document exported functions and types
- Use descriptive variable names

## Testing

- Unit tests: `go test ./...`
- Integration tests: `go test ./tests/e2e/...`
- Benchmarks: `go test -bench=. ./tests/benchmarks/...`

## Questions?

Open an issue or join our community channels.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
