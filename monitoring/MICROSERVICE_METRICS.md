# Essential Metrics for Microservice Monitoring

This document outlines the comprehensive list of metrics that should be monitored in a microservice architecture, based on industry best practices and SRE (Site Reliability Engineering) principles.

## Table of Contents

1. [Golden Signals (SRE)](#1-golden-signals-sre)
2. [Infrastructure & Resource Metrics](#2-infrastructure--resource-metrics)
3. [Application Metrics](#3-application-metrics)
4. [Service Health & Availability](#4-service-health--availability)
5. [Database Metrics](#5-database-metrics)
6. [Cache Metrics](#6-cache-metrics)
7. [Message Queue Metrics](#7-message-queue-metrics)
8. [API Gateway Metrics](#8-api-gateway-metrics)
9. [Security Metrics](#9-security-metrics)
10. [Distributed Tracing Metrics](#10-distributed-tracing-metrics)
11. [Container/Orchestration Metrics](#11-containerorchestration-metrics)
12. [Cost & Efficiency Metrics](#12-cost--efficiency-metrics)
13. [Priority Ranking](#priority-ranking)
14. [Implementation Recommendations](#implementation-recommendations)

---

## 1. Golden Signals (SRE)

The **Four Golden Signals** are the most critical metrics for monitoring any distributed system, as defined by Google's SRE team.

### Latency
- **Request Duration**: Time taken to process a request
  - P50 (median latency)
  - P95 (95th percentile)
  - P99 (99th percentile)
- **End-to-End Latency**: Total time across all services in a request chain
- **Time to First Byte (TTFB)**: Time until first response byte

### Traffic
- **Request Rate**: Requests per second (RPS)
- **Throughput**: Bytes per second
- **Concurrent Requests**: Number of simultaneous requests/connections
- **Request Volume**: Total requests over time period

### Errors
- **Error Rate**: Errors per second
- **Error Percentage**: Percentage of failed requests
- **Error Types**: Breakdown by HTTP status code (4xx, 5xx)
- **Error Distribution**: Which endpoints/services are failing

### Saturation
- **Resource Utilization**: CPU, memory, disk, network usage
- **Queue Depth**: Number of items waiting to be processed
- **Connection Pool Usage**: Database/cache connection utilization
- **Thread Pool Exhaustion**: Availability of worker threads

---

## 2. Infrastructure & Resource Metrics

### CPU Metrics
- **CPU Usage Percentage**: Current CPU utilization
- **CPU Throttling**: When CPU is being throttled
- **CPU Load Average**: System load over 1, 5, 15 minutes
- **CPU Cores**: Number of available CPU cores

### Memory Metrics
- **Memory Usage**: Used vs. available memory
- **Memory Pressure**: Memory swap usage
- **Heap Usage**: For JVM/Go services
- **Memory Leaks**: Gradual increase in memory usage
- **Out of Memory (OOM) Events**: When services run out of memory

### Disk Metrics
- **Disk I/O**: Read/write operations per second
- **Disk Space Usage**: Available vs. used disk space
- **Disk Latency**: Time for disk operations
- **Disk Throughput**: Bytes read/written per second

### Network Metrics
- **Network I/O**: Bytes transmitted and received
- **Network Packet Loss**: Percentage of lost packets
- **Network Latency**: Round-trip time between services
- **Network Bandwidth**: Available vs. used bandwidth
- **Connection Count**: Active network connections

---

## 3. Application Metrics

### Request Metrics
- **Total Requests**: Cumulative request count
- **Requests by Method**: GET, POST, PUT, DELETE, etc.
- **Requests by Endpoint**: Per-route/endpoint breakdown
- **Requests by Status Code**: 200, 404, 500, etc.
- **Request Size**: Incoming request payload size
- **Response Size**: Outgoing response payload size

### Business Metrics
- **Active Users**: Current users interacting with the service
- **Transactions Processed**: Business transactions over time
- **Business Events**: 
  - User signups
  - Purchases
  - Feature usage
  - Custom business KPIs
- **Queue Length**: Number of tasks/items waiting in queues
- **Processing Time**: Time to complete business operations

---

## 4. Service Health & Availability

### Service Availability
- **Uptime Percentage**: Proportion of time service is available
- **Downtime Incidents**: Count and duration of outages
- **Health Check Status**: Real-time service health indicators
- **Service Discovery Status**: Registration with service mesh/registry

### Dependency Health
- **Database Connection Status**: Can service connect to database
- **External API Availability**: Third-party service availability
- **Cache Hit/Miss Rates**: Cache effectiveness
- **Third-Party Service Response Times**: External dependency latency
- **Circuit Breaker Status**: Open/closed state of circuit breakers

---

## 5. Database Metrics

### Connection Pool Metrics
- **Active Connections**: Currently in-use connections
- **Idle Connections**: Available but unused connections
- **Connection Wait Time**: Time waiting for available connections
- **Connection Pool Exhaustion**: When pool runs out of connections
- **Connection Pool Size**: Max vs. current connections

### Query Performance
- **Query Duration**: Time to execute queries
- **Slow Queries**: Queries exceeding threshold
- **Query Throughput**: Queries per second
- **Database Locks**: Lock wait time and deadlocks
- **Transaction Rate**: Transactions per second
- **Query Errors**: Failed query attempts

---

## 6. Cache Metrics

### Redis/Cache Performance
- **Hit Rate**: Percentage of cache hits
- **Miss Rate**: Percentage of cache misses
- **Eviction Rate**: How often items are evicted
- **Memory Usage**: Cache memory consumption
- **Connection Count**: Active cache connections
- **Cache Latency**: Time to get/set cache values
- **Cache Size**: Number of items in cache

---

## 7. Message Queue Metrics

### Queue Health
- **Queue Depth**: Number of messages waiting
- **Message Processing Rate**: Messages processed per second
- **Message Age**: How long messages wait before processing
- **Dead Letter Queue Size**: Failed messages
- **Consumer Lag**: Delay between message production and consumption
- **Producer Rate**: Messages produced per second

---

## 8. API Gateway Metrics

### Request Routing
- **Requests per Route**: Traffic distribution across routes
- **Requests per Service**: Backend service traffic
- **Route Latency**: Time for gateway to route requests
- **Rate Limiting**: Number of throttled requests
- **Authentication Failures**: Failed auth attempts

### Gateway Health
- **Gateway Uptime**: API gateway availability
- **Backend Service Availability**: Health of upstream services
- **SSL/TLS Certificate Expiration**: Certificate validity
- **Gateway Errors**: Gateway-level failures

---

## 9. Security Metrics

### Authentication/Authorization
- **Failed Login Attempts**: Unsuccessful authentication
- **Token Expiration Rates**: Token refresh patterns
- **Authorization Failures**: Permission denied events
- **Session Duration**: Average session length

### Security Events
- **Suspicious Activity**: Unusual access patterns
- **Rate Limit Violations**: Exceeded rate limits
- **DDoS Attack Indicators**: Unusual traffic spikes
- **IP Blocking Events**: Blocked IP addresses
- **Security Policy Violations**: Policy enforcement failures

---

## 10. Distributed Tracing Metrics

### Trace Metrics
- **Trace Duration**: End-to-end request time
- **Spans per Trace**: Number of service calls in a trace
- **Trace Sampling Rate**: Percentage of requests traced
- **Trace Errors**: Failed or slow traces

### Service Dependencies
- **Service Dependency Graph**: Visual representation of dependencies
- **Cross-Service Call Latency**: Inter-service communication time
- **Dependency Failure Impact**: Which services are affected by failures
- **Service Call Count**: Number of calls between services

---

## 11. Container/Orchestration Metrics

### Pod/Container Metrics (Kubernetes/Docker)
- **Container CPU Usage**: CPU consumption per container
- **Container Memory Usage**: Memory consumption per container
- **Container Restarts**: Number of container restarts
- **Pod Status**: Running, pending, failed, etc.
- **Container Logs**: Error and info logs

### Orchestration Metrics
- **Deployment Success Rate**: Successful vs. failed deployments
- **Rolling Update Status**: Progress of rolling updates
- **Resource Quotas**: CPU/memory limits and usage
- **Node Health**: Kubernetes node status
- **Replica Count**: Desired vs. actual replicas

---

## 12. Cost & Efficiency Metrics

### Resource Efficiency
- **Cost per Request**: Infrastructure cost divided by requests
- **Resource Utilization Efficiency**: How well resources are used
- **Auto-Scaling Effectiveness**: Scaling decisions and outcomes
- **Waste Metrics**: Unused or over-provisioned resources

---

## Priority Ranking

### 🔴 Critical (Monitor First)
These metrics are essential for basic system health and should be implemented immediately:

1. **Request Rate (Traffic)**: Requests per second
2. **Error Rate**: Failed requests percentage
3. **Latency**: P95 and P99 response times
4. **Service Availability/Uptime**: Service health status
5. **CPU & Memory Usage**: Resource consumption

### 🟠 High Priority
Important for performance optimization and capacity planning:

6. **Database Connection Pool**: Connection availability
7. **Cache Hit Rate**: Cache effectiveness
8. **Queue Depth**: Message queue health
9. **Network I/O**: Network utilization
10. **Disk I/O**: Storage performance

### 🟡 Medium Priority
Useful for deeper insights and optimization:

11. **Business Metrics**: Domain-specific KPIs
12. **Security Metrics**: Authentication and security events
13. **Distributed Tracing**: Request flow analysis
14. **Message Queue Metrics**: Queue performance

### 🟢 Nice to Have
Advanced metrics for optimization and cost management:

15. **Cost Metrics**: Resource cost analysis
16. **Advanced Orchestration Metrics**: Kubernetes-specific metrics

---

## Implementation Recommendations

### Phase 1: Foundation (Week 1-2)
Start with the **Golden Signals**:
- ✅ Request rate (traffic)
- ✅ Error rate
- ✅ Latency (P95, P99)
- ✅ Service availability/uptime

### Phase 2: Infrastructure (Week 3-4)
Add infrastructure monitoring:
- ✅ CPU and memory usage
- ✅ Network I/O
- ✅ Disk I/O
- ✅ Basic health checks

### Phase 3: Application Metrics (Week 5-6)
Implement application-specific metrics:
- ✅ Request counts by endpoint
- ✅ Request/response sizes
- ✅ Business metrics (if applicable)
- ✅ Database connection pool

### Phase 4: Dependencies (Week 7-8)
Monitor external dependencies:
- ✅ Database performance
- ✅ Cache metrics (hit/miss rates)
- ✅ External API availability
- ✅ Message queue health

### Phase 5: Advanced (Ongoing)
Add advanced monitoring:
- ✅ Distributed tracing
- ✅ Security metrics
- ✅ Cost optimization metrics
- ✅ Advanced orchestration metrics

---

## Prometheus Metric Naming Conventions

When implementing these metrics, follow Prometheus naming conventions:

- **Counters**: Use `_total` suffix (e.g., `requests_total`)
- **Gauges**: Use descriptive names (e.g., `memory_usage_bytes`)
- **Histograms**: Use `_bucket`, `_sum`, `_count` suffixes
- **Summaries**: Use `_sum`, `_count` suffixes

### Example Metric Names

```
# Request metrics
http_requests_total{method="GET",status="200",endpoint="/api/users"}
http_request_duration_seconds{method="GET",endpoint="/api/users",quantile="0.95"}

# Resource metrics
process_cpu_seconds_total
process_resident_memory_bytes

# Database metrics
db_connections_active
db_query_duration_seconds{query="SELECT"}

# Cache metrics
cache_hits_total{cache="redis"}
cache_misses_total{cache="redis"}
```

---

## Dashboard Organization

### Recommended Dashboard Structure

1. **Overview Dashboard**
   - Golden Signals summary
   - Service health status
   - Top errors and slow endpoints

2. **Service-Specific Dashboards**
   - Per-service metrics
   - Request rates and latency
   - Error breakdown

3. **Infrastructure Dashboard**
   - CPU, memory, disk, network
   - Container/pod metrics
   - Resource utilization trends

4. **Dependencies Dashboard**
   - Database performance
   - Cache metrics
   - External API health
   - Message queue status

5. **Business Metrics Dashboard**
   - Domain-specific KPIs
   - User activity
   - Transaction metrics

---

## Alerting Recommendations

### Critical Alerts (Immediate Response)
- Service down (uptime = 0)
- Error rate > 5%
- P99 latency > 1 second
- CPU usage > 90% for 5 minutes
- Memory usage > 95%

### Warning Alerts (Investigation Needed)
- Error rate > 1%
- P95 latency > 500ms
- CPU usage > 70% for 15 minutes
- Cache hit rate < 80%
- Queue depth > 1000

### Info Alerts (Monitoring)
- Deployment completed
- High traffic spike (>2x normal)
- Certificate expiring soon (30 days)

---

## References

- [Google SRE Book - Monitoring](https://sre.google/sre-book/monitoring-distributed-systems/)
- [Prometheus Best Practices](https://prometheus.io/docs/practices/naming/)
- [The Four Golden Signals](https://sre.google/sre-book/monitoring-distributed-systems/)
- [Microservices Monitoring Guide](https://www.datadoghq.com/knowledge-center/monitoring-microservices/)

---

## Last Updated

**Date**: December 2024  
**Version**: 1.0  
**Maintained by**: MetaRGB DevOps Team

