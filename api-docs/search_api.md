# Search API Guide

## Summary
- `POST /api/search/users` returns up to five user matches by splitting the input `searchTerm` into words and checking names, codes, and related KYC-first/last names.
- `POST /api/search/features` looks up at most five feature property rows whose IDs or addresses partially match the provided `searchTerm`, returning feature metadata and geometry coordinates.
- `POST /api/search/isic-codes` performs a simple `LIKE` query over the `isic_codes` table and returns every match in a `{ data: [...] }` envelope.
- All three routes sit outside the authenticated middleware group; they do not require a signed-in session or verified email.

## Route Registry
| Method | Path | Middleware | Controller action | Notes |
| --- | --- | --- | --- | --- |
| POST | `/api/search/users` | _None_ | `SearchController@users` | Splits `searchTerm` on spaces and ORs each token across user and KYC fields. |
| POST | `/api/search/features` | _None_ | `SearchController@features` | Performs a case-insensitive partial match on `FeatureProperties.id` and `address`. |
| POST | `/api/search/isic-codes` | _None_ | `SearchController@isicCodes` | Queries the `isic_codes` table directly and returns `id`, `name`, and `code`. |

Registered routes in `routes/api.php`:

```260:263:routes/api.php
Route::controller(SearchController::class)->prefix('search')->group(function () {
    Route::post('users', 'users');
    Route::post('features', 'features');
    Route::post('isic-codes', 'isicCodes');
});
```

## Endpoint Behavior

### `POST /api/search/users`
- **Request body:** expects JSON with a `searchTerm` string (no validation enforces it; see _Validation Rules_).
- **Query logic:** The controller splits the term on spaces, then creates OR conditions across `users.name`, `users.code`, and KYC `fname`/`lname`. A maximum of five results are returned, eager-loading profile photos and limited KYC columns.

```24:45:app/Http/Controllers/Api/V1/SearchController.php
public function users(Request $request): AnonymousResourceCollection
{
    $searchTerms = explode(' ', $request->searchTerm);

    $users = User::where(function ($query) use ($searchTerms) {
        foreach ($searchTerms as $term) {
            $query->orWhere('name', 'like', '%' . $term . '%')
                ->orWhere('code', 'like', '%' . $term . '%');
        }
    })
        ->orWhereHas('kyc', function ($query) use ($searchTerms) {
            $query->where(function ($query) use ($searchTerms) {
                foreach ($searchTerms as $term) {
                    $query->orWhere('fname', 'like', '%' . $term . '%')
                        ->orWhere('lname', 'like', '%' . $term . '%');
                }
            });
        })
        ->with(['profilePhotos', 'kyc:user_id,fname,lname'])
        ->take(5)
        ->get();
    return SearchUserResultResource::collection($users);
}
```

- **Response shape:** `SearchUserResultResource` exposes user-facing fields and uppercases `code`. `name` uses KYC data when the user is verified, and `followers` counts the relationship size at serialization time.

```18:25:app/Http/Resources/SearchUserResultResource.php
return [
    'id' => $this->id,
    'code' => Str::upper($this->code),
    'name' => $this->verified() ? $this->kyc->fname . ' ' . $this->kyc->lname : $this->name,
    'followers' => $this->followers->count(),
    'level' => $this->latest_level?->name,
    'photo' => $this->profilePhotos->last()?->url,
];
```

- **Example request:**
```bash
curl -X POST https://{host}/api/search/users \
  -H 'Content-Type: application/json' \
  -d '{"searchTerm": "john doe"}'
```

### `POST /api/search/features`
- **Request body:** expects JSON with `searchTerm` string.
- **Query logic:** Applies a case-insensitive `LIKE` comparison to both the string primary key (`FeatureProperties.id`) and `address`, limits to five rows, and eager loads the owning feature, its owner, and nested geometry coordinates.

```52:59:app/Http/Controllers/Api/V1/SearchController.php
public function features(Request $request)
{
    $features = FeatureProperties::where('id', 'like', '%' . $request->searchTerm . '%')
        ->orWhere('address', 'like', '%' . $request->searchTerm . '%')
        ->with(['feature', 'feature.owner', 'feature.geometry.coordinates'])
        ->take(5)
        ->get();
    return SearchFeatureResultResource::collection($features);
}
```

