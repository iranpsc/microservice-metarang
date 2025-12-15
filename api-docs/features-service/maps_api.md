# Maps API (v2)

The Maps API exposes read-only endpoints for retrieving map polygons, feature rollups, and border coordinates for client-side rendering and analytics.

- **Base path:** `/api/v2/maps`
- **Primary controller:** `App\Http\Controllers\Api\V2\MapsController`
- **Resource transformer:** `App\Http\Resources\V2\MapResource`
- **Route definitions:** `routes/api_v2.php`

```71:77:routes/api_v2.php
Route::apiResource('maps', MapsController::class)->only(['index', 'show']);

Route::controller(MapsController::class)->prefix('maps')->as('maps.')->group(function () {
    Route::get('/', 'index')->name('index');
    Route::get('/{map}', 'show')->name('show');
    Route::get('/{map}/border', 'showBorder');
});
```

## Policy Rules & Access Control

- All routes are registered outside the `Route::middleware(['auth:sanctum', 'verified'])` group in `api_v2.php`, so they are publicly accessible without authentication.
- There is no dedicated Laravel `Policy` guarding `App\Models\Map`; access relies entirely on the absence of authentication middleware and implicit route model binding.
- Requests resolve the `map` parameter via implicit binding. If a record is not found, Laravel automatically issues a `404 Not Found`.

## Validation Rules

- No request bodies are accepted; all endpoints are `GET`.
- The only input is the path parameter `map`, which must identify an existing `maps` table record. Invalid identifiers return a `404`.
- Because responses rely on eager-loaded relationships (`Map::with('features')`), database integrity should ensure every published map has related features to avoid division-by-zero warnings when computing `sold_features_percentage`.

## Endpoints

### GET `/api/v2/maps`

- **Controller action:** `MapsController@index`
- **Description:** Returns every map with its associated features preloaded so clients can render polygons and compute local statistics.
- **Query parameters:** None.
- **Response:** A JSON array of `MapResource` payloads. Each entry includes identifiers, polygon color, centroid coordinates, and the percentage of features with an owner other than the placeholder `owner_id = 1`.
- **Errors:** `500` if database access fails.

### GET `/api/v2/maps/{map}`

- **Controller action:** `MapsController@show`
- **Description:** Returns a single map enriched with polygon metadata, publication date, and summarized feature counts partitioned by usage types (`maskoni`, `tejari`, `amoozeshi`).
- **Path parameters:**
  - `map` (integer): Route-model-bound primary key.
- **Response:** A single `MapResource` document with additional keys (`border_coordinates`, `area`, `address`, `published_at`, `features` grouping) only present on this route.
- **Errors:** `404` if the map does not exist; `500` on server errors.

### GET `/api/v2/maps/{map}/border`

- **Controller action:** `MapsController@showBorder`
- **Description:** Returns just the border coordinates for lightweight clients needing polygon outlines without full metadata.
- **Response:** JSON object with a `data.border_coordinates` array copied from the `maps.border_coordinates` column.

```38:44:app/Http/Controllers/Api/V2/MapsController.php
    public function showBorder(Map $map)
    {
        return response()->json([
            'data' => [
                'border_coordinates' => $map->border_coordinates,
            ]
        ]);
    }
```

## Response Schema (`MapResource`)

`App\Http\Resources\V2\MapResource` shapes the payloads returned by both list and detail endpoints.

```19:150:app/Http/Resources/V2/MapResource.php
        return [
            'id' => $this->id,
            'name' => $this->name,
            'color' => $this->polygon_color,
            'central_point_coordinates' => $this->central_point_coordinates,
            'sold_features_percentage' => number_format($this->features->where('owner_id', '<>', 1)->count() / $this->features->count() * 100, 2),
            $this->mergeWhen(request()->routeIs('maps.show'), [
                'border_coordinates' => $this->border_coordinates,
                'area' => $this->polygon_area,
                'address' => $this->polygon_address,
                'published_at' => $this->publish_date,
                'features' => [
                    'maskoni' => [
                        'sold' => $this->features->where('owner_id', '<>', 1)->where(function ($query) {
                            $query->select('karbari')
                                ->from('feature_properties')
                                ->whereColumn('features.id', 'feature_properties.feature_id')
                                ->limit(1);
                        }, 'm')->count(),
                        // ... existing code ...
                    ],
                    'tejari' => [
                        // ... existing code ...
                    ],
                    'amoozeshi' => [
                        // ... existing code ...
                    ],
                ]
            ]),
        ];
```

- `sold_features_percentage` is returned as a string formatted with two decimal places (e.g., `"57.32"`).
- The grouped `features` statistics rely on correlated subqueries against `feature_properties`, `sell_feature_requests`, and `trades` tables; ensure these tables remain indexed for performance.
- Detail responses include `border_coordinates` and `central_point_coordinates` arrays suitable for GeoJSON-style mapping.

## Related Models & Data Dependencies

- `App\Models\Map` defines `hasMany` relationships to `features` and `crs`, and is the implicit binding target for the `{map}` parameter.
- Each map expects associated `features` records with `owner_id`, `feature_properties`, `sell_feature_requests`, and `trades` data available to support aggregation queries.
- The border-only endpoint surfaces the raw `maps.border_coordinates` column exactly as stored; clients must be prepared to handle the serialized coordinate format (commonly a GeoJSON polygon array).

## Error Handling

- Missing records return Laravel's default JSON `404` payload (`{"message":"No query results for model .../Map"}`).
- Unhandled exceptions bubble up to the global exception handler and respond with a generic `500`. There is no custom error transformation within the controller.


