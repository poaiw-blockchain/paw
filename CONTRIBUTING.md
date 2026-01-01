# Contributing to PAW

Thank you for contributing to PAW! This guide covers our development workflow and standards.

## Code of Conduct

By participating, you agree to maintain a respectful and inclusive environment.

## Reporting Bugs

- Check existing issues to avoid duplicates
- Include steps to reproduce and system information (OS, Go version)

## Submitting Pull Requests

1. Fork and create a feature branch: `git checkout -b feature/your-feature`
2. Make changes following code style guidelines
3. Run tests: `go test ./... && pre-commit run --all-files && golangci-lint run`
4. Commit with conventional format (see below)
5. Push and create PR, address review feedback

## Development Setup

```bash
git clone https://github.com/YOUR_USERNAME/paw.git && cd paw
go mod download && pre-commit install
go build -o pawd ./cmd/...
```

## Code Style

### Go
- Follow [Effective Go](https://golang.org/doc/effective_go) and [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt`, `goimports`, and `golangci-lint`
- Avoid `panic()` in production; return errors
- Never use `sdk.MustAccAddressFromBech32` in production paths

### Protobuf
- Include comprehensive field documentation
- Run `make proto-gen` or `buf generate` after changes
- Follow Cosmos SDK proto conventions

## Commit Message Format

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`, `security`

**Examples**:
- `feat(dex): add multi-hop swap routing`
- `fix(oracle): prevent price manipulation`
- `test(compute): add fuzz tests for verification`

## Testing Requirements

### Required for All PRs
- Unit tests for new functions
- Integration tests for module changes
- Table-driven tests where appropriate
- Fuzz tests for input validation

### Module Development
1. Update protobuf definitions in `proto/`
2. Regenerate: `make proto-gen`
3. Implement keeper methods with error handling
4. Add/update genesis import/export
5. Write comprehensive tests (unit + integration)
6. Update documentation in `docs/`

### Coverage Standards
- Minimum 80% for new code
- 100% for security-critical paths

## Security

- Never commit secrets or credentials
- Run `gosec ./...` before submitting
- Follow Cosmos SDK security best practices
- Report vulnerabilities privately (see SECURITY.md)

## Pull Request Process

1. Ensure tests pass and linters are clean
2. Update documentation for user-facing changes
3. Add changelog entry if applicable
4. Request review from code owners (auto-assigned via CODEOWNERS)
5. Address feedback and squash commits if requested

## Questions?

- GitHub Discussions for general questions
- Documentation in `docs/`
- SECURITY.md for security-related questions

## License

Contributions are licensed under the project license (see LICENSE file).
