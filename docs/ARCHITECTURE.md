# MetaRGB Microservices Architecture

## Overview

MetaRGB microservices architecture transforms the Laravel monolith into 9 independent services communicating via gRPC, fronted by Kong API Gateway, secured with Istio service mesh, and monitored with Prometheus/Grafana.

## System Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Applications                       │
│                    (Web, Mobile, Desktop)                        │
└────────────────────────────┬────────────────────────────────────┘
                             │ HTTPS/REST
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Kong API Gateway                          │
│                  (REST → gRPC Translation)                       │
│            Authentication, Rate Limiting, CORS                   │
└─────────────┬───────────────────────────────────────────────────┘
              │ gRPC
              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Istio Service Mesh                        │
│           mTLS, Load Balancing, Circuit Breaking                 │
│           Retries, Timeouts, Fault Injection                     │
└─────┬───────┬───────┬───────┬───────┬───────┬───────┬──────┬────┘
      │       │       │       │       │       │       │      │
      ▼       ▼       ▼       ▼       ▼       ▼       ▼      ▼
  ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌─────┐ ┌────┐ ┌─────┐
  │Auth │ │Comm │ │Feat │ │Level│ │Dyna │ │Supp │ │Cal │ │Stor │
  │Svc  │ │Svc  │ │Svc  │ │Svc  │ │Svc  │ │Svc  │ │Svc │ │Svc  │
  └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ └──┬──┘ └─┬──┘ └──┬──┘
     │       │       │       │       │       │      │       │
     │       │       │       │       │       │      │       │
     └───────┴───────┴───────┴───────┴───────┴──────┴───────┘
                             │
                             ▼
                    ┌────────────────┐
                    │  MySQL Database │
                    │  (Shared)       │
                    └────────────────┘
