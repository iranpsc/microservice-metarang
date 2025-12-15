# Sell Requests API Guide

## Summary
- `GET /api/sell-requests` lists the authenticated seller’s open sell offers with feature coordinates and property metadata eagerly loaded for map rendering.
- `POST /api/sell-requests/store/{feature}` publishes a sell offer after enforcing dynamic pricing floors, recalculating feature price fields, and broadcasting realtime status updates.
- `DELETE /api/sell-requests/{sellRequest}` withdraws an outstanding offer, reverts feature RGB status, and emits websocket events so clients can refresh availability.

```118:122:routes/api.php
Route::controller(SellRequestsController::class)->prefix('sell-requests')->group(function () {
    Route::get('/', 'index');
    Route::post('/store/{feature}', 'store')->can('sell', 'feature');
    Route::delete('/{sellRequest}', 'destroy')->can('delete', 'sellRequest');
});
```

All routes sit inside the global `auth:sanctum`, `verified`, and `activity` middleware stack; `store` and `destroy` also inherit the controller-level `account.security` lock, while `can` policy guards enforce ownership rules.

```88:144:routes/api.php
Route::middleware(['auth:sanctum', 'verified', 'activity'])->group(function () {
    // ... existing code ...
    Route::controller(SellRequestsController::class)->prefix('sell-requests')->group(function () {
        Route::get('/', 'index');
        Route::post('/store/{feature}', 'store')->can('sell', 'feature');
        Route::delete('/{sellRequest}', 'destroy')->can('delete', 'sellRequest');
    });
    // ... existing code ...
});
```

## Route Registry
| Method | Path | Middleware (in order) | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| GET | `/api/sell-requests` | `auth:sanctum`, `verified`, `activity` | `SellRequestsController@index` | Fetch the caller’s active sell offers with feature projections. |
| POST | `/api/sell-requests/store/{feature}` | `auth:sanctum`, `verified`, `activity`, `account.security`, `can:sell,feature` | `SellRequestsController@store` | Create a new sell offer for the bound feature. |
| DELETE | `/api/sell-requests/{sellRequest}` | `auth:sanctum`, `verified`, `activity`, `account.security`, `can:delete,sellRequest` | `SellRequestsController@destroy` | Cancel an existing sell offer. |

Scoped bindings from the enclosing route group ensure `{feature}` and `{sellRequest}` resolve against the authenticated user context, preventing cross-account access.

## Domain Model Snapshot
- **SellFeatureRequest** — Stores the seller, optional buyer, feature id, requested prices (`price_psc`, `price_irr`), and `limit`/`minimum_price_percentage`, defaulting `status` to `0` (open).

```17:51:app/Models/SellFeatureRequest.php
protected $fillable = [
    'seller_id',
    'buyer_id',
    'feature_id',
    'status',
    'price_psc',
    'price_irr',
    'limit',
    'minimum_price_percentage'
];

protected $attributes = [
    'status' => 0,
];
```

- **Feature** — Provides helper methods to translate pricing actions into RGB status flags (`changeStatusToSoldAndPriced`, `changeStatusToSoldAndNotPriced`) and exposes relationships used to hydrate sell requests.

```122:145:app/Models/Feature.php
public function sellRequests()
{
    return $this->hasMany(SellFeatureRequest::class, 'feature_id');
}
```

```223:244:app/Models/Feature.php
public function changeStatusToSoldAndPriced()
{
    return match ($this->properties->karbari) {
        FeatureIndicators::Maskoni  => FeatureIndicators::MaskoniSoldAndPriced,
        FeatureIndicators::Tejari   => FeatureIndicators::TejariSoldAndPriced,
        FeatureIndicators::Amozeshi => FeatureIndicators::AmozeshiSoldAndPriced,
    };
}

public function changeStatusToSoldAndNotPriced()
{
    return match ($this->properties->karbari) {
        FeatureIndicators::Maskoni  => FeatureIndicators::MaskoniSoldAndNotPriced,
        FeatureIndicators::Tejari   => FeatureIndicators::TejariSoldAndNotPriced,
        FeatureIndicators::Amozeshi => FeatureIndicators::AmozeshiSoldAndNotPriced
    };
}
```

