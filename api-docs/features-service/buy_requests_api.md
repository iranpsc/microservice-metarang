# Buy Requests API Guide

## Summary
- `GET /api/buy-requests` lists purchase offers the authenticated user has placed, including feature metadata and seller identifiers.
- `GET /api/buy-requests/recieved` returns incoming offers for the authenticated seller with buyer avatars preloaded for quick UI rendering.
- `POST /api/buy-requests/store/{feature}` creates a buy offer, escrows funds (PSC + IRR) including platform fees, and notifies both parties.
- `POST /api/buy-requests/accept/{buyFeatureRequest}` finalizes the trade, transfers feature ownership, distributes funds, logs commissions, cancels competing offers, and emits realtime updates.
- `POST /api/buy-requests/reject/{buyFeatureRequest}` and `DELETE /api/buy-requests/delete/{buyFeatureRequest}` release escrowed funds and delete the offer.
- `POST /api/buy-requests/add-grace-period/{buyFeatureRequest}` lets sellers grant buyers extra time, up to 30 days, to complete payment milestones.

```124:132:routes/api.php
Route::controller(BuyRequestsController::class)->prefix('buy-requests')->group(function () {
    Route::get('/', 'index');
    Route::get('/recieved', 'recievedBuyRequests');
    Route::post('/store/{feature}', 'store')->can('sendBuyRequest', 'feature');
    Route::delete('/delete/{buyFeatureRequest}', 'destroy')->can('delete', 'buyFeatureRequest');
    Route::post('/accept/{buyFeatureRequest}', 'acceptBuyRequest')->can('accept', 'buyFeatureRequest');
    Route::post('/reject/{buyFeatureRequest}', 'rejectBuyRequest')->can('reject', 'buyFeatureRequest');
    Route::post('/add-grace-period/{buyFeatureRequest}', 'addGracePeriod')->can('addGracePeriod', 'buyFeatureRequest');
});
```

All routes inherit the enclosing `auth:sanctum`, `verified`, and `activity` middleware. The controller further applies `account.security` (and redundantly `verified`) to every mutating action; `index` and `recievedBuyRequests` are exempted from the controller-level checks but still require authentication and verification through the route group.

```28:31:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
public function __construct()
{
    $this->middleware(['account.security', 'verified'])->except(['index', 'recievedBuyRequests']);
}
```

## Route Registry
| Method | Path | Middleware (in order) | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| GET | `/api/buy-requests` | `auth:sanctum`, `verified`, `activity` | `BuyRequestsController@index` | Fetch the caller’s outgoing buy offers with nested feature + seller details. |
| GET | `/api/buy-requests/recieved` | `auth:sanctum`, `verified`, `activity` | `BuyRequestsController@recievedBuyRequests` | Retrieve buy offers received for features the caller owns, including buyer avatars. |
| POST | `/api/buy-requests/store/{feature}` | `auth:sanctum`, `verified`, `activity`, `account.security`, `can:sendBuyRequest,feature` | `BuyRequestsController@store` | Create a new buy offer and escrow funds. |
| DELETE | `/api/buy-requests/delete/{buyFeatureRequest}` | `auth:sanctum`, `verified`, `activity`, `account.security`, `can:delete,buyFeatureRequest` | `BuyRequestsController@destroy` | Cancel an outgoing offer and refund escrow. |
| POST | `/api/buy-requests/accept/{buyFeatureRequest}` | `auth:sanctum`, `verified`, `activity`, `account.security`, `can:accept,buyFeatureRequest` | `BuyRequestsController@acceptBuyRequest` | Finalize a sale, transfer ownership, and settle wallets. |
| POST | `/api/buy-requests/reject/{buyFeatureRequest}` | `auth:sanctum`, `verified`, `activity`, `account.security`, `can:reject,buyFeatureRequest` | `BuyRequestsController@rejectBuyRequest` | Decline an incoming offer and refund the buyer. |
| POST | `/api/buy-requests/add-grace-period/{buyFeatureRequest}` | `auth:sanctum`, `verified`, `activity`, `account.security`, `can:addGracePeriod,buyFeatureRequest` | `BuyRequestsController@addGracePeriod` | Extend the buyer’s payment deadline by 1–30 days. |

