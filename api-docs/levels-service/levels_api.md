# Levels API (v2)

The Levels API exposes read-only endpoints for browsing level metadata, detailed general information, and supporting assets (gems, gifts, licenses, and prizes). All endpoints are intended for public catalogue consumption and return JSON payloads.

- **Base path:** `/api/v2/levels`
- **Primary controller:** `App\Http\Controllers\Api\V2\LevelController`
- **Middleware:** None (public routes)

## Policy Rules & Access Control

- The routes are registered outside the authenticated `Route::middleware(['auth:sanctum', 'verified'])` group, so no authentication or verification is required.
- No Laravel `Policy` class or gate is associated with `Level` models. Access control relies solely on the absence of middleware and on route-model binding.
- Standard Laravel implicit route bindings enforce existence checks on the `{level:slug}` parameter and will return `404 Not Found` for unknown slugs.

```61:68:routes/api_v2.php
Route::controller(LevelController::class)->prefix('levels')->group(function () {
    Route::get('/', 'index');
    Route::get('/{level:slug}', 'show');
    Route::get('/{level:slug}/general-info', 'getGeneralInfo');
    Route::get('/{level:slug}/gem', 'gem');
    Route::get('/{level:slug}/gift', 'gift');
    Route::get('/{level:slug}/licenses', 'licenses');
    Route::get('/{level:slug}/prize', 'prizes');
});
```

## Validation & Binding Rules

- No request bodies or query parameters are accepted; all routes are read-only.
- `{level:slug}` uses implicit binding to `App\Models\Levels\Level`, guaranteeing a 404 when the slug is unknown.
- Nested resource lookups (`general-info`, `gem`, `gift`, `licenses`, `prize`) rely on the same resolved `Level` instance. Missing related records yield `null` payloads within the resource rather than validation errors.

```23:51:app/Http/Controllers/Api/V2/LevelController.php
public function show(Level $level)
{
    $level->load('image', 'generalInfo');
    return new LevelResource($level);
}

public function getGeneralInfo(Level $level)
{
    return new GeneralInfoResource($level->generalInfo);
}

public function gem(Level $level)
{
    return new GemResource($level->gem);
}

// ... existing code ...
```

## Shared Response Models

### `LevelResource`

```17:33:app/Http/Resources/V2/Level/LevelResource.php
return [
    'id' => $this->id,
    'name' => $this->name,
    'slug' => $this->slug,
    'image' => $this->whenLoaded('image', config('app.admin_panel_url') . '/uploads/' . $this->image->url),
    'background_image' => $this->whenNotNull($this->background_image),
    'general_info' => $this->whenLoaded('generalInfo', function () {
        return [
            'score' => $this->generalInfo->score,
            'rank' => $this->generalInfo->rank,
            'png_file' => $this->generalInfo->png_file,
            'fbx_file' => $this->generalInfo->fbx_file,
            'gif_file' => $this->generalInfo->gif_file,
            'description' => $this->generalInfo->description,
        ];
    })
];
```

- `image` expands to an absolute URL using `config('app.admin_panel_url')` when the polymorphic `image` relation is eager-loaded.
- `general_info` nests a subset of fields from `LevelGeneralInfo` when that relation is loaded (`show` endpoint only).
- `background_image` surfaces directly from the `levels` table when present.

### `GeneralInfoResource`

```17:33:app/Http/Resources/V2/Level/GeneralInfoResource.php
return [
    'id' => $this->id,
    'score' => $this->score,
    'description' => $this->description,
    'rank' => $this->rank,
    'subcategories' => $this->subcategories,
    'persian_font' => $this->persian_font,
    'english_font' => $this->english_font,
    'file_volume' => $this->file_volume,
    'used_colors' => $this->used_colors,
    'points' => $this->points,
    'lines' => $this->lines,
    'has_animation' => $this->has_animation,
    'designer' => $this->designer,
    'model_designer' => $this->model_designer,
    'creation_date' => $this->creation_date,
];
```

- Timestamp columns are hidden at the model level, so they never appear in responses.
- All attributes map 1:1 with the `level_general_infos` table, exposing typography, geometry, and design metadata.

### `GemResource`, `GiftResource`, `LicensesResource`

```15:17:app/Http/Resources/V2/Level/GemResource.php
return parent::toArray($request);
```

- These resources defer to the base `JsonResource` implementation, returning every non-hidden column from their respective tables (`level_gems`, `level_gifts`, `level_licenses`).
- Each model hides `created_at` and `updated_at`, ensuring clean payloads.

```12:18:app/Models/Levels/LevelGem.php
protected $hidden = [
    'created_at',
    'updated_at',
];

protected $guarded = [];
```

### `PrizeResource`

