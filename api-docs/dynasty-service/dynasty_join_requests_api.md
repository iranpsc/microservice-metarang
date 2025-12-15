# Dynasty Join Requests API

This document describes every endpoint involved in the dynasty family membership flow – sending, viewing, accepting, and rejecting join requests – plus the validation and authorization logic that governs them. All paths are relative to the `/api` prefix.

## Overview

- Primary controllers: `SendJoinRequestController` for outbound requests and `AcceptJoinRequestController` for inbound ones.
- Transport format: JSON requests/responses encoded in UTF-8. Timestamps are returned in Jalali (`Y/m/d` and `H:i`).
- Status codes: standard HTTP codes (`200`, `201`, `204`, `403`, `404`, `422`).
- Notifications: successful create/accept/reject actions trigger `JoinDynastyNotification` messages to both parties.

## Authentication & Middleware

All dynasty endpoints inherit the global middleware stack `auth:sanctum`, `verified`, and `activity`. Sensitive mutations additionally require `account.security`:

| Endpoint | Extra middleware |
| --- | --- |
| `POST /dynasty/add/member` | `account.security` |
| `DELETE /dynasty/requests/sent/{joinRequest}` | `account.security` (route points to `destroy`) |
| `POST /dynasty/requests/recieved/{joinRequest}` | `account.security` |
| `DELETE /dynasty/requests/recieved/{joinRequest}` | `account.security` |

## Join Request Lifecycle

| Status value | Meaning | Transitioned by |
| --- | --- | --- |
| `0` | Pending | Default on creation |
| `1` | Accepted | `POST /dynasty/requests/recieved/{id}` |
| `-1` | Rejected | `DELETE /dynasty/requests/recieved/{id}` |

Accepted requests add the accepting user to the dynasty family roster and may award dynasty prizes. Rejected requests remain queryable but cannot be re-opened; attempting to resend creates a policy violation.

## Authorization Policies

- `JoinRequestPolicy::view` allows either party to inspect a request.
- `JoinRequestPolicy::delete` permits the sender to cancel only while the request is pending (`status === 0`). `SendJoinRequestController::destroy` should invoke this policy.
- `JoinRequestPolicy::accept` / `reject` limit inbound decisions to the target user, again only while pending.
- `UserPolicy::addFamilyMember` enforces business rules before a request can be created: dynasty existence, verification checks, unique relationships (single parents/spouse), offspring and sibling caps, and rejection history.[^policy]

[^policy]: Policies referenced from `app/Policies/JoinRequestPolicy.php` and `app/Policies/UserPolicy.php`.

## Validation Rules

- `AddFamilyMemberRequest` powers `POST /dynasty/add/member` and enforces:[^add-family]
  - `user`: required existing user ID.
  - `relationship`: required enum; allowed values `brother`, `sister`, `father`, `mother`, `husband`, `wife`, `offspring`.
  - `permissions`: required array of at least 10 keys **only** when `relationship` is `offspring`; forbidden otherwise. Must include keys `BFR`, `SF`, `W`, `JU`, `DM`, `PIUP`, `PITC`, `PIC`, `ESOO`, `COTB`; each value castable to boolean/integer.
- `POST /dynasty/add/member/get/permissions`: requires `relationship=offspring`.
- `POST /dynasty/search`: requires a non-empty `searchTerm` string; returns transformed user cards.

[^add-family]: Validation defined in `app/Http/Requests/AddFamilyMemberRequest.php`.

## Data Shapes

### Sent Requests (`SentRequestsResource`)

```
```18:45:app/Http/Resources/Dynasty/SentRequestsResource.php
// ... existing code ...
```

- `status`: numeric lifecycle flag.
- `relationship`: localized Farsi label derived from `JoinRequest::getRelationShipTitle()`.
- `prize`: PSC value normalised by `Variable::getRate('psc')` plus satisfaction percentages.
- `message`: included only in `GET /dynasty/requests/sent/{id}`.

### Received Requests (`RecievedJoinRequest`)

```
```17:46:app/Http/Resources/Dynasty/RecievedJoinRequest.php
// ... existing code ...
```

- Adds pending permissions snapshot for offspring requests when viewed with `GET /dynasty/requests/recieved/{id}`.

## Endpoint Reference

### Sent Request Management (`SendJoinRequestController`)

