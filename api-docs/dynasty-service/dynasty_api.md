# Dynasty API Guide

## Summary
- `GET /api/dynasty` returns either the callers dynasty (via `DynastyResource`) or a feature/prize selection payload when no dynasty exists.
- `POST /api/dynasty/create/{feature}` and `POST /api/dynasty/{dynasty}/update/{feature}` are guarded by `DynastyPolicy` and `account.security`, enabling dynasty creation and feature swaps (with debt/lock handling for rapid changes).
- Family membership flows revolve around join requests: `SendJoinRequestController` issues requests, `AcceptJoinRequestController` reviews inbound ones, and both rely on `JoinRequestPolicy` plus `UserPolicy::addFamilyMember`.
- Children-specific permissions can be read from dynasty payloads and toggled via `POST /api/dynasty/children/{user}` after passing `UpdateChildrenPermissionsRequest` and `UserPolicy::controlPermissions`.
- Dynasty prizes are exposed under `/api/dynasty/prizes`; redeeming a prize updates wallet balances and growth variables before deleting the `RecievedPrize` record.

## Route Registry
| Method | Path | Middleware | Controller action | Notes |
| --- | --- | --- | --- | --- |
| GET | `/api/dynasty` | `auth:sanctum`, `verified`, `activity` | `DynastyController@index` | Returns dynasty data or available features + intro prizes when none exists. |
| POST | `/api/dynasty/create/{feature}` | `auth:sanctum`, `verified`, `activity`, `account.security` | `DynastyController@store` | Creates dynasty tied to selected residential feature; seeds owning member. |
| POST | `/api/dynasty/{dynasty}/update/{feature}` | `auth:sanctum`, `verified`, `activity`, `account.security` | `DynastyController@update` | Switches dynasty feature; fast repeats create debts and lock previous feature for one month. |
| GET | `/api/dynasty/{dynasty}/family/{family}` | `auth:sanctum`, `verified`, `activity` | `FamilyController@index` | Lists family members for provided dynasty/family binding. |
| GET | `/api/dynasty/requests/sent` | `auth:sanctum`, `verified`, `activity` | `SendJoinRequestController@index` | Paginates (10/page) outbound join requests with prize metadata. |
| GET | `/api/dynasty/requests/sent/{joinRequest}` | `auth:sanctum`, `verified`, `activity` | `SendJoinRequestController@show` | Requires `JoinRequestPolicy::view`; includes request message & prize.
| DELETE | `/api/dynasty/requests/sent/{joinRequest}` | `auth:sanctum`, `verified`, `activity`, `account.security` | `SendJoinRequestController@destroy` | Cancels pending request (`status === 0`) after passing `JoinRequestPolicy::delete`. |
| POST | `/api/dynasty/add/member/get/permissions` | `auth:sanctum`, `verified`, `activity` | `SendJoinRequestController@getPermissions` | Returns default offspring permission template. |
| POST | `/api/dynasty/add/member` | `auth:sanctum`, `verified`, `activity`, `account.security` | `SendJoinRequestController@store` | Validates `AddFamilyMemberRequest`; sends join request & notification pair. |
| POST | `/api/dynasty/search` | `auth:sanctum`, `verified`, `activity` | `SendJoinRequestController@search` | Performs user search via `UserSearchService`. |
| GET | `/api/dynasty/requests/recieved` | `auth:sanctum`, `verified`, `activity` | `AcceptJoinRequestController@index` | Lists inbound join requests awaiting action. |
| GET | `/api/dynasty/requests/recieved/{joinRequest}` | `auth:sanctum`, `verified`, `activity` | `AcceptJoinRequestController@show` | Requires `JoinRequestPolicy::view`; includes message and offspring permissions snapshot. |
| POST | `/api/dynasty/requests/recieved/{joinRequest}` | `auth:sanctum`, `verified`, `activity`, `account.security` | `AcceptJoinRequestController@accept` | Accepts request, links member, awards prize, syncs permissions. |
| DELETE | `/api/dynasty/requests/recieved/{joinRequest}` | `auth:sanctum`, `verified`, `activity`, `account.security` | `AcceptJoinRequestController@reject` | Rejects request and notifies both parties. |
| GET | `/api/dynasty/prizes` | `auth:sanctum`, `verified`, `activity` | `DynastyPrizeController@index` | Lists unclaimed prizes for caller. |
| GET | `/api/dynasty/prizes/{recievedPrize}` | `auth:sanctum`, `verified`, `activity` | `DynastyPrizeController@show` | Returns detailed prize (message only on this route). |
| POST | `/api/dynasty/prizes/{recievedPrize}` | `auth:sanctum`, `verified`, `activity` | `DynastyPrizeController@store` | Redeems prize, adjusting wallet and variables, then deletes receipt. |
| POST | `/api/dynasty/children/{user}` | `auth:sanctum`, `verified`, `activity` | `ChildernPermissionsController` | Updates a minors single permission bit after policy check.

