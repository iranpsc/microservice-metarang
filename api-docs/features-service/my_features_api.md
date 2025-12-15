# My Features API Guide

## Summary
- `GET /api/my-features` returns a paginated slice of the authenticated citizen's owned features with basic property metadata.
- `GET /api/my-features/{user}/features/{feature}` expands a single feature with images, latest trade data, and geometry.
- `POST /api/my-features/{user}/add-image/{feature}` uploads one or more media assets and attaches them to a feature.
- `POST /api/my-features/{user}/remove-image/{feature}/image/{image}` deletes a previously uploaded feature image.
- `POST /api/my-features/{user}/features/{feature}` recalculates and persists pricing fields based on a submitted minimum price percentage.

All routes sit behind the `auth:sanctum`, `verified`, and `activity` middleware stack. Mutating endpoints additionally enforce `account.security` via the controller constructor, requiring an active account-security window in production.

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| GET | `/api/my-features` | `auth:sanctum`, `verified`, `activity` | `FeatureController@index` | List current user's features (5-per-page simple pagination). |
| GET | `/api/my-features/{user}/features/{feature}` | `auth:sanctum`, `verified`, `activity` | `FeatureController@show` | Fetch a scoped feature with images, properties, and latest trade. |
| POST | `/api/my-features/{user}/add-image/{feature}` | `auth:sanctum`, `verified`, `activity`, `account.security` | `FeatureController@addFeatureImages` | Attach uploaded images to a feature. |
| POST | `/api/my-features/{user}/remove-image/{feature}/image/{image}` | `auth:sanctum`, `verified`, `activity`, `account.security` | `FeatureController@removeFeatureImage` | Remove a single feature image. |
| POST | `/api/my-features/{user}/features/{feature}` | `auth:sanctum`, `verified`, `activity`, `account.security` | `FeatureController@updateFeature` | Update feature pricing thresholds via minimum percentage. |

Routes are registered inside the authenticated user group in `routes/api.php`. The `Route::scopeBindings()` wrapper guarantees nested parameters respect their relationships—`{feature}` must belong to the `{user}`, and `{image}` must be associated with the `{feature}`.

## Domain Model Snapshot
- **Feature** – Owns a `properties` relation (pricing, stability, region, etc.), `images`, optional `geometry` payload, and `latestTraded` transaction metadata.
- **Feature Properties** – Houses `price_psc`, `price_irr`, `stability`, and `minimum_price_percentage`. Pricing is recalculated when the minimum percentage changes.
- **Feature Images** – Lightweight records containing an `id` and public `url`. CRUD guarded by feature-level policies (`addImage`, `removeImage`).
- **Account Security window** – Managed separately (`AccountSecurityController`); must be unlocked before mutating routes succeed in production environments.

## `GET /api/my-features` – List Authenticated Features
**Authentication:** Required (`Bearer` token via Sanctum).  
**Query parameters:** Standard `?page=<n>` simple paginator control; page size is fixed at 5.

### Behavior
- Filters features by ownership (`Feature::whereBelongsTo($request->user(), 'owner')`).
- Eager loads `properties` to avoid N+1 reads.
- Returns an anonymous resource collection of `UserFeatureResource`, wrapped by Laravel's simple pagination metadata (`data`, `links`, `meta`).

### Success Response
```startLine:endLine:app/Http/Resources/UserFeatureResource.php
// ... existing code ...
```
Each `data[]` item exposes:
- `id`
- `properties` (see `FeaturePropertiesResource` for fields such as `price_psc`, `stability`, `minimum_price_percentage`)
- `images` (always empty on this endpoint because only `properties` is eager-loaded)
- `seller` and `geometry` are `null` unless the relations were explicitly loaded elsewhere

### Error Modes
- `401` – Missing or invalid Sanctum token.
- `403` – Authorization failure (for example, feature policy denies access).
- `500` – Unexpected server issues.

### Example
```bash
curl -X GET "https://example.com/api/my-features?page=2" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json"
```

## `GET /api/my-features/{user}/features/{feature}` – Show Feature
**Authentication:** Required (`Bearer` token via Sanctum).  
**Path params:**
- `user` – User UUID or numeric id resolved via route-model binding.
- `feature` – Feature id scoped to the provided `user`.

### Behavior
- Loads `properties`, `images`, and `latestTraded` relationships before wrapping the feature with `UserFeatureResource`.
- Policies ensure the caller can view the target feature; the route automatically 404s if `{feature}` is not owned by `{user}` due to scoped bindings.