| Method & Path | Description | Response |
| --- | --- | --- |
| `GET /dynasty/requests/sent` | Paginated (10 per page) list of pending/processed requests initiated by the current user. Includes target profile summary and incentive preview. | `SentRequestsResource` collection |
| `GET /dynasty/requests/sent/{joinRequest}` | Detailed view gated by `JoinRequestPolicy::view`. Adds request message and localized timestamps. | `SentRequestsResource` item |
| `POST /dynasty/add/member/get/permissions` | Returns default permission template for offspring additions. Requires `{"relationship": "offspring"}`. | `{ "permissions": { ... } }` |
| `POST /dynasty/add/member` | Creates a join request after validation and `UserPolicy::addFamilyMember` authorization. Under-18 offspring trigger permission scaffolding before notifications are sent.[^send-store] | `SentRequestsResource` (201) |
| `DELETE /dynasty/requests/sent/{joinRequest}` | Cancels a pending request. Authorizes with `JoinRequestPolicy::delete` and now calls the corrected `destroy` action. | `204 No Content` |
| `POST /dynasty/search` | Fuzzy search over potential members using `UserSearchService`. Returns transformed users under `date` key (typo documented below). | `{ "date": [ ... ] }` |

Payload example for `POST /dynasty/add/member`:

```json
{
  "user": 42,
  "relationship": "offspring",
  "permissions": {
    "BFR": 1,
    "SF": 0,
    "W": 1,
    "JU": 1,
    "DM": 1,
    "PIUP": 0,
    "PITC": 1,
    "PIC": 0,
    "ESOO": 1,
    "COTB": 1
  }
}
```

Side effects on create:

- Message templates resolve placeholders (`[sender-code]`, `[relationship]`, etc.) via `DynastyMessage` lookups, then notify both parties.[^send-messages]
- Offspring requests auto-provision permissions (`verified=false`) for the child if under 18.[^send-permissions]

### Received Request Decisions (`AcceptJoinRequestController`)

| Method & Path | Description | Response |
| --- | --- | --- |
| `GET /dynasty/requests/recieved` | Paginated pending requests targeting the authenticated user. | `RecievedJoinRequest` collection |
| `GET /dynasty/requests/recieved/{joinRequest}` | Detailed request plus message and (for offspring) the permissions snapshot. Requires policy `view`. | `RecievedJoinRequest` item |
| `POST /dynasty/requests/recieved/{joinRequest}` | Accepts request, flips status to `1`, creates family member entry for the accepter, awards dynasty prize, and delivers accept notifications. Handles under-18 permission verification rules.[^accept-flow] | `RecievedJoinRequest` item |
| `DELETE /dynasty/requests/recieved/{joinRequest}` | Rejects request, sets status `-1`, and sends rejection notifications to both parties. | `RecievedJoinRequest` item |

Acceptance nuances:

- If the requester (original dynasty owner) is under 18 and relationship is `father`, default permissions are cloned from `DynastyPermission` and marked verified.
- If the accepting user is under 18 and relationship is `offspring`, their existing permissions are marked verified.
- Prize selection uses `DynastyPrize::where('member', relationship)` and stores a personalized message in `recievedDynastyPrizes`.

## Notifications & Side Effects

- Both controllers use `JoinDynastyNotification`, with message bodies chosen by `DynastyMessage` templates (`requester_confirmation_message`, `reciever_message`, `requester_accept_message`, `reciever_accept_message`, `reciever_reject_message`, `requester_reject_message`). Placeholders `[sender-code]`, `[reciever-code]`, `[relationship]`, `[created_at]`, `[sender-name]`, `[reciever-name]` are substituted at runtime.
- Notification dispatch happens synchronously within the controller actions, so network latency affects API timing.

## Known Issues & Caveats

- `AcceptJoinRequestController::accept` contains a typo `$$requestedUser` when creating permissions, which throws an error during acceptance for under-18 father scenarios. Patch required before documenting that path as stable.[^accept-bug]
- `POST /dynasty/search` responds with key `date` instead of `data`; clients should be aware or normalize response shape manually.[^search-bug]

[^send-store]: Flow documented in `app/Http/Controllers/Api/V1/Dynasty/SendJoinRequestController.php`.
[^send-messages]: Message preparation occurs in `prepareMessages()` within the same controller.
[^send-permissions]: Permission seeding handled by `setOffspringPermissions()` for under-18 offspring.
[^accept-flow]: Acceptance logic defined in `app/Http/Controllers/Api/V1/Dynasty/AcceptJoinRequestController.php`.
[^accept-bug]: Typo observed in `AcceptJoinRequestController::accept` (`$$requestedUser`).
[^search-bug]: `search()` response defined in `SendJoinRequestController::search`.

## Change Log

- 2025-11-09 – Initial draft derived from current Laravel implementation.

