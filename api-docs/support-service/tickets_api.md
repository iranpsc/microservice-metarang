# Tickets API Guide

## Summary
- `GET /api/tickets` and `GET /api/tickets?recieved=true` paginate tickets where the caller is sender or receiver, including Jalali-formatted timestamps and avatar URLs.
- `POST /api/tickets` opens a new ticket against either a user or a service department, attaches optional media, and notifies the receiver.
- `PUT /api/tickets/{ticket}` refreshes the ticket content and resets its status to `NEW`; the original sender is the only actor allowed to update.
- `POST /api/tickets/response/{ticket}` appends a response, updates the ticket status to `ANSWERED`, and alerts the original sender.
- `GET /api/tickets/close/{ticket}` transitions an open ticket to `CLOSED`; only the original sender can perform this action.

```156:163:routes/api.php
Route::controller(TicketController::class)->prefix('tickets')->group(function () {
    Route::post('/response/{ticket}', 'response')->name('tickets.response');
    Route::get('/close/{ticket}', 'close');
});

Route::apiResources([
    'tickets' => TicketController::class,
]);
```

All routes inherit the enclosing `auth:sanctum`, `verified`, and `activity` middleware bundle, and `TicketController` applies `authorizeResource` to wire policy checks to the RESTful verbs.

```17:85:app/Http/Controllers/Api/V1/TicketController.php
public function __construct()
{
    $this->authorizeResource(Ticket::class);
}

public function index(): AnonymousResourceCollection
{
    $tickets = Ticket::with('sender.latestProfilePhoto', 'reciever.latestProfilePhoto')
        ->when(request()->boolean('recieved') === true, function ($query) {
            $query->whereBelongsTo(request()->user(), 'reciever');
        }, function ($query) {
            $query->whereBelongsTo(request()->user(), 'sender');
        })
        ->orderByDesc('updated_at')
        ->simplePaginate(10);

    return TicketResource::collection($tickets);
}
// ... existing code ...
```

## Authentication & Middleware
- `auth:sanctum` ensures every request resolves to an authenticated user.
- `verified` requires the caller to have completed account verification.
- `activity` logs REST usage for auditing.
- Route-model binding auto-resolves `{ticket}` and fails with HTTP 404 if the ticket does not exist or cannot be accessed under policy rules.

## Route Registry
| Method | Path | Middleware → Policy Gate | Controller action | Notes |
| --- | --- | --- | --- | --- |
| GET | `/api/tickets` | `auth:sanctum`, `verified`, `activity` → `TicketPolicy@viewAny` | `TicketController@index` | Paginated (page, `cursor` parameters) list of tickets initiated by the caller; add `?recieved=true` to switch to tickets received by the caller. |
| POST | `/api/tickets` | `auth:sanctum`, `verified`, `activity` → `TicketPolicy@create` | `TicketController@store` | Creates a ticket. Either `reciever` **or** `department` must be supplied (mutually exclusive). Optional file attachment stored under `storage/app/public/tickets`. |
| GET | `/api/tickets/{ticket}` | `auth:sanctum`, `verified`, `activity` → `TicketPolicy@view` | `TicketController@show` | Returns ticket with sender, receiver, and eager-loaded responses. |
| PUT/PATCH | `/api/tickets/{ticket}` | `auth:sanctum`, `verified`, `activity` → `TicketPolicy@update` | `TicketController@update` | Sender-only edit; resets status to `NEW` and replaces attachment if supplied. |
| DELETE | `/api/tickets/{ticket}` | `auth:sanctum`, `verified`, `activity` → `TicketPolicy@delete` | `TicketController@destroy` | Always denied (policy returns `false`), producing HTTP 403. |
| POST | `/api/tickets/response/{ticket}` | `auth:sanctum`, `verified`, `activity` → `TicketPolicy@respond` | `TicketController@response` | Adds a response when caller is the receiver, or the sender while the ticket is still open. |
| GET | `/api/tickets/close/{ticket}` | `auth:sanctum`, `verified`, `activity` → `TicketPolicy@close` | `TicketController@close` | Sender-only close; flips status to `CLOSED` and returns updated resource. |

