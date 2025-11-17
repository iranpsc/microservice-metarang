# Storage Service - /upload Endpoint

## Overview

The storage service now exposes a public HTTP REST endpoint at `/api/upload` for chunk-based file uploads, matching the Laravel `FileUploadController` functionality. This endpoint **does not require authentication** and is accessible to all clients.

## Endpoint Details

```
POST /api/upload
```

- **Method**: POST
- **Authentication**: None required (public endpoint)
- **Content-Type**: multipart/form-data
- **Max File Size**: 100MB per request
- **Rate Limiting**: 50 requests/minute, 1000 requests/hour

## Request Format

### Form Data Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | File | ✅ Yes | The file chunk to upload |
| `upload_id` | String | ⚠️ Auto-generated | Unique identifier for upload session |
| `chunk_index` | Integer | ⚠️ Default: 0 | Index of current chunk (0-based) |
| `total_chunks` | Integer | ⚠️ Default: 1 | Total number of chunks |
| `filename` | String | ⚠️ Auto-detected | Original filename |
| `content_type` | String | ⚠️ Auto-detected | MIME type |
| `total_size` | Integer | ⚠️ Auto-detected | Total file size in bytes |
| `upload_path` | String | ❌ No | Custom upload path (optional) |

**Note**: Fields marked with ⚠️ are optional and will be auto-generated/detected if not provided.

## Response Format

### During Upload (Progress Response)

```json
{
  "success": true,
  "done": 50.0,
  "message": "Chunk 5/10 uploaded",
  "is_finished": false
}
```

### Upload Complete Response

```json
{
  "success": true,
  "done": 100.0,
  "message": "File uploaded successfully",
  "is_finished": true,
  "path": "upload/image-jpeg/2025-10-30/photo_a3f2d8e9b1c4f7a6.jpg",
  "name": "photo_a3f2d8e9b1c4f7a6.jpg",
  "mime_type": "image/jpeg"
}
```

### Response Fields

| Field | Type | Description |
|-------|------|-------------|
| `success` | Boolean | Always true if no error |
| `done` | Float | Upload progress percentage (0-100) |
| `message` | String | Status message |
| `is_finished` | Boolean | True when all chunks uploaded |
| `path` | String | File path/URL (only when finished) |
| `name` | String | Final filename (only when finished) |
| `mime_type` | String | File MIME type (only when finished) |

## Usage Examples

### Example 1: Single File Upload (No Chunks)

```bash
curl -X POST http://localhost:8000/api/upload \
  -F "file=@photo.jpg"
```

Response:
```json
{
  "success": true,
  "done": 100.0,
  "message": "File uploaded successfully",
  "is_finished": true,
  "path": "upload/image-jpeg/2025-10-30/photo_a3f2d8e9b1c4f7a6.jpg",
  "name": "photo_a3f2d8e9b1c4f7a6.jpg",
  "mime_type": "image/jpeg"
}
```

### Example 2: Chunked Upload (JavaScript)

```javascript
async function uploadFileInChunks(file) {
    const chunkSize = 1024 * 1024; // 1MB chunks
    const totalChunks = Math.ceil(file.size / chunkSize);
    const uploadId = `upload_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    
    console.log(`📤 Uploading ${file.name} in ${totalChunks} chunks...`);
    
    for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
        const start = chunkIndex * chunkSize;
        const end = Math.min(start + chunkSize, file.size);
        const chunk = file.slice(start, end);
        
        const formData = new FormData();
        formData.append('file', chunk);
        formData.append('upload_id', uploadId);
        formData.append('chunk_index', chunkIndex);
        formData.append('total_chunks', totalChunks);
        formData.append('filename', file.name);
        formData.append('content_type', file.type);
        formData.append('total_size', file.size);
        
        const response = await fetch('http://localhost:8000/api/upload', {
            method: 'POST',
            body: formData
        });
        
        const result = await response.json();
        console.log(`Progress: ${result.done}% - ${result.message}`);
        
        if (result.is_finished) {
            console.log('✅ Upload complete!');
            return {
                url: result.path,
                filename: result.name,
                mimeType: result.mime_type
            };
        }
    }
}

// Usage
document.querySelector('#fileInput').addEventListener('change', async (e) => {
    const file = e.target.files[0];
    if (file) {
        const result = await uploadFileInChunks(file);
        console.log('Uploaded:', result);
    }
});
```

### Example 3: React Component

```jsx
import React, { useState } from 'react';

