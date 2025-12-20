.PHONY: proto clean-proto gen-auth gen-commercial gen-features gen-levels gen-dynasty gen-support gen-training gen-notifications gen-calendar gen-storage gen-financial gen-all help build-all build-features build-levels build-phase4 build-phase5 deploy-all test kong-validate phase6-setup phase6-istio phase6-monitoring phase6-verify phase6-cleanup

# Proto generation
PROTO_DIR=shared/proto
PROTO_OUT_DIR=shared/pb

# Docker
DOCKER_REGISTRY=metargb
VERSION?=latest

# Docker Compose compatibility - auto-detect docker-compose or docker compose plugin
# Windows PowerShell doesn't support 'command -v', so default to 'docker compose' (modern Docker Desktop)
ifeq ($(OS),Windows_NT)
DOCKER_COMPOSE := docker compose
else
DOCKER_COMPOSE := $(shell command -v docker-compose 2> /dev/null || echo "docker compose")
endif

help:
	@echo "Available targets:"
	@echo ""
	@echo "Proto Generation:"
	@echo "  proto            - Generate all proto files"
	@echo "  gen-auth         - Generate auth service proto"
	@echo "  gen-commercial   - Generate commercial service proto"
	@echo "  gen-features     - Generate features service proto"
	@echo "  gen-levels       - Generate levels service proto"
	@echo "  clean-proto      - Clean generated proto files"
	@echo ""
	@echo "Build:"
	@echo "  build-all        - Build all service Docker images"
	@echo "  build-features   - Build features service Docker image"
	@echo "  build-levels     - Build levels service Docker image"
	@echo ""
	@echo "Deploy:"
	@echo "  deploy-features  - Deploy features service to Kubernetes"
	@echo "  deploy-levels    - Deploy levels service to Kubernetes"
	@echo ""
	@echo "Kong API Gateway:"
	@echo "  kong-validate    - Validate Kong configuration"
	@echo "  kong-status      - Check Kong container and API status"
	@echo "  kong-health      - Check Kong health endpoint"
	@echo "  kong-services    - List all registered services"
	@echo "  kong-routes      - List all registered routes"
	@echo "  kong-logs        - Show Kong logs (last 50 lines)"
	@echo "  kong-logs-follow - Follow Kong logs in real-time"
	@echo "  kong-reload      - Reload Kong configuration"
	@echo "  kong-debug       - Comprehensive debug information"
	@echo "  kong-test        - Run all Kong tests"
	@echo ""
	@echo "Test:"
	@echo "  test             - Run all integration tests"
	@echo ""
	@echo "Database:"
	@echo "  import-schema    - Import database schema only (schema.sql)"
	@echo "  import-database  - Import database with data (metargb_db.sql)"
	@echo ""
	@echo "Phase 6 - Service Mesh & Observability:"
	@echo "  phase6-setup     - Complete Phase 6 setup (Istio + Monitoring)"
	@echo "  phase6-istio     - Install and configure Istio service mesh"
	@echo "  phase6-monitoring- Deploy Prometheus, Grafana, and Jaeger"
	@echo "  phase6-verify    - Verify Phase 6 components are running"
	@echo "  phase6-cleanup   - Clean up Phase 6 components"

proto: clean-proto gen-all

gen-all: gen-common gen-auth gen-commercial gen-features gen-levels gen-dynasty gen-support gen-training gen-notifications gen-calendar gen-storage gen-financial gen-social

gen-auth:
	@echo "Generating auth proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/auth
	protoc --go_out=$(PROTO_OUT_DIR)/auth --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/auth --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/auth.proto

gen-commercial:
	@echo "Generating commercial proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/commercial
	protoc --go_out=$(PROTO_OUT_DIR)/commercial --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/commercial --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/commercial.proto

gen-common:
	@echo "Generating common proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/common
	protoc --go_out=$(PROTO_OUT_DIR)/common --go_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/common.proto

