FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
COPY ../../shared /app/shared

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/server .

EXPOSE 50058

CMD ["./server"]


