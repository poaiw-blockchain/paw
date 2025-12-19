# IBC Boundary Triage Runbook

1) Confirm alert details: port/channel/reason from Grafana alert; check `ibc_packet_validation_failed` on dashboard (`ibc-boundary.json`).
2) Inspect logs: `journalctl -u pawd -n 200 | grep "ibc_packet_validation_failed"` for the same port/channel to see errors.
3) Validate channel bindings: ensure port/channel allowlist matches counterparty; verify version/ordering via `pawd q ibc channel channels`.
4) Check relayer: Hermes/relayer health, client/connection state, and packet sequence continuity; restart relayer if stale.
5) If reason is "not authorized": cross-check allowlists/capabilities; close unauthorized channel if needed and reopen with correct port/version.
6) If nonce/timestamp invalid: look for replay attempts or clock skew; verify counterparty timeouts and resubmit packets.
7) After mitigation: watch dashboard to confirm rate drops to zero; acknowledge/close alert in Grafana/Alertmanager.
