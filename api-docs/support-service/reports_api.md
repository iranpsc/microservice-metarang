# Reports API Guide

## Summary
- `GET /api/reports` returns the current user's reports as a simple-paginated feed ordered by `created_at` descending.
- `GET /api/reports/{report}` loads a single report with its stored attachments resolved to public URLs.
- `POST /api/reports` opens a report against a target URL, persists optional media (≤ 5 files, 1 MB each), and links everything to the caller.

```146:147:routes/api.php
Route::apiResource('reports', ReportController::class)->only(['index', 'show', 'store']);
```

All routes inherit the enclosing `auth:sanctum`, `verified`, and `activity` middleware group.

## Controller Walkthrough

```17:63:app/Http/Controllers/Api/V1/ReportController.php
public function index()
{
    $reports = Report::whereBelongsTo(request()->user())
        ->select('id', 'user_id', 'title', 'subject', 'status', 'created_at')
        ->latest()
        ->simplePaginate(10);

    return ReportResource::collection($reports);
}

public function show(Report $report)
{
    $report->load('images');

    return new ReportResource($report);
}

public function store(ReportRequest $request)
{
    $report = $request->user()->reports()->create($request->only([
        'subject',
        'title',
        'content',
        'url'
    ]));

    if ($request->hasFile('attachments')) {
        foreach ($request->file('attachments') as $file) {
            $url = $file->store('reports', 'public');
            $report->images()->create(['url' => $url]);
        }
    }

    return new ReportResource($report);
}
```

- `index` scopes strictly to the authenticated user via `whereBelongsTo`.
- `show` relies on implicit route-model binding and eager-loads `images` so attachment URLs are emitted.
- `store` creates the report under the caller, stores uploaded files under `storage/app/public/reports`, and records each as a morph-many `Image`.
- Responses are wrapped with `ReportResource`, which Jalali-formats timestamps.

## Authentication & Middleware
- `auth:sanctum` resolves and authenticates the caller.
- `verified` ensures the account has completed verification.
- `activity` logs access for auditing and rate-limits via the `api` limiter (60/min by user or IP).
- No additional middleware is registered at the controller level.

## Policy Coverage
- There is **no** `ReportPolicy`, and `ReportController` does not call `authorizeResource`. Policy hooks are therefore unused.
- Authorization relies entirely on middleware and manual scoping inside controller methods.
- Consequence: any authenticated, verified, active user can request `GET /api/reports/{id}` for an arbitrary report id and receive the payload unless database constraints (e.g., non-existent id) block it. Consider adding explicit owner checks or a policy before exposing the endpoint publicly.

## Route Registry
| Method | Path | Middleware → Policy Gate | Controller action | Notes |
| --- | --- | --- | --- | --- |
| GET | `/api/reports` | `auth:sanctum`, `verified`, `activity` → _manual scope only_ | `ReportController@index` | Simple pagination (`simplePaginate(10)`); returns only the caller's reports with lightweight fields. |
| GET | `/api/reports/{report}` | `auth:sanctum`, `verified`, `activity` → _no policy_ | `ReportController@show` | Returns the bound report with attachments. No automatic ownership enforcement. |
| POST | `/api/reports` | `auth:sanctum`, `verified`, `activity` → _no policy_ | `ReportController@store` | Creates a report tied to the caller; accepts optional attachments saved to the public disk. |

## Request Contracts
- **Create Report (`POST /api/reports`)**
  - `subject` `string` (required) — must be one of `displayError`, `spellingError`, `codingError`, `FPSError`, `disrespect`.
  - `title` `string` (required, ≤ 130 chars).
  - `content` `string` (required, ≤ 2000 chars).
  - `url` `string` (required) — must pass `active_url`.
  - `attachments` `array` (optional, ≤ 5 items).
  - `attachments.*` `file` (optional) — MIME `png`, `jpg`, `jpeg`, `pdf`, size ≤ 1024 KB.

Validation failures return HTTP 422 with field errors. Authorization failures (e.g., missing middleware requirements) return HTTP 401/403 via upstream middleware.

## Response Shape

```17:29:app/Http/Resources/ReportResource.php
return [
    'id' => (string)$this->id,
    'title' => $this->title,
    'url' => $this->whenNotNull($this->url),
    'subject' => $this->subject,
    'content' => $this->whenNotNull($this->content),
    'attachments' => $this->whenLoaded('images', function () {
        return $this->images->map(function ($image) {
            return url('uploads/' . $image->url);
        });
    }),
    'datetime' => jdate($this->created_at)->format('Y/m/d H:i:s'),
];
```

- IDs are stringified.
- Attachment URLs point to `/uploads/{path}`; ensure `php artisan storage:link` is in place so the public disk symlink exists.
- `datetime` is Jalali-formatted using `jdate`.
- Empty optional fields (`content`, `url`, `attachments`) are omitted from the payload.

## Usage Notes
- Attachments are stored under `storage/app/public/reports`; clean up unused files manually if reports are deleted elsewhere.
- Because `index` uses simple pagination, responses omit `total`/`last_page`. Clients must rely on `next_page_url` for iteration.
- After `store`, attachments are not revalidated by `ReportResource`; clients should only trust URLs returned in the payload.
- Add an authorization layer before exposing the `show` endpoint to third parties to prevent cross-user data leakage.


