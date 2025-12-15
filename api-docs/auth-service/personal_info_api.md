# Personal Info API Guide

## Summary
- `GET /api/personal-info` returns the authenticated user’s personal profile payload (or an empty array) using `PersonalInfoController@show`.
- `PUT/PATCH /api/personal-info` upserts the caller’s personal profile through `PersonalInfoController@update`, delegating validation to `UpdatePersonalInfoRequest`.
- Both endpoints inherit the `auth:sanctum`, `verified`, and `activity` middleware stack from the API v1 group, so a verified authenticated session is mandatory.
- Data persists in the `PersonalInfo` model, which casts the `passions` attribute to an associative boolean array with sensible defaults.

## Route Registry
| Method | Path | Middleware | Controller action | Notes |
| --- | --- | --- | --- | --- |
| GET | `/api/personal-info` | `auth:sanctum`, `verified`, `activity` | `PersonalInfoController@show` | Returns the caller’s stored personal info or an empty list when none exists. |
| PUT/PATCH | `/api/personal-info` | `auth:sanctum`, `verified`, `activity` | `PersonalInfoController@update` | Creates or updates the caller’s personal info record. |

The endpoints are registered via `Route::apiSingleton('personal-info', PersonalInfoController::class);` inside the authenticated API v1 group defined in `routes/api.php`.

## Response Shape
`PersonalInfoController@show` reads the authenticated user’s related `personalInfo` record and responds with a `data` envelope:

```20:35:app/Http/Controllers/Api/V1/PersonalInfoController.php
return response()->json([
    'data' => is_null($personalInfo) ? [] : [
        'occupation' => $personalInfo->occupation,
        'education' => $personalInfo->education,
        'memory' => $personalInfo->memory,
        'loved_city' => $personalInfo->loved_city,
        'loved_country' => $personalInfo->loved_country,
        'loved_language' => $personalInfo->loved_language,
        'problem_solving' => $personalInfo->problem_solving,
        'prediction' => $personalInfo->prediction,
        'about' => $personalInfo->about,
        'passions' => $personalInfo->passions,
    ]
]);
```

- When no record exists, an empty array is returned for `data` to simplify client handling.
- Stored `passions` are exposed as a keyed map of booleans (e.g., `passions.music`, `passions.art`).

## Validation Rules
`PersonalInfoController@update` uses `UpdatePersonalInfoRequest`, which only allows nullable string fields and a boolean map for passions:

```24:51:app/Http/Requests/UpdatePersonalInfoRequest.php
return [
    'occupation' => 'nullable|string|max:255',
    'education' => 'nullable|string|max:255',
    'memory' => 'nullable|string|max:2000',
    'loved_city' => 'nullable|string|max:255',
    'loved_country' => 'nullable|string|max:255',
    'loved_language' => 'nullable|string|max:255',
    'problem_solving' => 'nullable|string|max:2000',
    'prediction' => 'nullable|string|max:10000',
    'about' => 'nullable|string|max:10000',
    'passions' => 'nullable|array',
    'passions.music' => 'nullable|boolean',
    'passions.sport_health' => 'nullable|boolean',
    'passions.art' => 'nullable|boolean',
    'passions.language_culture' => 'nullable|boolean',
    'passions.philosophy' => 'nullable|boolean',
    'passions.animals_nature' => 'nullable|boolean',
    'passions.aliens' => 'nullable|boolean',
    'passions.food_cooking' => 'nullable|boolean',
    'passions.travel_leature' => 'nullable|boolean',
    'passions.manufacturing' => 'nullable|boolean',
    'passions.science_technology' => 'nullable|boolean',
    'passions.space_time' => 'nullable|boolean',
    'passions.history' => 'nullable|boolean',
    'passions.politics_economy' => 'nullable|boolean',
];
```

- Every field is optional, enabling partial profile updates. Omitting a passion key leaves its current value unchanged.
- String fields have explicit length caps; beware of truncation if clients attempt to exceed them.
- Validation failures respond with Laravel’s standard `422` JSON structure.

### Model Defaults
The `PersonalInfo` model predefines all passion flags as `false` and casts the column to JSON:

```18:63:app/Models/User/PersonalInfo.php
protected $fillable = [
    'user_id', 'occupation', 'education', 'memory', 'loved_city', 'loved_country',
    'loved_language', 'problem_solving', 'prediction', 'about', 'passions',
];

protected $casts = [
    'passions' => 'array',
];

protected $attributes = [
    'passions' => '{
        "music": false,
        "sport_health": false,
        "art": false,
        "language_culture": false,
        "philosophy": false,
        "animals_nature": false,
        "aliens": false,
        "food_cooking": false,
        "travel_leature": false,
        "manufacturing": false,
        "science_technology": false,
        "space_time": false,
        "history": false,
        "politics_economy": false,
    }',
];
```

- When a record is first created, all passion categories default to `false` until explicitly set by the client.

## Policy & Authorization
- **Request-level authorization:** `UpdatePersonalInfoRequest::authorize()` returns `true`, relying on the parent middleware (`auth:sanctum`, `verified`, `activity`) to guarantee the caller is the owner of the personal-info record.
- **Controller safeguards:** `PersonalInfoController@show` and `@update` always scope operations to `request()->user()`, preventing cross-user access without needing an explicit policy class.
- **Implicit policy contract:** There is no dedicated `PersonalInfoPolicy`. Ownership is enforced by using the authenticated user’s ID during `PersonalInfo::updateOrCreate` and via the relationship call in `show`.

## Endpoint Behaviour
### `GET /api/personal-info`
- Returns `200 OK` with `{ "data": [...] }` containing the profile fields when present.
- Returns `200 OK` with `{ "data": [] }` when no record exists yet.
- Errors: `401 Unauthorized` (missing/invalid token), `403 Forbidden` (unverified/inactive session blocked by middleware), `500` on unexpected exceptions.

### `PUT/PATCH /api/personal-info`
- **Body parameters (JSON recommended):** Accepts any subset of the fields covered by the validation rules.
- **Behaviour:** Upserts the record via `PersonalInfo::updateOrCreate`, keyed by the authenticated user ID, and returns `204 No Content` with an empty JSON body on success.
- **Error modes:** `401` on authentication failure, `403` if middleware conditions fail, `422` on validation errors, `500` otherwise.

## Usage Examples
```bash
curl -X GET https://example.com/api/personal-info \
  -H "Authorization: Bearer <token>"
```

```bash
curl -X PATCH https://example.com/api/personal-info \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
        "occupation": "Software Engineer",
        "passions": {
          "science_technology": true,
          "art": true
        }
      }'
```

## Operational Notes
- Clients should treat the resource as a singleton; the server always targets the authenticated user, so no `user_id` needs to be supplied.
- Consider fetching the latest profile immediately after a successful `204` update, as the response body is empty.
- To clear a field, send it as an empty string (`""`) to pass validation and persist null, or omit it to leave the stored value unchanged.
- Expand the `passions` map with care—add new keys to both the validation rules and the model’s default JSON block to maintain consistency.

