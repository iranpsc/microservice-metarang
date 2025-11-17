# Storage Service - Chunk Upload Implementation

## Overview

The storage-service now supports **chunk upload functionality** similar to the Laravel `FileUploadController.php`. This feature enables:

- **Resumable uploads**: Upload large files in smaller chunks
- **Progress tracking**: Monitor upload progress in real-time
- **Automatic file assembly**: Chunks are automatically assembled when complete
- **Organized storage**: Files are organized by MIME type and date
- **Session management**: Automatic cleanup of expired upload sessions

## Features Comparison with Laravel FileUploadController

| Feature | Laravel Controller | Storage Service |
|---------|-------------------|-----------------|
| Chunk Reception | ✅ Pion\ChunkUpload | ✅ Custom ChunkManager |
| Progress Tracking | ✅ Percentage done | ✅ Real-time progress % |
| File Organization | ✅ MIME/Date folders | ✅ MIME/Date folders |
| Unique Filename | ✅ MD5 timestamp | ✅ MD5 timestamp |
| Session Management | ✅ Implicit | ✅ Explicit with cleanup |
| Concurrent Uploads | ✅ File-based | ✅ Session-based |

## Architecture

### Components

1. **ChunkManager** (`internal/service/chunk_manager.go`)
   - Manages upload sessions
   - Stores chunks temporarily
   - Assembles chunks into final file
   - Handles session cleanup

2. **StorageService** (`internal/service/storage_service.go`)
   - Integrates ChunkManager
   - Uploads assembled files to FTP
   - Generates file URLs

3. **StorageHandler** (`internal/handler/storage_handler.go`)
   - Exposes gRPC endpoint `ChunkUpload`
   - Validates requests
   - Returns progress information

### Proto Definition

```protobuf
service FileStorageService {
  rpc ChunkUpload(ChunkUploadRequest) returns (ChunkUploadResponse);
}

message ChunkUploadRequest {
  string upload_id = 1;        // Unique identifier for upload session
  bytes chunk_data = 2;        // The chunk data
  int32 chunk_index = 3;       // Index of this chunk (0-based)
  int32 total_chunks = 4;      // Total number of chunks
  string filename = 5;         // Original filename
  string content_type = 6;     // MIME type
  int64 total_size = 7;        // Total file size in bytes
  string upload_path = 8;      // Optional: custom upload path
}

message ChunkUploadResponse {
  bool success = 1;
  string message = 2;
  double percentage_done = 3;  // Upload progress (0-100)
  bool is_finished = 4;        // True when all chunks uploaded
  string file_url = 5;         // File URL (only when finished)
  string file_path = 6;        // File path in storage (only when finished)
  string final_filename = 7;   // Final filename (only when finished)
}
```

## How It Works

### 1. Upload Session Creation

When the first chunk is received:
1. ChunkManager creates a new upload session
2. A temporary directory is created for this session
3. Session metadata is stored in memory

### 2. Chunk Processing

For each chunk received:
1. Chunk data is written to a temporary file
2. Progress is calculated based on received chunks
3. Response includes current progress percentage

### 3. File Assembly

When all chunks are received:
1. Chunks are assembled in order
2. Unique filename is generated (like Laravel: `filename_md5hash.ext`)
3. File is organized into `upload/{mime-type}/{YYYY-MM-DD}/` structure
4. Assembled file is uploaded to FTP
5. Session is cleaned up

### 4. Session Cleanup

- Sessions older than 24 hours are automatically cleaned up
- Manual cleanup happens after successful upload
- Failed uploads can be retried with the same `upload_id`

## Usage Examples

### Example 1: Go Client

