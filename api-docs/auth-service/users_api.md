# Users API Guide

## Summary
- The users suite exposes six read-only endpoints under `/api/users` that surface public profile lists, individual profile snapshots, wallet balances, level progress, profile limitation records, and feature inventory totals.
- Responses are built via dedicated Laravel API resources (`UserResource`, `ProfileResource`, `WalletResource`, `ProfileLimitationResource`) to enforce formatting, privacy filters, and URL composition.
- Pagination, sorting, and search are handled server-side for the collection endpoint, while per-user endpoints leverage implicit route model binding (`{user}`) against the primary key.

## Environment & Dependencies
- `config('app.admin_panel_url')` must be set; it is used to prepend media URLs (level badges, profile images).
- The `check.profile.limitation` middleware must be registered; it injects the `profileLimitation` attribute consumed by `ProfileResource` privacy filters.
- Helper functions `formatCompactNumber()` and `jdate()` are expected to be globally available for wallet number formatting and Jalali date presentation.
- User relationships relied upon by the controller: `levels.image`, `latestProfilePhoto`, `kyc`, `wallet`, `followers`, `following`, `settings`, and `profilePhotos`.

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| GET | `/api/users` | `api` (explicitly skips `auth:sanctum`, `verified`) | `index` | Return a paginated list of public users with summary info. |
| GET | `/api/users/{user}/levels` | `api` | `getLevel` | Expose the user's latest level plus historical ladder data. |
| GET | `/api/users/{user}/profile` | `api`, `check.profile.limitation` | `getProfile` | Return the user's profile, honoring privacy and limitation rules. |
| GET | `/api/users/{user}/wallet` | `api` | `getWallet` | Return the user's wallet balances in compact format. |
| GET | `/api/users/{user}/profile-limitations` | `api`, `auth:sanctum` | `getProfileLimitations` | Show mutual profile limitation entry between the caller and the user. |
| GET | `/api/users/{user}/features/count` | `api` | `getFeaturesCount` | Return categorized counts of the user's real-estate features. |

> **Path parameter notes:** `{user}` resolves via implicit route model binding on the `App\Models\User` primary key (`id`). Passing a non-existent identifier yields a 404 JSON response from Laravel.

## Endpoint Behaviour

### `GET /api/users`
- **Query parameters**
  - `search` (string, optional): partial match on the `name` column.
  - `order-by` (string, optional): one of `score`, `registered_at_asc`, `registered_at_desc`. Defaults to descending score.
  - `page` (integer, optional): simple pagination cursor (`simplePaginate(20)`).
- **Processing**
  1. Filters out the reserved admin user (`code` = `hm-2000000`).
  2. Loads level thumbnails (`levels.image`), the latest profile photo, and verified KYC name data.
  3. Sorts based on the supplied `order-by` option (falls back to score).
- **Response**
  - Returns a `200 OK` JSON body containing a `data` array of `UserResource` objects plus Laravel's simple pagination keys (`links`, `meta`).
  - Each user entry exposes `id`, `name` (prefers KYC full name when available), `code`, `score`, `levels.current`, `levels.previous`, and `profile_photo`.
- **Failure modes**
  - Invalid `order-by` values are ignored (no validation); the request still succeeds.
  - No results yield an empty `data` array with standard pagination metadata.

### `GET /api/users/{user}/levels`
- **Processing**
  1. Checks whether the user has a `latest_level`. If absent, returns a zeroed structure.
  2. Retrieves all `Level` records with lower scores to build the `previous_levels` ladder.
  3. Computes the numeric progress toward the next level via `Level::getScorePercentageToNextLevel($user)`.
- **Response**
  - Success returns `200 OK` with a JSON payload:
    ```json
    {
      "data": {
        "latest_level": {
          "id": 1,
          "name": "Beginner",
          "score": 100,
          "slug": "beginner",
          "image": "https://admin.example.com/uploads/<path>"
        },
        "previous_levels": [ /* ordered ascending by score */ ],
        "score_percentage_to_next_level": 42.5
      }
    }
    ```
  - If `latest_level` is missing, `latest_level` is `null` and `previous_levels` is an empty array.
