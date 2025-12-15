# Bank Accounts API Guide

## Summary
- `GET /api/bank-accounts` returns the authenticated user’s bank accounts via `BankAccountResource`, enforcing `BankAccountPolicy::viewAny`.
- `POST /api/bank-accounts` creates a pending bank account for the caller after passing validation (`bank_name`, `ir_sheba`, `ir_bank_card_number`) and `BankAccountPolicy::create` (requires a verified user).
- `GET /api/bank-accounts/{bankAccount}` exposes a single record owned by the caller; authorisation relies on polymorphic ownership via `BankAccountPolicy::view`.
- `PUT /api/bank-accounts/{bankAccount}` allows edits only when the record belongs to the caller **and** is currently rejected; updates reset status to `0` (pending) and clear `errors`.
- `DELETE /api/bank-accounts/{bankAccount}` soft-removes the record by policy, limiting deletions to the owner.
- All routes are protected by the `auth:sanctum`, `verified`, and `activity` middleware stack.

## Route Registry
| Method | Path | Middleware | Controller action | Notes |
| --- | --- | --- | --- | --- |
| GET | `/api/bank-accounts` | `auth:sanctum`, `verified`, `activity` | `BankAccountController@index` | Lists the caller’s bank accounts. |
| POST | `/api/bank-accounts` | `auth:sanctum`, `verified`, `activity` | `BankAccountController@store` | Creates a new bank account in pending state. |
| GET | `/api/bank-accounts/{bankAccount}` | `auth:sanctum`, `verified`, `activity` | `BankAccountController@show` | Shows a single owned bank account. |
| PUT/PATCH | `/api/bank-accounts/{bankAccount}` | `auth:sanctum`, `verified`, `activity` | `BankAccountController@update` | Updates a rejected bank account and resubmits for review. |
| DELETE | `/api/bank-accounts/{bankAccount}` | `auth:sanctum`, `verified`, `activity` | `BankAccountController@destroy` | Removes an owned bank account. |

All routes are registered in `routes/api.php` via `Route::apiResource('bank-accounts', BankAccountController::class)` inside the authenticated group.

## Resource Schema
Responses are wrapped with `BankAccountResource`, yielding the following JSON structure:

```18:24:app/Http/Resources/BankAccountResource.php
return [
    'id' => $this->id,
    'bank_name' => $this->bank_name,
    'shaba_num' => $this->shaba_num,
    'card_num'  => $this->card_num,
    'status' => $this->status,
    'errors' => $this->whenNotNull($this->errors),
];
```

- `status` uses integer codes: `0` (pending review), `1` (verified/approved), `-1` (rejected). Helper methods on `BankAccount` expose these states (`pending()`, `verified()`, `rejected()`).
- `errors` (nullable JSON array) communicates back-office validation feedback when a submission is rejected.

Collections follow Laravel’s resource conventions: `{ "data": [ ... ] }`.

## Validation Rules
Both `store` and `update` actions perform on-request validation using `Request::validate()`:

- `bank_name`: `required|string|min:2` (create) and `required|string|max:255` (update).
- `shaba_num`: `required|ir_sheba|unique:bank_accounts,shaba_num[,<ignoreId>]`.
- `card_num`: `required|ir_bank_card_number|unique:bank_accounts,card_num[,<ignoreId>]`.

Additional behaviours:
- `store` and `update` normalise the record to `status = 0` (pending) and `errors = null` (on update) so the back-office can re-review.
- The unique rules ensure no duplicate SHABA or card numbers across the entire table; the update rule ignores the current record ID.
- Custom validators `ir_sheba` and `ir_bank_card_number` must be registered within the application's validation service provider; they enforce Iranian banking formats.

### Failure Responses
- Validation failures return `422 Unprocessable Entity` with Laravel’s standard error envelope (`{ "message": "...", "errors": { field: [...] } }`).
- Unique violations and format mismatches (`ir_sheba`, `ir_bank_card_number`) surface via the same 422 payload.

## Policy & Authorization
`BankAccountController` invokes `$this->authorizeResource(BankAccount::class)` in its constructor, mapping resource abilities to policy methods:

| Ability | Policy method | Rule |
| --- | --- | --- |
| `viewAny` | `BankAccountPolicy::viewAny` | Any authenticated user may list their bank accounts. |
| `view` | `BankAccountPolicy::view` | The authenticated user must own the record (`$bankAccount->bankable->is($user)`). |
| `create` | `BankAccountPolicy::create` | Caller must be a verified user (`$user->verified() === true`). |
| `update` | `BankAccountPolicy::update` | Caller must own the record **and** the account must be rejected (`status === -1`) to permit resubmission. |
| `delete` | `BankAccountPolicy::delete` | Caller must own the record; status does not restrict deletion. |

Middleware requirements (`auth:sanctum`, `verified`, `activity`) ensure:
- Requests include a valid Sanctum bearer token.
- The user has a verified email/phone.
- The activity middleware tracks the user session and may throttle inactive accounts.

## Endpoint Details
### `GET /api/bank-accounts`
- Returns the caller’s bank accounts without pagination.
- Output is a resource collection; empty array when the user has none.
- Errors: `401 Unauthorized` (missing/invalid token).

### `POST /api/bank-accounts`
- **Body parameters:**
  - `bank_name` (string, required, min length 2),
  - `shaba_num` (string, required, valid Iranian sheba, unique),
  - `card_num` (string, required, valid Iranian card number, unique).
- **Side effects:** Creates a `BankAccount` with `status = 0`.
- **Responses:** `201 Created` with the serialized resource.
- **Errors:** `401` (auth), `403` (policy rejects when user not verified), `422` (validation).

### `GET /api/bank-accounts/{bankAccount}`
- Retrieves a single record belonging to the caller.
- Automatic route model binding ensures 404 when the ID does not exist or does not belong to the caller.
- **Responses:** `200 OK` with resource; `403` forbidden when policy denies (e.g., requesting someone else’s account).

### `PUT /api/bank-accounts/{bankAccount}`
- Same payload structure as `POST`.
- Only callable for rejected records; otherwise policy returns `403 Forbidden`.
- Updates reset `status` to `0` and remove `errors` to trigger re-review.
- Returns updated resource on `200 OK`.
- Error modes: `401`, `403`, `404`, `422`.

### `DELETE /api/bank-accounts/{bankAccount}`
- Deletes the record (hard delete via Eloquent `delete()`).
- **Responses:** `204 No Content` on success; `401`/`403`/`404` when unauthorized or record not found.

## Usage Examples
```bash
curl -X POST https://example.com/api/bank-accounts \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "bank_name": "Tejarat",
    "shaba_num": "IR820540102680020817909002",
    "card_num": "6037997551234567"
  }'
```

```bash
curl -X PUT https://example.com/api/bank-accounts/42 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "bank_name": "Melli",
    "shaba_num": "IR820540102680020817909002",
    "card_num": "6037997551234567"
  }'
```

## Operational Notes
- Because bank accounts are scoped to the authenticated user via `request()->user()->bankAccounts()`, multi-tenant separation relies on polymorphic ownership (`bankable_type`, `bankable_id`); ensure no cross-tenant leakage in database queries.
- Consider adding pagination or ordering when users may register multiple bank accounts.
- If moderator feedback is returned through `errors`, remind clients to surface those messages before allowing resubmission.
- Re-verification flow: once the back-office approves, set `status` to `1` and (optionally) persist a timestamp to audit verification events.


