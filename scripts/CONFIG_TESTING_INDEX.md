# Configuration Testing - File Index

Quick navigation for the Phase 2.2 configuration testing framework.

## Start Here

**New to configuration testing?**
👉 Read: [`CONFIG_TESTING_README.md`](CONFIG_TESTING_README.md)

**Want to run tests now?**
👉 Read: [`CONFIG_TESTING_QUICK_REF.md`](CONFIG_TESTING_QUICK_REF.md)

## All Files

### Executable Script
| File | Size | Lines | Purpose |
|------|------|-------|---------|
| [`test-config-exhaustive.sh`](test-config-exhaustive.sh) | 43 KB | 1,314 | Main testing script |

### Documentation
| File | Size | Lines | Audience |
|------|------|-------|----------|
| [`CONFIG_TESTING_README.md`](CONFIG_TESTING_README.md) | 8.6 KB | 399 | Everyone - Overview and navigation |
| [`CONFIG_TESTING_QUICK_REF.md`](CONFIG_TESTING_QUICK_REF.md) | 5.7 KB | 227 | Users - Quick commands and examples |
| [`CONFIG_TESTING_GUIDE.md`](CONFIG_TESTING_GUIDE.md) | 12 KB | 404 | Users - Complete usage guide |
| [`CONFIG_TESTING_DESIGN.md`](CONFIG_TESTING_DESIGN.md) | 19 KB | 562 | Developers - Technical architecture |
| [`CONFIG_TESTING_DELIVERABLE_SUMMARY.txt`](CONFIG_TESTING_DELIVERABLE_SUMMARY.txt) | - | - | Summary of deliverable |
| **This file** | - | - | Navigation index |

### Makefile Integration
The main [`Makefile`](../Makefile) has been updated with three new targets:
- `make test-config` - Run exhaustive tests
- `make test-config-quick` - Run quick tests
- `make test-config-category CATEGORY=<name>` - Run category tests

## Documentation Map

```
                    START HERE
                        |
                        v
            CONFIG_TESTING_README.md
                 (Overview)
                        |
        +---------------+---------------+
        |                               |
        v                               v
Quick Reference                  User Guide
(Quick commands)              (Full documentation)
        |                               |
        v                               v
CONFIG_TESTING_              CONFIG_TESTING_
QUICK_REF.md                    GUIDE.md
                                        |
                                        v
                                 Advanced Users
                                        |
                                        v
                              CONFIG_TESTING_
                                 DESIGN.md
                            (Architecture & Design)
```

## By Use Case

### "I want to run tests right now"
1. Read: [`CONFIG_TESTING_QUICK_REF.md`](CONFIG_TESTING_QUICK_REF.md)
2. Run: `make test-config-quick`
3. Review: `config-test-report-*.md`

### "I need to understand what this does"
1. Read: [`CONFIG_TESTING_README.md`](CONFIG_TESTING_README.md)
2. Read: [`CONFIG_TESTING_GUIDE.md`](CONFIG_TESTING_GUIDE.md)
3. Run: `./scripts/test-config-exhaustive.sh --help`

### "I want to add new tests"
1. Read: [`CONFIG_TESTING_DESIGN.md`](CONFIG_TESTING_DESIGN.md) - Architecture section
2. Read: [`CONFIG_TESTING_GUIDE.md`](CONFIG_TESTING_GUIDE.md) - Extensibility section
3. Edit: [`test-config-exhaustive.sh`](test-config-exhaustive.sh) - Add run_test() calls

### "Tests are failing, what do I do?"
1. Read: [`CONFIG_TESTING_QUICK_REF.md`](CONFIG_TESTING_QUICK_REF.md) - Troubleshooting section
2. Read: [`CONFIG_TESTING_GUIDE.md`](CONFIG_TESTING_GUIDE.md) - Troubleshooting section
3. Run: `./scripts/test-config-exhaustive.sh --category <failing> --skip-cleanup`

## By Role

### QA Engineer / Tester
**Primary**: [`CONFIG_TESTING_QUICK_REF.md`](CONFIG_TESTING_QUICK_REF.md)
- Quick commands
- Common workflows
- Troubleshooting

