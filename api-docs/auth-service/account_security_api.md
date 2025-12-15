# Account Security API Guide

## Summary
- `POST /api/account/security` issues a one-time verification code tied to the authenticated user and (optionally) updates their phone number.
- `POST /api/account/security/verify` validates the submitted OTP, unlocks the account-security window for a configurable duration, and logs the action to user events.
- Both endpoints require authenticated, verified users and drive the `account.security` middleware gate that protects sensitive purchase/reset flows in production.

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| POST | `/api/account/security` | `auth:sanctum`, `verified`, `activity` | `AccountSecurityController@sendVerifyCode` | Generate & dispatch an OTP and (re)set security window length. |
| POST | `/api/account/security/verify` | `auth:sanctum`, `verified`, `activity` | `AccountSecurityController@verify` | Confirm OTP and unlock the account-security window. |

Routes are registered in `routes/api.php` inside the authenticated user group. In production, subsequent sensitive endpoints (for example, feature purchases and credential resets) also enforce the `account.security` middleware, which inspects the unlocked window opened by `verify`.

## Domain Concepts
- **AccountSecurity model** – Stores per-user flags:
  - `unlocked` (`bool`, default `false`): whether a security window is currently active.
  - `until` (`int|null`): Unix timestamp marking when the unlock expires.
  - `length` (`int` seconds): duration of the unlock window configured on the last OTP request.
- **OTP (One-Time Password)** – Backed by the polymorphic `otp` relation; codes are stored hashed via `Hash::make` and are single-use.
- **GetOtpNotification** – Queued notification that delivers the numeric code through Kavenegar SMS by default (`verifyLookup('verify', $code)`) or mail when explicitly configured.
- **AccountSecurityRequest** – Form request enforcing:
  - Authorization: the request user must own the target `AccountSecurity` record.
  - Validation: `time` is required (`int`, between 5–60 minutes); `phone` is required, unique, and `ir_mobile` formatted when the user has not verified a phone yet.

## `POST /api/account/security` – Request OTP
**Authentication:** Required (`Bearer` token via Sanctum).  
**Body:** JSON.

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `time` | integer | Yes | Minutes to keep the account unlocked after successful verification (`5 ≤ time ≤ 60`). Stored as seconds in `length`. |
| `phone` | string | When missing verified phone | Must be unique in `users.phone` and match the `ir_mobile` rule. If provided, the user's phone is updated before the OTP dispatch. |

### Behavior
- Creates an `AccountSecurity` record if one does not already exist, otherwise resets `unlocked` to `false`, clears `until`, and overwrites `length`.
- Generates a random 6-digit code (`random_int(100000, 999999)`), stores its hash on the associated OTP record (`Otp::updateOrCreate`), and queues a notification to the user.
- Returns HTTP 204 with no content.

### Error Modes
- `401` – Unauthenticated requests fail Sanctum guard.
- `403` – Authorization failure when the user lacks an `AccountSecurity` record (rare in practice; seeded during onboarding).
- `422` – Validation errors for `time` bounds, missing `phone`, or duplicate `phone`.
- `500` – Unhandled exceptions (for example, notification transport failures).

### Example
```bash
curl -X POST https://example.com/api/account/security \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"time": 15, "phone": "09123456789"}' \
  -i
```
Successful calls respond with `HTTP/1.1 204 No Content`.

## `POST /api/account/security/verify` – Confirm OTP
**Authentication:** Required (`Bearer` token via Sanctum).  
**Body:** JSON.

| Field | Type | Required | Notes |
| --- | --- | --- | --- |
| `code` | string | Yes | Must be exactly 6 numeric digits; compared against the hashed OTP. |

### Behavior
- Validates the code and ensures the user has a locked `AccountSecurity` record with an active OTP.
- On match:
  - Marks `phone_verified_at` with the current timestamp if the user previously lacked a verified phone.
  - Sets `unlocked` to `true` and `until` to `time() + length`, enabling guarded flows until the window expires.
  - Deletes the OTP record to prevent reuse.
  - Logs a user event (`events()->create`) with the Farsi label "غیر فعال سازی امنیت حساب کاربری", capturing request IP and user agent.
- Returns HTTP 204 with no content.

### Error Modes
- `400` – Triggered when no OTP is pending, the account is already unlocked, or the hashed comparison fails.
- `401` – Missing/invalid Sanctum token.
- `422` – Validation failure (non-numeric or wrong-length `code`).
- `500` – Unexpected server errors.

### Example
```bash
curl -X POST https://example.com/api/account/security/verify \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"code": "123456"}' \
  -i
```
Successful calls respond with `HTTP/1.1 204 No Content`.

## Interaction with `account.security` Middleware
- The middleware (`App\Http\Middleware\AccountSecurity`) is production-guarded: it bypasses checks in non-production environments.
- In production, requests hitting protected routes must have `unlocked === true` and `time() <= until`; otherwise:
  - JSON clients receive `410 Gone` with the message "جهت ادامه امنیت حساب کاربری خود را غیر فعال کنید!".
  - Browser clients are redirected to `RouteServiceProvider::HOME`.
- The unlock window closes automatically once `until` has elapsed; clients must re-run the OTP flow to regain access.

## Operational Notes
- OTP codes are not rate-limited in the controller; consider layering throttling middleware if abuse is observed.
- Because notifications are queued, ensure the queue worker is running; otherwise OTP delivery is delayed.
- Store `time` values conservatively. Short windows reduce social-engineering exposure; long windows favor usability but extend `account.security` bypass time.
- Persist access logs: the verification step already writes human-readable events that downstream analytics can leverage.