- **SystemVariable & Variable** — Supply dynamic pricing guardrails (`public_pricing_limit`, `under_18_pricing_limit`) and currency conversion rates used to compute the final offer payload.
- **SellRequestResource** — Shapes API responses with identifiers, pricing, status, and nested `feature_properties` / `feature_coordinates` resources.

```17:27:app/Http/Resources/SellRequestResource.php
return [
    'id' => $this->id,
    'feature_id' => $this->feature_id,
    'seller_id' => $this->seller_id,
    'price_psc' => $this->price_psc,
    'price_irr' => $this->price_irr,
    'status' => $this->status,
    'feature_properties' => new FeaturePropertiesResource($this->whenLoaded('feature.properties')),
    'feature_coordinates' => new CoordinatesResource($this->whenLoaded('feature.coordinates')),
    'created_at' => jdate($this->created_at)->format('Y/m/d'),
];
```

## `GET /api/sell-requests` — Seller Offer Listing
**Authentication:** Required (`Bearer <sanctum-token>`).  
**Authorization:** Implicit; results are filtered to the calling seller via `whereBelongsTo(request()->user(), 'seller')`.

### Behavior
- Retrieves all sell requests owned by the caller, eager loading coordinates and properties for each feature.
- Returns a `SellRequestResource` collection; each item nests `FeaturePropertiesResource` and `CoordinatesResource` when those relations are available.
- Dates are localized via `jdate(...)->format('Y/m/d')`, matching the broader Persian calendar presentation in the app.

```27:34:app/Http/Controllers/Api/V1/Feature/SellRequestsController.php
$sellRequests = SellFeatureRequest::whereBelongsTo(request()->user(), 'seller')
    ->with('feature.coordinates', 'feature.properties')
    ->get();

return SellRequestResource::collection($sellRequests);
```

### Response Payload
Each entry includes:
- `id`, `feature_id`, `seller_id`
- `price_psc` (float), `price_irr` (integer), `status` (0 = open, 1 = closed by downstream purchase)
- `feature_properties` — address, density, RGB status, price fields, etc.
- `feature_coordinates` — `[{ id, x, y }]` arrays for map plotting.
- `created_at` (formatted `Y/m/d`)

```18:31:app/Http/Resources/FeaturePropertiesResource.php
return [
    'id' => $this->id,
    'address' => $this->address,
    'density' => $this->density,
    'label' => $this->label,
    'karbari' => $this->karbari,
    'area' => $this->area,
    'stability' => $this->stability,
    'region' => $this->region,
    'owner' => $this->owner,
    'rgb' => $this->rgb,
    'price_psc' => $this->price_psc,
    'price_irr' => $this->price_irr,
    'minimum_price_percentage' => $this->minimum_price_percentage,
];
```

```17:21:app/Http/Resources/CoordinatesResource.php
return [
    'id' => $this->id,
    'x' => $this->x,
    'y' => $this->y,
];
```

### Example
```bash
curl -X GET "https://example.com/api/sell-requests" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json"
```

### Error Modes
- `401` — Missing or invalid Sanctum token.
- `403` — Email not verified (global middleware).
- `500` — Database or serialization failures while loading related models.

## `POST /api/sell-requests/store/{feature}` — Publish Sell Offer
**Authentication:** Required.  
**Authorization:** `can:sell,feature` ensures the caller owns the feature and is eligible to sell.  
**Path params:** `feature` uses scoped implicit binding; a 404 surfaces if the feature is not found or not owned by the caller.

### Request Body
Two mutually exclusive payload shapes are accepted:

| Field set | Required fields | Forbidden fields | Notes |
| --- | --- | --- | --- |
| Explicit pricing | `price_psc`, `price_irr` (`numeric`, `min:0`; at least one non-zero) | `minimum_price_percentage` | Pricing expressed directly in PSC tokens and IRR currency. |
| Floor percentage | `minimum_price_percentage` (`numeric`, `min:80`) | `price_psc`, `price_irr` | Backend computes prices from the feature’s stability and current color exchange rates. |

