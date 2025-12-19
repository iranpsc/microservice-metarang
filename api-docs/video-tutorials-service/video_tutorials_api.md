# Video Tutorials API (v2)

The Video Tutorials API exposes read-only endpoints for discovering tutorial videos, categories, and subcategories, along with an authenticated interaction endpoint for recording likes and dislikes. A legacy helper endpoint exists in the v1 namespace for resolving modal content from a video URL fragment.

- **Base path:** `/api/tutorials`
- **Primary controller:** `App\Http\Controllers\Api\V1\TutorialController`
- **Middleware:** Grouped under `auth:sanctum`, `verified`, but most read endpoints explicitly opt out; see Policy Rules below.

## Policy Rules & Access Control

- **Public catalogue routes** (`GET` and `POST /search`) call `Route::withoutMiddleware(['auth:sanctum', 'verified'])`, so they are publicly reachable without authentication.
- **Interactions** (`POST /{video}/interactions`) inherit the group middleware; callers must present a valid Sanctum token tied to a verified user. Requests from unverified or unauthenticated accounts are rejected with `401`/`403`.
- **Modal lookup** (`POST /api/video-tutorials`) is defined in the v1 route file and is public.

No dedicated Laravel `Policy` class guards these routes; access control relies entirely on the middleware stack.

## Shared Response Models

### `VideoTutorialResource`

Tutorial listings and detail responses serialize with `App\Http\Resources\VideoTutorialResource`, providing creator, category, and media URLs:

```17:47:app/Http/Resources/VideoTutorialResource.php
return [
    'id' => $this->id,
    'title' => $this->title,
    'slug' => $this->slug,
    'image_url' => $this->image_url,
    // ... existing code ...
    'video_url' => $this->video_url,
    'created_at' => jdate($this->created_at)->format('Y/m/d')
];
```

- `creator` is included when the relationship is eager-loaded and surfaces `name`, `code`, and the latest profile photo.
- `category` and `sub_category` entries are present when the `subCategory` relationship is loaded.
- Counter fields (`views_count`, `likes_count`, `dislikes_count`) are populated when the resource is returned from queries that eager load or append the relevant morph count attributes (see `App\Models\Video`).

### `VideoCategoryResource`

Category routes use `App\Http\Resources\V2\VideoCategoryResource`, which expands nested subcategories and videos when they are eager loaded.

```19:32:app/Http/Resources/VideoCategoryResource.php
return [
    'id' => $this->id,
    'name' => $this->name,
    'slug' => $this->slug,
    'image' => $this->image_url,
    'icon' => $this->icon_url,
    // ... existing code ...
    'videos' => VideoTutorialResource::collection($this->whenLoaded('videos')),
];
```

### `VideoSubCategoryResource`

Subcategory responses are serialized through `App\Http\Resources\V2\VideoSubCategoryResource`, including counts and optional parent category metadata.

```18:37:app/Http/Resources/VideoSubCategoryResource.php
return [
    'id' => $this->id,
    'name' => $this->name,
    'slug' => $this->slug,
    'image' => $this->image_url,
    // ... existing code ...
    'videos' => VideoTutorialResource::collection($this->whenLoaded('videos'))
];
```

## Endpoint Overview

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| `GET` | `/api/tutorials` | Public | Paginated list of the latest tutorial videos. |
| `GET` | `/api/tutorials/{slug}` | Public | Returns a specific tutorial and increments its view counter. |
| `POST` | `/api/tutorials/search` | Public | Searches tutorials by title substring. |
| `POST` | `/api/tutorials/{video}/interactions` | `auth:sanctum`, `verified` | Records a like/dislike interaction for the authenticated user. |
| `GET` | `/api/tutorials/categories` | Public | Lists video categories with aggregate counters. |
| `GET` | `/api/tutorials/categories/{category:slug}` | Public | Shows a single category with its subcategories and counters. |
| `GET` | `/api/tutorials/categories/{category:slug}/videos` | Public | Returns paginated videos for the specified category. |
| `GET` | `/api/tutorials/categories/{category:slug}/{subCategory:slug}` | Public | Shows a subcategory with category reference and video listing. |
| `POST` | `/api/video-tutorials` | Public (v1) | Resolves modal metadata by matching a partial file name. |

Each endpoint section below includes validation rules and notable side effects.

## Endpoints

### GET `/api/tutorials`