```go
package main

import (
    "context"
    "fmt"
    "io"
    "os"
    
    "google.golang.org/grpc"
    storagepb "metargb/shared/pb/storage"
)

func uploadFileInChunks(client storagepb.FileStorageServiceClient, filePath string) error {
    // Open file
    file, err := os.Open(filePath)
    if err != nil {
        return err
    }
    defer file.Close()
    
    // Get file info
    fileInfo, err := file.Stat()
    if err != nil {
        return err
    }
    
    // Configuration
    chunkSize := 1024 * 1024 // 1MB chunks
    totalSize := fileInfo.Size()
    totalChunks := int32((totalSize + int64(chunkSize) - 1) / int64(chunkSize))
    uploadID := fmt.Sprintf("upload_%d", time.Now().UnixNano())
    
    fmt.Printf("Uploading file: %s (%d bytes) in %d chunks\n", 
        filePath, totalSize, totalChunks)
    
    // Upload chunks
    buffer := make([]byte, chunkSize)
    for chunkIndex := int32(0); chunkIndex < totalChunks; chunkIndex++ {
        // Read chunk
        n, err := file.Read(buffer)
        if err != nil && err != io.EOF {
            return err
        }
        
        // Send chunk
        req := &storagepb.ChunkUploadRequest{
            UploadId:     uploadID,
            ChunkData:    buffer[:n],
            ChunkIndex:   chunkIndex,
            TotalChunks:  totalChunks,
            Filename:     fileInfo.Name(),
            ContentType:  "application/octet-stream",
            TotalSize:    totalSize,
        }
        
        resp, err := client.ChunkUpload(context.Background(), req)
        if err != nil {
            return fmt.Errorf("chunk %d upload failed: %w", chunkIndex, err)
        }
        
        fmt.Printf("Progress: %.2f%% - %s\n", resp.PercentageDone, resp.Message)
        
        if resp.IsFinished {
            fmt.Printf("✅ Upload complete!\n")
            fmt.Printf("   File URL: %s\n", resp.FileUrl)
            fmt.Printf("   File Path: %s\n", resp.FilePath)
            fmt.Printf("   Final Filename: %s\n", resp.FinalFilename)
        }
    }
    
    return nil
}
```

### Example 2: Laravel Integration

```php
<?php

namespace App\Services;

use Grpc\ChannelCredentials;
use Storage\FileStorageServiceClient;
use Storage\ChunkUploadRequest;

class GrpcStorageService
{
    private $client;
    
    public function __construct()
    {
        $this->client = new FileStorageServiceClient(
            env('STORAGE_SERVICE_HOST', 'localhost:50059'),
            ['credentials' => ChannelCredentials::createInsecure()]
        );
    }
    
    public function uploadFileInChunks($filePath, $chunkSize = 1048576)
    {
        $fileSize = filesize($filePath);
        $totalChunks = ceil($fileSize / $chunkSize);
        $uploadId = 'upload_' . uniqid();
        $filename = basename($filePath);
        $mimeType = mime_content_type($filePath);
        
        $handle = fopen($filePath, 'rb');
        $chunkIndex = 0;
        
        while (!feof($handle)) {
            $chunkData = fread($handle, $chunkSize);
            
            $request = new ChunkUploadRequest([
                'upload_id' => $uploadId,
                'chunk_data' => $chunkData,
                'chunk_index' => $chunkIndex,
                'total_chunks' => $totalChunks,
                'filename' => $filename,
                'content_type' => $mimeType,
                'total_size' => $fileSize,
            ]);
            
            [$response, $status] = $this->client->ChunkUpload($request)->wait();
            
            if ($status->code !== 0) {
                fclose($handle);
                throw new \Exception("Upload failed: " . $status->details);
            }
            
            if ($response->getIsFinished()) {
                fclose($handle);
                return [
                    'url' => $response->getFileUrl(),
                    'path' => $response->getFilePath(),
                    'filename' => $response->getFinalFilename(),
                ];
            }
            
            $chunkIndex++;
        }
        
        fclose($handle);
    }
}
```

### Example 3: Frontend JavaScript