- **Failure modes**
  - `404` if the user ID is invalid.

### `GET /api/users/{user}/profile`
- **Middleware**: `check.profile.limitation` ensures request attributes include any active limitation record between the caller and the target user.
- **Processing**
  1. Loads `settings` (privacy map), verified `kyc` identity, profile photos, and follower/following counts.
  2. Delegates privacy filtering to `ProfileResource::filterField`, which hides fields unless the viewer is the owner or the corresponding privacy flag equals `1`.
  3. When authenticated, limitation data on the request can further adjust downstream privacy (middleware responsibility).
- **Response**
  - Returns `200 OK` with a `ProfileResource` body containing:
    - `id`, `name`, `code`, `registered_at` (Jalali `Y/m/d` format), `profile_images` (array of URLs), `followers_count`, `following_count`.
    - Fields may be `null` if privacy settings disallow disclosure.
- **Failure modes**
  - `404` for unknown user.
  - The middleware may short-circuit with its own status codes (e.g., 423 Locked) if limitations block access; consult middleware implementation.

### `GET /api/users/{user}/wallet`
- **Processing**
  1. Retrieves the related `wallet` record for the user.
  2. Formats numeric balances using `formatCompactNumber()` and normalizes satisfaction to one decimal place.
- **Response**
  - `200 OK` with a `WalletResource` payload:
    ```json
    {
      "data": {
        "psc": "12.4K",
        "irr": "3.1M",
        "red": "532",
        "blue": "1.6K",
        "yellow": "713",
        "satisfaction": "87.5",
        "effect": 12
      }
    }
    ```
  - Missing wallet relationships trigger Laravel's default `null` to resource conversion, which surfaces as an empty object; ensure the wallet relation is enforced at the database level.
- **Failure modes**
  - `404` for invalid user.

### `GET /api/users/{user}/profile-limitations`
- **Middleware**: Requires a valid Sanctum bearer token (`auth:sanctum`).
- **Processing**
  1. Searches `ProfileLimitation` for a record where the caller limits the target or the target limits the caller.
  2. Returns the first match; the logic is symmetric to capture mutual limitations.
- **Response**
  - If a limitation exists, the `ProfileLimitationResource` exposes `id`, `limiter_user_id`, `limited_user_id`, `options` (JSON structure of toggles), and `note` (only when the caller is the limiter).
  - Without a matching record, the controller responds with `{"data": []}`.
- **Failure modes**
  - `401 Unauthenticated` when the Sanctum token is missing or invalid.
  - `404` for invalid user.

### `GET /api/users/{user}/features/count`
- **Processing**
  1. Executes three `loadCount` subqueries against the `features` relation, filtering by property usage (`karbari`) codes:
     - `m` → `maskoni_features_count`
     - `t` → `tejari_features_count`
     - `a` → `amoozeshi_features_count`
  2. Returns the computed counts without exposing raw feature data.
- **Response**
  - `200 OK` with:
    ```json
    {
      "data": {
        "maskoni_features_count": 5,
        "tejari_features_count": 2,
        "amoozeshi_features_count": 0
      }
    }
    ```
- **Failure modes**
  - `404` for invalid user.

## Shared Considerations
- **Implicit serialization**: single-resource endpoints return plain JSON via `response()->json()` or wrapped resources. Expect `data` root keys unless an empty array is returned.
- **Caching**: no caching is implemented; repeated calls hit the database each time.
- **Rate limiting**: controllers rely on the default `api` throttle (`throttle:api`), typically 60 requests/minute unless configured otherwise.
- **Localization**: Profile dates use `jdate()` for Jalali formatting; ensure the helper is globally available and localized appropriately.

## Extending the Users API
- Add additional filters (e.g., `level`, `verified`) to the `index` query by chaining further `when()` clauses.
- If client apps need detailed feature breakdowns, consider adding dedicated endpoints or expanding `getFeaturesCount` with more categories.
- Wrap resource responses in custom transformers or append metadata headers (e.g., `X-Total-Count`) if front-ends need aggregate stats without secondary queries.
- Introduce request validation for query parameters (form requests or `request()->validate`) to provide explicit error messages for unsupported `order-by` values.


