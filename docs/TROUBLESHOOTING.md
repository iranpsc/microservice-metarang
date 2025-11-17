# Troubleshooting Guide

Common issues and solutions for MetaRGB microservices.

## Table of Contents
- [Service Issues](#service-issues)
- [Database Issues](#database-issues)
- [Network and Connectivity](#network-and-connectivity)
- [Performance Issues](#performance-issues)
- [Data Consistency](#data-consistency)
- [Monitoring and Observability](#monitoring-and-observability)

---

## Service Issues

### Service Pod Not Starting

**Symptoms**: Pod stuck in `CrashLoopBackOff` or `Error` state

**Diagnosis**:
```bash
# Check pod status
kubectl get pods -n metargb

# Check pod events
kubectl describe pod <pod-name> -n metargb

# Check logs
kubectl logs <pod-name> -n metargb
kubectl logs <pod-name> -n metargb --previous  # Previous container logs
```

**Common Causes**:

1. **Database Connection Failure**
   - Check `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` in secrets
   - Verify MySQL pod is running: `kubectl get pod mysql-0 -n metargb`
   - Test connection from pod:
     ```bash
     kubectl exec -it <pod-name> -n metargb -- sh
     mysql -h mysql-service -u metargb_user -p
     ```

2. **Missing Environment Variables**
   - Check ConfigMap: `kubectl get configmap shared-config -n metargb -o yaml`
   - Check Secrets: `kubectl get secret app-secrets -n metargb`
   - Verify env vars in deployment:
     ```bash
     kubectl get deployment <service-name> -n metargb -o yaml | grep -A 20 env:
     ```

3. **Resource Limits**
   - Check resource usage: `kubectl top pod <pod-name> -n metargb`
   - Check OOMKilled: `kubectl describe pod <pod-name> -n metargb | grep -i oom`
   - Increase limits in deployment YAML if needed

**Solution**:
```bash
# Fix secrets/config
kubectl create secret generic app-secrets --from-env-file=.env.production -n metargb --dry-run=client -o yaml | kubectl apply -f -

# Restart deployment
kubectl rollout restart deployment/<service-name> -n metargb

# Watch rollout
kubectl rollout status deployment/<service-name> -n metargb
```

### Service Not Responding

**Symptoms**: 503 Service Unavailable from Kong

**Diagnosis**:
```bash
# Check if pods are ready
kubectl get pods -n metargb -l app=<service-name>

# Check service endpoints
kubectl get endpoints <service-name> -n metargb

# Check if Istio sidecar is ready
kubectl logs <pod-name> -n metargb -c istio-proxy

# Test directly from another pod
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- sh
curl http://<service-name>:50051/health
```

**Common Causes**:

1. **Health Check Failing**
   - Check readiness probe: `kubectl describe pod <pod-name> -n metargb | grep -A 5 Readiness`
   - Fix health check endpoint in service code

2. **Istio Sidecar Issues**
   - Check sidecar logs: `kubectl logs <pod-name> -n metargb -c istio-proxy`
   - Restart pod to reinject sidecar:
     ```bash
     kubectl delete pod <pod-name> -n metargb
     ```

3. **Port Mismatch**
   - Verify service port matches container port
   - Check service definition: `kubectl get svc <service-name> -n metargb -o yaml`

**Solution**:
```bash
# Update deployment with correct ports/health checks
kubectl apply -f k8s/<service-name>/deployment.yaml

# Force pod recreation
kubectl rollout restart deployment/<service-name> -n metargb
```

---

## Database Issues

### Connection Pool Exhausted

**Symptoms**: "too many connections" errors in logs

**Diagnosis**:
```bash
# Check current connections
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p -e "SHOW PROCESSLIST;"

# Count connections per user
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p -e "SELECT user, COUNT(*) FROM information_schema.processlist GROUP BY user;"
```

**Solution**:
```bash
# Increase max_connections in MySQL
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p -e "SET GLOBAL max_connections=500;"

# Update MySQL ConfigMap
kubectl edit configmap mysql-config -n metargb
# Add: max_connections=500

# Reduce connection pool size in services (shared/pkg/db/connection.go)
# SetMaxOpenConns(50) → SetMaxOpenConns(30)
```

### Slow Queries

**Symptoms**: High p95/p99 latencies on database-heavy endpoints

**Diagnosis**:
```bash
# Enable slow query log
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p -e "SET GLOBAL slow_query_log=1; SET GLOBAL long_query_time=0.5;"

# Check slow queries
kubectl exec -it mysql-0 -n metargb -- tail -f /var/lib/mysql/slow-query.log

# Check missing indexes
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p metargb_production -e "SHOW INDEX FROM transactions;"
```

**Solution**:
```bash
# Add missing indexes
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p metargb_production << EOF
CREATE INDEX idx_transactions_user_created ON transactions(user_id, created_at);
CREATE INDEX idx_features_user_status ON features(user_id, status);
EOF

# Analyze tables
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p metargb_production -e "ANALYZE TABLE transactions, features, wallets;"
```

### Deadlocks

**Symptoms**: "Deadlock found when trying to get lock" errors

**Diagnosis**:
```bash
# Check recent deadlocks
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p -e "SHOW ENGINE INNODB STATUS\G" | grep -A 50 "LATEST DETECTED DEADLOCK"
```

**Solution**:
- Ensure consistent lock ordering in code (always lock wallets in same order by user_id)
- Use `SELECT ... FOR UPDATE` explicitly
- Reduce transaction duration
- Implement retry logic with exponential backoff

---

## Network and Connectivity

### Cannot Reach External APIs

**Symptoms**: OAuth callbacks fail, Parsian payments fail, SMS not sending

**Diagnosis**:
```bash
# Check egress from pod
kubectl exec -it <pod-name> -n metargb -- sh
curl -v https://oauth-provider.com
curl -v https://pec.shaparak.ir

# Check Istio egress
kubectl get serviceentry -n metargb
kubectl logs -l app=istio-egressgateway -n istio-system
```

**Solution**:
```bash
# Create ServiceEntry for external APIs
cat <<EOF | kubectl apply -f -
apiVersion: networking.istio.io/v1beta1
kind: ServiceEntry
metadata:
  name: external-apis
  namespace: metargb
spec:
  hosts:
  - oauth-provider.com
  - pec.shaparak.ir
  - api.kavenegar.com
  ports:
  - number: 443
    name: https
    protocol: HTTPS
  location: MESH_EXTERNAL
  resolution: DNS
EOF
```

### gRPC Connection Refused

**Symptoms**: "connection refused" when service A calls service B

**Diagnosis**:
```bash
# Check if target service is running
kubectl get pods -l app=<target-service> -n metargb

# Check service DNS
kubectl exec -it <pod-name> -n metargb -- nslookup <target-service>

# Check Istio routing
kubectl get virtualservice <target-service> -n metargb -o yaml
kubectl get destinationrule <target-service> -n metargb -o yaml
```

**Solution**:
```bash
# Verify service name matches deployment
kubectl get svc -n metargb

# Update service call to use correct FQDN
# commercial-service → commercial-service.metargb.svc.cluster.local

# Check network policies
kubectl get networkpolicies -n metargb
```

### Circuit Breaker Triggered

**Symptoms**: 503 errors after 5xx errors from service

**Diagnosis**:
```bash
# Check Istio metrics
kubectl exec -it <pod-name> -n metargb -c istio-proxy -- curl localhost:15000/stats | grep outlier

# Check destination rule
kubectl get destinationrule <service-name> -n metargb -o yaml | grep -A 10 outlierDetection
```

**Solution**:
```bash
# Temporarily disable circuit breaker for testing
kubectl edit destinationrule <service-name> -n metargb
# Set consecutive5xxErrors to a higher value or remove outlierDetection

# Or fix underlying service issue first
```

---

## Performance Issues

### High Latency (p95 > 500ms)

**Diagnosis**:
```bash
# Check service metrics
kubectl port-forward -n monitoring svc/grafana 3000:3000
# Open http://localhost:3000 → MetaRGB Dashboard

# Check traces
kubectl port-forward -n istio-system svc/jaeger-query 16686:16686
# Open http://localhost:16686 → Search for slow traces

# Check resource usage
kubectl top pods -n metargb
kubectl top nodes
```

**Common Causes**:

1. **N+1 Query Problem**
   - Check traces for multiple DB queries per request
   - Solution: Use eager loading / JOIN queries

2. **External API Timeouts**
   - Set shorter timeouts on HTTP clients (default: 30s → 5s)
   - Implement circuit breakers

3. **Insufficient Resources**
   - Increase CPU/memory limits
   - Scale horizontally (increase replicas)

**Solution**:
```bash
# Scale up replicas
kubectl scale deployment/<service-name> --replicas=5 -n metargb

# Increase resources
kubectl set resources deployment/<service-name> -n metargb \
  --limits=cpu=2,memory=4Gi \
  --requests=cpu=1,memory=2Gi

# Enable HPA
kubectl autoscale deployment/<service-name> -n metargb \
  --cpu-percent=70 --min=3 --max=10
```

### Memory Leak

**Symptoms**: Memory usage continuously increases, OOMKilled

**Diagnosis**:
```bash
# Monitor memory over time
kubectl top pod <pod-name> -n metargb --containers

# Check for goroutine leaks (Go services)
kubectl exec -it <pod-name> -n metargb -- wget -O- http://localhost:6060/debug/pprof/goroutine

# Get heap profile
kubectl exec -it <pod-name> -n metargb -- wget -O- http://localhost:6060/debug/pprof/heap > heap.prof

# Analyze locally
go tool pprof heap.prof
```

**Solution**:
- Fix leaks in code (unclosed DB connections, goroutines)
- Add `/debug/pprof` endpoints in dev
- Set `GOGC` environment variable to trigger GC more frequently

---

## Data Consistency

### Wallet Balance Mismatch

**Symptoms**: Balance doesn't match sum of transactions

**Diagnosis**:
```bash
# Run consistency check
cd tests/database
go test -v -run TestDataConsistency

# Manual check
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p metargb_production << EOF
SELECT 
  u.id,
  u.username,
  w.psc AS wallet_balance,
  COALESCE(SUM(CASE WHEN t.type IN ('deposit', 'credit') THEN t.amount ELSE -t.amount END), 0) AS transaction_sum
FROM users u
JOIN wallets w ON u.id = w.user_id
LEFT JOIN transactions t ON u.id = t.user_id AND t.status = 'completed'
GROUP BY u.id
HAVING ABS(wallet_balance - transaction_sum) > 0.01;
EOF
```

**Solution**:
```bash
# If inconsistency found, create reconciliation script
# DO NOT auto-fix in production without investigation

# Log discrepancies
kubectl logs -l app=commercial-service -n metargb | grep "balance_mismatch"

# Investigate root cause (race condition, failed transaction commit, etc.)
```

### Lost Transactions

**Symptoms**: Payment succeeded but transaction not recorded

**Diagnosis**:
```bash
# Check Parsian callback logs
kubectl logs -l app=commercial-service -n metargb | grep "parsian_callback"

# Check idempotency (duplicate callbacks)
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p metargb_production -e "SELECT id, COUNT(*) FROM transactions GROUP BY id HAVING COUNT(*) > 1;"
```

**Solution**:
- Implement idempotency keys for payment callbacks
- Add retry logic for transaction creation
- Monitor callback failures with alerting

---

## Monitoring and Observability

### Metrics Not Showing in Grafana

**Diagnosis**:
```bash
# Check if Prometheus is scraping
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Open http://localhost:9090/targets

# Check service monitors
kubectl get servicemonitor -n monitoring

# Check if services expose metrics
kubectl exec -it <pod-name> -n metargb -- curl localhost:9090/metrics
```

**Solution**:
```bash
# Ensure services have metrics endpoint
# shared/pkg/metrics/metrics.go should register handlers

# Update ServiceMonitor
kubectl apply -f k8s/monitoring/prometheus/service-monitors.yaml

# Reload Prometheus
kubectl rollout restart deployment prometheus -n monitoring
```

### Traces Not Showing in Jaeger

**Diagnosis**:
```bash
# Check Jaeger collector
kubectl logs -l app=jaeger-collector -n istio-system

# Check if services are instrumented
kubectl logs <pod-name> -n metargb | grep "trace_id"

# Check Istio trace headers
kubectl exec -it <pod-name> -n metargb -c istio-proxy -- curl -v localhost:15000/stats | grep tracing
```

**Solution**:
```bash
# Increase sampling rate (temporarily)
kubectl edit configmap istio -n istio-system
# Set: tracing.sampling=100

# Restart Istio sidecars
kubectl rollout restart deployment -n metargb
```

### Alerts Not Firing

**Diagnosis**:
```bash
# Check Alertmanager
kubectl logs -l app=alertmanager -n monitoring

# Check alert rules
kubectl get prometheusrule -n monitoring
kubectl describe prometheusrule metargb-alerts -n monitoring

# Check current alerts
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Open http://localhost:9090/alerts
```

**Solution**:
```bash
# Update alert rules
kubectl apply -f k8s/monitoring/prometheus/alerting-rules.yaml

# Reload Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090
curl -X POST http://localhost:9090/-/reload
```

---

## Emergency Procedures

### Complete Service Outage

1. **Check cluster health**:
   ```bash
   kubectl get nodes
   kubectl get pods --all-namespaces
   ```

2. **Rollback to Laravel** (if microservices are down):
   ```bash
   kubectl apply -f k8s/kong/laravel-only.yaml
   kubectl scale deployment laravel --replicas=5 -n legacy
   ```

3. **Check recent changes**:
   ```bash
   kubectl rollout history deployment/<service-name> -n metargb
   kubectl rollout undo deployment/<service-name> -n metargb
   ```

### Database Corruption

1. **Stop all writes**:
   ```bash
   kubectl scale deployment --all --replicas=0 -n metargb
   ```

2. **Restore from backup**:
   ```bash
   kubectl exec -it mysql-0 -n metargb -- sh
   mysql -u root -p metargb_production < /backups/latest.sql
   ```

3. **Verify integrity**:
   ```bash
   cd tests/database
   go test -v -run TestSchemaGuard
   ```

4. **Resume services**:
   ```bash
   kubectl scale deployment --all --replicas=3 -n metargb
   ```

---

## Getting Help

### Collect Diagnostic Bundle

```bash
#!/bin/bash
# collect-diagnostics.sh

mkdir -p diagnostics
cd diagnostics

# Cluster info
kubectl cluster-info dump > cluster-info.txt

# Pods
kubectl get pods --all-namespaces -o wide > pods.txt
kubectl get pods -n metargb -o yaml > pods-metargb.yaml

# Services
kubectl get svc --all-namespaces > services.txt

# Logs (last 1000 lines)
for pod in $(kubectl get pods -n metargb -o name); do
  kubectl logs $pod -n metargb --tail=1000 > ${pod}-logs.txt
done

# Events
kubectl get events --all-namespaces --sort-by='.lastTimestamp' > events.txt

# Istio
istioctl analyze -n metargb > istio-analyze.txt
istioctl proxy-status > istio-proxy-status.txt

# Metrics (current)
kubectl top nodes > metrics-nodes.txt
kubectl top pods -n metargb > metrics-pods.txt

tar -czf diagnostics-$(date +%Y%m%d-%H%M%S).tar.gz *.txt *.yaml

echo "Diagnostics collected: $(ls -lh *.tar.gz)"
```

Run and share with support team.