gen-features:
	@echo "Generating features proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/features
	protoc --go_out=$(PROTO_OUT_DIR)/features --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/features --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/features.proto

gen-levels:
	@echo "Generating levels proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/levels
	protoc --go_out=$(PROTO_OUT_DIR)/levels --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/levels --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/levels.proto

gen-dynasty:
	@echo "Generating dynasty proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/dynasty
	protoc --go_out=$(PROTO_OUT_DIR)/dynasty --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/dynasty --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/dynasty.proto

gen-support:
	@echo "Generating support proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/support
	protoc --go_out=$(PROTO_OUT_DIR)/support --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/support --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/support.proto

gen-training:
	@echo "Generating training proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/training
	protoc --go_out=$(PROTO_OUT_DIR)/training --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/training --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/training.proto

gen-notifications:
	@echo "Generating notifications proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/notifications
	protoc --go_out=$(PROTO_OUT_DIR)/notifications --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/notifications --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/notifications.proto

gen-calendar:
	@echo "Generating calendar proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/calendar
	protoc --go_out=$(PROTO_OUT_DIR)/calendar --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/calendar --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/calendar.proto

gen-storage:
	@echo "Generating storage proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/storage
	protoc --go_out=$(PROTO_OUT_DIR)/storage --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/storage --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/storage.proto

gen-financial:
	@echo "Generating financial proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/financial
	protoc --go_out=$(PROTO_OUT_DIR)/financial --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/financial --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/financial.proto

gen-social:
	@echo "Generating social proto files..."
	@mkdir -p $(PROTO_OUT_DIR)/social
	protoc --go_out=$(PROTO_OUT_DIR)/social --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR)/social --go-grpc_opt=paths=source_relative \
		-I=$(PROTO_DIR) $(PROTO_DIR)/social.proto

clean-proto:
	@echo "Cleaning generated proto files..."
	@rm -rf $(PROTO_OUT_DIR)

# Build targets
build-all: build-features build-levels build-phase4 build-phase5

build-phase4: build-dynasty build-support build-training build-notifications

build-phase5: build-calendar build-storage build-websocket

build-dynasty:
	@echo "Building dynasty service Docker image..."
	docker build -f services/dynasty-service/Dockerfile -t $(DOCKER_REGISTRY)/dynasty-service:$(VERSION) .

build-support:
	@echo "Building support service Docker image..."
	docker build -f services/support-service/Dockerfile -t $(DOCKER_REGISTRY)/support-service:$(VERSION) .

build-training:
	@echo "Building training service Docker image..."
	docker build -f services/training-service/Dockerfile -t $(DOCKER_REGISTRY)/training-service:$(VERSION) .

build-notifications:
	@echo "Building notifications service Docker image..."
	docker build -f services/notifications-service/Dockerfile -t $(DOCKER_REGISTRY)/notifications-service:$(VERSION) .

build-calendar:
	@echo "Building calendar service Docker image..."
	docker build -f services/calendar-service/Dockerfile -t $(DOCKER_REGISTRY)/calendar-service:$(VERSION) .

build-storage:
	@echo "Building storage service Docker image..."
	docker build -f services/storage-service/Dockerfile -t $(DOCKER_REGISTRY)/storage-service:$(VERSION) .

build-websocket:
	@echo "Building websocket gateway Docker image..."
	docker build -f websocket-gateway/Dockerfile -t $(DOCKER_REGISTRY)/websocket-gateway:$(VERSION) .

build-features:
	@echo "Building features service Docker image..."
	docker build -f services/features-service/Dockerfile -t $(DOCKER_REGISTRY)/features-service:$(VERSION) .

build-levels:
	@echo "Building levels service Docker image..."
	docker build -f services/levels-service/Dockerfile -t $(DOCKER_REGISTRY)/levels-service:$(VERSION) .

# Deploy targets
deploy-features:
	@echo "Deploying features service to Kubernetes..."
	kubectl apply -f k8s/features-service/deployment.yaml