**Secondary**: [`CONFIG_TESTING_GUIDE.md`](CONFIG_TESTING_GUIDE.md)
- Test methodology
- Expected results
- Performance tips

### Developer / Contributor
**Primary**: [`CONFIG_TESTING_DESIGN.md`](CONFIG_TESTING_DESIGN.md)
- Architecture
- Component design
- Extensibility

**Secondary**: [`test-config-exhaustive.sh`](test-config-exhaustive.sh)
- Source code
- Test definitions
- Validation functions

### DevOps / SRE
**Primary**: [`CONFIG_TESTING_GUIDE.md`](CONFIG_TESTING_GUIDE.md)
- CI/CD integration
- Performance metrics
- Best practices

**Secondary**: [`CONFIG_TESTING_QUICK_REF.md`](CONFIG_TESTING_QUICK_REF.md)
- Exit codes
- Automation examples
- Monitoring

### Project Manager / Auditor
**Primary**: [`CONFIG_TESTING_README.md`](CONFIG_TESTING_README.md)
- Overview
- Coverage summary
- Integration status

**Secondary**: [`CONFIG_TESTING_DELIVERABLE_SUMMARY.txt`](CONFIG_TESTING_DELIVERABLE_SUMMARY.txt)
- Deliverable details
- Statistics
- Quality assurance

## Quick Commands Reference

```bash
# View this index
cat scripts/CONFIG_TESTING_INDEX.md

# Read overview
cat scripts/CONFIG_TESTING_README.md

# Quick reference
cat scripts/CONFIG_TESTING_QUICK_REF.md

# Full guide
cat scripts/CONFIG_TESTING_GUIDE.md

# Architecture
cat scripts/CONFIG_TESTING_DESIGN.md

# Script help
./scripts/test-config-exhaustive.sh --help

# Run quick tests
make test-config-quick

# Run full tests
make test-config

# Run category
make test-config-category CATEGORY=p2p
```

## Coverage Summary

### config.toml
- **base** (9 tests) - Core node settings
- **rpc** (14 tests) - RPC server
- **p2p** (20 tests) - P2P networking
- **mempool** (12 tests) - Transaction pool
- **consensus** (11 tests) - Consensus engine
- **statesync** (4 tests) - State sync
- **storage** (2 tests) - Storage options
- **tx_index** (2 tests) - Transaction indexer
- **instrumentation** (3 tests) - Metrics

### app.toml
- **app-base** (11 tests) - Application settings
- **telemetry** (3 tests) - App telemetry
- **api** (5 tests) - REST API
- **grpc** (3 tests) - gRPC server
- **state-sync** (3 tests) - Snapshots
- **app-mempool** (3 tests) - App mempool

**Total**: 15 categories, 150+ tests

## Testing Modes

| Mode | Tests | Time | Command |
|------|-------|------|---------|
| Quick | ~15 | ~5 min | `make test-config-quick` |
| Category | ~10-20 | ~5-10 min | `make test-config-category CATEGORY=<name>` |
| Full | 150+ | ~30 min | `make test-config` |

## Output Files

After running tests:
- **Report**: `config-test-report-YYYYMMDD-HHMMSS.md`
- **Test dirs**: `/tmp/paw-config-test-*/` (temporary)
- **Logs**: `<test-dir>/node.log` (per test)

## Related Documentation

- **Testing Plan**: [`../LOCAL_TESTING_PLAN.md`](../LOCAL_TESTING_PLAN.md) - Phase 2.2
- **Main README**: [`../README.md`](../README.md) - Project overview
- **Makefile**: [`../Makefile`](../Makefile) - Build and test targets

## Status

- ✅ Script created and validated
- ✅ Documentation complete
- ✅ Makefile integrated
- ⏸️ Not yet executed (as per request)

## Next Steps

1. Run: `make test-config-quick`
2. Review: Generated report
3. Fix: Any issues found
4. Run: `make test-config` (full)
5. Mark: Phase 2.2 complete in LOCAL_TESTING_PLAN.md

## Support

For questions or issues:
1. Check documentation above
2. Run: `./scripts/test-config-exhaustive.sh --help`
3. Review: Generated test reports

---

**Created**: 2025-12-13
**Part of**: Phase 2.2 - Exhaustive Configuration Testing
**Repository**: PAW Blockchain
