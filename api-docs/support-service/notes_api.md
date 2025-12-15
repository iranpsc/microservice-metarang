# Notes API Guide

## Summary
- Full CRUD for personal notes is exposed under `/api/notes`, guarded by the `auth:sanctum`, `verified`, and `activity` middleware stack.
- All create and update operations funnel through `NoteRequest`, enforcing required `title`/`content` fields and file-type limits on optional attachments.
- Responses are serialized with `NoteResource`, returning localized Jalali timestamps (`date`, `time`) and an attachment URL when present.
- No dedicated `NotePolicy` exists today; ownership is inferred by scoping queries to `request()->user()`, leaving cross-user access unguarded.

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| GET | `/api/notes` | `auth:sanctum`, `verified`, `activity` | `NoteController@index` | List the authenticated user’s notes. |
| POST | `/api/notes` | `auth:sanctum`, `verified`, `activity` | `NoteController@store` | Create a new note tied to the caller. |
| GET | `/api/notes/{note}` | `auth:sanctum`, `verified`, `activity` | `NoteController@show` | Fetch a specific note instance by route-model binding. |
| PUT / PATCH | `/api/notes/{note}` | `auth:sanctum`, `verified`, `activity` | `NoteController@update` | Overwrite a note’s title, content, and optional attachment. |
| DELETE | `/api/notes/{note}` | `auth:sanctum`, `verified`, `activity` | `NoteController@destroy` | Soft-delete the note record (currently a hard delete). |

Registrations live inside the authenticated group in `routes/api.php` (`Route::apiResources(['notes' => NoteController::class, ...]);`). Sanctum authentication plus email/phone verification and the activity tracker middleware are prerequisites for every call.

## Request Validation
`NoteRequest` (`app/Http/Requests/NoteRequest.php`) enforces:
- `title`: required string, max 130 characters.
- `content`: required string, max 2000 characters.
- `attachment`: optional upload; must be `png`, `jpg`, `jpeg`, or `pdf` and no larger than 5 MB.

Attachments are stored via `$request->file('attachment')->store('notes')`, producing a path like `notes/<file>`. The controller currently prefixes this with `url('uploads/' . $path)` before persistence, so the raw database column contains a full URL.

## Response Shape
`NoteResource` (`app/Http/Resources/NoteResource.php`) emits the following fields:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | integer | Primary key of the note. |
| `title` | string | Validated user-provided title. |
| `content` | string | Body content, up to 2000 characters. |
| `attachment` | string or `null` | When the database column is non-null, `NoteResource` wraps it with `url('uploads/' . $this->attachment)`; given the stored value is already a URL, consumers currently receive a double-prefixed URL (bug). |
| `date` | string | `updated_at` formatted with `jdate(...)->format('Y/m/d')` (Jalali calendar). |
| `time` | string | `updated_at` formatted with `jdate(...)->format('H:m:s')`. |

Collections are returned under the standard Laravel `data` wrapper with pagination disabled (the controller returns an in-memory collection).

## Policy Rules & Safeguards
- **Middleware gates:** All endpoints require a valid Sanctum bearer token, a verified account (`verified` middleware), and pass activity tracking (`activity` middleware).
- **Ownership scope:** `NoteController@index` queries `request()->user()->notes`, so list results are limited to the caller’s own records.
- **Missing authorization policy:** There is no `NotePolicy` registered in `AuthServiceProvider`, and controller actions do not call `$this->authorize()`. Because route-model binding resolves any `Note` by ID, an authenticated user can access, update, or delete another user’s note if they know the ID. Consider adding a `NotePolicy` with `view`, `update`, and `delete` checks to enforce ownership.
- **Attachment handling:** Files are stored under `storage/app/notes` via Laravel’s filesystem. Ensure public disk exposure is properly configured and sanitized to avoid arbitrary file uploads.

## Endpoint Details
### `GET /api/notes`
- Returns all notes for the authenticated user as a `NoteResource` collection.
- No pagination or filtering; clients should handle client-side sorting or reduction.

### `POST /api/notes`
- Accepts `title`, `content`, and optional `attachment` multipart upload.
- On success, returns `201 Created` with the persisted resource payload.
- Failures: `422 Unprocessable Entity` for validation errors, `401`/`403` for authentication failures, `500` for unexpected exceptions.

### `GET /api/notes/{note}`
- Route-model binding resolves `{note}` by primary key.
- Currently no authorization guard ensures the bound note belongs to the caller (see Policy section).
- Success returns `200 OK` with a `NoteResource` payload; missing IDs yield `404 Not Found`.

### `PUT/PATCH /api/notes/{note}`
- Applies the same validation as `store`.
- Replaces the note’s `title`, `content`, and optionally the attachment (new uploads overwrite the previous value; empty uploads leave the attachment as an empty string).

### `DELETE /api/notes/{note}`
- Calls `$note->delete()`, permanently removing the record.
- Responds with `204 No Content` when the deletion succeeds.

## Usage Examples
```bash
curl -X POST https://example.com/api/notes \
  -H "Authorization: Bearer <token>" \
  -F "title=Personal reminder" \
  -F "content=Review quarterly results" \
  -F "attachment=@/path/to/file.pdf"
```

```bash
curl -X GET https://example.com/api/notes/42 \
  -H "Authorization: Bearer <token>"
```

## Operational Notes
- Add a `NotePolicy` to enforce that `show`, `update`, and `destroy` actions are restricted to the owner.
- Consider paginating `index` to prevent large payloads when users accumulate many notes.
- The double URL prefix in `NoteResource` should be corrected to avoid malformed attachment links.
- If attachments reside on disk, schedule periodic cleanup when notes are deleted or attachments replaced.


