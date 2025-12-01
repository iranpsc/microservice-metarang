# Health Check Service

A comprehensive health monitoring service that implements **Section #4: Service Health & Availability** metrics from the MICROSERVICE_METRICS.md documentation.

## Features

### Service Availability Metrics

1. **Uptime Percentage**: Tracks the percentage of time each service has been available
2. **Downtime Incidents**: Counts and tracks duration of service outages
3. **Health Check Status**: Real-time health indicators for all services
4. **Service Discovery Status**: Tracks service registration (if using service mesh/registry)

### Dependency Health Metrics

1. **Database Connection Status**: 
   - Actual database connectivity (not just TCP)
   - Connection pool statistics (open, in-use, idle connections)
   - Connection latency

2. **Cache Metrics**:
   - Cache hit rate percentage
   - Cache miss rate percentage
   - Total hits and misses
   - Memory usage

3. **External API Availability**:
   - Health checks for external APIs (e.g., Parsian payment gateway)
   - Response latency tracking

4. **Third-Party Service Response Times**:
   - Latency tracking for third-party services

5. **Circuit Breaker Status**:
   - Circuit breaker state (if Istio is configured)

## Endpoints

### GET /health
Returns comprehensive health status in JSON format:
- Overall system status
- Individual service health
- Dependency health (database, cache, external APIs)
- Service availability metrics (uptime, downtime incidents)

### GET /metrics
Exposes Prometheus metrics for all monitored services and dependencies.

## Prometheus Metrics Exposed

### Service Health Metrics
- `service_health_status` - Service health status (1=healthy, 0=unhealthy)
- `service_health_total` - Total number of services checked
- `service_health_healthy` - Number of healthy services
- `service_health_unhealthy` - Number of unhealthy services

### Service Availability Metrics
- `service_uptime_percentage` - Service uptime percentage (0-100)
- `service_uptime_seconds_total` - Total uptime in seconds
- `service_downtime_seconds_total` - Total downtime in seconds
- `service_downtime_incidents_total` - Total number of downtime incidents

### Database Metrics
- `db_connection_status` - Database connection status (1=connected, 0=disconnected)
- `db_connection_latency_seconds` - Database connection latency
- `db_connection_pool_open` - Open connections in pool
- `db_connection_pool_in_use` - In-use connections in pool
- `db_connection_pool_idle` - Idle connections in pool

### Cache Metrics
- `cache_status` - Cache status (1=healthy, 0=unhealthy)
- `cache_hit_rate` - Cache hit rate percentage
- `cache_miss_rate` - Cache miss rate percentage
- `cache_hits_total` - Total cache hits
- `cache_misses_total` - Total cache misses
- `cache_memory_usage_bytes` - Cache memory usage in bytes

### External API Metrics
- `external_api_status` - External API status (1=healthy, 0=unhealthy)

## Configuration

The service can be configured via environment variables:

- `REDIS_URL` - Redis connection URL (default: `redis://redis:6379`)
- `DB_HOST` - Database host (default: `mysql`)
- `DB_PORT` - Database port (default: `3306`)
- `DB_USER` - Database user (default: `metargb_user`)
- `DB_PASSWORD` - Database password (default: `metargb_password`)
- `DB_DATABASE` - Database name (default: `metargb_db`)
- `PARSIAN_API_URL` - Parsian payment gateway URL (optional)
- `ISTIO_METRICS_URL` - Istio metrics endpoint URL (optional)

## Usage

### Docker Compose
The service is automatically included in the docker-compose.yml file.

### Standalone
```bash
cd services/health-check-service
go run main.go
```

The service will start on port 8090.

## Implementation Details

### Uptime Tracking
- Services are tracked continuously in the background
- Uptime and downtime are calculated based on service status changes
- Downtime incidents are recorded with start/end times and duration

### Database Health Checks
- Performs actual database queries (PING) to verify connectivity
- Tracks connection pool statistics
- Monitors connection latency

### Cache Metrics
- Queries Redis INFO command to get cache statistics
- Calculates hit/miss rates from keyspace statistics
- Monitors memory usage

### External API Monitoring
- Performs HTTP health checks on configured external APIs
- Tracks response latency
- Records last check timestamp

## Integration with Prometheus

The service is automatically scraped by Prometheus (configured in `monitoring/prometheus/prometheus.yml`). All metrics follow Prometheus naming conventions and are ready for use in Grafana dashboards.

## Example Health Response

```json
{
  "status": "healthy",
  "timestamp": "2024-12-19T10:30:00Z",
  "uptime": "3600s",
  "services": [
    {
      "service": "MySQL",
      "status": "healthy",
      "host": "mysql",
      "port": 3306,
      "latency": "5ms"
    }
  ],
  "dependencies": {
    "database_connection": {
      "status": "healthy",
      "host": "mysql",
      "port": 3306,
      "database": "metargb_db",
      "connected": true,
      "latency": "5ms",
      "pool_stats": {
        "open_connections": 5,
        "in_use": 2,
        "idle": 3
      }
    },
    "cache_metrics": {
      "status": "healthy",
      "hit_rate": 85.5,
      "miss_rate": 14.5,
      "hits": 8550,
      "misses": 1450,
      "memory_usage_bytes": 1048576
    }
  },
  "service_availability": {
    "MySQL": {
      "uptime_percentage": 99.95,
      "total_uptime": "3598s",
      "total_downtime": "2s",
      "downtime_incidents": 1,
      "current_status": "healthy"
    }
  }
}
```

