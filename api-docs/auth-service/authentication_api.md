# Authentication API Guide

## Summary
- The authentication suite exposes five endpoints under `/api/auth` that orchestrate an OAuth 2.0 Authorization Code flow against the external SSO server configured via the `OAUTH_*` environment variables.
- Laravel Sanctum issues the application session token after the OAuth callback; the OAuth access and refresh tokens are stored directly on the `users` table for later reuse by other services.
- Short-lived cache entries bridge the browser redirect journey (`state`, `redirect_to`, `back_url`) and are consumed once the callback completes.

## Environment & Dependencies
- `config/app.php` expects `OAUTH_SERVER_URL`, `OAUTH_CLIENT_ID`, and `OAUTH_CLIENT_SECRET` to be defined.
- Laravel Sanctum must be enabled; the `auth:sanctum` middleware protects the `me` and `logout` endpoints.
- The `web` guard handles session-based login during the callback (`Auth::guard('web')`).
- The controller relies on Laravel's HTTP client for outgoing OAuth requests and on the `User` model for persistence (`updateOrCreate`, `logedIn`, `logedOut` events).

## Route Registry
| Method | Path | Route name | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- | --- |
| POST | `/api/auth/register` | `auth.register` | `guest` | `register` | Produce the hosted registration URL on the OAuth server. |
| GET | `/api/auth/redirect` | `auth.redirect` | `guest` | `redirect` | Generate the OAuth authorize URL and redirect (or return JSON). |
| GET | `/api/auth/callback` | `auth.callback` | – | `callback` | Exchange the authorization code, sync the user record, establish session. |
| POST | `/api/auth/me` | `auth.me` | `auth:sanctum` | `me` | Return the authenticated user resource. |
| POST | `/api/auth/logout` | `auth.logout` | `auth:sanctum` | `logout` | Revoke Sanctum tokens and end the session. |

## Endpoint Behaviour

### `POST /api/auth/register`
1. Validates `back_url` (required URL) and optional `referral` code (`exists:users,code`).
2. Builds a registration query string with `client_id`, `redirect_uri` (`auth.redirect`), `referral`, and `back_url`.
3. Responds with `{ "url": "<OAUTH_SERVER_URL>/register?..."> }`. The client is responsible for final navigation.

### `GET /api/auth/redirect`
1. Validates optional `redirect_to` and `back_url` query parameters as URLs.
2. Generates a cryptographically random `state` (40 chars), caches it for five minutes, and separately caches the optional redirect URLs with the same TTL.
3. Builds the OAuth authorize URL with `response_type=code`, empty `scope`, and the configured client credentials.
4. Returns either a JSON body (`{ "url": "..." }`) when the request expects JSON or issues a 302 redirect to the authorization endpoint.

### `GET /api/auth/callback`
1. Retrieves and removes the cached `state`; throws `InvalidArgumentException` if it is missing or does not match the incoming `state` query parameter (CSRF defence).
2. Exchanges the authorization code for tokens via `POST /oauth/token` on the OAuth server (form-encoded body).
3. Fetches the remote user profile from `/api/user` using the received bearer token.
4. Upserts the local `users` record keyed by email, updating profile fields, IP, referral linkage (`getReferrerId`), and persisting the OAuth token set (`access_token`, `refresh_token`, `expires_in`, `token_type`). The local password is rotated to a random 10-character hash.
5. Signs the user into the `web` guard, regenerates the session ID, and hands off to `authenticated()`.

### `authenticated(Request $request, User $user)`
1. Fires the `logedIn` model event (listeners can record audit trails or metrics).
2. Eager loads `settings` to determine the session timeout (`automatic_logout`, default 55 minutes).
3. Issues a Sanctum personal access token (`token_{user_id}`) with an explicit expiry aligned to the automatic logout setting; only the token plain text is returned to the client.
4. Restores and consumes the cached `redirect_to` and `back_url` values, appending query parameters: `token` (Sanctum token) and `expires_at` (remaining minutes until expiration).
5. Responds with a redirect to whichever cached URL is present (prefers `redirect_to`, falls back to `back_url`). No JSON fallback is provided here.

### `POST /api/auth/me`
1. Requires a valid Sanctum bearer token.
2. Loads related data: `settings`, `profilePhotos`, `kyc`, `unreadNotifications`.
3. Returns an `AuthenticatedUserResource`, exposing the fields listed below.

### `POST /api/auth/logout`
1. Deletes all Sanctum tokens belonging to the authenticated user.
2. Fires the `logedOut` model event.
3. Logs out of the `web` guard, invalidates the session, and regenerates the CSRF token.
4. Returns an empty 204 No Content response.

## Authenticated User Resource Contract
`App\Http\Resources\AuthenticatedUserResource` aggregates several derived properties:
- `id`, `code`, `level`, `access_token`
- `name`: prioritises the verified KYC full name if available.
- `token`: Sanctum token taken from the resource or the current bearer token.
- `automatic_logout`: value (minutes) from user settings, defaulting to 55.
- `image`: latest profile photo URL when present.
- `notifications`: unread notification count.
- `score_percentage_to_next_level`, `unasnwered_questions_count`, `hourly_profit_time_percentage`: computed via helper functions.
- `verified_kyc`: boolean derived from `User::verified()`.
- `birthdate`: Jalali-formatted birthdate when KYC is verified; otherwise `null`.

## Cache & Session Notes
- Keys: `state`, `redirect_to`, and `back_url` are cached without user scoping; ensure the cache store supports atomic operations and short TTLs.
- All cache entries are deleted on first use (`pull` semantics), preventing reuse across multiple callbacks.
- The Sanctum token expiry is set explicitly via `expiresAt`, so revocation occurs automatically once the configured timeout lapses.

## Failure Modes & Considerations
- Invalid or missing `state` results in an `InvalidArgumentException` (HTTP 500 by default). Consider wrapping with a user-friendly response if exposing externally.
- Upstream OAuth errors propagate via the HTTP client; they are not explicitly handled so failed token exchanges or profile fetches will bubble up as 4xx/5xx responses.
- If neither `redirect_to` nor `back_url` is cached when `authenticated()` runs, `$url` becomes `null/?token=...`, yielding an invalid redirect target—callers should always supply one of the parameters during the initial redirect step.
- The `register` endpoint requires the referral code to pre-exist; otherwise validation fails with HTTP 422.

## Extending the Flow
- To change the downstream redirect target behaviour, adjust the caching logic in `redirect()` and `authenticated()`.
- If additional scopes are needed, expand the `scope` parameter in `redirect()`.
- Integrate refresh token rotation by adding a scheduled job that uses the stored `refresh_token` on the `users` table.
- Wrap the OAuth client calls in try/catch blocks to convert upstream failures into structured API error responses where appropriate.