All routes inherit the global `/api` prefix from `RouteServiceProvider`.

## Response Resources
### Dynasty payload

```17:47:app/Http/Resources/Dynasty/DynastyResource.php
return [
    'user-has-dynasty' => true,
    'id' => $this->id,
    'family_id' => $this->family->id,
    'created_at' => jdate($this->created_at)->format('Y/m/d'),
    'profile-image' => $this->user->profilePhotos->last()?->url,
    'dynasty-feature' => [
        'id' => $this->feature->id,
        'properties_id' => $this->feature->properties->id,
        'area' => $this->feature->properties->area,
        'density' => $this->feature->properties->density,
        'feature-profit-increase' => $this->feature->properties->stability > 10000
            ? number_format($this->feature->properties->stability / 10000 - 1, 3)
            : 0,
        'family-members-count' => $this->family->familyMembers->count(),
        'last-updated' => jdate($this->updated_at)->format('Y/m/d H:m:s')
    ],
    'features' => $request->user()->features
        ->reject(fn ($feature) => $feature->properties->karbari !== 'm' || $feature->id == $this->feature->id)
        ->map(fn ($feature) => [
            'id' => $feature->id,
            'properties_id' => $feature->properties->id,
            'density' => $feature->properties->density,
            'stability' => $feature->properties->stability,
            'area' => $feature->properties->area,
        ])
];
```

- When `user-has-dynasty` is `false`, `DynastyController@index` returns `features` (residential only) and intro `prizes` without the dynasty block.
- Jalali timestamps are used consistently; consumers should display them as-is or convert client-side.

### Family member projection

```17:37:app/Http/Resources/Dynasty/FamilyMemberResource.php
return [
    'id' => $this->user->id,
    'code' => $this->user->code,
    'profile_photo' => $this->user->profilePhotos->last()?->url,
    'online' => $this->user->isOnline(),
    'relationship' => $this->relationship,
    'level' => $this->user->level?->slug,
    $this->mergeWhen($this->user->isUnderEighteen(), [
        'permissions' => [
            'BFR' => $this->user->permissions?->BFR,
            'SF' => $this->user->permissions?->SF,
            'W' => $this->user->permissions?->W,
            'JU' => $this->user->permissions?->JU,
            'DM' => $this->user->permissions?->DM,
            'PIUP' => $this->user->permissions?->PIUP,
            'PITC' => $this->user->permissions?->PITC,
            'PIC' => $this->user->permissions?->PIC,
            'ESOO' => $this->user->permissions?->ESOO,
            'COTB' => $this->user->permissions?->COTB,
        ]
    ]),
];
```

### Sent join requests

```18:45:app/Http/Resources/Dynasty/SentRequestsResource.php
return [
    'id' => $this->id,
    'to_user' => $this->whenLoaded('toUser', [
        'id' => $this->toUser->id,
        'code' => $this->toUser->code,
        'name' => $this->toUser->name,
        'profile_photo' => $this->toUser->latestProfilePhoto?->url,
    ]),
    'status' => $this->status,
    'relationship' => $this->getRelationShipTitle(),
    'date' => jdate($this->created_at)->format('Y/m/d'),
    'time' => jdate($this->created_at)->format('H:i'),
    'prize' => $this->whenLoaded('requestPrize', [
        'id' => $this->requestPrize->id,
        'psc' => number_format($this->requestPrize->psc / Variable::getRate('psc'), 2),
        'satisfaction' => number_format($this->requestPrize->satisfaction * 100),
        'introducation_profit_increase' => number_format($this->requestPrize->introducation_profit_increase * 100),
        'accumulated_capital_reserve' => number_format($this->requestPrize->accumulated_capital_reserve * 100),
        'data_storage' => number_format($this->requestPrize->data_storage * 100),
    ]),
    $this->mergeWhen(request()->routeIs('dynasty.requests.sent.show'), [
        'message' => $this->message,
    ]),
];
```

### Received join requests

