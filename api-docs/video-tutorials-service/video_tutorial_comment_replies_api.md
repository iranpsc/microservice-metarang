# Video Tutorial Comment Replies API

Comprehensive reference for the comment–reply endpoints exposed under `api/comments/*`. These endpoints let clients browse, create, update, delete, and react to replies on tutorial video comments.

---

## Overview

- **Base URL:** `/api/comments`
- **Content Type:** `application/json`
- **Authentication:** Laravel Sanctum tokens with the `verified` middleware unless otherwise noted.
- **Resource shape:** Responses use `VideoCommentResource`, returning reply metadata, author profile snapshot, reaction counts, and creation date.

---

## Authentication & Authorization

| Ability | Checked Policy Method | Rule |
| --- | --- | --- |
| Reply to a comment | `CommentPolicy::reply` | User must not be the author of the target comment. |
| Update / delete a reply | `CommentPolicy::update` | User must be the author of the reply. |
| Like / dislike a reply | `CommentPolicy::like` / `CommentPolicy::dislike` | User must not be the author of the reply. |

- Unauthorized requests return HTTP `403` with the standard Laravel authorization error payload.
- All non-public routes also require a valid Sanctum token and a verified user; missing or invalid tokens trigger `401`/`419` responses handled by Laravel Sanctum.

---

## Reply Representation

```startLine:endLine:app/Http/Resources/VideoCommentResource.php
        return [
            'id' => $this->id,
            'video_id' => $this->commentable->id,
            'parent_id' => $this->parent_id,
            'user' => [
                'id' => $this->user->id,
                'name' => $this->user->name,
                'code' => $this->user->code,
                'image' => $this->user->latestProfilePhoto?->url,
            ],
            'content' => $this->content,
            'likes' => $this->likes_count,
            'dislikes' => $this->dislikes_count,
            'replies_count' => $this->replies_count,
            'is_reply' => $this->isReply(),
            'replies' => [...],
            'created_at' => jdate($this->created_at)->format('Y/m/d'),
        ];
```

- `replies` is only populated when explicitly eager-loaded (not returned by default in v2 reply calls).
- `likes`, `dislikes`, and `replies_count` appear when the resource is returned with counts, as in the list endpoint.

---

## Endpoints

| Method | Path | Auth | Description |
| --- | --- | --- | --- |
| GET | `/api/comments/{comment}/replies` | Optional | List replies for a comment (public). |
| POST | `/api/comments/{comment}/reply` | Required | Create a new reply beneath the parent comment. |
| PUT | `/api/comments/{comment}/replies/{reply}` | Required | Update an existing reply authored by the caller. |
| DELETE | `/api/comments/{comment}/replies/{reply}` | Required | Delete an existing reply authored by the caller. |
| POST | `/api/comments/{comment}/replies/{reply}/interactions` | Required | Like or dislike a reply. |

> **Route bindings:** `{comment}` and `{reply}` resolve to `App\Models\Comment`. The `{reply}` must belong to the `{comment}`; otherwise Laravel returns `404`.

---

### GET `/api/comments/{comment}/replies`

- **Access:** Public; middleware exclusions remove `auth:sanctum` and `verified`.
- **Pagination:** Simple pagination (`simplePaginate(10)`) with `page` query parameter.
- **Includes:** Automatically loads `user` (with `id`, `name`, `code`, `image`) and reaction counts.

**Query Parameters**

| Name | Type | Required | Description |
| --- | --- | --- | --- |
| `page` | integer | No | Page number (1-indexed). |

**Responses**

| Code | Description |
| --- | --- |
| `200` | Paginated list of replies (array of `VideoCommentResource`). |
| `404` | Comment not found. |

**Example**

```json
{
  "data": [
    {
      "id": 42,
      "video_id": 15,
      "parent_id": 7,
      "user": {
        "id": 3,
        "name": "Jane Doe",
        "code": "USR123",
        "image": "https://cdn.example.com/photos/123.jpg"
      },
      "content": "Loved this walkthrough!",
      "likes": 12,
      "dislikes": 0,
      "is_reply": true,
      "created_at": "2025/11/08"
    }
  ],
  "links": {
    "next": null
  }
}
```

---

### POST `/api/comments/{comment}/reply`

- **Access:** Authenticated & verified users only.
- **Authorization:** `CommentPolicy::reply` — callers cannot reply to their own comment.
- **Validation Rules:**
  - `content`: `required|string|max:2000`
- **Behavior:**
  - Replies are always attached to the top-level parent of the referenced comment. Attempting to reply to an existing reply will redirect the association to the parent comment, preserving a two-level hierarchy.

**Request Body**

```json
{
  "content": "Can you clarify the shortcut you used?"
}
```

**Responses**

| Code | Description |
| --- | --- |
| `200` | Reply created; returns the new `VideoCommentResource`. |
| `422` | Validation error (missing/invalid `content`). |
| `403` | Authorization failure (replying to own comment). |
| `404` | Comment not found. |

---

### PUT `/api/comments/{comment}/replies/{reply}`

- **Access:** Authenticated & verified users only.
- **Authorization:** `CommentPolicy::update` — only the reply author may update.
- **Validation Rules:**
  - `content`: `required|string|max:2000`
- **Behavior:** Updates the reply content and re-asserts the author to the acting user.

**Responses**

| Code | Description |
| --- | --- |
| `200` | Updated reply (`VideoCommentResource`). |
| `422` | Validation error. |
| `403` | Caller is not the reply author. |
| `404` | Comment or reply not found / not related. |

---

### DELETE `/api/comments/{comment}/replies/{reply}`

- **Access:** Authenticated & verified users only.
- **Authorization:** `CommentPolicy::update` — only the reply author may delete.
- **Behavior:** Deletes the reply and cascades removal of its interactions (`likes`/`dislikes`).

**Responses**

| Code | Description |
| --- | --- |
| `200` | Empty JSON object on success. |
| `403` | Caller is not the reply author. |
| `404` | Comment or reply not found / not related. |

---

### POST `/api/comments/{comment}/replies/{reply}/interactions`

- **Access:** Authenticated & verified users only.
- **Authorization:** `CommentPolicy::like` / `CommentPolicy::dislike` — users cannot react to their own replies.
- **Validation Rules:**
  - `liked`: `required|boolean`
- **Behavior:**
  - Uses `updateOrCreate` to upsert the user’s reaction on the reply.
  - Stores the caller’s IP address when recording the interaction.

**Request Body**

```json
{
  "liked": true
}
```

**Responses**

| Code | Description |
| --- | --- |
| `200` | Reaction recorded (empty JSON payload). |
| `422` | Validation error (missing or non-boolean `liked`). |
| `403` | Caller is the reply’s author. |
| `404` | Comment or reply not found / not related. |

---

## Error Handling

- **Validation errors** return standard Laravel validation responses with field-specific messages; HTTP status `422` when `Accept: application/json` is provided (default for API routes).
- **Authorization errors** return `403` with `{"message": "This action is unauthorized."}`.
- **Model binding failures** (invalid IDs, mismatched reply/comment) return `404`.

---

## Related Models & Constraints

- Replies are stored in the polymorphic `comments` table (`App\Models\Comment`), linked to tutorials through `commentable`.
- Interactions are managed through `App\Models\Interaction` via a morph-many relationship (`likeable`).
- Replies are limited to a maximum depth of 2 levels due to the controller forcing replies onto the top-level parent.

---

## Changelog

- **v2** (current): Introduced dedicated reply endpoints under `/api/comments`. Previous versions handled replies alongside top-level comments.


