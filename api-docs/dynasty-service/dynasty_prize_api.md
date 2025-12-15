# Dynasty Prize API

## Overview
- Dynasty prize endpoints live under `/api/dynasty/prizes` and are registered inside the global `auth:sanctum`, `verified`, and `activity` middleware group.
- Each route works with `RecievedPrize` models that belong to the authenticated user and are transformed through `DynastyPrizeResource`.
- Redeeming a prize (`POST`) credits the caller’s wallet and growth variables, then deletes the receipt to prevent double-claims.

## Authentication & Authorization
- **Middleware**: `auth:sanctum`, `verified`, `activity` are enforced for all prize routes via the shared group in `routes/api.php`.
- **Policies**: No explicit policy or `Gate::can` check is attached to these routes. Access control relies on authentication plus route model binding to resolve `RecievedPrize`. Consumers should ensure they only expose identifiers for prizes owned by the current user; otherwise, additional checks (e.g., custom policy or scoped binding) should be introduced.
- **Route binding**: `Route::scopeBindings()` is applied to the enclosing group, but because the prize routes have no parent parameters, the framework performs a simple `RecievedPrize::findOrFail($id)` lookup. A non-existent or unauthorized ID returns HTTP 404.

## Endpoints
| Method | Path | Controller | Description |
| --- | --- | --- | --- |
| GET | `/api/dynasty/prizes` | `DynastyPrizeController@index` | Lists all unclaimed dynasty prizes for the authenticated user. |
| GET | `/api/dynasty/prizes/{recievedPrize}` | `DynastyPrizeController@show` | Returns a single prize, including its congratulatory `message`. |
| POST | `/api/dynasty/prizes/{recievedPrize}` | `DynastyPrizeController@store` | Redeems the referenced prize, updates balances, and removes the receipt (204 No Content). |

The namespace for all actions is `App\Http\Controllers\Api\V1\Dynasty\DynastyPrizeController`.

```17:38:app/Http/Controllers/Api/V1/Dynasty/DynastyPrizeController.php
public function store(Request $request, RecievedPrize $recievedPrize)
{
    $user = $request->user();
    $prize = $recievedPrize->prize;

    $user->wallet->increment('psc', ($prize->psc / Variable::getRate('psc')));
    $user->wallet->increment('satisfaction', $prize->satisfaction);

    $variables = $user->variables;

    $variables->update([
        'referral_profit' => $variables->referral_profit + ($variables->referral_profit * $prize->introduction_profit_increase),
        'data_storage' => $variables->data_storage + ($variables->data_storage * $prize->data_storage),
        'withdraw_profit' => $variables->withdraw_profit + ($variables->withdraw_profit * $prize->accumulated_capital_reserve),
    ]);

    $recievedPrize->delete();
    return response()->noContent();
}
```

## Request & Response Details
### GET `/api/dynasty/prizes`
- **Request body**: none.
- **Response 200**: JSON array of `DynastyPrizeResource` objects.
- **Resource fields** (`index` route):
  - `id` (integer): receipt identifier.
  - `psc` (numeric): raw PSC amount stored on the underlying `DynastyPrize`.
  - `satisfaction` (string): formatted percentage (e.g., `"12"` for 12%).
  - `introducation_profit_increase` (string): formatted percentage boost to referral profit.
  - `accumulated_capital_reserve` (string): formatted percentage boost to withdraw profit.
  - `data_storage` (string): formatted percentage boost to storage capacity.
- **Includes**: The congratulatory `message` is omitted in list responses.

```17:27:app/Http/Resources/Dynasty/DynastyPrizeResource.php
return [
    'id' => $this->id,
    'psc' => $this->prize->psc,
    'satisfaction' => number_format($this->prize->satisfaction * 100),
    'introducation_profit_increase' => number_format($this->prize->introduction_profit_increase * 100),
    'accumulated_capital_reserve' => number_format($this->prize->accumulated_capital_reserve * 100),
    'data_storage' => number_format($this->prize->data_storage * 100),
    $this->mergeWhen(request()->routeIs('prizes.show'), [
        'message' => $this->message,
    ])
];
```

### GET `/api/dynasty/prizes/{recievedPrize}`
- **Route parameter**: `recievedPrize` must be the numeric ID of a `received_prizes` record.
- **Response 200**: Same payload as index with the addition of the `message` field (string). The message is only exposed when the request matches the named route `prizes.show`.
- **Errors**:
  - `404 Not Found` when the ID does not resolve via route binding.

### POST `/api/dynasty/prizes/{recievedPrize}`
- **Request body**: none required; the route parameter identifies the prize.
- **Validation**: No explicit form request or `validate()` call. The action relies on the model binding succeeding and the authenticated user having the necessary relations (`wallet`, `variables`). If these relations are missing, a 500-level error would be thrown.
- **Processing**:
  - Converts PSC to wallet units using the dynamic rate from `Variable::getRate('psc')`.
  - Adds raw satisfaction points to `wallet.satisfaction`.
  - Amplifies user `variables` (`referral_profit`, `data_storage`, `withdraw_profit`) multiplicatively by the prize percentages.
  - Deletes the `RecievedPrize` row to prevent repeat redemption.
- **Response 204**: Empty body on success.
- **Errors**:
  - `404 Not Found` if the prize cannot be resolved.
  - `500 Internal Server Error` if dependent relations (`wallet`, `variables`) are missing; no guard rails exist in the controller.

## Data Model Relationships
- `RecievedPrize` belongs to `User` and `DynastyPrize`, with fillable fields `user_id`, `prize_id`, `message`.
- `User::recievedDynastyPrizes()` exposes a one-to-many relation used by the index endpoint.
- `DynastyPrize` stores the immutable prize definition (psc value and multipliers). It also provides a `getMemberTitle()` helper that localizes the member role.

## Validation Summary
- No explicit validation rules are declared for any dynasty prize route.
- Input surface is limited to the `recievedPrize` identifier resolved via route model binding.
- Consumers should implement client-side guards ensuring they only submit IDs drawn from the authenticated user’s `GET /api/dynasty/prizes` response.
- If stricter server validation is required, consider introducing a form request or custom route binding that scopes prizes to `request()->user()->recievedDynastyPrizes()`.

## Operational Notes
- **Idempotency**: Redemption is destructive—calling `POST` twice will fail on the second attempt with 404 because the record is deleted after the first success.
- **Concurrency**: No explicit locking is applied. If two redemption attempts race, one succeeds, the other receives 404 due to deletion.
- **Currency conversion**: PSC payouts are divided by the dynamic rate from the `variables` table, allowing administrators to adjust payouts without code changes.

## Change Recommendations
- Add a policy or scoped binding to guarantee that `recievedPrize` belongs to the authenticated user before redemption.
- Wrap critical wallet/variable updates inside a database transaction for atomicity.
- Validate presence of `wallet` and `variables` relations (or eager load them) to produce clearer 422 responses if the user profile is misconfigured.


