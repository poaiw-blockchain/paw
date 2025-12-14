# PAW Control Center - Quick Start

## 🚀 5-Minute Setup

### 1. Prerequisites
- Docker and docker-compose installed
- Ports 11200-11203 available

### 2. Configure
```bash
cd dashboards/control-center
cp .env.example .env

# Generate secure secrets
openssl rand -base64 32  # Use for JWT_SECRET
openssl rand -base64 24  # Use for REDIS_PASSWORD
openssl rand -base64 24  # Use for POSTGRES_PASSWORD

# Edit .env and paste the secrets
vim .env
```

### 3. Launch
```bash
# Option A: Minimal (Dashboard + Auth)
./start.sh minimal

# Option B: Full Stack (with blockchain)
./start.sh full
```

### 4. Access
- **Dashboard**: http://localhost:11200
- **Login**: `admin` / `admin123`
- **Change password immediately!**

## 🔐 Default Accounts

| User     | Password    | Role     | Permissions |
|----------|-------------|----------|-------------|
| admin    | admin123    | Admin    | Full access |
| operator | operator123 | Operator | Read/Write  |
| viewer   | viewer123   | Viewer   | Read-only   |

## 📊 Analytics

The dashboard integrates with 6 analytics endpoints:
- Network Health
- Transaction Volume
- DEX Analytics
- Address Growth
- Gas Analytics
- Validator Performance

Ensure explorer is running on port 11080 for analytics.

## 🔍 Health Checks

```bash
# Dashboard
curl http://localhost:11200/health.html

# Auth Service
curl http://localhost:11201/health

# All Services
./start.sh status
```

## 📝 Useful Commands

```bash
./start.sh minimal      # Start minimal setup
./start.sh full         # Start full stack
./start.sh stop         # Stop all services
./start.sh status       # Show status
./start.sh logs [svc]   # View logs
```

## 🛠️ Troubleshooting

**Dashboard not loading?**
```bash
docker-compose ps
docker-compose logs control-center
```

**Auth failing?**
```bash
curl http://localhost:11201/health
docker-compose logs auth-service
```

**Analytics not showing?**
```bash
curl http://localhost:11080/api/v1/analytics/network-health
```

## 📚 Full Documentation

- **Production Guide**: README.md
- **Deployment Checklist**: DEPLOYMENT_CHECKLIST.md
- **Implementation Details**: SUMMARY.md
- **User Guide**: USER_GUIDE.md

## 🔒 Security Reminder

⚠️ **BEFORE PRODUCTION:**
- [ ] Change ALL default passwords
- [ ] Set unique JWT_SECRET (32+ chars)
- [ ] Use strong Redis/PostgreSQL passwords
- [ ] Enable HTTPS
- [ ] Configure firewall

## 🎯 Ports Reference

| Service       | Port  | URL                        |
|---------------|-------|----------------------------|
| Dashboard     | 11200 | http://localhost:11200     |
| Auth API      | 11201 | http://localhost:11201     |
| Redis         | 11202 | localhost:11202            |
| PostgreSQL    | 11203 | localhost:11203            |
| Explorer      | 11080 | http://localhost:11080     |
| PAW RPC       | 11001 | http://localhost:11001     |
| PAW REST      | 11002 | http://localhost:11002     |

## 🤝 Support

Issues? Check:
1. DEPLOYMENT_CHECKLIST.md
2. README.md troubleshooting section
3. Service logs: `./start.sh logs <service>`

---

**Status**: Production Ready ✅
**Version**: 1.0.0
