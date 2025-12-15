# Build Feature API (v2)

## Overview
- Manage the lifecycle of a building tied to a feature (parcel) within MetaRGB.
- Covers requesting available build packages, starting construction, retrieving construction state, updating active builds, and removing constructed models.
- All routes live under `Route::prefix('features')` in `routes/api_v2.php` and require the v2 stack (`/api/v2` prefix when served by Laravel).

## Authentication & Permissions
- Middleware: all routes run behind `auth:sanctum` and `verified`. Anonymous access is not possible.
- Route model binding is used for both `feature` and `buildingModel`. The `buildingModel` parameter resolves by `model_id`.
- Policy hook: `FeaturePolicy@build` (invoked through `$this->authorize('build', [$feature, $buildingModel])`) is currently permissive (`true`), but request-level authorization guards real access:
  - `StartBuildingFeatureRequest::authorize()` ensures the authenticated user owns the feature and no building is already attached.
  - `UpdateBuildingFeatureRequest::authorize()` ensures the authenticated user owns the feature being modified.
- `getBuildPackage()` performs an ownership check via `throw_unless($feature->owner->id === auth()->id(), AuthorizationException::class)`.

## Endpoint Summary

| Method | Route | Controller Method | Purpose |
| --- | --- | --- | --- |
| GET | `/api/v2/features/{feature}/build/package` | `getBuildPackage` | Fetch pre-calculated building package and feature coordinates. |
| POST | `/api/v2/features/{feature}/build/{buildingModel:model_id}` | `buildFeature` | Start construction of a building model on the feature. |
| GET | `/api/v2/features/{feature}/build/buildings` | `getBuildings` | List building model(s) attached to the feature. |
| PUT | `/api/v2/features/{feature}/build/buildings/{buildingModel:model_id}` | `updateBuilding` | Update construction details for an attached building. |
| DELETE | `/api/v2/features/{feature}/build/buildings/{buildingModel:model_id}` | `destroyBuilding` | Detach a building from the feature and reactivate hourly profits. |

## Endpoint Details

### GET `/api/v2/features/{feature}/build/package`
- **Purpose:** Request a build package from the external 3D Meta service for the selected feature.
- **Query Parameters:** `page` (optional, defaults to `1`). Forwarded to the remote API.
- **Behavior:**
  - Loads feature properties (`area`, `density`, `karbari`) and owner.
  - Rejects if the authenticated user is not the feature owner (`403`).
  - Calls the external API at `config('app.three_d_meta_url') . '/api/v1/build-package'`.
  - Enriches the remote response with:
    - `required_satisfaction` per model (`area * karbariCoefficient * density * 0.1 / 100`).
    - `feature.coordinates`: feature polygon points in `[x, y]` string format.
  - Persists or updates remote building models locally via `BuildingModel::upsert`.
- **Response Shape:** Mirrors the remote API payload with added `feature.coordinates` and augmented `data[].required_satisfaction`. On failure to reach 3D Meta, returns an error payload (`message`, `error`).

### POST `/api/v2/features/{feature}/build/{buildingModel:model_id}`
- **Purpose:** Start constructing a building model on the given feature.
- **Prerequisites:**
  - User must own the feature.
  - Feature must not already have a building (enforced by `StartBuildingFeatureRequest::authorize()`).
  - `buildingModel` must exist locally.
