#!/usr/bin/env python3
"""
Script to check load test results against performance thresholds.
Usage: python3 check_thresholds.py results-*.json
"""

import json
import sys
from typing import Dict, Any, List

# Performance thresholds
THRESHOLDS = {
    'p95_latency_ms': 500,      # p95 should be under 500ms
    'error_rate_percent': 0.1,  # Error rate should be under 0.1%
    'success_rate_percent': 99.9,  # Success rate should be above 99.9%
}

class Color:
    """ANSI color codes"""
    GREEN = '\033[92m'
    RED = '\033[91m'
    YELLOW = '\033[93m'
    BLUE = '\033[94m'
    END = '\033[0m'

def load_results(file_path: str) -> Dict[str, Any]:
    """Load k6 results from JSON file"""
    try:
        with open(file_path, 'r') as f:
            return json.load(f)
    except Exception as e:
        print(f"{Color.RED}Error loading {file_path}: {e}{Color.END}")
        return {}

def extract_metrics(data: Dict[str, Any]) -> Dict[str, float]:
    """Extract relevant metrics from k6 results"""
    metrics = {}
    
    if 'metrics' not in data:
        return metrics
    
    # HTTP request duration
    if 'http_req_duration' in data['metrics']:
        duration = data['metrics']['http_req_duration']['values']
        metrics['avg_latency'] = duration.get('avg', 0)
        metrics['p95_latency'] = duration.get('p(95)', 0)
        metrics['p99_latency'] = duration.get('p(99)', 0)
        metrics['max_latency'] = duration.get('max', 0)
    
    # HTTP request failure rate
    if 'http_req_failed' in data['metrics']:
        failed = data['metrics']['http_req_failed']['values']
        metrics['error_rate'] = failed.get('rate', 0) * 100  # Convert to percentage
    
    # Total requests
    if 'http_reqs' in data['metrics']:
        reqs = data['metrics']['http_reqs']['values']
        metrics['total_requests'] = reqs.get('count', 0)
        metrics['requests_per_second'] = reqs.get('rate', 0)
    
    # Iterations
    if 'iterations' in data['metrics']:
        iterations = data['metrics']['iterations']['values']
        metrics['total_iterations'] = iterations.get('count', 0)
    
    # VUs
    if 'vus' in data['metrics']:
        vus = data['metrics']['vus']['values']
        metrics['max_vus'] = vus.get('max', 0)
    
    return metrics

def check_threshold(name: str, value: float, threshold: float, lower_is_better: bool = True) -> bool:
    """Check if a metric meets its threshold"""
    if lower_is_better:
        passed = value <= threshold
        symbol = '≤'
    else:
        passed = value >= threshold
        symbol = '≥'
    
    status = f"{Color.GREEN}✓ PASS{Color.END}" if passed else f"{Color.RED}✗ FAIL{Color.END}"
    comparison = f"{value:.2f} {symbol} {threshold:.2f}"
    
    print(f"  {status} {name}: {comparison}")
    
    return passed

def analyze_results(file_path: str) -> bool:
    """Analyze load test results and check against thresholds"""
    print(f"\n{Color.BLUE}Analyzing: {file_path}{Color.END}")
    print("=" * 60)
    
    data = load_results(file_path)
    if not data:
        return False
    
    metrics = extract_metrics(data)
    if not metrics:
        print(f"{Color.RED}No metrics found in results{Color.END}")
        return False
    
    # Print summary
    print(f"\n{Color.BLUE}Summary:{Color.END}")
    print(f"  Total Requests: {metrics.get('total_requests', 0):.0f}")
    print(f"  RPS: {metrics.get('requests_per_second', 0):.2f}")
    print(f"  Max VUs: {metrics.get('max_vus', 0):.0f}")
    print(f"  Avg Latency: {metrics.get('avg_latency', 0):.2f}ms")
    print(f"  P95 Latency: {metrics.get('p95_latency', 0):.2f}ms")
    print(f"  P99 Latency: {metrics.get('p99_latency', 0):.2f}ms")
    print(f"  Error Rate: {metrics.get('error_rate', 0):.2f}%")
    
    # Check thresholds
    print(f"\n{Color.BLUE}Threshold Checks:{Color.END}")
    
    all_passed = True
    
    # Check P95 latency
    if 'p95_latency' in metrics:
        passed = check_threshold(
            'P95 Latency',
            metrics['p95_latency'],
            THRESHOLDS['p95_latency_ms'],
            lower_is_better=True
        )
        all_passed = all_passed and passed
    
    # Check error rate
    if 'error_rate' in metrics:
        passed = check_threshold(
            'Error Rate',
            metrics['error_rate'],
            THRESHOLDS['error_rate_percent'],
            lower_is_better=True
        )
        all_passed = all_passed and passed
    
    # Calculate and check success rate
    if 'error_rate' in metrics:
        success_rate = 100 - metrics['error_rate']
        passed = check_threshold(
            'Success Rate',
            success_rate,
            THRESHOLDS['success_rate_percent'],
            lower_is_better=False
        )
        all_passed = all_passed and passed
    
    return all_passed

def main():
    """Main function"""
    if len(sys.argv) < 2:
        print(f"Usage: {sys.argv[0]} results-*.json")
        sys.exit(1)
    
    result_files = sys.argv[1:]
    
    print(f"{Color.BLUE}Load Test Threshold Checker{Color.END}")
    print("=" * 60)
    print(f"\nThresholds:")
    print(f"  P95 Latency: ≤ {THRESHOLDS['p95_latency_ms']}ms")
    print(f"  Error Rate: ≤ {THRESHOLDS['error_rate_percent']}%")
    print(f"  Success Rate: ≥ {THRESHOLDS['success_rate_percent']}%")
    
    all_passed = True
    for file_path in result_files:
        passed = analyze_results(file_path)
        all_passed = all_passed and passed
    
    # Final summary
    print("\n" + "=" * 60)
    if all_passed:
        print(f"{Color.GREEN}✓ All tests passed thresholds!{Color.END}")
        sys.exit(0)
    else:
        print(f"{Color.RED}✗ Some tests failed to meet thresholds{Color.END}")
        sys.exit(1)

if __name__ == '__main__':
    main()

