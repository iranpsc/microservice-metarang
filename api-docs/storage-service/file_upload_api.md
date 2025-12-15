# File Upload API Guide

## Summary
- `POST /api/upload` accepts large binary payloads through [pion/laravel-chunk-upload](https://github.com/pionl/laravel-chunk-upload) and automatically assembles chunks on the backend.
- Files are grouped by resolved MIME type and current date, then persisted under `storage/app/upload/{mime}/{YYYY-MM-DD}/`.
- Every completed upload responds with the persisted path, generated filename, and MIME label so clients can reference or move the file later.
- Chunk progress responses return a numeric `done` percentage while a transfer is still streaming.

## Access Control
- **Middleware:** the route is registered outside the authenticated API group, so no middleware runs by default; any caller can hit the endpoint unless upstream infrastructure blocks it.
- **Authentication expectations:** callers who need restricted access must enforce it at the edge (e.g., Sanctum token via reverse proxy) or relocate the route into an auth-protected group.
- **Authorization policy:** there is no Laravel policy or guard; uploads are accepted purely on the presence of the `file` field.

```246:246:routes/api.php
Route::post('upload', [FileUploadController::class, 'upload']);
```

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| POST | `/api/upload` | *(none)* | `FileUploadController@upload` | Stream a file (single or chunked) to backend storage. |

## Request Workflow
- **Entry point:** the controller instantiates a `FileReceiver` bound to the `file` form field; the handler is selected automatically based on request headers (Dropzone, FineUploader, etc.).
- **Chunk handling:** on partial uploads the handler returns the current chunk progress and waits for more data; once the last chunk arrives `saveFile()` is invoked.
- **Finalization:** the file is moved from its temporary chunk folder into the permanent storage path, then a JSON payload is returned.

```26:82:app/Http/Controllers/Api/FileUploadController.php
        $receiver = new FileReceiver("file", $request, HandlerFactory::classFromRequest($request));
        // ... existing code ...
        if ($save->isFinished()) {
            return $this->saveFile($save->getFile());
        }
        // ... existing code ...
        return response()->json([
            "done" => $handler->getPercentageDone(),
        ]);
```

## Validation & Constraints
- **Required field:** the request **must** include a `file` upload field; otherwise `UploadMissingFileException` is thrown and surfaces as an HTTP 400 response.
- **Size & type:** the controller does **not** impose explicit `max` or MIME validation. Practical limits derive from PHP’s `upload_max_filesize`/`post_max_size` and web server constraints. Clients should self-enforce acceptable size and type rules.
- **Chunk naming:** the package prefixes chunk names with the session id (when provided) to avoid collisions and cleans up stale chunks after ~3 hours via the scheduled task.

```10:44:config/chunk-upload.php
    'storage' => [
        'chunks' => 'chunks',
        'disk' => 'local',
    ],
    'clear' => [
        'timestamp' => '-3 HOURS',
        'schedule' => [
            'enabled' => true,
            'cron' => '25 * * * *',
        ],
    ],
```

## Response Payloads
- **In-progress chunk:** `200 OK` with `{ "done": <float 0-100> }` representing server-side completion percentage.
- **Completed upload:** `200 OK` with `{ "path": "upload/<mime>/<date>/", "name": "<original>_<hash>.<ext>", "mime_type": "<mime>" }`.
- **Missing file:** `400 Bad Request` (exception bubbled by chunk library) when no `file` field is present.
- **Server errors:** `500` if filesystem paths are misconfigured or the move operation fails.

```64:81:app/Http/Controllers/Api/FileUploadController.php
        $filePath = "upload/{$mime}/{$dateFolder}/";
        $finalPath = storage_path("app/" . $filePath);
        $file->move($finalPath, $fileName);
        return response()->json([
            'path' => $filePath,
            'name' => $fileName,
            'mime_type' => $mime
        ]);
```

## Storage & Retention
- **Permanent location:** files land under `storage/app/upload/...`; expose them publicly only after copying or linking to a web-facing disk.
- **Chunk temp directory:** intermediate chunks live at `storage/app/chunks`; the scheduled cleanup wipes anything older than three hours to reclaim disk space.
- **Collision avoidance:** filenames append an MD5 hash of the current timestamp to the client-provided basename, reducing the risk of overwriting existing assets.

## Operational Notes
- Place the route behind authentication middleware if anonymous uploads are not desired; alternatively, apply rate limiting via `$request->middleware('throttle:...')`.
- Ensure the `storage/app/upload` tree has appropriate permissions for the web server user; misconfigured permissions surface as `500` errors when moving completed files.
- To serve uploaded assets publicly, set up a subsequent job or listener that copies the stored file to a public disk (e.g., `public` or S3) and records the accessible URL.
- Monitor chunk cleanup by registering Laravel’s scheduler (`php artisan schedule:work`) so old partial uploads do not accumulate indefinitely.

