# Video Tutorial Comments API (v2)

## Overview
- Manage comment threads for tutorial videos: listing, posting, editing, deleting, reporting, and reacting to comments.
- Public comment browsing is available, while all write operations require an authenticated, verified user under the v2 middleware stack.
- Authorization relies on `CommentPolicy` to gate edits, deletions, reports, and reactions to the appropriate actor.
- Responses return `VideoCommentResource` payloads, exposing comment metadata, interaction counts, reply status, and Jalali-formatted timestamps.

## Authentication & Authorization
- The route group is registered inside `routes/api_v2.php` and inherits the `/api` prefix applied to all v2 endpoints.
- `GET /api/tutorials/{video}/comments` removes both `auth:sanctum` and `verified`; any caller can read the first-level comments for a video.
- All other routes run behind `auth:sanctum` and `verified`, so callers must present a valid Sanctum token and pass email verification.
- Policy hooks enforced by `VideoCommentsController`:
  - `update` and `destroy` require ownership of the comment (`CommentPolicy@update` / `@delete`).
  - `interactions` invokes `CommentPolicy@like` or `@dislike`, which block self-reactions.
  - `report` leverages `CommentPolicy@report`, preventing users from reporting their own comments.

```41:48:routes/api_v2.php
Route::controller(VideoCommentsController::class)->prefix('tutorials')->group(function () {
    Route::get('/{video}/comments', 'index')->withoutMiddleware(['auth:sanctum', 'verified']);
    Route::post('/{video}/comments', 'store');
    Route::put('/{video}/comments/{comment}', 'update');
    Route::delete('/{video}/comments/{comment}', 'destroy');
    Route::post('/{video}/comments/{comment}/report', 'report');
    Route::post('/{video}/comments/{comment}/interactions', 'interactions');
});
```

```31:66:app/Policies/CommentPolicy.php
public function update(User $user, Comment $comment)
{
    return $comment->user->is($user);
}

public function report(User $user, Comment $comment)
{
    return $comment->user->isNot($user);
}
```

## Route Registry
| Method | Path | Middleware | Controller action | Notes |
| --- | --- | --- | --- | --- |
| GET | `/api/tutorials/{video}/comments` | none | `index` | Publicly lists top-level comments with like/dislike/reply counts. |
| POST | `/api/tutorials/{video}/comments` | `auth:sanctum`, `verified` | `store` | Authenticated users post new parent comments. |
| PUT | `/api/tutorials/{video}/comments/{comment}` | `auth:sanctum`, `verified` | `update` | Comment owners update their content. |
| DELETE | `/api/tutorials/{video}/comments/{comment}` | `auth:sanctum`, `verified` | `destroy` | Comment owners delete their comment; related interactions are purged. |
| POST | `/api/tutorials/{video}/comments/{comment}/report` | `auth:sanctum`, `verified` | `report` | Non-owners submit a report payload tied to the video. |
| POST | `/api/tutorials/{video}/comments/{comment}/interactions` | `auth:sanctum`, `verified` | `interactions` | Non-owners like or dislike a comment (toggle stored per user). |

## Response Shape
- Both list and single-item responses wrap results with `VideoCommentResource`. Fields include identifiers, user snapshot, interaction counters, reply state, and Jalali-formatted creation date.
- Collection responses from `index` use `simplePaginate(10)`, returning `links`/`meta` pagination keys.
- Nested replies are only included when the `replies` relationship is eager-loaded (not done in these endpoints, but supported by the resource).

```23:36:app/Http/Controllers/Api/V1/VideoCommentsController.php
$comments = $video->comments()
    ->with('user.latestProfilePhoto')
    ->withCount(['likes', 'dislikes', 'replies'])
    ->whereNull('parent_id')
    ->orderBy('likes_count', 'desc')
    ->simplePaginate(10);
```

```17:36:app/Http/Resources/VideoCommentResource.php
return [
    'id' => $this->id,
    'video_id' => $this->commentable->id,
    'user' => [
        'id' => $this->user->id,
        'name' => $this->user->name,
        'code' => $this->user->code,
        'image' => $this->user->latestProfilePhoto?->url
    ],
    // ...
    'likes' => $this->whenCounted('likes_count'),
    'dislikes' => $this->whenCounted('dislikes_count'),
    'replies_count' => $this->whenCounted('replies_count'),
    'created_at' => jdate($this->created_at)->format('Y/m/d')
];
```

## Endpoint Details
### GET `/api/tutorials/{video}/comments`
- **Purpose:** Retrieve the most-liked parent comments for the given video.
- **Behavior:** Orders by `likes_count` descending, loads the commenting user’s profile photo, counts likes/dislikes/replies, filters out replies (`parent_id` null), paginates in pages of 10.
- **Response:** `200 OK` with a `data` array of comments and pagination scaffolding.