```17:27:app/Http/Resources/V2/Level/PrizeResource.php
return [
    'id' => $this->id,
    'level_id' => $this->level_id,
    'psc' => $this->psc,
    'yellow' => $this->yellow,
    'blue' => $this->blue,
    'red' => $this->red,
    'effect' => $this->effect,
    'satisfaction' => number_format($this->satisfaction, 2),
    'created_at' => jdate($this->created_at)->format('Y/m/d H:i:s'),
];
```

- `satisfaction` is rounded to two decimal places to stabilize downstream UI rendering.
- `created_at` is localized via the `jdate` helper into `Y/m/d H:i:s` format.

## Endpoint Overview

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/v2/levels` | Public | Lists every level with basic metadata and optional image URLs. |
| `GET` | `/api/v2/levels/{level:slug}` | Public | Returns a single level with general info and image when available. |
| `GET` | `/api/v2/levels/{level:slug}/general-info` | Public | Provides the full `LevelGeneralInfo` record for the specified level. |
| `GET` | `/api/v2/levels/{level:slug}/gem` | Public | Returns gem configuration data associated with the level. |
| `GET` | `/api/v2/levels/{level:slug}/gift` | Public | Returns the gift configuration for the level. |
| `GET` | `/api/v2/levels/{level:slug}/licenses` | Public | Returns licensing metadata for the level. |
| `GET` | `/api/v2/levels/{level:slug}/prize` | Public | Returns prize thresholds and satisfaction metrics for the level. |

## Endpoints

### GET `/api/v2/levels`

- **Controller method:** `LevelController@index`
- **Route binding:** None; returns all levels ordered by default primary key.
- **Validation:** Not applicable.
- **Response:** `200 OK` with `LevelResource` collection containing `id`, `name`, `slug`, optional `image`, and `background_image`.
- **Notes:** Images are only included if the polymorphic `image` relation is populated (`select` + `with('image')`).

### GET `/api/v2/levels/{level:slug}`

- **Controller method:** `LevelController@show`
- **Route binding:** Resolves the `Level` by `slug`. Unknown slugs produce `404`.
- **Validation:** Not applicable.
- **Response:** `200 OK` with a single `LevelResource`. The controller eager-loads `image` and `generalInfo`, so nested general info fields are populated when the relation exists.
- **Notes:** If `generalInfo` or `image` records are missing, the corresponding keys return `null` or are omitted due to the resource’s `whenLoaded` helpers.

### GET `/api/v2/levels/{level:slug}/general-info`

- **Controller method:** `LevelController@getGeneralInfo`
- **Route binding:** Reuses the bound `Level`. Missing general info returns `null` fields.
- **Validation:** Not applicable.
- **Response:** `200 OK` with `GeneralInfoResource`, exposing typography, scoring, and design metadata.
- **Notes:** Consumers should tolerate empty strings or `null` for optional fields like `persian_font`.

### GET `/api/v2/levels/{level:slug}/gem`

- **Controller method:** `LevelController@gem`
- **Route binding:** Reuses the bound `Level` model.
- **Validation:** Not applicable.
- **Response:** `200 OK` with `GemResource`. All non-hidden columns from `level_gems` are emitted.
- **Notes:** Expect numeric gem counts and color/channel configuration fields exactly as stored.

### GET `/api/v2/levels/{level:slug}/gift`

- **Controller method:** `LevelController@gift`
- **Route binding:** Reuses the bound `Level` model.
- **Validation:** Not applicable.
- **Response:** `200 OK` with `GiftResource`. Contains gift entitlement fields as stored in `level_gifts`.
- **Notes:** Payload mirrors database columns; no transformation beyond hidden timestamps.

### GET `/api/v2/levels/{level:slug}/licenses`

- **Controller method:** `LevelController@licenses`
- **Route binding:** Reuses the bound `Level` model.
- **Validation:** Not applicable.
- **Response:** `200 OK` with `LicensesResource`, exposing licensing booleans and metadata from `level_licenses`.
- **Notes:** Useful for gating feature access in clients; ensure consumers handle `null` when a license record has not been created.

### GET `/api/v2/levels/{level:slug}/prize`

- **Controller method:** `LevelController@prizes`
- **Route binding:** Reuses the bound `Level` model.
- **Validation:** Not applicable.
- **Response:** `200 OK` with `PrizeResource`, including PSC thresholds, color coded prize counts, `effect`, decimal `satisfaction`, and localized `created_at`.
- **Notes:** Dates appear in Jalali format via `jdate`. Front-ends expecting Gregorian should convert accordingly.

## Data Notes & Testing Checklist

- Confirm that requested slugs resolve correctly and return `404` for unknown entries.
- Validate that optional relations (image, general info, gem, gift, licenses, prize) can be absent without causing errors; clients must handle `null` payloads.
- Verify that `satisfaction` in prize responses is string-formatted with two decimals and `created_at` uses `Y/m/d H:i:s` Jalali formatting.
- Check that resource payloads remain stable when database schemas add columns—the `Gem`, `Gift`, and `Licenses` resources automatically include new non-hidden fields.

