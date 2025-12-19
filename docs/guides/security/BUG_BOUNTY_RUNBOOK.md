# PAW Bug Bounty Operations Runbook

This runbook documents the canonical workflow for running the PAW Blockchain bug bounty program from launch preparation through payout and metrics. It ties together the public-facing policy in `docs/BUG_BOUNTY.md`, the tooling under `scripts/bug-bounty/`, and the private coordination channels used by the core security team.

---

## 1. Program Launch Checklist

| Item | Description | Owners |
| --- | --- | --- |
| ✅ Scope Review | Confirm `docs/BUG_BOUNTY.md` accurately reflects current repos/modules, reward tiers, and exclusions. Update before launch. | Module leads + Security |
| ✅ Contact Channels | Provision `security@pawchain.io`, `bugbounty@pawchain.io`, and the encrypted PGP mailbox documented in `archive/bug-bounty/PGP_KEY_SETUP.md`. | Security Ops |
| ✅ Submission Intake | Deploy `scripts/bug-bounty/validate-submission.sh` to the hardened triage host. Wire into helpdesk or run ad-hoc for each report. | Security Ops |
| ✅ Reward Wallets | Top up the dedicated stablecoin + PAW token multisig. Minimum runway: 2× projected max payout ($200k). | Finance + Security |
| ✅ Disclosure Policy | Link `docs/BUG_BOUNTY.md` from `SECURITY.md`, portal FAQ, and status site so researchers have a single source of truth. | DevRel |

Once all boxes are checked, announce the live program through:
1. Blog + forum post referencing `docs/BUG_BOUNTY.md`
2. Status page banner linking to `docs/BUG_BOUNTY.md#reporting`
3. Validator + partner email blast with disclosure expectations

---

## 2. Intake & Triage Workflow

1. **Acknowledge** within 12 hours via `security@` using the template from `archive/bug-bounty/TRIAGE_PROCESS.md`.
2. **Validate Submission Quality**:
   ```bash
   cd scripts/bug-bounty
   ./validate-submission.sh /path/to/report.md
   ```
   Resolve any checklist gaps before starting engineering review.
3. **Assign Severity Owner** based on module (DEX/Oracle/Compute/Core). Log assignment in the private tracker (Notion or Jira).
4. **Reproduce** in a clean environment (see `tests/chaos/` + `tests/security/`). Capture logs and PoC steps in the triage record.
5. **Decide Severity** using:
   - [`archive/bug-bounty/SEVERITY_MATRIX.md`] for baseline scoring
   - [`docs/BUG_BOUNTY.md#severity-classification`] for reward tiers
6. **Patch & Verify**: Engage module owners for fix + tests. Reference existing security test harnesses (`tests/security/`, `tests/chaos/`, fuzz suites).
7. **Coordinate Disclosure**: Work with the reporter on patch timelines and public disclosure target (default 30 days, extendable).

All status transitions must be mirrored in the internal tracker and appended to the report thread for transparency.

---

## 3. Payout & Communication

| Step | Details |
| --- | --- |
| Reward Determination | Use severity score + exploit sophistication. Optional 10% bonus for high-quality PoCs. |
| Finance Approval | File payout request with Finance referencing case ID, amount, currency (USDC or PAW), and destination wallet. |
| Legal Review | For critical cases, Legal verifies export controls and sanctions compliance before transferring funds. |
| Execution | Release funds from multisig; attach on-chain tx hash to the ticket. |
| Hall of Fame | Add qualifying researchers to the Hall-of-Fame section in `docs/BUG_BOUNTY.md` + community blog post. |

Notify the researcher when payout is initiated and again when confirmed on-chain. Templates live in `archive/bug-bounty/TRIAGE_PROCESS.md#reward-notifications`.

---

## 4. Metrics & Reporting

Track the following KPIs monthly and after major releases:

- Submissions received / accepted / rejected
- Mean acknowledgment and remediation times
- Total payouts + average bounty size
- Module distribution (DEX/Oracle/Compute/Core/Wallet)
- Severity distribution
- Coverage of mitigations (tests added, monitoring, docs)

Publish a quarterly summary to `docs/guides/security/THREAT_INTEL_REPORT.md` (create if missing) and include highlights in `TESTNET_STATUS.md` to keep validators informed.

---

## 5. Ongoing Maintenance

1. **Scope Drift**: When new services (wallets, explorers, bridges) ship, update `docs/BUG_BOUNTY.md` and announce scope changes to avoid invalid submissions.
2. **Tooling Upgrades**: Extend `scripts/bug-bounty/validate-submission.sh` with new module templates and integrate with CI so triage metadata stays standardized.
3. **Chaos / Stress Alignment**: Feed high-value bug bounty findings into `tests/chaos/` and `tests/stress/` suites to prevent regressions.
4. **Program Pauses**: If executing upgrades or responding to incidents, temporarily pause the bounty program by:
   - Updating `docs/BUG_BOUNTY.md` and status page
   - Emailing the bug bounty list
   - Recording the pause window in this runbook
5. **Third-Party Coordination**: Align the bug bounty schedule with third-party audits (`docs/guides/security/THIRD_PARTY_AUDIT_PLAN.md`) so findings do not collide (audits take precedence).

---

## 6. Contact Matrix

| Role | Responsibility | Contact |
| --- | --- | --- |
| Security Lead | Final severity decisions, disclosure coordination | `security-lead@pawchain.io` |
| Module Liaisons | Implement fixes within DEX/Oracle/Compute/Core | `<module>-lead@pawchain.io` |
| Finance | Multisig payouts | `finance@pawchain.io` |
| DevRel | External announcements & Hall of Fame updates | `devrel@pawchain.io` |
| Legal | Export/sanctions review | `legal@pawchain.io` |

Keep this matrix synced with the org chart and update quarterly.

---

## References

- Public program policy: `docs/BUG_BOUNTY.md`
- Severity guidelines: `archive/bug-bounty/SEVERITY_MATRIX.md`
- Triage process templates: `archive/bug-bounty/TRIAGE_PROCESS.md`
- PGP setup guide: `archive/bug-bounty/PGP_KEY_SETUP.md`
- Submission tooling: `scripts/bug-bounty/validate-submission.sh`
- Portal FAQ entry: `archive/portal/content/faq.md`

Maintaining this runbook as a living document ensures our bug bounty program stays operationally mature as we head into the public testnet and mainnet launches.