### Response Payload Highlights
- `properties` – Address, density, pricing, and stability data.
- `images` – Array of `{ id, url }`.
- `seller` – Latest trade seller summary (`id`, `name`, `code`), nullable.
- `geometry` – `coordinates` array when the relation exists.

### Example
```bash
curl -X GET "https://example.com/api/my-features/42/features/73" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json"
```

## `POST /api/my-features/{user}/add-image/{feature}` – Upload Feature Images
**Authentication:** Required (`Bearer` token + unlocked account-security window).  
**Content type:** `multipart/form-data`.

### Validation (`FeatureImageRequest`)
| Field | Rules |
| --- | --- |
| `images` | `required|array|min:1` |
| `images.*` | `required|file|mimes:png,jpg,bmp|distinct|min:1|max:1024` (size in kilobytes) |

### Behavior
- Authorizes via `FeaturePolicy@addImage`.
- Stores each uploaded file on the `public` disk under `features/`, then persists an `Image` record with the fully-qualified public URL (`url('uploads/'.$path)`).
- Returns a collection of `FeatureImageResource` with the updated image list.

### Example
```bash
curl -X POST "https://example.com/api/my-features/42/add-image/73" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json" \
  -F "images[0]=@/path/to/front.png" \
  -F "images[1]=@/path/to/plan.jpg"
```

### Error Modes
- `401` – Unauthenticated.
- `403` – Account-security window locked or feature policy denial.
- `404` – Feature not owned by user or image relation mismatch.
- `422` – Validation failures (invalid mime types, duplicate files, oversized uploads).

## `POST /api/my-features/{user}/remove-image/{feature}/image/{image}` – Delete Image
**Authentication:** Required (`Bearer` + unlocked account-security window).  
**Behavior:**
- Authorizes via `FeaturePolicy@removeImage`, ensuring the image belongs to the feature.
- Deletes the `Image` model record; file removal from storage is not handled here (consider a queued job if required).
- Responds with HTTP 200 and an empty body (`response()->noContent(200)`).

### Error Modes
- `401` – Unauthenticated.
- `403` – Policy or account-security rejection.
- `404` – Image not bound to the feature or feature not bound to the user.

### Example
```bash
curl -X POST "https://example.com/api/my-features/42/remove-image/73/image/5" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json"
```

## `POST /api/my-features/{user}/features/{feature}` – Update Minimum Price Percentage
**Authentication:** Required (`Bearer` + unlocked account-security window).  
**Body:** JSON.

### Validation
| Field | Type | Rules | Notes |
| --- | --- | --- | --- |
| `minimum_price_percentage` | integer | `required|min:80` | Additional runtime check: if the user is under 18 (`$request->user()->isUnderEighteen()`), the value must be >= 110. |

### Behavior
- Authorizes via `FeaturePolicy@update`.
- Calculates new pricing using current stability and rate variables:
  - `totalPrice = stability × Variable::getRate(color) × minimum_price_percentage / 100`
  - Splits the total into PSC and IRR halves, using `Variable::getRate('psc')` for conversion.
- Updates the related `FeatureProperties` record fields (`price_psc`, `price_irr`, `minimum_price_percentage`).
- Returns HTTP 204 on success.

### Error Modes
- `401` – Unauthenticated.
- `403` – Policy failure or locked account-security window.
- `422` – Validation errors, including the `<110` rule for minors.
- `500` – Issues fetching rate variables or persisting the update.

### Example
```bash
curl -X POST "https://example.com/api/my-features/42/features/73" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"minimum_price_percentage": 125}'
```

## Operational Notes
- **Guard expectations:** All routes assume `auth:sanctum` tokens representing end users. Admin tokens without the relevant ownership relationships will resolve in 404/403 responses.
- **Scoped bindings:** Rely on the nested route order. When crafting API clients, always supply both `user` and nested identifiers from the listing endpoint to avoid binding failures.
- **Pagination UX:** `simplePaginate(5)` omits total counts; client UIs should rely on `links.next` to detect more pages.
- **Storage strategy:** Uploaded images are stored on the `public` disk; ensure `php artisan storage:link` has been executed and CDN caching policies respect updates.
- **Policy coverage:** Feature policies gate all mutating actions; ensure tests cover authorization edges (e.g., attempting updates on transferred features).


