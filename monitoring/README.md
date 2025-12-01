# MetaRGB Monitoring Stack

This directory contains the Prometheus and Grafana configuration for monitoring MetaRGB microservices.

## Components

### Prometheus
- **Port**: 9090
- **URL**: http://localhost:9090
- **Config**: `prometheus/prometheus.yml`
- **Data**: Stored in Docker volume `prometheus_data`

### Grafana
- **Port**: 3001
- **URL**: http://localhost:3001
- **Default Credentials**:
  - Username: `admin`
  - Password: `admin123`
- **Data**: Stored in Docker volume `grafana_data`

## Quick Start

### Start Monitoring Stack
```bash
docker-compose up -d prometheus grafana
```

### Access Dashboards
1. **Prometheus**: http://localhost:9090
2. **Grafana**: http://localhost:3001
   - Login with `admin` / `admin123`
   - Navigate to Dashboards → MetaRGB Microservices Overview

### Verify Services
```bash
# Check Prometheus health
curl http://localhost:9090/-/healthy

# Check Grafana health
curl http://localhost:3001/api/health
```

## Configuration

### Prometheus Scrape Targets

Prometheus is configured to scrape metrics from:
- All microservices on port 9090 (metrics endpoint)
- Kong API Gateway on port 8001
- Health Check Service on port 8090

**Note**: Services need to expose metrics on port 9090 at `/metrics` endpoint.

### Grafana Dashboards

Pre-configured dashboards:
- **MetaRGB Microservices Overview**: Main dashboard showing service health, request rates, latency, and errors

### Adding Custom Dashboards

**Quick Method:**
1. Create dashboard JSON file in `monitoring/grafana/dashboards/`
2. Restart Grafana: `docker-compose restart grafana`
3. Dashboard will be auto-loaded

**Detailed Guide:** See `ADD_DASHBOARDS.md` for:
- Step-by-step instructions
- Multiple methods (file-based, UI import, export)
- Dashboard templates and examples
- PromQL query examples
- Troubleshooting tips

## Metrics Exposed

### Service Metrics (when implemented)
- `metargb_<service>_requests_total` - Total request count
- `metargb_<service>_request_duration_seconds` - Request duration histogram
- `metargb_<service>_requests_in_flight` - Current in-flight requests
- `metargb_<service>_db_connection_pool` - Database connection pool stats

### Infrastructure Metrics
- `up` - Service availability (1 = up, 0 = down)
- Kong metrics (when Kong exporter is configured)

## Troubleshooting

### Prometheus not scraping services
- Verify services expose metrics on port 9090
- Check Prometheus targets: http://localhost:9090/targets
- Review Prometheus logs: `docker logs metargb-prometheus`

### Grafana not showing data
- Verify Prometheus datasource is configured
- Check datasource connection: Grafana → Configuration → Data Sources
- Ensure services are exposing metrics

### Error: "dial tcp [::1]:9090: connect: connection refused"

**Problem**: Grafana can't connect to Prometheus using `localhost:9090`

**Solution**: 
1. Go to **Configuration → Data Sources → Prometheus**
2. Change **URL** from `http://localhost:9090` to `http://prometheus:9090`
3. Ensure **Access** is set to **Server (default)**
4. Click **Save & Test**

**Why**: Inside Docker containers, `localhost` refers to the container itself. Use the Docker service name `prometheus` instead.

See `FIX_CONNECTION_ERROR.md` for detailed steps.

## Connecting Grafana to Prometheus

### Automatic Configuration (Already Set Up)

The Prometheus datasource is automatically configured via provisioning files:
- **Config File**: `monitoring/grafana/datasources/prometheus.yml`
- **Prometheus URL**: `http://prometheus:9090` (Docker service name)
- **Status**: Automatically loaded when Grafana starts

### Manual Configuration via Web UI

If you need to manually configure or verify the connection:

1. **Access Grafana Web UI**
   - Open http://localhost:3001 in your browser
   - Login with credentials: `admin` / `admin123`

2. **Navigate to Data Sources**
   - Click the **⚙️ (Configuration)** icon in the left sidebar
   - Select **Data sources** from the menu

3. **Add/Edit Prometheus Data Source**
   - If Prometheus is already listed, click on it to edit
   - If not, click **Add data source** button
   - Select **Prometheus** from the list

