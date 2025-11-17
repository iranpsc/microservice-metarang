# MetaRGB Microservices Tests

This directory contains the test suite for verifying API compatibility and functionality of the microservices.

## Test Types

### 1. Integration Tests (`integration/`)

Tests that verify end-to-end functionality of services by making actual gRPC calls.

**Run integration tests:**
```bash
# Start all services first
cd services/auth-service && go run cmd/server/main.go &
cd services/commercial-service && go run cmd/server/main.go &

# Run tests
go test ./tests/integration/... -v
```

**Coverage:**
- Auth service: Register, Redirect, Callback, GetMe, Logout, ValidateToken
- Commercial service: Wallet operations, Transactions, Payments
- Cross-service interactions

### 2. Golden JSON Tests (`golden/`)

Byte-for-byte comparison tests that ensure microservice responses exactly match Laravel outputs.

**Run golden tests:**
```bash
# Capture golden responses from Laravel first (see golden/testdata/README.md)

# Run tests
go test ./tests/golden/... -v
```

**Coverage:**
- All critical API endpoints
- Response structure validation
- Data type verification
- Jalali date format checking
- Compact number format validation

### 3. Unit Tests (in each service)

Service-specific unit tests for business logic.

**Run unit tests:**
```bash
cd services/auth-service
go test ./internal/... -v

cd services/commercial-service
go test ./internal/... -v
```

## Test Database

Integration and golden tests require a test database with known data:

```bash
# Create test database
mysql -u root -p -e "CREATE DATABASE metargb_test;"

# Import schema
mysql -u root -p metargb_test < scripts/schema.sql

# Import test fixtures
mysql -u root -p metargb_test < tests/fixtures/test_data.sql
```

## Environment Setup

```bash
# Copy environment file for tests
cp .env.test.example .env.test

# Edit with test database credentials
DB_DATABASE=metargb_test
DB_USER=test_user
DB_PASSWORD=test_password
```

## CI/CD Integration

Tests are run automatically in GitHub Actions:

```yaml
# .github/workflows/test.yml
- name: Run integration tests
  run: go test ./tests/integration/... -v

- name: Run golden JSON tests
  run: go test ./tests/golden/... -v
```

## Test Coverage Goals

- **Unit tests**: > 80% code coverage per service
- **Integration tests**: All gRPC methods tested
- **Golden tests**: All Phase 2 endpoints verified

## Adding New Tests

1. **Integration test**:
   - Create `tests/integration/{service}_test.go`
   - Connect to service via gRPC
   - Make requests and verify responses

2. **Golden test**:
   - Capture Laravel response to `tests/golden/testdata/{endpoint}.json`
   - Add test case in `tests/golden/golden_test.go`
   - Run comparison

3. **Unit test**:
   - Create test file next to source: `{filename}_test.go`
   - Mock dependencies
   - Test business logic

## Troubleshooting

**Connection refused:**
- Ensure services are running on correct ports
- Check firewall settings
- Verify database connectivity

**Golden test failures:**
- Check date/time field formats (Jalali)
- Verify number formatting (compact notation)
- Ensure test data matches golden capture state

**Flaky tests:**
- Use fixed timestamps in test data
- Mock external dependencies (OAuth, Parsian)
- Avoid race conditions in parallel tests

