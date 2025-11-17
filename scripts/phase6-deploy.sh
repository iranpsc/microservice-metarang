#!/bin/bash

# MetaRGB Phase 6 Deployment Script
# Service Mesh & Observability Setup
# This script automates the installation and configuration of Istio, Prometheus, Grafana, and Jaeger

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

log_success() {
    echo -e "${GREEN}✓${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

log_error() {
    echo -e "${RED}✗${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found. Please install kubectl first."
        exit 1
    fi
    log_success "kubectl found"
    
    # Check istioctl
    if ! command -v istioctl &> /dev/null; then
        log_warning "istioctl not found. Installing Istio CLI..."
        curl -L https://istio.io/downloadIstio | sh -
        export PATH=$PWD/istio-*/bin:$PATH
        log_success "istioctl installed"
    else
        log_success "istioctl found"
    fi
    
    # Check cluster connection
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    log_success "Connected to Kubernetes cluster"
}

install_istio() {
    log_info "Installing Istio service mesh..."
    
    # Create namespaces
    log_info "Creating namespaces..."
    kubectl apply -f k8s/istio/namespace-injection.yaml
    
    # Install Istio
    log_info "Installing Istio control plane (this may take 3-5 minutes)..."
    istioctl install -f k8s/istio/istio-install.yaml -y
    
    # Wait for Istio to be ready
    log_info "Waiting for Istio pods to be ready..."
    kubectl wait --for=condition=ready pod -l app=istiod -n istio-system --timeout=300s
    
    # Apply mTLS configuration
    log_info "Configuring mTLS..."
    kubectl apply -f k8s/istio/peer-authentication.yaml
    
    # Apply traffic management rules
    log_info "Configuring VirtualServices..."
    kubectl apply -f k8s/istio/virtual-services.yaml
    
    log_info "Configuring DestinationRules..."
    kubectl apply -f k8s/istio/destination-rules.yaml
    
    # Restart pods to inject sidecars
    log_info "Restarting application pods to inject Istio sidecars..."
    kubectl rollout restart deployment -n metargb 2>/dev/null || log_warning "No deployments found in metargb namespace"
    
    log_success "Istio installation complete!"
}

install_monitoring() {
    log_info "Installing monitoring stack..."
    
    # Create monitoring namespace
    log_info "Creating monitoring namespace..."
    kubectl apply -f k8s/monitoring/prometheus/namespace.yaml
    
    # Install Prometheus
    log_info "Deploying Prometheus..."
    kubectl apply -f k8s/monitoring/prometheus/prometheus-deployment.yaml
    kubectl apply -f k8s/monitoring/prometheus/alerting-rules.yaml
    
    log_info "Waiting for Prometheus to be ready..."
    kubectl wait --for=condition=ready pod -l app=prometheus -n monitoring --timeout=300s
    
    # Install ServiceMonitors
    log_info "Deploying ServiceMonitors..."
    kubectl apply -f k8s/monitoring/prometheus/service-monitors.yaml
    
    # Install Grafana
    log_info "Deploying Grafana..."
    kubectl apply -f k8s/monitoring/grafana/grafana-deployment.yaml
    kubectl apply -f k8s/monitoring/grafana/dashboards-configmap.yaml
    
    log_info "Waiting for Grafana to be ready..."
    kubectl wait --for=condition=ready pod -l app=grafana -n monitoring --timeout=300s
    
    # Install Jaeger
    log_info "Deploying Jaeger..."
    kubectl apply -f k8s/monitoring/jaeger/jaeger-deployment.yaml
    kubectl apply -f k8s/monitoring/jaeger/jaeger-istio-config.yaml
    
    log_info "Waiting for Jaeger to be ready..."
    kubectl wait --for=condition=ready pod -l app=jaeger -n istio-system --timeout=300s
    
    log_success "Monitoring stack deployment complete!"
}

verify_installation() {
    log_info "Verifying installation..."
    echo ""
    
    echo "=== Istio Components ==="
    kubectl get pods -n istio-system
    echo ""
    
    echo "=== Monitoring Components ==="
    kubectl get pods -n monitoring
    echo ""
    
    echo "=== Service Mesh Configuration ==="
    kubectl get virtualservices -n metargb 2>/dev/null || log_warning "No VirtualServices found"
    kubectl get destinationrules -n metargb 2>/dev/null || log_warning "No DestinationRules found"
    echo ""
    
    # Check sidecar injection
    log_info "Checking sidecar injection..."
    if kubectl get pods -n metargb -o jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.spec.containers[*].name}{"\n"}{end}' | grep -q istio-proxy; then
        log_success "Istio sidecars are injected"
    else
        log_warning "No Istio sidecars found. Deploy some services first."
    fi
}

print_access_info() {
    echo ""
    echo "========================================="
    log_success "Phase 6 deployment complete!"
    echo "========================================="
    echo ""
    echo "📊 Access Monitoring Tools:"
    echo ""
    echo "1. Grafana (Dashboards):"
    echo "   kubectl port-forward -n monitoring svc/grafana 3000:3000"
    echo "   URL: http://localhost:3000"
    echo "   Credentials: admin / changeme123!"
    echo ""
    echo "2. Prometheus (Metrics):"
    echo "   kubectl port-forward -n monitoring svc/prometheus 9090:9090"
    echo "   URL: http://localhost:9090"
    echo ""
    echo "3. Jaeger (Distributed Tracing):"
    echo "   kubectl port-forward -n istio-system svc/jaeger-query 16686:16686"
    echo "   URL: http://localhost:16686"
    echo ""
    echo "4. Kiali (Service Mesh Visualization):"
    echo "   kubectl port-forward -n istio-system svc/kiali 20001:20001"
    echo "   URL: http://localhost:20001"
    echo ""
    echo "📝 Next Steps:"
    echo "   1. Change Grafana password (admin/changeme123!)"
    echo "   2. Review Grafana dashboards for service metrics"
    echo "   3. Check Prometheus alerting rules"
    echo "   4. Verify distributed traces in Jaeger"
    echo "   5. Explore service mesh in Kiali"
    echo ""
}

# Main execution
main() {
    echo "========================================="
    echo "  MetaRGB Phase 6 Deployment"
    echo "  Service Mesh & Observability"
    echo "========================================="
    echo ""
    
    check_prerequisites
    
    echo ""
    log_info "Starting Phase 6 deployment..."
    echo ""
    
    install_istio
    echo ""
    
    install_monitoring
    echo ""
    
    verify_installation
    echo ""
    
    print_access_info
}

# Handle script interruption
trap 'log_error "Deployment interrupted. You may need to run cleanup and retry."; exit 1' INT TERM

# Run main function
main "$@"

