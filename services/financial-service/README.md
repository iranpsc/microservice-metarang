# Financial Service

A Go microservice that handles order creation, Parsian payment gateway integration, and store package management.

## Features

### 1. Order Service
- Create orders for purchasing virtual assets (psc, irr, red, blue, yellow)
- Integration with Parsian payment gateway
- Handle payment callbacks and verification
- First order bonus system (50% bonus for first-time buyers)
- Referral commission processing (via commercial-service integration)

### 2. Store Service
- Retrieve store package details by codes
- Package pricing with current asset rates
- Image URL retrieval for packages

## API Endpoints

### POST /api/order
Creates an order and returns Parsian payment URL.

**Request:**
```json
{
  "amount": 10,
  "asset": "psc"
}
```

**Response:**
```json
{
  "link": "https://pec.shaparak.ir/NewIPG/?token=..."
}
```

### POST /api/parsian/callback
Handles Parsian payment gateway callback (public endpoint).

**Request:** Form-encoded data from Parsian gateway

**Response:** HTTP 302 redirect to frontend verification URL

### POST /api/store
Retrieves store package details.

**Request:**
```json
{
  "codes": ["PACK1", "PACK2"]
}
```

**Response:**
```json
[
  {
    "id": 1,
    "code": "PACK1",
    "asset": "psc",
    "amount": 100,
    "unitPrice": 1000.0,
    "image": "http://example.com/image.jpg"
  }
]
```

## Architecture

Follows the standard microservice architecture:
- **Handler Layer**: gRPC handlers for request/response conversion
- **Service Layer**: Business logic (order creation, callback handling, store packages)
- **Repository Layer**: Database operations

## Configuration

Copy `config.env.sample` to `config.env` and configure:

```env
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_DATABASE=metargb_db

GRPC_PORT=50058

PARSIAN_MERCHANT_ID=your_merchant_id
PARSIAN_LOAN_ACCOUNT_MERCHANT_ID=your_loan_merchant_id
PARSIAN_CALLBACK_URL=https://rgb.irpsc.com/api/parsian/callback
FRONTEND_URL=https://rgb.irpsc.com
```

## Setup

1. Generate proto files:
```bash
make gen-financial
```

2. Install dependencies:
```bash
go mod download
go mod tidy
```

3. Run tests:
```bash
go test ./...
```

4. Run service:
```bash
go run cmd/server/main.go
```

## Testing

Comprehensive tests are provided for:
- Service layer (order and store services)
- Handler layer (gRPC handlers)
- Repository layer (database operations)

Run tests:
```bash
go test ./... -v
```

## Dependencies

- Database: MySQL (shared with Laravel)
- gRPC: For service communication
- Parsian Payment Gateway: For payment processing

## Integration

The service integrates with:
- **Commercial Service**: For wallet operations and referral processing (via gRPC)
- **Auth Service**: For user authentication and token validation
- **Notifications Service**: For transaction notifications (TODO)

## Notes

- Orders start with status `-138` (pending Parsian verification)
- First order bonus: 50% for first-time buyers (non-irr assets only)
- Referral commissions processed for non-irr assets
- Store packages require at least 2 codes in request
