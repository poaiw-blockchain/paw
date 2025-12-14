# PAW Unified Control Center

A comprehensive operational control center integrating all monitoring, administrative, and testing capabilities for the PAW blockchain network.

## Features

- **Role-Based Access Control (RBAC)**: 4 role levels (Viewer, Operator, Admin, SuperAdmin)
- **Real-Time Monitoring**: Live metrics, alerts, and network health via WebSocket
- **Module Controls**: Circuit breakers for DEX, Oracle, and Compute modules
- **Emergency Controls**: Chain halt, maintenance mode, forced upgrades
- **Audit Logging**: Complete immutable trail of all admin actions
- **Alert Management**: Centralized alert routing from Alertmanager
- **Grafana Integration**: Embedded dashboards for visualization
- **Testing Tools**: Inherited from archived testing dashboard

## Quick Start

```bash
# Clone repository
cd control-center

# Set JWT secret
export JWT_SECRET="your-secure-random-secret"

# Start all services
docker-compose up -d

# Access Control Center
open http://localhost:11200
```

**Default Login:**
- Email: `admin@paw.network`
- Password: `admin123`
- **CHANGE IMMEDIATELY!**

## Architecture

```
┌─────────────────────────────────────────────┐
│          Frontend (Port 11200)              │
│  HTML/CSS/JS - Based on Testing Dashboard  │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│        Backend API (Port 11201)             │
│  Go - Auth, Admin API, Audit, WebSocket    │
└─────────────────────────────────────────────┘
        ↓           ↓           ↓
┌───────────┐ ┌──────────┐ ┌──────────┐
│ PostgreSQL│ │  Redis   │ │Prometheus│
│ Audit Log │ │ Sessions │ │ Metrics  │
└───────────┘ └──────────┘ └──────────┘
```

## Documentation

- **[UNIFIED_CONTROL_CENTER_GUIDE.md](UNIFIED_CONTROL_CENTER_GUIDE.md)** - Complete user guide
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Technical architecture details
- **[API.md](API.md)** - API reference (if created)

## Components

### Backend (`/backend`)
- **Go 1.22+**
- Authentication (JWT + RBAC)
- Admin API (parameters, circuit breakers, emergency controls)
- Audit logging service
- WebSocket server
- Integration with Prometheus, Grafana, Alertmanager

### Frontend (`/frontend`)
- **HTML/CSS/JavaScript** (no build step)
- Based on existing testing dashboard
- Real-time updates via WebSocket
- Module control panels
- Embedded Grafana dashboards

### Configuration (`/config`)
- Prometheus scrape configs
- Grafana dashboard provisioning
- Alertmanager routing rules
- Database initialization scripts

### Deployment (`/deployments`)
- Docker Compose for local/dev
- Kubernetes manifests for production (if created)
- Terraform for cloud deployment (if created)

## Security

**CRITICAL: Follow these steps for production:**

1. Change default admin password
2. Generate strong JWT secret: `openssl rand -base64 64`
3. Configure IP whitelist for admin access
4. Enable HTTPS (use nginx reverse proxy)
5. Enable 2FA for SuperAdmin accounts
6. Regular audit log reviews and backups
7. Set up alerts for suspicious activity

## API Endpoints

### Public (No Auth)
- `POST /api/auth/login` - User login
- `GET /api/blocks` - Recent blocks (read-only)
- `GET /api/validators` - Validator list

### Admin (Auth Required)
- `GET/POST /api/admin/params/:module` - Module parameters
- `POST /api/admin/circuit-breaker/:module/pause` - Pause module
- `POST /api/admin/emergency/halt-chain` - Emergency halt (SuperAdmin + 2FA)
- `GET /api/admin/audit-log` - Query audit log

### WebSocket
- `WS /ws/updates?token=<JWT>` - Real-time updates

See [UNIFIED_CONTROL_CENTER_GUIDE.md](UNIFIED_CONTROL_CENTER_GUIDE.md) for complete API reference.

## Development

### Prerequisites
- Go 1.22+
- Docker and Docker Compose
- PostgreSQL 16+
- Redis 7+

### Build Backend

```bash
cd backend
go mod download
go build -o control-center-api .
./control-center-api
```

### Run Tests

```bash
cd backend
go test ./... -v
```

### Frontend Development

The frontend uses vanilla JavaScript (no build step):

```bash
cd frontend
# Serve with any static server
python3 -m http.server 8080
```

## Monitoring

The Control Center monitors itself:

- **Prometheus**: http://localhost:11090
- **Grafana**: http://localhost:11030
- **Alertmanager**: http://localhost:11093

Metrics exposed:
- API request rate and latency
- WebSocket connection count
- Circuit breaker state changes
- Audit log entry rate
- Authentication success/failure rate

## Troubleshooting

### Backend won't start

```bash
# Check logs
docker-compose logs control-center-backend

# Common issues:
# - DATABASE_URL incorrect
# - JWT_SECRET not set
# - Ports already in use
```

### WebSocket not connecting

```bash
# Test WebSocket
curl -i -N \
  -H "Connection: Upgrade" \
  -H "Upgrade: websocket" \
  -H "Host: localhost:11202" \
  -H "Origin: http://localhost:11200" \
  http://localhost:11202/ws/updates?token=<JWT>
```

### Audit log not recording

```bash
# Check database
docker-compose exec postgres psql -U paw -d paw_control_center
SELECT COUNT(*) FROM audit_log;
```

See [UNIFIED_CONTROL_CENTER_GUIDE.md](UNIFIED_CONTROL_CENTER_GUIDE.md) for more troubleshooting.

## Roadmap

- [ ] Multi-tenancy support
- [ ] Advanced analytics (ML-based anomaly detection)
- [ ] Custom dashboard builder
- [ ] Automated incident response
- [ ] Mobile app
- [ ] Distributed tracing integration
- [ ] Compliance reporting (SOC2, ISO27001)

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new features
4. Submit a pull request

## License

MIT License - see LICENSE file for details.

## Support

- Documentation: [UNIFIED_CONTROL_CENTER_GUIDE.md](UNIFIED_CONTROL_CENTER_GUIDE.md)
- Issues: https://github.com/paw-chain/paw/issues
- Email: support@paw.network

---

**Built with ❤️ for the PAW Community**
