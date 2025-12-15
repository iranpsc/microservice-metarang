# Settings API Guide

## Summary
- `GET /api/settings` returns the authenticated citizen’s account-management snapshot including checkout cadence and reset counters.
- `POST /api/settings` lets users adjust logout thresholds and toggle profile status, level visibility, and detail exposure.
- `GET /api/general-settings` + `PUT /api/general-settings/{setting}` expose and mutate notification delivery preferences subject to the `SettingPolicy@update` rule.
- `GET /api/privacy` + `POST /api/privacy` retrieve and flip granular profile-privacy toggles across dozens of data points.

## Access Control
- **Middleware stack:** every route sits inside the global `auth:sanctum`, `verified`, and `activity` middleware group (`routes/api.php`), so callers must use a verified Sanctum token and pass activity tracking.
- **Authorization policy:** the `UpdateSettingNotificationsRequest` delegates to `SettingPolicy@update`, ensuring users can only update their own `Setting` records.
- **Route model binding:** `PUT /api/general-settings/{setting}` uses implicit binding; the `Setting` instance must exist and belong to the authenticated user or policy enforcement denies the call.

## Route Registry
| Method | Path | Middleware | Controller action | Purpose |
| --- | --- | --- | --- | --- |
| GET | `/api/settings` | `auth:sanctum`, `verified`, `activity` | `SettingController@showSettings` | Fetch the caller’s core settings snapshot. |
| POST | `/api/settings` | `auth:sanctum`, `verified`, `activity` | `SettingController@update` | Update logout thresholds or toggle profile exposure flags. |
| GET | `/api/general-settings` | `auth:sanctum`, `verified`, `activity` | `SettingController@showGeneralSettings` | Return notification channel preferences. |
| PUT | `/api/general-settings/{setting}` | `auth:sanctum`, `verified`, `activity`, policy `can:update,setting` | `SettingController@updateGeneralSettings` | Persist notification channel preferences. |
| GET | `/api/privacy` | `auth:sanctum`, `verified`, `activity` | `SettingController@getPrivacySettings` | View privacy toggles for all supported profile fields. |
| POST | `/api/privacy` | `auth:sanctum`, `verified`, `activity` | `SettingController@updatePrivacySettings` | Flip a single privacy toggle by key. |

## Data Model Reference
- **Setting resource:** exposes checkout cadence and reset counters; `message` appears only when present on the model.

```21:27:app/Http/Resources/SettingResource.php
        return [
            'checkout_days_count' => $this->checkout_days_count,
            'automatic_logout' => $this->automatic_logout,
        ];
```

- **Notification resource:** normalizes the `notifications` JSON column to booleans.

```18:28:app/Http/Resources/NotificationSettingsResource.php
        return [
            'announcements_sms'           => $this->notifications['announcements_sms'],
            'announcements_email'           => $this->notifications['announcements_email'],
            'reports_sms'                 => $this->notifications['reports_sms'],
            'reports_email'                 => $this->notifications['reports_email'],
            'login_verification_sms'    => $this->notifications['login_verification_sms'],
            'login_verification_email'    => $this->notifications['login_verification_email'],
            'transactions_sms'   => $this->notifications['transactions_sms'],
            'transactions_email'   => $this->notifications['transactions_email'],
            'trades_sms'         => $this->notifications['trades_sms'],
            'trades_email'         => $this->notifications['trades_email'],
        ];
```

- **Persistent defaults:** the `Setting` model casts `privacy` and `notifications` to arrays and seeds defaults for every switch, while `automatic_logout` defaults to 60 minutes.

```24:51:app/Models/Setting.php
    protected function casts()
    {
        return [
            'automatic_logout' => 'integer',
            'privacy' => 'array',
            'notifications' => 'array',
        ];
    }
```

## Endpoint Details

### `GET /api/settings` – Settings Snapshot
**Authentication:** required (Sanctum).  
**Response:** `200 OK` with `SettingResource`. Missing `message` keys are simply omitted.  
**Use cases:** preload dashboard toggles, surface remaining reset attempts, display auto-logout schedule.

**Errors**
- `401 Unauthorized` when the Sanctum token is missing or invalid.
- `403 Forbidden` if the user is not verified (middleware short-circuits before controller).

### `POST /api/settings` – Update Core Settings
**Authentication:** required (Sanctum).  
**Content type:** `application/json`.  
**Behavior:** two mutually optional payload segments can be included in a single request:
- **Checkout cadence:** when `checkout_days_count` is present, payload must also include `automatic_logout`. Validation enforces integers `3–1000` for checkout days and `1–55` for automatic logout minutes.
- **Profile exposure toggle:** when `setting` is present, payload must also include `status`. `setting` accepts only `status`, `level`, or `details`; `status` must be boolean. The controller updates the named attribute to the provided status.

**Validation summary:**

```34:53:app/Http/Controllers/Api/V1/SettingController.php
        if ($request->has('setting')) {
            $request->validate([
                'setting' => 'required|in:status,level,details',
                'status' => 'required|boolean',
            ]);
            $settings->update([
                $request->input('setting') => $request->input('status'),
            ]);
        }
```

**Response:** `204 No Content` on success.  
**Errors:** `422 Unprocessable Entity` for validation failures; `401/403` middleware failures as above.

### `GET /api/general-settings` – Notification Preferences
**Authentication:** required.  
**Response:** `200 OK` with `NotificationSettingsResource`. All ten channels return booleans (`0` or `1`) mapped from the stored JSON blob.

