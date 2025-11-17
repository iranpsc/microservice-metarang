# Load Testing with k6

Performance and load testing for MetaRGB microservices.

## Overview

This directory contains k6 load testing scripts that simulate realistic user traffic patterns and validate performance thresholds.

## Prerequisites

### Install k6

**macOS**:
```bash
brew install k6
```

**Linux**:
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows**:
```powershell
choco install k6
```

### Install Python (for threshold checking)
```bash
python3 --version  # Should be 3.7+
```

## Running Tests

### Individual Service Tests

**Auth Service**:
```bash
k6 run --duration=5m --vus=100 --env API_URL=http://localhost:8000 auth_test.js
```

**Features Service**:
```bash
k6 run --duration=5m --vus=100 --env API_URL=http://localhost:8000 features_test.js
```

**Commercial Service**:
```bash
k6 run --duration=5m --vus=100 --env API_URL=http://localhost:8000 commercial_test.js
```

### Run All Tests

Using Makefile:
```bash
make load-test-all
```

Or manually:
```bash
for test in auth_test.js features_test.js commercial_test.js; do
  k6 run --duration=5m --vus=100 --env API_URL=http://localhost:8000 $test
done
```

### Custom Parameters

```bash
# Custom duration
k6 run --duration=10m --vus=100 auth_test.js

# Custom VUs (virtual users)
k6 run --duration=5m --vus=200 auth_test.js

# Custom test token
k6 run --env API_URL=http://production.com --env TEST_TOKEN=xyz auth_test.js

# Output to file
k6 run --out json=results-auth.json auth_test.js
```

## Test Scenarios

### Auth Test (`auth_test.js`)

**Endpoints Tested**:
- POST `/api/auth/login`
- POST `/api/auth/me`
- GET `/api/auth/validate`
- POST `/api/auth/logout`

**Load Profile**:
```
Stage 1: Ramp up to 20 VUs (1 min)
Stage 2: Ramp up to 100 VUs (3 min)
Stage 3: Stay at 100 VUs (5 min)
Stage 4: Spike to 200 VUs (2 min)
Stage 5: Scale back to 100 VUs (3 min)
Stage 6: Ramp down to 0 (1 min)
```

**Custom Metrics**:
- `errors` - Application error rate
- `auth_latency` - Authentication endpoint latency trend

### Features Test (`features_test.js`)

**Endpoints Tested**:
- GET `/api/features?bbox=...`
- GET `/api/features/{id}`
- GET `/api/my-features`
- POST `/api/features/buy/{id}` (10% of requests)

**Load Profile**:
```
Stage 1: Ramp up to 50 VUs (2 min)
Stage 2: Stay at 100 VUs (5 min)
Stage 3: Spike to 150 VUs (2 min)
Stage 4: Scale back to 100 VUs (3 min)
Stage 5: Ramp down to 0 (1 min)
```

**Custom Metrics**:
- `feature_load_time` - Feature list/detail load times
- `purchase_attempts` - Total purchase attempts
- `purchase_success` - Successful purchases

### Commercial Test (`commercial_test.js`)

**Endpoints Tested**:
- GET `/api/user/wallet`
- GET `/api/user/transactions?page=X`
- GET `/api/user/transactions/latest`
- POST `/api/order` (5% of requests)

**Load Profile**:
```
Stage 1: Ramp up to 30 VUs (1 min)
Stage 2: Ramp up to 80 VUs (3 min)
Stage 3: Stay at 80 VUs (5 min)
Stage 4: Spike to 120 VUs (2 min)
Stage 5: Scale back to 80 VUs (2 min)
Stage 6: Ramp down to 0 (1 min)
```

**Custom Metrics**:
- `wallet_load_time` - Wallet query latency
- `transaction_queries` - Total transaction queries

## Performance Thresholds

All tests enforce these thresholds:

```javascript
thresholds: {
  http_req_duration: ['p(95)<500'],    // 95% < 500ms
  http_req_failed: ['rate<0.01'],       // < 1% errors
  errors: ['rate<0.001'],                // < 0.1% app errors
}
```

If any threshold fails, k6 exits with non-zero status.

## Checking Thresholds

After running tests, use the Python script to validate results:

```bash
python3 check_thresholds.py results-*.json
```