deploy-levels:
	@echo "Deploying levels service to Kubernetes..."
	kubectl apply -f k8s/levels-service/deployment.yaml

# Deploy targets
deploy-all: deploy-features deploy-levels deploy-phase4 deploy-phase5

deploy-phase4:
	@echo "Deploying Phase 4 services..."
	kubectl apply -f k8s/dynasty-service/
	kubectl apply -f k8s/support-service/
	kubectl apply -f k8s/training-service/
	kubectl apply -f k8s/notifications-service/

deploy-phase5:
	@echo "Deploying Phase 5 services..."
	kubectl apply -f k8s/calendar-service/
	kubectl apply -f k8s/storage-service/
	kubectl apply -f k8s/websocket-gateway/

# Kong Gateway
kong-validate:
	@echo "Validating Kong configuration..."
	@if [ -f kong/kong.yml ]; then \
		docker run --rm -v "$$(pwd)/kong:/kong:ro" kong:3.4 kong config parse /kong/kong.yml || \
		(echo "⚠️  Docker volume mount failed. Trying alternative validation..." && \
		 docker run --rm -i kong:3.4 kong config parse < kong/kong.yml); \
	else \
		echo "❌ kong/kong.yml not found"; \
		exit 1; \
	fi

kong-reload:
	@echo "Reloading Kong configuration..."
	@if docker ps --filter "name=metargb-kong" --format "{{.Names}}" | grep -q "metargb-kong"; then \
		docker exec metargb-kong kong reload; \
		echo "✅ Kong configuration reloaded"; \
	else \
		echo "❌ Kong container is not running. Start it with: make up"; \
		exit 1; \
	fi

kong-status:
	@echo "📊 Checking Kong status..."
	@if docker ps --filter "name=metargb-kong" --format "{{.Names}}" | grep -q "metargb-kong"; then \
		echo "✅ Kong container is running"; \
		docker ps --filter "name=metargb-kong" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"; \
		echo ""; \
		echo "Testing Kong Admin API..."; \
		curl -s http://localhost:8001/status | jq '.' 2>/dev/null || curl -s http://localhost:8001/status; \
	else \
		echo "❌ Kong container is not running"; \
		echo "Start Kong with: make up"; \
		exit 1; \
	fi

kong-health:
	@echo "🏥 Checking Kong health..."
	@curl -s http://localhost:8001/status | jq '.' 2>/dev/null || curl -s http://localhost:8001/status || (echo "❌ Cannot reach Kong Admin API" && exit 1)

kong-services:
	@echo "📋 Listing Kong services..."
	@curl -s http://localhost:8001/services | jq -r '.data[] | "  - \(.name) -> \(.url) (\(.protocol))"' 2>/dev/null || \
	curl -s http://localhost:8001/services | grep -o '"name":"[^"]*"' | sed 's/"name":"/  - /' | sed 's/"$//' || \
	(echo "❌ Cannot fetch services" && exit 1)

kong-routes:
	@echo "🛣️  Listing Kong routes..."
	@curl -s http://localhost:8001/routes | jq -r '.data[] | "  \(.paths[] // "N/A") -> \(.service.name // "N/A")"' 2>/dev/null || \
	curl -s http://localhost:8001/routes | grep -o '"paths":\[[^]]*\]' | head -20 || \
	(echo "❌ Cannot fetch routes" && exit 1)

kong-logs:
	@echo "📝 Showing Kong logs (last 50 lines)..."
	@docker logs --tail 50 metargb-kong 2>&1 || (echo "❌ Kong container not found" && exit 1)

kong-logs-follow:
	@echo "📝 Following Kong logs (Ctrl+C to stop)..."
	@docker logs -f metargb-kong 2>&1 || (echo "❌ Kong container not found" && exit 1)

kong-test:
	@echo "🧪 Running Kong tests..."
	@./scripts/test-kong.sh all

