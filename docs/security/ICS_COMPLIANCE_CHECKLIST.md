# ICS Compliance Checklist (PAW)

- Scope: ICS-004 channel handshake and packet validation across DEX, Oracle, Compute.
- Channel ordering & version: shared `ChannelOpenValidator` enforces expected ordering + version on Init/Try/Ack; Compute mirrors the same rules manually.
- Port validation: Init and Try now reject mismatched ports for all modules (shared validator + Compute-specific guard) to block misrouted channels.
- Capability safety: channel capabilities are claimed on Init/Try to satisfy ICS-004 restart/recovery expectations.
- Packet validation: allowlisted ports/channels, nonce/timestamp presence, packet `ValidateBasic` enforced before processing; rejects zero nonce/timestamp.
- Ack size cap: acknowledgements over 1MB are refused before JSON unmarshal to avoid memory abuse.
- Memo guard: ante decorator caps transaction memo to 256 bytes to align with ICS-20 memo expectations.
- Observability: `ibc_packet_validation_failed` and `{module}_ibc_packet_validation_failed` events emitted on auth/data/nonce failures; compute unauthorized-channel test covers event emission.
- Metrics: validation failures increment telemetry counters labeled by port/channel/reason for Grafana/Prometheus dashboards.
- Tests: `go test ./x/shared/ibc/... ./x/dex/... ./x/oracle/... ./x/compute/...` (pass) covers handshake validators, packet validation, and module IBC handlers.