```28:58:app/Http/Requests/SellFeatureRequestValidate.php
'price_psc' => [
    'nullable',
    'numeric',
    'min:0',
    Rule::requiredIf(fn () => !request()->has('minimum_price_percentage')),
    Rule::prohibitedIf(fn () => request()->has('minimum_price_percentage')),
    function ($attribute, $value, $fail) {
        if (request()->price_irr == 0 && $value == 0) {
            $fail("{$attribute} must be greater than 0!");
        }
    }
],
// ... existing code ...
'minimum_price_percentage' => [
    'nullable',
    'numeric',
    'min:80',
    Rule::requiredIf(fn () => !request()->has('price_irr') && !request()->has('price_psc')),
    Rule::prohibitedIf(fn () => request()->has('price_irr') || request()->has('price_psc')),
],
```

### Pricing Guards
- Public sellers cannot list below the configured floor (`public_pricing_limit`, default 80%).
- Sellers under 18 have a higher floor (`under_18_pricing_limit`, default 110%). Violation triggers HTTP 403 with localized messaging.
- When explicit prices are provided, the controller calculates the implied percentage (`pricing_percentage`) to re-check floor compliance.
- When only a percentage is supplied, PSC/IRR amounts are computed from feature stability multiplied by the color rate, then split 50/50 between PSC and IRR balances.

```45:75:app/Http/Controllers/Api/V1/Feature/SellRequestsController.php
$publicPricingLimit = SystemVariable::getByKey('public_pricing_limit') ?? 80;
$under18PricingLimit = SystemVariable::getByKey('under_18_pricing_limit') ?? 110;
// ... existing code ...
if ($request->has('minimum_price_percentage')) {
    if ($request->user()->isUnderEighteen() && $request->minimum_price_percentage < $under18PricingLimit) {
        abort(403, sprintf("شما مجاز به فروش زمین خود به کمتر از %s درصد قیمت خرید ملک نمی باشید", $under18PricingLimit));
    } elseif ($request->minimum_price_percentage < $publicPricingLimit) {
        abort(403, sprintf("شما مجاز به فروش زمین خود به کمتر از %s درصد قیمت خرید ملک نمی باشید", $publicPricingLimit));
    }
    $totalPrice = $feature->properties->stability * Variable::getRate($feature->getColor()) * $request->minimum_price_percentage / 100;
    $requestedPrice_psc = $totalPrice / Variable::getRate('psc') * 0.5;
    $requestedPrice_irr = $totalPrice * 0.5;
    $pricing_percentage = $request->minimum_price_percentage;
} else {
    $totalRequested_price = $request->price_psc * Variable::getRate('psc') + $request->price_irr;
    $totalTradedPrice = $feature->properties->stability * Variable::getRate($feature->getColor());

    $pricing_percentage = $totalTradedPrice > 0 ? intval($totalRequested_price / $totalTradedPrice * 100) : 100;

    if ($request->user()->isUnderEighteen() && $pricing_percentage < $under18PricingLimit) {
        abort(403, sprintf("شما مجاز به فروش زمین خود به کمتر از %s درصد قیمت خرید ملک نمی باشید", $under18PricingLimit));
    } elseif ($pricing_percentage < $publicPricingLimit) {
        abort(403, sprintf("شما مجاز به فروش زمین خود به کمتر از %s درصد قیمت خرید ملک نمی باشید", $publicPricingLimit));
    }
}
```

### Persisted Changes
On success the controller:
1. Creates a `SellFeatureRequest` with seller id, feature id, PSC/IRR prices, and the resolved `limit` percentage.
2. Updates `feature.properties` with new RGB status (`changeStatusToSoldAndPriced`), PSC/IRR values, and `minimum_price_percentage` to keep UI in sync.
3. Broadcasts `FeatureStatusChanged` so realtime clients refresh map overlays.
4. Dispatches `SellRequestNotification` to the seller, providing in-app feedback.