4. **Configure Prometheus Connection**
   - **Name**: `Prometheus` (or any name you prefer)
   - **URL**: 
     - For Docker Compose: `http://prometheus:9090` (internal Docker network)
     - For external access: `http://localhost:9090` (if accessing from host)
   - **Access**: Select **Server (default)** or **Browser** based on your setup
   - **HTTP Method**: `POST` (recommended for better performance)

5. **Advanced Settings** (Optional)
   - **Scrape interval**: `15s` (should match Prometheus scrape interval)
   - **Query timeout**: `60s`
   - **HTTP Method**: `POST`
   - **Timeout**: `60s`

6. **Test Connection**
   - Scroll down and click **Save & Test** button
   - You should see a green success message: "Data source is working"

7. **Set as Default** (Optional)
   - Check the **Default** checkbox if you want this to be the default datasource
   - Click **Save & Test** again

### Verifying the Connection

**Method 1: Via Grafana UI**
1. Go to **Configuration → Data Sources**
2. Click on **Prometheus** datasource
3. Click **Save & Test** - should show green success message

**Method 2: Test Query**
1. Go to **Explore** (compass icon) in left sidebar
2. Select **Prometheus** from the datasource dropdown
3. Type a test query: `up`
4. Click **Run query** - should return results

**Method 3: Check Logs**
```bash
# Check Grafana logs for datasource errors
docker logs metargb-grafana | grep -i prometheus

# Check if Prometheus is accessible from Grafana container
docker exec metargb-grafana wget -O- http://prometheus:9090/api/v1/query?query=up
```

### Troubleshooting Connection Issues

**Issue: "dial tcp [::1]:9090: connect: connection refused"**

This error means Grafana is using `localhost:9090` instead of the Docker service name.

**Quick Fix:**
- Go to **Configuration → Data Sources → Prometheus**
- Change **URL** to: `http://prometheus:9090`
- Set **Access** to: **Server (default)**
- Click **Save & Test**

See `FIX_CONNECTION_ERROR.md` for detailed troubleshooting.

**Issue: "Data source is not working" error**

1. **Check Prometheus is running**
   ```bash
   docker ps | grep prometheus
   curl http://localhost:9090/-/healthy
   ```

2. **Verify network connectivity**
   - Both containers must be on the same Docker network (`metargb-network`)
   - Check `docker-compose.yml` network configuration

3. **Check URL format**
   - Use `http://prometheus:9090` (service name) when both are in Docker
   - Use `http://localhost:9090` only if accessing from host machine
   - Do NOT use `http://127.0.0.1:9090` from Grafana container

4. **Verify datasource file**
   ```bash
   # Check if provisioning file exists and is correct
   cat monitoring/grafana/datasources/prometheus.yml
   ```

5. **Restart Grafana to reload provisioning**
   ```bash
   docker-compose restart grafana
   ```

6. **Check Grafana logs**
   ```bash
   docker logs metargb-grafana
   ```

### Services not exposing metrics
Services need to:
1. Import `metargb/shared/pkg/metrics`
2. Start HTTP server on port 9090 with `/metrics` endpoint
3. Register Prometheus metrics handler

Example:
```go
import (
    "net/http"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

go func() {
    http.Handle("/metrics", promhttp.Handler())
    http.ListenAndServe(":9090", nil)
}()
```

## Data Retention

- **Prometheus**: 30 days (configurable in `prometheus.yml`)
- **Grafana**: Persistent (stored in Docker volume)

## Backup

To backup Grafana dashboards and Prometheus data:
```bash
# Backup Grafana
docker run --rm -v metargb_grafana_data:/data -v $(pwd):/backup alpine tar czf /backup/grafana-backup.tar.gz /data

# Backup Prometheus
docker run --rm -v metargb_prometheus_data:/data -v $(pwd):/backup alpine tar czf /backup/prometheus-backup.tar.gz /data
```

## Production Considerations

1. **Change default passwords** in `docker-compose.yml`
2. **Enable authentication** for Prometheus
3. **Configure alerting** rules in Prometheus
4. **Set up persistent storage** for production
5. **Configure resource limits** for containers
6. **Enable HTTPS** for Grafana
7. **Set up backup strategy** for metrics data

