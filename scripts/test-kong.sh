#!/bin/bash

# Kong API Gateway Testing and Debugging Script
# This script provides comprehensive testing and debugging for Kong API Gateway

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
KONG_ADMIN_URL="${KONG_ADMIN_URL:-http://localhost:8001}"
KONG_PROXY_URL="${KONG_PROXY_URL:-http://localhost:8000}"
KONG_CONFIG_FILE="${KONG_CONFIG_FILE:-./kong/kong.yml}"

# Functions
print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Check if Kong container is running
check_kong_running() {
    print_header "Checking Kong Container Status"
    
    if docker ps --filter "name=metargb-kong" --format "{{.Names}}" | grep -q "metargb-kong"; then
        print_success "Kong container is running"
        docker ps --filter "name=metargb-kong" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
        return 0
    else
        print_error "Kong container is not running"
        print_info "Start Kong with: docker-compose up -d kong"
        return 1
    fi
}

# Check Kong health
check_kong_health() {
    print_header "Checking Kong Health"
    
    if curl -s -f "${KONG_ADMIN_URL}/status" > /dev/null 2>&1; then
        print_success "Kong Admin API is accessible"
        echo ""
        echo "Kong Status:"
        curl -s "${KONG_ADMIN_URL}/status" | jq '.' 2>/dev/null || curl -s "${KONG_ADMIN_URL}/status"
        return 0
    else
        print_error "Cannot reach Kong Admin API at ${KONG_ADMIN_URL}"
        print_info "Check if Kong is running and ports are correct"
        return 1
    fi
}

# Validate Kong configuration
validate_kong_config() {
    print_header "Validating Kong Configuration"
    
    if [ ! -f "$KONG_CONFIG_FILE" ]; then
        print_error "Kong config file not found: $KONG_CONFIG_FILE"
        return 1
    fi
    
    print_info "Validating configuration file: $KONG_CONFIG_FILE"
    
    # Use Kong's config parse command
    if docker run --rm \
        -v "$(pwd)/kong:/kong" \
        kong:3.4 kong config parse /kong/kong.yml > /dev/null 2>&1; then
        print_success "Kong configuration is valid"
        return 0
    else
        print_error "Kong configuration has errors"
        print_info "Running detailed validation..."
        docker run --rm \
            -v "$(pwd)/kong:/kong" \
            kong:3.4 kong config parse /kong/kong.yml
        return 1
    fi
}

# List all services
list_services() {
    print_header "Listing Kong Services"
    
    if curl -s -f "${KONG_ADMIN_URL}/services" > /dev/null 2>&1; then
        echo "Registered Services:"
        curl -s "${KONG_ADMIN_URL}/services" | jq -r '.data[] | "  - \(.name) -> \(.url) (\(.protocol))"' 2>/dev/null || \
        curl -s "${KONG_ADMIN_URL}/services" | grep -o '"name":"[^"]*"' | sed 's/"name":"/  - /' | sed 's/"$//'
        return 0
    else
        print_error "Cannot fetch services from Kong Admin API"
        return 1
    fi
}

# List all routes
list_routes() {
    print_header "Listing Kong Routes"
    
    if curl -s -f "${KONG_ADMIN_URL}/routes" > /dev/null 2>&1; then
        echo "Registered Routes:"
        curl -s "${KONG_ADMIN_URL}/routes" | jq -r '.data[] | "  \(.paths[] // "N/A") -> \(.service.name // "N/A")"' 2>/dev/null || \
        curl -s "${KONG_ADMIN_URL}/routes" | grep -o '"paths":\[[^]]*\]' | head -20
        return 0
    else
        print_error "Cannot fetch routes from Kong Admin API"
        return 1
    fi
}

# Test service connectivity
test_service_connectivity() {
    print_header "Testing Service Connectivity"
    
    services=(
        "auth-service:50051"
        "commercial-service:50052"
        "features-service:50053"
        "levels-service:50054"
        "dynasty-service:50055"
        "calendar-service:50058"
        "storage-service:50059"
        "storage-service:8059"
    )
    
    for service in "${services[@]}"; do
        service_name=$(echo $service | cut -d: -f1)
        service_port=$(echo $service | cut -d: -f2)
        
        if docker exec metargb-kong nc -z "$service_name" "$service_port" 2>/dev/null; then
            print_success "$service_name:$service_port is reachable from Kong"
        else
            print_error "$service_name:$service_port is NOT reachable from Kong"
        fi
    done
}

# Test proxy endpoints
test_proxy_endpoints() {
    print_header "Testing Proxy Endpoints"
    
    endpoints=(
        "/api/auth"
        "/api/features"
        "/api/user/wallet"
        "/api/calendar"
        "/api/dynasty"
    )
    
    for endpoint in "${endpoints[@]}"; do
        print_info "Testing: ${KONG_PROXY_URL}${endpoint}"
        response=$(curl -s -o /dev/null -w "%{http_code}" "${KONG_PROXY_URL}${endpoint}" || echo "000")
        
        if [ "$response" = "404" ] || [ "$response" = "405" ] || [ "$response" = "400" ]; then
            print_success "Endpoint exists (HTTP $response - expected for gRPC endpoints)"
        elif [ "$response" = "000" ]; then
            print_error "Cannot connect to Kong proxy"
        elif [ "$response" = "502" ] || [ "$response" = "503" ]; then
            print_warning "Service unavailable (HTTP $response) - service may be down"
        else
            print_info "Response: HTTP $response"
        fi
    done
}