Scoped bindings ensure `{feature}` and `{buyFeatureRequest}` resolve within the authenticated user’s domain, preventing cross-account lookups.

## Domain Model Snapshot
- **BuyFeatureRequest** — Tracks buyer, seller, feature, pricing (PSC + IRR), textual note, `status` (`0` pending, `1` completed), and optional `requested_grace_period`. Soft deletes preserve historical trades.

```17:41:app/Models/BuyFeatureRequest.php
protected $fillable = [
    'seller_id',
    'buyer_id',
    'feature_id',
    'status',
    'note',
    'price_psc',
    'price_irr',
    'requested_grace_period',
];

protected function casts()
{
    return [
        'seller_id' => 'int',
        'feature_id' => 'int',
        'buyer_id' => 'int',
        'status' => 'int',
        'requested_grace_period' => 'datetime',
    ];
}
```

- **LockedAsset** (related via `lockedAsset()`) stores the PSC/IRR values frozen during an open offer. The controller uses it to refund buyers or forward proceeds.
- **Trade** and **Comission** models capture finalized deals and revenue splits when offers are accepted.
- **BuyRequestResource** shapes API responses with buyer/seller profiles, feature projections, formatted Jalali timestamps, and optional grace period data.

```17:47:app/Http/Resources/BuyRequestResource.php
return [
    'id' => $this->id,
    'buyer' => $this->whenLoaded('buyer', function () {
        return [
            'id' => $this->buyer->id,
            'code' => $this->buyer->code,
            'profile_photo' => $this->buyer->latestProfilePhoto?->url,
        ];
    }),
    'seller' => $this->whenLoaded('seller', function () {
        return [
            'id' => $this->seller->id,
            'code' => $this->seller->code,
        ];
    }),
    'feature_id' => $this->feature_id,
    'status' => $this->status,
    'note' => $this->note,
    'price_psc' => $this->price_psc,
    'price_irr' => $this->price_irr,
    'feature_properties' => new FeaturePropertiesResource($this->whenLoaded('feature', function () {
        return $this->feature->properties;
    })),
    'feature_coordinates' => CoordinatesResource::collection($this->whenLoaded('feature', function () {
        return $this->feature->coordinates;
    })),
    'created_at' => jdate($this->created_at)->format('Y/m/d'),
    'requested_grace_period' => $this->when($this->requested_grace_period, function () {
        return jdate($this->requested_grace_period)->format('Y/m/d H:i:s');
    }),
];
```

## `GET /api/buy-requests` — Buyer Offer Listing
**Authentication:** Required (`Bearer <sanctum-token>`).  
**Authorization:** Implicit; `whereBelongsTo(request()->user(), 'buyer')` binds results to the caller.

### Behavior
- Fetches every buy request where the user is the buyer.
- Eager loads feature coordinates/properties and the seller’s lightweight profile to minimize subsequent API calls.
- Returns a `BuyRequestResource` collection; Jalali `created_at` keeps date formatting consistent with other dashboards.

```40:46:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
$buyRequests = BuyFeatureRequest::whereBelongsTo(request()->user(), 'buyer')
    ->with('feature.coordinates', 'feature.properties', 'seller:id,code')
    ->latest()
    ->get();

return BuyRequestResource::collection($buyRequests);
```

### Example
```bash
curl -X GET "https://example.com/api/buy-requests" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json"
```

### Error Modes
- `401` — Missing or invalid Sanctum token.
- `403` — Email not verified (from group middleware).
- `500` — Database or serialization failure loading relationships.

## `GET /api/buy-requests/recieved` — Seller Inbox
**Authentication:** Required.  
**Authorization:** Results restricted to features owned by the caller via `whereBelongsTo(request()->user(), 'seller')`.

### Behavior
- Hydrates buyer code and latest profile photo for UI cards.
- Filters and sorts identically to the buyer listing, yielding a `BuyRequestResource` collection.

