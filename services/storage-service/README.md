# Storage Service

A microservice for handling file uploads with chunk support, matching Laravel `FileUploadController` functionality.

## 🎯 Features

- ✅ Chunk-based file uploads (resumable)
- ✅ Progress tracking
- ✅ Automatic file assembly
- ✅ MIME type organization
- ✅ Unique filename generation (MD5 hash)
- ✅ Session management with auto-cleanup
- ✅ Dual server architecture (gRPC + HTTP REST)
- ✅ Laravel-compatible response format

## 🏗️ Architecture

**⚠️ IMPORTANT: All client requests MUST go through Kong API Gateway!**

```
Client → Kong Gateway (8000) → Storage Service (8059) → FTP/Storage
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for detailed architecture documentation.

## 🚀 Quick Start

### Production Mode (Recommended)

Run with Docker Compose (includes Kong API Gateway):

```bash
# Start all services
cd metargb-microservices
docker-compose up -d storage-service kong

# Test through Kong (correct way)
curl -X POST http://localhost:8000/api/upload \
  -F "file=@test.jpg"
```

**Client Endpoint:** `http://localhost:8000/api/upload`

### Development Mode (Testing Only)

Run standalone for debugging (bypasses Kong):

```bash
# Copy config
cp config.env.sample config.env

# Edit config as needed
nano config.env

# Run test server
go run test_server.go
```

**Test Endpoint:** `http://localhost:8059/upload` (⚠️ Development only!)

## 📡 API Endpoints

### Public Endpoint (via Kong)

```
POST http://localhost:8000/api/upload
```

**No authentication required** - Public endpoint for file uploads

**Request:**
- Content-Type: `multipart/form-data`
- Body: `file` (file data)
- Optional fields:
  - `upload_id` - Session identifier
  - `chunk_index` - Current chunk index (0-based)
  - `total_chunks` - Total number of chunks
  - `filename` - Original filename
  - `content_type` - MIME type
  - `total_size` - Total file size

**Response:**
```json
{
  "success": true,
  "done": 100.0,
  "message": "File uploaded successfully",
  "is_finished": true,
  "path": "upload/image-jpeg/2025-10-30/photo_abc123.jpg",
  "name": "photo_abc123.jpg",
  "mime_type": "image/jpeg"
}
```

### Health Check

```
GET http://localhost:8059/health
```

## 🧪 Testing

### 1. With Interactive HTML Page

```bash
# Ensure services are running
docker-compose up -d storage-service kong

# Serve test page
cd services/storage-service
python3 -m http.server 9000 &

# Open in browser
open http://localhost:9000/test_upload.html
```

The test page will use `http://localhost:8000/api/upload` (through Kong).

### 2. With cURL

**Through Kong (Production):**
```bash
curl -X POST http://localhost:8000/api/upload \
  -F "file=@photo.jpg"
```

**Direct (Testing only):**
```bash
curl -X POST http://localhost:8059/upload \
  -F "file=@photo.jpg"
```

### 3. Chunked Upload Example

```bash
# Create test file
dd if=/dev/urandom of=large.bin bs=1M count=10

# Upload in chunks (JavaScript example in test_upload.html)
```

## ⚙️ Configuration

### Environment Variables

```bash
# HTTP Server
HTTP_PORT=8059

# gRPC Server (for internal microservice communication)
GRPC_PORT=50059

# Database
DB_HOST=localhost
DB_PORT=3306
DB_DATABASE=metargb_db
DB_USER=root
DB_PASSWORD=

# FTP (Production)
FTP_HOST=ftp.metargb.com
FTP_PORT=21
FTP_USER=metargb_uploads
FTP_PASSWORD=ftp_password
FTP_BASE_URL=https://cdn.metargb.com/uploads

# Chunk Upload
TEMP_DIR=/tmp/storage-chunks
```

### For Testing (Mock FTP)

The `test_server.go` uses a mock FTP client that saves files locally:

```bash
UPLOAD_DIR=/tmp/storage-uploads
```

## 📂 File Organization

Files are automatically organized:

```
upload/
├── image-jpeg/
│   └── 2025-10-30/
│       ├── photo_a3f2d8e9b1c4f7a6.jpg
│       └── avatar_d4e5f6g7h8i9j0k1.jpg
├── video-mp4/
│   └── 2025-10-30/
│       └── video_b2c3d4e5f6g7h8i9.mp4
└── application-pdf/
    └── 2025-10-30/
        └── document_c3d4e5f6g7h8i9j0.pdf
```

## 🔒 Security

