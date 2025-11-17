# Kong API Gateway Testing & Debugging Guide

This guide provides comprehensive instructions for testing and debugging Kong API Gateway in the MetaRGB microservices architecture.

## Quick Start

### Start Kong and Services
```bash
# Start all services including Kong
make up

# Or start just Kong (requires dependencies)
docker-compose up -d kong
```

### Quick Health Check
```bash
# Check Kong status
make kong-status

# Check Kong health
make kong-health

# Run all tests
make kong-test
```

## Available Commands

### Makefile Commands

| Command | Description |
|---------|-------------|
| `make kong-validate` | Validate Kong configuration file |
| `make kong-status` | Check Kong container and API status |
| `make kong-health` | Check Kong health endpoint |
| `make kong-services` | List all registered services |
| `make kong-routes` | List all registered routes |
| `make kong-logs` | Show last 50 lines of Kong logs |
| `make kong-logs-follow` | Follow Kong logs in real-time |
| `make kong-reload` | Reload Kong configuration |
| `make kong-debug` | Comprehensive debug information |
| `make kong-test` | Run all Kong tests |

### Testing Script

The `scripts/test-kong.sh` script provides interactive and command-line testing:

```bash
# Interactive mode
./scripts/test-kong.sh

# Command-line mode
./scripts/test-kong.sh all          # Run all tests
./scripts/test-kong.sh status       # Check container status
./scripts/test-kong.sh health       # Check health
./scripts/test-kong.sh validate     # Validate config
./scripts/test-kong.sh services     # List services
./scripts/test-kong.sh routes       # List routes
./scripts/test-kong.sh connectivity # Test service connectivity
./scripts/test-kong.sh endpoints    # Test proxy endpoints
./scripts/test-kong.sh logs         # Check logs
./scripts/test-kong.sh cors         # Test CORS
./scripts/test-kong.sh plugins      # Check plugins
./scripts/test-kong.sh reload       # Reload configuration
```

## Testing Checklist

### 1. Container Status
```bash
make kong-status
```
**Expected:** Kong container running, Admin API accessible

### 2. Configuration Validation
```bash
make kong-validate
```
**Expected:** Configuration file is valid (no syntax errors)

### 3. Health Check
```bash
make kong-health
```
**Expected:** Health endpoint returns status information

### 4. Service Registration
```bash
make kong-services
```
**Expected:** All microservices are registered:
- auth-service
- commercial-service
- features-service
- levels-service
- dynasty-service
- calendar-service
- storage-service

### 5. Route Registration
```bash
make kong-routes
```
**Expected:** Routes are registered for each service path

### 6. Service Connectivity
```bash
./scripts/test-kong.sh connectivity
```
**Expected:** All services are reachable from Kong container

### 7. Proxy Endpoints
```bash
./scripts/test-kong.sh endpoints
```
**Expected:** Endpoints return appropriate responses (404/405/400 for gRPC is normal)

### 8. CORS Configuration
```bash
./scripts/test-kong.sh cors
```
**Expected:** CORS preflight requests succeed

## Common Issues & Solutions

### Issue: Kong container not starting

**Symptoms:**
- `docker-compose ps` shows Kong as exited or unhealthy
- `make kong-status` shows container not running

**Solutions:**
1. Check Kong logs:
   ```bash
   make kong-logs
   ```

2. Verify configuration:
   ```bash
   make kong-validate
   ```

3. Check dependencies:
   ```bash
   docker-compose ps
   ```
   Ensure required services (auth-service, etc.) are running

4. Restart Kong:
   ```bash
   docker-compose restart kong
   ```

### Issue: Services not reachable

**Symptoms:**
- `make kong-services` shows services but connectivity test fails
- Proxy returns 502/503 errors

**Solutions:**
1. Verify services are running:
   ```bash
   docker-compose ps
   ```

2. Test connectivity from Kong container:
   ```bash
   docker exec metargb-kong nc -z auth-service 50051
   ```

3. Check network:
   ```bash
   docker network inspect metargb-microservices_metargb-network
   ```

4. Verify service ports match configuration:
   - Check `kong/kong.yml` service URLs
   - Check `docker-compose.yml` service ports

### Issue: Routes not working

**Symptoms:**
- Routes registered but requests fail
- 404 errors on valid endpoints