```158:164:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
$receivedBuyRequests = BuyFeatureRequest::whereBelongsTo(request()->user(), 'seller')
    ->with('feature.coordinates', 'feature.properties', 'buyer:id,code', 'buyer.latestProfilePhoto')
    ->latest()
    ->get();

return BuyRequestResource::collection($receivedBuyRequests);
```

### Error Modes
- `401` / `403` — Same as above.
- `500` — Relationship loading problems.

## `POST /api/buy-requests/store/{feature}` — Submit Offer
**Authentication:** Required.  
**Authorization:** `can:sendBuyRequest,feature` enforces `FeaturePolicy::sendBuyRequest`. The policy requires:
- The caller is not the feature owner and not the platform RGB user (`code` = `hm-2000000`).
- The feature is not tied to a dynasty (`$feature->dynasty` is `null`).
- The caller has no other pending (`status = 0`) buy requests for the same feature.  
**Path param:** `feature` resolves via scoped binding; 404 if unavailable or unauthorized.

### Request Body
| Field | Type | Rules | Notes |
| --- | --- | --- | --- |
| `note` | string | optional, max 500 | Stored verbatim. |
| `price_psc` | numeric | required, `min:0`, cannot be zero when `price_irr` is zero | PSC tokens offered. |
| `price_irr` | numeric | required, `min:0`, cannot be zero when `price_psc` is zero | IRR cash portion. |

```26:48:app/Http/Requests/BuyFeatureRequestValidate.php
'price_psc' => [
    'required',
    'numeric',
    'min:0',
    function ($attribute, $value, $fail) {
        if (request()->price_irr == 0 && $value == 0) {
            $fail("{$attribute} must be greater than 0!");
        }
    }
],
```

### Pricing and Balance Guards
- Computes the total offer value (IRR + PSC converted via `Variable::getRate('psc')`) and compares it to the feature’s last traded price times its minimum floor percentage. If below the permitted floor, a localized `403` is thrown.
- Requires sufficient wallet balance to cover both PSC and IRR components plus the platform fee (`config('rgb.fee')`). Validation exceptions target the offending field.
- Applies the fee (PSC + IRR) after validation, deducts funds, and locks them in `lockedwallet`.

```67:113:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
if ($totalRequestedPrice / $totalFeaturePrice * 100 < $floor_price_percentage) {
    abort(403, sprintf("شما به مجاز به ارسال درخواست خرید به کمتر از %s قیمت ملک نمی باشید!", $floor_price_percentage));
}

if ($buyer->wallet->psc < $price_psc + $price_psc * config('rgb.fee')) {
    throw ValidationException::withMessages([
        'price_psc' => 'موجودی psc شما کافی نیست!'
    ]);
} elseif ($buyer->wallet->irr < $price_irr + $price_irr * config('rgb.fee')) {
    throw ValidationException::withMessages([
        'price_irr' => 'موجودی ریال شما کافی نیست!'
    ]);
}
```

### Side Effects
1. Creates a `BuyFeatureRequest` row to persist the offer.
2. Deducts PSC/IRR balances inclusive of fees, records `lockedwallet` escrow for later release.
3. Adds paired withdrawal transactions to the polymorphic `transactions()` relationship.
4. Notifies both buyer and seller via `BuyRequestNotification`, differentiating templates by `type` (`buyer` vs `seller`).

```88:145:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
$buyer->lockedwallet()->create([
    'buy_feature_request_id' => $buyFeatureRequest->id,
    'feature_id'             => $buyFeatureRequest->feature->id,
    'psc'                    => $price_psc,
    'irr'                    => $price_irr
]);

$buyFeatureRequest->transactions()->create([
    'user_id' => $buyer->id,
    'asset'   => 'psc',
    'amount'  => $price_psc,
    'action'  => 'withdraw',
]);
```

### Response
- Returns the newly created request as `BuyRequestResource`.

### Error Modes
- `401` — Missing token.
- `403` — Floor price or policy violation, account security lock absent.
- `404` — Feature not found in scoped binding.
- `422` — Validation failure (zero pricing, malformed note).
- `500` — Wallet mutation or notification failures.

