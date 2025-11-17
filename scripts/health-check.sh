#!/bin/bash
# System Health Check Script
# Checks all microservices and infrastructure

set -e

TIMEOUT=2
HEALTH_STATUS="healthy"
SERVICES_STATUS=()

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

check_service() {
    local name=$1
    local host=$2
    local port=$3
    
    if nc -z -w $TIMEOUT $host $port 2>/dev/null; then
        SERVICES_STATUS+=("{\"service\":\"$name\",\"status\":\"healthy\",\"host\":\"$host\",\"port\":$port}")
        echo -e "${GREEN}✅${NC} $name ($host:$port)"
        return 0
    else
        SERVICES_STATUS+=("{\"service\":\"$name\",\"status\":\"unhealthy\",\"host\":\"$host\",\"port\":$port,\"error\":\"Connection refused\"}")
        echo -e "${RED}❌${NC} $name ($host:$port) - Connection failed"
        HEALTH_STATUS="degraded"
        return 1
    fi
}

check_http() {
    local name=$1
    local url=$2
    
    if curl -s -f -m $TIMEOUT "$url" > /dev/null 2>&1; then
        SERVICES_STATUS+=("{\"service\":\"$name\",\"status\":\"healthy\",\"url\":\"$url\"}")
        echo -e "${GREEN}✅${NC} $name ($url)"
        return 0
    else
        SERVICES_STATUS+=("{\"service\":\"$name\",\"status\":\"unhealthy\",\"url\":\"$url\",\"error\":\"HTTP request failed\"}")
        echo -e "${RED}❌${NC} $name ($url) - HTTP request failed"
        HEALTH_STATUS="degraded"
        return 1
    fi
}

echo "🏥 MetaRGB Microservices Health Check"
echo "======================================"
echo ""

# Infrastructure Services
echo "📦 Infrastructure Services:"
check_service "MySQL" "localhost" "3308"
check_service "Redis" "localhost" "6379"
echo ""

# Core Microservices (gRPC)
echo "🔧 Core Microservices (gRPC):"
check_service "Auth Service" "localhost" "50051"
check_service "Commercial Service" "localhost" "50052"
check_service "Features Service" "localhost" "50053"
check_service "Levels Service" "localhost" "50054"
check_service "Dynasty Service" "localhost" "50055"
check_service "Calendar Service" "localhost" "50058"
check_service "Storage Service (gRPC)" "localhost" "50059"
echo ""

# Gateway Services
echo "🌐 Gateway Services:"
check_http "Kong API Gateway" "http://localhost:8001/status"
check_http "Kong Admin API" "http://localhost:8001/status"
check_http "WebSocket Gateway" "http://localhost:3000/health"
check_http "Storage Service (HTTP)" "http://localhost:8059/health"
echo ""

# Generate JSON output
JSON_OUTPUT="{\"status\":\"$HEALTH_STATUS\",\"timestamp\":\"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",\"services\":["
JSON_OUTPUT+=$(IFS=','; echo "${SERVICES_STATUS[*]}")
JSON_OUTPUT+="]}"

echo ""
echo "📊 Summary:"
if [ "$HEALTH_STATUS" = "healthy" ]; then
    echo -e "${GREEN}✅ All services are healthy${NC}"
    echo "$JSON_OUTPUT" | jq '.' 2>/dev/null || echo "$JSON_OUTPUT"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some services are unhealthy${NC}"
    echo "$JSON_OUTPUT" | jq '.' 2>/dev/null || echo "$JSON_OUTPUT"
    exit 1
fi