**Output**:
```
Load Test Threshold Checker
============================================================

Thresholds:
  P95 Latency: ≤ 500ms
  Error Rate: ≤ 0.1%
  Success Rate: ≥ 99.9%

Analyzing: results-auth.json
============================================================

Summary:
  Total Requests: 15234
  RPS: 50.78
  Max VUs: 200
  Avg Latency: 123.45ms
  P95 Latency: 387.23ms
  P99 Latency: 512.34ms
  Error Rate: 0.02%

Threshold Checks:
  ✓ PASS P95 Latency: 387.23 ≤ 500.00
  ✓ PASS Error Rate: 0.02 ≤ 0.10
  ✓ PASS Success Rate: 99.98 ≥ 99.90

✓ All tests passed thresholds!
```

## Interpreting Results

### Key Metrics

**Request Rate (RPS)**:
- Requests per second
- Higher is better (indicates throughput)

**Latency (p95, p99)**:
- 95th/99th percentile response time
- Most users experience this or better
- Lower is better

**Error Rate**:
- Percentage of failed requests
- Should be < 0.1% for production
- Lower is better

**VUs (Virtual Users)**:
- Concurrent simulated users
- More VUs = more load

### What Good Looks Like

✅ **Passing Results**:
- p95 latency < 500ms
- Error rate < 0.1%
- Consistent performance across load stages
- No significant spike in p99 latency

❌ **Failing Results**:
- p95 > 500ms (slow service)
- Error rate > 1% (reliability issues)
- Timeout errors (capacity problems)
- Memory/CPU spikes (resource exhaustion)

## CI/CD Integration

Tests automatically run in GitHub Actions:

```yaml
# .github/workflows/load-tests.yml
- name: Run load tests
  run: |
    k6 run --duration=5m --vus=100 tests/load/auth_test.js
    k6 run --duration=5m --vus=100 tests/load/features_test.js
    k6 run --duration=5m --vus=100 tests/load/commercial_test.js

- name: Check thresholds
  run: python3 tests/load/check_thresholds.py results-*.json
```

## Troubleshooting

### "connection refused" Errors

Services not running or wrong URL:
```bash
# Verify services are up
kubectl get pods -n metargb

# Get correct URL
export API_URL=$(kubectl get svc kong-proxy -n metargb -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
k6 run --env API_URL=http://$API_URL auth_test.js
```

### High Error Rates

Check service logs:
```bash
kubectl logs -l app=auth-service -n metargb --tail=100
```

### Timeouts

Increase timeout in k6:
```javascript
export const options = {
  thresholds: {
    http_req_duration: ['p(95)<1000'],  // Increase to 1s
  },
};
```

Or scale services:
```bash
kubectl scale deployment/auth-service --replicas=5 -n metargb
```

## Advanced Usage

### Custom Scenarios

Create custom test scenarios:

```javascript
export const options = {
  scenarios: {
    smoke: {
      executor: 'constant-vus',
      vus: 10,
      duration: '1m',
    },
    load: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '5m', target: 100 },
        { duration: '10m', target: 100 },
      ],
    },
    stress: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '2m', target: 200 },
        { duration: '5m', target: 200 },
        { duration: '2m', target: 500 },
        { duration: '5m', target: 500 },
      ],
    },
  },
};
```

### Cloud Execution

Run tests from k6 Cloud:
```bash
# Login to k6 Cloud
k6 login cloud

# Run test on cloud
k6 cloud auth_test.js
```

### Distributed Execution

For very high load, use k6-operator on Kubernetes:
```bash
helm install k6-operator grafana/k6-operator
kubectl apply -f k6-test-job.yaml
```

## Best Practices

1. **Start Small**: Begin with 10-20 VUs, gradually increase
2. **Monitor Services**: Watch metrics during tests
3. **Test Realistic Scenarios**: Match actual user behavior
4. **Run Regularly**: Schedule weekly/monthly load tests
5. **Compare Results**: Track trends over time
6. **Test Under Load**: Run during business hours (staging only!)
7. **Clean Up Data**: Remove test data after runs

## Resources

- [k6 Documentation](https://k6.io/docs/)
- [k6 Examples](https://k6.io/docs/examples/)
- [Load Testing Patterns](https://k6.io/docs/test-types/load-testing/)