```javascript
async function uploadFileInChunks(file, uploadUrl) {
    const chunkSize = 1024 * 1024; // 1MB
    const totalChunks = Math.ceil(file.size / chunkSize);
    const uploadId = `upload_${Date.now()}`;
    
    console.log(`Uploading ${file.name} in ${totalChunks} chunks`);
    
    for (let chunkIndex = 0; chunkIndex < totalChunks; chunkIndex++) {
        const start = chunkIndex * chunkSize;
        const end = Math.min(start + chunkSize, file.size);
        const chunk = file.slice(start, end);
        
        // Convert chunk to base64 for gRPC-Web
        const chunkData = await chunk.arrayBuffer();
        
        const response = await fetch(uploadUrl, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                upload_id: uploadId,
                chunk_data: btoa(String.fromCharCode(...new Uint8Array(chunkData))),
                chunk_index: chunkIndex,
                total_chunks: totalChunks,
                filename: file.name,
                content_type: file.type,
                total_size: file.size,
            }),
        });
        
        const result = await response.json();
        
        console.log(`Progress: ${result.percentage_done}% - ${result.message}`);
        
        if (result.is_finished) {
            console.log('✅ Upload complete!');
            return {
                url: result.file_url,
                path: result.file_path,
                filename: result.final_filename,
            };
        }
    }
}

// Usage
const fileInput = document.querySelector('input[type="file"]');
fileInput.addEventListener('change', async (e) => {
    const file = e.target.files[0];
    const result = await uploadFileInChunks(file, '/api/chunk-upload');
    console.log('File uploaded:', result);
});
```

## Configuration

### Environment Variables

Add to `config.env` or `.env`:

```bash
# Temporary directory for chunk storage
TEMP_DIR=/tmp/storage-chunks

# FTP configuration (existing)
FTP_HOST=ftp.example.com
FTP_PORT=21
FTP_USER=ftpuser
FTP_PASSWORD=ftppass
FTP_BASE_URL=https://cdn.example.com

# gRPC configuration
GRPC_PORT=50059
```

### Chunk Size Recommendations

| Use Case | Recommended Chunk Size |
|----------|----------------------|
| Mobile apps | 256 KB - 512 KB |
| Web browsers | 512 KB - 1 MB |
| Internal services | 1 MB - 5 MB |
| High-bandwidth | 5 MB - 10 MB |

## File Organization

Files are automatically organized in the following structure:

```
upload/
├── image-jpeg/
│   ├── 2025-10-30/
│   │   ├── photo_a3f2d8e9b1c4f7a6.jpg
│   │   └── avatar_d4e5f6g7h8i9j0k1.jpg
│   └── 2025-10-31/
├── video-mp4/
│   └── 2025-10-30/
│       └── video_b2c3d4e5f6g7h8i9.mp4
└── application-pdf/
    └── 2025-10-30/
        └── document_c3d4e5f6g7h8i9j0.pdf
```

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `upload_id is required` | Missing upload_id | Generate unique ID per upload |
| `invalid chunk_index` | Index out of range | Ensure 0 ≤ index < total_chunks |
| `chunk_data is empty` | Empty chunk sent | Check file reading logic |
| `failed to assemble file` | Missing chunks | Retry failed chunks |
| `failed to upload file` | FTP error | Check FTP credentials |

### Retry Logic Example

```go
func uploadChunkWithRetry(client storagepb.FileStorageServiceClient, 
                         req *storagepb.ChunkUploadRequest, 
                         maxRetries int) error {
    var err error
    for i := 0; i < maxRetries; i++ {
        _, err = client.ChunkUpload(context.Background(), req)
        if err == nil {
            return nil
        }
        time.Sleep(time.Second * time.Duration(i+1))
    }
    return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
}
```

## Performance Considerations

### Memory Usage

- Chunks are stored on disk, not in memory
- Memory footprint per session: ~1-2 KB (metadata only)
- Disk usage cleaned up automatically after 24 hours