### Example
```bash
curl -X POST "https://example.com/api/buy-requests/store/18342" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
        "price_psc": 15.5,
        "price_irr": 9000000,
        "note": "Ready to close this week."
      }'
```

## `POST /api/buy-requests/accept/{buyFeatureRequest}` — Finalize Trade
**Authentication:** Required.  
**Authorization:** `can:accept,buyFeatureRequest` maps to `BuyFeatureRequestPolicy::accept` and requires the caller to be the bound seller and the request `status` to remain `0` (pending).  

### Pre-Acceptance Guards
- Blocks acceptance if the feature is currently “underpriced” and the seller recently traded under the floor within the last 24 hours. The controller checks the latest underpriced sell request and its trade timestamp, aborting with a localized `403` that includes a wait time.

```177:189:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
if ($feature->underPriced()) {
    $latestUnderPricedRequest = SellFeatureRequest::latestUnderPriceRequests($feature->owner, $feature)->last();
    if ($latestUnderPricedRequest) {
        $featureTrade = Trade::latestFeatureTrades($latestUnderPricedRequest->feature)->last();
        if ($featureTrade->created_at->addHours(24) > now()) {
            // ... existing code ...
            abort(403, 'شما در ۲۴ ساعت گذشته ملکی با زیر قیمت ۱۰۰٪ بفروش رسانده اید. برای پذیرش این درخواست باید ' . $elapsedTime . 'صبر کنید.');
        }
    }
}
```

### Settlement Workflow
1. Calls `releaseAsset()` to distribute funds:
   - Credits seller wallets with PSC/IRR net of fees.
   - Credits the platform (user code `hm-2000000`) with doubled fees.
   - Creates a `Trade` record and associated deposits, plus a `Comission`.
   - Marks buyer withdrawal transactions as settled (`status = 1`).
   - Cancels competing buy requests on the same feature (refunds and deletes).
2. Transfers feature ownership to the buyer and recalculates property metadata (RGB status, pricing, minimum price percentage based on buyer age).
3. Moves hourly profit accrual from seller to buyer, ensuring withdrawal deadlines respect buyer-specific variables.
4. Marks both parties as `traded()`, closes outstanding sell requests on the feature, and soft-deletes the accepted buy request (status set to `1` before deletion).
5. Dispatches `BuyFeatureNotification` and `sellFeature` notifications to both parties, then broadcasts `FeatureStatusChanged` with the new RGB.

```321:387:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
$seller->wallet->increment('psc', $psc_amount - $pscFee);
$seller->wallet->increment('irr', $irr_amount - $irrFee);

$trade = Trade::create([
    'feature_id' => $buyFeatureRequest->feature->id,
    'buyer_id' => $buyer->id,
    'seller_id' => $seller->id,
    'irr_amount' => $buyFeatureRequest->price_irr,
    'psc_amount' => $buyFeatureRequest->price_psc,
    'date' => now()
]);

$this->cancelOthereRequests($buyFeatureRequest);
```

### Response
- Returns the (now soft-deleted) `BuyFeatureRequest` resource snapshot for client confirmation.

### Error Modes
- `401` / `403` — Authentication or policy violations, including underpriced cooldowns or missing account security session.
- `404` — Offer not found / not accessible.
- `409` — Business logic conflicts (e.g., hourly profit entry missing) surface as generic 500 unless guarded elsewhere.
- `500` — Wallet persistence, trade creation, or notification/broadcast failures.

## `POST /api/buy-requests/reject/{buyFeatureRequest}` — Decline Offer
**Authentication:** Required.  
**Authorization:** `can:reject,buyFeatureRequest` uses `BuyFeatureRequestPolicy::reject`, limiting the action to the seller on the request.  

### Behavior
- Refunds escrowed PSC/IRR back to the buyer.
- Deletes linked transactions, the `lockedAsset`, and finally the buy request.
- Returns an empty response with status code 200 (`response()->noContent(200)`).

```273:288:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
$buyer->wallet->increment('psc', $psc_amount);
$buyer->wallet->increment('irr', $irr_amount);

$buyFeatureRequest->transactions()->delete();
$buyFeatureRequest->lockedAsset->delete();
$buyFeatureRequest->delete();
return response()->noContent(200);
```

