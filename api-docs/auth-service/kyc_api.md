# KYC API Guide

## Summary
- `GET /api/kyc` returns the authenticated user’s KYC record via `KycResource` when `KycPolicy::view` authorises access; otherwise it yields an empty JSON object.
- `PUT/PATCH /api/kyc` (same payload for both verbs) creates the caller’s first KYC record or resubmits a rejected one after passing `UpdateKycRequest` validation and `KycPolicy::update`.
- All routes live inside the `auth:sanctum`, `verified`, and `activity` middleware group, so a valid Sanctum token, verified account, and active session are required.
- File uploads (`melli_card` image, `video` artefact) are moved into the public `storage/app/public/kyc` disk; successful writes surface publicly resolvable URLs under `/uploads/kyc/...`.

## Route Registry
| Method | Path | Middleware | Controller action | Notes |
| --- | --- | --- | --- | --- |
| GET | `/api/kyc` | `auth:sanctum`, `verified`, `activity` | `KycController@show` | Returns the caller’s KYC record when it exists and policy permits viewing; otherwise returns `{}`. |
| PUT/PATCH | `/api/kyc` | `auth:sanctum`, `verified`, `activity` | `KycController@update` | Validates & (re)submits KYC data; auto-creates the record on first submission. |

Routes are registered in `routes/api.php` via `Route::apiSingleton('kyc', KycController::class);`, inheriting the `/api` prefix and shared middleware stack defined for API v1.

## Resource Schema
Responses wrap the model with `KycResource`, exposing the following structure:

```17:29:app/Http/Resources/KycResource.php
return [
    'id' => (string)$this->id,
    'melli_card' => $this->melli_card,
    'fname' => $this->fname,
    'lname' => $this->lname,
    'melli_code' => $this->melli_code,
    'birthdate' => jdate($this->birthdate)->format('Y/m/d'),
    'province' => $this->province,
    'status' => $this->status,
    'video' => $this->video,
    'errors' => $this->whenNotNull($this->errors),
    'gender' => $this->gender,
];
```

- `status` is an integer lifecycle flag: `-1` = rejected, `0` = pending (default), `1` = approved. Helper methods on `Kyc` (`rejected()`, `pending()`, `approved()`) make these states explicit.
- `errors` is populated by back-office reviewers to return rejection feedback; it is automatically cleared on every resubmission.
- `birthdate` is delivered in Jalali format, even though the record persists it as a Gregorian `Y-m-d` date.

Collections are not applicable because the resource is a singleton; responses are a single object.

## Validation Rules
`KycController@update` delegates validation and authorization to `UpdateKycRequest`:

```31:47:app/Http/Requests/UpdateKycRequest.php
return [
    'fname' => 'required|string|min:2|max:255',
    'lname' => 'required|string|min:2|max:255',
    'melli_code' => [
        'required',
        'ir_national_code',
        Rule::unique('kycs', 'melli_code')->ignore($this->user()->id, 'user_id'),
    ],
    'birthdate' => 'required|shamsi_date',
    'province' => 'required|string|max:255',
    'melli_card' => 'required|image|max:5000',
    'video' => 'required|array',
    'verify_text_id' => 'required|integer|exists:kyc_verify_texts,id',
    'gender' => 'required|string|in:male,female,other',
];
```

Additional request preparation enforces:

- `birthdate` is normalised from Jalali to Gregorian (`jalali_to_carbon(...)->format('Y-m-d')`) before persistence.
- `status` is reset to `0` (pending review) and `errors` cleared to `null` on every update, ensuring back-office moderation starts from a clean slate.

### File payload expectations
- `melli_card` must be provided as an uploaded image (`multipart/form-data`), capped at ~5 MB (`max:5000` kilobytes) and stored under `public/kyc`.
- `video` must be a JSON object containing at least `path` and `name`. The controller relocates the temporary file at `storage/app/<path>/<name>` into `storage/app/public/kyc/<name>` and exposes it at `/uploads/kyc/<name>`.