```18:46:app/Http/Resources/Dynasty/RecievedJoinRequest.php
return [
    'id' => $this->id,
    'from_user' => $this->whenLoaded('fromUser', [
        'id' => $this->fromUser->id,
        'code' => $this->fromUser->code,
        'name' => $this->fromUser->name,
        'profile_photo' => $this->fromUser->latestProfilePhoto?->url,
    ]),
    'status' => $this->status,
    'relationship' => $this->getRelationShipTitle(),
    'date' => jdate($this->created_at)->format('Y/m/d'),
    'time' => jdate($this->created_at)->format('H:i'),
    $this->mergeWhen(request()->routeIs('joinRequests.recieved.show'), [
        'message' => $this->message,
        $this->mergeWhen($this->relationship === 'offspring', [
            'permissions' => [
                'BFR' => $this->toUser->permissions?->BFR,
                'SF' => $this->toUser->permissions?->SF,
                'W' => $this->toUser->permissions?->W,
                'JU' => $this->toUser->permissions?->JU,
                'DM' => $this->toUser->permissions?->DM,
                'PIUP' => $this->toUser->permissions?->PIUP,
                'PITC' => $this->toUser->permissions?->PITC,
                'PIC' => $this->toUser->permissions?->PIC,
                'ESOO' => $this->toUser->permissions?->ESOO,
                'COTB' => $this->toUser->permissions?->COTB,
            ]
        ])
    ])
];
```

### Dynasty prizes

```17:27:app/Http/Resources/Dynasty/DynastyPrizeResource.php
return [
    'id' => $this->id,
    'psc' => $this->prize->psc,
    'satisfaction' => number_format($this->prize->satisfaction * 100),
    'introducation_profit_increase' => number_format($this->prize->introduction_profit_increase * 100),
    'accumulated_capital_reserve' => number_format($this->prize->accumulated_capital_reserve * 100),
    'data_storage' => number_format($this->prize->data_storage * 100),
    $this->mergeWhen(request()->routeIs('prizes.show'), [
        'message' => $this->message,
    ])
];
```

## Validation Rules
- **Create dynasty**: The controller currently typehints `Request` but authorises with `DynastyPolicy::create`. No dedicated form request exists; instead, the policy enforces ownership, verification, residential usage (`karbari === 'm'`), and absence of pending feature requests.
- **Add family member** (`POST /api/dynasty/add/member`) uses `AddFamilyMemberRequest`:

```27:44:app/Http/Requests/AddFamilyMemberRequest.php
return [
    'user' => 'required|integer|exists:users,id',
    'relationship' => [
        'required',
        'string',
        new Enum(FamilyRelationships::class)
    ],
    'permissions' => [
        'required_if:relationship,offspring',
        'array',
        'min:10',
        'required_array_keys:BFR,SF,W,JU,DM,PIUP,PITC,PIC,ESOO,COTB',
        Rule::prohibitedIf(fn () => request()->input('relationship') !== 'offspring'),
    ],
    'permissions.*' => 'integer|boolean'
];
```

- **Fetch default permissions** (`POST /api/dynasty/add/member/get/permissions`) validates `relationship` as required string limited to `offspring`.
- **Search users** (`POST /api/dynasty/search`) requires a non-empty string `searchTerm`.
- **Update child permission** (`POST /api/dynasty/children/{user}`) leverages `UpdateChildrenPermissionsRequest`:

```24:29:app/Http/Requests/UpdateChildrenPermissionsRequest.php
return [
    'permission' => 'required|string|in:BFR,SF,W,JU,DM,PIUP,PITC,PIC,ESOO,COTB',
    'status'     => 'required_with:permission|boolean',
];
```

Validation failures return Laravel-standard `422` responses with field-specific errors.

## Policy & Authorization
| Ability | Policy method | Applied in | Rule summary |
| --- | --- | --- | --- |
| `create` | `DynastyPolicy::create` | `DynastyController@store` | Caller must be verified, own the feature, have no existing dynasty, and the feature must be residential with no pending requests. |
| `update` | `DynastyPolicy::update` | `DynastyController@update` | Feature must be owned by caller, differ from current dynasty feature, and have no pending requests; dynasty must belong to caller. |
| `addFamilyMember` | `UserPolicy::addFamilyMember` | `SendJoinRequestController@store` | Ensures dynasty exists, target user is verified, no duplicate/rejected/pending requests, and relationship-specific caps (e.g., max 4 siblings/offspring, single parents/spouse). |
| `view` | `JoinRequestPolicy::view` | `SendJoinRequestController@show`, `AcceptJoinRequestController@show` | Either sender or receiver may inspect the request. |
| `delete` | `JoinRequestPolicy::delete` | `SendJoinRequestController@destroy` | Only senders may delete pending requests. |
| `accept`/`reject` | `JoinRequestPolicy::{accept,reject}` | `AcceptJoinRequestController@accept/reject` | Only intended recipient can act while request status is pending (`0`). |
| `controlPermissions` | `UserPolicy::controlPermissions` | `ChildernPermissionsController` | Allows guardians to toggle minors permissions only when child is under 18 and part of their family. |