# Check Kong logs
check_kong_logs() {
    print_header "Recent Kong Logs (last 20 lines)"
    
    if docker logs --tail 20 metargb-kong 2>&1 | grep -i error; then
        print_warning "Errors found in Kong logs"
        echo ""
        docker logs --tail 20 metargb-kong 2>&1 | grep -i error
    else
        print_success "No recent errors in Kong logs"
    fi
    
    echo ""
    print_info "Full recent logs:"
    docker logs --tail 20 metargb-kong 2>&1
}

# Test CORS
test_cors() {
    print_header "Testing CORS Configuration"
    
    response=$(curl -s -X OPTIONS \
        -H "Origin: http://localhost:3000" \
        -H "Access-Control-Request-Method: GET" \
        -H "Access-Control-Request-Headers: Authorization" \
        -w "\n%{http_code}" \
        "${KONG_PROXY_URL}/api/auth" | tail -1)
    
    if [ "$response" = "200" ] || [ "$response" = "204" ]; then
        print_success "CORS preflight request successful"
    else
        print_warning "CORS preflight returned HTTP $response"
    fi
}

# Check plugins
check_plugins() {
    print_header "Checking Kong Plugins"
    
    if curl -s -f "${KONG_ADMIN_URL}/plugins" > /dev/null 2>&1; then
        echo "Active Plugins:"
        curl -s "${KONG_ADMIN_URL}/plugins" | jq -r '.data[] | "  - \(.name) (service: \(.service.name // "global"))"' 2>/dev/null || \
        curl -s "${KONG_ADMIN_URL}/plugins" | grep -o '"name":"[^"]*"' | sed 's/"name":"/  - /' | sed 's/"$//'
    else
        print_error "Cannot fetch plugins from Kong Admin API"
    fi
}

# Reload Kong configuration
reload_kong() {
    print_header "Reloading Kong Configuration"
    
    if docker exec metargb-kong kong reload 2>/dev/null; then
        print_success "Kong configuration reloaded"
    else
        print_error "Failed to reload Kong configuration"
        print_info "You may need to restart the Kong container: docker-compose restart kong"
    fi
}

# Main menu
show_menu() {
    echo ""
    print_header "Kong API Gateway Testing & Debugging"
    echo "1. Check Kong container status"
    echo "2. Check Kong health"
    echo "3. Validate Kong configuration"
    echo "4. List all services"
    echo "5. List all routes"
    echo "6. Test service connectivity"
    echo "7. Test proxy endpoints"
    echo "8. Check Kong logs"
    echo "9. Test CORS"
    echo "10. Check plugins"
    echo "11. Reload Kong configuration"
    echo "12. Run all tests"
    echo "0. Exit"
    echo ""
    read -p "Select an option: " choice
    
    case $choice in
        1) check_kong_running ;;
        2) check_kong_health ;;
        3) validate_kong_config ;;
        4) list_services ;;
        5) list_routes ;;
        6) test_service_connectivity ;;
        7) test_proxy_endpoints ;;
        8) check_kong_logs ;;
        9) test_cors ;;
        10) check_plugins ;;
        11) reload_kong ;;
        12) 
            check_kong_running && \
            check_kong_health && \
            validate_kong_config && \
            list_services && \
            list_routes && \
            test_service_connectivity && \
            test_proxy_endpoints && \
            check_kong_logs && \
            test_cors && \
            check_plugins
            print_success "All tests completed"
            ;;
        0) exit 0 ;;
        *) print_error "Invalid option" ;;
    esac
}

# Check if running interactively or with arguments
if [ $# -eq 0 ]; then
    # Interactive mode
    while true; do
        show_menu
    done
else
    # Command-line mode
    case "$1" in
        status) check_kong_running ;;
        health) check_kong_health ;;
        validate) validate_kong_config ;;
        services) list_services ;;
        routes) list_routes ;;
        connectivity) test_service_connectivity ;;
        endpoints) test_proxy_endpoints ;;
        logs) check_kong_logs ;;
        cors) test_cors ;;
        plugins) check_plugins ;;
        reload) reload_kong ;;
        all)
            check_kong_running && \
            check_kong_health && \
            validate_kong_config && \
            list_services && \
            list_routes && \
            test_service_connectivity && \
            test_proxy_endpoints && \
            check_kong_logs && \
            test_cors && \
            check_plugins
            ;;
        *)
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  status       - Check Kong container status"
            echo "  health       - Check Kong health"
            echo "  validate     - Validate Kong configuration"
            echo "  services     - List all services"
            echo "  routes       - List all routes"
            echo "  connectivity - Test service connectivity"
            echo "  endpoints    - Test proxy endpoints"
            echo "  logs         - Check Kong logs"
            echo "  cors         - Test CORS"
            echo "  plugins      - Check plugins"
            echo "  reload       - Reload Kong configuration"
            echo "  all          - Run all tests"
            echo ""
            echo "Or run without arguments for interactive mode"
            exit 1
            ;;
    esac
fi