### `PUT /api/general-settings/{setting}` – Update Notification Preferences
**Authentication:** required.  
**Authorization:** enforced by the `SettingPolicy@update`; the bound `Setting` must belong to the authenticated user.

```16:22:app/Policies/SettingPolicy.php
    public function update(User $user, Setting $setting)
    {
        return $setting->user->is($user);
    }
```

**Validation:** all ten notification flags are required and must be boolean.

```27:36:app/Http/Requests/UpdateSettingNotificationsRequest.php
        return [
            'announcements_sms' => 'required|boolean',
            'announcements_email' => 'required|boolean',
            'reports_sms' => 'required|boolean',
            'reports_email' => 'required|boolean',
            'login_verification_sms' => 'required|boolean',
            'login_verification_email' => 'required|boolean',
            'transactions_sms' => 'required|boolean',
            'transactions_email' => 'required|boolean',
            'trades_sms' => 'required|boolean',
            'trades_email' => 'required|boolean',
        ];
```

**Response:** `200 OK` with the updated `NotificationSettingsResource`.  
**Errors:** `403 Forbidden` if the policy blocks access; `404 Not Found` when the `{setting}` id does not resolve; `422` for validation failures.

### `GET /api/privacy` – Privacy Matrix
**Authentication:** required.  
**Response:** `200 OK` with `{ "data": { <key>: <0|1>, ... } }`, echoing all privacy switches. Defaults are seeded to `1` (public) except a handful of contact fields.

### `POST /api/privacy` – Update a Privacy Toggle
**Authentication:** required.  
**Payload:** specify a single `key` and `value`.

```29:130:app/Http/Requests/UpdatePrivacyRequest.php
            'key' => [
                'required',
                Rule::in(
                    'nationality',
                    'fname',
                    'birthdate',
                    'phone',
                    'email',
                    'address',
                    'about',
                    'name',
                    'registered_at',
                    'position',
                    'level',
                    'score',
                    'licenses',
                    'license_score',
                    'avatar',
                    'occupation',
                    'education',
                    'loved_city',
                    'loved_country',
                    'loved_language',
                    'prediction',
                    'memory',
                    'passions',
                    'amoozeshi_features',
                    'maskoni_features',
                    'tejari_features',
                    'gardeshgari_features',
                    'fazasabz_features',
                    'behdashti_features',
                    'edari_features',
                    'nemayeshgah_features',
                    'bought_golden_keys',
                    'used_golden_keys',
                    'recieved_golden_keys',
                    'bought_bronze_keys',
                    'used_bronze_keys',
                    'recieved_bronze_keys',
                    'establish_store_license',
                    'establish_union_license',
                    'establish_taxi_license',
                    'establish_amoozeshgah_license',
                    'reporter_license',
                    'cooporation_license',
                    'developer_license',
                    'inspection_license',
                    'trading_license',
                    'lawyer_license',
                    'city_council_license',
                    'governer_license',
                    'ostandar_license',
                    'level_one_judge_license',
                    'level_two_judge_license',
                    'level_three_judge_license',
                    'gate_license',
                    'all_licenses',
                    'referrals',
                    'irr_income',
                    'psc_income',
                    'complaint',
                    'warnings',
                    'commited_crimes',
                    'satisfaction',
                    'referral_profit',
                    'irr_transactions',
                    'psc_transactions',
                    'blue_transactions',
                    'yellow_transactions',
                    'red_transactions',
                    'sold_features',
                    'bought_features',
                    'sold_products',
                    'bought_products',
                    'recieved_irr_prizes',
                    'recieved_psc_prizes',
                    'recieved_yellow_prizes',
                    'recieved_blue_prizes',
                    'recieved_red_prizes',
                    'recieved_satisfaction_prizes',
                    'dynasty_members_photo',
                    'dynasty_members_info',
                    'recieved_dynasty_satisfaction_prizes',
                    'recieved_dynasty_referral_profit_prizes',
                    'recieved_dynasty_accumulated_capital_reserve_prizes',
                    'recieved_dynasty_data_storage_prizes',
                    'followers',
                    'followers_count',
                    'following',
                    'following_count',
                    'violations',
                    'breaking_laws',
                    'paid_psc_fine',
                    'paid_irr_fine',
                    'life_style',
                    'negative_score',
                    'code'
                ),
            ],
            'value' => 'required|numeric|boolean'
```

**Notes:**
- `value` accepts boolean or numeric representations (`true`/`false`, `1`/`0`).
- Only one key can be toggled per request; repeated calls are idempotent.

**Response:** `204 No Content` on success.  
**Errors:** `422` for invalid keys or non-boolean values.

## Example Usage
```bash
curl -X PUT "https://example.com/api/general-settings/12" \
  -H "Authorization: Bearer <token>" \
  -H "Accept: application/json" \
  -H "Content-Type: application/json" \
  -d '{
        "announcements_sms": false,
        "announcements_email": true,
        "reports_sms": true,
        "reports_email": true,
        "login_verification_sms": true,
        "login_verification_email": true,
        "transactions_sms": false,
        "transactions_email": true,
        "trades_sms": true,
        "trades_email": true
      }'
```

## Common Failure Modes
- `401 Unauthorized`: missing or invalid Sanctum token.
- `403 Forbidden`: account not verified or policy denies ownership (`PUT /api/general-settings/{setting}`).
- `404 Not Found`: the `{setting}` identifier does not resolve to an owned record.
- `422 Unprocessable Entity`: validation breaks (out-of-range checkout days, invalid privacy key, missing boolean flag).