- **Validation:** See [Validation Rules](#validation-rules).
- **Behavior:**
  - Authorization uses `FeaturePolicy@build`.
  - Derives a construction duration: `buildingModel.required_satisfaction * 288000 / launched_satisfaction`.
  - Calculates end timestamp via `getConstructionEndDate()`.
  - Optionally creates an `IsicCode` record when `activity_line` is supplied.
  - Attaches the building through the pivot table (`building`) with metadata (start/end time, satisfaction spent, optional business info, rotation, position).
  - Sets all records in `FeatureHourlyProfit` for this feature to inactive.
  - Computes `bubble_diameter` from model attributes and updates the pivot.
- **Response:** Empty JSON with HTTP 200 on success.

### GET `/api/v2/features/{feature}/build/buildings`
- **Purpose:** Return all building models linked to the feature.
- **Response:** Collection of `BuildingModelResource`.

```24:36:app/Http/Resources/V2/BuildingModelResource.php
return [
    'id' => $this->id,
    'model_id' => $this->model_id,
    'name' => $this->name,
    'sku' => $this->sku,
    'images' => $this->images,
    'attributes' => $this->attributes,
    'file' => $this->file,
    'required_satisfaction' => number_format($this->required_satisfaction, 4),
    'building' => [
        'model_id' => $this->building->model_id,
        'feature_id' => $this->building->feature_id,
        'construction_start_date' => jdate(...)->format('Y/m/d H:i:s'),
        'construction_end_date' => jdate(...)->format('Y/m/d H:i:s'),
        'launched_satisfaction' => number_format(..., 4),
        'information' => $this->building->information,
        'rotation' => $this->building->rotation,
        'position' => $this->building->position,
        'bubble_diameter' => $this->building->bubble_diameter,
    ],
];
```

Dates are returned in Jalali format via `jdate()`.

### PUT `/api/v2/features/{feature}/build/buildings/{buildingModel:model_id}`
- **Purpose:** Update an existing building attachment (e.g., adjust satisfaction or metadata).
- **Prerequisites:** User owns the feature.
- **Validation:** Same rules as build creation (see below).
- **Behavior:**
  - Re-authorizes through `FeaturePolicy@build`.
  - Recomputes construction end date using the updated satisfaction value.
  - Updates pivot details (`construction_*`, `launched_satisfaction`, `information`, `rotation`, `position`).
- **Response:** Empty JSON with HTTP 200.

### DELETE `/api/v2/features/{feature}/build/buildings/{buildingModel:model_id}`
- **Purpose:** Remove the building association from the feature.
- **Behavior:**
  - Authorizes via policy.
  - Detaches the pivot record.
  - Reactivates `FeatureHourlyProfit` rows for the feature by setting `is_active` to `true`.
- **Response:** Empty JSON with HTTP 200.

## Validation Rules

### Shared Field Constraints

| Field | Type | Rules | Notes |
| --- | --- | --- | --- |
| `activity_line` | string | nullable, max 255 | Triggers `IsicCode::firstOrCreate`. |
| `name` | string | nullable, max 255 | Optional business name. |
| `address` | string | nullable, max 255 | Mailing or site address. |
| `postal_code` | string | nullable, `ir_postal_code` | Iran postal code validation. |
| `website` | string | nullable, `active_url`, max 255 | Must resolve via DNS. |
| `description` | string | nullable, max 5000 | Free-form description. |
| `launched_satisfaction` | numeric | required, min = `buildingModel.required_satisfaction`, max = authenticated user wallet satisfaction | Governs build duration and resource expenditure. |
| `rotation` | numeric | required | Rotation in degrees (assumed). |
| `position` | string | required, regex `^(-?\d+(\.\d+)?),\s*(-?\d+(\.\d+)?)$` | Comma-separated X,Y coordinates (allow negatives and decimals). |

### Request-Specific Authorization

```27:44:app/Http/Requests/StartBuildingFeatureRequest.php
return [
    'launched_satisfaction' => [
        'required',
        'numeric',
        'min:' . $this->route('buildingModel')->required_satisfaction,
        'max:' . $this->user()->wallet->satisfaction,
    ],
    'rotation' => 'required|numeric',
    'position' => [
        'required',
        'regex:/^(-?\d+(\.\d+)?),\s*(-?\d+(\.\d+)?)$/'
    ],
];
```

- `authorize()` additionally ensures only the feature owner can initiate a build and prevents duplicate active builds.

```27:44:app/Http/Requests/UpdateBuildingFeatureRequest.php
return [
    'launched_satisfaction' => [
        'required',
        'numeric',
        'min:' . $this->route('buildingModel')->required_satisfaction,
        'max:' . $this->user()->wallet->satisfaction,
    ],
    'rotation' => 'required|numeric',
    'position' => [
        'required',
        'regex:/^(-?\d+(\.\d+)?),\s*(-?\d+(\.\d+)?)$/'
    ],
];
```

- `authorize()` ensures only the owner may update existing builds.

## Data & Side Effects
- `BuildingModel::upsert()` caches remote models locally to reduce subsequent lookups.
- Pivot table (`building`) stores timestamps, satisfaction, optional business metadata, rotation, position, and computed bubble diameter.
- Starting a build deactivates `FeatureHourlyProfit` entries; deleting a build reactivates them.
- Calculated bubble diameter uses model attribute slugs: expects `width`, `length`, and `density` to be present in `BuildingModel.attributes`.

## External Dependencies
- 3D Meta service (`config('app.three_d_meta_url')/api/v1/build-package`) supplies the available building models.
- Jalali date formatting relies on the global `jdate()` helper.
- Postal codes validate against the custom `ir_postal_code` rule defined elsewhere in the application.

## Versioning & Future Considerations
- Routes are scoped under API v2; breaking changes should either extend this controller or create v3 counterparts.
- `FeaturePolicy@build` currently returns `true`; if stricter logic is required, update the policy while ensuring controller checks remain consistent.
- Consider caching or paginating `getBuildings` responses if the pivot expands to many records in future iterations.


