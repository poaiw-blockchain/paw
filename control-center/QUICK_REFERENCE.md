# Control Center Quick Reference

## Start Control Center

```bash
cd control-center
export JWT_SECRET="your-secure-secret"
docker-compose up -d
```

## Access Points

| Service | URL | Default Login |
|---------|-----|---------------|
| Control Center UI | http://localhost:11200 | admin@paw.network / admin123 |
| Backend API | http://localhost:11201 | - |
| WebSocket | ws://localhost:11202 | - |
| Prometheus | http://localhost:11090 | - |
| Grafana | http://localhost:11030 | admin / admin |
| Alertmanager | http://localhost:11093 | - |

## API Quick Reference

### Login
```bash
curl -X POST http://localhost:11201/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@paw.network","password":"admin123"}'
```

### Pause Module
```bash
curl -X POST http://localhost:11201/api/admin/circuit-breaker/dex/pause \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"reason":"Emergency maintenance","auto_recover":true,"recover_in":60}'
```

### Get Audit Log
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:11201/api/admin/audit-log?limit=50"
```

## Role Permissions

| Role | Read | Test | Admin | Emergency |
|------|------|------|-------|-----------|
| Viewer | ✅ | ❌ | ❌ | ❌ |
| Operator | ✅ | ✅ | ❌ | ❌ |
| Admin | ✅ | ✅ | ✅ | ❌ |
| SuperAdmin | ✅ | ✅ | ✅ | ✅ |

## Circuit Breaker Commands

```bash
# Pause DEX
curl -X POST -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/circuit-breaker/dex/pause \
  -d '{"reason":"Market anomaly"}'

# Resume DEX
curl -X POST -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/circuit-breaker/dex/resume

# Get Status
curl -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/circuit-breaker/status
```

## Emergency Commands (Requires SuperAdmin + 2FA)

```bash
# Halt Chain
curl -X POST -H "Authorization: Bearer <token>" \
  -H "X-2FA-Code: 123456" \
  http://localhost:11201/api/admin/emergency/halt-chain \
  -d '{"reason":"Critical vulnerability"}'
```

## WebSocket Connection

```javascript
const ws = new WebSocket('ws://localhost:11202/ws/updates?token=<JWT>');

ws.onopen = () => {
  // Subscribe to channels
  ws.send(JSON.stringify({
    type: 'subscribe',
    channels: ['metrics', 'alerts', 'audit']
  }));
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log(message.type, message.data);
};
```

## Common Tasks

### Create New Admin User
```bash
curl -X POST -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/users \
  -d '{
    "email":"newadmin@paw.network",
    "password":"secure_pass_123",
    "role":"Admin"
  }'
```

### Update Module Parameters
```bash
curl -X POST -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/params/dex \
  -d '{
    "swap_fee":"0.003",
    "max_slippage":"0.05"
  }'
```

### Export Audit Log
```bash
curl -H "Authorization: Bearer <token>" \
  "http://localhost:11201/api/admin/audit-log/export?format=csv" > audit.csv
```

## Troubleshooting

### Service Not Starting
```bash
# Check logs
docker-compose logs control-center-backend

# Check health
curl http://localhost:11201/health
```

### Database Connection Failed
```bash
# Verify PostgreSQL is running
docker-compose ps postgres

# Check connection
docker-compose exec postgres psql -U paw -d paw_control_center -c "SELECT 1;"
```

### WebSocket Not Connecting
```bash
# Test WebSocket upgrade
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  http://localhost:11202/ws/updates?token=<JWT>
```

## Monitoring

### Check Metrics
```bash
# Prometheus
curl http://localhost:11090/api/v1/query?query=up

# Control Center Stats
curl -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/metrics
```

### View Alerts
```bash
curl -H "Authorization: Bearer <token>" \
  http://localhost:11201/api/admin/alerts
```

## Security Checklist

- [ ] Change default admin password
- [ ] Generate strong JWT secret (`openssl rand -base64 64`)
- [ ] Configure IP whitelist (ADMIN_WHITELIST env var)
- [ ] Enable HTTPS with reverse proxy
- [ ] Enable 2FA for SuperAdmin accounts
- [ ] Review audit logs weekly
- [ ] Backup audit logs daily
- [ ] Rotate JWT secret monthly
- [ ] Update dependencies regularly
- [ ] Monitor for suspicious activity

## Backup & Restore

### Backup Audit Log
```bash
docker-compose exec postgres pg_dump -U paw paw_control_center > backup.sql
```

### Restore from Backup
```bash
docker-compose exec -T postgres psql -U paw paw_control_center < backup.sql
```

## Documentation Links

- **Full User Guide**: [UNIFIED_CONTROL_CENTER_GUIDE.md](UNIFIED_CONTROL_CENTER_GUIDE.md)
- **Architecture**: [ARCHITECTURE.md](ARCHITECTURE.md)
- **Implementation Summary**: [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)
- **Main README**: [README.md](README.md)

## Support

- GitHub Issues: https://github.com/paw-chain/paw/issues
- Email: support@paw.network
- Documentation: [UNIFIED_CONTROL_CENTER_GUIDE.md](UNIFIED_CONTROL_CENTER_GUIDE.md)
