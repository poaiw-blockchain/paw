# Loki Query Examples for PAW Blockchain

## Basic Queries

### View All PAW Logs
```logql
{job="paw-docker"}
```

### Specific Container
```logql
{container_name="paw-node1"}
```

### Multiple Containers
```logql
{container_name=~"paw-node[1-4]"}
```

### By Service Name
```logql
{service_name="prometheus"}
```

## Log Level Filtering

### Errors Only
```logql
{job="paw-docker"} |= "ERROR"
```

### Warnings and Errors
```logql
{job="paw-docker"} |~ "(?i)(ERROR|WARN)"
```

### Case-Insensitive Critical Logs
```logql
{job="paw-docker"} |~ "(?i)(panic|fatal|critical|error)"
```

### Exclude Debug Logs
```logql
{job="paw-docker"} != "DEBUG"
```

### JSON Parsed Level
```logql
{job="paw-docker"} | json | level="ERROR"
```

## Blockchain-Specific Queries

### Consensus Logs
```logql
{job="paw-blockchain"} |~ "height=\\d+ round=\\d+"
```

### Transaction Logs
```logql
{job="paw-blockchain"} |~ "tx_hash=[A-Fa-f0-9]{64}"
```

### Validator Activity
```logql
{job="paw-blockchain"} |~ "validator"
```

### Block Commits
```logql
{job="paw-blockchain"} |~ "Committed block"
```

### Peer Connections
```logql
{job="paw-blockchain"} |~ "peer=(\\w+)"
```

### Mempool Activity
```logql
{job="paw-blockchain"} |~ "(?i)mempool"
```

## Rate and Count Queries

### Error Rate (5 minutes)
```logql
rate({job="paw-docker"} |= "ERROR" [5m])
```

### Errors Per Hour by Container
```logql
sum by(container_name) (count_over_time({job="paw-docker"} |= "ERROR" [1h]))
```

### Log Volume by Service
```logql
sum by(service_name) (rate({job="paw-docker"}[5m]))
```

### Count Unique Errors
```logql
count_over_time({job="paw-docker"} |= "ERROR" [1h])
```

## Pattern Extraction

### Extract Transaction Hashes
```logql
{job="paw-blockchain"}
  | regexp "tx_hash=(?P<tx>[A-Fa-f0-9]{64})"
```

### Extract Block Heights
```logql
{job="paw-blockchain"}
  | regexp "height=(?P<height>\\d+)"
```

### Parse JSON Fields
```logql
{job="paw-docker"}
  | json
  | line_format "{{.level}} - {{.msg}}"
```

### Extract Module Names
```logql
{job="paw-docker"}
  | json module
  | module != ""
```

## Time-Based Queries

### Last 15 Minutes
```logql
{container_name="paw-node1"} [15m]
```

### Specific Time Range (in Grafana UI)
Set time picker to custom range, then:
```logql
{job="paw-docker"} |= "ERROR"
```

### Events Per Minute
```logql
sum(count_over_time({job="paw-docker"} [1m]))
```

## Label Filtering

### Filter by Multiple Labels
```logql
{job="paw-blockchain", container_name="paw-node1"}
```

### Exclude Specific Container
```logql
{job="paw-docker", container_name!="paw-prometheus"}
```

### Multiple Job Types
```logql
{job=~"paw-blockchain|paw-monitoring"}
```

## Aggregations

### Total Logs by Container (1h)
```logql
topk(10, sum by(container_name) (count_over_time({job="paw-docker"}[1h])))
```

### Error Percentage
```logql
sum(count_over_time({job="paw-docker"} |= "ERROR" [5m]))
/
sum(count_over_time({job="paw-docker"} [5m]))
```

### Logs per Second
```logql
sum(rate({job="paw-docker"} [1m]))
```

## Advanced Filtering

### Multi-Stage Pipeline
```logql
{job="paw-blockchain"}
  | json
  | level="ERROR"
  | module="consensus"
```

### Regex with Negative Lookahead
```logql
{job="paw-docker"}
  |~ "ERROR"
  != "healthcheck"
```

### Line Format Transform
```logql
{job="paw-docker"}
  | json
  | line_format "{{.timestamp}} {{.level}} [{{.module}}] {{.msg}}"
```

### Label Format
```logql
{job="paw-docker"}
  | label_format level=`{{.extracted_level}}`
```

## Performance Monitoring

### Slow Queries
```logql
{job="paw-blockchain"}
  |~ "query took"
  | regexp "took (?P<duration>\\d+\\.\\d+)s"
  | duration > 1.0
```

### High Memory Usage
```logql
{job="paw-monitoring"}
  |~ "memory usage"
  | regexp "usage: (?P<mem>\\d+)%"
  | mem > 80
```

### Network Errors
```logql
{job="paw-blockchain"}
  |~ "(?i)(connection|network|timeout)"
  |= "ERROR"
```

## Security & Audit

### Authentication Events
```logql
{job="paw-docker"}
  |~ "(?i)(auth|login|logout)"
```

### Failed Transactions
```logql
{job="paw-blockchain"}
  |~ "tx failed"
  | json
```

### Access Denied
```logql
{job="paw-docker"}
  |~ "(?i)(denied|unauthorized|forbidden)"
```

## Dashboard Query Examples

### Error Timeline Panel
```logql
sum by (container_name) (
  count_over_time({job="paw-docker"} |= "ERROR" [1m])
)
```

### Log Stream Panel
```logql
{container_name=~"paw-node.*"}
  | json
  | level != "DEBUG"
```

### Top Errors Table
```logql
topk(20, sum by (error) (
  count_over_time({job="paw-docker"}
    | json
    | error != "" [24h])
))
```

## Troubleshooting Queries

### Missing Logs from Container
```logql
{container_name="paw-node1"} | count_over_time([5m])
```

### Check Log Gaps
```logql
absent_over_time({container_name="paw-node1"}[5m])
```

### Promtail Status
```logql
{container_name="paw-promtail"}
```

## Query Best Practices

1. **Use Label Filters First**: Start with `{job="..."}` for performance
2. **Limit Time Range**: Narrow queries to specific time windows
3. **Combine Filters**: Use multiple stages for complex filtering
4. **Use Regex Sparingly**: Line filters (`|=`) are faster than regex (`|~`)
5. **Index Labels**: Leverage indexed labels (job, container_name, service_name)
6. **Aggregate Before Visualize**: Use `sum`, `count_over_time` for dashboards
7. **Test with Limits**: Add `| limit 100` while developing queries

## Testing in CLI

```bash
# Query recent logs
curl -G -s "http://localhost:11025/loki/api/v1/query" \
  --data-urlencode 'query={job="paw-docker"}' \
  --data-urlencode 'limit=10' | jq

# Query range with time
curl -G -s "http://localhost:11025/loki/api/v1/query_range" \
  --data-urlencode 'query={job="paw-docker"}' \
  --data-urlencode 'start=now-1h' \
  --data-urlencode 'end=now' | jq

# Check labels
curl -s "http://localhost:11025/loki/api/v1/labels" | jq

# Get label values
curl -s "http://localhost:11025/loki/api/v1/label/container_name/values" | jq
```
