# Third-Party Security Audit Plan

## Purpose
Codify how PAW engages external auditors (Trail of Bits, Halborn, etc.) before mainnet launches or major upgrades.

## Vendor Selection
- Maintain short list of vetted firms with Cosmos/Cosmos-SDK experience.
- Require:
  - Public portfolio or references
  - Insurance and disclosure policy
  - Clear NDA + full report sharing rights

## Audit Scope
- Core modules: `app`, `x/dex`, `x/oracle`, `x/compute`
- Infrastructure: `scripts/devnet`, `monitoring/`, logging stack
- Wallets (desktop/mobile/extension) during feature freeze
- Smart contracts / off-chain services if applicable

## Deliverables
1. Kickoff checklist (architecture docs, threat models, coverage reports)
2. Daily/weekly status updates
3. Final PDF report with:
   - Executive summary
   - Severity ratings (High/Medium/Low/Informational)
   - Proof-of-concept traces
   - Remediation recommendations
4. Signed attestation once fixes validated

## Process
1. **Preparation (T-4 weeks)**
   - Freeze code at commit `audit-X`.
   - Generate fresh docs bundle:
     - `docs/architecture/PROJECT_ARCHITECTURE.md`
     - `docs/guides/deployment/GOVERNANCE_TIMELOCKS.md`
     - `docs/guides/security/THIRD_PARTY_AUDIT_PLAN.md`
   - Provide `go test ./...` results and coverage report.

2. **Fieldwork (2–4 weeks)**
   - Auditors get SSH-less access via container images.
   - Bug bounty paused to avoid duplicate submissions.

3. **Remediation (2 weeks)**
   - Track findings in `SECURITY_AUDIT_TRACKER.md`.
   - Hotfix critical issues under embargo.

4. **Verification**
   - Auditors retest patches.
   - Publish sanitized report + changelog to community.

## Success Criteria
- No outstanding High severity findings before release.
- Medium issues require mitigation plan + monitoring.
- Low/Info can roll into backlog.

## Artifacts Checklist
- [ ] Architecture diagram (updated)
- [ ] Threat model doc
- [ ] Test coverage report
- [ ] Dafny/formal proofs (if applicable)
- [ ] Wallet UX threat matrix
- [ ] CI configuration summary

## Integration with Release Lifecycle
- Required milestone for every consensus-breaking upgrade.
- Document audit status in `docs/upgrades/<version>.md`.
- Link roadmap item to audit report hash to maintain traceability.
