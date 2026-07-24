import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const featureLoadTime = new Trend('feature_load_time');
const purchaseAttempts = new Counter('purchase_attempts');
const purchaseSuccess = new Counter('purchase_success');

export const options = {
  stages: [
    { duration: '2m', target: 50 },   // Ramp up
    { duration: '5m', target: 100 },  // Steady state
    { duration: '2m', target: 150 },  // Spike
    { duration: '3m', target: 100 },  // Scale back
    { duration: '1m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
    errors: ['rate<0.001'],
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8000';
const TEST_TOKEN = __ENV.TEST_TOKEN || 'test_token';

// Test data - bounding boxes for different regions
const BBOXES = [
  '35.0,51.0,36.0,52.0',  // Tehran region
  '36.0,59.0,37.0,60.0',  // Mashhad region
  '29.0,52.0,30.0,53.0',  // Shiraz region
];

export default function () {
  const token = authenticateUser();
  
  if (!token) {
    errorRate.add(1);
    return;
  }

  // Test 1: List features with bbox
  testListFeatures(token);
  sleep(2);

  // Test 2: Get feature details
  const featureId = getRandomFeature(token);
  if (featureId) {
    testGetFeature(token, featureId);
    sleep(1);
  }

  // Test 3: Get user's features
  testMyFeatures(token);
  sleep(1);

  // Test 4: Attempt purchase (10% chance)
  if (Math.random() < 0.1 && featureId) {
    testPurchaseFeature(token, featureId);
  }

  sleep(Math.random() * 3);
}

function authenticateUser() {
  // In real scenario, this would authenticate
  // For load testing, we use pre-generated tokens
  return TEST_TOKEN;
}

function testListFeatures(token) {
  const bbox = BBOXES[Math.floor(Math.random() * BBOXES.length)];
  const startTime = new Date().getTime();

  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    tags: { name: 'ListFeatures' },
  };

  const res = http.get(`${BASE_URL}/api/features?bbox=${bbox}`, params);
  
  const duration = new Date().getTime() - startTime;
  featureLoadTime.add(duration);

  const success = check(res, {
    'list features status is 200': (r) => r.status === 200,
    'list features returns data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body.data);
      } catch {
        return false;
      }
    },
    'response time < 500ms': (r) => duration < 500,
  });

  errorRate.add(!success);
}

function getRandomFeature(token) {
  const bbox = BBOXES[0];
  const res = http.get(`${BASE_URL}/api/features?bbox=${bbox}`, {
    headers: { 'Authorization': `Bearer ${token}` },
  });

  if (res.status === 200) {
    try {
      const body = JSON.parse(res.body);
      if (body.data && body.data.length > 0) {
        const randomIndex = Math.floor(Math.random() * body.data.length);
        return body.data[randomIndex].id;
      }
    } catch {}
  }
  return null;
}

function testGetFeature(token, featureId) {
  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    tags: { name: 'GetFeature' },
  };

  const res = http.get(`${BASE_URL}/api/features/${featureId}`, params);

  const success = check(res, {
    'get feature status is 200': (r) => r.status === 200,
    'feature has geometry': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data?.geometry !== undefined;
      } catch {
        return false;
      }
    },
    'feature has properties': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data?.property !== undefined;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
}

function testMyFeatures(token) {
  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    tags: { name: 'MyFeatures' },
  };

  const res = http.get(`${BASE_URL}/api/my-features`, params);

  const success = check(res, {
    'my features status is 200': (r) => r.status === 200,
    'my features returns array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body.data);
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
}

function testPurchaseFeature(token, featureId) {
  purchaseAttempts.add(1);

  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    tags: { name: 'PurchaseFeature' },
  };

  const res = http.post(`${BASE_URL}/api/features/buy/${featureId}`, null, params);

  const success = check(res, {
    'purchase response received': (r) => r.status === 200 || r.status === 400 || r.status === 422,
  });

  if (res.status === 200) {
    purchaseSuccess.add(1);
  }

  errorRate.add(!success);
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data),
    'load-test-results-features.json': JSON.stringify(data),
  };
}

function textSummary(data) {
  let summary = '\n  Load Test Summary - Features Service\n';
  summary += '  =====================================\n\n';

  const httpReqs = data.metrics.http_reqs.values.count;
  const httpReqDuration = data.metrics.http_req_duration.values;
  const httpReqFailed = data.metrics.http_req_failed.values.rate;

  summary += `  Total Requests: ${httpReqs}\n`;
  summary += `  Request Duration:\n`;
  summary += `    - avg: ${httpReqDuration.avg.toFixed(2)}ms\n`;
  summary += `    - p95: ${httpReqDuration['p(95)'].toFixed(2)}ms\n`;
  summary += `    - p99: ${httpReqDuration['p(99)'].toFixed(2)}ms\n`;
  summary += `  Error Rate: ${(httpReqFailed * 100).toFixed(2)}%\n\n`;

  if (data.metrics.purchase_attempts) {
    const attempts = data.metrics.purchase_attempts.values.count;
    const success = data.metrics.purchase_success?.values.count || 0;
    summary += `  Purchase Stats:\n`;
    summary += `    - Attempts: ${attempts}\n`;
    summary += `    - Success: ${success}\n`;
    summary += `    - Success Rate: ${((success / attempts) * 100).toFixed(2)}%\n\n`;
  }

  if (data.metrics.feature_load_time) {
    const loadTime = data.metrics.feature_load_time.values;
    summary += `  Feature Load Time:\n`;
    summary += `    - avg: ${loadTime.avg.toFixed(2)}ms\n`;
    summary += `    - p95: ${loadTime['p(95)'].toFixed(2)}ms\n`;
  }

  return summary;
}

