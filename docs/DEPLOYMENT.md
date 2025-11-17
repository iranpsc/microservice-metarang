# Deployment Runbook

Complete guide for deploying MetaRGB microservices to production.

## Prerequisites

### Required Tools
```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install istioctl
curl -L https://istio.io/downloadIstio | sh -
export PATH=$PATH:$HOME/.istioctl/bin

# Install k6 (for load testing)
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

### Required Secrets
Create a `.env.production` file:
```bash
DB_HOST=mysql-service.metargb.svc.cluster.local
DB_PORT=3306
DB_DATABASE=metargb_production
DB_USER=metargb_user
DB_PASSWORD=<SECURE_PASSWORD>

REDIS_HOST=redis-service.metargb.svc.cluster.local
REDIS_PORT=6379

OAUTH_CLIENT_ID=<CLIENT_ID>
OAUTH_CLIENT_SECRET=<CLIENT_SECRET>

PARSIAN_PIN=<PARSIAN_PIN>
PARSIAN_TERMINAL_ID=<TERMINAL_ID>

KAVENEGAR_API_KEY=<KAVENEGAR_KEY>

FTP_HOST=<FTP_HOST>
FTP_USER=<FTP_USER>
FTP_PASSWORD=<FTP_PASSWORD>
```

## Phase 1: Infrastructure Setup

### Step 1.1: Kubernetes Cluster
```bash
# For cloud providers (example: GKE)
gcloud container clusters create metargb-prod \
  --zone us-central1-a \
  --num-nodes 3 \
  --machine-type n1-standard-4 \
  --enable-autoscaling \
  --min-nodes 3 \
  --max-nodes 10

# Get credentials
gcloud container clusters get-credentials metargb-prod --zone us-central1-a

# Verify
kubectl cluster-info
kubectl get nodes
```

### Step 1.2: Create Namespace
```bash
cd k8s
kubectl apply -f namespace.yaml

# Set default namespace
kubectl config set-context --current --namespace=metargb
```

### Step 1.3: Deploy MySQL
```bash
# Create secrets
kubectl create secret generic mysql-secrets \
  --from-literal=root-password=<ROOT_PASSWORD> \
  --from-literal=user-password=<USER_PASSWORD> \
  -n metargb

# Deploy MySQL StatefulSet
kubectl apply -f mysql-statefulset.yaml

# Wait for MySQL to be ready
kubectl wait --for=condition=ready pod/mysql-0 -n metargb --timeout=300s

# Import schema
kubectl exec -it mysql-0 -n metargb -- mysql -u root -p < ../scripts/schema.sql
```

### Step 1.4: Deploy Redis
```bash
kubectl apply -f redis-deployment.yaml

kubectl wait --for=condition=ready pod -l app=redis -n metargb --timeout=120s
```

### Step 1.5: Create ConfigMaps and Secrets
```bash
# Create from .env.production
kubectl create secret generic app-secrets \
  --from-env-file=../.env.production \
  -n metargb

# Create shared config
kubectl apply -f config/configmap.yaml
```

## Phase 2: Deploy Services

### Step 2.1: Build Docker Images
```bash
cd ..

# Build all services
make build-all

# Or build individually
docker build -t metargb/auth-service:v1.0.0 services/auth-service/
docker build -t metargb/commercial-service:v1.0.0 services/commercial-service/
# ... etc

# Push to registry
docker push metargb/auth-service:v1.0.0
docker push metargb/commercial-service:v1.0.0
# ... etc
```

### Step 2.2: Deploy Services
```bash
cd k8s

# Deploy in order (dependencies first)
kubectl apply -f auth-service/deployment.yaml
kubectl apply -f commercial-service/deployment.yaml
kubectl apply -f features-service/deployment.yaml
kubectl apply -f levels-service/deployment.yaml
kubectl apply -f dynasty-service/deployment.yaml
kubectl apply -f support-service/deployment.yaml
kubectl apply -f training-service/deployment.yaml
kubectl apply -f notifications-service/deployment.yaml
kubectl apply -f calendar-service/deployment.yaml
kubectl apply -f storage-service/deployment.yaml
kubectl apply -f websocket-gateway/deployment.yaml

# Wait for all deployments
kubectl wait --for=condition=available deployment --all -n metargb --timeout=600s

# Verify
kubectl get pods -n metargb
kubectl get services -n metargb
```

## Phase 3: Service Mesh (Istio)

### Step 3.1: Install Istio
```bash
cd k8s/istio

# Install Istio
istioctl install -f istio-install.yaml -y

# Enable sidecar injection
kubectl apply -f namespace-injection.yaml

# Restart pods to inject sidecars
kubectl rollout restart deployment -n metargb
```

### Step 3.2: Configure Istio
```bash
# Apply mTLS
kubectl apply -f peer-authentication.yaml

# Apply traffic management
kubectl apply -f virtual-services.yaml
kubectl apply -f destination-rules.yaml

# Verify Istio
istioctl analyze -n metargb
kubectl get virtualservices -n metargb
kubectl get destinationrules -n metargb
```

## Phase 4: API Gateway (Kong)

### Step 4.1: Deploy Kong
```bash
cd k8s/kong

# Install via Helm
helm repo add kong https://charts.konghq.com
helm repo update

helm install kong kong/kong \
  --namespace metargb \
  --set proxy.type=LoadBalancer \
  --set ingressController.installCRDs=false \
  --set env.database=off \
  --set env.declarative_config=/kong/kong.yml

# Wait for LoadBalancer IP
kubectl get svc kong-kong-proxy -n metargb -w
```

### Step 4.2: Configure Kong Routes
```bash
# Apply Kong configuration
kubectl create configmap kong-config \
  --from-file=kong.yml=../../kong/kong.yml \
  -n metargb

