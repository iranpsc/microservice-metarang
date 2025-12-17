# Health Check Service Dockerfile
FROM golang:1.24-alpine AS builder

# Install build dependencies (git needed for GOPROXY=direct)
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY services/health-check-service/go.mod services/health-check-service/go.sum* ./

# Download dependencies
ENV GOPROXY=https://goproxy.io,direct
RUN go mod download

# Copy source
COPY services/health-check-service/main.go .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o health-check-service main.go

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/health-check-service .

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8090/health || exit 1

CMD ["./health-check-service"]

