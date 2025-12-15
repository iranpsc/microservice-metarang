# Follow API Guide

## Summary
- `GET /api/followers` and `GET /api/following` return the authenticated user’s follower and following lists, serialized through `FollowResource`.
- `GET /api/follow/{user}` creates a follow relationship after passing the `UserPolicy@follow` gate and fires the `User::followed` model event on the target profile.
- `GET /api/unfollow/{user}` detaches an existing follow relationship; `GET /api/remove/{user}` removes the target from the caller’s followers.
- All routes sit behind `auth:sanctum`, `verified`, and `activity` middleware, so only signed-in, verified, active users may call them.

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| GET | `/api/followers` | `auth:sanctum`, `verified`, `activity` | `FollowController@followers` | List profiles that follow the caller. |
| GET | `/api/following` | `auth:sanctum`, `verified`, `activity` | `FollowController@followings` | List profiles the caller follows. |
| GET | `/api/follow/{user}` | `auth:sanctum`, `verified`, `activity` | `FollowController@follow` | Start following the given user (route-model binding on `user`). |
| GET | `/api/unfollow/{user}` | `auth:sanctum`, `verified`, `activity` | `FollowController@unfollow` | Stop following the given user. |
| GET | `/api/remove/{user}` | `auth:sanctum`, `verified`, `activity` | `FollowController@remove` | Remove the given user from the caller’s followers. |

The routes are registered in `routes/api.php` inside the authenticated grouping. Although follow state changes typically use `POST`/`DELETE`, the current implementation accepts `GET` and responds with an empty `200 OK` body.

## Response Shape
Follower and following collections are wrapped with `FollowResource`:

| Field | Type | Notes |
| --- | --- | --- |
| `id` | integer | The user’s primary key (`users.id`). |
| `name` | string | Display name. |
| `code` | string | Public identifier exposed to clients. |
| `profile_photos` | string or array | Latest profile photo URL or empty array when missing. |
| `level` | string | Latest level slug; empty string when unset. |
| `online` | boolean | `true` when the user is online (via `User::isOnline()`). |

Collections are returned as JSON arrays under the top-level `data` key per Laravel resource conventions.

## Policy Rules & Safeguards
- **Authorization check:** `FollowController@follow` calls `authorize('follow', $user)`, delegating to `UserPolicy::follow`. The policy denies the request when:
  - The caller tries to follow themselves.
  - The caller already has an active follow relationship.
  - The target has an active `ProfileLimitation` record with `options['follow'] === false` either specifically against the caller (`limited_user_id = caller.id`) or globally (`limited_user_id = target.id`).
- **Middleware gates:** All endpoints require the Sanctum bearer token, a verified email/phone (per the `verified` middleware), and pass the `activity` middleware (tracks user activity).
- **Event hook:** Successful follow actions invoke `User::followed()`, firing any listeners subscribed to the `followed` model event (e.g., notifications or engagement metrics).

## Endpoint Details
### `GET /api/followers`
- Returns the authenticated user’s followers.
- No query parameters are supported today; consumer-side filtering/pagination must be implemented client-side.

### `GET /api/following`
- Returns the authenticated user’s outbound follow relationships.
- Uses the same serialization as `/api/followers`.

### `GET /api/follow/{user}`
- **Path parameter:** `{user}` leverages implicit binding on the `User` model (`id` by default; customize with `getRouteKeyName` if needed).
- **Behavior:** Attaches the target to `$request->user()->following()`, persists the pivot record in the `follows` table, dispatches the `followed` event on the target, and responds with HTTP `200` and an empty body.
- **Error modes:**
  - `401 Unauthorized` – Missing/invalid Sanctum token.
  - `403 Forbidden` – Policy rejection (self-follow, already following, profile limitation).
  - `404 Not Found` – Target user not found or route binding fails.
  - `500` – Unexpected server exceptions.

### `GET /api/unfollow/{user}`
- **Behavior:** Detaches the pivot record from `$request->user()->following()`.
- **Responses:** Always returns `200` with no body, even when the relationship did not previously exist.
- **Error modes:** Same as `/api/follow/{user}`, minus the policy gate (no `authorize` call).

### `GET /api/remove/{user}`
- **Behavior:** Removes the target from `$request->user()->followers()`, effectively forcing an existing follower to stop following the caller.
- **Responses:** `200` with no body.
- **Error modes:** `401`, `404`, or `500` as described above.

## Usage Examples
```bash
curl -X GET https://example.com/api/followers \
  -H "Authorization: Bearer <token>"
```

```bash
curl -X GET https://example.com/api/follow/123 \
  -H "Authorization: Bearer <token>"
```

Both commands return `HTTP/1.1 200 OK` with empty bodies for write operations and JSON collections for list operations.

## Operational Notes
- Because mutating actions are exposed as `GET` requests, ensure CSRF protections or API gateway rules mitigate cross-site request forgery and caching side-effects (e.g., set `Cache-Control: no-store`).
- Consider adding pagination or lightweight filters to the listing endpoints if follower counts are expected to grow large.
- Add rate limiting or activity auditing around follow/unfollow actions to detect abuse or automation.


