# Profile Photos API Guide

## Summary
- `GET /api/profilePhotos` returns the authenticated user’s gallery in upload order, already transformed through `ProfilePhotoResource`.
- `POST /api/profilePhotos` accepts a single PNG or JPEG (≤1 MB), stores it on the public disk, and links it to the caller via the polymorphic `profilePhotos` relation.
- `GET /api/profilePhotos/{profilePhoto}` exposes the raw image metadata for any stored id (no ownership check).
- `DELETE /api/profilePhotos/{profilePhoto}` removes a photo, but only when the bound record belongs to the authenticated user; otherwise the request aborts with `403 Unauthorized`.

## Access Control
- **Middleware stack:** all routes reside inside the global `auth:sanctum`, `verified`, and `activity` middleware group configured in `routes/api.php`. Callers must present a verified Sanctum token and satisfy the `activity` tracker before hitting the controller.
- **Authorization rule:** deletion does not use a policy class; instead the controller enforces ownership inline via `abort_if($profilePhoto->imageable->isNot(request()->user()), 403, 'Unauthorized');`. Reads (`index`, `show`) and creates (`store`) rely solely on middleware.
- **Route model binding:** `{profilePhoto}` resolves to an `Image` model instance through implicit binding. A missing or non-numeric id yields `404 Not Found` before controller logic runs.

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| GET | `/api/profilePhotos` | `auth:sanctum`, `verified`, `activity` | `ProfilePhotoController@index` | List the caller’s stored profile photos. |
| POST | `/api/profilePhotos` | `auth:sanctum`, `verified`, `activity` | `ProfilePhotoController@store` | Upload a new profile photo for the caller. |
| GET | `/api/profilePhotos/{profilePhoto}` | `auth:sanctum`, `verified`, `activity` | `ProfilePhotoController@show` | Retrieve a single photo record by id. |
| DELETE | `/api/profilePhotos/{profilePhoto}` | `auth:sanctum`, `verified`, `activity` | `ProfilePhotoController@destroy` | Delete one of the caller’s photos (owner-only). |

> `Route::apiResource('profilePhotos', ...)` also registers `PUT/PATCH /api/profilePhotos/{profilePhoto}`, but the controller does not implement `update`, so these verbs resolve to a 404 “Not Found”.

## Data Model Reference
- **Resource shape:** profile photos serialize to `{ "id": <int>, "url": <string> }` through `ProfilePhotoResource`.

```17:20:app/Http/Resources/ProfilePhotoResource.php
        return [
            'id'    => $this->id,
            'url'   => $this->url,
        ];
```

- **Relationship:** `User::profilePhotos()` is a `morphMany` relation targeting the shared `Image` model. Uploads attach via this relation, and deletion checks the `imageable` back-reference.

```549:552:app/Models/User.php
    public function profilePhotos()
    {
        return $this->morphMany(Image::class, 'imageable');
    }
```

- **Image persistence:** uploads are saved to the `public` disk under the `profile` directory; the controller wraps the stored path with `url('uploads/...')`, so the returned URL points at `/uploads/profile/{filename}` on the CDN or app domain.

```30:33:app/Http/Controllers/Api/V1/ProfilePhotoController.php
        $url = url('uploads/'.$request->file('image')->store('profile', 'public'));
        $image = $request->user()->profilePhotos()->create(['url' => $url]);
        return new ProfilePhotoResource($image);
```

## Endpoint Details

### `GET /api/profilePhotos` – List Profile Photos
- **Authentication:** required (Sanctum, verified).
- **Response:** `200 OK` with `{ "data": [ { "id": 12, "url": "https://..." }, ... ] }`.
- **Behavior:** returns the current user’s entire collection ordered by the underlying relation (default insertion order).
- **Errors:** `401/403` for failed middleware; `500` if storage URLs are misconfigured.

### `POST /api/profilePhotos` – Upload New Photo
- **Authentication:** required.
- **Content type:** `multipart/form-data` with a single `image` file field.
- **Validation:** `image` is required, must be an actual image with MIME `png`, `jpg`, or `jpeg`, and the payload must be ≤1024 KB.

```30:31:app/Http/Controllers/Api/V1/ProfilePhotoController.php
        $request->validate(['image' => 'required|image|mimes:png,jpg,jpeg|max:1024']);
```

- **Storage flow:** the file is stored on the `public` disk (`storage/app/public/profile`), then exposed through `/uploads/profile/...` via `url()`. The resulting `Image` record is attached to the caller and returned as a `ProfilePhotoResource`.
- **Response:** `201 Created` with the serialized photo on success.
- **Errors:** `422 Unprocessable Entity` for validation failures; `401/403` middleware failures; `500` if the `public` disk is misconfigured.

### `GET /api/profilePhotos/{profilePhoto}` – Show Photo Metadata
- **Authentication:** required.
- **Response:** `200 OK` with the `ProfilePhotoResource` payload for the bound id.
- **Authorization:** no owner check is performed; any authenticated, verified user can fetch any `Image` row as long as they know the id. Handle sensitive exposure accordingly at the client or consider tightening authorization.
- **Errors:** `404 Not Found` if the id does not resolve; `401/403` middleware failures.

### `DELETE /api/profilePhotos/{profilePhoto}` – Delete Photo
- **Authentication:** required.
- **Authorization:** succeeds only when the bound `Image` belongs to the caller (`imageable` morph matches `request()->user()`). Otherwise the controller aborts with `403 Unauthorized`.
- **Response:** `204 No Content` on success.
- **Side effects:** only the database row is removed; the underlying file remains on disk unless another process handles cleanup.
- **Errors:** `403 Forbidden` for ownership mismatches; `404 Not Found` if the id is invalid; `401`/`403` middleware failures.

## Operational Notes
- Ensure the `public` filesystem disk is linked (`php artisan storage:link`) so `/uploads/profile/...` URLs are reachable.
- The controller does not throttle or cap the number of stored photos; clients should enforce their own limits if needed.
- Consider adding an authorization policy or additional middleware if exposing photo metadata across accounts is undesirable.

