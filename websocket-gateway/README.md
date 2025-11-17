# MetaRGB WebSocket Gateway

Real-time event broadcasting gateway for MetaRGB microservices architecture using Socket.IO and Redis Pub/Sub.

## Features

- **Real-time Communication**: Bidirectional WebSocket connections using Socket.IO
- **Authentication**: Sanctum token validation via Auth service gRPC
- **Redis Pub/Sub**: Subscribes to events from microservices
- **Room Management**: User-specific rooms for targeted broadcasting
- **Scalability**: Supports multiple gateway instances with Redis adapter
- **Health Checks**: Built-in health and metrics endpoints

## Events

### Client Events
- `connected` - Sent when client successfully connects
- `user-status-changed` - User activity updates
- `feature-status-changed` - Feature ownership changes
- `notification-received` - Real-time notifications
- `pong` - Response to ping (heartbeat)

### Server Events (Client → Server)
- `ping` - Heartbeat check

## Installation

```bash
npm install
```

## Configuration

Create a `.env` file based on `.env.example`:

```env
PORT=3000
REDIS_URL=redis://localhost:6379
AUTH_SERVICE_ADDR=localhost:50051
CORS_ORIGIN=http://localhost:3000
NODE_ENV=development
```

## Running

### Development
```bash
npm run dev
```

### Production
```bash
npm start
```

### Docker
```bash
docker build -t metargb/websocket-gateway .
docker run -p 3000:3000 --env-file .env metargb/websocket-gateway
```

## Client Usage

### JavaScript/Browser
```javascript
import io from 'socket.io-client';

const socket = io('http://localhost:3000', {
  auth: {
    token: 'your-sanctum-token-here'
  },
  transports: ['websocket', 'polling']
});

socket.on('connect', () => {
  console.log('Connected to WebSocket gateway');
});

socket.on('connected', (data) => {
  console.log('Welcome:', data);
});

socket.on('notification-received', (notification) => {
  console.log('New notification:', notification);
  // Show notification to user
});

socket.on('user-status-changed', (data) => {
  console.log('User status changed:', data);
});

socket.on('feature-status-changed', (data) => {
  console.log('Feature status changed:', data);
});

socket.on('disconnect', () => {
  console.log('Disconnected from WebSocket gateway');
});

socket.on('error', (error) => {
  console.error('Socket error:', error);
});

// Heartbeat
setInterval(() => {
  socket.emit('ping');
}, 30000);

socket.on('pong', (data) => {
  console.log('Pong received:', data.timestamp);
});
```

## Service Integration

### Publishing Events from Microservices

Services publish events to Redis, and the WebSocket Gateway broadcasts them to connected clients.

#### Example: Auth Service (User Status Change)
```go
// Publish to Redis when user activity detected
func (s *AuthService) UpdateLastSeen(ctx context.Context, userID uint64) {
    // Update database...
    
    // Publish to Redis
    event := map[string]interface{}{
        "user_id": userID,
        "last_seen": time.Now().Format(time.RFC3339),
        "status": "online",
    }
    data, _ := json.Marshal(event)
    redisClient.Publish(ctx, "user-status", data)
}
```

#### Example: Features Service (Feature Ownership Change)
```go
// Publish when feature ownership changes
func (s *FeatureService) TransferOwnership(ctx context.Context, featureID, oldOwnerID, newOwnerID uint64) {
    // Transfer logic...
    
    // Publish event
    event := map[string]interface{}{
        "feature_id": featureID,
        "old_owner_id": oldOwnerID,
        "new_owner_id": newOwnerID,
        "timestamp": time.Now().Format(time.RFC3339),
    }
    data, _ := json.Marshal(event)
    redisClient.Publish(ctx, "feature-status", data)
}
```

#### Example: Notifications Service
```go
// Publish notification
func (s *NotificationService) SendNotification(ctx context.Context, userID uint64, notification *Notification) {
    // Save to database...
    
    // Publish for real-time delivery
    event := map[string]interface{}{
        "id": notification.ID,
        "user_id": userID,
        "type": notification.Type,
        "title": notification.Title,
        "message": notification.Message,
        "data": notification.Data,
        "created_at": notification.CreatedAt,
    }
    data, _ := json.Marshal(event)
    redisClient.Publish(ctx, "notifications", data)
}
```

## Endpoints

### Health Check
```
GET /health
```

Response:
```json
{
  "status": "healthy",
  "connections": 42,
  "users": 35,
  "timestamp": "2024-01-15T10:30:00.000Z"
}
```

### Metrics
```
GET /metrics
```

Response:
```json
{
  "totalConnections": 42,
  "totalUsers": 35,
  "usersList": [1, 5, 10, ...],
  "uptime": 3600,
  "memory": {
    "rss": 50000000,
    "heapTotal": 30000000,
    "heapUsed": 20000000
  },
  "timestamp": "2024-01-15T10:30:00.000Z"
}
```

## Architecture

```
Clients (Web/Mobile)
       ↓ WebSocket
WebSocket Gateway (Socket.IO)
       ↓ Redis Pub/Sub
Microservices (Auth, Features, Notifications, etc.)
```

## Scaling

For horizontal scaling with multiple gateway instances, use Redis adapter:

```javascript
const { createAdapter } = require('@socket.io/redis-adapter');

const pubClient = new Redis(process.env.REDIS_URL);
const subClient = pubClient.duplicate();

io.adapter(createAdapter(pubClient, subClient));
```

This allows events to be shared across all gateway instances.

## Monitoring

- Use `/health` endpoint for health checks
- Use `/metrics` endpoint for Prometheus scraping
- Monitor Redis connection status
- Track active connections and users
- Log authentication failures

## Security

- All connections require valid Sanctum tokens
- Tokens are validated via Auth service gRPC
- CORS configuration restricts origins
- Use HTTPS in production (terminate at load balancer)
- Rate limiting should be applied at Kong Gateway level

## License

MIT