```77:99:app/Http/Controllers/Api/V1/Feature/SellRequestsController.php
$sellRequest = SellFeatureRequest::create([
    'seller_id' => $feature->owner->id,
    'feature_id' => $feature->id,
    'price_psc' => $requestedPrice_psc,
    'price_irr' => $requestedPrice_irr,
    'limit'     => $pricing_percentage,
]);

$feature->properties->update([
    'rgb' => $feature->changeStatusToSoldAndPriced(),
    'price_psc' => $sellRequest->price_psc,
    'price_irr' => $sellRequest->price_irr,
    'minimum_price_percentage' => $pricing_percentage
]);

broadcast(new FeatureStatusChanged([
    'id'  => $feature->id,
    'rgb' => $feature->changeStatusToSoldAndPriced(),
]));

$request->user()->notify(new SellRequestNotification($feature));

return new SellRequestResource($sellRequest);
```

### Response
- Returns `201` semantics via a `SellRequestResource` payload containing the newly created offer.^[99:99:app/Http/Controllers/Api/V1/Feature/SellRequestsController.php]

### Error Modes
- `401` — Missing/invalid token.
- `403` — Account security session absent, feature policy denied, or pricing below allowed thresholds.
- `404` — Feature not resolved through scoped binding (e.g., not owned).
- `422` — Validation failures (mutually exclusive fields, zero pricing).
- `500` — Failures while persisting property updates or sending notifications.

### Example
Create using explicit prices:
```bash
curl -X POST "https://example.com/api/sell-requests/store/18342" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
        "price_psc": 12.5,
        "price_irr": 8500000
      }'
```

Create using percentage floor:
```bash
curl -X POST "https://example.com/api/sell-requests/store/18342" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
        "minimum_price_percentage": 125
      }'
```

## `DELETE /api/sell-requests/{sellRequest}` — Withdraw Offer
**Authentication:** Required.  
**Authorization:** `can:delete,sellRequest` ensures only the seller (or privileged roles) can withdraw the request.^[121:122:routes/api.php]

### Behavior
- Retrieves the bound sell request, updates the feature RGB status back to `SoldAndNotPriced`, and deletes the row.
- Broadcasts `FeatureStatusChanged` with the reverted color state so map tiles clear the for-sale overlay.

```109:124:app/Http/Controllers/Api/V1/Feature/SellRequestsController.php
$feature = $sellRequest->feature;

$feature->properties->update([
    'rgb' => $feature->changeStatusToSoldAndNotPriced()
]);

$sellRequest->delete();

broadcast(new FeatureStatusChanged([
    'id'  => $feature->id,
    'rgb' => $feature->changeStatusToSoldAndNotPriced()
]));

return response()->noContent(200);
```

### Response
- Returns an empty `200` response (`response()->noContent(200)`), which some HTTP clients interpret as `204` with a custom status code. Consumers should prepare for either interpretation.

### Error Modes
- `401` — Missing token.
- `403` — Account security lock inactive or policy denial.
- `404` — Sell request not found / not owned.
- `500` — Database or broadcasting failure during the withdrawal.

## Side Effects & Integrations
- **Event streaming:** Both `store` and `destroy` broadcast `FeatureStatusChanged`; subscribe to the corresponding channel to keep front-end maps synchronized.
- **Notifications:** Sellers receive `SellRequestNotification` immediately after publishing an offer, signaling success in UI layers.
- **Feature properties:** Pricing actions mutate `feature.properties` to ensure other APIs (`GET /api/features/{feature}`, marketplace listings) reflect current pricing without additional joins.
- **Status flags:** RGB strings returned by `Feature::changeStatusToSoldAndPriced` / `changeStatusToSoldAndNotPriced` align with client-side color logic; consumers should treat them as authoritative.

## Operational Notes
- **Account Security Workflow:** The `account.security` middleware means sellers must unlock their account security session shortly before mutating endpoints. Automate a pre-flight unlock in client flows to avoid 403s.
- **Under-18 Sellers:** Clients should surface the higher pricing floor early in the UI to prevent repeated 403 responses for teenage users.
- **Idempotency:** Duplicate `store` calls with the same payload generate additional sell requests. Guard at the client level (e.g., disable submit button after success) if you need strict idempotency.
- **Localization:** Error messages for pricing floors are localized Farsi strings. International clients should be ready to display or translate server-supplied text.

