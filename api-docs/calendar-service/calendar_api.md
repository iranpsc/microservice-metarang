# Calendar API

This document describes the public calendar endpoints exposed by the Laravel API. The controller that backs these routes is `App\Http\Controllers\Api\V1\CalendarController`.

- Base path: `/api/calendar`
- Available without authentication unless noted.
- Dates in requests must be Shamsi (Jalali) strings understood by the custom `shamsi_date` validator (for example `1403/07/01`). Responses return Jalali-formatted strings via `jdate()`.

## Authentication Summary

- `GET` endpoints are public. When a user is authenticated with Sanctum, the API augments responses with per-user interaction data.
- `POST /api/calendar/events/{event}/interact` requires a valid Sanctum bearer token.

## Resource Shape

Calendar responses use `App\Http\Resources\EventResource`. The payload differs for regular events versus version entries:

- Common fields: `id`, `title`, `description`, `starts_at` (`Y/m/d H:i` Jalali).
- Event entries (`is_version = 0`) add: `ends_at`, `views`, `btn_name`, `btn_link`, `color`, `image`, `likes`, `dislikes`, and when authenticated `user_interaction` with `has_liked`/`has_disliked` booleans.
- Version entries (`is_version = 1`) expose `version_title`. Counts and interaction data are omitted.

## Endpoints

### List Calendar Items

`GET /api/calendar`

| Query param | Type | Default | Description |
| ----------- | ---- | ------- | ----------- |
| `type` | string | `event` | `event` for regular events, `version` for application version announcements. Any other value falls back to the default. |
| `search` | string | `''` | Filters events whose `title` contains the provided substring. |
| `date` | string | `null` | Jalali date. When present, returns all entries spanning the given day. Results are returned as a collection (no pagination) in descending order. |

Behavior:

- Without `date`, results are `simplePaginate()`d. Expect the usual Laravel pagination wrapper (`data`, `links`, `meta`).
- With `date`, the controller restricts to entries whose `starts_at`–`ends_at` range covers the provided day. Includes `likes`, `dislikes`, and `views` counts, with optional `user_interaction`.

### Retrieve Single Calendar Entry

`GET /api/calendar/{event}`

- `event` is a `Calendar` model id.
- Loads like/dislike/view counts every time.
- If authenticated, includes the caller’s interaction state.
- Response body is a single `EventResource`.

### Interact With an Event

`POST /api/calendar/events/{event}/interact`

- Requires Sanctum authentication.
- Request body (JSON form):

```
{
  "liked": 1 | 0 | -1
}
```

Meaning:

- `1`: like the event.
- `0`: dislike the event.
- `-1`: remove any existing interaction.

On success, returns the updated `EventResource`, refreshed with like/dislike counts and the caller’s `user_interaction`.

### Latest Version Title

`GET /api/calendar/latest-version`

- Returns the newest version entry (ordered by `starts_at`) as a minimal JSON payload:

```
{
  "data": {
    "version_title": "v1.2.3"
  }
}
```

- `version_title` is `null` when no version entries exist.

### Filter Events by Date Range

`GET /api/calendar/filter`

| Query param | Type | Required | Notes |
| ----------- | ---- | -------- | ----- |
| `start_date` | string | yes | Jalali date string. |
| `end_date` | string | yes | Jalali date string. Must be the same day or after `start_date`. |

The endpoint returns only non-version events that overlap the requested range. Each item is a compact structure:

```
{
  "data": [
    {
      "id": 123,
      "title": "New Year Promotion",
      "starts_at": "1403/01/01",
      "ends_at": "1403/01/05",
      "color": "#FFAA00"
    }
  ]
}
```

Notes:

- Overlap logic covers events that start, end, or span entirely within the provided range.
- Dates are Jalali (`Y/m/d`). No time component is returned.
- The response is an unpaginated array ordered by `latest()` (descending `created_at`).

## Pagination & Counts

- Listing endpoints eager-load like/dislike/view counts via `withCount`.
- When authenticated, the controller eager-loads `userInteraction` to prevent N+1 queries. The resource converts this relationship into booleans (`has_liked`, `has_disliked`).
- View counts increase elsewhere (`Calendar::incrementViews()`); listing endpoints do not mutate state.

## Error Handling

- Validation errors (e.g., missing or malformed dates) return standard Laravel `422` JSON responses with validation messages.
- Requests to `interact` without authentication yield `401 Unauthorized`.
- Interactions on non-existent events respond with `404` due to route-model binding.

## Related Code

- Controller: `app/Http/Controllers/Api/V1/CalendarController.php`
- Model & scopes: `app/Models/Calendar.php`
- Resource transformer: `app/Http/Resources/EventResource.php`

