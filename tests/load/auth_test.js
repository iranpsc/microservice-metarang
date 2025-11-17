import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const authLatency = new Trend('auth_latency');

// Test configuration
export const options = {
  stages: [
    { duration: '1m', target: 20 },   // Ramp up to 20 users
    { duration: '3m', target: 100 },  // Ramp up to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 200 },  // Spike to 200 users
    { duration: '3m', target: 100 },  // Scale back to 100 users
    { duration: '1m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests must complete below 500ms
    http_req_failed: ['rate<0.01'],   // Less than 1% error rate
    errors: ['rate<0.001'],            // Less than 0.1% application errors
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8000';

// Test data
const TEST_USERS = [
  { username: 'load_test_user_1', password: 'password123' },
  { username: 'load_test_user_2', password: 'password123' },
  { username: 'load_test_user_3', password: 'password123' },
  { username: 'load_test_user_4', password: 'password123' },
  { username: 'load_test_user_5', password: 'password123' },
];

export default function () {
  // Select random user
  const user = TEST_USERS[Math.floor(Math.random() * TEST_USERS.length)];

  // Test 1: Login
  testLogin(user);
  sleep(1);

  // Test 2: Get user info
  const token = login(user);
  if (token) {
    testGetMe(token);
    sleep(1);

    // Test 3: Validate token (simulates middleware)
    testValidateToken(token);
    sleep(1);

    // Test 4: Logout
    testLogout(token);
  }

  sleep(Math.random() * 3); // Random think time
}

function testLogin(user) {
  const startTime = new Date().getTime();
  
  const payload = JSON.stringify({
    username: user.username,
    password: user.password,
  });

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: { name: 'Login' },
  };

  const res = http.post(`${BASE_URL}/api/auth/login`, payload, params);
  
  const duration = new Date().getTime() - startTime;
  authLatency.add(duration);

  const success = check(res, {
    'login status is 200': (r) => r.status === 200,
    'login returns token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.token || body.data?.token;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
}

function login(user) {
  const payload = JSON.stringify({
    username: user.username,
    password: user.password,
  });

  const res = http.post(`${BASE_URL}/api/auth/login`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (res.status === 200) {
    try {
      const body = JSON.parse(res.body);
      return body.token || body.data?.token;
    } catch {
      return null;
    }
  }
  return null;
}

function testGetMe(token) {
  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    tags: { name: 'GetMe' },
  };

  const res = http.post(`${BASE_URL}/api/auth/me`, null, params);

  const success = check(res, {
    'getme status is 200': (r) => r.status === 200,
    'getme returns user data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data?.id !== undefined;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
}

function testValidateToken(token) {
  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    tags: { name: 'ValidateToken' },
  };

  const res = http.get(`${BASE_URL}/api/auth/validate`, params);

  const success = check(res, {
    'validate status is 200': (r) => r.status === 200,
  });

  errorRate.add(!success);
}

function testLogout(token) {
  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    tags: { name: 'Logout' },
  };

  const res = http.post(`${BASE_URL}/api/auth/logout`, null, params);

  const success = check(res, {
    'logout status is 200': (r) => r.status === 200,
  });

  errorRate.add(!success);
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'load-test-results-auth.json': JSON.stringify(data),
  };
}

function textSummary(data, options) {
  const indent = options?.indent || '';
  let summary = `\n${indent}Load Test Summary - Auth Service\n`;
  summary += `${indent}================================\n\n`;

  // Request statistics
  const httpReqs = data.metrics.http_reqs.values.count;
  const httpReqDuration = data.metrics.http_req_duration.values;
  const httpReqFailed = data.metrics.http_req_failed.values.rate;

  summary += `${indent}Total Requests: ${httpReqs}\n`;
  summary += `${indent}Request Duration:\n`;
  summary += `${indent}  - avg: ${httpReqDuration.avg.toFixed(2)}ms\n`;
  summary += `${indent}  - p95: ${httpReqDuration['p(95)'].toFixed(2)}ms\n`;
  summary += `${indent}  - p99: ${httpReqDuration['p(99)'].toFixed(2)}ms\n`;
  summary += `${indent}  - max: ${httpReqDuration.max.toFixed(2)}ms\n`;
  summary += `${indent}Error Rate: ${(httpReqFailed * 100).toFixed(2)}%\n\n`;

  // Custom metrics
  if (data.metrics.errors) {
    summary += `${indent}Application Error Rate: ${(data.metrics.errors.values.rate * 100).toFixed(2)}%\n`;
  }
  if (data.metrics.auth_latency) {
    const authLat = data.metrics.auth_latency.values;
    summary += `${indent}Auth Latency:\n`;
    summary += `${indent}  - avg: ${authLat.avg.toFixed(2)}ms\n`;
    summary += `${indent}  - p95: ${authLat['p(95)'].toFixed(2)}ms\n`;
  }

  // Threshold results
  summary += `\n${indent}Thresholds:\n`;
  for (const [metric, thresholds] of Object.entries(data.metrics)) {
    if (thresholds.thresholds) {
      for (const [name, result] of Object.entries(thresholds.thresholds)) {
        const status = result.ok ? '✓' : '✗';
        summary += `${indent}  ${status} ${metric}: ${name}\n`;
      }
    }
  }

  return summary;
}