### POST `/api/tutorials/{video}/comments`
- **Purpose:** Create a new top-level comment tied to the video.
- **Authorization:** Caller must be authenticated and verified; policy allows any authenticated user to create.
- **Validation:** `content` is required, string, max 2000 characters.
- **Behavior:** Persists the comment linked to the video and authenticated user, returns the resource representation with HTTP 201 semantics (`VideoCommentResource` with default 200 from controller).

```41:51:app/Http/Controllers/Api/V1/VideoCommentsController.php
$request->validate(['content' => 'required|string|max:2000']);

$comment = $video->comments()->create([
    'user_id' => $request->user()->id,
    'content' => $request->content
]);
```

### PUT `/api/tutorials/{video}/comments/{comment}`
- **Purpose:** Update comment text.
- **Authorization:** Requires ownership; enforced via `authorize('update', $comment)`.
- **Validation:** Same as creation (`content` required string ≤ 2000 characters).
- **Behavior:** Replaces the comment content with the new text, then refreshes before returning the resource.

```60:72:app/Http/Controllers/Api/V1/VideoCommentsController.php
$this->authorize('update', $comment);
$request->validate(['content' => 'required|string|max:2000']);
$comment->update([
    'user_id' => $request->user()->id,
    'content' => $request->content
]);
```

### DELETE `/api/tutorials/{video}/comments/{comment}`
- **Purpose:** Remove a comment authored by the caller.
- **Authorization:** Ownership enforced via `authorize('update', $comment)` (same rule as edit).
- **Behavior:** Deletes the comment record and any related interaction entries; responds with empty JSON and `200 OK`.

```80:88:app/Http/Controllers/Api/V1/VideoCommentsController.php
$this->authorize('update', $comment);
$comment->delete();
$comment->interactions()->delete();
return new JsonResponse([], 200);
```

### POST `/api/tutorials/{video}/comments/{comment}/interactions`
- **Purpose:** Record a like or dislike from the caller.
- **Authorization:** Users cannot react to their own comments (`CommentPolicy@like`/`@dislike`).
- **Validation:** `liked` must be present and boolean (`true` for like, `false` for dislike).
- **Behavior:** Upserts a morph interaction record keyed by the user, storing the like flag and caller IP; returns empty JSON with `200 OK`.

```99:117:app/Http/Controllers/Api/V1/VideoCommentsController.php
$request->validate(['liked' => 'required|boolean']);
$comment->interactions()->updateOrCreate(
    ['user_id' => $request->user()->id],
    ['liked' => $likedBool, 'ip_address' => $request->ip()]
);
```

### POST `/api/tutorials/{video}/comments/{comment}/report`
- **Purpose:** File a report against another user’s comment on the video.
- **Authorization:** `CommentPolicy@report` disallows reporting self-authored comments.
- **Validation:** `content` required, string, max 2000 characters.
- **Behavior:** Stores the report via the video’s `reports()` relationship with the caller ID and target comment; returns empty JSON with `200 OK`.

```120:132:app/Http/Controllers/Api/V1/VideoCommentsController.php
$this->authorize('report', $comment);
$request->validate(['content' => 'required|string|max:2000']);
$video->reports()->create([
    'user_id' => $request->user()->id,
    'comment_id' => $comment->id,
    'content' => $request->content
]);
```

## Validation Rules

| Endpoint | Field | Rules | Notes |
| --- | --- | --- | --- |
| `POST /api/tutorials/{video}/comments` | `content` | required, string, max:2000 | Enforces non-empty comment body within 2k characters. |
| `PUT /api/tutorials/{video}/comments/{comment}` | `content` | required, string, max:2000 | Same constraint as creation; update fails if empty. |
| `POST /api/tutorials/{video}/comments/{comment}/interactions` | `liked` | required, boolean | Accepts `true` (like) or `false` (dislike). |
| `POST /api/tutorials/{video}/comments/{comment}/report` | `content` | required, string, max:2000 | Requires detailed report message; duplicates allowed. |

## Error Handling
- Validation errors produce standard Laravel `422 Unprocessable Entity` payloads with field-specific messages.
- Authorization failures surface as `403 Forbidden` responses via Laravel’s policy enforcement.
- Model binding returns `404 Not Found` when the `{video}` or `{comment}` identifiers do not resolve.

## Related Features
- Comment replies are managed by `CommentReplyController` under `/api/comments/{comment}/...`. See dedicated documentation when working with nested conversations.