### Error Modes
- `401` / `403` — Auth or policy failures.
- `404` — Offer not found or already processed.
- `500` — Wallet or database update failures.

## `DELETE /api/buy-requests/delete/{buyFeatureRequest}` — Buyer Cancels Offer
**Authentication:** Required.  
**Authorization:** `can:delete,buyFeatureRequest` defers to `BuyFeatureRequestPolicy::delete`, ensuring only the buyer who created the request can cancel it.  

### Behavior
- Mirrors the reject flow but intended for the buyer to withdraw their own offer.
- Responds with HTTP 204 (`response()->noContent()`).

```298:313:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
$buyer->wallet->increment('psc', $psc_amount);
$buyer->wallet->increment('irr', $irr_amount);

$buyFeatureRequest->transactions()->delete();
$buyFeatureRequest->lockedAsset->delete();
$buyFeatureRequest->delete();
return response()->noContent();
```

### Error Modes
- `401` / `403` — Authentication, verification, or missing account security session.
- `404` — Offer not found or inaccessible.
- `500` — Wallet/DB rollback errors.

### Example
```bash
curl -X DELETE "https://example.com/api/buy-requests/delete/54721" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json"
```

## `POST /api/buy-requests/add-grace-period/{buyFeatureRequest}` — Extend Deadline
**Authentication:** Required.  
**Authorization:** `can:addGracePeriod,buyFeatureRequest` is gated by `BuyFeatureRequestPolicy::addGracePeriod`, which allows only the seller to extend grace time while the request `status` is still `0`.  

### Request Body
| Field | Type | Rules |
| --- | --- | --- |
| `grace_period` | integer | required, between 1 and 30 |

```417:427:app/Http/Controllers/Api/V1/Feature/BuyRequestsController.php
$request->validate([
    'grace_period' => 'required|integer|min:1|max:30'
]);

$buyFeatureRequest->update([
    'requested_grace_period' => now()->addDays($request->integer('grace_period'))
]);
```

### Behavior
- Validates range, checks authorization, then sets `requested_grace_period` to a future timestamp relative to the current time.
- Returns an empty JSON response (`{}`) with status 200 on success.

### Error Modes
- `401` / `403` — Authentication, verification, account security, or policy failures.
- `404` — Offer not accessible.
- `422` — Validation failure if `grace_period` out of bounds.
- `500` — Database update failure.

## Side Effects & Integrations
- **Escrow Accounting:** The controller stores fee-inclusive amounts in `lockedwallet` and `transactions`, enabling later refund or release. Client UIs should surface the fee impact so balances reconcile with backend deductions.
- **Realtime Updates:** Accepting an offer broadcasts `FeatureStatusChanged`, which map clients should subscribe to for updating feature availability.
- **Notifications:** Buyers and sellers receive notifications at key milestones (creation, acceptance) enabling in-app and push workflows.
- **Commissions:** Accepting an offer doubles the fee credited to the `rgb` system user and records a `Comission`, ensuring revenue reports stay aligned with wallet movements.
- **Cooldown Enforcement:** Underpriced acceptance checks prevent sellers from rapidly liquidating multiple features below 100% within a 24-hour window.
- **Grace Period UX:** Since the server stores `requested_grace_period` as a timestamp, clients should display the formatted Jalali date from the resource rather than re-deriving it.

## Operational Notes
- **Account Security Sessions:** Mutating endpoints require the temporary `account.security` unlock; ensure clients refresh the session token shortly before calling store/accept/reject/destroy/add-grace-period.
- **Balance Synchronization:** Because PSC and IRR wallets are debited with fees immediately, clients should poll wallet endpoints post-offer to show reduced balances even while the offer is pending.
- **Idempotency Considerations:** Repeated `store` calls create multiple buy requests and duplicate fund locks; disable submit buttons or add client-side deduplication if necessary.
- **Localization:** Error responses (especially floor price and cooldown messages) are delivered in Farsi; surfaces should respect or translate these strings rather than substituting generic text.