function FileUploader() {
    const [progress, setProgress] = useState(0);
    const [uploading, setUploading] = useState(false);
    const [result, setResult] = useState(null);
    
    const uploadFile = async (file) => {
        setUploading(true);
        const chunkSize = 1024 * 1024; // 1MB
        const totalChunks = Math.ceil(file.size / chunkSize);
        const uploadId = `upload_${Date.now()}`;
        
        for (let i = 0; i < totalChunks; i++) {
            const chunk = file.slice(i * chunkSize, (i + 1) * chunkSize);
            const formData = new FormData();
            
            formData.append('file', chunk);
            formData.append('upload_id', uploadId);
            formData.append('chunk_index', i);
            formData.append('total_chunks', totalChunks);
            formData.append('filename', file.name);
            formData.append('content_type', file.type);
            formData.append('total_size', file.size);
            
            const response = await fetch('/api/upload', {
                method: 'POST',
                body: formData
            });
            
            const data = await response.json();
            setProgress(data.done);
            
            if (data.is_finished) {
                setResult(data);
                setUploading(false);
                return data;
            }
        }
    };
    
    const handleFileChange = (e) => {
        const file = e.target.files[0];
        if (file) uploadFile(file);
    };
    
    return (
        <div>
            <input 
                type="file" 
                onChange={handleFileChange} 
                disabled={uploading}
            />
            {uploading && <progress value={progress} max="100">{progress}%</progress>}
            {result && (
                <div>
                    <p>✅ Uploaded: {result.name}</p>
                    <p>URL: {result.path}</p>
                </div>
            )}
        </div>
    );
}

export default FileUploader;
```

### Example 4: PHP/Laravel Client

```php
<?php

use Illuminate\Support\Facades\Http;

class StorageServiceClient
{
    private $baseUrl;
    
    public function __construct()
    {
        $this->baseUrl = env('STORAGE_SERVICE_URL', 'http://localhost:8000');
    }
    
    public function uploadFile($filePath, $chunkSize = 1048576)
    {
        $fileSize = filesize($filePath);
        $totalChunks = ceil($fileSize / $chunkSize);
        $uploadId = 'upload_' . uniqid() . '_' . time();
        $filename = basename($filePath);
        $mimeType = mime_content_type($filePath);
        
        $handle = fopen($filePath, 'rb');
        $chunkIndex = 0;
        
        while (!feof($handle)) {
            $chunkData = fread($handle, $chunkSize);
            
            // Create temporary file for chunk
            $tempFile = tmpfile();
            fwrite($tempFile, $chunkData);
            $tempPath = stream_get_meta_data($tempFile)['uri'];
            
            $response = Http::attach('file', $chunkData, $filename)
                ->post("{$this->baseUrl}/api/upload", [
                    'upload_id' => $uploadId,
                    'chunk_index' => $chunkIndex,
                    'total_chunks' => $totalChunks,
                    'filename' => $filename,
                    'content_type' => $mimeType,
                    'total_size' => $fileSize,
                ]);
            
            fclose($tempFile);
            
            if (!$response->successful()) {
                fclose($handle);
                throw new \Exception('Upload failed: ' . $response->body());
            }
            
            $result = $response->json();
            
            if ($result['is_finished'] ?? false) {
                fclose($handle);
                return [
                    'path' => $result['path'],
                    'name' => $result['name'],
                    'mime_type' => $result['mime_type'],
                ];
            }
            
            $chunkIndex++;
        }
        
        fclose($handle);
        throw new \Exception('Upload incomplete');
    }
}

// Usage
$client = new StorageServiceClient();
$result = $client->uploadFile('/path/to/file.jpg');
echo "Uploaded to: {$result['path']}";
```

### Example 5: Python Client

```python
import requests
import math
import os
import time

def upload_file_in_chunks(file_path, chunk_size=1024*1024):
    """Upload a file in chunks to the storage service"""
    
    file_size = os.path.getsize(file_path)
    total_chunks = math.ceil(file_size / chunk_size)
    upload_id = f"upload_{int(time.time())}"
    filename = os.path.basename(file_path)
    
    print(f"📤 Uploading {filename} in {total_chunks} chunks...")
    
    with open(file_path, 'rb') as f:
        for chunk_index in range(total_chunks):
            chunk_data = f.read(chunk_size)
            
            files = {'file': (filename, chunk_data)}
            data = {
                'upload_id': upload_id,
                'chunk_index': chunk_index,
                'total_chunks': total_chunks,
                'filename': filename,
                'content_type': 'application/octet-stream',
                'total_size': file_size
            }
            
            response = requests.post(
                'http://localhost:8000/api/upload',
                files=files,
                data=data
            )
            
            result = response.json()
            print(f"Progress: {result['done']}% - {result['message']}")
            
            if result.get('is_finished'):
                print('✅ Upload complete!')
                return {
                    'path': result['path'],
                    'name': result['name'],
                    'mime_type': result['mime_type']
                }
    
    raise Exception('Upload incomplete')

