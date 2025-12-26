# PAW Operator Runbook

## Alert Quick Reference

| Alert | Severity | Dashboard | Response |
|-------|----------|-----------|----------|
| PAWConsensusStalled | critical | Consensus | Check logs, consensus_state |
| PAWLowPeerCount | warning | P2P | Add peers, restart |
| PAWValidatorDown | critical | Overview | Check pod status, logs |
| PAWHighMemory/Disk | warning | Resources | Scale or prune |
| IBCChannelDown | critical | IBC | Check hermes, counterparty |
| IBCRelayerDown | critical | IBC | Restart hermes deployment |

## Access

- **Grafana**: `http://<cluster>:11030`
- **Alertmanager**: `http://<cluster>:9093`
- **Prometheus**: `http://<cluster>:9090`

## Severity & Escalation

- **critical**: 15min response, PagerDuty + #paw-ibc-critical
- **warning**: 1hr response, #paw-alerts
- **info**: Next business day, email

## Common Commands

```bash
# Check validator logs
kubectl logs -l app=paw-validator -n paw-blockchain --tail=100

# Check peer count
kubectl exec -it <pod> -n paw-blockchain -- curl localhost:26657/net_info | jq '.result.n_peers'

# Check IBC relayer
kubectl logs -l app=hermes -n paw-blockchain --tail=100

# Check disk usage
kubectl exec -it <pod> -- df -h /root/.paw

# Check consensus
curl http://localhost:26657/consensus_state | jq
```

## Config Files

- **Alert rules**: `k8s/monitoring-configmaps.yaml`, `k8s/prometheus-genesis-rules.yaml`
- **Receivers**: `k8s/monitoring-configmaps.yaml` (Slack, PagerDuty, Email)
- **Full troubleshooting**: `docs/TROUBLESHOOTING.md`