```

## Service Responsibilities

### 1. Auth Service (Port 50051)
**Purpose**: Authentication, authorization, user management

**Endpoints**:
- `POST /api/auth/register` - OAuth registration initiation
- `GET /api/auth/callback` - OAuth callback handler
- `POST /api/auth/me` - Get authenticated user details
- `POST /api/auth/logout` - Revoke token
- `GET /api/auth/validate` - Validate token (internal)

**gRPC Methods**:
- `ValidateToken(TokenRequest) → User`
- `GetUser(UserID) → UserDetails`
- `UpdateProfile(UpdateRequest) → User`

**Database Tables**: `users`, `personal_access_tokens`, `otps`, `kycs`, `bank_accounts`, `profile_limitations`, `privacies`, `settings`

**Dependencies**: External OAuth server, Notifications service (for OTP)

### 2. Commercial Service (Port 50052)
**Purpose**: Wallet management, transactions, payments

**Endpoints**:
- `GET /api/user/wallet` - Get wallet balances
- `GET /api/user/transactions` - List transactions
- `GET /api/user/transactions/latest` - Latest transaction
- `POST /api/order` - Initiate payment
- `POST /api/parsian/callback` - Payment callback

**gRPC Methods**:
- `GetWallet(UserID) → Wallet`
- `DeductBalance(DeductRequest) → TransactionID`
- `AddBalance(AddRequest) → TransactionID`
- `CreateTransaction(TxRequest) → Transaction`

**Database Tables**: `wallets`, `transactions`, `payments`, `orders`, `variables`, `comissions`, `referrals`

**Dependencies**: Parsian SOAP API, Auth service

**Critical Features**:
- DECIMAL(20,10) precision for balances
- VARCHAR primary keys for transaction IDs
- Atomic balance operations with row locking

### 3. Features Service (Port 50053)
**Purpose**: Land/feature marketplace, ownership, buildings

**Endpoints**:
- `GET /api/features` - List features (with bbox filtering)
- `GET /api/my-features` - User's features
- `GET /api/features/{id}` - Feature details
- `POST /api/features/buy/{id}` - Direct purchase
- `POST /api/buy-requests/store/{id}` - Send buy request
- `POST /api/buy-requests/accept/{id}` - Accept request
- `GET /api/hourly-profits` - Profit history

**gRPC Methods**:
- `ListFeatures(BboxRequest) → FeatureList`
- `GetFeature(FeatureID) → FeatureDetails`
- `BuyFeature(BuyRequest) → Transaction`
- `UpdateFeatureOwner(UpdateRequest) → Success`

**Database Tables**: `features`, `feature_properties`, `geometries`, `coordinates`, `buy_feature_requests`, `sell_feature_requests`, `buildings`, `images`

**Dependencies**: Commercial service (wallet deduction), 3D environment API

**Critical Features**:
- Soft deletes on buy_feature_requests
- Spatial queries for bbox filtering
- Background worker for hourly profits

### 4. Levels Service (Port 50054)
**Purpose**: User progression, activities, challenges

**Endpoints**:
- `GET /api/users/{id}/levels` - User level info
- `GET /api/challenge/timings` - Challenge schedule
- `POST /api/challenge/question` - Get question
- `POST /api/challenge/answer` - Submit answer

**gRPC Methods**:
- `GetUserLevel(UserID) → LevelInfo`
- `LogActivity(ActivityRequest) → Score`
- `GetChallengeQuestion() → Question`
- `SubmitAnswer(AnswerRequest) → Result`

**Database Tables**: `levels`, `level_user`, `user_activities`, `user_logs`, `questions`, `answers`, `prizes`

**Dependencies**: Auth service

### 5. Dynasty Service (Port 50055)
**Purpose**: Dynasty/family management

**Endpoints**:
- `POST /api/dynasty/create/{feature}` - Create dynasty
- `POST /api/dynasty/add/member` - Send join request
- `POST /api/dynasty/requests/received/{id}` - Accept request
- `GET /api/dynasty/requests/sent` - List requests
- `POST /api/dynasty/children/{user}` - Set permissions

**Database Tables**: `dynasties`, `families`, `family_members`, `join_requests`, `children_permissions`

### 6. Support Service (Port 50056)
**Purpose**: Tickets, reports

**Endpoints**:
- `POST /api/tickets` - Create ticket
- `POST /api/tickets/response/{id}` - Add response
- `GET /api/tickets/{id}/close` - Close ticket
- `POST /api/reports` - Submit report

**Database Tables**: `tickets`, `ticket_responses`, `reports`, `user_event_reports`

### 7. Training Service (Port 50057)
**Purpose**: Video tutorials, comments

**Endpoints**:
- `GET /api/v2/tutorials` - List videos
- `GET /api/v2/tutorials/{slug}` - Video details
- `POST /api/v2/tutorials/{id}/comments` - Add comment
- `POST /api/v2/tutorials/{id}/interactions` - Like/view

**Database Tables**: `videos`, `video_categories`, `comments`, `interactions`, `likes`, `views`

### 8. Notifications Service (Port 50058)
**Purpose**: Multi-channel notifications (DB, SMS, Email)

**Endpoints**:
- `GET /api/notifications` - List notifications
- `POST /api/notifications/read/{id}` - Mark as read
- `POST /api/notifications/read/all` - Mark all read

**gRPC Methods**:
- `SendNotification(NotifRequest) → NotificationID`
- `SendSMS(SMSRequest) → Success`
- `SendEmail(EmailRequest) → Success`

**Database Tables**: `notifications`

**Dependencies**: Kavenegar SMS API, SMTP server

### 9. Calendar Service (Port 50059)
**Purpose**: Events management

**Endpoints**:
- `GET /api/calendar` - List events
- `GET /api/calendar/filter` - Date range filter
- `POST /api/calendar/events/{id}/interact` - Record interaction

**Database Tables**: `calendars`, `interactions`

### 10. File Storage Service (Port 50060)
**Purpose**: File uploads, FTP integration

**Endpoints**:
- `POST /api/upload` - Chunk upload

**gRPC Methods**:
- `UploadFile(stream FileChunk) → FileURL`
- `GetFile(FileID) → FileData`
- `DeleteFile(FileID) → Success`

**Database Tables**: `images` (polymorphic)

**Dependencies**: FTP server

### 11. WebSocket Gateway (Port 3000)
**Purpose**: Real-time events

**Channels**:
- `user-status-changed` - User online/offline
- `feature-status-changed` - Feature updates

**Dependencies**: Redis pub/sub, Auth service (token validation)

## Data Flow Examples

### Feature Purchase Flow
```
1. Client → Kong → Features Service: POST /api/features/buy/{id}
2. Features Service → Auth Service: ValidateToken(token)
3. Features Service → Commercial Service: DeductBalance(userID, amount)
4. Commercial Service: BEGIN TRANSACTION
5. Commercial Service: UPDATE wallets SET psc = psc - amount WHERE user_id = ?
6. Commercial Service: INSERT INTO transactions (...)
7. Commercial Service: COMMIT
8. Features Service: UPDATE features SET user_id = ? WHERE id = ?
9. Features Service → WebSocket: Publish feature-status-changed
10. Features Service → Client: Success response
```

### Authentication Flow
```
1. Client → Kong: POST /api/auth/register
2. Kong → Auth Service: Register(userInfo)
3. Auth Service → External OAuth: Redirect with state
4. Client redirected to OAuth server
5. OAuth callback → Kong → Auth Service: Callback(code, state)
6. Auth Service → OAuth Server: Exchange code for token
7. Auth Service: Create user in DB
8. Auth Service: Create personal_access_token
9. Auth Service → Client: Return token
```

## Cross-Service Communication

### Authentication Propagation
- Kong validates Sanctum tokens via Auth.ValidateToken
- User ID injected into gRPC context metadata
- All services extract user_id from context

### Transaction Consistency
- Wallet operations use database transactions with row locking
- Distributed transactions avoided (eventual consistency preferred)
- Idempotency keys for payment callbacks

### Error Handling
- gRPC status codes: OK, INVALID_ARGUMENT, NOT_FOUND, PERMISSION_DENIED, INTERNAL
- Retry logic: Exponential backoff for transient failures
- Circuit breakers: Open after 5 consecutive failures

## Performance Characteristics

### Target SLAs
- p95 latency: < 500ms
- p99 latency: < 1000ms
- Error rate: < 0.1%
- Availability: 99.9%

### Scaling Strategy
- Horizontal: HPA based on CPU/memory (target 70%)
- Vertical: 2 CPU, 4GB RAM per service (baseline)
- Database: Connection pooling (max 100 connections per service)

### Caching Strategy
- Redis for:
  - Session data (Sanctum tokens)
  - Rate limiting counters
  - Pub/sub for WebSocket events
- No application-level caching (stateless services)

## Security

### mTLS (via Istio)
- All service-to-service traffic encrypted
- Certificate rotation: 24 hours
- STRICT mode enforced

### API Authentication
- Sanctum bearer tokens
- Token validation on every request
- Token TTL: 7 days

### Authorization
- Policy-based (ported from Laravel policies)
- User roles: admin, verified_user, user
- Resource ownership checks

## Monitoring

### Metrics (Prometheus)
- RED metrics: Rate, Errors, Duration
- Database connection pool usage
- gRPC method latencies
- Business metrics: purchases, transactions, signups

### Tracing (Jaeger)
- Distributed traces across services
- Sampling: 10% in production, 100% in dev
- Retention: 7 days

### Logging
- Structured JSON logs
- Log levels: DEBUG, INFO, WARN, ERROR
- Request ID propagation
- Centralized aggregation (optional: ELK stack)

## Disaster Recovery

### Backups
- MySQL: Daily full backup, hourly incrementals
- Retention: 30 days
- Tested restore procedure

### Rollback
- Kubernetes rollout undo
- Docker images tagged by git SHA
- Database migrations: Reversible where possible

### Failover
- Multi-zone Kubernetes cluster
- Database replicas (read replicas for scaling)
- Redis Sentinel for high availability

## Development Workflow

1. Developer commits code
2. GitHub Actions triggers:
   - Unit tests
   - Integration tests
   - Build Docker image
3. On merge to main:
   - Deploy to staging
   - Run smoke tests
   - Manual approval for production
4. Production deployment:
   - Canary: 5% traffic for 30 minutes
   - If metrics OK: 100% traffic
   - If errors: Automatic rollback

