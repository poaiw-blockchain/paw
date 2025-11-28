# Docker Compose Configurations

This directory contains all Docker Compose configuration files for the PAW blockchain project.

## Available Compose Files

### docker-compose.yml (Main Configuration)
The primary Docker Compose file for running the PAW blockchain node in production mode.

**Services:**
- PAW blockchain node
- Core blockchain infrastructure

**Usage:**
```bash
docker-compose up -d
```

### docker-compose.dev.yml
Development environment configuration with additional tools and debugging capabilities.

**Features:**
- Hot reload support
- Debugging ports exposed
- Development dependencies
- Volume mounts for live code updates

**Usage:**
```bash
docker-compose -f docker-compose.yml -f docker-compose.dev.yml up -d
```

### docker-compose.devnet.yml
Devnet (development network) configuration for multi-node testing.

**Features:**
- Multiple blockchain nodes
- Simulated network environment
- Testing infrastructure

**Usage:**
```bash
docker-compose -f docker-compose.devnet.yml up -d
```

### docker-compose.monitoring.yml
Monitoring and observability stack configuration.

**Services:**
- Prometheus (metrics collection)
- Grafana (visualization)
- Alert Manager
- Log aggregation

**Usage:**
```bash
# Run with main services
docker-compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d

# Run monitoring stack standalone
docker-compose -f docker-compose.monitoring.yml up -d
```

## Common Commands

### Start all services
```bash
docker-compose up -d
```

### Stop all services
```bash
docker-compose down
```

### View logs
```bash
docker-compose logs -f
```

### Rebuild and restart
```bash
docker-compose up -d --build
```

## Combining Compose Files

You can combine multiple compose files for different scenarios:

```bash
# Development with monitoring
docker-compose -f docker-compose.yml -f docker-compose.dev.yml -f docker-compose.monitoring.yml up -d

# Production with monitoring
docker-compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d
```

## Notes

- Run commands from either the project root or the `compose/` directory
- Environment variables can be configured in `.env` file in the project root
- For production deployments, use `docker-compose.yml` as the base configuration
