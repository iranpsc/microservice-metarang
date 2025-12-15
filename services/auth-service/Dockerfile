# Multi-stage build for Auth Service
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make protoc protobuf-dev

WORKDIR /workspace

# Create the proper directory structure
RUN mkdir -p /workspace/metargb/auth-service /workspace/metargb/shared

# Copy only go.mod files first for better layer caching
COPY ./shared/go.mod /workspace/metargb/shared/
COPY ./services/auth-service/go.mod ./services/auth-service/go.sum* /workspace/metargb/auth-service/

# Setup go workspace (before copying source to optimize caching)
WORKDIR /workspace
RUN echo 'go 1.24.0' > go.work && \
    echo '' >> go.work && \
    echo 'use (' >> go.work && \
    echo '    ./metargb/auth-service' >> go.work && \
    echo '    ./metargb/shared' >> go.work && \
    echo ')' >> go.work

# Download dependencies first (this layer will be cached if go.mod doesn't change)
WORKDIR /workspace/metargb/auth-service

# Update replace directive for Docker context
RUN sed -i 's|replace metargb/shared => ../../shared|replace metargb/shared => /workspace/metargb/shared|g' go.mod

# Use GOPROXY with fallback instead of direct-only (faster and more reliable)
ENV GOPROXY=https://proxy.golang.org,direct
RUN go mod download

# Now copy the rest of the source code (this invalidates cache only when source changes)
WORKDIR /workspace
COPY ./shared/ /workspace/metargb/shared/
COPY ./services/auth-service/ /workspace/metargb/auth-service/

# Fix replace directive again after copying source (since go.mod gets overwritten)
WORKDIR /workspace/metargb/auth-service
RUN sed -i 's|replace metargb/shared => ../../shared|replace metargb/shared => /workspace/metargb/shared|g' go.mod

# Build the application  
WORKDIR /workspace
ENV GOWORK=/workspace/go.work
RUN cd /workspace/metargb/auth-service && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o /app/auth-service ./cmd/server

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/auth-service .

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# Expose gRPC port
EXPOSE 50051

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD nc -z localhost 50051 || exit 1

# Run the application
ENTRYPOINT ["/app/auth-service"]