### Validation failures
- Respond with `422 Unprocessable Entity` following Laravel’s standard error envelope (`{ "message": "...", "errors": { field: [...] } }`).
- `ir_national_code` and `shamsi_date` are custom validators that enforce Iranian national code format and Shamsi date validity, respectively; clients should send Jalali dates (e.g., `1403/01/15`) for `birthdate`.

## Policy & Authorization
Authorisation is handled through `KycPolicy` and `UpdateKycRequest::authorize()`:

| Ability | Policy method | Rule |
| --- | --- | --- |
| `view` | `KycPolicy::view` | Only the owner can view their KYC record. |
| `update` | `KycPolicy::update` | The owner can update only when the existing record is rejected (`status === -1`). |

`UpdateKycRequest::authorize()` short-circuits to `true` when the user has no KYC record yet, allowing the first submission; otherwise it calls `can('update', $kyc)`, enforcing the rejection prerequisite.

The `show` endpoint manually checks `request()->user()->can('view', $kyc)` and returns an empty JSON response when the policy denies access, avoiding a `403` but signalling the absence or inaccessibility of the record.

## Endpoint Details
### `GET /api/kyc`
- Returns `200 OK` with the serialised KYC resource when one exists and the caller owns it.
- Returns `200 OK` with `{}` when the user has never submitted KYC or the policy denies access (e.g., attempting to view somebody else’s record).
- Errors: `401 Unauthorized` (invalid/missing Sanctum token), `403 Forbidden` rarely surfaces because the controller masks it, `500` for unexpected server issues.

### `PUT/PATCH /api/kyc`
- **Body parameters (multipart/form-data recommended):**
  - `fname`, `lname`: strings, 2–255 characters.
  - `melli_code`: Iranian national code, unique across KYC records.
  - `birthdate`: Jalali date string (`YYYY/MM/DD`), converted server-side.
  - `province`: province name, max 255 characters.
  - `melli_card`: required file upload; image type.
  - `video[path]`, `video[name]`: identify a previously uploaded temp file to promote into public storage.
  - `verify_text_id`: integer referencing `kyc_verify_texts.id`.
  - `gender`: one of `male`, `female`, `other`.
- **Behaviour:**
  - `Kyc::updateOrCreate()` ensures idempotent submissions—first call creates the record; subsequent approved submissions are blocked by policy unless the record is rejected.
  - Successful updates refresh the resource and return `200 OK` with the latest values (including public URLs for uploaded assets).
- **Error modes:**
  - `401` when unauthenticated.
  - `403` when policy denies resubmission (e.g., trying to update while status is pending/approved).
  - `422` on validation errors (missing file, invalid date, duplicate national code, etc.).

## Usage Examples
```bash
curl -X GET https://example.com/api/kyc \
  -H "Authorization: Bearer <token>"
```

```bash
curl -X PUT https://example.com/api/kyc \
  -H "Authorization: Bearer <token>" \
  -F "fname=Ali" \
  -F "lname=Karimi" \
  -F "melli_code=1234567890" \
  -F "birthdate=1403/01/15" \
  -F "province=Tehran" \
  -F "gender=male" \
  -F "verify_text_id=2" \
  -F "melli_card=@/path/to/melli-card.jpg" \
  -F "video[path]=tmp/uploads" \
  -F "video[name]=selfie-123.mp4"
```

## Operational Notes
- The controller relies on a prior upload pipeline that stages videos under `storage/app/<path>/<name>`; ensure the frontend honours that contract before calling the update endpoint.
- Consider surfacing the `status` and `errors` fields prominently in clients so users know when a resubmission is required.
- `status` resets to pending on every update; back-office tooling should set it to `1` (approved) or `-1` (rejected) after review.
- When extending the policy (e.g., permitting updates during pending review), adjust both `KycPolicy::update` and `UpdateKycRequest::authorize()` to keep behaviour consistent.