**Solutions:**
1. Verify route configuration:
   ```bash
   make kong-routes
   ```

2. Check route paths match requests:
   - Paths in `kong/kong.yml` should match API calls
   - Check `strip_path` setting

3. Test with curl:
   ```bash
   curl -v http://localhost:8000/api/auth
   ```

4. Check Kong logs:
   ```bash
   make kong-logs-follow
   ```

### Issue: Configuration errors

**Symptoms:**
- `make kong-validate` fails
- Kong fails to start

**Solutions:**
1. Check YAML syntax:
   ```bash
   yamllint kong/kong.yml
   ```

2. Verify service URLs:
   - URLs should match docker-compose service names
   - Ports should match service ports

3. Check for duplicate service/route names

4. Verify plugin configurations

### Issue: CORS not working

**Symptoms:**
- Browser CORS errors
- Preflight requests fail

**Solutions:**
1. Test CORS:
   ```bash
   ./scripts/test-kong.sh cors
   ```

2. Check CORS plugin configuration in `kong/kong.yml`

3. Verify allowed origins, methods, and headers

## Debugging Workflow

### Step 1: Check Container Status
```bash
make kong-status
```

### Step 2: Check Health
```bash
make kong-health
```

### Step 3: Validate Configuration
```bash
make kong-validate
```

### Step 4: Check Services & Routes
```bash
make kong-services
make kong-routes
```

### Step 5: Test Connectivity
```bash
./scripts/test-kong.sh connectivity
```

### Step 6: Check Logs
```bash
make kong-logs
# Or follow logs
make kong-logs-follow
```

### Step 7: Run Full Debug
```bash
make kong-debug
```

## Kong Admin API

Kong Admin API is available at `http://localhost:8001`

### Useful Endpoints

- **Status:** `GET http://localhost:8001/status`
- **Services:** `GET http://localhost:8001/services`
- **Routes:** `GET http://localhost:8001/routes`
- **Plugins:** `GET http://localhost:8001/plugins`
- **Health:** `GET http://localhost:8001/health`

### Example Queries

```bash
# Get all services
curl http://localhost:8001/services | jq '.'

# Get specific service
curl http://localhost:8001/services/auth-service | jq '.'

# Get all routes
curl http://localhost:8001/routes | jq '.'

# Get service routes
curl http://localhost:8001/services/auth-service/routes | jq '.'
```

## Proxy Testing

Kong Proxy is available at `http://localhost:8000`

### Test Endpoints

```bash
# Test auth endpoint
curl -v http://localhost:8000/api/auth

# Test with CORS
curl -v -X OPTIONS \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: GET" \
  http://localhost:8000/api/auth

# Test features endpoint
curl -v http://localhost:8000/api/features
```

## Configuration File

Kong configuration is in `kong/kong.yml`

### Key Configuration Points

1. **Service URLs:** Must match docker-compose service names
2. **Ports:** Must match service ports in docker-compose.yml
3. **Protocols:** Use `grpc` for gRPC services, `http` for HTTP services
4. **Routes:** Paths should match API endpoint patterns
5. **Plugins:** Configured per service or globally

### Recent Fixes

- Fixed commercial-service port from 50054 to 50052
- Verified all service URLs match docker-compose configuration

## Monitoring

### Logs
```bash
# View logs
make kong-logs

# Follow logs
make kong-logs-follow

# Search for errors
docker logs metargb-kong 2>&1 | grep -i error
```

### Metrics
Kong exposes metrics that can be scraped by Prometheus (if configured)

## Troubleshooting Tips

1. **Always check logs first:** `make kong-logs`
2. **Verify services are running:** `docker-compose ps`
3. **Test connectivity:** `./scripts/test-kong.sh connectivity`
4. **Validate configuration:** `make kong-validate`
5. **Check network:** Ensure all services are on the same Docker network
6. **Reload configuration:** `make kong-reload` after config changes
7. **Restart if needed:** `docker-compose restart kong`

## Additional Resources

- [Kong Documentation](https://docs.konghq.com/)
- [Kong Declarative Config](https://docs.konghq.com/gateway/latest/production/deployment-topologies/db-less-and-declarative-config/)
- [Kong Admin API](https://docs.konghq.com/gateway/latest/admin-api/)

