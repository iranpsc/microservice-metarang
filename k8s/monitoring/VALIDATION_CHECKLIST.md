# Phase 6 Validation Checklist

Use this checklist to verify Phase 6 deployment is successful.

## Pre-Deployment Checklist

- [ ] Kubernetes cluster is running (v1.24+)
- [ ] kubectl is configured and connected to cluster
- [ ] istioctl is installed (or will be installed automatically)
- [ ] All Phase 1-5 services are deployed to `metargb` namespace
- [ ] Sufficient cluster resources available (see resource requirements)

## Deployment Verification

### 1. Istio Service Mesh

**Check Istio Installation:**
```bash
istioctl version
# Should show both client and control plane versions
```

- [ ] Istio control plane is running
- [ ] istioctl command works

**Check Istio Pods:**
```bash
kubectl get pods -n istio-system
```

Expected pods:
- [ ] `istiod-*` (1/1 Running)
- [ ] `istio-ingressgateway-*` (1/1 Running)
- [ ] `istio-egressgateway-*` (1/1 Running)
- [ ] `jaeger-*` (1/1 Running)

**Check Namespace Injection:**
```bash
kubectl get namespace metargb -o yaml | grep istio-injection
```

- [ ] Label `istio-injection: enabled` is present

**Check Sidecar Injection:**
```bash
kubectl get pods -n metargb -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].name}{"\n"}{end}'
```

- [ ] All pods show `istio-proxy` container

**Check mTLS Configuration:**
```bash
kubectl get peerauthentication -n metargb
```

- [ ] PeerAuthentication `default` exists with mode STRICT

**Check Traffic Management:**
```bash
kubectl get virtualservices -n metargb
kubectl get destinationrules -n metargb
```

- [ ] 8 VirtualServices exist (one per service)
- [ ] 8 DestinationRules exist (one per service)

### 2. Prometheus

**Check Prometheus Pod:**
```bash
kubectl get pods -n monitoring -l app=prometheus
```

- [ ] Prometheus pod is Running (1/1)

**Check Prometheus Storage:**
```bash
kubectl get pvc -n monitoring
```

- [ ] PVC `prometheus-storage` is Bound (50Gi)

**Check Prometheus Targets:**
```bash
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# Visit: http://localhost:9090/targets
```

- [ ] All service targets are UP
- [ ] Kubernetes API targets are UP
- [ ] Istio mesh targets are UP

**Test Prometheus Query:**
```bash
curl http://localhost:9090/api/v1/query?query=up
```

- [ ] Query returns valid JSON with data

**Check Alerting Rules:**
```bash
# Visit: http://localhost:9090/rules
```

- [ ] 7 rule groups are loaded
- [ ] No errors in rule evaluation

### 3. Grafana

**Check Grafana Pod:**
```bash
kubectl get pods -n monitoring -l app=grafana
```

- [ ] Grafana pod is Running (1/1)

**Check Grafana Storage:**
```bash
kubectl get pvc -n monitoring
```

- [ ] PVC `grafana-storage` is Bound (10Gi)

**Access Grafana:**
```bash
kubectl port-forward -n monitoring svc/grafana 3000:3000
# Visit: http://localhost:3000
```

- [ ] Grafana UI loads successfully
- [ ] Can login with admin/changeme123!

**Check Data Sources:**
```
Grafana → Configuration → Data Sources
```

- [ ] Prometheus data source exists
- [ ] Prometheus data source test is successful
- [ ] Jaeger data source exists

**Check Dashboards:**
```
Grafana → Dashboards → Browse
```

- [ ] "Service Overview" dashboard exists and loads
- [ ] "Auth Service" dashboard exists and loads
- [ ] "Commercial Service" dashboard exists and loads
- [ ] "Features Service" dashboard exists and loads
- [ ] "Database Metrics" dashboard exists and loads
- [ ] All dashboards show data (if services are running)

### 4. Jaeger

**Check Jaeger Pod:**
```bash
kubectl get pods -n istio-system -l app=jaeger
```

- [ ] Jaeger pod is Running (1/1)

**Check Jaeger Storage:**
```bash
kubectl get pvc -n istio-system
```

- [ ] PVC `jaeger-badger-storage` is Bound (20Gi)

**Check Jaeger Services:**
```bash
kubectl get svc -n istio-system | grep jaeger
```

- [ ] `jaeger-query` service exists
- [ ] `jaeger-collector` service exists
- [ ] `jaeger-agent` service exists

**Access Jaeger:**
```bash
kubectl port-forward -n istio-system svc/jaeger-query 16686:16686
# Visit: http://localhost:16686
```

- [ ] Jaeger UI loads successfully
- [ ] Service list shows MetaRGB services
- [ ] Can search for traces (if traffic exists)

**Check Tracing Configuration:**
```bash
kubectl get telemetry -n istio-system
kubectl get telemetry -n metargb
```

- [ ] Telemetry configurations exist with Jaeger provider

### 5. ServiceMonitors

**Check ServiceMonitors:**
```bash
kubectl get servicemonitor -n metargb
```

Expected ServiceMonitors:
- [ ] `auth-service`
- [ ] `commercial-service`
- [ ] `features-service`
- [ ] `levels-service`
- [ ] `dynasty-service`
- [ ] `support-service`
- [ ] `calendar-service`
- [ ] `storage-service`

### 6. Metrics Collection

