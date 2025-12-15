# Dynasty Children Permissions API

## Overview
This endpoint lets guardians toggle individual permissions for minors who are already part of their dynasty. It updates the child’s existing `childrenPermission` record and returns an empty JSON body on success.

> Requires: existing family relationship between caller and child, child under 18, and a previously created permissions record (established during dynasty join workflows).

## Authentication & Middleware
- `auth:sanctum` – caller must provide a valid bearer token.
- `verified` – the authenticated account must be email/identity verified.
- `activity` – account must not be suspended or inactive.

## Authorization Policy
| Ability | Policy method | Source | Rules |
| --- | --- | --- | --- |
| `controlPermissions` | `UserPolicy::controlPermissions` | `ChildernPermissionsController` | Guardian cannot target themselves, child must be under 18, and must belong to guardian’s dynasty family roster. |

## Endpoint Summary
- **Method & URL:** `POST /api/dynasty/children/{user}`
- **Path parameter:** `{user}` – numeric user id resolved via route-model binding. 404 is returned if the user does not exist.

## Request Payload
| Field | Type | Required | Validation | Description |
| --- | --- | --- | --- | --- |
| `permission` | string | Yes | `required|string|in:BFR,SF,W,JU,DM,PIUP,PITC,PIC,ESOO,COTB` | Code of the permission flag to toggle. |
| `status` | boolean | Yes (when `permission` present) | `required_with:permission|boolean` | Target boolean value for the selected permission. |

### Permission Codes
- `BFR` – Buy from RGB marketplace
- `SF` – Sell features
- `W` – Withdraw funds
- `JU` – Join unions
- `DM` – Dynasty management access
- `PIUP` – Participate in union projects
- `PITC` – Participate in challenges
- `PIC` – Participate in contests
- `ESOO` – Establish store or office
- `COTB` – Construction of buildings

## Successful Response
- **Status:** `200 OK`
- **Body:** Empty JSON array `[]`

### Example
```json
POST /api/dynasty/children/42
{
  "permission": "W",
  "status": true
}

HTTP/1.1 200 OK
[]
```

## Error Responses
- `401 Unauthorized` – Missing/invalid token.
- `403 Forbidden` – Policy denies the action (adult child, not in family, or self-targeting).
- `404 Not Found` – Target user id does not resolve.
- `422 Unprocessable Entity` – Validation failure (`permission` outside allowed list or non-boolean `status`).

## Implementation Notes
- Controller: `app/Http/Controllers/Api/V1/Dynasty/ChildernPermissionsController.php`
- Validation: `app/Http/Requests/UpdateChildrenPermissionsRequest.php`
- Policy: `app/Policies/UserPolicy.php::controlPermissions`
- Permissions model: `app/Models/Dynasty/childrenPermission.php` (fields include all permission flags and `verified` status).
- On success, the controller calls `$user->permissions->update([$request->permission => $request->status]);`. If the child lacks a permissions record, Laravel will throw `ErrorException`; ensure dynasty join flows create the record before toggling.


