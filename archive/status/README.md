# PAW Blockchain - Network Status Page

A comprehensive, production-ready network status monitoring dashboard for the PAW blockchain network. Real-time health monitoring, incident management, performance metrics, and historical uptime tracking.

## Features

### 🔍 Real-Time Monitoring
- **Component Status**: Monitor Blockchain, API, WebSocket, Explorer, and Faucet services
- **Live Updates**: Automatic refresh every 30 seconds
- **Health Checks**: Automated endpoint monitoring with response time tracking
- **Status Indicators**: Clear visual status (Operational, Degraded, Down)

### 📊 Performance Metrics
- **Transactions Per Second (TPS)**: Real-time transaction throughput
- **Block Time**: Average block creation time
- **Connected Peers**: Number of active network peers
- **API Response Time**: REST API endpoint latency
- **Network Statistics**: Block height, validators, hash rate
- **Interactive Charts**: Chart.js visualizations with historical data

### 🚨 Incident Management
- **Active Incidents**: Display current service disruptions
- **Incident Timeline**: Complete history of past incidents
- **Severity Levels**: Critical, Major, Minor classifications
- **Status Updates**: Real-time incident progress tracking
- **Automated Detection**: Automatic incident creation based on health checks
- **Manual Posting**: Admin interface for manual incident reporting

### 📈 Uptime Tracking
- **30-Day History**: Visual uptime calendar
- **Percentage Calculation**: Overall uptime percentage
- **Per-Component Tracking**: Individual component uptime metrics
- **Historical Analysis**: Long-term reliability trends

### 🔔 Notifications & Subscriptions
- **Email Subscriptions**: Subscribe to status updates
- **RSS Feed**: Machine-readable status feed
- **Webhook Support**: Integration with external monitoring tools
- **Customizable Alerts**: Configure notification preferences

### 🌐 Service Dependencies
- **Dependency Graph**: Visual representation of service relationships
- **Impact Analysis**: Understand cascading failures
- **SVG Visualization**: Interactive dependency mapping

### 📡 Public API
- **RESTful Endpoints**: Complete API for status data
- **JSON Responses**: Structured, machine-readable data
- **CORS Enabled**: Cross-origin resource sharing support
- **Rate Limiting Ready**: Production-grade API protection

## Architecture

```
status/
├── frontend/               # Web UI
│   ├── index.html         # Main dashboard page
│   ├── app.js             # Frontend application logic
│   └── styles.css         # Responsive styling
├── backend/               # Go backend server
│   ├── main.go           # Server entry point
│   ├── pkg/
│   │   ├── api/          # HTTP handlers
│   │   ├── config/       # Configuration management
│   │   ├── health/       # Health monitoring
│   │   ├── incidents/    # Incident management
│   │   └── metrics/      # Metrics collection
├── tests/                 # Comprehensive tests
│   ├── unit/             # Unit tests
│   └── integration/      # Integration tests
├── docker-compose.yml     # Docker deployment
├── Dockerfile            # Container image
└── README.md             # This file
```

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
cd status/

# Start the status server
docker-compose up -d

# Access the dashboard
open http://localhost:8080
```

### Manual Setup

#### Prerequisites
- Go 1.21 or higher
- Modern web browser

#### Backend Setup

```bash
cd backend/

# Install dependencies
go mod download

# Run the server
go run main.go
```

#### Frontend Setup

The frontend is static HTML/CSS/JavaScript and is served by the backend server.

Simply open http://localhost:8080 in your browser.

## Configuration

### Environment Variables

```bash
# Server Configuration
PORT=8080                              # HTTP server port
MONITOR_INTERVAL=30s                   # Health check interval
METRICS_RETENTION=168h                 # Metrics retention (7 days)
INCIDENT_RETENTION=2160h               # Incident retention (90 days)

# Service Endpoints
BLOCKCHAIN_RPC_URL=http://localhost:26657
API_ENDPOINT=http://localhost:1317
WEBSOCKET_ENDPOINT=ws://localhost:26657/websocket
EXPLORER_ENDPOINT=http://localhost:3000
FAUCET_ENDPOINT=http://localhost:8000

# Email Notifications
ALERT_EMAIL=admin@pawchain.io
SMTP_SERVER=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=notifications@pawchain.io
SMTP_PASSWORD=your-password

# Webhooks
INCIDENT_WEBHOOK_URL=https://hooks.slack.com/services/your-webhook
```

### Docker Compose Configuration

Edit `docker-compose.yml` to customize:
- Port mappings
- Environment variables
- Volume mounts
- Network settings

## API Documentation

### Health Check
```
GET /api/v1/health
```
Returns basic health status of the status server itself.

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2025-01-19T12:00:00Z",
  "version": "1.0.0"
}
```

### Overall Status
```
GET /api/v1/status
```
Returns current status of all monitored components.

**Response:**
```json
{
  "overall_status": "operational",
  "message": "All systems operational",
  "components": [
    {
      "name": "Blockchain",
      "status": "operational",
      "description": "Core blockchain network",
      "uptime": "99.99%",
      "response_time": "45ms",
      "last_checked": "2025-01-19T12:00:00Z"
    }
  ],
  "updated_at": "2025-01-19T12:00:00Z"
}
```