# Reload Kong
kubectl rollout restart deployment kong-kong -n metargb
```

## Phase 5: Monitoring

### Step 5.1: Deploy Prometheus
```bash
cd k8s/monitoring/prometheus

kubectl apply -f namespace.yaml
kubectl apply -f prometheus-deployment.yaml
kubectl apply -f service-monitors.yaml
kubectl apply -f alerting-rules.yaml

kubectl wait --for=condition=ready pod -l app=prometheus -n monitoring --timeout=300s
```

### Step 5.2: Deploy Grafana
```bash
cd ../grafana

kubectl apply -f grafana-deployment.yaml
kubectl apply -f dashboards-configmap.yaml

kubectl wait --for=condition=ready pod -l app=grafana -n monitoring --timeout=300s

# Get Grafana password
kubectl get secret grafana-admin -n monitoring -o jsonpath="{.data.password}" | base64 --decode
```

### Step 5.3: Deploy Jaeger
```bash
cd ../jaeger

kubectl apply -f jaeger-deployment.yaml
kubectl apply -f jaeger-istio-config.yaml

kubectl wait --for=condition=ready pod -l app=jaeger -n istio-system --timeout=300s
```

## Phase 6: Testing

### Step 6.1: Smoke Tests
```bash
# Get Kong external IP
KONG_IP=$(kubectl get svc kong-kong-proxy -n metargb -o jsonpath='{.status.loadBalancer.ingress[0].ip}')

# Test auth endpoint
curl -X POST http://$KONG_IP/api/auth/me \
  -H "Authorization: Bearer test_token"

# Test features endpoint
curl http://$KONG_IP/api/features?bbox=35.0,51.0,36.0,52.0 \
  -H "Authorization: Bearer test_token"

# Test wallet endpoint
curl http://$KONG_IP/api/user/wallet \
  -H "Authorization: Bearer test_token"
```

### Step 6.2: Integration Tests
```bash
cd tests/integration

# Update test config with production endpoints
export KONG_URL=http://$KONG_IP
export TEST_TOKEN=<VALID_TOKEN>

go test -v ./...
```

### Step 6.3: Load Tests
```bash
cd tests/load

# Run load tests (start with lower VUs)
k6 run --duration=5m --vus=50 --env API_URL=http://$KONG_IP auth_test.js
k6 run --duration=5m --vus=50 --env API_URL=http://$KONG_IP features_test.js
k6 run --duration=5m --vus=50 --env API_URL=http://$KONG_IP commercial_test.js

# Check thresholds
python3 check_thresholds.py results-*.json
```

## Phase 7: Gradual Rollout

### Step 7.1: Canary Deployment (5%)
```bash
# Update Kong to route 5% to new services
kubectl apply -f k8s/kong/canary-5-percent.yaml

# Monitor for 30 minutes
kubectl logs -f -l app=kong -n metargb

# Check metrics
kubectl port-forward -n monitoring svc/grafana 3000:3000
# Open http://localhost:3000 and check dashboards
```

### Step 7.2: Increase to 25%
```bash
# If 5% looks good, increase
kubectl apply -f k8s/kong/canary-25-percent.yaml

# Monitor for 1 hour
```

### Step 7.3: Increase to 50%
```bash
kubectl apply -f k8s/kong/canary-50-percent.yaml

# Monitor for 2 hours
```

### Step 7.4: Full Cutover
```bash
# Route 100% traffic to microservices
kubectl apply -f k8s/kong/production.yaml

# Monitor closely for 4 hours
```

## Phase 8: Cleanup Laravel Monolith

```bash
# After 24 hours of stable operation
# Scale down Laravel instances
kubectl scale deployment laravel --replicas=0 -n legacy

# Keep for 7 days as backup, then delete
kubectl delete deployment laravel -n legacy
```

## Rollback Procedures

### Rollback Service Deployment
```bash
# Rollback specific service
kubectl rollout undo deployment/auth-service -n metargb

# Rollback to specific revision
kubectl rollout history deployment/auth-service -n metargb
kubectl rollout undo deployment/auth-service --to-revision=2 -n metargb
```

### Rollback Kong Configuration
```bash
# Restore previous Kong config
kubectl apply -f k8s/kong/kong.yml.backup

kubectl rollout restart deployment kong-kong -n metargb
```

### Emergency: Full Rollback to Laravel
```bash
# Route all traffic back to Laravel
kubectl apply -f k8s/kong/laravel-only.yaml

# Scale up Laravel
kubectl scale deployment laravel --replicas=3 -n legacy
```

## Post-Deployment Verification

### Health Checks
```bash
# Check all pods
kubectl get pods -n metargb

# Check pod logs
kubectl logs -l app=auth-service -n metargb --tail=100

# Check Istio mesh
istioctl proxy-status

# Check metrics
kubectl top pods -n metargb
kubectl top nodes
```

### Performance Validation
```bash
# Run full load test suite
cd tests/load
./run_all_tests.sh $KONG_IP

# Check results
python3 check_thresholds.py results-*.json
```

### Data Consistency Check
```bash
# Compare wallet balances (sample)
cd tests/database
go test -v -run TestDataConsistency
```

## Monitoring URLs

After deployment, access monitoring tools:

```bash
# Grafana
kubectl port-forward -n monitoring svc/grafana 3000:3000
# http://localhost:3000 (admin/changeme123!)

# Prometheus
kubectl port-forward -n monitoring svc/prometheus 9090:9090
# http://localhost:9090

# Jaeger
kubectl port-forward -n istio-system svc/jaeger-query 16686:16686
# http://localhost:16686

# Kiali (Istio UI)
istioctl dashboard kiali
```

## Troubleshooting

See [TROUBLESHOOTING.md](./TROUBLESHOOTING.md) for common issues and solutions.