Additionally, `SendJoinRequestController@store` manually forbids setting permissions for adult offspring, and `AcceptJoinRequestController@accept` upgrades permissions for minors when appropriate.

## Endpoint Details
### `GET /api/dynasty`
- Returns `200 OK` with either the dynasty payload (`DynastyResource`) or a response containing `user-has-dynasty: false`, available residential features, and introduction prizes (`IntroductionPrizeResource::collection`).
- Useful to determine whether to show creation UI or dynasty dashboard.

### `POST /api/dynasty/create/{feature}`
- Requires authenticated, verified, active user with `account.security` verification.
- On success, creates dynasty, instantiates family, attaches caller as `owner`, and sends `DynastyCreatedNotification` referencing the features property id.
- Returns the newly-created dynasty wrapped in `DynastyResource`.

### `POST /api/dynasty/{dynasty}/update/{feature}`
- Same middleware + policy as creation.
- Swaps dynastys feature; if the previous feature was changed less than 30 days ago, user incurs a debt keyed by feature color, previous property label is set to `locked`, and a `LockedFeature` record blocks reuse for one month before new `DynastyFeatureChangedNotification` is emitted.
- Response is the refreshed dynasty resource.

### `GET /api/dynasty/{dynasty}/family/{family}`
- Returns a `FamilyMemberResource` collection. Includes per-member permissions when the member is a minor.
- Relies on route-model binding to ensure the family belongs to the dynasty.

### Join request lifecycle
- **Send request** (`POST /api/dynasty/add/member`): Validates payload, ensures policy requirements (verified target, relationship caps, etc.), optionally stores default permissions for minors, and enqueues paired `JoinDynastyNotification` messages using configurable templates from `DynastyMessage`.
- **List sent requests** (`GET /api/dynasty/requests/sent`): Simple pagination (10 per page) with to-user details and prize snapshot.
- **View sent request** (`GET /api/dynasty/requests/sent/{joinRequest}`): Adds `message` field; prize values are pre-scaled for display.
- **Delete sent request** (`DELETE /api/dynasty/requests/sent/{joinRequest}`): Deletes pending requests once authorized via `JoinRequestPolicy::delete`.
- **List received requests** (`GET /api/dynasty/requests/recieved`): Shows inbound pending requests with sender profile/photo.
- **View received request** (`GET /api/dynasty/requests/recieved/{joinRequest}`): Adds invite message and, for offspring, current permission flags.
- **Accept request** (`POST /api/dynasty/requests/recieved/{joinRequest}`): Marks request `status` as `1`, creates a `FamilyMember` record for the recipient (relationship stored from request), handles children permissions (creating defaults for underage parents or verifying child flags), awards the relevant `DynastyPrize`, and notifies both parties using acceptance templates.
- **Reject request** (`DELETE /api/dynasty/requests/recieved/{joinRequest}`): Marks request `status` as `-1` and notifies both sides of the rejection.

### `POST /api/dynasty/add/member/get/permissions`
- Accepts only `relationship=offspring` and returns the default `DynastyPermission` record for UI prefill.

### `POST /api/dynasty/search`
- Requires body `{ "searchTerm": "..." }` and returns `{ "data": [...] }` with transformed user results (key name is likely a typo; clients should expect it).

### `POST /api/dynasty/children/{user}`
- Toggles a single permission flag (`permission`, `status`) for a minor already in the callers dynasty.
- Policy rejects attempts on adults, non-family members, or self-control.
- Returns `200 OK` with empty JSON body.

### Dynasty prizes
- **List** (`GET /api/dynasty/prizes`): Returns unclaimed prizes as `DynastyPrizeResource` collection.
- **Show** (`GET /api/dynasty/prizes/{recievedPrize}`): Adds context `message` explaining the award.
- **Redeem** (`POST /api/dynasty/prizes/{recievedPrize}`): Adds PSC (converted using `Variable::getRate('psc')`) and satisfaction to wallet, amplifies `variables` metrics (referral profit, data storage, withdraw profit) proportionally, then deletes the record.

## Operational Notes
- All dynasty routes depend on Sanctum authentication plus `verified` and `activity` middlewares; ensure API clients attach bearer tokens and operate with verified accounts.
- `account.security` middleware enforces recent 2FA/OTP confirmation on sensitive mutations (create/update dynasty, send/delete requests, accept/reject requests).
- Jalali date formatting permeates dynasty responses; conversions, if needed, must be handled client-side.
- Prize values surface both raw PSC units and percentage multipliers scaled by `*100`; plan UI labelling accordingly.