### Kong API Gateway (Layer 1)
- Rate limiting: 50 requests/minute
- Request size: 100MB maximum
- CORS enabled
- Request logging

### Service Validation (Layer 2)
- Input validation
- Chunk verification
- Session management
- Error handling

### Storage Protection (Layer 3)
- Unique filenames (prevents overwrites)
- Organized directories
- Automatic cleanup (24-hour sessions)

## 🐳 Docker

### Build

```bash
docker build -t metargb/storage-service:latest .
```

### Run with Docker Compose

```bash
docker-compose up -d storage-service
```

### Environment in Docker

```yaml
storage-service:
  environment:
    HTTP_PORT: 8059
    GRPC_PORT: 50059
    DB_HOST: mysql
    FTP_HOST: ftp.metargb.com
    TEMP_DIR: /tmp/storage-chunks
```

## 📊 Ports

| Port | Protocol | Purpose | Access |
|------|----------|---------|--------|
| 8059 | HTTP | REST API | Via Kong only |
| 50059 | gRPC | Internal microservices | Service-to-service |

**⚠️ Never expose port 8059 directly to clients in production!**

## 🔧 Development

### Prerequisites

- Go 1.21+
- MySQL 8.0+
- FTP server (or use mock for testing)

### Setup

```bash
# Clone repository
cd metargb-microservices/services/storage-service

# Install dependencies
go mod download

# Copy config
cp config.env.sample config.env

# Run tests
go test ./...

# Run service
go run cmd/server/main.go
```

### Project Structure

```
storage-service/
├── cmd/
│   └── server/
│       └── main.go           # Main application
├── internal/
│   ├── ftp/
│   │   ├── client.go         # FTP client
│   │   ├── mock_client.go    # Mock FTP (testing)
│   │   └── interface.go      # FTP interface
│   ├── handler/
│   │   ├── http_handler.go   # HTTP REST handlers
│   │   ├── storage_handler.go # gRPC handlers
│   │   └── image_handler.go  # Image handlers
│   ├── models/
│   │   └── image.go          # Data models
│   ├── repository/
│   │   └── image_repository.go # Database layer
│   └── service/
│       ├── chunk_manager.go  # Chunk upload logic
│       ├── storage_service.go # Business logic
│       └── image_service.go  # Image operations
├── test_server.go            # Standalone test server
├── test_upload.html          # Interactive test page
├── ARCHITECTURE.md           # Architecture documentation
└── README.md                 # This file
```

## 📝 API Comparison with Laravel

| Feature | Laravel Controller | Storage Service |
|---------|-------------------|-----------------|
| Endpoint | `/api/upload` | `/api/upload` |
| Method | POST | POST |
| Auth | None | None |
| Request | multipart/form-data | multipart/form-data |
| Response | `done`, `path`, `name`, `mime_type` | Same ✅ |
| Chunks | ✅ | ✅ |
| Progress | ✅ | ✅ |
| Organization | MIME/Date | MIME/Date |
| Filename | MD5 hash | MD5 hash |

## 🐛 Troubleshooting

### Issue: Connection Refused

**Problem:** Can't connect to `http://localhost:8000/api/upload`

**Solution:** Ensure Kong is running:
```bash
docker-compose up -d kong
docker-compose logs kong
```

### Issue: 404 Not Found

**Problem:** Route not found in Kong

**Solution:** Verify Kong configuration:
```bash
# Check Kong routes
curl http://localhost:8001/routes | jq '.data[] | select(.paths[] | contains("upload"))'

# Restart Kong
docker-compose restart kong
```

### Issue: File Upload Fails

**Problem:** Upload fails with FTP error

**Solution:** Check FTP configuration or use mock FTP:
```bash
# Use mock FTP for testing
go run test_server.go
```

### Issue: CORS Error

**Problem:** Browser blocks request

**Solution:** Ensure Kong CORS plugin is enabled:
```bash
curl http://localhost:8001/plugins | jq '.data[] | select(.name=="cors")'
```

## 📚 Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md) - Detailed architecture
- [Upload Endpoint API](../../docs/UPLOAD_ENDPOINT.md) - API documentation
- [Chunk Upload Implementation](../../docs/STORAGE_CHUNK_UPLOAD.md) - Technical details

## 🤝 Contributing

1. Follow the microservices architecture
2. All client access MUST go through Kong
3. Write tests for new features
4. Update documentation

## 📄 License

Part of MetaRGB platform.

---

**Remember:** In production, clients should access `http://api.metargb.com/api/upload`, which routes through Kong to the storage service. Direct service access is only for development/testing!

