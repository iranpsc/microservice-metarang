package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"
)

type ServiceStatus struct {
	Service string `json:"service"`
	Status  string `json:"status"`
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`
	URL     string `json:"url,omitempty"`
	Error   string `json:"error,omitempty"`
	Latency string `json:"latency,omitempty"`
}

type HealthResponse struct {
	Status    string          `json:"status"`
	Timestamp string          `json:"timestamp"`
	Uptime    string          `json:"uptime"`
	Services  []ServiceStatus `json:"services"`
	Summary   struct {
		Total    int `json:"total"`
		Healthy  int `json:"healthy"`
		Unhealthy int `json:"unhealthy"`
	} `json:"summary"`
}

var startTime = time.Now()

func main() {
	http.HandleFunc("/health", healthCheckHandler)
	http.HandleFunc("/api/health", healthCheckHandler)
	
	port := "8090"
	log.Printf("🏥 Health Check Service starting on port %s", port)
	log.Printf("📊 Health check endpoint: http://localhost:%s/health", port)
	
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start health check service: %v", err)
	}
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	
	services := []ServiceStatus{}
	
	// Infrastructure Services
	services = append(services, checkTCP(ctx, "MySQL", "mysql", 3306))
	services = append(services, checkTCP(ctx, "Redis", "redis", 6379))
	
	// Core Microservices (gRPC)
	services = append(services, checkTCP(ctx, "Auth Service", "auth-service", 50051))
	services = append(services, checkTCP(ctx, "Commercial Service", "commercial-service", 50052))
	services = append(services, checkTCP(ctx, "Features Service", "features-service", 50053))
	services = append(services, checkTCP(ctx, "Levels Service", "levels-service", 50054))
	services = append(services, checkTCP(ctx, "Dynasty Service", "dynasty-service", 50055))
	services = append(services, checkTCP(ctx, "Calendar Service", "calendar-service", 50058))
	services = append(services, checkTCP(ctx, "Storage Service (gRPC)", "storage-service", 50059))
	
	// Gateway Services (HTTP)
	services = append(services, checkHTTP(ctx, "Kong API Gateway", "http://kong:8001/status"))
	services = append(services, checkHTTP(ctx, "Kong Admin API", "http://kong:8001/status"))
	services = append(services, checkHTTP(ctx, "WebSocket Gateway", "http://websocket-gateway:3000/health"))
	services = append(services, checkHTTP(ctx, "Storage Service (HTTP)", "http://storage-service:8059/health"))
	
	// Calculate summary
	healthy := 0
	unhealthy := 0
	for _, s := range services {
		if s.Status == "healthy" {
			healthy++
		} else {
			unhealthy++
		}
	}
	
	// Determine overall status
	overallStatus := "healthy"
	if unhealthy > 0 {
		overallStatus = "degraded"
	}
	if unhealthy > len(services)/2 {
		overallStatus = "unhealthy"
	}
	
	uptime := time.Since(startTime)
	response := HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    fmt.Sprintf("%.0fs", uptime.Seconds()),
		Services:  services,
	}
	response.Summary.Total = len(services)
	response.Summary.Healthy = healthy
	response.Summary.Unhealthy = unhealthy
	
	statusCode := http.StatusOK
	if overallStatus == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if overallStatus == "degraded" {
		statusCode = http.StatusOK // Still return 200 but with degraded status
	}
	
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

func checkTCP(ctx context.Context, name, host string, port int) ServiceStatus {
	start := time.Now()
	
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", host, port))
	latency := time.Since(start)
	
	if err != nil {
		return ServiceStatus{
			Service: name,
			Status:  "unhealthy",
			Host:    host,
			Port:    port,
			Error:   err.Error(),
			Latency: latency.String(),
		}
	}
	conn.Close()
	
	return ServiceStatus{
		Service: name,
		Status:  "healthy",
		Host:    host,
		Port:    port,
		Latency: latency.String(),
	}
}

func checkHTTP(ctx context.Context, name, url string) ServiceStatus {
	start := time.Now()
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return ServiceStatus{
			Service: name,
			Status:  "unhealthy",
			URL:     url,
			Error:   err.Error(),
		}
	}
	
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	latency := time.Since(start)
	
	if err != nil {
		return ServiceStatus{
			Service: name,
			Status:  "unhealthy",
			URL:     url,
			Error:   err.Error(),
			Latency: latency.String(),
		}
	}
	defer resp.Body.Close()
	
	status := "healthy"
	if resp.StatusCode >= 400 {
		status = "unhealthy"
	}
	
	return ServiceStatus{
		Service: name,
		Status:  status,
		URL:     url,
		Latency: latency.String(),
	}
}

