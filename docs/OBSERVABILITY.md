# Observability and Health Checks

## Prometheus
- Enable in `~/.paw/config/config.toml`: `prometheus = true`, `prometheus_listen_addr = ":26660"`.
- Protect metrics behind firewall or a reverse proxy; do not expose publicly without auth.
- Reference dashboards/alerts: see `docs/METRICS.md` for Prometheus config, alert rules, and Grafana dashboards.

## Logging
- Default `log_format = "json"` in `config/node-config.toml.template` for aggregation.
- Recommend logrotate (example):
  ```
  /var/log/pawd.log {
    daily
    rotate 7
    compress
    missingok
    copytruncate
  }
  ```
  Point `pawd` output to `/var/log/pawd.log` (systemd unit) to activate rotation.

## Health Checks
- Script: `scripts/health-check.sh` (uses curl + jq).
  - Env: `RPC_ENDPOINT` (default `http://127.0.0.1:26657`), `MAX_LAG_BLOCKS` (default `3`), `TIMEOUT` (default `5`).
  - Probes `/status`, `/net_info`, `/validators`; fails if catching up, zero peers/validators, or height too low.
  - Example:
    ```bash
    RPC_ENDPOINT=http://127.0.0.1:26657 ./scripts/health-check.sh
    ```
- For remote scrapes, front with TLS/auth; do not expose raw RPC publicly.