- **Controller method:** `TutorialController@index`
- **Pagination:** Standard Laravel pagination (`page` query parameter). Page size fixed at 18.
- **Side effects:** None.
- **Response:** `200 OK` with `VideoTutorialResource` collection. Videos load `subCategory.category`, `creator.profilePhotos` (limited to one), and their counters.

### GET `/api/tutorials/{video:slug}`

- **Controller method:** `TutorialController@show`
- **Route binding:** Resolves a `Video` by slug.
- **Side effects:** Calls `Video::incrementViews()`, which creates a morph `views` record with the requester’s IP (`request()->ip()`).
- **Response:** `200 OK` with a single `VideoTutorialResource`.

### POST `/api/tutorials/search`

- **Controller method:** `TutorialController@search`
- **Validation:** `{ searchTerm: required|string }`
- **Behaviour:** Performs a case-insensitive `like` query against `videos.title`. Eager loads each video’s creator code, subcategory, category, and latest profile photo.
- **Response:** `200 OK` with a simplified array (not the resource class) including core counters, category/slugs, and creator image URL.
- **Errors:** `422 Unprocessable Entity` if `searchTerm` is missing or blank.

### POST `/api/tutorials/{video}/interactions`

- **Controller method:** `TutorialController@interactions`
- **Middleware:** Requires authenticated, verified user.
- **Validation:** `{ liked: required|boolean }`
  - The controller retrieves the value via `$request->query('liked')`; provide either a boolean JSON field or a query-string parameter (`?liked=1` or `?liked=true`).
- **Behaviour:** Upserts into `video->interactions()` (polymorphic `Interaction` model) scoped to the authenticated user, storing the `liked` flag and request IP.
- **Response:** `200 OK` with an empty JSON body.
- **Errors:** `422` for invalid boolean payloads, `401/403` when unauthenticated/unverified.

### GET `/api/tutorials/categories`

- **Controller method:** `TutorialController@getCategories`
- **Validation:** No explicit validation; supports optional `count` query parameter to override the page size (default 30).
- **Behaviour:** Loads `VideoCategory` models with counts for videos, views, likes, and dislikes, ordering descending by `likes_count`.
- **Response:** `200 OK` with a paginated `VideoCategoryResource` collection.

### GET `/api/tutorials/categories/{category:slug}`

- **Controller method:** `TutorialController@showCategory`
- **Behaviour:** Eager loads subcategories with their video/interaction counts (`withCount`) and attaches overall category counts.
- **Response:** `200 OK` with `VideoCategoryResource` (includes nested subcategories).

### GET `/api/tutorials/categories/{category:slug}/videos`

- **Controller method:** `TutorialController@showCategoryVideos`
- **Validation:** Optional `per_page` query parameter controls pagination size (default 18).
- **Behaviour:** Fetches the category’s videos ordered by recency, with creator info, subcategory, and count aggregations.
- **Response:** `200 OK` with `VideoTutorialResource` collection.

### GET `/api/tutorials/categories/{category:slug}/{subCategory:slug}`

- **Controller method:** `TutorialController@showSubCategory`
- **Behaviour:** Loads the subcategory’s videos (with creators and profile photos), parent category, and aggregate counts via `loadCount`.
- **Response:** `200 OK` with `VideoSubCategoryResource`.

### POST `/api/video-tutorials` (v1)

- **Controller method:** `TutorialController@showModalTutorial`
- **Validation:** `{ url: required|string }`
- **Behaviour:** Finds the first `Video` where `fileName` contains the provided `url` fragment, increments its view counter, and returns a flat data structure (id, title, description, media URLs, counters, creator code).
- **Response:** `200 OK` with `data` object.
- **Errors:** `404 Not Found` when no video matches; `422` when validation fails.

## Data & Side-Effect Summary

1. Views are tracked via the polymorphic `views()` relation. Both `show` and `showModalTutorial` call `incrementViews()`, which logs the requester IP.
2. Interactions persist in the `interactions()` morph relation, guaranteeing one like/dislike record per user via `updateOrCreate`.
3. Category and subcategory counters rely on Eloquent `withCount` to reflect real-time aggregates of likes, dislikes, views, and videos.

## Testing Checklist

- Confirm public routes respond without authentication and enforce validation (`422`) on missing payloads (`searchTerm`, `url`).
- Verify the interactions route rejects unauthenticated callers and accepts `liked=1/0`.
- Observe view counters increment after calling the detail or modal endpoints.
- Check that category pagination honors `count`/`per_page` overrides and that responses include nested resources as expected.

