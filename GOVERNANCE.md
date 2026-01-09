# Governance

PAW uses the Cosmos SDK governance module for on-chain decision-making.

## Roles

### Core Maintainers
- Review and merge PRs to protected branches
- Coordinate releases and security patches
- Triage issues and manage milestones

### Contributors
- Submit PRs following CONTRIBUTING.md
- Participate in discussions and code reviews
- Report bugs and suggest improvements

### Validators
- Participate in on-chain governance voting
- Secure the network through staking
- Relay IBC packets (optional)

## Becoming a Maintainer

1. Demonstrate sustained quality contributions
2. Show understanding of codebase architecture
3. Receive nomination from existing maintainer
4. Approval via consensus of current maintainers

## On-Chain Governance

PAW uses the standard Cosmos SDK `x/gov` module:

| Parameter | Value |
|-----------|-------|
| Min Deposit | 512 PAW |
| Deposit Period | 14 days |
| Voting Period | 14 days |
| Quorum | 40% |
| Pass Threshold | 50% |
| Veto Threshold | 33.4% |

### Proposal Types
- **Text**: Non-binding signaling proposals
- **Parameter Change**: Modify chain parameters
- **Software Upgrade**: Coordinate binary upgrades
- **Community Spend**: Fund ecosystem initiatives

### CLI Commands

```bash
# Submit a proposal
pawd tx gov submit-proposal --title "Title" --description "Desc" --type text --from <key>

# Deposit to a proposal
pawd tx gov deposit <proposal-id> 512upaw --from <key>

# Vote on a proposal
pawd tx gov vote <proposal-id> yes --from <key>

# Query proposals
pawd query gov proposals
```

## Off-Chain Governance

- **GitHub Issues**: Feature requests and bug reports
- **GitHub Discussions**: Architecture decisions and RFCs
- **Discord**: Real-time community discussion

## Decision Process

1. **Proposal**: Author creates GitHub issue or governance proposal
2. **Discussion**: Community provides feedback and refinements
3. **Voting**: On-chain vote or maintainer consensus (for code changes)
4. **Implementation**: Approved changes are merged or executed
5. **Review**: Post-implementation assessment

## Code Owners

See `.github/CODEOWNERS` for module ownership assignments.

## License

All governance decisions apply to code under Apache 2.0 license.
