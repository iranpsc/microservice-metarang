# Feature Hourly Profit API Guide

## Summary
- `GET /api/hourly-profits` paginates the caller’s hourly profit records (10 per page) and returns aggregate totals grouped by application type (`karbari`).
- `POST /api/hourly-profits` validates the requested `karbari` bucket (`m`, `t`, `a`), rolls its pending profits into the caller’s wallet, zeroes the balances, resets withdrawal deadlines, and dispatches a `FeatureHourlyProfitDeposit` notification.
- `POST /api/hourly-profits/{featureHourlyProfit}` targets a single profit entry via route-model binding, credits the user wallet, resets the record, and responds with `HourlyProfitResource`.
- All routes inherit the enclosing middleware stack `auth:sanctum`, `verified`, `activity`; controller methods additionally scope database work to `request()->user()` to prevent cross-account access.

## Route Registry
| Method | Path | Middleware | Controller action | Notes |
| --- | --- | --- | --- | --- |
| GET | `/api/hourly-profits` | `auth:sanctum`, `verified`, `activity` | `FeatureHourlyProfitController@index` | Simple-paginates (`per_page=10`) hourly profits belonging to caller and adds aggregate totals per `karbari`. |
| POST | `/api/hourly-profits` | `auth:sanctum`, `verified`, `activity` | `FeatureHourlyProfitController@getProfitsByApplication` | Requires `karbari` filter; iterates caller’s profits of that type, credits wallet, clears balances, and schedules next availability. |
| POST | `/api/hourly-profits/{featureHourlyProfit}` | `auth:sanctum`, `verified`, `activity` | `FeatureHourlyProfitController@getSingleProfit` | Credits one profit entry, resets it, and returns its resource payload. Ensure the bound record belongs to the caller before use. |

## Domain Notes
- `FeatureHourlyProfit` holds `amount`, `asset`, `dead_line`, `is_active`, and belongs both to a `Feature` (with nested `properties.karbari`) and a `User`.
- `karbari` values map to application types: `m` (maskoni/residential), `t` (tejari/commercial), `a` (amozeshi/educational). Notification payloads translate these to Persian labels (`مسکونی`, `تجاری`, `آموزشی`) and color-coded assets (`yellow`, `red`, `blue` respectively).
- Withdrawal cooldown derives from `user->variables->withdraw_profit` (days). After each payout the controller resets `dead_line` to `now()->addSeconds(days * 86400)` so future accrual is delayed.

## Endpoint Details

### `GET /api/hourly-profits`
- **Purpose**: Retrieve a paginated view of the caller’s hourly profits alongside aggregate totals per `karbari`.
- **Security**: Covered by global middleware; query uses `FeatureHourlyProfit::whereBelongsTo(request()->user())`, preventing leakage.
- **Request params**: None (pagination cursor handled automatically by `simplePaginate(10)`).
- **Response body**: Returns a `data` array of `HourlyProfitResource` items plus an `additional.total_*` payload containing formatted sums.

```18:26:app/Http/Resources/HourlyProfitResource.php
return [
    'id'         => $this->id,
    'feature_db_id' => $this->feature->id,
    'feature_id' => $this->feature->properties->id,
    'amount'     => number_format($this->amount, 3),
    'karbari'    => $this->feature->properties->karbari,
    'dead_line'  => jdate($this->dead_line)->format('Y/m/d'),
    // ... existing code ...
];
```

- **Additional totals**: `total_maskoni_profit`, `total_tejari_profit`, `total_amozeshi_profit` (each formatted with `number_format(..., 2)`).

### `POST /api/hourly-profits`
- **Purpose**: Withdraw all accumulated profits for one `karbari` bucket to the caller’s wallet balance.
- **Validation**: `karbari` is `required|in:m,t,a`. Requests lacking the field or using unexpected codes fail with HTTP 422.
- **Processing**:
  - Chunks caller-owned profits (`chunkById(100)`) to avoid memory spikes.
  - For each matching record: increments `user->wallet` by `amount` (using the record’s `asset` column), zeroes the stored amount, and extends `dead_line`.
  - Accumulates the total withdrawal value; if positive, dispatches a queued `FeatureHourlyProfitDeposit` notification with color + localized label.
- **Response**: Always returns HTTP 200 with an empty JSON object (`{}`).

### `POST /api/hourly-profits/{featureHourlyProfit}`
- **Purpose**: Withdraw a single `FeatureHourlyProfit` record instead of the entire bucket.
- **Security**: No explicit policy guard; ensure your caller only passes IDs they own. Consider adding a `can` middleware or manual ownership check if exposing beyond trusted clients.
- **Processing**:
  - Increments `user->wallet` with the record’s `asset`/`amount`.
  - Sends `FeatureHourlyProfitDeposit` if `amount > 0`, populating `asset` via `Feature::getColor()` and `id` with `properties.id`.
  - Zeros `amount` and bumps `dead_line` by the withdraw interval.
- **Response**: Returns the updated `FeatureHourlyProfit` wrapped by `HourlyProfitResource`.

## Policies & Guards
- All hourly-profit endpoints sit inside the top-level `Route::middleware(['auth:sanctum', 'verified', 'activity'])` group, so access requires authenticated, verified, and active sessions.
- Controller methods rely on `whereBelongsTo($request->user())` when reading collections, but `getSingleProfit` depends on route-model binding. If you surface this endpoint publicly, add either a policy (`FeatureHourlyProfitPolicy`) or an inline `$this->authorize('view', $featureHourlyProfit)` to prevent cross-account withdrawals.

## Validation Summary
- `POST /api/hourly-profits`: body must include `karbari` string ∈ `{m, t, a}`.
- Other endpoints perform implicit validation via model binding; ensure 404 behavior by only exposing IDs belonging to the caller.

## Notifications & Side Effects
- `FeatureHourlyProfitDeposit` notification is queued and notifies the caller after successful withdrawals, including asset color, amount, `karbari` label, and optional feature/property ID in the single-withdrawal flow.
- Wallet adjustments use `user->wallet->increment($asset, $amount)`, so wallet columns must exist for each asset code persisted in `FeatureHourlyProfit`.