kong-debug:
	@echo "🐛 Kong Debug Information"
	@echo ""
	@echo "=== Container Status ==="
	@docker ps --filter "name=metargb-kong" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" || echo "Kong not running"
	@echo ""
	@echo "=== Health Check ==="
	@curl -s http://localhost:8001/status 2>/dev/null | jq '.database.reachable' 2>/dev/null || echo "Cannot reach Kong"
	@echo ""
	@echo "=== Recent Logs (errors only) ==="
	@docker logs --tail 30 metargb-kong 2>&1 | grep -i error || echo "No recent errors"
	@echo ""
	@echo "=== Service Connectivity ==="
	@docker exec metargb-kong nc -z auth-service 50051 2>/dev/null && echo "✅ auth-service:50051 reachable" || echo "❌ auth-service:50051 NOT reachable"
	@docker exec metargb-kong nc -z commercial-service 50052 2>/dev/null && echo "✅ commercial-service:50052 reachable" || echo "❌ commercial-service:50052 NOT reachable"
	@docker exec metargb-kong nc -z features-service 50053 2>/dev/null && echo "✅ features-service:50053 reachable" || echo "❌ features-service:50053 NOT reachable"

# =============================================================================
# PHASE 7: Testing Targets
# =============================================================================

# Unit tests
test-unit:
	@echo "🧪 Running unit tests for all services..."
	@for service in services/*/; do \
		if [ -f "$$service/go.mod" ]; then \
			echo "Testing $$(basename $$service)..."; \
			cd $$service && go test ./internal/... -v -race -coverprofile=coverage.out || exit 1; \
			cd ../..; \
		fi \
	done
	@echo "✅ All unit tests passed"

# Integration tests
test-integration:
	@echo "🧪 Running integration tests..."
	cd tests/integration && go test -v ./...

# Golden JSON tests
test-golden:
	@echo "🧪 Running golden JSON comparison tests..."
	cd tests/golden && go test -v ./...

# Database tests
test-database:
	@echo "🧪 Running database schema and concurrency tests..."
	cd tests/database && go test -v ./...

# Run all tests
test-all: test-unit test-integration test-golden test-database
	@echo "✅ All test suites passed"

# Legacy test target (kept for backward compatibility)
test: test-integration

# =============================================================================
# PHASE 7: Load Testing
# =============================================================================

load-test-auth:
	@echo "⚡ Running auth service load test..."
	k6 run --duration=5m --vus=100 tests/load/auth_test.js

load-test-features:
	@echo "⚡ Running features service load test..."
	k6 run --duration=5m --vus=100 tests/load/features_test.js

load-test-commercial:
	@echo "⚡ Running commercial service load test..."
	k6 run --duration=5m --vus=100 tests/load/commercial_test.js

load-test-all: load-test-auth load-test-features load-test-commercial
	@echo "✅ All load tests complete"
	python3 tests/load/check_thresholds.py tests/load/results-*.json

# =============================================================================
# PHASE 7: Golden Response Management
# =============================================================================

capture-golden:
	@echo "📸 Capturing golden responses from Laravel..."
	./scripts/capture_golden_responses.sh

validate-golden:
	@echo "✅ Validating golden JSON files..."
	@for file in tests/golden/testdata/*.json; do \
		if [ -f "$$file" ]; then \
			jq empty "$$file" 2>/dev/null || (echo "❌ Invalid JSON: $$file" && exit 1); \
		fi \
	done
	@echo "✅ All golden files are valid"

# =============================================================================
# PHASE 6: Service Mesh & Observability
# =============================================================================

phase6-setup: phase6-istio phase6-monitoring
	@echo "✅ Phase 6 setup complete!"
	@echo ""
	@echo "Access URLs (use kubectl port-forward):"
	@echo "  Grafana:  kubectl port-forward -n monitoring svc/grafana 3000:3000"
	@echo "            http://localhost:3000 (admin/changeme123!)"
	@echo "  Prometheus: kubectl port-forward -n monitoring svc/prometheus 9090:9090"
	@echo "              http://localhost:9090"
	@echo "  Jaeger:   kubectl port-forward -n istio-system svc/jaeger-query 16686:16686"
	@echo "            http://localhost:16686"
	@echo "  Kiali:    kubectl port-forward -n istio-system svc/kiali 20001:20001"
	@echo "            http://localhost:20001"

phase6-istio:
	@echo "🚀 Installing Istio service mesh..."
	@echo "Step 1: Creating namespaces..."
	kubectl apply -f k8s/istio/namespace-injection.yaml
	@echo ""
	@echo "Step 2: Installing Istio (this may take a few minutes)..."
	@echo "Note: Requires istioctl to be installed"
	@echo "Run: curl -L https://istio.io/downloadIstio | sh -"
	@echo "Then: export PATH=$$PATH:$$HOME/.istioctl/bin"
	@if command -v istioctl >/dev/null 2>&1; then \
		istioctl install -f k8s/istio/istio-install.yaml -y; \
	else \
		echo "⚠️  istioctl not found. Please install Istio manually."; \
		echo "Visit: https://istio.io/latest/docs/setup/getting-started/"; \
		exit 1; \
	fi
	@echo ""
	@echo "Step 3: Applying mTLS configuration..."
	kubectl apply -f k8s/istio/peer-authentication.yaml
	@echo ""
	@echo "Step 4: Configuring VirtualServices..."
	kubectl apply -f k8s/istio/virtual-services.yaml
	@echo ""
	@echo "Step 5: Configuring DestinationRules..."
	kubectl apply -f k8s/istio/destination-rules.yaml
	@echo ""
	@echo "Step 6: Restarting pods to inject Istio sidecars..."
	kubectl rollout restart deployment -n metargb
	@echo ""
	@echo "✅ Istio installation complete!"

phase6-monitoring:
	@echo "📊 Deploying monitoring stack..."
	@echo "Step 1: Creating monitoring namespace..."
	kubectl apply -f k8s/monitoring/prometheus/namespace.yaml
	@echo ""
	@echo "Step 2: Deploying Prometheus..."
	kubectl apply -f k8s/monitoring/prometheus/prometheus-deployment.yaml
	kubectl apply -f k8s/monitoring/prometheus/alerting-rules.yaml
	@echo "Waiting for Prometheus to be ready..."
	kubectl wait --for=condition=ready pod -l app=prometheus -n monitoring --timeout=300s
	@echo ""
	@echo "Step 3: Deploying ServiceMonitors..."
	kubectl apply -f k8s/monitoring/prometheus/service-monitors.yaml
	@echo ""
	@echo "Step 4: Deploying Grafana..."
	kubectl apply -f k8s/monitoring/grafana/grafana-deployment.yaml
	kubectl apply -f k8s/monitoring/grafana/dashboards-configmap.yaml
	@echo "Waiting for Grafana to be ready..."
	kubectl wait --for=condition=ready pod -l app=grafana -n monitoring --timeout=300s
	@echo ""
	@echo "Step 5: Deploying Jaeger..."
	kubectl apply -f k8s/monitoring/jaeger/jaeger-deployment.yaml
	kubectl apply -f k8s/monitoring/jaeger/jaeger-istio-config.yaml
	@echo "Waiting for Jaeger to be ready..."
	kubectl wait --for=condition=ready pod -l app=jaeger -n istio-system --timeout=300s
	@echo ""
	@echo "✅ Monitoring stack deployment complete!"

phase6-verify:
	@echo "🔍 Verifying Phase 6 components..."
	@echo ""
	@echo "=== Istio Status ==="
	@istioctl version 2>/dev/null || echo "istioctl not found"
	@kubectl get pods -n istio-system
	@echo ""
	@echo "=== Monitoring Stack Status ==="
	@kubectl get pods -n monitoring
	@echo ""
	@echo "=== Service Mesh Status ==="
	@kubectl get virtualservices -n metargb
	@kubectl get destinationrules -n metargb
	@kubectl get peerauthentication -n metargb
	@echo ""
	@echo "=== Sidecar Injection Status ==="
	@kubectl get pods -n metargb -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].name}{"\n"}{end}' | grep istio-proxy && echo "✅ Sidecars injected" || echo "⚠️  No sidecars found"
	@echo ""
	@echo "=== Service Endpoints ==="
	@echo "Prometheus: http://localhost:9090"
	@echo "Grafana: http://localhost:3000 (admin/changeme123!)"
	@echo "Jaeger: http://localhost:16686"
	@echo "Kiali: http://localhost:20001"
	@echo ""
	@echo "Run the following commands to access services:"
	@echo "  kubectl port-forward -n monitoring svc/prometheus 9090:9090"
	@echo "  kubectl port-forward -n monitoring svc/grafana 3000:3000"
	@echo "  kubectl port-forward -n istio-system svc/jaeger-query 16686:16686"

phase6-cleanup:
	@echo "🧹 Cleaning up Phase 6 components..."
	@echo "WARNING: This will remove all Istio and monitoring components!"
	@read -p "Are you sure? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		echo "Removing monitoring stack..."; \
		kubectl delete -f k8s/monitoring/jaeger/jaeger-deployment.yaml --ignore-not-found=true; \
		kubectl delete -f k8s/monitoring/grafana/grafana-deployment.yaml --ignore-not-found=true; \
		kubectl delete -f k8s/monitoring/prometheus/prometheus-deployment.yaml --ignore-not-found=true; \
		kubectl delete -f k8s/monitoring/prometheus/alerting-rules.yaml --ignore-not-found=true; \
		kubectl delete namespace monitoring --ignore-not-found=true; \
		echo "Removing Istio..."; \
		kubectl delete -f k8s/istio/destination-rules.yaml --ignore-not-found=true; \
		kubectl delete -f k8s/istio/virtual-services.yaml --ignore-not-found=true; \
		kubectl delete -f k8s/istio/peer-authentication.yaml --ignore-not-found=true; \
		istioctl uninstall --purge -y 2>/dev/null || echo "Istio uninstall skipped"; \
		kubectl delete namespace istio-system --ignore-not-found=true; \
		echo "✅ Cleanup complete!"; \
	else \
		echo "Cleanup cancelled."; \
	fi


# =============================================================================
# Docker Compose Management
# =============================================================================

.PHONY: up down restart logs ps build clean import-schema import-database help-docker dev-up dev-down dev-build dev-logs dev-restart dev-ps

up:
	@echo "🚀 Starting all microservices..."
	$(DOCKER_COMPOSE) up -d
	@echo "✅ All services started!"
	@echo ""
	@echo "Services available at:"
	@echo "  Kong API Gateway: http://localhost:8000"
	@echo "  Kong Admin:       http://localhost:8001"
	@echo "  WebSocket:        http://localhost:3000"
	@echo ""
	@echo "Run 'make ps' to check service status"
	@echo "Run 'make logs' to view logs"

down:
	@echo "🛑 Stopping all microservices..."
	$(DOCKER_COMPOSE) down
	@echo "✅ All services stopped"

restart:
	@echo "🔄 Restarting all microservices..."
	$(DOCKER_COMPOSE) restart
	@echo "✅ All services restarted"

logs:
	$(DOCKER_COMPOSE) logs -f

ps:
	@echo "📊 Service Status:"
	@echo ""
	$(DOCKER_COMPOSE) ps
	@echo ""
	@echo "Healthy services:"
	@docker ps --filter "health=healthy" --format "  ✅ {{.Names}}"
	@echo ""
	@echo "Unhealthy services:"
	@docker ps --filter "health=unhealthy" --format "  ❌ {{.Names}}"

build:
	@echo "🔨 Building all services..."
	$(DOCKER_COMPOSE) build
	@echo "✅ Build complete"

build-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "❌ Please specify SERVICE=service-name"; \
		echo "Example: make build-service SERVICE=auth-service"; \
		exit 1; \
	fi
	@echo "🔨 Building $(SERVICE)..."
	$(DOCKER_COMPOSE) build $(SERVICE)
	@echo "✅ $(SERVICE) built successfully"

clean:
	@echo "🧹 Cleaning up Docker resources..."
	$(DOCKER_COMPOSE) down -v
	docker system prune -f
	@echo "✅ Cleanup complete"

import-schema:
	@echo "📥 Importing database schema..."
	@if [ ! -f scripts/schema.sql ]; then \
		echo "❌ scripts/schema.sql not found!"; \
		exit 1; \
	fi
	docker exec -i metargb-mysql mysql -uroot -proot_password metargb_db < scripts/schema.sql
	@echo "✅ Schema imported successfully"
	@echo ""
	@echo "Verifying tables..."
	@docker exec metargb-mysql mysql -uroot -proot_password metargb_db -e "SELECT COUNT(*) as table_count FROM information_schema.tables WHERE table_schema='metargb_db';" 2>/dev/null | grep -v table_count || echo "Could not verify"

import-database:
	@echo "Importing database (schema + data) from metargb_db.sql..."
	@echo "Dropping and recreating database..."
ifeq ($(OS),Windows_NT)
	@docker exec -i metargb-mysql mysql -uroot -proot_password -e "DROP DATABASE IF EXISTS metargb_db; CREATE DATABASE metargb_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>nul
	@echo "Importing data..."
	@powershell -Command "Get-Content scripts\metargb_db.sql | docker exec -i metargb-mysql mysql -uroot -proot_password metargb_db"
else
	@docker exec -i metargb-mysql mysql -uroot -proot_password -e "DROP DATABASE IF EXISTS metargb_db; CREATE DATABASE metargb_db CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null || true
	@echo "Importing data..."
	@docker exec -i metargb-mysql mysql -uroot -proot_password metargb_db < scripts/metargb_db.sql
endif
	@echo "Database imported successfully"
	@echo ""
	@echo "Verifying import..."
ifeq ($(OS),Windows_NT)
	@docker exec metargb-mysql mysql -uroot -proot_password metargb_db -e "SELECT COUNT(*) as table_count FROM information_schema.tables WHERE table_schema='metargb_db';" 2>nul | findstr /v table_count || echo "Could not verify table count"
	@docker exec metargb-mysql mysql -uroot -proot_password metargb_db -e "SELECT COUNT(*) as row_count FROM account_securities;" 2>nul | findstr /v row_count || echo "Could not verify data"
else
	@docker exec metargb-mysql mysql -uroot -proot_password metargb_db -e "SELECT COUNT(*) as table_count FROM information_schema.tables WHERE table_schema='metargb_db';" 2>/dev/null | grep -v table_count || echo "Could not verify table count"
	@docker exec metargb-mysql mysql -uroot -proot_password metargb_db -e "SELECT COUNT(*) as row_count FROM account_securities;" 2>/dev/null | grep -v row_count || echo "Could not verify data"
endif

dev:
	@echo "🚀 Starting development environment..."
	@if [ ! -f .env ]; then \
		echo "⚠️  .env file not found. Creating from .env.example..."; \
		cp .env.example .env; \
		echo "📝 Please edit .env file with your configuration"; \
		exit 1; \
	fi
	@echo "Starting MySQL and Redis..."
	$(DOCKER_COMPOSE) up -d mysql redis
	@echo "Waiting for database to be ready..."
	@sleep 10
	@echo "Checking if schema needs to be imported..."
	@TABLE_COUNT=$$(docker exec metargb-mysql mysql -uroot -proot_password metargb_db -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='metargb_db';" 2>/dev/null | tail -1); \
	if [ "$$TABLE_COUNT" = "0" ]; then \
		echo "Importing schema..."; \
		make import-schema; \
	else \
		echo "✅ Database already initialized ($$TABLE_COUNT tables)"; \
	fi
	@echo ""
	@echo "Starting all services..."
	$(DOCKER_COMPOSE) up -d
	@echo ""
	@echo "✅ Development environment ready!"
	@make ps

stop-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "❌ Please specify SERVICE=service-name"; \
		exit 1; \
	fi
	$(DOCKER_COMPOSE) stop $(SERVICE)

start-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "❌ Please specify SERVICE=service-name"; \
		exit 1; \
	fi
	$(DOCKER_COMPOSE) start $(SERVICE)

logs-service:
	@if [ -z "$(SERVICE)" ]; then \
		echo "❌ Please specify SERVICE=service-name"; \
		exit 1; \
	fi
	$(DOCKER_COMPOSE) logs -f $(SERVICE)

# =============================================================================
# Development with Hot Reloading
# =============================================================================

dev-up:
	@echo "🚀 Starting development environment with hot reloading..."
	$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml up -d
	@echo "✅ Development services started with hot reloading!"
	@echo ""
	@echo "Services available at:"
	@echo "  Kong API Gateway: http://localhost:8000"
	@echo "  Kong Admin:       http://localhost:8001"
	@echo "  WebSocket:        http://localhost:3000"
	@echo ""
	@echo "Changes to Go files will automatically trigger rebuilds via Air"
	@echo "Changes to Node.js files will automatically restart via nodemon"
	@echo ""
	@echo "Run 'make dev-logs' to view logs"
	@echo "Run 'make dev-down' to stop services"

dev-down:
	@echo "🛑 Stopping development services..."
	$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml down
	@echo "✅ Development services stopped"

dev-build:
	@echo "🔨 Building development images with hot reloading support..."
	$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml build
	@echo "✅ Development images built successfully"

dev-logs:
	@echo "📝 Following development service logs (Ctrl+C to stop)..."
	$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml logs -f

dev-restart:
	@echo "🔄 Restarting development services..."
	$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml restart
	@echo "✅ Development services restarted"

dev-ps:
	@echo "📊 Development Service Status:"
	@echo ""
	$(DOCKER_COMPOSE) -f docker-compose.yml -f docker-compose.dev.yml ps
	@echo ""
	@echo "Healthy services:"
	@docker ps --filter "health=healthy" --format "  ✅ {{.Names}}"
	@echo ""
	@echo "Unhealthy services:"
	@docker ps --filter "health=unhealthy" --format "  ❌ {{.Names}}"

help-docker:
	@echo "Docker Compose Commands:"
	@echo ""
	@echo "  make dev              - Start complete development environment"
	@echo "  make up               - Start all services"
	@echo "  make down             - Stop all services"
	@echo "  make restart          - Restart all services"
	@echo "  make ps               - Show service status"
	@echo "  make logs             - Follow all service logs"
	@echo "  make build            - Build all services"
	@echo "  make clean            - Stop services and remove volumes"
	@echo "  make import-schema    - Import database schema only"
	@echo "  make import-database  - Import database with data (metargb_db.sql)"
	@echo ""
	@echo "Development (Hot Reloading):"
	@echo "  make dev-up           - Start services with hot reloading"
	@echo "  make dev-down         - Stop hot reloading services"
	@echo "  make dev-build        - Build dev images with hot reloading support"
	@echo "  make dev-logs         - View logs from dev services"
	@echo ""
	@echo "Service-specific commands:"
	@echo "  make build-service SERVICE=auth-service   - Build specific service"
	@echo "  make start-service SERVICE=auth-service   - Start specific service"
	@echo "  make stop-service SERVICE=auth-service    - Stop specific service"
	@echo "  make logs-service SERVICE=auth-service    - View service logs"
	@echo ""
	@echo "Examples:"
	@echo "  make dev                                  - Complete setup"
	@echo "  make dev-up                               - Start with hot reloading"
	@echo "  make logs-service SERVICE=auth-service    - View auth logs"
	@echo "  make restart                              - Restart everything"
