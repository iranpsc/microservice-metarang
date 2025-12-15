# Features Marketplace API Guide

## Summary
- `GET /api/features` renders a geospatial slice of tradable features using bounding-box points, optionally including 3D building metadata. Open to unauthenticated callers when the `activity` middleware allows.
- `GET /api/features/{feature}` hydrates a single feature with pricing, imagery, seller context, hourly profit status, and optional construction snapshots.
- `POST /api/features/buy/{feature}` executes the feature purchase workflow, handling RGB-owned parcels, peer-to-peer sales, and special limited campaigns with automatic wallet movements, trade logging, and notifications.

These routes live inside the global `auth:sanctum`, `verified`, and `activity` stack, but `GET` operations explicitly remove the first two middleware and the account-security check, leaving only `activity` applied by default. The buy endpoint remains fully guarded and additionally leverages the `account.security` middleware and authorization policy `can:buy,feature`.

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| GET | `/api/features` | `activity` (global), optional caller token | `BuyFeatureController@index` | List features intersecting supplied coordinates; optionally marks ownership for authenticated callers. |
| GET | `/api/features/{feature}` | `activity` (global) | `BuyFeatureController@show` | Fetch a fully hydrated feature resource. |
| POST | `/api/features/buy/{feature}` | `auth:sanctum`, `verified`, `activity`, `account.security`, `can:buy,feature` | `BuyFeatureController@buy` | Purchase a feature from RGB, a peer seller, or within a limited campaign. |

Routes sit within the `Route::scopeBindings()` group, ensuring `{feature}` resolves consistently with policy expectations and owned relationships.

## Domain Model Snapshot
- **Feature** – Core tradable asset with relations to `properties`, `images`, `geometry.coordinates`, `hourlyProfit`, `buildingModels`, `latestTraded`, and `buyRequests` / `sellRequests`.
- **Feature Properties** – Holds pricing, stability, region, owner label, and the current `rgb` status flag (`for_sale`, `sold`, etc.).
- **Wallets** – Users (including RGB) expose `wallet` balances in color assets, `psc`, and `irr`. Purchases debit and credit these wallets following distinct flows.
- **FeatureLimit** – Campaign windows that restrict sale price behavior and per-user purchase counts for limited-feature runs. Joined by `LimitedFeaturePurchase` records.
- **Trade / Transaction / Comission** – Persist purchase history, ledger movements, and platform fees for auditing.
- **SystemVariable & Variable** – Provide dynamic tuning knobs (`public_pricing_limit`, `under_18_pricing_limit`, rate conversions) used during pricing recalculations.

## `GET /api/features` – Bounding Box Discovery
**Authentication:** Optional. Without a bearer token, results do not include `is_owned_by_auth_user`.  
**Content type:** `application/json`.  
**Validation (`FeatureRepository@all`):**

| Field | Type | Rules | Notes |
| --- | --- | --- | --- |
| `points` | array\<string\> | `required|min:4` | Four comma-separated points (`"x,y"`) defining the bounding box. |
| `points.*` | string | `regex:/^([0-9]+(\.[0-9]+)?,[0-9]+(\.[0-9]+)?)$/` | Ensures numeric X/Y pairs. Order is `[topLeft, topRight, bottomLeft, bottomRight]`. |
| `load_buildings` | boolean | `nullable` | When `true`, eager loads `buildingModels` plus pivot metadata (`construction_start_date`, `construction_end_date`, `rotation`, `position`). |
| `user_features_location` | boolean | `nullable` | Reserved flag; repository currently ignores it but callers should keep payload compatible. |

### Behavior
- Converts point strings into numeric coordinate pairs and determines intersecting `geometry_id` values via the `coordinates` table.
- Loads matching `Feature` rows, selecting:
  - `id`
  - `owner` (mapped from `owner_id`)
  - `properties:id,feature_id,rgb`
  - `geometry.coordinates:id,geometry_id,x,y`
- Adds `buildingModels` data when the flag is set.
- When a bearer token is present and resolves to an authenticated user, each feature includes `is_owned_by_auth_user` (boolean).

### Success Payload
Returns an array under `data`:

```startLine:endLine:app/Repositories/FeatureRepository.php
// ... existing code ...
```

Each feature object contains:
- `id`
- `owner` (owner user id)
- `properties` (`id`, `feature_id`, `rgb`)
- `geometry` (`coordinates` array of `{ id, geometry_id, x, y }`)
- Optional `building_models` array with pivot metadata if requested
- Optional `is_owned_by_auth_user`

### Error Modes
- `422` – Missing/invalid points array or malformed coordinates.
- `401` – Only when the caller supplies an invalid Sanctum token.
- `500` – Issues resolving coordinates or eager-loaded relations.

### Example
```bash
curl -X GET "https://example.com/api/features" \
  -H "Accept: application/json" \
  -d '{
        "points": ["51.328,35.732", "51.431,35.732", "51.328,35.667", "51.431,35.667"],
        "load_buildings": true
      }'
```

## `GET /api/features/{feature}` – Feature Detail
**Authentication:** Optional; guarded only by the global `activity` middleware.  
**Path params:**  
- `feature` – Integer ID resolved by implicit model binding; scoped to enforce existence but not ownership.

### Behavior
- Eager loads `properties`, `images`, `latestTraded.seller`, `hourlyProfit:id,feature_id,is_active`, and `buildingModels` with pivot data.
- Wraps the result using `FeatureResource`, exposing a consistent JSON envelope.