# Usage
result = upload_file_in_chunks('/path/to/file.mp4')
print(f"Uploaded to: {result['path']}")
```

## Architecture Flow

```
┌─────────────┐         ┌──────────────┐         ┌─────────────────┐
│   Client    │────────▶│     Kong     │────────▶│ Storage Service │
│ (Browser/   │         │  API Gateway │         │  (HTTP Server)  │
│  Mobile)    │◀────────│              │◀────────│   Port: 8059    │
└─────────────┘         └──────────────┘         └─────────────────┘
                              │                            │
                              │                            │
                              │                            ▼
                              │                   ┌─────────────────┐
                              │                   │ Chunk Manager   │
                              │                   │  (Session Mgmt) │
                              │                   └─────────────────┘
                              │                            │
                              │                            ▼
                              │                   ┌─────────────────┐
                              └──────────────────▶│   FTP Server    │
                                                  │  (File Storage) │
                                                  └─────────────────┘
```

## File Organization

Files are automatically organized following the same structure as Laravel:

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

## Error Responses

### 400 Bad Request

```json
{
  "success": false,
  "error": "Failed to parse form"
}
```

### 500 Internal Server Error

```json
{
  "success": false,
  "error": "Upload failed: FTP connection error"
}
```

## Testing

### Test with cURL (Single File)

```bash
curl -X POST http://localhost:8000/api/upload \
  -F "file=@test.jpg" \
  -v
```

### Test with cURL (Chunked)

```bash
# Split file into chunks
split -b 1M large_file.mp4 chunk_

# Upload each chunk
UPLOAD_ID="test_$(date +%s)"
for i in {0..4}; do
  curl -X POST http://localhost:8000/api/upload \
    -F "file=@chunk_aa" \
    -F "upload_id=$UPLOAD_ID" \
    -F "chunk_index=$i" \
    -F "total_chunks=5" \
    -F "filename=large_file.mp4" \
    -F "content_type=video/mp4" \
    -F "total_size=5242880"
done
```

## Rate Limiting

The endpoint has the following rate limits:
- **50 requests per minute** per IP
- **1000 requests per hour** per IP

When rate limit is exceeded, you'll receive a `429 Too Many Requests` response.

## Best Practices

1. **Generate Unique Upload IDs**: Use timestamp + random string to avoid conflicts
2. **Handle Network Failures**: Implement retry logic for failed chunks
3. **Show Progress**: Update UI based on `done` percentage
4. **Validate Files**: Check file types and sizes before uploading
5. **Clean Up on Cancel**: If user cancels, chunks will auto-expire in 24 hours

## Comparison with Laravel

| Feature | Laravel Route | Microservice Route |
|---------|--------------|-------------------|
| Endpoint | `POST /api/upload` | `POST /api/upload` |
| Authentication | ❌ None | ❌ None |
| Chunk Support | ✅ | ✅ |
| Progress Tracking | ✅ | ✅ |
| Response Format | Same | Same |
| File Organization | MIME/Date | MIME/Date |
| Max Upload Size | 100MB | 100MB |

## Deployment

### Docker Compose

The HTTP endpoint is automatically exposed on port `8059`:

```yaml
storage-service:
  ports:
    - "50059:50059"  # gRPC
    - "8059:8059"    # HTTP REST API
```

### Kong Configuration

Kong routes `/api/upload` to the HTTP server without authentication:

```yaml
- name: storage-service-upload
  url: http://storage-service:8059
  routes:
    - paths: ["/api/upload"]
      methods: ["POST", "OPTIONS"]
  plugins:
    - name: cors
    - name: request-size-limiting
```

## Monitoring

### Health Check

```bash
curl http://localhost:8059/health
```

Response:
```json
{
  "status": "healthy",
  "service": "storage-service",
  "version": "1.0.0"
}
```

### Logs

Watch for these log messages:
```
✅ HTTP server listening on port 8059
📤 Chunk upload endpoint: http://localhost:8059/upload
Chunk 1/10 uploaded
Progress: 50.00%
File uploaded successfully
```

## Troubleshooting

### Issue: 404 Not Found

**Cause**: Kong not routing properly  
**Solution**: Check Kong configuration and restart Kong

### Issue: 413 Payload Too Large

**Cause**: Chunk exceeds 100MB limit  
**Solution**: Reduce chunk size

### Issue: CORS errors

**Cause**: Missing CORS headers  
**Solution**: Ensure Kong CORS plugin is enabled

## Conclusion

The `/api/upload` endpoint provides a drop-in replacement for the Laravel `FileUploadController` with enhanced features including automatic cleanup, better concurrency handling, and RESTful HTTP access alongside gRPC support.

For more details on the underlying implementation, see [STORAGE_CHUNK_UPLOAD.md](./STORAGE_CHUNK_UPLOAD.md).

