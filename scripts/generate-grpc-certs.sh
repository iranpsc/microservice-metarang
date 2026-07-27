#!/usr/bin/env bash
# Generate self-signed CA + server/client certificates for inter-service gRPC mTLS.
# Usage: ./scripts/generate-grpc-certs.sh [output_dir]
set -euo pipefail

OUT_DIR="${1:-./certs/grpc}"
mkdir -p "$OUT_DIR"

CA_KEY="$OUT_DIR/ca.key"
CA_CERT="$OUT_DIR/ca.crt"
SERVER_KEY="$OUT_DIR/server.key"
SERVER_CERT="$OUT_DIR/server.crt"
CLIENT_KEY="$OUT_DIR/client.key"
CLIENT_CERT="$OUT_DIR/client.crt"

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl is required" >&2
  exit 1
fi

openssl genrsa -out "$CA_KEY" 4096
openssl req -x509 -new -nodes -key "$CA_KEY" -sha256 -days 3650 -subj "/CN=metarang-grpc-ca" -out "$CA_CERT"

openssl genrsa -out "$SERVER_KEY" 4096
openssl req -new -key "$SERVER_KEY" -subj "/CN=metarang-grpc-server" -out "$OUT_DIR/server.csr"
openssl x509 -req -in "$OUT_DIR/server.csr" -CA "$CA_CERT" -CAkey "$CA_KEY" -CAcreateserial \
  -out "$SERVER_CERT" -days 825 -sha256 \
  -extfile <(printf "subjectAltName=DNS:localhost,DNS:*.metarang-network,DNS:auth-service,DNS:commercial-service,DNS:features-service,DNS:levels-service,DNS:dynasty-service,DNS:financial-service,DNS:notifications-service,DNS:calendar-service,DNS:support-service,DNS:training-service,DNS:social-service,DNS:storage-service,DNS:grpc-gateway")

openssl genrsa -out "$CLIENT_KEY" 4096
openssl req -new -key "$CLIENT_KEY" -subj "/CN=metarang-grpc-client" -out "$OUT_DIR/client.csr"
openssl x509 -req -in "$OUT_DIR/client.csr" -CA "$CA_CERT" -CAkey "$CA_KEY" -CAcreateserial \
  -out "$CLIENT_CERT" -days 825 -sha256

rm -f "$OUT_DIR/server.csr" "$OUT_DIR/client.csr" "$OUT_DIR/ca.srl"

echo "Generated gRPC TLS certificates in $OUT_DIR"
echo "Set GRPC_TLS_ENABLED=true in .env; docker compose will copy from $OUT_DIR or generate into the grpc_certs volume"
