# MetaRGB Microservices - Setup & Testing Guide

Complete guide for setting up, running, and testing the MetaRGB microservices project.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Development Setup (Docker Compose)](#development-setup-docker-compose)
3. [Local Development (Without Docker)](#local-development-without-docker)
4. [Configuration Files](#configuration-files)
5. [Running Services](#running-services)
6. [Testing](#testing)
7. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

```bash
# 1. Go (Golang) 1.23 or higher
# Download from: https://go.dev/dl/
go version  # Should show 1.23+

# 2. Protocol Buffers Compiler
# macOS:
brew install protobuf

# Linux (Ubuntu/Debian):
sudo apt-get install protobuf-compiler

# Verify:
protoc --version  # Should show 3.x or higher

# 3. Docker & Docker Compose
# Download from: https://www.docker.com/products/docker-desktop
docker --version
docker-compose --version

# 4. MySQL 8.0 (if running locally without Docker)
# macOS:
brew install mysql@8.0

# Linux:
sudo apt-get install mysql-server-8.0

# 5. Redis (if running locally without Docker)
# macOS:
brew install redis

# Linux:
sudo apt-get install redis-server

# 6. Node.js 18+ (for WebSocket gateway)
# Download from: https://nodejs.org/
node --version  # Should show v18+

# 7. Make (usually pre-installed)
make --version

# 8. Optional: k6 (for load testing)
# macOS:
brew install k6

# Linux:
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg \
  --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | \
  sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### Required External Services/APIs

You'll need API keys/credentials for:
- **OAuth Server** (for authentication)
- **Kavenegar API** (for SMS notifications)
- **Parsian Payment Gateway** (for payments)
- **FTP Server** (for file storage)

---

## Development Setup (Docker Compose)

### Step 1: Clone and Navigate

```bash
cd /Applications/XAMPP/xamppfiles/htdocs/metargb-microservices
```

### Step 2: Generate Protocol Buffer Files

```bash
# Generate all proto files
make proto

# Or generate individually:
make gen-auth
make gen-commercial
make gen-features
make gen-levels
make gen-dynasty
make gen-calendar
make gen-storage
```

### Step 3: Create Environment File

Create a `.env` file in the project root (or copy from example if available):

```bash
# .env file for docker-compose
KAVENEGAR_API_KEY=your_kavenegar_api_key
OAUTH_SERVER_URL=https://your-oauth-server.com
OAUTH_CLIENT_ID=your_oauth_client_id
OAUTH_CLIENT_SECRET=your_oauth_client_secret
PARSIAN_PIN=your_parsian_pin
FTP_HOST=ftp.metargb.com
FTP_PORT=21
FTP_USER=your_ftp_user
FTP_PASSWORD=your_ftp_password
FTP_BASE_URL=https://cdn.metargb.com/uploads
CORS_ORIGIN=http://localhost:3000,http://localhost:8080
NODE_ENV=development
```

### Step 4: Start Infrastructure Services

```bash
# Start MySQL and Redis only
docker-compose up -d mysql redis

# Wait for services to be healthy (about 10-15 seconds)
docker-compose ps
```

### Step 5: Import Database Schema

```bash
# Wait for MySQL to be ready
sleep 10

# Import schema
make import-schema

# Or manually:
docker exec -i metargb-mysql mysql -uroot -proot_password metargb_db < scripts/schema.sql
```

### Step 6: Start All Services

```bash
# Start all services
make dev

# Or using docker-compose directly:
docker-compose up -d

# Check service status
make ps

# View logs
make logs

# Or view logs for specific service:
make logs-service SERVICE=auth-service
```

### Step 7: Verify Services

```bash
# Check all services are running
docker-compose ps

# Test Kong API Gateway
curl http://localhost:8000/api/auth/me

# Test WebSocket Gateway
curl http://localhost:3000/health

# Check service health
docker-compose exec auth-service nc -z localhost 50051 && echo "Auth service OK"
```

---

## Local Development (Without Docker)

### Step 1: Setup Database

```bash
# Start MySQL
mysql.server start  # macOS
# or
sudo systemctl start mysql  # Linux

# Create database
mysql -u root -p << EOF
CREATE DATABASE metargb_db;
CREATE USER 'metargb_user'@'localhost' IDENTIFIED BY 'metargb_password';
GRANT ALL PRIVILEGES ON metargb_db.* TO 'metargb_user'@'localhost';
FLUSH PRIVILEGES;
EOF

# Import schema
mysql -u root -p metargb_db < scripts/schema.sql
```

### Step 2: Setup Redis

```bash
# Start Redis
redis-server  # macOS/Linux

# Or as service:
brew services start redis  # macOS
sudo systemctl start redis  # Linux

# Test connection
redis-cli ping  # Should return PONG
```

### Step 3: Configure Each Service

For each service, create a `config.env` file from the sample:

```bash
# Auth Service
cd services/auth-service
cp config.env.sample config.env
# Edit config.env with your settings

# Commercial Service
cd ../commercial-service
cp config.env.sample config.env
# Edit config.env

# Repeat for other services...
```

### Step 4: Generate Proto Files

```bash
# From project root
make proto
```

### Step 5: Run Services Individually

Open multiple terminal windows:

```bash
# Terminal 1: Auth Service
cd services/auth-service
go run cmd/server/main.go

# Terminal 2: Commercial Service
cd services/commercial-service
go run cmd/server/main.go

# Terminal 3: Features Service
cd services/features-service
go run cmd/server/main.go

# Terminal 4: Levels Service
cd services/levels-service
go run cmd/server/main.go

# Terminal 5: Dynasty Service
cd services/dynasty-service
go run cmd/server/main.go

# Terminal 6: Calendar Service
cd services/calendar-service
go run cmd/server/main.go

# Terminal 7: Storage Service
cd services/storage-service
go run cmd/server/main.go

# Terminal 8: WebSocket Gateway
cd websocket-gateway
npm install
npm start
```

---

## Configuration Files

### Service Configuration Templates

Each service has a `config.env.sample` file. Copy and customize:

#### Auth Service (`services/auth-service/config.env`)
```bash
DB_HOST=localhost
DB_PORT=3306
DB_USER=metargb_user
DB_PASSWORD=metargb_password
DB_DATABASE=metargb_db

OAUTH_SERVER_URL=https://oauth.example.com
OAUTH_CLIENT_ID=your-client-id
OAUTH_CLIENT_SECRET=your-client-secret

GRPC_PORT=50051

REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

KAVENEGAR_API_KEY=your_kavenegar_key
```

#### Commercial Service (`services/commercial-service/config.env`)
```bash
DB_HOST=localhost
DB_PORT=3306
DB_USER=metargb_user
DB_PASSWORD=metargb_password
DB_DATABASE=metargb_db

PARSIAN_MERCHANT_ID=your_merchant_id
PARSIAN_PIN=your_pin
PARSIAN_CALLBACK_URL=https://your-domain.com/api/parsian/callback
PARSIAN_LOAN_ACCOUNT_MERCHANT_ID=your_loan_merchant_id
PARSIAN_LOAN_ACCOUNT_PIN=your_loan_pin

GRPC_PORT=50052
```

#### Storage Service (`services/storage-service/config.env`)
```bash
GRPC_PORT=50059
HTTP_PORT=8059

DB_HOST=localhost
DB_PORT=3306
DB_DATABASE=metargb_db
DB_USER=metargb_user
DB_PASSWORD=metargb_password

FTP_HOST=ftp.metargb.com
FTP_PORT=21
FTP_USER=metargb_uploads
FTP_PASSWORD=ftp_password
FTP_BASE_PATH=/uploads
FTP_BASE_URL=https://cdn.metargb.com/uploads

TEMP_DIR=/tmp/storage-chunks
```

### Docker Compose Environment Variables

Create `.env` in project root:

```bash
# Database (used by docker-compose.yml)
MYSQL_ROOT_PASSWORD=root_password
MYSQL_DATABASE=metargb_db
MYSQL_USER=metargb_user
MYSQL_PASSWORD=metargb_password

# External APIs
KAVENEGAR_API_KEY=your_key
OAUTH_SERVER_URL=https://oauth.example.com
OAUTH_CLIENT_ID=your_client_id
OAUTH_CLIENT_SECRET=your_client_secret
PARSIAN_PIN=your_pin

# FTP
FTP_HOST=ftp.metargb.com
FTP_PORT=21
FTP_USER=your_ftp_user
FTP_PASSWORD=your_ftp_password
FTP_BASE_URL=https://cdn.metargb.com/uploads

# WebSocket Gateway
CORS_ORIGIN=http://localhost:3000,http://localhost:8080
NODE_ENV=development
```

---

## Running Services

### Using Docker Compose (Recommended)

```bash
# Start all services
make dev

# Or
docker-compose up -d

# Check status
make ps

# View logs
make logs

# Stop all services
make down

# Restart a specific service
make restart-service SERVICE=auth-service

# View logs for specific service
make logs-service SERVICE=auth-service
```

### Using Make Commands

```bash
# Build all services
make build-all

# Build specific service
make build-service SERVICE=auth-service

# Start development environment
make dev

# Check service status
make ps

# View logs
make logs

# Stop services
make down

# Clean up (removes volumes)
make clean
```

### Service Ports

| Service | Port | Protocol |
|---------|------|----------|
| Kong API Gateway | 8000 | HTTP |
| Kong Admin | 8001 | HTTP |
| WebSocket Gateway | 3000 | WebSocket |
| Auth Service | 50051 | gRPC |
| Commercial Service | 50052 | gRPC |
| Features Service | 50053 | gRPC |
| Levels Service | 50054 | gRPC |
| Dynasty Service | 50055 | gRPC |
| Calendar Service | 50058 | gRPC |
| Storage Service (gRPC) | 50059 | gRPC |
| Storage Service (HTTP) | 8059 | HTTP |
| MySQL | 3308 | TCP |
| Redis | 6379 | TCP |

---

## Testing

### 1. Unit Tests

```bash
# Run unit tests for all services
make test-unit

# Run unit tests for specific service
cd services/auth-service
go test ./internal/... -v -race -coverprofile=coverage.out
```

### 2. Integration Tests

```bash
# Ensure all services are running first
make dev

# Run integration tests
make test-integration

# Or manually:
cd tests/integration
go test -v ./...
```

### 3. Golden JSON Tests

```bash
# First, capture golden responses from Laravel (if available)
make capture-golden

# Validate golden JSON files
make validate-golden

# Run golden tests
make test-golden

# Or manually:
cd tests/golden
go test -v ./...
```

### 4. Database Tests

```bash
# Run database schema and concurrency tests
make test-database

# Or manually:
cd tests/database
go test -v ./...
```

### 5. Load Tests (Optional)

```bash
# Ensure services are running and accessible
# Update tests/load/*.js with correct API URLs

# Run load tests
make load-test-auth
make load-test-features
make load-test-commercial

# Or run all load tests
make load-test-all
```

### 6. Run All Tests

```bash
# Run complete test suite
make test-all
```

### Test Database Setup

```bash
# Create test database
mysql -u root -p << EOF
CREATE DATABASE metargb_test;
GRANT ALL PRIVILEGES ON metargb_test.* TO 'metargb_user'@'localhost';
FLUSH PRIVILEGES;
EOF

# Import schema
mysql -u root -p metargb_test < scripts/schema.sql

# Import test fixtures (if available)
mysql -u root -p metargb_test < tests/fixtures/test_data.sql
```

### Manual API Testing

```bash
# Test Kong Gateway
curl http://localhost:8000/api/auth/me \
  -H "Authorization: Bearer your_token_here"

# Test features endpoint
curl "http://localhost:8000/api/features?bbox=35.0,51.0,36.0,52.0" \
  -H "Authorization: Bearer your_token_here"

# Test wallet endpoint
curl http://localhost:8000/api/user/wallet \
  -H "Authorization: Bearer your_token_here"

# Test WebSocket health
curl http://localhost:3000/health
```

---

## Troubleshooting

### Common Issues

#### 1. Services Not Starting

```bash
# Check logs
docker-compose logs auth-service

# Check if database is accessible
docker-compose exec mysql mysql -uroot -proot_password -e "SELECT 1"

# Check if Redis is accessible
docker-compose exec redis redis-cli ping
```

#### 2. Database Connection Errors

```bash
# Verify MySQL is running
docker-compose ps mysql

# Check MySQL logs
docker-compose logs mysql

# Test connection
docker-compose exec mysql mysql -uroot -proot_password metargb_db -e "SHOW TABLES;"

# Re-import schema if needed
make import-schema
```

#### 3. Proto Generation Errors

```bash
# Clean and regenerate
make clean-proto
make proto

# Verify protoc is installed
protoc --version

# Check Go protobuf plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

#### 4. Port Already in Use

```bash
# Find process using port
lsof -i :50051  # macOS
netstat -tulpn | grep 50051  # Linux

# Kill process or change port in docker-compose.yml
```

#### 5. Kong Configuration Errors

```bash
# Validate Kong config
make kong-validate

# Check Kong logs
docker-compose logs kong

# Restart Kong
docker-compose restart kong
```

### Debugging Commands

```bash
# Enter service container
docker-compose exec auth-service sh

# Check service health
docker-compose exec auth-service nc -z localhost 50051

# View service logs in real-time
docker-compose logs -f auth-service

# Check network connectivity between services
docker-compose exec auth-service ping commercial-service

# Check environment variables
docker-compose exec auth-service env | grep DB_
```

### Reset Everything

```bash
# Stop all services and remove volumes
make clean

# Or manually:
docker-compose down -v
docker system prune -f

# Restart from scratch
make dev
```

---

## Next Steps

1. **Read Documentation**:
   - `docs/ARCHITECTURE.md` - System architecture
   - `docs/DEPLOYMENT.md` - Production deployment guide
   - `docs/TROUBLESHOOTING.md` - Common issues and solutions

2. **Explore Services**:
   - Check individual service READMEs in `services/*/README.md`
   - Review proto definitions in `shared/proto/`

3. **Development Workflow**:
   - Make changes to service code
   - Update proto files if needed
   - Run `make proto` to regenerate
   - Test locally
   - Run test suite

4. **Production Deployment**:
   - Follow `docs/DEPLOYMENT.md` for Kubernetes setup
   - Configure monitoring (Prometheus, Grafana, Jaeger)
   - Set up Istio service mesh

---

## Quick Reference

```bash
# Start everything
make dev

# Check status
make ps

# View logs
make logs

# Run tests
make test-all

# Stop everything
make down

# Clean up
make clean

# Generate proto files
make proto

# Build services
make build-all
```

For more detailed information, see the individual documentation files in the `docs/` directory.