### Concurrency

- Multiple uploads can happen simultaneously
- Each upload has its own session directory
- Thread-safe session management with mutex locks

### Best Practices

1. **Choose appropriate chunk size**: Balance between network efficiency and memory usage
2. **Use unique upload IDs**: Prevents session conflicts
3. **Implement retry logic**: Handle network failures gracefully
4. **Monitor progress**: Update UI to show upload progress
5. **Clean up on failure**: Cancel upload and cleanup if user aborts

## Testing

### Manual Test with cURL

```bash
# Create a test file
dd if=/dev/urandom of=test_file.bin bs=1M count=5

# Split into chunks
split -b 1M test_file.bin chunk_

# Upload chunks
UPLOAD_ID="test_$(date +%s)"
for i in {0..4}; do
  grpcurl -plaintext \
    -d "{
      \"upload_id\": \"$UPLOAD_ID\",
      \"chunk_data\": \"$(base64 < chunk_a$i)\",
      \"chunk_index\": $i,
      \"total_chunks\": 5,
      \"filename\": \"test_file.bin\",
      \"content_type\": \"application/octet-stream\",
      \"total_size\": 5242880
    }" \
    localhost:50059 storage.FileStorageService/ChunkUpload
done
```

### Unit Test Example

```go
func TestChunkUpload(t *testing.T) {
    // Create test chunk manager
    tempDir := t.TempDir()
    cm, err := service.NewChunkManager(tempDir)
    require.NoError(t, err)
    
    // Create test session
    session, err := cm.GetOrCreateSession(
        "test-upload",
        "test.txt",
        "text/plain",
        3,
        300,
        "",
    )
    require.NoError(t, err)
    
    // Upload chunks
    for i := int32(0); i < 3; i++ {
        chunk := []byte(fmt.Sprintf("chunk %d data", i))
        err := cm.SaveChunk(session, i, chunk)
        require.NoError(t, err)
    }
    
    // Verify completion
    assert.True(t, cm.IsComplete(session))
    assert.Equal(t, 100.0, cm.GetProgress(session))
    
    // Assemble file
    data, path, err := cm.AssembleFile(session)
    require.NoError(t, err)
    assert.NotEmpty(t, data)
    assert.Contains(t, path, "upload/text-plain")
}
```

## Monitoring and Debugging

### Logs to Watch For

```
Chunk manager initialized with temp directory: /tmp/storage-chunks
Chunk 1/10 uploaded
Chunk 2/10 uploaded
...
Chunk 10/10 uploaded
File uploaded successfully
```

### Metrics to Monitor

- Active upload sessions count
- Average upload time per file
- Chunk upload failure rate
- Temp directory disk usage
- Session cleanup frequency

## Migration from Laravel Controller

If migrating from the Laravel `FileUploadController` to the storage-service:

1. **Client changes**: Replace HTTP multipart uploads with gRPC chunk calls
2. **File paths**: Update code to use returned `file_url` from response
3. **Progress tracking**: Use `percentage_done` from response instead of JavaScript progress events
4. **Error handling**: Handle gRPC errors instead of HTTP status codes

## Troubleshooting

### Issue: Uploads fail with "session not found"

**Cause**: Session expired or cleaned up  
**Solution**: Ensure upload completes within 24 hours or adjust cleanup interval

### Issue: Assembled file is corrupted

**Cause**: Chunks received out of order or missing  
**Solution**: Verify all chunks sent and `chunk_index` is correct

### Issue: High disk usage in temp directory

**Cause**: Many incomplete uploads or cleanup not running  
**Solution**: Check cleanup goroutine is running and consider reducing cleanup interval

## Conclusion

The chunk upload feature provides a robust, production-ready solution for handling large file uploads in the storage-service, matching and extending the functionality of the Laravel `FileUploadController.php`.

For questions or issues, please refer to the main [Storage Service Documentation](../services/storage-service/README.md).