### Response Payload Highlights
- `id`, `owner_id`
- `properties` – Full `FeaturePropertiesResource` (`address`, `density`, `stability`, `price_psc`, `price_irr`, `minimum_price_percentage`, etc.).
- `images` – Array of `{ id, url }`.
- `seller` – Latest seller summary if a trade exists.
- `is_hourly_profit_active` – Boolean derived from the related hourly profit record (defaults to `false` if the relation is missing).
- `geometry` – Coordinates array when the `geometry` relation exists.
- `construction_status` – Array of building models with `model_id`, `name`, `file`, `images`, and status string (`completed` vs `in progress` based on `construction_end_date`).

### Error Modes
- `404` – Feature not found or filtered out by scoped bindings.
- `500` – Unexpected data-loading failures.

### Example
```bash
curl -X GET "https://example.com/api/features/18342" \
  -H "Accept: application/json"
```

## `POST /api/features/buy/{feature}` – Execute Purchase
**Authentication:** Required (`Bearer` token via Sanctum) with a current account-security session.  
**Authorization:** `can:buy,feature` policy gate.  
**Path params:**  
- `feature` – Target feature id scoped through bindings.

### High-Level Flow
1. Loads `properties` and `owner`.
2. Checks whether the feature is part of a limited campaign (`FeatureIndicators::MaskoniTradingLimited`, etc.). If yes, delegates to `handleLimitedFeature`.
3. If not limited and current owner is the platform's RGB account (`code=hm-2000000`), executes `buyFromRGB`.
4. Otherwise performs a peer-to-peer purchase via `buyFromUser`.

### Limited Feature Handling
- Looks up an active `FeatureLimit` covering the feature id; aborts with HTTP 400 if none is found, ensuring misconfigured campaigns cannot execute.
- Validates buyer color balance (`checkColorBalance`) when `price_limit` is enforced; aborts with HTTP 403 if the wallet lacks sufficient liters of the feature's color.
- Calculates price using `properties.stability` and the feature color.
- Debits buyer color balance and credits seller accordingly.
- Transfers ownership to the buyer and resets presentation fields (`label`, `owner`, `rgb` via `changeStatusToSoldAndNotPriced`).
- Sets `minimum_price_percentage` based on whether the buyer is under 18 (`under_18_pricing_limit` system variable falls back to 110, otherwise `public_pricing_limit` default 80).
- Optionally records an entry in `LimitedFeaturePurchase` to enforce per-user limits.
- Creates the `Trade` record plus a single `withdraw` transaction.
- Initializes an hourly profit record for the buyer, using their `withdraw_profit` variable to determine deadline.
- Broadcasts `FeatureStatusChanged` and notifies the buyer via `BuyFeatureNotification`.

### Buying from RGB
- Confirms the buyer's color wallet contains at least `properties.stability` liters through `checkColorBalance`; aborts with HTTP 403 when insufficient.
- Debits the buyer and credits the RGB wallet in the feature color.
- Transfers ownership and resets metadata similar to the limited flow.
- Records a trade with a single withdraw transaction (asset = feature color).
- Creates buyer hourly profit entry with `dead_line = now + withdraw_profit_days`.
- Notifies the buyer and broadcasts status change.

### Buying from Another User
- Detects under-priced offers: if the feature is marked `underPriced`, inspects the seller's latest under-priced request and most recent trade. A guard exists to block rapid trades within 24 hours, though the current implementation gathers timing data without emitting an abort—client teams should monitor future updates.
- Calls `$buyer->checkBalance($feature)` to ensure both `psc` and `irr` funds are available.
- `chargeBuyer()` withdraws `price_psc` and `price_irr` plus platform fees (`config('rgb.fee')` multiplier).
- `paySeller()` deposits net proceeds (price minus fee) into the seller's wallet.
- Creates a `Trade` with dual currency amounts and associated `withdraw` transactions for the buyer, plus deposit entries for seller proceeds.
- Credits the RGB wallet with doubled fee amounts (noting both the platform and state share).
- Persists a `Comission` record capturing platform earnings.
- Transfers ownership, resets UI fields, and updates `minimum_price_percentage` based on age bracket.
- Marks open sell requests from this seller as `status = 1` (closed) and iterates pending buy requests to refund locked assets and delete them.
- Flags both participants as having traded (`traded()`), migrates hourly profit ownership to the buyer, and triggers buyer/seller notifications (`BuyFeatureNotification`, `sellFeature`).
- Broadcasts `FeatureStatusChanged` with the updated RGB status.

### Error Modes
- `401` – Missing or invalid Sanctum token.
- `403` – Locked account security session, failed policy check, insufficient wallet balance, or age-based color deficit.
- `404` – Feature no longer meets binding/policy criteria.
- `400` – Limited feature purchased outside an active campaign.
- `422` – Validation failures surfaced by underlying wallet or policy checks.
- `500` – Database or notification failures during trade creation.

### Example
```bash
curl -X POST "https://example.com/api/features/buy/18342" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json"
```

Successful responses return the updated `FeatureResource` payload reflecting new ownership and state.

## Operational Notes
- **Account security cadence:** Ensure the account-security unlock workflow (`POST /api/account/security`) has run recently before calling the buy endpoint in production; otherwise expect HTTP 403.
- **Event listeners:** Purchases emit `FeatureStatusChanged`, enabling real-time map updates or websocket feeds. Clients should subscribe to maintain parity.
- **Fee configuration:** Platform fee multiplier comes from `config('rgb.fee')`; adjust carefully as it compounds into wallet flows, commissions, and RGB earnings.
- **Concurrency:** No explicit locking is applied; client applications should anticipate race conditions (e.g., retry strategies or optimistic UI updates when two buyers target the same feature).
- **Auditing:** Trade records, transactions, and commissions form the canonical ledger trail—use them for financial reconciliation and dispute resolution dashboards.


