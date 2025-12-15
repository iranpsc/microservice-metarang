# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
COPY shared/go.mod shared/go.sum ./shared/
COPY services/financial-service/go.mod services/financial-service/go.sum ./services/financial-service/

# Download dependencies
ENV GOPROXY=https://proxy.golang.org,direct
RUN go mod download

# Copy source code
COPY . .

# Build the application
WORKDIR /app/services/financial-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/services/financial-service/server .
COPY --from=builder /app/services/financial-service/config.env.sample ./config.env.sample

EXPOSE 50058

CMD ["./server"]