### Incidents
```
GET /api/v1/incidents
```
Returns all incidents (active and history).

```
POST /api/v1/incidents
```
Create a new incident (admin only).

**Request:**
```json
{
  "title": "API Rate Limiting Issues",
  "description": "Some users experiencing rate limiting",
  "severity": "major",
  "components": ["API"]
}
```

```
GET /api/v1/incidents/{id}
```
Get a specific incident by ID.

```
POST /api/v1/incidents/{id}/update
```
Update an incident with new information.

**Request:**
```json
{
  "message": "Issue has been identified and fix is being deployed",
  "status": "identified"
}
```

### Metrics
```
GET /api/v1/metrics
```
Returns all collected metrics with historical data.

**Response:**
```json
{
  "tps": [
    {"timestamp": "2025-01-19T12:00:00Z", "value": 150.5}
  ],
  "block_time": [
    {"timestamp": "2025-01-19T12:00:00Z", "value": 6.5}
  ],
  "peers": [
    {"timestamp": "2025-01-19T12:00:00Z", "value": 42}
  ],
  "response_time": [
    {"timestamp": "2025-01-19T12:00:00Z", "value": 120}
  ],
  "network_stats": {
    "block_height": 1234567,
    "total_validators": 150,
    "active_validators": 125,
    "hash_rate": "1.2 TH/s"
  }
}
```

```
GET /api/v1/metrics/summary
```
Returns summary of current metrics.

### Status History
```
GET /api/v1/status/history?days=30
```
Returns uptime history for the specified number of days (max 90).

### Subscribe/Unsubscribe
```
POST /api/v1/subscribe
```
Subscribe to status update notifications.

**Request:**
```json
{
  "email": "user@domain.com",
  "preferences": {
    "incidents": true,
    "maintenance": true
  }
}
```

```
POST /api/v1/unsubscribe
```
Unsubscribe from notifications.

**Request:**
```json
{
  "email": "user@domain.com"
}
```

### RSS Feed
```
GET /api/v1/status/rss
```
Returns RSS feed of status updates in XML format.

## Testing

### Run Unit Tests

```bash
cd backend/
go test ./tests/unit/... -v
```

### Run Integration Tests

```bash
cd backend/
go test ./tests/integration/... -v
```

### Run All Tests

```bash
cd backend/
go test ./... -v -cover
```

### Test Coverage

```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Monitoring Best Practices

### Component Health Checks

Each component should implement a `/health` endpoint that returns:
- HTTP 200 for healthy
- HTTP 503 for unhealthy
- Response time < 5 seconds

### Incident Response

1. **Detection**: Automated health checks detect issues
2. **Creation**: Incident automatically created or manually posted
3. **Investigation**: Team investigates and posts updates
4. **Resolution**: Issue resolved and incident closed
5. **Post-Mortem**: Analysis and preventive measures

### Uptime Calculation

Uptime percentage is calculated as:
```
(Successful Checks / Total Checks) * 100
```

Components are checked every 30 seconds by default.

## Deployment

### Production Deployment

1. **Configure Environment**: Set all required environment variables
2. **Enable HTTPS**: Use Nginx reverse proxy with SSL certificates
3. **Database**: Implement persistent storage for metrics and incidents
4. **Monitoring**: Set up alerts for the status page itself
5. **Backup**: Regular backups of incident and metric data

### High Availability

For production environments:
- Deploy multiple instances behind a load balancer
- Use shared database for state synchronization
- Implement Redis for distributed caching
- Set up monitoring for the status page itself

### Scaling

The status page is designed to be lightweight and can handle:
- Thousands of concurrent users
- Monitoring 10+ services
- 30-second check intervals
- 7-day metric retention

For larger deployments, consider:
- Increasing check intervals
- Implementing metric aggregation
- Using time-series database (InfluxDB, Prometheus)
- CDN for static assets

## Security

### Authentication

For production:
- Implement API key authentication for admin endpoints
- Use OAuth2 for user authentication
- Restrict incident creation to authorized users

### Data Protection

- HTTPS/TLS encryption for all communications
- Input validation and sanitization
- SQL injection prevention
- XSS protection
- CSRF tokens for state-changing operations

### Rate Limiting

Implement rate limiting for:
- API endpoints: 100 requests/minute per IP
- Subscription endpoints: 10 requests/hour per IP
- RSS feed: 60 requests/hour per IP

## Troubleshooting

### Status Server Won't Start

```bash
# Check if port is already in use
netstat -an | grep 8080

# Check logs
docker-compose logs status-server

# Verify configuration
docker-compose config
```

### Metrics Not Updating

1. Check service endpoints are accessible
2. Verify CORS configuration
3. Check browser console for errors
4. Review server logs for health check failures

### High Memory Usage

- Reduce metrics retention period
- Decrease check interval
- Implement data aggregation
- Use database for historical data

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Write tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

Copyright (c) 2025 PAW Blockchain

## Support

For issues and questions:
- Email: support@pawchain.io
- Discord: https://discord.gg/pawchain

## Changelog

### Version 1.0.0 (2025-01-19)
- Initial release
- Real-time component monitoring
- Incident management system
- Performance metrics collection
- 30-day uptime tracking
- Email subscriptions
- RSS feed
- Public REST API
- Docker deployment
- Comprehensive test suite
