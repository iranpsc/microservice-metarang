# Profile Limitations API Guide

## Summary
- Three authenticated REST endpoints live under `/api/profile-limitations` (POST, PUT, DELETE) for managing visibility or interaction restrictions between two users.
- All mutations are guarded by `auth:sanctum` plus `ProfileLimitationPolicy`, which enforces one active record per limiter/limited pair and restricts updates/deletions to the limiter.
- Responses use `ProfileLimitationResource` so the limiter sees both the options map and optional note while the limited user only receives structural data.
- Consumers can retrieve the active record between the caller and another user through `GET /api/users/{user}/profile-limitations`, which reuses the same resource payload.

## Environment & Dependencies
- Sanctum must be configured; `ProfileLimitationController` applies the `auth:sanctum` middleware in its constructor.
- Authorization mapping is registered via `ProfileLimitationController::__construct()` calling `$this->authorizeResource(ProfileLimitation::class, 'profileLimitation')`; Laravel will resolve `ProfileLimitationPolicy`.
- Eloquent model `App\Models\ProfileLimitation` casts the `options` column to an array and seeds defaults with every flag enabled.
- Downstream privacy logic (e.g., `check.profile.limitation` middleware) expects the `options` payload to contain six boolean keys.

## Options Schema
| Key | Description | Default in DB | Typical Impact |
| --- | --- | --- | --- |
| `follow` | Whether the limited user can follow the limiter. | `true` | Disabled follow button, unfollows existing relation. |
| `send_message` | Permission to initiate direct messages. | `true` | Messaging UI hides composer. |
| `share` | Ability to share limiter’s content. | `true` | Blocks share CTA and share API calls. |
| `send_ticket` | Permission to open support tickets toward the limiter. | `true` | Prevents ticket creation targeting the limiter. |
| `view_profile_images` | Visibility of profile photo gallery. | `true` | Gallery component checks this flag before rendering. |
| `view_features_locations` | Visibility into feature (listing) map coordinates. | `true` | Map layers turn off if set to `false`. |

All options must be submitted as booleans in the request payload. Missing keys trigger `422 Unprocessable Entity`.

## Route Registry
| Method | Path | Middleware | Controller Action | Purpose |
| --- | --- | --- | --- | --- |
| POST | `/api/profile-limitations` | `api`, `auth:sanctum` | `ProfileLimitationController@store` | Create a new limitation record owned by the caller. |
| PUT | `/api/profile-limitations/{profileLimitation}` | `api`, `auth:sanctum` | `ProfileLimitationController@update` | Update options or note on an existing record. |
| DELETE | `/api/profile-limitations/{profileLimitation}` | `api`, `auth:sanctum` | `ProfileLimitationController@destroy` | Remove an existing record entirely. |
| GET | `/api/users/{user}/profile-limitations` | `api`, `auth:sanctum` | `UserController@getProfileLimitations` | Fetch the mutual limitation (if any) between caller and the specified user. |

> **Route parameters:** `{profileLimitation}` leverages implicit binding on `App\Models\ProfileLimitation`. The policy ensures the bound model belongs to the caller. `{user}` binds to `App\Models\User`.

## Endpoint Behaviour

### `POST /api/profile-limitations`
- **Body schema**
  ```json
  {
    "limited_user_id": 1234,
    "options": {
      "follow": false,
      "send_message": false,
      "share": true,
      "send_ticket": true,
      "view_profile_images": false,
      "view_features_locations": true
    },
    "note": "Temporarily restricting interactions until project completion."
  }
  ```
- **Validation**
  - `limited_user_id` must reference an existing `users.id`.
  - `options` must be an array containing exactly the six allowed keys; each value must be boolean.
  - `note` is optional text up to 500 characters.
- **Authorization**
  - `ProfileLimitationPolicy@create` denies requests if a record already exists for the limiter/limited pair, returning `403`.
- **Response**
  - `201 Created` with a `ProfileLimitationResource` body:
    ```json
    {
      "data": {
        "id": 7,
        "limiter_user_id": 42,
        "limited_user_id": 1234,
        "options": {
          "follow": false,
          "send_message": false,
          "share": true,
          "send_ticket": true,
          "view_profile_images": false,
          "view_features_locations": true
        },
        "note": "Temporarily restricting interactions until project completion."
      }
    }
    ```
- **Failure cases**
  - `422` for validation errors (missing keys, non-boolean values, oversized note).
  - `403` if the caller already has a limitation for `limited_user_id`.
  - `404` if `limited_user_id` fails binding.

### `PUT /api/profile-limitations/{profileLimitation}`
- **Body schema**
  ```json
  {
    "options": {
      "follow": true,
      "send_message": false,
      "share": true,
      "send_ticket": false,
      "view_profile_images": false,
      "view_features_locations": false
    },
    "note": "Allowing follow again; keeping DMs off."
  }
  ```
- **Validation**
  - Same rules as `store` for `options` and `note`.
  - Route binding plus `ProfileLimitationPolicy@update` ensure the caller is the limiter; otherwise Laravel returns `403`.
- **Response**
  - `200 OK` with the updated `ProfileLimitationResource`. The `note` field is included only when the caller is the limiter (`auth()->id() === limiter_user_id`).
- **Failure cases**
  - `403` if the authenticated user does not own the record.
  - `422` for payload validation failures.

### `DELETE /api/profile-limitations/{profileLimitation}`
- **Processing**
  - Route model binding resolves the record; `ProfileLimitationPolicy@delete` verifies ownership.
  - Upon success, the model is deleted permanently.
- **Response**
  - `204 No Content` with an empty body.
- **Failure cases**
  - `403` if the caller is not the limiter.
  - `404` if the ID does not resolve.

### `GET /api/users/{user}/profile-limitations`
- **Purpose**
  - Returns the shared `ProfileLimitationResource` for the caller and the specified user, regardless of who set it.
  - If no record exists, the response is `200 OK` with `{ "data": [] }`.
- **Output nuances**
  - When the caller is the limited user, `note` is omitted by `ProfileLimitationResource::toArray`.
  - Only one record can exist per relationship pair; the controller queries both directions and returns the first hit.

## Error Handling & Status Codes
- Laravel’s validator returns field-specific error messages under a `422` response.
- Authorization failures throw `AuthorizationException`, surfaced as `403` JSON with `message` describing the denial.
- Missing resources (invalid IDs) produce `404` JSON responses aligned with Laravel’s default API exception handler.

## Testing Tips
- Use Sanctum tokens or session-authenticated requests; anonymous calls receive `401 Unauthorized`.
- Attempt a duplicate POST with the same `limited_user_id` to confirm the `403` guard works.
- Verify the note visibility rule by hitting the GET endpoint both as the limiter and as the limited user.
- Deleting a record should immediately allow re-creating it for the same pair, demonstrating the policy constraint reset.

