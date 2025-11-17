# Quick Start Checklist

Follow these steps to get the MetaRGB microservices running quickly.

## ✅ Prerequisites Check

- [ ] Go 1.23+ installed (`go version`)
- [ ] Protocol Buffers compiler installed (`protoc --version`)
- [ ] Docker & Docker Compose installed (`docker --version`)
- [ ] Node.js 18+ installed (`node --version`)
- [ ] Make installed (`make --version`)

## ✅ Initial Setup (One-time)

### 1. Generate Protocol Buffer Files
```bash
make proto
```

### 2. Create Environment File
Create `.env` in project root:
```bash
KAVENEGAR_API_KEY=your_key
OAUTH_SERVER_URL=https://your-oauth-server.com
OAUTH_CLIENT_ID=your_client_id
OAUTH_CLIENT_SECRET=your_client_secret
PARSIAN_PIN=your_pin
FTP_HOST=ftp.metargb.com
FTP_USER=your_ftp_user
FTP_PASSWORD=your_ftp_password
FTP_BASE_URL=https://cdn.metargb.com/uploads
CORS_ORIGIN=http://localhost:3000,http://localhost:8080
NODE_ENV=development
```

### 3. Start Infrastructure
```bash
docker-compose up -d mysql redis
sleep 10  # Wait for MySQL to be ready
```

### 4. Import Database Schema
```bash
make import-schema
```

## ✅ Daily Development

### Start All Services
```bash
make dev
```

### Check Status
```bash
make ps
```

### View Logs
```bash
# All services
make logs

# Specific service
make logs-service SERVICE=auth-service
```

### Stop Services
```bash
make down
```

## ✅ Testing

### Run All Tests
```bash
make test-all
```

### Run Specific Test Suite
```bash
make test-unit        # Unit tests
make test-integration # Integration tests
make test-golden      # Golden JSON tests
make test-database    # Database tests
```

## ✅ Common Commands

```bash
# Build services
make build-all

# Restart service
make restart-service SERVICE=auth-service

# Clean everything
make clean

# Validate Kong config
make kong-validate
```

## ✅ Service Endpoints

- **Kong API Gateway**: http://localhost:8000
- **Kong Admin**: http://localhost:8001
- **WebSocket Gateway**: http://localhost:3000
- **MySQL**: localhost:3308
- **Redis**: localhost:6379

## ✅ Test API

```bash
# Health check
curl http://localhost:3000/health

# Auth endpoint (requires token)
curl http://localhost:8000/api/auth/me \
  -H "Authorization: Bearer your_token"
```

## ⚠️ Troubleshooting

### Services not starting?
```bash
docker-compose logs auth-service
```

### Database connection issues?
```bash
docker-compose exec mysql mysql -uroot -proot_password -e "SELECT 1"
```

### Port already in use?
```bash
lsof -i :50051  # Find process
# Kill process or change port
```

### Need to reset everything?
```bash
make clean
make dev
```

## 📚 Full Documentation

See `SETUP_GUIDE.md` for detailed instructions.