- **Response shape:** `SearchFeatureResultResource` returns feature IDs, addresses, price fields, owner code (uppercased), and an array of `{ id, x, y }` coordinate objects.

```19:27:app/Http/Resources/SearchFeatureResultResource.php
return [
    'id' => $this->feature->id,
    'feature_properties_id' => Str::upper($this->id),
    'address' => $this->address,
    'karbari' => $this->feature->getApplicationTitle(),
    'price_psc' => $this->price_psc,
    'price_irr' => $this->price_irr,
    'owner_code' => Str::upper($this->feature->owner->code),
    'coordinates' => CoordinatesResource::collection($this->feature->geometry->coordinates),
];
```

- **Example request:**
```bash
curl -X POST https://{host}/api/search/features \
  -H 'Content-Type: application/json' \
  -d '{"searchTerm": "TEH-"}'
```

### `POST /api/search/isic-codes`
- **Request body:** expects JSON with `searchTerm` string.
- **Query logic:** Fetches all rows whose `name` contains the provided term and returns only `id`, `name`, and `code` fields. Unlike the other endpoints, this call does **not** cap the number of results.

```66:72:app/Http/Controllers/Api/V1/SearchController.php
public function isicCodes(Request $request): JsonResponse
{
    $isicCodes = DB::table('isic_codes')
        ->where('name', 'like', '%' . $request->searchTerm . '%')
        ->select('id', 'name', 'code')
        ->get();
    return response()->json(['data' => $isicCodes]);
}
```

- **Example request:**
```bash
curl -X POST https://{host}/api/search/isic-codes \
  -H 'Content-Type: application/json' \
  -d '{"searchTerm": "manufacturing"}'
```

## Authorization & Policies
- **Authentication:** None of the search routes inherit the `auth:sanctum`, `verified`, or `activity` middleware stack. Any caller (including unauthenticated clients) can invoke them.
- **Policies:** The controller does not call `$this->authorize()` or `Gate::allows()`, and no resource-specific policy is registered. This means all data returned is solely filtered by the query conditions. Maintain awareness that `SearchUserResultResource` exposes follower counts and the latest level for each matched profile to anonymous callers.

## Validation Rules
- No route uses a `FormRequest` or inline `$request->validate(...)` call. Missing or non-string `searchTerm` values fall through to the controller, which relies on PHP’s dynamic typing.
- Because `explode()` requires a string, omitting `searchTerm` (or sending a non-string null value) in `/api/search/users` raises a `TypeError`, producing an unhandled 500 response. Clients must always provide a non-empty string.
- The feature and ISIC code endpoints coerce the provided value into the SQL `LIKE` expressions. An empty string effectively returns the first five feature rows or the full ISIC table respectively.

## Response Examples

### Users
```json
{
  "data": [
    {
      "id": 42,
      "code": "USR123",
      "name": "John Doe",
      "followers": 18,
      "level": "Citizen Level 3",
      "photo": "https://cdn.example.com/storage/photos/42/latest.jpg"
    }
  ]
}
```

### Features
```json
{
  "data": [
    {
      "id": 128,
      "feature_properties_id": "PROP-A12",
      "address": "Tehran, District 1, Example St.",
      "karbari": "Residential",
      "price_psc": "2.5",
      "price_irr": "3500000000",
      "owner_code": "CIT998",
      "coordinates": [
        { "id": 1, "x": 51.1234, "y": 35.6789 }
      ]
    }
  ]
}
```

### ISIC Codes
```json
{
  "data": [
    { "id": 101, "name": "Manufacture of textiles", "code": "1311" },
    { "id": 205, "name": "Manufacture of beverages", "code": "1104" }
  ]
}
```

## Implementation Notes & Edge Cases
- Each endpoint uses case-insensitive `LIKE` queries; database collation determines whether comparisons are accent/case sensitive.
- The users endpoint counts followers lazily, so each result triggers a `SELECT COUNT(*)` on the followers pivot unless the relationship is already cached. High-volume searches may exhibit N+1 behavior.
- Feature searches upper-case ID and owner code in the payload (`Str::upper()`), aligning with other API surfaces that treat codes as uppercase tokens.
- ISIC search returns the raw query array without pagination. Consider implementing a limit or pagination if the table grows large.