## Request Contracts
- **Create / Update Ticket (`POST`, `PUT`, `PATCH`)**
  - `title` `string` (required, ≤ 250 chars)
  - `content` `string` (required, ≤ 500 chars)
  - `attachment` optional file (`png`, `jpg`, `jpeg`, `pdf`, ≤ 5 MB)
  - `reciever` nullable `int` (`users.id`); required if `department` absent and prohibited when `department` provided.
  - `department` nullable enum; required if `reciever` absent, prohibited when `reciever` provided. Allowed values:
    - `technical_support`, `citizens_safety`, `investment`, `inspection`, `protection`, `ztb`.
- **Respond to Ticket (`POST /response/{ticket}`)**
  - `response` `string` (required, ≤ 500 chars)
  - `attachment` optional file (`png`, `jpg`, `pdf`, ≤ 5 MB)

Validation failures return HTTP 422 with field-specific error messages.

## Response Shape

Tickets serialize through `TicketResource`, returning Jalali-formatted timestamps and nested profile data when the relationships are eager-loaded.

```18:43:app/Http/Resources/TicketResource.php
return [
    'id' => $this->id,
    'title' => $this->title,
    'sender' => $this->whenLoaded('sender', function () {
        return [
            'name' => $this->sender->name,
            'code' => $this->sender->code,
            'profile-photo' => $this->sender->latestProfilePhoto?->url,
        ];
    }),
    'reciever' => $this->whenLoaded('reciever', function () {
        return [
            'name' => $this->reciever->name,
            'code' => $this->sender->code,
            'profile-photo' => $this->reciever->latestProfilePhoto?->url,
        ];
    }),
    'department' => $this->whenNotNull($this->department),
    'code' => $this->code,
    'attachment' => $this->attachment,
    'content' => $this->content,
    'status' => $this->status,
    'date' => jdate($this->updated_at)->format('Y/m/d'),
    'time' => jdate($this->updated_at)->format('H:m:s'),
    'responses' => TicketResponseResource::collection($this->whenLoaded('responses')),
];
```

Responses expose an array of `TicketResponseResource` objects, each containing responder metadata and Jalali timestamps. Empty relationships are omitted.

## Status Lifecycle

`Ticket` statuses are integer-coded constants:

| Constant | Value | Meaning | Mutations |
| --- | --- | --- | --- |
| `NEW` | `0` | Newly created or edited ticket awaiting response. | Set on create and on update. |
| `ANSWERED` | `1` | Ticket has at least one response. | Applied when `response()` succeeds. |
| `RESOLVED` | `2` | Reserved for future flows; not set inside this controller. | Manual via model methods elsewhere. |
| `UNRESOLVED` | `3` | Reserved flag for escalation. | Manual via model methods elsewhere. |
| `TRACKING` | `4` | Reserved for long-running cases. | Manual via model methods elsewhere. |
| `CLOSED` | `5` | Workflow finished. `close()` endpoint and `Ticket::close()` helper both use this value. |

Tickets are considered *open* whenever `status !== CLOSED`, which controls responder permissions.

## Policy Rules
- **viewAny**: Any authenticated user can view their own tickets list.
- **view**: Access granted if the caller is the ticket sender or receiver.
- **create**: Denied when the targeted receiver has disabled ticket intake via `ProfileLimitation` (`options['send_ticket'] === false`); otherwise allowed.
- **update**: Sender-only.
- **delete**: Always denied, ensuring historical record retention.
- **respond**: Allowed for the receiver at any time, and for the sender while the ticket remains open.
- **close**: Sender-only while the ticket is open.

Denied actions return HTTP 403 with the localized denial message when available.

## Notifications & Side Effects
- Creating a ticket against a specific user triggers `TicketRecieved` for the receiver.
- Responding to a ticket pings the ticket sender via the same notification.
- File attachments are stored under `storage/app/public/tickets`; the controller emits full URLs via `url('uploads/...')`. Ensure the storage symlink (`php artisan storage:link`) is in place for direct downloads.

## Usage Notes
- The index action uses simple pagination (`simplePaginate(10)`). Supply a `page` query string to page forward; there is no `total` count in the payload.
- `TicketController@show` loads responses and profile photos; consider caching for high-traffic inbox UIs.
- Attachments are overwritten on update if a new file is provided; senders must re-upload previous files if they want to preserve them.
- Because `close` uses `GET`, CSRF protection is not applied; clients should treat it as an idempotent mutation and call it explicitly rather than relying on implicit closure.


