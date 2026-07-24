# metarang Microservices Tests

This directory contains the test suite for verifying functionality of the microservices.

## Test Types

### 1. Service Test Modules (`*-service/`)

Dedicated test modules per service under `tests/`, covering handlers, services, and repositories.

**Run service test modules:**
```bash
make test-services
```

### 2. Unit Tests (in each service)

Service-specific unit tests for business logic.

**Run unit tests:**
```bash
make test-unit
```

### 3. Database Tests (`database/`)

Schema and concurrency tests against MySQL.

**Run database tests:**
```bash
make test-database
```

## Running All Tests

```bash
make test-all
```

## Test Database

Database tests require a test database with the project schema:

```bash
# Create test database
mysql -u root -p -e "CREATE DATABASE metarang_test;"

# Import schema
mysql -u root -p metarang_test < scripts/schema.sql
```

## CI/CD Integration

Tests run automatically in GitHub Actions via the Services CI/CD workflow (`service-ci.yml`), which runs unit tests for each changed service against MySQL/Redis containers.

## Test Coverage Goals

- **Unit tests**: > 80% code coverage per service
- **Service test modules**: Handler and service layer coverage for critical paths

## Adding New Tests

1. **Service test module**:
   - Add tests under `tests/{service-name}/internal/...`
   - Use existing `testutil` helpers where available

2. **Unit test**:
   - Create test file next to source: `{filename}_test.go`
   - Mock dependencies
   - Test business logic

## Troubleshooting

**Connection refused:**
- Ensure services are running on correct ports
- Check firewall settings
- Verify database connectivity

**Flaky tests:**
- Use fixed timestamps in test data
- Mock external dependencies (OAuth, Parsian)
- Avoid race conditions in parallel tests
