import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const walletLoadTime = new Trend('wallet_load_time');
const transactionQueries = new Counter('transaction_queries');

export const options = {
  stages: [
    { duration: '1m', target: 30 },
    { duration: '3m', target: 80 },
    { duration: '5m', target: 80 },
    { duration: '2m', target: 120 },
    { duration: '2m', target: 80 },
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'],
    http_req_failed: ['rate<0.01'],
    errors: ['rate<0.001'],
  },
};

const BASE_URL = __ENV.API_URL || 'http://localhost:8000';
const TEST_TOKEN = __ENV.TEST_TOKEN || 'test_token';

export default function () {
  const token = TEST_TOKEN;

  // Test 1: Get wallet balance
  testGetWallet(token);
  sleep(2);

  // Test 2: Get transactions
  testGetTransactions(token);
  sleep(1);

  // Test 3: Get latest transaction
  testGetLatestTransaction(token);
  sleep(1);

  // Test 4: Initiate payment (5% chance)
  if (Math.random() < 0.05) {
    testInitiatePayment(token);
  }

  sleep(Math.random() * 3);
}

function testGetWallet(token) {
  const startTime = new Date().getTime();

  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    tags: { name: 'GetWallet' },
  };

  const res = http.get(`${BASE_URL}/api/user/wallet`, params);
  
  const duration = new Date().getTime() - startTime;
  walletLoadTime.add(duration);

  const success = check(res, {
    'wallet status is 200': (r) => r.status === 200,
    'wallet has psc balance': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data?.psc !== undefined;
      } catch {
        return false;
      }
    },
    'wallet has rgb balance': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data?.rgb !== undefined;
      } catch {
        return false;
      }
    },
    'balances are strings': (r) => {
      try {
        const body = JSON.parse(r.body);
        return typeof body.data?.psc === 'string' && 
               typeof body.data?.rgb === 'string';
      } catch {
        return false;
      }
    },
    'response time < 300ms': (r) => duration < 300,
  });

  errorRate.add(!success);
}

function testGetTransactions(token) {
  transactionQueries.add(1);

  const page = Math.floor(Math.random() * 5) + 1;
  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    tags: { name: 'GetTransactions' },
  };

  const res = http.get(`${BASE_URL}/api/user/transactions?page=${page}&per_page=20`, params);

  const success = check(res, {
    'transactions status is 200': (r) => r.status === 200,
    'transactions returns array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body.data);
      } catch {
        return false;
      }
    },
    'pagination meta exists': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.meta?.total !== undefined;
      } catch {
        return false;
      }
    },
    'transaction IDs are strings': (r) => {
      try {
        const body = JSON.parse(r.body);
        if (body.data && body.data.length > 0) {
          return typeof body.data[0].id === 'string';
        }
        return true; // Empty array is OK
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
}

function testGetLatestTransaction(token) {
  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
    },
    tags: { name: 'LatestTransaction' },
  };

  const res = http.get(`${BASE_URL}/api/user/transactions/latest`, params);

  const success = check(res, {
    'latest transaction status is 200': (r) => r.status === 200,
    'returns transaction object': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.data?.id !== undefined || body.data === null;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
}

function testInitiatePayment(token) {
  const amount = Math.floor(Math.random() * 100000) + 10000;
  
  const payload = JSON.stringify({
    amount: amount,
    return_url: 'http://localhost:3000/payment/callback',
  });

  const params = {
    headers: {
      'Authorization': `Bearer ${token}`,
      'Content-Type': 'application/json',
    },
    tags: { name: 'InitiatePayment' },
  };

  const res = http.post(`${BASE_URL}/api/order`, payload, params);

  const success = check(res, {
    'payment initiation response received': (r) => 
      r.status === 200 || r.status === 400 || r.status === 422,
  });

  errorRate.add(!success);
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data),
    'load-test-results-commercial.json': JSON.stringify(data),
  };
}

function textSummary(data) {
  let summary = '\n  Load Test Summary - Commercial Service\n';
  summary += '  =======================================\n\n';

  const httpReqs = data.metrics.http_reqs.values.count;
  const httpReqDuration = data.metrics.http_req_duration.values;
  const httpReqFailed = data.metrics.http_req_failed.values.rate;

  summary += `  Total Requests: ${httpReqs}\n`;
  summary += `  Request Duration:\n`;
  summary += `    - avg: ${httpReqDuration.avg.toFixed(2)}ms\n`;
  summary += `    - p95: ${httpReqDuration['p(95)'].toFixed(2)}ms\n`;
  summary += `    - p99: ${httpReqDuration['p(99)'].toFixed(2)}ms\n`;
  summary += `  Error Rate: ${(httpReqFailed * 100).toFixed(2)}%\n\n`;

  if (data.metrics.wallet_load_time) {
    const walletTime = data.metrics.wallet_load_time.values;
    summary += `  Wallet Load Time:\n`;
    summary += `    - avg: ${walletTime.avg.toFixed(2)}ms\n`;
    summary += `    - p95: ${walletTime['p(95)'].toFixed(2)}ms\n\n`;
  }

  if (data.metrics.transaction_queries) {
    summary += `  Transaction Queries: ${data.metrics.transaction_queries.values.count}\n`;
  }

  return summary;
}