**Test Service Metrics:**
```bash
# In Prometheus UI (http://localhost:9090)
# Run query: grpc_server_handled_total{namespace="metargb"}
```

- [ ] Metrics are being collected from services

**Test Istio Metrics:**
```bash
# In Prometheus UI
# Run query: istio_requests_total
```

- [ ] Istio mesh metrics are being collected

### 7. Alerting

**Check Alert Rules:**
```bash
# Visit: http://localhost:9090/rules
```

Alert groups should exist:
- [ ] service_availability
- [ ] error_rates
- [ ] latency
- [ ] resource_usage
- [ ] database
- [ ] circuit_breaker
- [ ] websocket
- [ ] storage

**Trigger Test Alert:**
```bash
# Scale down a service to trigger ServiceDown alert
kubectl scale deployment auth-service -n metargb --replicas=0

# Wait 2-3 minutes, then check alerts
# Visit: http://localhost:9090/alerts
```

- [ ] Alert fires after configured time
- [ ] Alert appears in Grafana

**Restore Service:**
```bash
kubectl scale deployment auth-service -n metargb --replicas=2
```

### 8. End-to-End Verification

**Generate Sample Traffic:**
```bash
# Make requests to your services through Kong or directly
# This generates metrics and traces
```

**Verify Metrics Pipeline:**
```bash
# 1. Check metrics in Prometheus
# Visit: http://localhost:9090
# Query: sum(rate(grpc_server_handled_total{namespace="metargb"}[5m]))

# 2. Check dashboard in Grafana
# Visit: http://localhost:3000
# Open: Service Overview Dashboard

# 3. Should see request rate > 0
```

- [ ] Metrics flow from service → Prometheus → Grafana

**Verify Tracing Pipeline:**
```bash
# 1. Make a request through your service
# 2. Wait 10-30 seconds
# 3. Check Jaeger UI
# Visit: http://localhost:16686
# Search for recent traces
```

- [ ] Traces appear in Jaeger
- [ ] Spans show correct service hierarchy
- [ ] Trace duration is reasonable

**Verify Circuit Breaker:**
```bash
# 1. Deploy a failing service or make it return errors
# 2. Send multiple requests (>10)
# 3. Check Prometheus
# Query: istio_requests_total{response_code=~"5..",destination_workload_namespace="metargb"}
```

- [ ] Error rate increases
- [ ] Circuit breaker alert fires (if threshold met)

### 9. Security Verification

**Check mTLS:**
```bash
istioctl authn tls-check <pod-name>.<namespace>
```

- [ ] All connections show "STRICT" mode

**Check Secrets:**
```bash
kubectl get secrets -n monitoring
```

- [ ] `grafana-credentials` secret exists
- [ ] No plaintext passwords in YAML files

### 10. Resource Usage

**Check Resource Consumption:**
```bash
kubectl top pods -n istio-system
kubectl top pods -n monitoring
kubectl top pods -n metargb
```

Expected approximate usage:
- [ ] Istio control plane: ~500Mi memory, 200m CPU
- [ ] Prometheus: ~2Gi memory, 500m CPU
- [ ] Grafana: ~128Mi memory, 100m CPU
- [ ] Jaeger: ~512Mi memory, 200m CPU
- [ ] Sidecars: ~50Mi memory, 10m CPU each

**Check Storage Usage:**
```bash
kubectl exec -n monitoring -it deploy/prometheus -- df -h /prometheus
kubectl exec -n istio-system -it deploy/jaeger -- df -h /badger
```

- [ ] Storage usage is within expected range
- [ ] Sufficient space available

## Post-Deployment Actions

**Security:**
- [ ] Change Grafana admin password
- [ ] Review and rotate any default credentials
- [ ] Configure RBAC for monitoring access

**Configuration:**
- [ ] Adjust Jaeger sampling rate for production (10%)
- [ ] Configure alert notification channels (Slack, email)
- [ ] Set up Prometheus remote write (optional)
- [ ] Configure Grafana SSO (optional)

**Documentation:**
- [ ] Document team access procedures
- [ ] Create runbooks for common alerts
- [ ] Document monitoring architecture
- [ ] Share dashboard URLs with team

**Training:**
- [ ] Train team on Grafana dashboards
- [ ] Demonstrate Jaeger trace analysis
- [ ] Review alert response procedures
- [ ] Share Prometheus query examples

## Troubleshooting

If any checks fail, see:
- `../PHASE6_IMPLEMENTATION.md` - Troubleshooting section
- `kubectl logs -n <namespace> <pod-name>` - Pod logs
- `kubectl describe pod -n <namespace> <pod-name>` - Pod events
- `kubectl get events -n <namespace>` - Namespace events

## Cleanup (if needed)

If deployment fails and you need to start over:

```bash
make phase6-cleanup
# Or
./scripts/phase6-deploy.sh --cleanup
```

## Success Criteria

Phase 6 is successfully deployed when:

✅ All pods in istio-system namespace are Running
✅ All pods in monitoring namespace are Running
✅ Sidecars injected in all metargb pods
✅ Prometheus shows all targets as UP
✅ Grafana dashboards load and display data
✅ Jaeger shows traces from services
✅ Alerts are defined and functional
✅ mTLS is enforced (STRICT mode)
✅ Circuit breakers are configured
✅ No errors in pod logs

---

**Validation Status**: [ ] Complete

**Validated By**: _______________

**Date**: _______________

**Notes**:

