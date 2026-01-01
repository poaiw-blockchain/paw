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

## Developer Certificate of Origin (DCO)

All contributions to this project must be signed off to indicate that you have the right to submit the work under our open source license. This is done using the `--signoff` (or `-s`) flag when committing:

```bash
git commit -s -m "feat: your feature description"
```

This adds a `Signed-off-by:` line to your commit message, certifying that you agree to the [Developer Certificate of Origin](https://developercertificate.org/):

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.

Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project or the open source license(s) involved.
```

### Configuring Git for Sign-off

To automatically sign off all commits, configure your Git identity:

```bash
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

### Amending Unsigned Commits

If you forgot to sign off a commit, you can amend it:

```bash
git commit --amend -s
```

For multiple commits, use interactive rebase with the `exec` command.

## License

Contributions are licensed under the Apache License 2.0 (see LICENSE file).
